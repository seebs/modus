package g

import (
	"sync"

	math "github.com/chewxy/math32"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

type Particles struct {
	X, Y                    float32
	DX, DY                  float32
	Size                    float32
	Theta                   float32
	Alpha                   float32
	r                       RenderType
	palette                 *Palette
	particles               []*Particle
	vertices                []ebiten.Vertex
	indices                 []uint16
	status                  string
	scale, offsetX, offsetY float32
}

var particleTextureSetup sync.Once

func newParticles(size float32, r RenderType, p *Palette, scale, offsetX, offsetY float32) *Particles {
	particleTextureSetup.Do(textureSetup)
	return &Particles{Size: size, r: r, palette: p, scale: scale, offsetX: offsetX, offsetY: offsetY}
}

func (ps *Particles) Add(anim ParticleAnimation, p Paint, X0, Y0 float32) *Particle {
	np := &Particle{Anim: anim, P: p, Scale: 1.0, Alpha: 1.0, X: 0, Y: 0, DX: 0, DY: 0, Tick: 0}
	np.psSin, np.psCos = math.Sincos(ps.Theta)
	np.psTheta = ps.Theta
	np.x0 = ((ps.X + (X0*np.psCos - Y0*np.psSin)) * ps.scale) + ps.offsetX
	np.y0 = ((ps.Y + (X0*np.psSin + Y0*np.psCos)) * ps.scale) + ps.offsetY
	ps.particles = append(ps.particles, np)
	offset := uint16(len(ps.vertices))
	ps.vertices = append(ps.vertices, dotData.vsByR[ps.r]...)
	ps.indices = append(ps.indices,
		offset+0, offset+1, offset+2,
		offset+2, offset+1, offset+3)
	return np
}

// Tick returns true when it's done, at which point the emitter is probably done.
func (ps *Particles) Tick() bool {
	j := 0
	for _, p := range ps.particles {
		if !p.Anim.Tick(p) {
			ps.particles[j] = p
			j++
		}
	}
	ps.particles = ps.particles[:j]
	ps.X, ps.Y = ps.X+ps.DX, ps.Y+ps.DY
	return len(ps.particles) == 0
}

func (ps *Particles) Draw(target *ebiten.Image, scale float32) {
	opt := ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter}
	offset := 0
	r := dotData.vsByR[ps.r]
	thickness := ps.Size * dotData.scales[ps.r]
	for _, p := range ps.particles {
		vs := ps.vertices[offset : offset+4]
		x, y := p.x0+((p.X*p.psCos-p.Y*p.psSin)*ps.Size), p.y0+((p.X*p.psSin+p.Y*p.psCos)*ps.Size)
		size := thickness * p.Scale
		var sin, cos float32
		if p.Theta+p.psTheta == 0 {
			cos = 1
		} else {
			sin, cos = math.Sincos(p.Theta + p.psTheta)
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
		vs[0].SrcX, vs[0].SrcY = r[0].SrcX, r[0].SrcY
		vs[1].SrcX, vs[1].SrcY = r[1].SrcX, r[1].SrcY
		vs[2].SrcX, vs[2].SrcY = r[2].SrcX, r[2].SrcY
		vs[3].SrcX, vs[3].SrcY = r[3].SrcX, r[3].SrcY
		r, g, b, _ := ps.palette.Float32(p.P)
		vs[0].ColorR, vs[0].ColorG, vs[0].ColorB, vs[0].ColorA = r, g, b, p.Alpha
		vs[1].ColorR, vs[1].ColorG, vs[1].ColorB, vs[1].ColorA = r, g, b, p.Alpha
		vs[2].ColorR, vs[2].ColorG, vs[2].ColorB, vs[2].ColorA = r, g, b, p.Alpha
		vs[3].ColorR, vs[3].ColorG, vs[3].ColorB, vs[3].ColorA = r, g, b, p.Alpha
		offset += 4
	}
	if target != nil {
		target.DrawTriangles(ps.vertices, ps.indices[:len(ps.particles)*6], dotData.img, &opt)
		ebitenutil.DebugPrint(target, ps.status)
	}
}

type Particle struct {
	P            Paint
	Scale        float32
	Alpha        float32
	Theta        float32
	DTheta       float32
	X, Y         float32 // relative to X0, Y0
	x0, y0       float32 // starting point
	psSin, psCos float32 // particle system's sin/cos when we were emitted
	psTheta      float32 // particle system's theta when we were emittted
	DX, DY       float32
	Tick         int
	Delay        int
	Anim         ParticleAnimation
}

type ParticleAnimation interface {
	Tick(*Particle) bool
}

type Splasher struct {
	duration int
}

func (s Splasher) Tick(p *Particle) bool {
	if p.Delay > 0 {
		p.Delay--
		return false
	}
	if p.Tick >= s.duration {
		p.Alpha = 0
		return true
	}
	t := float32(p.Tick) * math.Pi / float32(s.duration)
	p.Alpha = math.Sin(t)
	p.X += p.DX
	p.Y += p.DY
	p.DX *= 0.95
	p.DY *= 0.95
	p.Theta += p.DTheta
	p.Tick++
	return false
}

var SecondSplasher = Splasher{duration: 30}
