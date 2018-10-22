package g

import (
	"fmt"
	"math"
	"os"

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
	sx, sy    float64
	vertices  []ebiten.Vertex
	indices   []uint16
}

var lineTexture *ebiten.Image

func init() {
	var err error
	lineTexture, err = ebiten.NewImage(1, 1, ebiten.FilterDefault)
	if err != nil {
		panic("can't create line image!")
	}
	err = lineTexture.Fill(color.NRGBA{255,255,255,255})
	if err != nil {
		panic("can't fill line image!")
	}
}

// A LinePoint is one point in a PolyLine, containing both
// a location and a Paint corresponding to the PolyLine's Palette.
type LinePoint struct {
	X, Y float64
	P    Paint
}

// NewPolyLine creates a new PolyLine using the specified sprite and palette.
func NewPolyLine(p *Palette, depth int) *PolyLine {
	pl := &PolyLine{Palette: p, Depth: depth, Blend: true}
	return pl
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

// Draw renders the line on the target, using the sprite's drawimage
// options modified by color and location of line segments.
func (pl PolyLine) Draw(target *ebiten.Image, alpha64 float64) {
	alpha := float32(alpha64)
	thickness := pl.Thickness
	// no invisible lines plz
	if thickness == 0 {
		thickness = 0.7
	}
	halfthick := thickness / 2
	// we need four points per line segment per depth
	segments := len(pl.Points) - 1
	if segments < 1 {
		// fail
		fmt.Fprintf(os.Stderr, "polyline of %d segments can't be drawn\n", segments)
		return
	}
	// populate with the SrcX, SrcY values.
	if len(pl.vertices) < segments*4 {
		fv := fourVertices // ByDepth[pl.Depth]
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
	op := ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(100, 100)
	target.DrawImage(lineTexture, &op)
	op.GeoM.Reset()
	r0, g0, b0, _ := pl.Palette.Float32(prev.P)
	count := 0
	for _, next := range pl.Points[1:] {
		dx, dy := (next.X - prev.X), (next.Y - prev.Y)
		if dx == 0 && dy == 0 {
			// don't draw 0-length line, don't divide by zero, but
			// do update the point so we use the right color to draw
			// the next segment.
			prev = next
			r0, g0, b0, _ = pl.Palette.Float32(next.P)
			continue
		}
		l := math.Sqrt(dx*dx + dy*dy)
		// compute normal x/y values, scaled to unit length
		nx, ny := dy/l, -dx/l
		r1, g1, b1, _ := pl.Palette.Float32(next.P)
		offset := uint16(count * 4)
		v := pl.vertices[offset : offset+4]
		v[0].DstX = float32(prev.X + nx*halfthick)
		v[0].DstY = float32(prev.Y + ny*halfthick)
		v[1].DstX = float32(prev.X - nx*halfthick)
		v[1].DstY = float32(prev.Y - ny*halfthick)
		v[2].DstX = float32(next.X + nx*halfthick)
		v[2].DstY = float32(next.Y + ny*halfthick)
		v[3].DstX = float32(next.X - nx*halfthick)
		v[3].DstY = float32(next.Y - ny*halfthick)
		if pl.Blend {
			v[0].ColorR, v[0].ColorG, v[0].ColorB, v[0].ColorA = r0, g0, b0, alpha
			v[1].ColorR, v[1].ColorG, v[1].ColorB, v[1].ColorA = r0, g0, b0, alpha
		} else {
			v[0].ColorR, v[0].ColorG, v[0].ColorB, v[0].ColorA = r1, g1, b1, alpha
			v[1].ColorR, v[1].ColorG, v[1].ColorB, v[1].ColorA = r1, g1, b1, alpha
		}
		v[2].ColorR, v[2].ColorG, v[2].ColorB, v[2].ColorA = r1, g1, b1, alpha
		v[3].ColorR, v[3].ColorG, v[3].ColorB, v[3].ColorA = r1, g1, b1, alpha

		// rotate colors
		r0, g0, b0 = r1, g1, b1
		// rotate points
		prev = next
		// bump count since we drew a segment
		count++
	}
	fmt.Printf("vertices: %#v\nindices: %#v\n", pl.vertices[:count*4], pl.indices[:count*6])
	// draw the triangles
	target.DrawTriangles(pl.vertices[:count*4], pl.indices[:count*6], lineTexture, &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter})
}

// Length yields the number of points in the line.
func (pl PolyLine) Length() int {
	return len(pl.Points)
}

// Point yields a given point within the line.
func (pl PolyLine) Point(i int) *LinePoint {
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
