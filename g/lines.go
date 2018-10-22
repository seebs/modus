package g

import (
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"sync"

	"image/color"

	"github.com/hajimehoshi/ebiten"
)

// A PolyLine represents a series of line segments, each
// with a distinct start/end color. In the absence of per-vertex
// colors, we use the end color for each line segment.
//
// A PolyLine is intended to be rendered onto a given display
// by scaling/rotating quads, because things like ebiten (or
// Corona) have limited line-drawing capabilities, so abstracting
// that away and presenting a polyline interface is more
// convenient.
type PolyLine struct {
	Points    []LinePoint
	Thickness float64
	Depth     int
	Palette   *Palette
	Blend     bool
	Joined    bool // one segment per point past the first, rather than each pair a segment
	debug     *PolyLine
	vertices  []ebiten.Vertex
	indices   []uint16
}

var lineTexture *ebiten.Image

// A LinePoint is one point in a PolyLine, containing both
// a location and a Paint corresponding to the PolyLine's Palette.
type LinePoint struct {
	X, Y float64
	P    Paint
}

var (
	initLineTexture sync.Once
)

// each depth from 0..3 gets a 16x16 box, of which it is the
// 14x14 pixels in the middle
//
var (
	depths = [4][14]byte{
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		{127, 127, 127, 127, 255, 255, 255, 255, 255, 255, 127, 127, 127, 127},
		{85, 85, 127, 127, 127, 255, 255, 255, 255, 127, 127, 127, 85, 85},
		{63, 63, 127, 127, 191, 191, 255, 255, 191, 191, 127, 127, 63, 63},
	}
)

func createLineTexture() {
	img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 64, Y: 64}})
	for depth := 0; depth < 4; depth++ {
		offsetX := (depth%2)*16 + 1
		offsetY := (depth/2)*16 + 1
		offsetXf := float32(offsetX)
		offsetYf := float32(offsetY)
		scalef := float32(14)
		for r := 0; r < 14; r++ {
			v := depths[depth][r]
			col := color.RGBA{v, v, v, v}
			for c := 0; c < 14; c++ {
				img.Set(offsetX+c, offsetY+r, col)
			}
		}
		triVertices := make([]ebiten.Vertex, 4)
		for i := 0; i < 4; i++ {
			triVertices[i] = fourVertices[i]
			triVertices[i].SrcX = offsetXf + triVertices[i].SrcX*scalef
			triVertices[i].SrcY = offsetYf + triVertices[i].SrcY*scalef
		}
		fourVerticesByDepth = append(fourVerticesByDepth, triVertices)
	}
	var err error
	lineTexture, err = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
	if err != nil {
		log.Fatal("couldn't create image: %s", err)
	}
}

// NewPolyLine creates a new PolyLine using the specified sprite and palette.
func NewPolyLine(p *Palette, depth int) *PolyLine {
	initLineTexture.Do(createLineTexture)
	if depth > 3 {
		depth = 3
	}
	pl := &PolyLine{Palette: p, Depth: depth, Blend: true}
	return pl
}

func (pl *PolyLine) Debug(enable bool) {
	if enable {
		pl.debug = NewPolyLine(pl.Palette, 1)
		pl.debug.Thickness = 2
	} else {
		pl.debug = nil
	}
}

// corresponding indices:
// 1 0 2, 2 3 1
// the unchanging parts
var fourVertices = []ebiten.Vertex{
	{SrcX: 0, SrcY: 0, ColorA: 1.0}, // prev + nx,ny
	{SrcX: 0, SrcY: 1, ColorA: 1.0}, // prev - nx,ny
	{SrcX: 1, SrcY: 0, ColorA: 1.0}, // next + nx,ny
	{SrcX: 1, SrcY: 1, ColorA: 1.0}, // next - nx,ny
}
var fourVerticesByDepth [][]ebiten.Vertex

type LineBits struct {
	dx, dy float64 // delta x, delta y
	l      float64 // length
	nx, ny float64 // normal vector, normalized to unit length
}

