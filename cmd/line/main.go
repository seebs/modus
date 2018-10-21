package main

import (
	"errors"
	"fmt"
	"log"
	"math"
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
	line     *g.PolyLine
	timedOut <-chan time.Time
	tx       = 640
	ty       = 240
	dtx      = 0
	dty      = 1
	theta    = 0.0
)

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
	if held(ebiten.KeyRight) {
		theta += .01
	}
	if held(ebiten.KeyLeft) {
		theta -= .01
	}
	pt1 := line.Point(1)
	pt2 := line.Point(2)
	s, c := math.Sincos(theta)
	pt2.X = pt1.X + (c * 200)
	pt2.Y = pt1.Y + (s * 200)
	pt3 := line.Point(3)
	pt3.Y = pt2.Y
	line.Draw(screen, 1.0)
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
	line = g.NewPolyLine(g.Palettes["rainbow"], 3)
	line.Blend = false
	line.Add(100, 100, 0)
	line.Add(400, 100, 0)
	line.Add(600, 100, 3)
	line.Add(100, 100, 4)
	line.Thickness = 80

	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
