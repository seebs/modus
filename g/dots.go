package g

import (
	"fmt"
	"sync"

	math "github.com/chewxy/math32"
	"github.com/hajimehoshi/ebiten"
)

// A Dots display represents an arrangement of dots, with locations determined
// geometrically in some way from their nominal location parameters.
type DotGrid struct {
	Palette       *Palette
	Thickness     float32
	depth         int
	W, H          int
	states        [][][]float32
	Compute       func(x0, y0 float32) (x, y float32, p Paint, a float32, scale float32)
	vertices      []ebiten.Vertex
	depthVertices [][]ebiten.Vertex
	indices       []uint16
	sx, sy        float32
	scale         float32
	alphaDecay    float32
}

type DotGridState struct {
	X, Y float32
	P    Paint
	A    float32
	S    float32
}

var (
	initDotData sync.Once
)

func dotSetup() {
	textureSetup()
}

func makeDotGridHeight(w, sx, sy int) (int, int, float32) {
	var h int
	var scale float32
	for {
		if sx > sy {
			scale = math.Floor(float32(sx) / float32(w))
			h = int(math.Floor(float32(sy) / scale))
		} else {
			// compute sizes for portrait mode, then flip x and y
			scale = math.Floor(float32(sy) / float32(w))
			h = int(math.Floor(float32(sx) / scale))
			w, h = h, w
		}
		if w*h*4 < 65530 {
			break
		}
		if w < 8 {
			h--
		} else {
			w--
		}
	}
	return w, h, scale
}

// NewPolyLine creates a new PolyLine using the specified sprite and palette.
func newDotGrid(w int, thickness float32, depth int, r RenderType, p *Palette, sx, sy int) *DotGrid {
	initDotData.Do(dotSetup)
	if thickness < 1 {
		thickness = 1
	}
	var h int
	var scale float32
	w, h, scale = makeDotGridHeight(w, sx, sy)

	dg := &DotGrid{Palette: p, Thickness: thickness, W: w, H: h, sx: float32(sx), sy: float32(sy), scale: scale, depth: depth}
	if depth > 1 {
		dg.alphaDecay = math.Pow(1/float32(depth), 1/float32(depth-1))
		fmt.Printf("depth %d, decay %f\n", depth, dg.alphaDecay)
	} else {
		// unsed
		dg.alphaDecay = 1.0
	}
	// each dot is a quad, which means it's 4 vertices and 6 indices, and
	// the indices don't change
	quads := 4 * dg.W * dg.H
	dg.vertices = make([]ebiten.Vertex, depth*quads)
	dg.depthVertices = make([][]ebiten.Vertex, depth)
	dg.indices = make([]uint16, 0, depth*6*dg.W*dg.H)
	offset := uint16(0)
	for d := 0; d < dg.depth; d++ {
		// fmt.Printf("[%d]: [%d:%d]/%d\n", d, quads*d, quads*(d+1), len(dg.vertices))
		dg.depthVertices[d] = dg.vertices[quads*d : quads*(d+1)]
		for i := 0; i < dg.W; i++ {
			for j := 0; j < dg.H; j++ {
				vs := dg.vertices[offset : offset+4]
				vs[0].SrcX, vs[0].SrcY = 1, 1
				vs[1].SrcX, vs[1].SrcY = 15, 1
				vs[2].SrcX, vs[2].SrcY = 1, 15
				vs[3].SrcX, vs[3].SrcY = 15, 15
				/*
				 * 0  1
				 * 2  3
				 * -> 012, 213
				 */
				dg.indices = append(dg.indices,
					offset+0, offset+1, offset+2,
					offset+2, offset+1, offset+3)
				offset += 4
			}
		}
	}
	return dg
}

// Draw renders the line on the target, using the sprite's drawimage
// options modified by color and location of line segments.
func (dg *DotGrid) Draw(target *ebiten.Image, scale float32) {
	if dg.Compute == nil {
		return
	}
	thickness := dg.Thickness
	dvs := dg.depthVertices[0]
	offset := 0
	for i := 0; i < dg.W; i++ {
		x0 := float32(i) / float32(dg.W-1)
		for j := 0; j < dg.H; j++ {
			y0 := float32(j) / float32(dg.H-1)
			vs := dvs[offset : offset+4]
			x, y, p, a, s := dg.Compute(x0, y0)
			// scale is a multiplier on the base thickness/size of
			// dots
			x, y = x*dg.sx, y*dg.sy
			s *= thickness
			vs[0].DstX, vs[0].DstY = x-s, y-s
			vs[1].DstX, vs[1].DstY = x+s, y-s
			vs[2].DstX, vs[2].DstY = x-s, y+s
			vs[3].DstX, vs[3].DstY = x+s, y+s
			r, g, b, _ := dg.Palette.Float32(p)
			vs[0].ColorR, vs[0].ColorG, vs[0].ColorB, vs[0].ColorA = r, g, b, a
			vs[1].ColorR, vs[1].ColorG, vs[1].ColorB, vs[1].ColorA = r, g, b, a
			vs[2].ColorR, vs[2].ColorG, vs[2].ColorB, vs[2].ColorA = r, g, b, a
			vs[3].ColorR, vs[3].ColorG, vs[3].ColorB, vs[3].ColorA = r, g, b, a
			offset += 4
		}
	}
	// draw the triangles
	target.DrawTriangles(dg.vertices, dg.indices, dotTexture, &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter})
}

func (dg *DotGrid) Tick() {
	last := dg.depthVertices[dg.depth-1]
	copy(dg.depthVertices[1:], dg.depthVertices)
	dg.depthVertices[0] = last
	// dim the older ones
	quads := dg.W * dg.H
	for d := 1; d < dg.depth; d++ {
		dvs := dg.depthVertices[d]
		offset := 0
		for i := 0; i < quads; i++ {
			vs := dvs[offset : offset+4]
			vs[0].ColorA *= 0.7
			vs[1].ColorA *= 0.7
			vs[2].ColorA *= 0.7
			vs[3].ColorA *= 0.7
			offset += 4
		}
	}
}
