package g

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
)

// SquareGrid represents a grid of squares.
type SquareGrid struct {
	Width, Height int
	Squares       [][]Cell
	Palette       *Palette
	vertices      []ebiten.Vertex
	base          []ebiten.Vertex
	indices       []uint16
	// not really a depth anymore; selects which of several textures to use
	render RenderType
	ox, oy int
	scale  float64 // actual size in pixels. integer plz.
}

func (gr *SquareGrid) RandRow() int {
	return int(rand.Int31n(int32(gr.Height)))
}

func (gr *SquareGrid) RandCol() int {
	return int(rand.Int31n(int32(gr.Width)))
}

func (gr *SquareGrid) NewLoc() ILoc {
	return ILoc{X: gr.RandCol(), Y: gr.RandRow()}
}

func (gr *SquareGrid) Add(l ILoc, v IVec) ILoc {
	x, y := (l.X+v.X)%gr.Width, (l.Y+v.Y)%gr.Height
	if x < 0 {
		x += gr.Width
	}
	if y < 0 {
		y += gr.Height
	}
	return ILoc{X: x, Y: y}
}

func newSquareGrid(w int, r RenderType, sx, sy int) *SquareGrid {
	var h int
	var scale float64
	if sx > sy {
		scale = math.Floor(float64(sx) / float64(w))
		h = int(math.Floor(float64(sy) / scale))
	} else {
		// compute sizes for portrait mode, then flip x and y
		scale = math.Floor(float64(sy) / float64(w))
		h = int(math.Floor(float64(sx) / scale))
		w, h = h, w
	}
	textureSetup()
	gr := SquareGrid{
		Width:  w,
		Height: h,
		render: r,
		scale:  scale,
		ox:     (sx - int(scale)*w) / 2,
		oy:     (sy - int(scale)*h) / 2,
		base:   squareVerticesByDepth[r],
	}
	gr.Squares = make([][]Cell, gr.Width)
	for idx := range gr.Squares {
		col := make([]Cell, gr.Height)
		gr.Squares[idx] = col
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
		// fmt.Printf("sqVBD[%d]: len %d\n", gr.Depth, len(squareVerticesByDepth))
		gr.vertices = append(gr.vertices, squareVerticesByDepth[gr.render]...)
		gr.indices = append(gr.indices,
			offset+0, offset+1, offset+2,
			offset+2, offset+1, offset+3)
	}
	return &gr
}

type Grid interface {
	At(ILoc) *Cell
	IncP(ILoc, int)
	IncAlpha(ILoc, float32)
	IncTheta(ILoc, float32)
	Neighbors(ILoc, GridFunc)
	Splash(ILoc, int, int, GridFunc)
	Iterate(GridFunc)
}

// A SquareGridFunc is a general callback for operations on the grid.
type GridFunc func(gr Grid, l ILoc, n int, c *Cell)

// Iterate runs fn on the entire grid.
func (gr *SquareGrid) Iterate(fn GridFunc) {
	for i, col := range gr.Squares {
		for j := range col {
			fn(gr, ILoc{X: i, Y: j}, 1, &col[j])
		}
	}
}

func (gr *SquareGrid) At(l ILoc) *Cell {
	return &gr.Squares[l.X][l.Y]
}

func (gr *SquareGrid) IncP(l ILoc, n int) {
	sq := &gr.Squares[l.X][l.Y]
	sq.P = gr.Palette.Inc(sq.P, n)
}

func (gr *SquareGrid) IncAlpha(l ILoc, a float32) {
	gr.Squares[l.X][l.Y].IncAlpha(a)
}

func (gr *SquareGrid) IncTheta(l ILoc, t float32) {
	gr.Squares[l.X][l.Y].IncTheta(t)
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
		fn(gr, l, 0, &gr.Squares[l.X][l.Y])
		min++
	}
	for depth := min; depth <= max; depth++ {
		for i, j := 0, depth; i < depth; i, j = i+1, j-1 {
			there := gr.Add(l, IVec{i, j})
			fn(gr, there, depth, &gr.Squares[there.X][there.Y])
			there = gr.Add(l, IVec{j, -i})
			fn(gr, there, depth, &gr.Squares[there.X][there.Y])
			there = gr.Add(l, IVec{-i, -j})
			fn(gr, there, depth, &gr.Squares[there.X][there.Y])
			there = gr.Add(l, IVec{-j, i})
			fn(gr, there, depth, &gr.Squares[there.X][there.Y])
		}
	}
}

// Neighbors runs fn on the nearby cells.
func (gr *SquareGrid) Neighbors(l ILoc, fn GridFunc) {
	gr.Splash(l, 1, 1, fn)
}

// Draw displays the grid on the target screen.
func (gr *SquareGrid) Draw(target *ebiten.Image, scale float32) {
	w, h := target.Size()
	// if xscale and yscale aren't identical, how should rotation work? well, at 90 degree rotations,
	// we want the square to fit cleanly. so. rotate a theoretical unit square, then scale by {x, y}
	xscale := float32(w) / float32(gr.Width) / 2
	yscale := float32(h) / float32(gr.Height) / 2
	op := &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter}
	gr.Iterate(func(generic Grid, l ILoc, n int, c *Cell) {
		gr := generic.(*SquareGrid)
		offset := ((l.Y * gr.Width) + l.X) * 4
		vs := gr.vertices[offset : offset+4]
		// xscale and yscale are actually half the size of a default square.
		// thus, dx/dy are the offsets (whether positive or negative) of
		// the sides of the square, scaled appropriately for this individual
		// square's scale.
		dx, dy := xscale*c.Scale, yscale*c.Scale
		// we want to be a half-square offset, and we have a half-square size,
		// so X*2+1 => the center of square X.
		ox, oy := xscale*(float32(l.X*2)+1), yscale*(float32(l.Y*2)+1)
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
		r, g, b, _ := gr.Palette.Float32(c.P)
		vs[0].ColorR, vs[0].ColorG, vs[0].ColorB, vs[0].ColorA = r, g, b, c.Alpha
		vs[1].ColorR, vs[1].ColorG, vs[1].ColorB, vs[1].ColorA = r, g, b, c.Alpha
		vs[2].ColorR, vs[2].ColorG, vs[2].ColorB, vs[2].ColorA = r, g, b, c.Alpha
		vs[3].ColorR, vs[3].ColorG, vs[3].ColorB, vs[3].ColorA = r, g, b, c.Alpha
	})
	target.DrawTriangles(gr.vertices, gr.indices, squareTexture, op)
}
