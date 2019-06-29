package modes

import (
	"fmt"

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

// simpleDemo is just a trivial test case
func simpleDemoInit(s *vectorScene) {
	points := make([]g.LinePoint, 6)
	s.pl.Joined = true
	s.pl.Points = points
	points[0].X, points[0].Y, points[0].P = -1, -1, 0
	points[1].X, points[1].Y, points[1].P = 0, -1, 1
	points[2].X, points[2].Y, points[2].P = 0, 0, 2
	points[3].X, points[3].Y, points[3].P = 1, 0, 3
	points[4].X, points[4].Y, points[4].P = -1, 1, 4
	points[5].X, points[5].Y, points[5].P = 0, 0, 5
	s.pl.Dirty()
}

func simpleDemo(s *vectorScene) string {
	pt := s.pl.Point(3)
	pt.X, pt.Y = math.Sincos(s.t0 / 256.0)
	s.pl.Dirty()
	return fmt.Sprintf("t0 %g, theta %g, pt[2].P %d, pt[3].P %d\n", s.t0, s.t0/256.0, s.pl.Point(2).P, s.pl.Point(3).P)
}

type vectorScene struct {
	palette *g.Palette
	gctx    *g.Context
	mode    vectorMode
	pl      *g.PolyLine
	detail  int
	cycle   int
	t0      float32
}

func newVectorScene(m vectorMode, gctx *g.Context, detail int, p *g.Palette) (*vectorScene, error) {
	sc := &vectorScene{mode: m, gctx: gctx, detail: detail, palette: p}
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
	s.pl = s.gctx.NewPolyline(32, 3, s.palette)
	s.pl.SetGlow(true)
	if s.mode.computeInit != nil {
		s.mode.computeInit(s)
	}
	return nil
}

func (s *vectorScene) Hide() error {
	s.pl = nil
	return nil
}

func (s *vectorScene) Tick(voice *sound.Voice, km keys.Map) (bool, error) {
	s.cycle = (s.cycle + 1) % s.mode.cycleTime
	s.t0++
	if s.cycle != 0 {
		return false, nil
	}
	if s.mode.compute != nil {
		s.pl.SetStatus(s.mode.compute(s))
	}
	return true, nil
}

func (s *vectorScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.pl.Draw(t, 1.0, scale)
	})
	return nil
}
