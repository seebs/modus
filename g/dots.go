package g

import (
	"fmt"
	"sync"

	math "github.com/chewxy/math32"
	"github.com/hajimehoshi/ebiten"
	// "github.com/hajimehoshi/ebiten/ebitenutil"
)

// A Dots display represents an arrangement of dots, with locations determined
// geometrically in some way from their nominal location parameters and state.
// Locations are represented in -1/+1 around the center of the screen.
type DotGrid struct {
	Palette       *Palette
	Thickness     float32
	depth         int
	W, H          int
	baseDots      [][]DotGridBase
	states        [][][]DotGridState
	Compute       DotCompute
	vertices      []ebiten.Vertex
	depthVertices [][]ebiten.Vertex
	depthDirty    []bool
	indices       []uint16
	sx, sy        float32
	scale         float32
	alphaDecay    float32
	alphaDecays   []float32
}

// DotGridBase represents the underlying qualities of a point; it's populated
// by default with the "innate" X/Y coordinates, and everything else set to zero.
// A given mode gets to define how it uses the other members; the DotGridBase
// objects are shared between rendering passes.
type DotGridBase struct {
	X, Y   float32
	DX, DY float32
	Aux    float32
}

// DotGridState reports the state of a given dot after a rendering pass. States
// are used to generate vertices when drawing passes happen.
type DotGridState struct {
	X, Y float32
	P    Paint
	A    float32
	S    float32
}

// DotCompute is a function which operates on a DotGridBase, looks at the
// previous state if it wants to, and computes the next state.
type DotCompute func(base [][]DotGridBase, prev [][]DotGridState, next [][]DotGridState)

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
	if dg.depth > 1 {
		dg.alphaDecay = math.Pow(1/float32(dg.depth), 1/float32(dg.depth-1))
	} else {
		// unsed
		dg.alphaDecay = 1.0
	}
	dg.alphaDecays = make([]float32, dg.depth)
	alpha := float32(1.0)
	// precompute alpha for each depth
	for i := range dg.alphaDecays {
		dg.alphaDecays[i] = alpha
		alpha *= dg.alphaDecay
	}
	// each dot is a quad, which means it's 4 vertices and 6 indices, and
	// the indices don't change
	quads := dg.W * dg.H
	dg.vertices = make([]ebiten.Vertex, depth*4*quads)
	fmt.Printf("vertices: %p (quads %d)\n", &dg.vertices[0], quads)
	dg.depthVertices = make([][]ebiten.Vertex, dg.depth)
	dg.indices = make([]uint16, 0, 6*quads)
	dg.states = make([][][]DotGridState, dg.depth)
	dg.baseDots = make([][]DotGridBase, dg.W)
	dg.depthDirty = make([]bool, dg.depth)

	offset := uint16(0)
	for i := range dg.baseDots {
		dots := make([]DotGridBase, dg.H)
		dg.baseDots[i] = dots
		x0 := ((float32(i) / float32(dg.W-1)) - 0.5) * 2
		for j := range dg.baseDots[i] {
			y0 := ((float32(j) / float32(dg.H-1)) - 0.5) * 2
			dots[j].X, dots[j].Y = x0, y0
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
	offset = 0
	for d := 0; d < dg.depth; d++ {
		// this is the default, but to clarify: a thing isn't considered
		// dirty until it gets computed.
		dg.depthDirty[d] = false
		dg.states[d] = make([][]DotGridState, dg.W)
		dg.depthVertices[d] = dg.vertices[quads*d*4 : quads*(d+1)*4]
		for i := 0; i < dg.W; i++ {
			dg.states[d][i] = make([]DotGridState, dg.H)
			for j := 0; j < dg.H; j++ {
				vs := dg.vertices[offset : offset+4]
				vs[0].SrcX, vs[0].SrcY = 1, 1
				vs[1].SrcX, vs[1].SrcY = 15, 1
				vs[2].SrcX, vs[2].SrcY = 1, 15
				vs[3].SrcX, vs[3].SrcY = 15, 15
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
	opt := ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter}
	for d, dirty := range dg.depthDirty {
		if dirty {
			dg.drawVertices(dg.states[d], dg.depthVertices[d])
			dg.depthDirty[d] = false
		}
		opt.ColorM.Reset()
		opt.ColorM.Scale(1.0, 1.0, 1.0, float64(dg.alphaDecays[d]))
		target.DrawTriangles(dg.depthVertices[d], dg.indices[:dg.W*dg.H*6], dotTexture, &opt)
	}
}

// drawVertices computes the actual vertex contents given the state
func (dg *DotGrid) drawVertices(state [][]DotGridState, dvs []ebiten.Vertex) {
	offset := 0
	for i := 0; i < dg.W; i++ {
		for j := 0; j < dg.H; j++ {
			s := &state[i][j]
			vs := dvs[offset : offset+4]
			// scale is a multiplier on the base thickness/size of
			// dots
			x, y := (s.X/2+0.5)*dg.sx, (s.Y/2+0.5)*dg.sy
			scale := dg.Thickness * s.S
			vs[0].DstX, vs[0].DstY = x-scale, y-scale
			vs[1].DstX, vs[1].DstY = x+scale, y-scale
			vs[2].DstX, vs[2].DstY = x-scale, y+scale
			vs[3].DstX, vs[3].DstY = x+scale, y+scale
			r, g, b, _ := dg.Palette.Float32(s.P)
			vs[0].ColorR, vs[0].ColorG, vs[0].ColorB, vs[0].ColorA = r, g, b, s.A
			vs[1].ColorR, vs[1].ColorG, vs[1].ColorB, vs[1].ColorA = r, g, b, s.A
			vs[2].ColorR, vs[2].ColorG, vs[2].ColorB, vs[2].ColorA = r, g, b, s.A
			vs[3].ColorR, vs[3].ColorG, vs[3].ColorB, vs[3].ColorA = r, g, b, s.A
			offset += 4
		}
	}
}

// Tick updates the grid -- it shuffles out the oldest depth, then runs Compute
// to generate a new one and marks that one dirty.
func (dg *DotGrid) Tick() {
	// we rotate the depthvertices and indices, so at any point, [0] is
	// the most recent, [1] the next most recent, and so on. this lets
	// us draw them correctly elsewhere.
	lastVS := dg.depthVertices[dg.depth-1]
	copy(dg.depthVertices[1:], dg.depthVertices)
	dg.depthVertices[0] = lastVS

	copy(dg.depthDirty[1:], dg.depthDirty)
	dg.depthDirty[0] = true

	lastState := dg.states[dg.depth-1]
	copy(dg.states[1:], dg.states)
	dg.states[0] = lastState

	dg.Compute(dg.baseDots, dg.states[1], dg.states[0])
}
