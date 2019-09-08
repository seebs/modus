package modes

import (
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/keys"
	"seebs.net/modus/sound"
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
	k         int // knights
	cycleTime int // number of ticks to go by between updates
}

const knightCycleTime = 10

var knightModes = []knightMode{
	{k: 1, cycleTime: knightCycleTime},
	{k: 2, cycleTime: knightCycleTime},
	{k: 3, cycleTime: knightCycleTime},
	{k: 4, cycleTime: knightCycleTime},
	{k: 5, cycleTime: knightCycleTime},
	{k: 6, cycleTime: knightCycleTime},
}

func init() {
	for _, mode := range knightModes {
		defaultList.Add(mode)
	}
}

func (m knightMode) Name() string {
	return fmt.Sprintf("knights%d", m.k)
}

func (m knightMode) Description() string {
	return fmt.Sprintf("%d knights jumping", m.k)
}

func (m knightMode) New(gctx *g.Context, detail int, p *g.Palette) (Scene, error) {
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
	mode       knightMode
	knights    []knight
	detail     int
	gr         *g.SquareGrid
	cycle      int
}

func newKnightScene(m knightMode, gctx *g.Context, detail int, p *g.Palette) (*knightScene, error) {
	sc := &knightScene{mode: m, gctx: gctx, detail: detail, palette: p, knights: make([]knight, m.k)}
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
	p := s.gr.Palette().Paint(0)
	s.gr.Iterate(func(generic g.Grid, l g.ILoc, n int, c *g.Cell) {
		c.P = p
	})
	for i := 0; i < s.mode.k; i++ {
		c := s.gr.NewExtraCell().(*g.FloatingCellBase)
		c.Cell.Scale = 0.7
		l := s.gr.NewLoc()
		*c.Loc() = g.FLoc{X: float32(l.X), Y: float32(l.Y)}
		c.Cell.Alpha = 0
		s.knights[i].c = c
		s.knights[i].P = g.Paint(i)
		s.knights[i].apply()
	}
	return nil
}

func (s *knightScene) Hide() error {
	for i := 0; i < s.mode.k; i++ {
		s.knights[i].c = nil
	}
	s.gr = nil
	return nil
}

func (s *knightScene) Tick(voice *sound.Voice, km keys.Map) (bool, error) {
	s.cycle = (s.cycle + 1) % s.mode.cycleTime
	if s.cycle != 0 {
		return false, nil
	}
	s.gr.Iterate(func(gr g.Grid, l g.ILoc, n int, c *g.Cell) {
		c.IncAlpha(-0.001)
	})
	k := &s.knights[s.nextKnight]
	k.ILoc, _ = s.gr.Add(k.ILoc, knightMove())
	k.P = s.gr.IncP(k.ILoc, 2)
	k.c.Cell.Alpha = 1
	k.apply()
	s.gr.IncAlpha(k.ILoc, 0.2)
	voice.Play(int(s.gr.Cells[k.X][k.Y].P), 75)
	s.gr.Splash(k.ILoc, 1, 1, func(gr g.Grid, l g.ILoc, n int, c *g.Cell) {
		gr.IncP(l, 1)
		c.IncAlpha(0.1)
	})
	s.nextKnight = (s.nextKnight + 1) % s.mode.k
	return true, nil
}

func (s *knightScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.gr.Draw(t, scale)
	})
	return nil
}
