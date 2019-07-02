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
	"seebs.net/modus/keys"
	"seebs.net/modus/modes"
	"seebs.net/modus/sound"

	"github.com/hajimehoshi/ebiten"

	"github.com/seebs/gogetopt"
)

var (
	gctx         *g.Context
	timedOut     <-chan time.Time
	tx           = 640
	ty           = 240
	dtx          = 0
	dty          = 1
	num          = 20
	voice        *sound.Voice
	allModes     []modes.Mode
	currentMode  int
	scene        modes.Scene
	step         bool
	screenWidth  = 1280
	screenHeight = 960
)

var pause = false

var km = keys.NewMap(ebiten.KeyA, ebiten.KeyD, ebiten.KeyQ, ebiten.KeyS, ebiten.KeyW, ebiten.KeyPeriod, ebiten.KeySpace, ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyUp)

var frames = 0
var tps float64
var tpsStarted bool
var useSound = true

func update(screen *ebiten.Image) error {
	cTPS := ebiten.CurrentTPS()
	if cTPS > 0 {
		tpsStarted = true
	}
	if tpsStarted {
		tps += cTPS
		frames++
	}
	km.Update()

	if km.Released(ebiten.KeyQ) {
		return errors.New("quit requested")
	}
	if km.Pressed(ebiten.KeySpace) {
		pause = !pause
	}
	if km.Pressed(ebiten.KeyUp) {
		err := newMode()
		if err != nil {
			return err
		}
	}
	if km.Released(ebiten.KeyRight) {
		step = true
	}

	if !pause || step {
		stepped, err := scene.Tick(voice, km)
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
	opts, _, err := gogetopt.GetOpt(os.Args[1:], "amn#pPqs#x#y#")
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
				fmt.Fprintf(os.Stderr, "can't create heap-profile.dat: %s\n", err)
			} else {
				pprof.Lookup("heap").WriteTo(f, 0)
			}
			f, err = os.Create("alloc-profile.dat")
			if err != nil {
				fmt.Fprintf(os.Stderr, "can't create alloc-profile.dat: %s\n", err)
			} else {
				pprof.Lookup("allocs").WriteTo(f, 0)
			}
		}()
	}
	if opts.Seen("s") {
		timedOut = time.After(time.Duration(opts["s"].Int) * time.Second)
	}
	if opts.Seen("x") != opts.Seen("y") {
		fmt.Fprintf(os.Stderr, "x and y must be used together\n")
	}
	if opts.Seen("x") && opts.Seen("y") {
		screenWidth = opts["x"].Int
		screenHeight = opts["y"].Int
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
