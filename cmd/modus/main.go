package main

import (
	"errors"
	"fmt"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"seebs.net/modus/g"

	"github.com/hajimehoshi/ebiten"

	"github.com/seebs/gogetopt"
)

const (
	screenWidth  = 1280
	screenHeight = 960
)

var (
	gctx     *g.Context
	grid     *g.SquareGrid
	hg       *g.HexGrid
	line     *g.PolyLine
	spirals  []*g.Spiral
	knights  []g.ILoc
	knight   int
	timedOut <-chan time.Time
	tx       = 640
	ty       = 240
	dtx      = 0
	dty      = 1
	num      = 20
	voice    *Voice
	splat    *ebiten.Image
)

var knightMoves = []g.IVec{
	{-2, -1},
	{-1, -2},
	{1, -2},
	{2, -1},
	{2, 1},
	{1, 2},
	{-1, 2},
	{-2, 1},
}

func knightMove() g.IVec {
	return knightMoves[int(rand.Int31n(int32(len(knightMoves))))]
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

var frames = 0
var tps float64
var tpsStarted bool
var sound = true
var px, py int
var prevLocs []g.ILoc

func update(screen *ebiten.Image) error {
	cTPS := ebiten.CurrentTPS()
	if cTPS > 0 {
		tpsStarted = true
	}
	if tpsStarted {
		tps += cTPS
		frames++
	}
	keys.Update()

	if keys.Released(ebiten.KeyQ) {
		return errors.New("quit requested")
	}
	if keys.Pressed(ebiten.KeySpace) {
		pause = !pause
	}

	if !pause || keys.Released(ebiten.KeyRight) {
		grid.Iterate(func(gr g.Grid, l g.ILoc, c *g.Cell) {
			c.IncAlpha(-0.001)
		})
		k := &knights[knight]
		*k = grid.Add(*k, knightMove())
		grid.IncP(*k, 2)
		grid.IncAlpha(*k, 0.2)
		grid.Splash(*k, 1, 1, func(gr g.Grid, l g.ILoc, c *g.Cell) {
			gr.IncP(l, 1)
			c.IncAlpha(0.1)
		})
		hg.Iterate(func(gr g.Grid, l g.ILoc, c *g.Cell) {
			c.IncAlpha(-0.001)
		})

		knight = (knight + 1) % len(knights)
		for idx, s := range spirals {
			bounced, note, loc := s.Update()
			if bounced && sound {
				voice.Play(note+5*idx, 90)
			}
			hl, hcell := hg.CellAt(int(loc.X), int(loc.Y))
			if hcell != nil {
				if hl != prevLocs[idx] {
					prevLocs[idx] = hl
					hcell.P = g.Paint(note)
					hcell.Alpha = 1
				}
			}
		}
	}

	gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		grid.Draw(t, scale)
		// hg.Draw(t, scale)
		for _, s := range spirals {
			s.Draw(t, scale)
		}
	})

	select {
	case <-timedOut:
		return errors.New("regular termination")
	default:
		return nil
	}
}

func main() {
	opts, _, err := gogetopt.GetOpt(os.Args[1:], "amn#pPqs#")
	if err != nil {
		log.Fatalf("option parsing failed: %s\n", err)
	}
	if opts.Seen("P") {
		pause = true
	}
	if opts.Seen("p") {
		f, err := os.Create("cpu-profile.dat")
		if err != nil {
			log.Fatalf("can't create cpu-profile.dat: %s", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if opts.Seen("q") {
		sound = false
	}
	if opts.Seen("n") {
		num = opts["n"].Int
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
	gctx = g.NewContext(screenWidth, screenHeight, opts.Seen("a"))
	grid = gctx.NewSquareGrid(40, 1)
	grid.Palette = g.Palettes["rainbow"]
	hg = gctx.NewHexGrid(num, 0)
	hg.Palette = g.Palettes["rainbow"]

	grid.Iterate(func(generic g.Grid, l g.ILoc, c *g.Cell) {
		gr := generic.(*g.SquareGrid)
		gr.Squares[l.X][l.Y].P = gr.Palette.Paint(3)
	})
	for i := 0; i < 1; i++ {
		knights = append(knights, g.ILoc{X: 10, Y: 10})
	}
	for i := 0; i < 3; i++ {
		spiral := gctx.NewSpiral(11, 1, 400, g.Palettes["rainbow"], 3, i*2)
		spiral.Center = g.MovingPoint{Loc: g.Point{X: float32(screenWidth) / 2, Y: float32(screenHeight) / 2}}
		spiral.Target = g.MovingPoint{Loc: g.Point{X: rand.Float32() * screenWidth, Y: rand.Float32() * screenHeight}, Velocity: g.Vec{X: rand.Float32()*30 - 15, Y: rand.Float32()*30 - 15}}
		spiral.Target.SetBounds(screenWidth, screenHeight)
		spiral.Theta = 8 * math.Pi
		spiral.Step = 2
		spirals = append(spirals, spiral)
		prevLocs = append(prevLocs, g.ILoc{})
	}
	voice, err = NewVoice("breath", 8)
	if err = ebiten.Run(update, screenWidth, screenHeight, 1, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "frames: %d, TPS %.2f\n", frames, tps/float64(frames))
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
