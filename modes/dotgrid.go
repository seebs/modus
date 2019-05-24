package modes

import (
	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/sound"

	math "github.com/chewxy/math32"
)

// knightMode is one of the internal modes based on knight moves
type dotGridMode struct {
	cycleTime int // number of ticks to go by between updates
	compute   func(x0, y0 float32, s *dotGridScene) (x, y float32, p g.Paint, a float32, scale float32)
	name      string
	depth     int
}

const dotGridCycleTime = 1

var dotGridModes = []dotGridMode{
	{name: "boring", depth: 5, cycleTime: dotGridCycleTime, compute: boringCompute},
}

func boringCompute(x0, y0 float32, s *dotGridScene) (x, y float32, p g.Paint, a float32, scale float32) {
	p = s.palette.Paint(int(x0*20 + y0*30 + s.t0))
	t := (y0*float32(s.gr.H)*float32(s.gr.W) + x0*float32(s.gr.W)) + (s.pcycle * math.Pi * 2)
	x = s.pinv*x0 + s.pulse*(1+math.Sin(t))/2
	y = s.pinv*y0 + s.pulse*(1+math.Cos(t))/2
	// x = s.pulse*x0 + (1-s.pulse)*math.Sin(x0*2*math.Pi)
	// y = s.pulse*y0 + (1-s.pulse)*math.Cos(y0*2*math.Pi)
	return x, y, p, 1, 0.5
}

func init() {
	for _, mode := range dotGridModes {
		allModes = append(allModes, mode)
	}
}

func (m dotGridMode) Name() string {
	return m.name
}

func (m dotGridMode) Description() string {
	return "dots"
}

func (m dotGridMode) New(gctx *g.Context, detail int, p *g.Palette) (Scene, error) {
	return newDotGridScene(m, gctx, detail, p)
}

type dotGridScene struct {
	gr       *g.DotGrid
	detail   int
	palette  *g.Palette
	gctx     *g.Context
	cycle    int
	mode     dotGridMode
	t0       float32
	pcycleCt int
	pcycle   float32
	pulse    float32
	pinv     float32
}

func newDotGridScene(m dotGridMode, gctx *g.Context, detail int, p *g.Palette) (*dotGridScene, error) {
	sc := &dotGridScene{mode: m, gctx: gctx, detail: detail}
	err := sc.Reset(detail, p)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *dotGridScene) Mode() Mode {
	return s.mode
}

func (s *dotGridScene) Reset(detail int, p *g.Palette) error {
	_ = s.Hide()
	s.palette = p.Interpolate(20)
	err := s.Display()
	if err != nil {
		return err
	}
	// and then reset the grid, and the knights, i guess?
	return nil
}

func (s *dotGridScene) Display() error {
	s.gr = s.gctx.NewDotGrid(s.detail, 16, s.mode.depth, 1, s.palette)
	s.gr.Compute = func(x0, y0 float32) (x, y float32, p g.Paint, a float32, scale float32) {
		return s.mode.compute(x0, y0, s)
	}
	return nil
}

func (s *dotGridScene) Hide() error {
	s.gr = nil
	return nil
}

func (s *dotGridScene) Tick(voice *sound.Voice) (bool, error) {
	s.cycle = (s.cycle + 1) % s.mode.cycleTime
	if s.cycle != 0 {
		return false, nil
	}
	s.t0++
	s.pcycleCt++
	if s.pcycleCt > 128 {
		s.pcycleCt = 0
	}
	s.pcycle = float32(s.pcycleCt) / 128
	s.pulse = (1 + math.Sin(s.t0/128)) / 2
	s.pinv = 1 - s.pulse
	s.gr.Tick()
	return true, nil
}

func (s *dotGridScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.gr.Draw(t, scale)
	})
	return nil
}
