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
	"seebs.net/modus/g"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"

	"github.com/seebs/gogetopt"
)

const (
	screenWidth  = 1280
	screenHeight = 960
)

var (
	square   *ebiten.Image
	op       = &ebiten.DrawImageOptions{}
	grid     g.Grid
	spirals  []*g.Spiral
	line     *g.PolyLine
	knights  []g.Loc
	knight   int
	timedOut <-chan time.Time
	tx       = 640
	ty       = 240
	dtx      = 0
	dty      = 1
)

var knightMoves = []g.Mov{
	{-2, -1},
	{-1, -2},
	{1, -2},
	{2, -1},
	{2, 1},
	{1, 2},
	{-1, 2},
	{-2, 1},
}

func knightMove() g.Mov {
	return knightMoves[int(rand.Int31n(int32(len(knightMoves))))]
}

func squareAt(screen *ebiten.Image, x, y int) {
	op := ebiten.DrawImageOptions{
		SourceRect: &image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{32, 32},
		},
	}
	geo := ebiten.GeoM{}
	geo.Translate(-16, -16)
	geo.Scale(0.5, 0.5)
	geo.Translate(float64(x), float64(y))
	op.GeoM = geo
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

var frames = 0
var ttps = 0.0

func update(screen *ebiten.Image) error {
	tps := ebiten.CurrentTPS()
	if tps > 0 {
		frames++
		ttps += tps
	}
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
	grid.Neighbors(*k, func(gr *g.Grid, l g.Loc, p *g.Paint) {
		gr.Inc(l, 1)
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
		for _, s := range spirals {
			s.Update()
		}
	}
	for _, s := range spirals {
		s.Draw(screen)
	}
	if frames > 1000 {
		return fmt.Errorf("%.2f TPS average", ttps / float64(frames))
	}
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
	square, _, err = ebitenutil.NewImageFromFile("square.png", ebiten.FilterLinear)
	if err != nil {
		log.Fatal(err)
	}
	grid = g.NewGrid(80, 60, square)
	grid.Palette = g.Palettes["rainbow"]
	grid.Iterate(func(gr *g.Grid, l g.Loc, p *g.Paint) {
		gr.Squares[l.X][l.Y] = gr.Palette.Paint(0)
	})
	for i := 0; i < 6; i++ {
		knights = append(knights, grid.NewLoc())
	}
	g.NewSprite("indented", square, image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 32, Y: 32}})
	g.NewSprite("white", square, image.Rectangle{Min: image.Point{X: 32, Y: 0}, Max: image.Point{X: 64, Y: 32}})
	for i := 0; i < 3; i++ {
		spiral := g.NewSpiral(11, 1600, g.Palettes["rainbow"], 3, 66 * i)
		spiral.Center = g.MovingPoint{Loc: g.Point{X: float64(screenWidth) / 2, Y: float64(screenHeight) / 2}}
		spiral.Target = g.MovingPoint{Loc: g.Point{X: rand.Float64() * screenWidth, Y: rand.Float64() * screenHeight}, Velocity: g.Point{X: rand.Float64() * 30 - 15, Y: rand.Float64() * 30 - 15}}
		spiral.Target.SetBounds(screenWidth, screenHeight)
		spiral.Theta = 8 * math.Pi
		spiral.Step = 2
		spirals = append(spirals, spiral)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
