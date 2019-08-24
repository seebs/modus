package g

import (
	"math/rand"

	math "github.com/chewxy/math32"

	"github.com/hajimehoshi/ebiten"
)

// SquareGrid represents a grid of squares.
type SquareGrid struct {
	Width, Height int
	Cells         [][]Cell
	palette       *Palette
	ExtraCells    []*FloatingCellBase
	vertices      []ebiten.Vertex
	base          []ebiten.Vertex
	indices       []uint16
	// not really a depth anymore; selects which of several textures to use
	render RenderType
	ox, oy int
	scale  float32 // actual size in pixels. integer plz.
}

// RandRow yields a random valid row.
func (gr *SquareGrid) RandRow() int {
	return int(rand.Int31n(int32(gr.Height)))
}

// RandCol yields a random valid column.
func (gr *SquareGrid) RandCol() int {
	return int(rand.Int31n(int32(gr.Width)))
}

// NewLoc yields a random valid location.
func (gr *SquareGrid) NewLoc() ILoc {
	return ILoc{X: gr.RandCol(), Y: gr.RandRow()}
}

// Add adds the provided vector and location, then wraps to produce a value
// within the bounds of the grid.
func (gr *SquareGrid) Add(l ILoc, v IVec) (ILoc, bool) {
	wrapped := false
	x, y := (l.X+v.X)%gr.Width, (l.Y+v.Y)%gr.Height
	if (x < l.X && v.X > 0) || (y < l.Y && v.Y > 0) {
		wrapped = true
	}
	if x < 0 {
		wrapped = true
		x += gr.Width
	}
	if y < 0 {
		wrapped = true
		y += gr.Height
	}
	return ILoc{X: x, Y: y}, wrapped
}

func (gr *SquareGrid) Palette() *Palette {
	return gr.palette
}

func newSquareGrid(w int, r RenderType, p *Palette, sx, sy int) *SquareGrid {
	var h int
	var scale float32
	if sx > sy {
		scale = math.Floor(float32(sx) / float32(w))
		h = int(math.Floor(float32(sy) / scale))
	} else {
		// compute sizes for portrait mode, then flip x and y
		scale = math.Floor(float32(sy) / float32(w))
		h = int(math.Floor(float32(sx) / scale))
		w, h = h, w
	}
	textureSetup()
	gr := SquareGrid{
		Width:   w,
		Height:  h,
		render:  r,
		palette: p,
		scale:   scale,
		ox:      (sx - int(scale)*w) / 2,
		oy:      (sy - int(scale)*h) / 2,
		base:    squareData.vsByR[r],
	}
	gr.Cells = make([][]Cell, gr.Width)
	for idx := range gr.Cells {
		col := make([]Cell, gr.Height)
		gr.Cells[idx] = col
		for j := range col {
			// set alpha to opaque by default
			col[j].Alpha = 1
			col[j].Scale = 1
		}
	}
	squares := gr.Width * gr.Height
	gr.vertices = make([]ebiten.Vertex, 0, squares*4)
	gr.indices = make([]uint16, 0, squares*6)
	for i := 0; i < squares; i++ {
		offset := uint16(i * 4)
		// 0   1
		// +---+
		// |   |
		// +---+
		// 2   3
		//
		// 0->1->2, 2->1->3
		// baseVertices currently live in lines.go, but it's the same here.
		// fmt.Printf("sqVBD[%d]: len %d\n", gr.Depth, len(squareData.vsByR))
		gr.vertices = append(gr.vertices, squareData.vsByR[gr.render]...)
		gr.indices = append(gr.indices,
			offset+0, offset+1, offset+2,
			offset+2, offset+1, offset+3)
	}
	return &gr
}

type Grid interface {
	NewLoc() ILoc
	Add(ILoc, IVec) (ILoc, bool)
	At(ILoc) *Cell
	IncP(ILoc, int) Paint
	IncAlpha(ILoc, float32)
	IncTheta(ILoc, float32)
	Neighbors(ILoc, GridFunc)
	Palette() *Palette
	Splash(ILoc, int, int, GridFunc)
	Iterate(GridFunc)
	NewExtraCell() FloatingCell
}

// GridFunc is a general callback for operations on the grid.
type GridFunc func(gr Grid, l ILoc, n int, c *Cell)

// Iterate runs fn on the entire grid.
func (gr *SquareGrid) Iterate(fn GridFunc) {
	for i, col := range gr.Cells {
		for j := range col {
			fn(gr, ILoc{X: i, Y: j}, 1, &col[j])
		}
	}
}

// NewExtraCell yields a new FloatingCell, which is stored in ExtraCells.
func (gr *SquareGrid) NewExtraCell() FloatingCell {
	c := &FloatingCellBase{Cell: Cell{Scale: 1.0, Alpha: 1.0}}
	gr.ExtraCells = append(gr.ExtraCells, c)
	// add vertex storage for extra cell
	offset := uint16(len(gr.vertices))
	gr.vertices = append(gr.vertices, squareData.vsByR[gr.render]...)
	gr.indices = append(gr.indices,
		offset+0, offset+1, offset+2,
		offset+2, offset+1, offset+3)
	return c
}

