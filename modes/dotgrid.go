package modes

import (
	"fmt"

	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/sound"

	math "github.com/chewxy/math32"
)

// knightMode is one of the internal modes based on knight moves
type dotGridMode struct {
	cycleTime   int // number of ticks to go by between updates
	compute     func(*dotGridScene, [][]g.DotGridBase, [][]g.DotGridState, [][]g.DotGridState) string
	computeInit func(*dotGridScene, [][]g.DotGridBase, [][]g.DotGridState)
	detail      func(int) int
	name        string
	depth       int
}

const dotGridCycleTime = 1

var dotGridModes = []dotGridMode{
	{name: "gravity", depth: 2, cycleTime: dotGridCycleTime, compute: gravityCompute, computeInit: gravityComputeInit, detail: gravityDetail},
	{name: "distance", depth: 8, cycleTime: dotGridCycleTime, compute: distanceCompute},
	{name: "boring", depth: 8, cycleTime: dotGridCycleTime, compute: boringCompute},
}

func (m *dotGridMode) Detail(base int) int {
	if m != nil && m.detail != nil {
		return m.detail(base)
	}
	return base * 4
}

// type DotCompute func(base [][]DotGridBase, prev [][]DotGridState, next [][]DotGridState) string

func reallyBoringCompute(s *dotGridScene, base [][]g.DotGridBase, prev [][]g.DotGridState, next [][]g.DotGridState) string {
	for i := range base {
		for j := range base[i] {
			b := &base[i][j]
			next[i][j] = g.DotGridState{X: b.X, Y: b.Y, P: 0, A: 1, S: 1}
		}
	}
	return ""
}

// gravity has to be cautious
func gravityDetail(base int) int {
	out := base * 2
	if out > 44 {
		out = 44
	}
	return out
}

// let's do... gravity!
func gravityCompute(s *dotGridScene, base [][]g.DotGridBase, prev [][]g.DotGridState, next [][]g.DotGridState) string {
	cshift := s.t0 / 256
	factor := float32(len(base) * len(base[0]))
	computed := 0
	t := (s.t0 / 60)
	s.cx, s.cy = math.Sincos(t)
	s.cx /= 4
	s.cy /= 4
	cx := make([]float32, 4)
	cy := make([]float32, 4)
	cx[0], cy[0] = s.cx, s.cy
	cx[1], cy[1] = s.cy, -s.cx
	cx[2], cy[2] = -s.cx, -s.cy
	cx[3], cy[3] = -s.cy, s.cx
	cidx := 0
	ncx, ncy := -s.cx, -s.cy
	for i := len(base) - 1; i >= 0; i-- {
		for j := len(base[i]) - 1; j >= 0; j-- {
			var myCx, myCy float32
		inner:
			for k := range base {
				for l := range base[k] {
					if k == i && l == k {
						break inner
					}
					computed++
					dx, dy := prev[k][l].X-prev[i][j].X, prev[k][l].Y-prev[i][j].Y
					if dx != 0 || dy != 0 {
						gscale := dx*dx + dy*dy
						if gscale < 0.01 {
							gscale = 0.01
						}
						gscale += 0.1 + s.pulse
						gscale *= factor * 5000
						dx, dy = dx/gscale, dy/gscale
						base[i][j].DX += dx
						base[i][j].DY += dy
						base[k][l].DX -= dx
						base[k][l].DY -= dy
					} else {
						// if things have the same location, nudge them apart
						base[i][j].DX += 0.0001
						base[i][j].DY += 0.0001
						base[k][l].DX -= 0.0001
						base[k][l].DY -= 0.0001
					}
				}
			}
			// pull things towards nominal center
			if (i+j)&1 == 1 {
				myCx, myCy = s.cx, s.cy
				cidx = 1
			} else {
				myCx, myCy = ncx, ncy
				cidx = 3
			}
			dx, dy := prev[i][j].X-myCx, prev[i][j].Y-myCy
			dist := math.Sqrt(dx*dx + dy*dy)
			speed := math.Sqrt(base[i][j].DX*base[i][j].DX + base[i][j].DY*base[i][j].DY)
			// damping factor: push towards center of screen
			base[i][j].DX -= dx / 10000
			base[i][j].DY -= dy / 10000
			next[i][j].X = prev[i][j].X + base[i][j].DX
			next[i][j].Y = prev[i][j].Y + base[i][j].DY
			// made it quite a ways off screen... move to your center and emit
			if dist > 2 {
				dirx, diry := cx[cidx], cy[cidx]
				// cidx = (cidx + 1) % 4
				// since cx = sin t, cy = cos t, the center is moving
				// in the direction of their derivatives... which are
				// cos t and -sin t, respectively.
				base[i][j].DX = (base[i][j].DX * 0.05) + (dirx * .05)
				base[i][j].DY = (base[i][j].DY * 0.05) + (diry * .05)
				next[i][j].X = myCx
				next[i][j].Y = myCy
			}
			sinv := 1 - (speed * 30)
			if sinv < 0.05 {
				sinv = 0.05
			}
			if (i+j)&1 == 1 {
				next[i][j].P = g.Paint(int(speed*900+cshift) - 10)
			} else {
				next[i][j].P = g.Paint(int(speed*900+cshift) + 26)
			}
			next[i][j].A = 1
			next[i][j].S = sinv

		}
	}
	return fmt.Sprintf("%d computed. [0][0]: dx/dy %.3f,%.3f, %.3f,%.3f -> %.3f,%.3f",
		computed,
		base[0][0].DX, base[0][0].DY,
		prev[0][0].X, prev[0][0].Y,
		next[0][0].X, next[0][0].Y)
}

