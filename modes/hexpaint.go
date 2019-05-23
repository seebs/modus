package modes

import (
	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/sound"
)

// knightMode is one of the internal modes based on knight moves
type hexPaintMode struct {
	cycleTime int // number of ticks to go by between updates
}

const hexPaintCycleTime = 10

var hexPaintModes = []hexPaintMode{
	{cycleTime: hexPaintCycleTime},
}

func init() {
	for _, mode := range hexPaintModes {
		allModes = append(allModes, mode)
	}
}

func (m hexPaintMode) Name() string {
	return "hexpaint"
}

func (m hexPaintMode) Description() string {
	return "painting hexes"
}

func (m hexPaintMode) New(gctx *g.Context, detail int, p *g.Palette) (Scene, error) {
	return newHexPaintScene(m, gctx, detail, p)
}

type hexPainter struct {
	g.ILoc
	dir g.HexDir
	P   g.Paint
	c   *g.FloatingHexCell
}

func (p *hexPainter) apply() {
	if p.c != nil {
		p.c.P = p.P
		*p.c.X() = float32(p.X)
		*p.c.Y() = float32(p.Y)
	}
}

type hexPaintScene struct {
	nextPainter int
	palette     *g.Palette
	gctx        *g.Context
	painters    []hexPainter
	detail      int
	gr          *g.HexGrid
	cycle       int
	mode        hexPaintMode
}

func newHexPaintScene(m hexPaintMode, gctx *g.Context, detail int, p *g.Palette) (*hexPaintScene, error) {
	sc := &hexPaintScene{mode: m, gctx: gctx, detail: detail, palette: p, painters: make([]hexPainter, 6)}
	err := sc.Reset(detail, p)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *hexPaintScene) Mode() Mode {
	return s.mode
}

func (s *hexPaintScene) Reset(detail int, p *g.Palette) error {
	_ = s.Hide()
	s.palette = p
	err := s.Display()
	if err != nil {
		return err
	}
	// and then reset the grid, and the knights, i guess?
	return nil
}

func (s *hexPaintScene) Display() error {
	s.gr = s.gctx.NewHexGrid(s.detail, 1, s.palette)
	p := s.gr.Palette().Paint(0)
	s.gr.Iterate(func(generic g.Grid, l g.ILoc, n int, c *g.Cell) {
		c.P = p
	})
	for i := 0; i < 6; i++ {
		c := s.gr.NewExtraCell().(*g.FloatingHexCell)
		c.Cell.Scale = 0.7
		l := s.gr.NewLoc()
		*c.Loc() = g.FLoc{X: float32(l.X), Y: float32(l.Y)}
		c.Cell.Alpha = 0
		s.painters[i].c = c
		s.painters[i].P = g.Paint(i)
		s.painters[i].dir = s.gr.NewDir()
		s.painters[i].apply()
	}
	return nil
}

func (s *hexPaintScene) Hide() error {
	for i := 0; i < 6; i++ {
		s.painters[i].c = nil
	}
	s.gr = nil
	return nil
}

func (s *hexPaintScene) Tick(voice *sound.Voice) (bool, error) {
	s.cycle = (s.cycle + 1) % s.mode.cycleTime
	if s.cycle != 0 {
		return false, nil
	}
	s.gr.Iterate(func(gr g.Grid, l g.ILoc, n int, c *g.Cell) {
		c.IncAlpha(-0.001)
	})
	p := &s.painters[s.nextPainter]
	p.ILoc = s.gr.Add(p.ILoc, p.dir.IVec())
	p.P = s.gr.IncP(p.ILoc, 2)
	p.c.Cell.Alpha = 1
	p.apply()
	s.gr.IncAlpha(p.ILoc, 0.2)
	voice.Play(int(s.gr.Cells[p.X][p.Y].P), 75)
	s.gr.Splash(p.ILoc, 1, 1, func(gr g.Grid, l g.ILoc, n int, c *g.Cell) {
		gr.IncP(l, 1)
		c.IncAlpha(0.1)
	})
	s.nextPainter = (s.nextPainter + 1) % 6
	return true, nil
}

func (s *hexPaintScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.gr.Draw(t, scale)
	})
	return nil
}
