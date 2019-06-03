package main

import (
	"errors"
	"fmt"
	_ "image/png"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"seebs.net/modus/g"
	"seebs.net/modus/modes"
	"seebs.net/modus/sound"

	"github.com/hajimehoshi/ebiten"

	"github.com/seebs/gogetopt"
)

const (
	screenWidth  = 1280
	screenHeight = 960
)

var (
	gctx        *g.Context
	timedOut    <-chan time.Time
	tx          = 640
	ty          = 240
	dtx         = 0
	dty         = 1
	num         = 20
	voice       *sound.Voice
	allModes    []modes.Mode
	currentMode int
	scene       modes.Scene
	step        bool
)

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
	ebiten.KeyLeft:  0,
	ebiten.KeyRight: 0,
	ebiten.KeyUp:    0,
}

var frames = 0
var tps float64
var tpsStarted bool
var useSound = true
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
	if keys.Pressed(ebiten.KeyUp) {
		err := newMode()
		if err != nil {
			return err
		}
	}
	if keys.Released(ebiten.KeyRight) {
		step = true
	}

	if !pause || step {
		stepped, err := scene.Tick(voice)
		if stepped {
			step = false
		}
		if err != nil {
			return err
		}
	}
	err := scene.Draw(screen)
	if err != nil {
		return err
	}

	select {
	case <-timedOut:
		return errors.New("regular termination")
	default:
		return nil
	}
}

func newMode() error {
	currentMode = (currentMode + 1) % len(allModes)
	mode := allModes[currentMode]
	fmt.Printf("new mode: %s\n", mode.Name())
	if scene != nil {
		scene.Hide()
		scene = nil
	}
	var err error
	scene, err = mode.New(gctx, num, g.Palettes["rainbow"])
	return err
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
		useSound = false
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
	allModes = modes.ListModes()
	currentMode = -1
	err = newMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "scene error: %v\n", err)
		os.Exit(1)
	}
	if useSound {
		voice, err = sound.NewVoice("breath", 8)
		if err != nil {
			fmt.Fprintf(os.Stderr, "voice error: %v\n", err)
			os.Exit(1)
		}
	}
	if err = ebiten.Run(update, screenWidth, screenHeight, 1, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "frames: %d, TPS %.2f\n", frames, tps/float64(frames))
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
