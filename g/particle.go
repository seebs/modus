package g

import (
	"errors"
	"fmt"
	"sync"

	math "github.com/chewxy/math32"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

// The need for the []*Particle, and for Particle to be large even when
// a specific implementation could be small, is a flaw. To fix it, we really
// need to have a couple of implementations of differing complexity, or
// something like that, I think?

type ParticleSystem struct {
	X, Y                    float32
	DX, DY                  float32
	Size                    float32
	Theta                   float32
	Alpha                   float32
	r                       RenderType
	palette                 *Palette
	particles               Particles
	vertices                []ebiten.Vertex
	indices                 []uint16
	status                  string
	scale, offsetX, offsetY float32
	Anim                    ParticleAnimation
}

type ParticlePos struct {
	X, Y, Theta float32
}

type ParticleState struct {
	ParticlePos
	Scale float32
	P     Paint
	Alpha float32
}

// ParticleParams are the things you might wish to specify for a particle.
// Note that delta.scale/p/alpha are ignored.
type ParticleParams struct {
	State ParticleState
	Delta ParticlePos
	Delay int
}

// ParticleAnimation's interface
type ParticleAnimation interface {
	Tick() bool
}

type Particles interface {
	Animation(name string, params ...interface{}) (ParticleAnimation, error)
	Add(ps *ParticleSystem, params ParticleParams) error
	Drawable() []ParticleState
}

var particleTextureSetup sync.Once

func newParticles(size float32, r RenderType, p *Palette, scale, offsetX, offsetY float32, particles Particles) *ParticleSystem {
	particleTextureSetup.Do(textureSetup)
	return &ParticleSystem{particles: particles, Size: size, r: r, palette: p, scale: scale, offsetX: offsetX, offsetY: offsetY}
}

func (ps *ParticleSystem) Add(params ParticleParams) error {
	if len(ps.vertices)+len(dotData.vsByR[ps.r]) > 65535 {
		return errors.New("too many vertices")
	}
	if err := ps.particles.Add(ps, params); err != nil {
		return err
	}
	offset := uint16(len(ps.vertices))
	ps.vertices = append(ps.vertices, dotData.vsByR[ps.r]...)
	ps.indices = append(ps.indices,
		offset+0, offset+1, offset+2,
		offset+2, offset+1, offset+3)
	return nil
}

// Tick returns true when it's done, at which point the emitter is probably done.
func (ps *ParticleSystem) Tick() bool {
	if ps.Anim == nil {
		fmt.Printf("nil anim\n")
		return true
	}
	if ps.Anim.Tick() {
		return true
	}
	ps.X, ps.Y = ps.X+ps.DX, ps.Y+ps.DY
	return false
}

// Project computes screen-space coordinates for a given x0/y0. So, 0, 0
// should give the center of the particle system, and 1, 0 gives a point one
// particle-system unit in +X, rotated according to particle system's theta.
func (ps *ParticleSystem) Project(x0, y0 float32) (x1, y1 float32) {
	var sin, cos float32
	if ps.Theta != 0 {
		sin, cos = math.Sincos(ps.Theta)
	} else {
		cos = 1
	}
	x1 = ((ps.X + (x0*cos - y0*sin)) * ps.scale) + ps.offsetX
	y1 = ((ps.Y + (x0*sin + y0*cos)) * ps.scale) + ps.offsetY
	return x1, y1
}

// ProjectWithDelta also translates dx/dy values, which don't get offset
func (ps *ParticleSystem) ProjectWithDelta(x0, y0 float32, dx, dy float32) (x1, y1 float32, dx1, dy1 float32) {
	var sin, cos float32
	if ps.Theta != 0 {
		sin, cos = math.Sincos(ps.Theta)
	} else {
		cos = 1
	}
	x1 = ((ps.X + (x0*cos - y0*sin)) * ps.scale) + ps.offsetX
	y1 = ((ps.Y + (x0*sin + y0*cos)) * ps.scale) + ps.offsetY
	dx1, dy1 = (dx*cos-dy*sin)*ps.Size, (dx*sin+dy*cos)*ps.Size
	return x1, y1, dx1, dy1
}

func (ps *ParticleSystem) Draw(target *ebiten.Image, scale float32) {
	opt := ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter}
	offset := 0
	// r := dotData.vsByR[ps.r]
	thickness := ps.Size * dotData.scales[ps.r]
	states := ps.particles.Drawable()
	for i := range states {
		p := &states[i]
		if p.Alpha == 0 || p.Scale == 0 {
			continue
		}
		vs := ps.vertices[offset : offset+4]
		x, y := p.X, p.Y
		size := thickness * p.Scale
		// if i == 0 {
		// 	fmt.Printf("particle 0: size %.2f, x, y %.2f, %.2f, alpha %.2f\n", size, x, y, p.Alpha)
		// }
		var sin, cos float32
		if p.Theta == 0 {
			cos = 1
		} else {
			sin, cos = math.Sincos(p.Theta)
		}
		sin, cos = sin*size, cos*size
		// we want points which are rotated around x, y, and which
		// correspond to +/- size pixels out, only rotated by sin/cos.
		// Points:
		// 0: -1, -1
		// 1: 1, -1
		// 2: -1, 1
		// 3: 1, 1
		// In each case, X' is (x*cos-y*sin), and Y' is (x*sin+y*cos)
		// So that gives us:
		// 0: -cos+sin, -sin-cos
		// 1: +cos+sin, +sin-cos
		// 2: -cos-sin, -sin+cos
		// 3: +cos-sin, +sin+cos
		// so 0/3 and 1/2 are opposites, which is unsurprising.
		vs[0].DstX, vs[0].DstY = (x-cos+sin)*scale, (y-sin-cos)*scale
		vs[1].DstX, vs[1].DstY = (x+cos+sin)*scale, (y+sin-cos)*scale
		vs[2].DstX, vs[2].DstY = (x-cos-sin)*scale, (y-sin+cos)*scale
		vs[3].DstX, vs[3].DstY = (x+cos-sin)*scale, (y+sin+cos)*scale
		// vs[0].SrcX, vs[0].SrcY = r[0].SrcX, r[0].SrcY
		// vs[1].SrcX, vs[1].SrcY = r[1].SrcX, r[1].SrcY
		// vs[2].SrcX, vs[2].SrcY = r[2].SrcX, r[2].SrcY
		// vs[3].SrcX, vs[3].SrcY = r[3].SrcX, r[3].SrcY
		r, g, b, _ := ps.palette.Float32(p.P)
		vs[0].ColorR, vs[0].ColorG, vs[0].ColorB, vs[0].ColorA = r, g, b, p.Alpha
		vs[1].ColorR, vs[1].ColorG, vs[1].ColorB, vs[1].ColorA = r, g, b, p.Alpha
		vs[2].ColorR, vs[2].ColorG, vs[2].ColorB, vs[2].ColorA = r, g, b, p.Alpha
		vs[3].ColorR, vs[3].ColorG, vs[3].ColorB, vs[3].ColorA = r, g, b, p.Alpha
		offset += 4
	}
	if target != nil {
		target.DrawTriangles(ps.vertices, ps.indices[:(offset/4)*6], dotData.img, &opt)
		ebitenutil.DebugPrint(target, ps.status)
	}
}

