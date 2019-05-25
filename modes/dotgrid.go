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
	compute   func(*dotGridScene, [][]g.DotGridBase, [][]g.DotGridState, [][]g.DotGridState) string
	name      string
	depth     int
}

const dotGridCycleTime = 1

var dotGridModes = []dotGridMode{
	{name: "distance", depth: 8, cycleTime: dotGridCycleTime, compute: distanceCompute},
	{name: "boring", depth: 8, cycleTime: dotGridCycleTime, compute: boringCompute},
}

// type DotCompute func(base [][]DotGridBase, prev [][]DotGridState, next [][]DotGridState) string

func boringCompute(s *dotGridScene, base [][]g.DotGridBase, prev [][]g.DotGridState, next [][]g.DotGridState) string {
	pulse := s.pulse
	pinv := s.pinv
	if pulse > 0.95 {
		pulse = 0.95
		pinv = 1 - pulse
	}
	if pulse < 0.05 {
		pulse = 0.05
		pinv = 1 - pulse
	}
	for i := range base {
		for j := range base[i] {
			b := &base[i][j]
			old := &prev[i][j]
			x0, y0 := b.X, b.Y
			t := (y0*float32(s.gr.H)*float32(s.gr.W) + x0*float32(s.gr.W)) + (s.pcycle * math.Pi * 2)
			x := pinv*x0 + pulse*math.Sin(t)
			y := pinv*y0 + pulse*math.Cos(t)
			dx, dy := x-old.X, y-old.Y
			p := s.palette.Paint(int(math.Sqrt(dx*dx+dy*dy)*1800 + s.t0/16))
			scale := math.Abs(x0) + math.Abs(y0) + (s.t0 / 64)
			scale = math.Mod(scale, 2)
			if scale > 1 {
				scale = 2 - scale
			}
			if scale < 0.2 {
				scale = 0.2
			}
			next[i][j] = g.DotGridState{X: x, Y: y, P: p, A: 0.7, S: scale}
		}
	}
	return ""
	// return fmt.Sprintf("t0: %.1f pulse: %.2f pinv: %.2f", s.t0, s.pulse, s.pinv)
}

func distanceCompute(s *dotGridScene, base [][]g.DotGridBase, prev [][]g.DotGridState, next [][]g.DotGridState) string {
	pulse := s.pulse
	pinv := s.pinv
	if pulse > 0.95 {
		pulse = 0.95
		pinv = 1 - pulse
	}
	if pulse < 0.05 {
		pulse = 0.05
		pinv = 1 - pulse
	}
	_ = pinv
	for i := range base {
		for j := range base[i] {
			b := &base[i][j]
			old := &prev[i][j]
			x0, y0 := b.X, b.Y
			distance := math.Sqrt(x0*x0 + y0*y0)
			var t float32
			if distance >= 1 {
				t = distance*(s.pulse/2) - (s.t0 / 300)
			} else {
				t = (math.Abs(distance-0.5) * 2 * (s.pulse * 2)) + (s.t0 / 192)
			}
			sin, cos := math.Sincos(t)
			x := cos*x0 + sin*y0
			y := cos*y0 - sin*x0
			dx, dy := x-old.X, y-old.Y
			p := s.palette.Paint(int(math.Sqrt(dx*dx+dy*dy)*1800 + (s.t0 / 100) + (distance * 30)))
			scale := math.Abs(x0) + math.Abs(y0) + (s.t0 / 64)
			scale = math.Mod(scale, 2)
			if scale > 1 {
				scale = 2 - scale
			}
			if scale < 0.2 {
				scale = 0.2
			}
			next[i][j] = g.DotGridState{X: x, Y: y, P: p, A: 0.7, S: scale}
		}
	}
	return ""
	// return fmt.Sprintf("t0: %.1f pulse: %.2f pinv: %.2f", s.t0, s.pulse, s.pinv)
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
	s.palette = p.Interpolate(12)
	err := s.Display()
	if err != nil {
		return err
	}
	// and then reset the grid, and the knights, i guess?
	return nil
}

func (s *dotGridScene) Display() error {
	s.gr = s.gctx.NewDotGrid(s.detail*4, 8, s.mode.depth, 1, s.palette)
	s.gr.Compute = func(base [][]g.DotGridBase, prev [][]g.DotGridState, next [][]g.DotGridState) string {
		return s.mode.compute(s, base, prev, next)
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
	if s.pcycleCt >= 128 {
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
