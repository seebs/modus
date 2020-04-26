package g

import (
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
	Render                     int
	depth                      int
	W, H, Major, Minor         int
	baseDots                   DotGridBase
	baseOffset                 float32
	coordOffsetX, coordOffsetY float32
	states                     []DotGridState
	Compute                    DotCompute
	ComputeInit                DotComputeInit
	vertices                   []ebiten.Vertex
	depthVertices              [][]ebiten.Vertex
	depthDirty                 []bool
	indices                    []uint16
	quads                      int
	// coordinate space to screen space
	scale, ox, oy float32
	gridScale     float32
	alphaDecay    float32
	alphaDecays   []float32
	status        string
}

// DotGridBase represents the underlying qualities of a point; it's populated
// by default with the "innate" X/Y coordinates, and everything else set to zero.
// A given mode gets to define how it uses the other members; the DotGridBase
// objects are shared between rendering passes.
type DotGridBase struct {
	Locs []FLoc
	Vecs []FVec
}

// DotGridState reports the state of a given dot after a rendering pass. States
// are used to generate vertices when drawing passes happen.
type DotGridState struct {
	Locs []FLoc
	P    []Paint
	A    []float32
	S    []float32
}

// DotCompute is a function which operates on a DotGridBase, looks at the
// previous state if it wants to, and computes the next state.
type DotCompute func(w, h int, base DotGridBase, prev DotGridState, next DotGridState) string
type DotComputeInit func(w, h int, base DotGridBase, initial DotGridState)

var (
	initDotData sync.Once
)

func dotSetup() {
	textureSetup()
}