// At returns the cell at a grid location.
func (gr *SquareGrid) At(l ILoc) *Cell {
	return &gr.Cells[l.X][l.Y]
}

// IncP increments P (paint color) at a given location.
func (gr *SquareGrid) IncP(l ILoc, n int) Paint {
	sq := &gr.Cells[l.X][l.Y]
	sq.P = gr.palette.Inc(sq.P, n)
	return sq.P
}

// IncAlpha increments alpha at a given location.
func (gr *SquareGrid) IncAlpha(l ILoc, a float32) {
	gr.Cells[l.X][l.Y].IncAlpha(a)
}

// IncTheta increments theta at a given location.
func (gr *SquareGrid) IncTheta(l ILoc, t float32) {
	gr.Cells[l.X][l.Y].IncTheta(t)
}

// Splash splashes out from the given square, hitting squares
// between min and max out. A distance of 1 means the 4 adjacent
// squares, distance 2 means the 8 squares next out from those.
func (gr *SquareGrid) Splash(l ILoc, min, max int, fn GridFunc) {
	if min < 0 {
		min = 0
	}
	// zero radius is the square itself
	if min == 0 {
		fn(gr, l, 0, &gr.Cells[l.X][l.Y])
		min++
	}
	for depth := min; depth <= max; depth++ {
		for i, j := 0, depth; i < depth; i, j = i+1, j-1 {
			there, _ := gr.Add(l, IVec{i, j})
			fn(gr, there, depth, &gr.Cells[there.X][there.Y])
			there, _ = gr.Add(l, IVec{j, -i})
			fn(gr, there, depth, &gr.Cells[there.X][there.Y])
			there, _ = gr.Add(l, IVec{-i, -j})
			fn(gr, there, depth, &gr.Cells[there.X][there.Y])
			there, _ = gr.Add(l, IVec{-j, i})
			fn(gr, there, depth, &gr.Cells[there.X][there.Y])
		}
	}
}

// Neighbors runs fn on the nearby cells.
func (gr *SquareGrid) Neighbors(l ILoc, fn GridFunc) {
	gr.Splash(l, 1, 1, fn)
}

func (gr *SquareGrid) drawCell(vs []ebiten.Vertex, c *Cell, l FLoc, xscale, yscale float32) {
	vs = vs[0:4]
	// xscale and yscale are actually half the size of a default square.
	// thus, dx/dy are the offsets (whether positive or negative) of
	// the sides of the square, scaled appropriately for this individual
	// square's scale.
	dx, dy := xscale*c.Scale, yscale*c.Scale
	// we want to be a half-square offset, and we have a half-square size,
	// so X*2+1 => the center of square X.
	ox, oy := xscale*((l.X*2)+1), yscale*((l.Y*2)+1)
	if c.Theta != 0 {
		a := IdentityAffine()
		a.Rotate(c.Theta)
		a.Scale(dx, dy)
		a.E, a.F = ox, oy
		vs[0].DstX, vs[0].DstY = a.Project(-1, 1)
		vs[1].DstX, vs[1].DstY = a.Project(1, 1)
		vs[2].DstX, vs[2].DstY = a.Project(-1, -1)
		vs[3].DstX, vs[3].DstY = a.Project(1, -1)
	} else {
		// no rotation, so we can just adjust up or down
		// by the scale we're using.
		vs[0].DstX, vs[0].DstY = ox-dx, oy-dy
		vs[1].DstX, vs[1].DstY = ox+dx, oy-dy
		vs[2].DstX, vs[2].DstY = ox-dx, oy+dy
		vs[3].DstX, vs[3].DstY = ox+dx, oy+dy
	}
	r, g, b, _ := gr.palette.Float32(c.P)
	vs[0].ColorR, vs[0].ColorG, vs[0].ColorB, vs[0].ColorA = r, g, b, c.Alpha
	vs[1].ColorR, vs[1].ColorG, vs[1].ColorB, vs[1].ColorA = r, g, b, c.Alpha
	vs[2].ColorR, vs[2].ColorG, vs[2].ColorB, vs[2].ColorA = r, g, b, c.Alpha
	vs[3].ColorR, vs[3].ColorG, vs[3].ColorB, vs[3].ColorA = r, g, b, c.Alpha
}

// Draw displays the grid on the target screen.
func (gr *SquareGrid) Draw(target *ebiten.Image, scale float32) {
	xscale := scale / 2
	yscale := scale / 2
	op := &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter}
	var offset int
	gr.Iterate(func(generic Grid, l ILoc, n int, c *Cell) {
		gr := generic.(*SquareGrid)
		offset = ((l.Y * gr.Width) + l.X) * 4
		gr.drawCell(gr.vertices[offset:offset+4], c, l.FLoc(), xscale, yscale)
	})
	offset = gr.Width * gr.Height * 4
	// draw extra cells
	for _, c := range gr.ExtraCells {
		vs := gr.vertices[offset : offset+4]
		copy(vs, squareData.vsByR[c.Cell.R])
		gr.drawCell(vs, &c.Cell, *c.Loc(), xscale, yscale)
		offset += 4
	}
	if target != nil {
		target.DrawTriangles(gr.vertices, gr.indices, squareData.img, op)
	}
}
