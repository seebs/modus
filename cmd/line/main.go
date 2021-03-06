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

	"github.com/seebs/gogetopt"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
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
	if held(ebiten.KeyF) {
		theta += .01
	}
	if held(ebiten.KeyG) {
		theta -= .01
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
	pt2.X = pt1.X + (c * 100)
	pt2.Y = pt1.Y + (s * 100)
	pt3 := line.Point(3)
	pt3.Y = pt2.Y
	line.Dirty()
	line.Draw(screen, 1.0, 1.0)
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
	gctx := g.NewContext(screenWidth, screenHeight, false)
	line = gctx.NewPolyline(g.Palettes["rainbow"], 1, 1)
	line.Blend = true
	// line.DebugColor = true
	line.Add(50, 50, 0)
	line.Add(200, 50, 0)
	line.Add(300, 50, 3)
	line.Add(50, 50, 4)
	line.Thickness = 40
	line.Joined = true

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
