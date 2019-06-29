package g

import (
	math "github.com/chewxy/math32"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

type Particles struct {
	X, Y      float32
	DX, DY    float32
	Size      float32
	sx, sy    int
	r         RenderType
	palette   *Palette
	particles []*Particle
	vertices  []ebiten.Vertex
	indices   []uint16
	status    string
}

func newParticles(w int, r RenderType, p *Palette, sx, sy int) *Particles {
	return &Particles{Size: float32(sx) / float32(w), r: r, palette: p, sx: sx, sy: sy}
}

func (ps *Particles) Add(anim ParticleAnimation, p Paint) *Particle {
	np := &Particle{Anim: anim, P: p, Scale: 1.0, Alpha: 1.0, X: 0, Y: 0, DX: 0, DY: 0, Tick: 0}
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
	for idx, p := range ps.particles {
		if p.Anim.Tick(p) {
			ps.particles[idx] = nil
		}
	}
	j := 0
	for i := 0; i < len(ps.particles); i++ {
		if ps.particles[i] != nil {
			ps.particles[j] = ps.particles[i]
			j++
		}
	}
	ps.particles = ps.particles[:j]
	return len(ps.particles) == 0
}

func (ps *Particles) Draw(target *ebiten.Image, scale float32) {
	opt := ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter}
	offset := 0
	r := dotData.vsByR[ps.r]
	thickness := ps.Size * dotData.scales[ps.r]
	for _, p := range ps.particles {
		vs := ps.vertices[offset : offset+4]
		// scale is a multiplier on the base thickness/size of
		// dots
		x, y := ps.X+p.X0+(p.X*ps.Size), ps.Y+p.Y0+(p.Y*ps.Size)
		size := thickness * p.Scale
		vs[0].DstX, vs[0].DstY = (x-size)*scale, (y-size)*scale
		vs[1].DstX, vs[1].DstY = (x+size)*scale, (y-size)*scale
		vs[2].DstX, vs[2].DstY = (x-size)*scale, (y+size)*scale
		vs[3].DstX, vs[3].DstY = (x+size)*scale, (y+size)*scale
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
	target.DrawTriangles(ps.vertices, ps.indices[:len(ps.particles)*6], dotData.img, &opt)
	ebitenutil.DebugPrint(target, ps.status)
}

type Particle struct {
	P      Paint
	Scale  float32
	Alpha  float32
	X, Y   float32 // relative to X0, Y0
	X0, Y0 float32 // starting point
	DX, DY float32
	Tick   int
	Delay  int
	Anim   ParticleAnimation
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
	p.Tick++
	return false
}

var SecondSplasher = Splasher{duration: 30}