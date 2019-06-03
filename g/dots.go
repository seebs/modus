package g

import (
	"fmt"
	"sync"

	math "github.com/chewxy/math32"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

// A Dots display represents an arrangement of dots, with locations determined
// geometrically in some way from their nominal location parameters and state.
// Locations are represented in -1/+1 around the center of the screen.
type DotGrid struct {
	Palette                    *Palette
	Thickness                  float32
	depth                      int
	W, H, Major, Minor         int
	baseDots                   [][]DotGridBase
	baseOffset                 float32
	coordOffsetX, coordOffsetY float32
	states                     [][][]DotGridState
	Compute                    DotCompute
	ComputeInit                DotComputeInit
	vertices                   []ebiten.Vertex
	depthVertices              [][]ebiten.Vertex
	depthDirty                 []bool
	indices                    []uint16
	sx, sy                     float32
	scale                      float32
	alphaDecay                 float32
	alphaDecays                []float32
	status                     string
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
type DotCompute func(base [][]DotGridBase, prev [][]DotGridState, next [][]DotGridState) string
type DotComputeInit func(base [][]DotGridBase, initial [][]DotGridState)

var (
	initDotData sync.Once
)

func dotSetup() {
	textureSetup()
}

func makeDotGridHeight(w, sx, sy int) (int, int, int, int, float32) {
	var h int
	var major, minor *int
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
		if w > h {
			major = &w
			minor = &h
		} else {
			major = &h
			minor = &w
		}
		if (*major)*(*major)*6 < ebiten.MaxIndicesNum {
			break
		}
		if w < 8 {
			h--
		} else {
			w--
		}
	}
	fmt.Printf("%dx%d (size %d) [%d indices]\n", w, h, *major, (*major)*(*major)*6)
	return w, h, *major, *minor, scale
}

// NewPolyLine creates a new PolyLine using the specified sprite and palette.
func newDotGrid(w int, thickness float32, depth int, r RenderType, p *Palette, sx, sy int) *DotGrid {
	initDotData.Do(dotSetup)
	var h, major, minor int
	var scale float32
	w, h, major, minor, scale = makeDotGridHeight(w, sx, sy)
	// thickness was originally calculated by reference to width=20 on a 1280px screen, so...
	// 1280px/20 width => 64px spacing
	thickness *= float32(sx) / float32(w) / 32
	if thickness < 2 {
		thickness = 2
	}

	dg := &DotGrid{Palette: p, Thickness: thickness, W: w, H: h, Major: major, Minor: minor, sx: float32(sx), sy: float32(sy), scale: scale, depth: depth}
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
	quads := dg.Major * dg.Major
	dg.vertices = make([]ebiten.Vertex, depth*4*quads)
	dg.depthVertices = make([][]ebiten.Vertex, dg.depth)
	dg.indices = make([]uint16, 0, 6*quads)
	dg.states = make([][][]DotGridState, dg.depth)
	dg.baseDots = make([][]DotGridBase, dg.Major)
	dg.depthDirty = make([]bool, dg.depth)
	dg.baseOffset = float32(dg.Major-dg.Minor) / 2
	if dg.W == dg.Major {
		dg.coordOffsetX = dg.baseOffset / float32(dg.Minor)
	} else {
		dg.coordOffsetY = dg.baseOffset / float32(dg.Minor)

	}

	fmt.Printf("%dx%d [actually %d/%d]: baseOffset %.2f, coordOffset %.2f/%.2f\n",
		dg.W, dg.H, dg.Major, dg.Minor, dg.baseOffset, dg.coordOffsetX, dg.coordOffsetY)
	offset := uint16(0)
	for i := range dg.baseDots {
		dots := make([]DotGridBase, dg.Major)
		dg.baseDots[i] = dots
		x0 := (((float32(i) - dg.baseOffset) / float32(dg.H-1)) - 0.5) * 2
		for j := range dots {
			y0 := (((float32(j) - dg.baseOffset) / float32(dg.H-1)) - 0.5) * 2
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
	vOffset := 0
	for d := 0; d < dg.depth; d++ {
		// this is the default, but to clarify: a thing isn't considered
		// dirty until it gets computed.
		dg.depthDirty[d] = false
		dg.states[d] = make([][]DotGridState, dg.Major)
		dg.depthVertices[d] = dg.vertices[quads*d*4 : quads*(d+1)*4]
		for i := 0; i < dg.Major; i++ {
			dg.states[d][i] = make([]DotGridState, dg.Major)
			for j := 0; j < dg.Major; j++ {
				vs := dg.vertices[vOffset : vOffset+4]
				copy(vs, dotData.vsByR[1])
				vOffset += 4
			}
		}
	}
	return dg
}

// Draw renders the line on the target, using the sprite's drawimage
// options modified by color and location of line segments.
func (dg *DotGrid) Draw(target *ebiten.Image, scale float32) {
	opt := ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter}
	for d, dirty := range dg.depthDirty {
		if dirty {
			dg.drawVertices(dg.states[d], dg.depthVertices[d], scale)
			dg.depthDirty[d] = false
		}
		opt.ColorM.Reset()
		opt.ColorM.Scale(1.0, 1.0, 1.0, float64(dg.alphaDecays[d]))
		target.DrawTriangles(dg.depthVertices[d], dg.indices, dotData.img, &opt)
	}
	ebitenutil.DebugPrint(target, dg.status)
}

// drawVertices computes the actual vertex contents given the state
func (dg *DotGrid) drawVertices(state [][]DotGridState, dvs []ebiten.Vertex, scale float32) {
	offset := 0
	for i := 0; i < dg.Major; i++ {
		for j := 0; j < dg.Major; j++ {
			s := &state[i][j]
			vs := dvs[offset : offset+4]
			// scale is a multiplier on the base thickness/size of
			// dots
			x, y := (s.X/2+0.5+dg.coordOffsetX)*dg.sy, (s.Y/2+0.5+dg.coordOffsetY)*dg.sy
			size := dg.Thickness * s.S
			vs[0].DstX, vs[0].DstY = (x-size)*scale, (y-size)*scale
			vs[1].DstX, vs[1].DstY = (x+size)*scale, (y-size)*scale
			vs[2].DstX, vs[2].DstY = (x-size)*scale, (y+size)*scale
			vs[3].DstX, vs[3].DstY = (x+size)*scale, (y+size)*scale
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

	dg.status = dg.Compute(dg.baseDots, dg.states[1], dg.states[0])
}

func (dg *DotGrid) InitCompute() {
	if dg.ComputeInit != nil {
		dg.ComputeInit(dg.baseDots, dg.states[0])
	}
}
