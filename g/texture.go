package g

import (
	"image"
	"log"
	"sync"

	"image/color"

	"github.com/hajimehoshi/ebiten"
)

// Here, we create textures for other parts of g to use.


var (
	// line textures: each line gets a 32x32 box, which is a pixel-doubled
	// 16x16 box, although only the middle 14x14 (28x28) are supposed to be
	// used as the texture. The idea is to have boundaries around the part of
	// the texture we use to keep the edges/ends from being rendered darker
	// due when rendered with FilterLinear, even though actually I don't plan
	// to use FilterLinear anymore anyway.
	lineDepths = [4][16]byte{
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		{127, 127, 127, 127, 127, 255, 255, 255, 255, 255, 255, 127, 127, 127, 127, 127},
		{85, 85, 85, 127, 127, 127, 255, 255, 255, 255, 127, 127, 127, 85, 85, 85},
		{63, 63, 63, 127, 127, 191, 191, 255, 255, 191, 191, 127, 127, 63, 63, 63},
	}
	// squares store a series of rings around the central point.
	squareDepths = [4][32]byte {
		// white
		{
		255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
		},
		// the original: faded grey, with a bright line and a dimmer line at the edge
		{
		224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224,
		224, 224, 224, 224, 224, 224, 224, 224, 240, 240, 240, 240, 192, 192, 192, 192,
		},
		// fancy!
		{
		96, 96, 96, 96, 128, 128, 192, 192, 200, 200, 128, 200, 200, 200, 200, 224,
		224, 224, 224, 240, 240, 240, 224, 224, 200, 192, 192, 192, 64, 0, 0, 0,
		},
	}
	baseVertices = []ebiten.Vertex{
		{SrcX: 0, SrcY: 0, ColorA: 1.0},   // prev + nx,ny
		{SrcX: 0, SrcY: 1, ColorA: 1.0},   // prev - nx,ny
		{SrcX: 1, SrcY: 0, ColorA: 1.0},   // next + nx,ny
		{SrcX: 1, SrcY: 1, ColorA: 1.0},   // next - nx,ny
		{SrcX: 0, SrcY: 0.5, ColorA: 1.0}, // prev
		{SrcX: 1, SrcY: 0.5, ColorA: 1.0}, // next
	}
	triVerticesByDepth [][]ebiten.Vertex
	squareVerticesByDepth [][]ebiten.Vertex
	lineTexture *ebiten.Image
	squareTexture *ebiten.Image
	hexTexture *ebiten.Image
)

func createTextures() {
	img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 64, Y: 64}})
	for depth := 0; depth < 4; depth++ {
		offsetX := (depth%2)*32 + 2
		offsetY := (depth/2)*32 + 2
		offsetXf := float32(offsetX)
		offsetYf := float32(offsetY)
		scalef := float32(28)
		for r := 0; r < 32; r++ {
			v := lineDepths[depth][r/2]
			col := color.RGBA{v, v, v, v}
			for c := 0; c < 14; c++ {
				img.Set(offsetX+c*2, offsetY+r-1, col)
				img.Set(offsetX+c*2+1, offsetY+r-1, col)
			}
		}
		triVertices := make([]ebiten.Vertex, 6)
		for i := 0; i < 6; i++ {
			triVertices[i] = baseVertices[i]
			// pull X in from the ends, so it doesn't dim at the ends.
			triVertices[i].SrcX = offsetXf + 2 + triVertices[i].SrcX*(scalef-4)
			triVertices[i].SrcY = offsetYf + triVertices[i].SrcY*scalef
		}
		triVerticesByDepth = append(triVerticesByDepth, triVertices)
	}
	var err error
	lineTexture, err = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	if err != nil {
		log.Fatalf("couldn't create image: %s", err)
	}
	img = image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 128, Y: 128}})
	for depth := 0; depth < 4; depth++ {
		offsetX := (depth%2)*64
		offsetY := (depth/2)*64
		offsetXf := float32(offsetX)
		offsetYf := float32(offsetY)
		scalef := float32(60)
		c := 32
		for r := 0; r < 32; r++ {
			// zero value is transparent black
			var col color.RGBA
			v := squareDepths[depth][r]
			if v != 0 {
				// leave 0 values transparent
				col = color.RGBA{v, v, v, 255}
			}
			// radius 0 = the points immediately adjacent to center square,
			// thus, [c-r-1, c-r-1] through [c+r][c+r], inclusive
			min := c - r - 1
			max := c + r
			for i := min; i <= max; i++ {
				img.Set(offsetX + i, offsetY + min, col)
				img.Set(offsetX + i, offsetY + max, col)
				img.Set(offsetX + min, offsetY + i, col)
				img.Set(offsetX + max, offsetY + i, col)
			}
		}
		squareVertices := make([]ebiten.Vertex, 4)
		for i := 0; i < 4; i++ {
			squareVertices[i] = baseVertices[i]
			squareVertices[i].SrcX = offsetXf + 2 + squareVertices[i].SrcX*scalef
			squareVertices[i].SrcY = offsetYf + 2 + squareVertices[i].SrcY*scalef
		}
		squareVerticesByDepth = append(squareVerticesByDepth, squareVertices)
	}
	squareTexture, err = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	if err != nil {
		log.Fatalf("couldn't create image: %s", err)
	}
}

var (
	createTexturesOnce sync.Once
)

func textureSetup() {
	createTexturesOnce.Do(createTextures)
}