func makeDotGridHeight(w int, sx, sy float32) (int, int, int, int, float32) {
	var h int
	var major, minor *int
	var scale float32
	for {
		if sx > sy {
			scale = math.Floor(sx / float32(w))
			h = int(math.Floor(sy / scale))
		} else {
			// compute sizes for portrait mode, then flip x and y
			scale = math.Floor(sy / float32(w))
			h = int(math.Floor(sx / scale))
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
	// fmt.Printf("%dx%d (size %d) [%d indices], grid scale %f\n", w, h, *major, (*major)*(*major)*6, scale*2)
	return w, h, *major, *minor, scale * 2
}

// NewPolyLine creates a new PolyLine using the specified sprite and palette.
func newDotGrid(w int, thickness float32, depth int, r RenderType, p *Palette, scale, ox, oy, coordOffsetX, coordOffsetY float32) *DotGrid {
	initDotData.Do(dotSetup)
	var h, major, minor int
	var gridScale float32
	w, h, major, minor, gridScale = makeDotGridHeight(w, ox*2, oy*2)
	// thickness was originally calculated by reference to width=20 on a 1280px screen, so...
	// 1280px/20 width => 64px spacing. but now we're looking at the center-offset, not the
	// screen size, so...
	thickness *= ox / float32(h) / 16
	if thickness < 2 {
		thickness = 2
	}

	dg := &DotGrid{
		Palette:   p,
		Thickness: thickness,
		W:         w, H: h,
		Major: major, Minor: minor,
		scale: scale,
		ox:    ox, oy: oy,
		coordOffsetX: coordOffsetX, coordOffsetY: coordOffsetY,
		gridScale: gridScale,
		depth:     depth,
	}
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
	dg.quads = dg.Major * dg.Major
	dg.vertices = make([]ebiten.Vertex, depth*4*dg.quads)
	dg.depthVertices = make([][]ebiten.Vertex, dg.depth)
	dg.indices = make([]uint16, 0, 6*dg.quads)
	dg.states = make([]DotGridState, dg.depth)
	dg.baseDots = DotGridBase{
		Locs: make([]FLoc, dg.quads),
		Vecs: make([]FVec, dg.quads),
	}
	dg.depthDirty = make([]bool, dg.depth)
	if dg.W == dg.Major {
		dg.baseOffset = dg.coordOffsetX
	} else {
		dg.baseOffset = dg.coordOffsetY
	}

	// fmt.Printf("%dx%d [actually %d/%d]: baseOffset %.2f, coordOffset %.2f/%.2f\n",
	//	dg.W, dg.H, dg.Major, dg.Minor, dg.baseOffset, dg.coordOffsetX, dg.coordOffsetY)
	offset := uint16(0)
	for i := 0; i < dg.Major; i++ {
		dots := dg.baseDots.Locs[i*dg.Major : (i+1)*dg.Major]
		x0 := (((float32(i) / float32(dg.H-1)) - 0.5) * 2) - dg.baseOffset
		for j := range dots {
			y0 := (((float32(j) / float32(dg.H-1)) - 0.5) * 2) - dg.baseOffset
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
		dg.states[d] = DotGridState{
			Locs: make([]FLoc, dg.quads),
			P:    make([]Paint, dg.quads),
			A:    make([]float32, dg.quads),
			S:    make([]float32, dg.quads),
		}
		dg.depthVertices[d] = dg.vertices[dg.quads*d*4 : dg.quads*(d+1)*4]
		// copy in an initial row
		for j := 0; j < dg.Major; j++ {
			vs := dg.vertices[vOffset : vOffset+4]
			copy(vs, dotData.vsByR[dg.Render])
			vOffset += 4
		}
		for i := 1; i < dg.Major; i++ {
			states := dg.vertices[vOffset*i : vOffset*(i+1)]
			// copy in that first row
			copy(states, dg.vertices[:vOffset])
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
		if target != nil {
			target.DrawTriangles(dg.depthVertices[d], dg.indices, dotData.img, &opt)
		}
	}
	if target != nil {
		ebitenutil.DebugPrint(target, dg.status)
	}
}

// drawVertices computes the actual vertex contents given the state
func (dg *DotGrid) drawVertices(state DotGridState, dvs []ebiten.Vertex, scale float32) {
	offset := 0
	r := dotData.vsByR[dg.Render]
	thickness := dg.Thickness * dotData.scales[dg.Render]
	locs := state.Locs[:dg.quads]
	a := state.A[:dg.quads]
	s := state.S[:dg.quads]
	p := state.P[:dg.quads]
	for i := 0; i < dg.quads; i++ {
		vs := dvs[offset : offset+4]
		// scale is a multiplier on the base thickness/size of
		// dots
		x, y := (locs[i].X*dg.scale)+dg.ox, (locs[i].Y*dg.scale)+dg.oy
		size := thickness * s[i]
		vs[0].DstX, vs[0].DstY = (x-size)*scale, (y-size)*scale
		vs[1].DstX, vs[1].DstY = (x+size)*scale, (y-size)*scale
		vs[2].DstX, vs[2].DstY = (x-size)*scale, (y+size)*scale
		vs[3].DstX, vs[3].DstY = (x+size)*scale, (y+size)*scale
		vs[0].SrcX, vs[0].SrcY = r[0].SrcX, r[0].SrcY
		vs[1].SrcX, vs[1].SrcY = r[1].SrcX, r[1].SrcY
		vs[2].SrcX, vs[2].SrcY = r[2].SrcX, r[2].SrcY
		vs[3].SrcX, vs[3].SrcY = r[3].SrcX, r[3].SrcY
		r, g, b, _ := dg.Palette.Float32(p[i])
		ai := a[i]
		vs[0].ColorR, vs[0].ColorG, vs[0].ColorB, vs[0].ColorA = r, g, b, ai
		vs[1].ColorR, vs[1].ColorG, vs[1].ColorB, vs[1].ColorA = r, g, b, ai
		vs[2].ColorR, vs[2].ColorG, vs[2].ColorB, vs[2].ColorA = r, g, b, ai
		vs[3].ColorR, vs[3].ColorG, vs[3].ColorB, vs[3].ColorA = r, g, b, ai
		offset += 4
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

	dg.status = dg.Compute(dg.W, dg.H, dg.baseDots, dg.states[1], dg.states[0])
}

func (dg *DotGrid) InitCompute() {
	if dg.ComputeInit != nil {
		dg.ComputeInit(dg.W, dg.H, dg.baseDots, dg.states[0])
	}
}