func gravityComputeInit(s *dotGridScene, base [][]g.DotGridBase, init [][]g.DotGridState) {
	for i := range base {
		for j := range base[i] {
			init[i][j].A = 1
			init[i][j].S = 1
			init[i][j].P = g.Paint(0)
			init[i][j].X = base[i][j].X
			init[i][j].Y = base[i][j].Y
		}
	}
}

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
			dist := math.Sqrt(dx*dx + dy*dy)
			p := s.palette.Paint(int(dist*2400 + s.t0/16))
			scale := math.Abs(x0) + math.Abs(y0) + (s.t0 / 64)
			scale = math.Mod(scale, 2)
			if scale > 1 {
				scale = 2 - scale
			}
			scale -= dist * 5
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
				t = distance*(s.pulse-0.5) - (s.t0 / 100)
			} else {
				t = (math.Abs(distance-0.5) * 2 * ((s.pulse - 0.5) * 2)) + (s.t0 / 80)
			}
			sin, cos := math.Sincos(t)
			x := cos*x0 + sin*y0
			y := cos*y0 - sin*x0
			dx, dy := x-old.X, y-old.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			p := s.palette.Paint(int(dist*1800 + (s.t0 / 100) + (distance * 30)))
			scale := math.Abs(x0) + math.Abs(y0) + (s.t0 / 64)
			scale = math.Mod(scale, 2)
			if scale > 1 {
				scale = 2 - scale
			}
			scale -= (dist * 2)
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
	cx, cy   float32
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
	s.gr = s.gctx.NewDotGrid(s.mode.Detail(s.detail), 8, s.mode.depth, 1, s.palette)
	s.gr.Compute = func(base [][]g.DotGridBase, prev [][]g.DotGridState, next [][]g.DotGridState) string {
		return s.mode.compute(s, base, prev, next)
	}
	if s.mode.computeInit != nil {
		s.gr.ComputeInit = func(base [][]g.DotGridBase, init [][]g.DotGridState) {
			s.mode.computeInit(s, base, init)
		}
	}
	s.gr.InitCompute()
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
