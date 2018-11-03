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
	voice    *Voice
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

// handle keypresses
const (
	PRESS   = 0x01
	RELEASE = 0x02
	HOLD    = 0x03
)

type keyMap map[ebiten.Key]byte

// State returns the current state of a key
func (km keyMap) State(k ebiten.Key) byte {
	if _, ok := km[k]; !ok {
		km[k] = 0
	}
	return km[k] & HOLD
}

func (km keyMap) Pressed(k ebiten.Key) bool {
	return km.State(k) == PRESS
}

func (km keyMap) Released(k ebiten.Key) bool {
	return km.State(k) == RELEASE
}

func (km keyMap) Held(k ebiten.Key) bool {
	return km.State(k) == HOLD
}

func (km keyMap) Update() {
	for i := range km {
		state := byte(0)
		if ebiten.IsKeyPressed(i) {
			state = 1
		}
		km[i] = ((km[i] & 0x1) << 1) | state
	}
}

var keys = keyMap{
	ebiten.KeyQ:     0,
	ebiten.KeySpace: 0,
	ebiten.KeyRight: 0,
}

func update(screen *ebiten.Image) error {
	keys.Update()

	if keys.Released(ebiten.KeyQ) {
		return errors.New("quit requested")
	}
	if keys.Pressed(ebiten.KeySpace) {
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
	if !pause || keys.Released(ebiten.KeyRight) {
		for idx, s := range spirals {
			if bounced, note := s.Update(); bounced {
				voice.Play(note+5*idx, 90)
			}
		}
	}
	for _, s := range spirals {
		s.Draw(screen)
	}
	select {
	case <-timedOut:
		return errors.New("regular termination")
	default:
		return nil
	}
}

func main() {
	opts, _, err := gogetopt.GetOpt(os.Args[1:], "mps#")
	if err != nil {
		log.Fatalf("option parsing failed: %s\n", err)
	}
	if opts.Seen("p") {
		f, err := os.Create("cpu-profile.dat")
		if err != nil {
			log.Fatalf("can't create cpu-profile.dat: %s", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if opts.Seen("m") {
		defer func() {
			f, err := os.Create("heap-profile.dat")
			if err != nil {
				fmt.Fprintf(os.Stderr, "can't create heap-profile.dat: %s", err)
			} else {
				pprof.Lookup("heap").WriteTo(f, 0)
			}
			f, err = os.Create("alloc-profile.dat")
			if err != nil {
				fmt.Fprintf(os.Stderr, "can't create alloc-profile.dat: %s", err)
			} else {
				pprof.Lookup("allocs").WriteTo(f, 0)
			}
		}()
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
		spiral := g.NewSpiral(11, 400, g.Palettes["rainbow"], 3, i*2)
		spiral.Center = g.MovingPoint{Loc: g.Point{X: float64(screenWidth) / 2, Y: float64(screenHeight) / 2}}
		spiral.Target = g.MovingPoint{Loc: g.Point{X: rand.Float64() * screenWidth, Y: rand.Float64() * screenHeight}, Velocity: g.Point{X: rand.Float64()*30 - 15, Y: rand.Float64()*30 - 15}}
		spiral.Target.SetBounds(screenWidth, screenHeight)
		spiral.Theta = 8 * math.Pi
		spiral.Step = 2
		spirals = append(spirals, spiral)
	}
	voice, err = NewVoice("breath", 8)
	if err = ebiten.Run(update, screenWidth, screenHeight, 1, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
