package modes

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
)

var knightMoves = []g.IVec{
	{X: -2, Y: -1},
	{X: -1, Y: -2},
	{X: 1, Y: -2},
	{X: 2, Y: -1},
	{X: 2, Y: 1},
	{X: 1, Y: 2},
	{X: -1, Y: 2},
	{X: -2, Y: 1},
}

func knightMove() g.IVec {
	return knightMoves[int(rand.Int31n(int32(len(knightMoves))))]
}

// knightMode is one of the internal modes based on knight moves
type knightMode struct {
}

var knights1 knightMode

func init() {
	allModes = append(allModes, &knights1)
}

func (m *knightMode) Name() string {
	return "knights"
}

func (m *knightMode) Description() string {
	return "knights jumping"
}

func (m *knightMode) New(gctx *g.Context, detail int, p *g.Palette) (Scene, error) {
	return newKnightScene(m, gctx, detail, p)
}

type knight struct {
	g.ILoc
	P g.Paint
	c *g.FloatingCellBase
}

func (k *knight) apply() {
	if k.c != nil {
		k.c.P = k.P
		*k.c.X() = float32(k.X)
		*k.c.Y() = float32(k.Y)
	}
}

type knightScene struct {
	nextKnight int
	palette    *g.Palette
	gctx       *g.Context
	mode       *knightMode
	knights    []knight
	detail     int
	gr         *g.SquareGrid
}

func newKnightScene(m *knightMode, gctx *g.Context, detail int, p *g.Palette) (*knightScene, error) {
	sc := &knightScene{mode: m, gctx: gctx, detail: detail, palette: p, knights: make([]knight, 6)}
	err := sc.Reset(detail, p)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *knightScene) Mode() Mode {
	return s.mode
}

func (s *knightScene) Reset(detail int, p *g.Palette) error {
	_ = s.Hide()
	s.palette = p
	err := s.Display()
	if err != nil {
		return err
	}
	// and then reset the grid, and the knights, i guess?
	return nil
}

func (s *knightScene) Display() error {
	s.gr = s.gctx.NewSquareGrid(s.detail, 1, s.palette)
	p := s.gr.Palette().Paint(3)
	s.gr.Iterate(func(generic g.Grid, l g.ILoc, n int, c *g.Cell) {
		c.P = p
	})
	for i := 0; i < 6; i++ {
		s.knights[i].c = s.gr.NewExtraCell().(*g.FloatingCellBase)
		s.knights[i].c.Cell.Scale = 0.7
		l := s.gr.NewLoc()
		*s.knights[i].c.Loc() = g.FLoc{X: float32(l.X), Y: float32(l.Y)}
		s.knights[i].apply()
	}
	return nil
}

func (s *knightScene) Hide() error {
	for i := 0; i < 6; i++ {
		s.knights[i].c = nil
	}
	s.gr = nil
	return nil
}

func (s *knightScene) Tick() error {
	s.gr.Iterate(func(gr g.Grid, l g.ILoc, n int, c *g.Cell) {
		c.IncAlpha(-0.001)
	})
	k := &s.knights[s.nextKnight]
	k.ILoc = s.gr.Add(k.ILoc, knightMove())
	s.gr.IncP(k.ILoc, 2)
	s.gr.IncAlpha(k.ILoc, 0.2)
	s.gr.Splash(k.ILoc, 1, 1, func(gr g.Grid, l g.ILoc, n int, c *g.Cell) {
		gr.IncP(l, 1)
		c.IncAlpha(0.1)
	})
	s.nextKnight = (s.nextKnight + 1) % 6
	return nil
}

func (s *knightScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.gr.Draw(t, scale)
	})
	return nil
}
