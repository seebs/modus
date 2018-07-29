package main

import (
	"fmt"
	"errors"
	"image/color"
	_ "image/png"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"

	"github.com/seebs/gogetopt"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Paint int

var rawPalette = []color.RGBA{
	color.RGBA{255, 0, 0, 255},
	color.RGBA{240, 90, 0, 255},
	color.RGBA{220, 220, 0, 255},
	color.RGBA{0, 200, 0, 255},
	color.RGBA{0, 0, 255, 255},
	color.RGBA{180, 0, 200, 255},
}

func initPalette() {
	for _, c := range rawPalette {
		cm := &ebiten.ColorM{}
		cm.Scale(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 1.0)
		palette = append(palette, cm)
	}
}

var palette []*ebiten.ColorM

func (p Paint) Color() *ebiten.ColorM {
	return palette[int(p)%len(palette)]
}

func (p *Paint) Inc(n int) {
	*p = Paint((int(*p) + n) % len(palette))
}

type Grid struct {
	Width, Height int
	Squares       [][]Paint
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

func NewGrid(width, height int) Grid {
	g := Grid{Width: width, Height: height}
	g.Squares = make([][]Paint, g.Width)
	for idx := range g.Squares {
		g.Squares[idx] = make([]Paint, g.Height)
	}
	return g
}

// A GridFunc is a general callback for operations on the grid.
type GridFunc func(g *Grid, l Loc, p *Paint)

// Iterate runs fn on the entire grid.
func (g *Grid) Iterate(fn GridFunc) {
	for i, col := range grid.Squares {
		for j := range col {
			fn(g, Loc{X: i, Y: j}, &col[j])
		}
	}
}

func (g *Grid) At(l Loc) *Paint {
	return &g.Squares[l.X][l.Y]
}

// Neighbors runs fn on the nearby cells.
func (g *Grid) Neighbors(l Loc, fn GridFunc) {
	for _, m := range []Mov{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		l := g.Add(l, m)
		fn(g, l, g.At(l))
	}
}

func (g *Grid) Draw(screen *ebiten.Image) {
	xscale := screenWidth / grid.Width
	yscale := screenHeight / grid.Height
	grid.Iterate(func(g *Grid, l Loc, _ *Paint) {
		op.GeoM.Reset()
		op.GeoM.Translate(float64(l.X*xscale), float64(l.Y*yscale))
		op.ColorM = *g.Squares[l.X][l.Y].Color()
		screen.DrawImage(square, op)
	})
}

var (
	square  *ebiten.Image
	op      = &ebiten.DrawImageOptions{}
	grid    Grid
	knights []Loc
	knight  int
	timedOut <-chan time.Time
)

var knightMoves = []Mov{
	{-2, -1},
	{-1, -2},
	{1, -2},
	{2, -1},
	{2, 1},
	{1, 2},
	{-1, 2},
	{-2, 1},
}

func knightMove() Mov {
	return knightMoves[int(rand.Int31n(int32(len(knightMoves))))]
}

func update(screen *ebiten.Image) error {
	k := &knights[knight]
	*k = grid.Add(*k, knightMove())
	grid.Squares[k.X][k.Y].Inc(2)
	grid.Neighbors(*k, func(g *Grid, l Loc, p *Paint) {
		p.Inc(1)
	})
	knight = (knight + 1) % len(knights)
	grid.Draw(screen)
	select {
	case <- timedOut:
		return errors.New("regular termination")
	default:
		return nil
	}
}

func main() {
	opts, _, err := gogetopt.GetOpt(os.Args[1:], "ps#")
	if opts.Seen("p") {
		f, err := os.Create("profile.dat")
		if err != nil {
			log.Fatalf("can't create profile.dat: %s", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if opts.Seen("s") {
		timedOut = time.After(time.Duration(opts["s"].Int) * time.Second)
	}
	initPalette()
	grid = NewGrid(80, 60)
	grid.Iterate(func(g *Grid, l Loc, p *Paint) {
		g.Squares[l.X][l.Y] = Paint(0)
	})
	for i := 0; i < 6; i++ {
		knights = append(knights, grid.NewLoc())
	}
	square, _, err = ebitenutil.NewImageFromFile("square.png", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Lights Out?"); err != nil {
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
