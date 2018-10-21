package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"

	"github.com/seebs/gogetopt"
)

const (
	screenWidth  = 1280
	screenHeight = 960
)

// A Renderable is a thing, like a spiral or a grid, which has
// logic for drawing itself on, for instance, a screen.
type Renderable interface {
	Update()
	Draw(*ebiten.Image)
}

// Grid represents a grid of squares.
type Grid struct {
	Width, Height int
	Squares       [][]Paint
	Palette       *Palette
	X             int
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

func (g *Grid) Inc(l Loc, n int) {
	pt := &g.Squares[l.X][l.Y]
	*pt = g.Palette.Inc(*pt, n)
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
		op.ColorM = g.Palette.ColorM(g.Squares[l.X][l.Y])
		screen.DrawImage(square, op)
	})
}

var (
	square   *ebiten.Image
	op       = &ebiten.DrawImageOptions{}
	grid     Grid
	spiral   *Spiral
	line     *PolyLine
	knights  []Loc
	knight   int
	timedOut <-chan time.Time
	tx       = 640
	ty       = 240
	dtx      = 0
	dty      = 1
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

func squareAt(screen *ebiten.Image, x, y int) {
	op := ebiten.DrawImageOptions{
		SourceRect: &image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{32, 32},
		},
	}
	g := ebiten.GeoM{}
	g.Translate(-16, -16)
	g.Scale(0.5, 0.5)
	g.Translate(float64(x), float64(y))
	op.GeoM = g
	screen.DrawImage(square, &op)
}

var lagCounter = 0
var pause = false

var keyStates [ebiten.KeyMax]byte

const (
	PRESS   = 0x01
	RELEASE = 0x02
	HOLD    = 0x03
)

func pressed(key ebiten.Key) bool {
	return (keyStates[key] & HOLD) == PRESS
}

func released(key ebiten.Key) bool {
	return (keyStates[key] & HOLD) == RELEASE
}

func held(key ebiten.Key) bool {
	return (keyStates[key] & HOLD) == HOLD
}

func update(screen *ebiten.Image) error {
	for i := ebiten.Key(0); i < ebiten.KeyMax; i++ {
		keyStates[i] = (keyStates[i] << 1)
		if ebiten.IsKeyPressed(i) {
			keyStates[i] |= 1
		}
	}
	if released(ebiten.KeyQ) {
		return errors.New("quit requested")
	}
	if pressed(ebiten.KeySpace) {
		pause = !pause
	}
	k := &knights[knight]
	*k = grid.Add(*k, knightMove())
	grid.Inc(*k, 2)
	grid.Neighbors(*k, func(g *Grid, l Loc, p *Paint) {
		g.Inc(l, 1)
	})
	knight = (knight + 1) % len(knights)
	// grid.Draw(screen)
	op := ebiten.DrawImageOptions{
		SourceRect: &image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{32, 32},
		},
	}

	for i := 0; i < 6; i++ {
		for j := 0; j < 4; j++ {
			g := ebiten.GeoM{}
			g.Translate(-16, -16)
			g.Scale(2, 2)
			g.Rotate(float64(j) * math.Pi / 8)
			g.Translate(64*float64(i)+32, 64*float64(j)+32)
			op.GeoM = g
			// screen.DrawImage(square, &op)
		}
	}
	if !pause || released(ebiten.KeyRight) {
		spiral.Update()
	}
	spiral.Draw(screen)
	select {
	case <-timedOut:
		return errors.New("regular termination")
	default:
		return nil
	}
}

func main() {
	opts, _, err := gogetopt.GetOpt(os.Args[1:], "ps#")
	if err != nil {
		log.Fatalf("option parsing failed: %s\n", err)
	}
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
	grid = NewGrid(80, 60)
	grid.Palette = Palettes["rainbow"]
	grid.Iterate(func(g *Grid, l Loc, p *Paint) {
		g.Squares[l.X][l.Y] = g.Palette.Paint(0)
	})
	for i := 0; i < 6; i++ {
		knights = append(knights, grid.NewLoc())
	}
	square, _, err = ebitenutil.NewImageFromFile("square.png", ebiten.FilterLinear)
	if err != nil {
		log.Fatal(err)
	}
	NewSprite("indented", square, image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 32, Y: 32}})
	NewSprite("white", square, image.Rectangle{Min: image.Point{X: 32, Y: 0}, Max: image.Point{X: 64, Y: 32}})
	spiral = NewSpiral(11, 360, Palettes["rainbow"], 3)
	spiral.Center = MovingPoint{Loc: Point{X: float64(screenWidth) / 2, Y: float64(screenHeight) / 2}}
	spiral.Target = MovingPoint{Loc: Point{X: screenWidth, Y: screenHeight}, Velocity: Point{X: 15, Y: 15}}
	spiral.Target.SetBounds(screenWidth, screenHeight)
	spiral.Theta = 8 * math.Pi
	spiral.Step = 2
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
