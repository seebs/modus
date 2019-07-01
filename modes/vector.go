package modes

import (
	"fmt"

	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/keys"
	"seebs.net/modus/sound"
)

// vectorMode is one of the internal modes based on vector graphics
type vectorMode struct {
	name        string
	cycleTime   int // number of ticks to go by between updates
	compute     func(*vectorScene) string
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
			{X: 0, Y: -0.5},
			{X: .25, Y: 0},
			{X: 0.0, Y: 0.25},
			{X: -.25, Y: 0},
			{X: 0, Y: -0.5},
			{X: -.125, Y: 0, Skip: true, P: 4},
			{X: .125, Y: 0, P: 4},
		},
	},
}

// simpleDemo is just a trivial test case
func simpleDemoInit(s *vectorScene) {
	for i := 0; i < 4; i++ {
		proto := sampleKnots["ship"]
		k1 := s.wv.NewKnot(len(proto.pts))
		b := bouncer{k: k1}
		copy(k1.Points, proto.pts)
		k1.Dirty()
		k1.Size = float32(1.0) / float32(i+1)
		k1.X, k1.Y = -0.5+float32(i&1), -0.5+float32(i>>1)
		b.pt = g.MovingPoint{Loc: g.Point{X: k1.X, Y: k1.Y}, Velocity: g.Vec{X: -k1.X * .002 * (float32(i) + 1), Y: -k1.Y * .002 * (float32(i) + 1)}, Bounds: s.bounds}
		s.bouncers = append(s.bouncers, b)
	}
}

func simpleDemo(s *vectorScene) string {
	for idx := range s.bouncers {
		b := &s.bouncers[idx]
		b.pt.Update()
		b.k.X, b.k.Y = b.pt.Loc.X, b.pt.Loc.Y
		b.k.Theta = s.t0 / ((float32(idx) + 1) * 256)
		b.k.Dirty()
	}
	return ""
}

type vectorScene struct {
	palette  *g.Palette
	gctx     *g.Context
	mode     vectorMode
	wv       *g.Weave
	detail   int
	cycle    int
	t0       float32
	bouncers []bouncer
	bounds   g.Region
}

type bouncer struct {
	pt g.MovingPoint
	k  *g.Knot
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
	s.wv = s.gctx.NewWeave(16, s.palette)
	if s.mode.computeInit != nil {
		s.mode.computeInit(s)
	}
	return nil
}

func (s *vectorScene) Hide() error {
	s.wv = nil
	return nil
}

func (s *vectorScene) Tick(voice *sound.Voice, km keys.Map) (bool, error) {
	s.cycle = (s.cycle + 1) % s.mode.cycleTime
	s.t0++
	if s.cycle != 0 {
		return false, nil
	}
	if s.mode.compute != nil {
		s.wv.SetStatus(s.mode.compute(s))
	}
	return true, nil
}

func (s *vectorScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.wv.Draw(t, 1.0, scale)
	})
	return nil
}
