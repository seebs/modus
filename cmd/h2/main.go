package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"
	"seebs.net/modus/g"
	"image/color"

	"github.com/hajimehoshi/ebiten"
	// "github.com/hajimehoshi/ebiten/ebitenutil"

	"github.com/seebs/gogetopt"
)

const (
	screenWidth  = 512
	screenHeight = 512
)

var (
	timedOut <-chan time.Time
	solid    *ebiten.Image
	theta    float64
	hg1, hg2	*g.HexGrid
)

var lagCounter = 0
var pause = false

var keyStates [ebiten.KeyMax + 1]byte

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

var (
	h2x, h2y = 324, 240
)

func update(screen *ebiten.Image) error {
	for i := ebiten.Key(0); i <= ebiten.KeyMax; i++ {
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
	anyHeld := false
	if held(ebiten.KeyRight) {
		h2x++
		anyHeld = true
	}
	if held(ebiten.KeyLeft) {
		h2x--
		anyHeld = true
	}
	if held(ebiten.KeyDown) {
		h2y++
		anyHeld = true
	}
	if held(ebiten.KeyUp) {
		h2y--
		anyHeld = true
	}
	if !anyHeld && (released(ebiten.KeyLeft) || released(ebiten.KeyRight) || released(ebiten.KeyDown) || released(ebiten.KeyUp)) {
		fmt.Printf("h2x, h2y: %d, %d\n", h2x, h2y)
	}
	hg1.Draw(screen)
	hg2.Draw(screen)
	select {
	case <-timedOut:
		return errors.New("regular termination")
	default:
		return nil
	}
}

const (
	radius = 255
)

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

	solid, err = ebiten.NewImage(16, 16, ebiten.FilterNearest)
	solid.Fill(color.RGBA{255, 255, 255, 255})
	hg1 = g.NewHexGrid(1)
	hg2 = g.NewHexGrid(2)

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
