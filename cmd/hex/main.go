package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"runtime/pprof"
	"time"
	// "seebs.net/modus/g"
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
	hex(160, 128, 96, screen)
	hex(h2x, h2y, 96, screen)
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

func rect(x0, y0, x1, y1 float32, screen *ebiten.Image) {
	vertices := make([]ebiten.Vertex, 0, 4)
	indices := make([]uint16, 0, 6)
	vertices = append(vertices, ebiten.Vertex{
		SrcX: 2, SrcY: 2,
		DstX: x0, DstY: y0,
		ColorR: 0.5,
		ColorB: 0.0,
		ColorG: 0.0,
		ColorA: 1.0,
	})
	vertices = append(vertices, ebiten.Vertex{
		SrcX: 2, SrcY: 2,
		DstX: x1, DstY: y0,
		ColorR: 0.5,
		ColorB: 0.0,
		ColorG: 0.0,
		ColorA: 1.0,
	})
	vertices = append(vertices, ebiten.Vertex{
		SrcX: 2, SrcY: 2,
		DstX: x0, DstY: y1,
		ColorR: 0.5,
		ColorB: 0.0,
		ColorG: 0.0,
		ColorA: 1.0,
	})
	vertices = append(vertices, ebiten.Vertex{
		SrcX: 2, SrcY: 2,
		DstX: x1, DstY: y1,
		ColorR: 0.5,
		ColorB: 0.0,
		ColorG: 0.0,
		ColorA: 1.0,
	})
	indices = append(indices, []uint16{0, 1, 2, 2, 1, 3}...)
	screen.DrawTriangles(vertices, indices, solid, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterNearest, CompositeMode: ebiten.CompositeModeLighter})
}

func hex(xI, yI, rI int, screen *ebiten.Image) {
	r := float64(rI)
	x := float32(xI)
	y := float32(yI)
	minX, minY, maxX, maxY := x, y, x, y
	vertices := make([]ebiten.Vertex, 0, 16)
	indices := make([]uint16, 0, 21)
	vertices = append(vertices, ebiten.Vertex{
		SrcX: 2, SrcY: 2,
		DstX: x, DstY: y,
		ColorR: 0.0,
		ColorB: 0.5,
		ColorG: 0.0,
		ColorA: 1.0,
	})
	s, c := math.Sincos(0)
	for i := 0; i < 6; i++ {
		vertices = append(vertices, ebiten.Vertex{
			SrcX: 14, SrcY: 2,
			DstX: x + float32(r*c), DstY: y + float32(r*s),
			ColorR: 0.0,
			ColorB: 0.5,
			ColorG: 0.0,
			ColorA: 1.0,
		})
		s, c = math.Sincos((float64(i+1) * math.Pi) / 3)
		vertices = append(vertices, ebiten.Vertex{
			SrcX: 2, SrcY: 14,
			DstX: x + float32(r*c), DstY: y + float32(r*s),
			ColorR: 0.0,
			ColorB: 0.5,
			ColorG: 0.0,
			ColorA: 1.0,
		})
		indices = append(indices, 0, uint16(len(vertices)-1), uint16(len(vertices)-2))
	}
	for i := 0; i < len(vertices); i++ {
		if vertices[i].DstX < minX {
			minX = vertices[i].DstX
		}
		if vertices[i].DstX > maxX {
			maxX = vertices[i].DstX
		}
		if vertices[i].DstY < minY {
			minY = vertices[i].DstY
		}
		if vertices[i].DstY > maxY {
			maxY = vertices[i].DstY
		}
	}
	vx := vertices[2]
	vx.DstX = vertices[0].DstX
	vx.DstY += vx.DstY - y
	vx.ColorB = 0.0
	vx.ColorG = 0.5
	vertices = append(vertices, vx)
	vx = vertices[9]
	vx.DstX -= float32(r)
	vx.ColorB = 0.0
	vx.ColorG = 0.5
	vertices = append(vertices, vx)
	vx = vertices[11]
	vx.DstX += float32(r)
	vx.ColorB = 0.0
	vx.ColorG = 0.5
	vertices = append(vertices, vx)
	indices = append(indices, []uint16{13, 14, 15}...)
	// triangle: bottom two vertexes +/- radius, center + 2x the Y offset of the top vertexes.
	rect(minX, minY, maxX, maxY, screen)
	screen.DrawTriangles(vertices, indices, solid, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterNearest, CompositeMode: ebiten.CompositeModeLighter})
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

	solid, err = ebiten.NewImage(16, 16, ebiten.FilterNearest)
	solid.Fill(color.RGBA{255, 255, 255, 255})

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Miracle Modus"); err != nil {
		fmt.Fprintf(os.Stderr, "exiting: %s\n", err)
	}
}
