package modes

import (
	"fmt"

	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/keys"
	"seebs.net/modus/sound"

	math "github.com/chewxy/math32"
)

// knightMode is one of the internal modes based on knight moves
type dotGridMode struct {
	cycleTime   int // number of ticks to go by between updates
	compute     func(int, int, *dotGridScene, g.DotGridBase, g.DotGridState, g.DotGridState) string
	computeInit func(int, int, *dotGridScene, g.DotGridBase, g.DotGridState)
	detail      func(int) int
	name        string
	depth       int
}

const dotGridCycleTime = 1

var dotGridModes = []dotGridMode{
	{name: "gravity", depth: 8, cycleTime: dotGridCycleTime, compute: gravityCompute, computeInit: gravityComputeInit, detail: gravityDetail},
	{name: "distance", depth: 8, cycleTime: dotGridCycleTime, compute: distanceCompute},
	{name: "boring", depth: 8, cycleTime: dotGridCycleTime, compute: boringCompute},
	{name: "reallyBoring", depth: 8, cycleTime: dotGridCycleTime, compute: reallyBoringCompute},
}

func (m *dotGridMode) Detail(base int) int {
	if m != nil && m.detail != nil {
		return m.detail(base)
	}
	return base * 4
}

func reallyBoringCompute(w, h int, s *dotGridScene, base g.DotGridBase, prev g.DotGridState, next g.DotGridState) string {
	for i := range base.Locs {
		next.Locs[i] = base.Locs[i]
		next.S[i] = 1
		next.A[i] = 1
		next.P[i] = 0
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
func gravityCompute(w, h int, s *dotGridScene, base g.DotGridBase, prev g.DotGridState, next g.DotGridState) string {
	cshift := s.t0 / 256
	factor := float32(w*h) * 5000
	gScaleMod := 0.1 + s.pulse
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
	rowCount := 0
	alternate := 0
	for idx := len(base.Locs) - 1; idx >= 0; idx-- {
		rowCount++
		if rowCount == w {
			alternate = 1 - alternate
			rowCount = 0
		}
		var myCx, myCy float32
		px, py := prev.Locs[idx].X, prev.Locs[idx].Y
		pSub := prev.Locs[:idx]
		bSub := base.Vecs[:idx]
		bDX, bDY := base.Vecs[idx].X, base.Vecs[idx].Y
		for kidx := range pSub {
			dx, dy := pSub[kidx].X-px, pSub[kidx].Y-py
			gscale := dx*dx + dy*dy
			if gscale < 0.01 {
				gscale = 0.01
			}
			gscale = (gscale + gScaleMod) * factor
			dx, dy = dx/gscale, dy/gscale
			bDX += dx
			bDY += dy
			bSub[kidx].X -= dx
			bSub[kidx].Y -= dy
		}
		computed += idx
		// pull things towards nominal center
		if (idx+alternate)&1 == 1 {
			myCx, myCy = s.cx, s.cy
			cidx = 1
		} else {
			myCx, myCy = ncx, ncy
			cidx = 3
		}
		dx, dy := px-myCx, py-myCy
		dist2 := dx*dx + dy*dy
		speed := math.Sqrt(bDX*bDX + bDY*bDY)
		// damping factor: push towards center of screen
		bDX -= dx / 10000
		bDY -= dy / 10000
		next.Locs[idx].X = px + bDX
		next.Locs[idx].Y = py + bDY
		// made it quite a ways off screen... move to your center and emit
		if dist2 > 4 {
			dirx, diry := cx[cidx], cy[cidx]
			// cidx = (cidx + 1) % 4
			// since cx = sin t, cy = cos t, the center is moving
			// in the direction of their derivatives... which are
			// cos t and -sin t, respectively.
			bDX = (bDX * 0.05) + (dirx * .05)
			bDY = (bDY * 0.05) + (diry * .05)
			next.Locs[idx].X = myCx
			next.Locs[idx].Y = myCy
		}
		sinv := 1 - (speed * 30)
		if sinv < 0.05 {
			sinv = 0.05
		}
		if (idx+alternate)&1 == 1 {
			next.P[idx] = g.Paint(int(speed*900+cshift) - 10)
		} else {
			next.P[idx] = g.Paint(int(speed*900+cshift) + 26)
		}
		next.A[idx] = 1
		next.S[idx] = sinv
		base.Vecs[idx].X, base.Vecs[idx].Y = bDX, bDY
	}
	return fmt.Sprintf("%d computed. [0][0]: dx/dy %.3f,%.3f, %.3f,%.3f -> %.3f,%.3f",
		computed,
		base.Vecs[0].X, base.Vecs[0].Y,
		prev.Locs[0].X, prev.Locs[0].Y,
		next.Locs[0].X, next.Locs[0].Y)
}

func gravityComputeInit(w, h int, s *dotGridScene, base g.DotGridBase, init g.DotGridState) {
	for idx := range base.Locs {
		init.A[idx] = 1
		init.S[idx] = 1
		init.P[idx] = g.Paint(0)
		init.Locs[idx] = base.Locs[idx]
	}
}

func boringCompute(w, h int, s *dotGridScene, base g.DotGridBase, prev g.DotGridState, next g.DotGridState) string {
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
	for idx := range base.Locs {
		old := prev.Locs[idx]
		x0, y0 := base.Locs[idx].X, base.Locs[idx].Y
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
		next.A[idx] = 0.7
		next.S[idx] = scale
		next.P[idx] = p
		next.Locs[idx] = g.FLoc{X: x, Y: y}
	}
	return ""
	// return fmt.Sprintf("t0: %.1f pulse: %.2f pinv: %.2f", s.t0, s.pulse, s.pinv)
}

func distanceCompute(w, h int, s *dotGridScene, base g.DotGridBase, prev g.DotGridState, next g.DotGridState) string {
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
	for idx := range base.Locs {
		old := prev.Locs[idx]
		x0, y0 := base.Locs[idx].X, base.Locs[idx].Y
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
		next.A[idx] = 0.7
		next.S[idx] = scale
		next.P[idx] = p
		next.Locs[idx] = g.FLoc{X: x, Y: y}
	}
	return ""
	// return fmt.Sprintf("t0: %.1f pulse: %.2f pinv: %.2f", s.t0, s.pulse, s.pinv)
}

func init() {
	for _, mode := range dotGridModes {
		defaultList.Add(mode)
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
	s.gr.Compute = func(w, h int, base g.DotGridBase, prev g.DotGridState, next g.DotGridState) string {
		return s.mode.compute(w, h, s, base, prev, next)
	}
	if s.mode.computeInit != nil {
		s.gr.ComputeInit = func(w, h int, base g.DotGridBase, init g.DotGridState) {
			s.mode.computeInit(w, h, s, base, init)
		}
	}
	s.gr.InitCompute()
	return nil
}

func (s *dotGridScene) Hide() error {
	s.gr = nil
	return nil
}

func (s *dotGridScene) Tick(voice *sound.Voice, km keys.Map) (bool, error) {
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
	if km.Released(ebiten.KeyLeft) {
		s.gr.Render = (s.gr.Render + 1) % 2
	}
	s.gr.Tick()
	return true, nil
}

func (s *dotGridScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.gr.Draw(t, scale)
	})
	return nil
}
