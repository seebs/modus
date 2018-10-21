package g

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten"
)

// Grid represents a grid of squares.
type Grid struct {
	Width, Height int
	Squares       [][]Paint
	Palette       *Palette
	X             int
	Image         *ebiten.Image
}

func (g *Grid) RandRow() int {
	return int(rand.Int31n(int32(g.Height)))
}

func (g *Grid) RandCol() int {
	return int(rand.Int31n(int32(g.Width)))
}

func (g *Grid) NewLoc() Loc {
	return Loc{X: g.RandCol(), Y: g.RandRow()}
}

func (g *Grid) Add(l Loc, m Mov) Loc {
	return Loc{X: (l.X + m.X + g.Width) % g.Width, Y: (l.Y + m.Y + g.Height) % g.Height}
}

// A Loc represents a location within a grid. (Contrast time.Time.)
type Loc struct {
	X, Y int
}

// A Mov represents movement within a grid. (Contrast time.Duration.)
type Mov struct {
	X, Y int
}

func NewGrid(width, height int, image *ebiten.Image) Grid {
	gr := Grid{Width: width, Height: height}
	gr.Squares = make([][]Paint, gr.Width)
	gr.Image = image
	for idx := range gr.Squares {
		gr.Squares[idx] = make([]Paint, gr.Height)
	}
	return gr
}

// A GridFunc is a general callback for operations on the grid.
type GridFunc func(gr *Grid, l Loc, p *Paint)

// Iterate runs fn on the entire grid.
func (gr *Grid) Iterate(fn GridFunc) {
	for i, col := range gr.Squares {
		for j := range col {
			fn(gr, Loc{X: i, Y: j}, &col[j])
		}
	}
}

func (gr *Grid) At(l Loc) *Paint {
	return &gr.Squares[l.X][l.Y]
}

func (gr *Grid) Inc(l Loc, n int) {
	pt := &gr.Squares[l.X][l.Y]
	*pt = gr.Palette.Inc(*pt, n)
}

// Neighbors runs fn on the nearby cells.
func (gr *Grid) Neighbors(l Loc, fn GridFunc) {
	for _, m := range []Mov{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		l := gr.Add(l, m)
		fn(gr, l, gr.At(l))
	}
}

func (gr *Grid) Draw(screen *ebiten.Image) {
	w, h := screen.Size()
	xscale := w / gr.Width
	yscale := h / gr.Height
	op := &ebiten.DrawImageOptions{}
	gr.Iterate(func(gr *Grid, l Loc, _ *Paint) {
		op.GeoM.Reset()
		op.GeoM.Translate(float64(l.X*xscale), float64(l.Y*yscale))
		op.ColorM = gr.Palette.ColorM(gr.Squares[l.X][l.Y])
		screen.DrawImage(gr.Image, op)
	})
}