// Draw renders the line on the target, using the sprite's drawimage
// options modified by color and location of line segments.
func (pl *PolyLine) Draw(target *ebiten.Image, alpha64 float64) {
	alpha := float32(alpha64)
	thickness := pl.Thickness
	// no invisible lines plz
	if thickness == 0 {
		thickness = 0.7
	}
	halfthick := thickness / 2
	// we need four points per line segment per depth
	var segments int
	if pl.Joined {
		segments = len(pl.Points) - 1
	} else {
		segments = len(pl.Points) / 2
	}
	if segments < 1 {
		// fail
		fmt.Fprintf(os.Stderr, "polyline of %d segments can't be drawn\n", segments)
		return
	}
	// populate with the SrcX, SrcY values.
	if len(pl.vertices) < segments*4 {
		fv := fourVerticesByDepth[pl.Depth]
		pl.vertices = make([]ebiten.Vertex, 0, segments*4)
		for i := 0; i < segments; i++ {
			pl.vertices = append(pl.vertices, fv...)
		}
	}
	// indices can never change, conveniently!
	if len(pl.indices) < segments*6 {
		for i := len(pl.indices) / 6; i < segments; i++ {
			offset := uint16(i * 4)
			pl.indices = append(pl.indices,
				offset+1, offset+0, offset+2,
				offset+2, offset+3, offset+1)
		}
	}
	prev := pl.Points[0]
	r0, g0, b0, _ := pl.Palette.Float32(prev.P)
	count := 0

	if pl.debug != nil {
		pl.debug.Reset()
	}
	// Joined: We will draw one segment for each point past the first.
	// Unjoined: We draw one segment for each pair.
	var plb LineBits
	for idx, next := range pl.Points[1:] {
		lb := LineBits{dx: next.X - prev.X, dy: next.Y - prev.Y}
		lb.l = math.Sqrt(lb.dx*lb.dx + lb.dy*lb.dy)
		if (lb.l == 0) || (!pl.Joined && (idx%2) == 1) {
			// don't draw 0-length line, don't divide by zero, but
			// do update the point so we use the right color to draw
			// the next segment.
			prev = next
			r0, g0, b0, _ = pl.Palette.Float32(next.P)
			plb = lb
			continue
		}
		// compute normal x/y values, scaled to unit length
		lb.nx, lb.ny = lb.dy/lb.l, -lb.dx/lb.l
		r1, g1, b1, _ := pl.Palette.Float32(next.P)
		offset := uint16(count * 4)
		v := pl.vertices[offset : offset+4]
		v[0].DstX = float32(prev.X + lb.nx*halfthick)
		v[0].DstY = float32(prev.Y + lb.ny*halfthick)
		v[1].DstX = float32(prev.X - lb.nx*halfthick)
		v[1].DstY = float32(prev.Y - lb.ny*halfthick)
		v[2].DstX = float32(next.X + lb.nx*halfthick)
		v[2].DstY = float32(next.Y + lb.ny*halfthick)
		v[3].DstX = float32(next.X - lb.nx*halfthick)
		v[3].DstY = float32(next.Y - lb.ny*halfthick)
		if pl.Blend {
			v[0].ColorR, v[0].ColorG, v[0].ColorB, v[0].ColorA = r0, g0, b0, alpha
			v[1].ColorR, v[1].ColorG, v[1].ColorB, v[1].ColorA = r0, g0, b0, alpha
		} else {
			v[0].ColorR, v[0].ColorG, v[0].ColorB, v[0].ColorA = r1, g1, b1, alpha
			v[1].ColorR, v[1].ColorG, v[1].ColorB, v[1].ColorA = r1, g1, b1, alpha
		}
		v[2].ColorR, v[2].ColorG, v[2].ColorB, v[2].ColorA = r1, g1, b1, alpha
		v[3].ColorR, v[3].ColorG, v[3].ColorB, v[3].ColorA = r1, g1, b1, alpha

		// add debugging segments
		if pl.debug != nil && idx > 0 {
			vX, vY := plb.ny-lb.ny, lb.nx-plb.nx
			vX, vY = vX*halfthick, vY*halfthick
			pl.debug.Add(prev.X, prev.Y, 4)
			pl.debug.Add(prev.X+vX, prev.Y+vY, 4)
		}

		// rotate colors
		r0, g0, b0 = r1, g1, b1
		// rotate points
		prev = next
		plb = lb
		// bump count since we drew a segment
		count++
	}
	// draw the triangles
	target.DrawTriangles(pl.vertices[:count*4], pl.indices[:count*6], lineTexture, &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter})
	if pl.debug != nil {
		pl.debug.Draw(target, alpha64)
	}
}

// Length yields the number of points in the line.
func (pl *PolyLine) Length() int {
	return len(pl.Points)
}

// Reset removes all points.
func (pl *PolyLine) Reset() {
	if pl.Points != nil {
		pl.Points = pl.Points[:0]
	}
}

// Point yields a given point within the line.
func (pl *PolyLine) Point(i int) *LinePoint {
	if i < 0 || i >= len(pl.Points) {
		return nil
	}
	return &pl.Points[i]
}

// Add adds a new point to the line.
func (pl *PolyLine) Add(x, y float64, p Paint) {
	pt := LinePoint{X: x, Y: y, P: p}
	pl.Points = append(pl.Points, pt)
}