type particleMotionState struct {
	ParticlePos
	tick  int
	delay int
	skip  bool
}

type MovingParticles struct {
	States []ParticleState
	Deltas []particleMotionState
}

type splasher struct {
	particles *MovingParticles
	duration  int
	alphas    []float32
}

func (m *MovingParticles) Animation(name string, params ...interface{}) (ParticleAnimation, error) {
	if name != "splasher" {
		return nil, fmt.Errorf("unknown name %q, only support splasher", name)
	}
	if len(params) > 1 {
		return nil, fmt.Errorf("splasher only accepts duration parameter")
	}
	s := &splasher{particles: m, duration: 30}
	if len(params) == 1 {
		var ok bool
		if s.duration, ok = params[0].(int); !ok {
			return nil, fmt.Errorf("splasher duration must be integer")
		}
		if s.duration < 0 || s.duration >= (1<<16) {
			return nil, fmt.Errorf("splasher duration must be 1-65535")
		}
	}
	s.alphas = make([]float32, s.duration)
	thetaPerTick := math.Pi / float32(s.duration)
	theta := thetaPerTick
	for i := 1; i < s.duration; i++ {
		s.alphas[i] = math.Sin(theta)
		theta += thetaPerTick
	}
	return s, nil
}

func (m *MovingParticles) Drawable() []ParticleState {
	return m.States
}

// Add adds the given particle.
// All parameters are converted to be relative to the particle system.
func (m *MovingParticles) Add(ps *ParticleSystem, params ParticleParams) error {
	// translate starting location by particle system's location and rotation
	x0, y0, dx, dy := ps.ProjectWithDelta(params.State.X, params.State.Y, params.Delta.X, params.Delta.Y)
	// fmt.Printf("input X: %.2f, size %.2f, ps scale %.2f, theta %.2f, ps X %.2f, new x %.2f\n",
	// 	params.State.X, ps.Size, ps.scale, ps.Theta, ps.X, x0)
	state := params.State
	state.X = x0
	state.Y = y0
	state.Theta = ps.Theta
	state.Alpha = 0
	delta := particleMotionState{
		ParticlePos: ParticlePos{X: dx, Y: dy, Theta: params.Delta.Theta},
		tick:        0,
		delay:       params.Delay,
		skip:        false,
	}
	m.States = append(m.States, state)
	m.Deltas = append(m.Deltas, delta)
	return nil
}

func (s splasher) Tick() bool {
	found := 0
	states := s.particles.States
	deltas := s.particles.Deltas
	for i := range deltas {
		delta := &deltas[i]
		if delta.skip {
			continue
		}
		state := &states[i]
		found++
		if delta.delay > 0 {
			delta.delay--
			continue
		}
		if delta.tick >= s.duration {
			state.Alpha = 0
			delta.skip = true
			// things that have run out count as done
			found--
			continue
		}
		state.Alpha = s.alphas[delta.tick]
		state.X += delta.X
		delta.X *= 0.95
		state.Y += delta.Y
		delta.Y *= 0.95
		state.Theta += delta.Theta
		delta.tick++
	}
	if found == 0 {
		return true
	}
	// strip extras
	if found < len(s.particles.States)/2 {
		states := s.particles.States
		deltas := s.particles.Deltas
		n := 0
		for i := range deltas {
			if !deltas[i].skip {
				states[n] = states[i]
				deltas[n] = deltas[i]
				n++
			}
		}
		s.particles.States = states[:n]
		s.particles.Deltas = deltas[:n]
	}
	return false
}
