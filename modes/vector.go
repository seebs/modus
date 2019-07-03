package modes

import (
	"fmt"
	"math/rand"

	math "github.com/chewxy/math32"
	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/keys"
	"seebs.net/modus/sound"
)

// vectorMode is one of the internal modes based on vector graphics
type vectorMode struct {
	name        string
	cycleTime   int // number of ticks to go by between updates
	compute     func(*vectorScene, keys.Map) string
	computeInit func(*vectorScene)
}

const vectorCycleTime = 1

var vectorModes = []vectorMode{
	{name: "Test", cycleTime: vectorCycleTime, compute: simpleDemo, computeInit: simpleDemoInit},
}

func init() {
	for _, mode := range vectorModes {
		allModes = append(allModes, mode)
	}
}

func (m vectorMode) Name() string {
	return fmt.Sprintf("vector%s", m.name)
}

func (m vectorMode) Description() string {
	return fmt.Sprintf("vector: %s", m.name)
}

func (m vectorMode) New(gctx *g.Context, detail int, p *g.Palette) (Scene, error) {
	return newVectorScene(m, gctx, detail, p)
}

type knotProto struct {
	pts []g.LinePoint
}

var sampleKnots = map[string]knotProto{
	"ship": knotProto{
		pts: []g.LinePoint{
			{X: 0.5, Y: 0},
			{X: 0, Y: .25},
			{X: -0.25, Y: 0},
			{X: 0, Y: -0.25},
			{X: 0.5, Y: 0},
			{X: 0, Y: .125, Skip: true, P: 3},
			{X: 0, Y: -.125, P: 3},
		},
	},
}

// simpleDemo is just a trivial test case
func simpleDemoInit(s *vectorScene) {
	for i := 0; i < 4; i++ {
		proto := sampleKnots["ship"]
		k1 := s.wv.NewKnot(len(proto.pts))
		b := bouncer{k: k1, pOffset: i}
		copy(k1.Points, proto.pts)
		k1.Dirty()
		k1.Size = float32(1.0) / float32(i+1)
		k1.X, k1.Y = -0.5+float32(i&1), -0.5+float32(i>>1)
		for j := range k1.Points {
			k1.Points[j].P = s.palette.Inc(k1.Points[j].P, b.pOffset)
		}
		b.pt = g.MovingPoint{Loc: g.Point{X: k1.X, Y: k1.Y}, Velocity: g.Vec{X: -k1.X * .002 * (float32(i) + 1), Y: -k1.Y * .002 * (float32(i) + 1)}, Bounds: s.bounds}
		s.bouncers = append(s.bouncers, b)
	}
}

func simpleDemo(s *vectorScene, km keys.Map) string {
	b := &s.bouncers[0]
	sin, cos := math.Sincos(b.k.Theta)
	if km.Down(ebiten.KeyW, ebiten.KeyUp) {
		b.pt.Velocity.X += cos * .0001
		b.pt.Velocity.Y += sin * .0001
		p := s.pt.Add(g.SecondSplasher, g.Paint(b.pOffset+1))
		p.Alpha = 0
		p.Scale = rand.Float32()/4 + 0.125
		p.X0, p.Y0 = -0.03, 0
		p.DX = -(0.05 + (rand.Float32() / 8))
		p.DY = (rand.Float32() - 0.5) / 8
		if math.Abs(p.DY) > 0.05 {
			p.P++
		}
		p.DTheta = p.DY
	}
	if km.Down(ebiten.KeyA, ebiten.KeyLeft) {
		b.k.Theta -= .05
	}
	if km.Down(ebiten.KeyD, ebiten.KeyRight) {
		b.k.Theta += 0.05
	}
	// s.pt.Theta += .01
	s.pt.X, s.pt.Y = b.pt.Loc.X-cos*.1, b.pt.Loc.Y-sin*.1
	s.pt.Theta = b.k.Theta
	for idx := range s.bouncers {
		b := &s.bouncers[idx]
		b.pt.Update()
		b.k.X, b.k.Y = b.pt.Loc.X, b.pt.Loc.Y
		b.k.Dirty()
	}
	return ""
}

type vectorScene struct {
	palette  *g.Palette
	gctx     *g.Context
	mode     vectorMode
	wv       *g.Weave
	pt       *g.Particles
	detail   int
	cycle    int
	t0       float32
	bouncers []bouncer
	bounds   g.Region
}

type bouncer struct {
	pt      g.MovingPoint
	pOffset int
	k       *g.Knot
}

func newVectorScene(m vectorMode, gctx *g.Context, detail int, p *g.Palette) (*vectorScene, error) {
	sc := &vectorScene{mode: m, gctx: gctx, detail: detail, palette: p}
	_, _, _, cx, cy := gctx.Centered()
	sc.bounds = g.Region{
		Min: g.Point{X: -1 - cx, Y: -1 - cy},
		Max: g.Point{X: 1 + cx, Y: 1 + cy},
	}
	err := sc.Reset(detail, p)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *vectorScene) Mode() Mode {
	return s.mode
}

func (s *vectorScene) Reset(detail int, p *g.Palette) error {
	_ = s.Hide()
	s.palette = p
	err := s.Display()
	if err != nil {
		return err
	}
	// and then reset the grid, and the knights, i guess?
	return nil
}

func (s *vectorScene) Display() error {
	s.wv = s.gctx.NewWeave(8, s.palette)
	s.pt = s.gctx.NewParticles(16, 1, s.palette)
	if s.mode.computeInit != nil {
		s.mode.computeInit(s)
	}
	return nil
}

func (s *vectorScene) Hide() error {
	s.wv = nil
	s.pt = nil
	return nil
}

func (s *vectorScene) Tick(voice *sound.Voice, km keys.Map) (bool, error) {
	s.cycle = (s.cycle + 1) % s.mode.cycleTime
	s.t0++
	if s.cycle != 0 {
		return false, nil
	}
	if s.mode.compute != nil {
		s.wv.SetStatus(s.mode.compute(s, km))
		s.pt.Tick()
	}
	return true, nil
}

func (s *vectorScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.wv.Draw(t, 1.0, scale)
		s.pt.Draw(t, scale)
	})
	return nil
}
