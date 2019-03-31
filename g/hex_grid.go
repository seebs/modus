package g

import (
	math "github.com/chewxy/math32"

	"github.com/hajimehoshi/ebiten"
)

// HexVec is a 3-coordinate vector that can be used to modify
// a location on a grid. Z = +X-Y. We don't enforce consistency
// on this; we just add the coordinates and expect it to work out.
type HexVec struct {
	X, Y, Z int
}

var hexDirections = []IVec{
	{X: 1, Y: 0},
	{X: 1, Y: -1}, // +Z
	{X: 0, Y: -1},
	{X: -1, Y: 0},
	{X: -1, Y: 1}, // -Z
	{X: 0, Y: 1},
}

// IVec converts a HexVec back to an IVec, by applying the Z
// vector changes.
func (v HexVec) IVec() IVec {
	return IVec{X: v.X + v.Z, Y: v.Y - v.Z}
}

// HexVec converts an IVec to a hex vector with Z=0.
func (v IVec) HexVec() HexVec {
	return HexVec{X: v.X, Y: v.Y}
}

// FloatingHexCell represents a floating cell which also supports Z
// translation.
type FloatingHexCell struct {
	FloatingCellBase
	z float32
}

func (f *FloatingHexCell) Z() *float32 {
	return &f.z
}

// HexGrid represents a hexagonal grid, approximately using axial
// coordinates.
//
// https://www.redblobgames.com/grids/hexagons/
//
// We use offsets, so each direction has the same impact on coordinates
// regardless of row. Coordinates wrap; on a 5x4 grid, for instance,
// the coordinates {-1,2} and {4,2} denote the same location.
//
//   0,0     1,0     2,0     3,0     4,0
//       0,1     1,1     2,1     3,1     4,1
//  -1,2     0,2     1,2     2,2     3,2
//      -1,3     0,3     1,3     2,3     3,3
//
// For 3 coordinates, we call the direction with +X/-Y "+Z", and
// the direction with -X/+Y "-Z".
type HexGrid struct {
	Width, Height       int
	hexWidth, hexHeight float32
	perHexHeight        float32
	Palette             *Palette
	Cells               [][]Cell
	ExtraCells          []*FloatingHexCell
	render              RenderType
	vertices            []ebiten.Vertex
	indices             []uint16
	ox, oy              float32 // offset to draw grid at for centering
}

// make a new hex grid. since hexes aren't interchangeable, we can't
// just flip X and Y...
//
// we start with the easy one: we use the flat ends, so the width of
// the row is trivial, except we need an extra half-hex, because a second
// row of hexes will be half a hex offset.
func newHexGrid(w int, r RenderType, sx, sy int) *HexGrid {
	gr := &HexGrid{render: r, Width: w}
	hexWidth := math.Floor(float32(sx) / (float32(w) + 0.5))
	// make it an even number, so the half-hex offset rows don't
	// look funny
	if int(hexWidth)&1 == 1 {
		hexWidth--
	}

	// the full height of a hex is 2/sqrt(3) times the width, but
	// each additional row costs only 3/4 that much.
	hexHeight := math.Floor(2 / math.Sqrt(3) * hexWidth)
	// the first row costs a full hexHeight. Every row after it costs
	// 3/4 of that. So:
	// h = (3n+1) * x/4
	// 4h/x = 3n + 1
	// (4h/x - 1) = 3n
	// (4h/x - 1)/3 = n
	vHexes := math.Floor((float32(sy)*4/hexHeight - 1) / 3)

	gr.hexWidth = float32(hexWidth)
	gr.hexHeight = float32(hexHeight)
	gr.perHexHeight = 3 * hexHeight / 4
	totalHeight := float32((3*vHexes + 1) * hexHeight / 4)
	totalWidth := float32(hexWidth * (float32(w) + 0.5))
	gr.ox, gr.oy = (float32(sx)-totalWidth)/2, (float32(sy)-totalHeight)/2
	gr.Height = int(vHexes)

	// fmt.Printf("sx %d, w %d, hexWidth %.1f\n", sx, w, hexWidth)
	// fmt.Printf("hexHeight %.1f, sy %d, vHexes %f, total %f\n", hexHeight, sy, vHexes, totalHeight)
	// fmt.Printf("ox %.1f, oy %.1f\n", gr.ox, gr.oy)

	gr.Cells = make([][]Cell, gr.Width)
	gr.vertices = make([]ebiten.Vertex, 0, 3*gr.Width*gr.Height)
	gr.indices = make([]uint16, 3*gr.Width*gr.Width)
	for col := range gr.Cells {
		r := make([]Cell, gr.Height)
		for row := range r {
			r[row] = Cell{Alpha: 1, Scale: .95}
			offset := uint16(len(gr.vertices))
			gr.vertices = append(gr.vertices, hexVerticesByDepth[gr.render]...)
			gr.indices = append(gr.indices, offset+0, offset+1, offset+2)
		}
		gr.Cells[col] = r
	}
	return gr
}

// NewExtraCell yields a new FloatingCell, in ExtraCells.
func (gr *HexGrid) NewExtraCell() FloatingCell {
	c := &FloatingHexCell{}
	gr.ExtraCells = append(gr.ExtraCells, c)
	return c
}

// yields the center of the hex at [row][col].
func (gr *HexGrid) center(row, col int, scale float32) (x, y float32) {
	// move columns over every two rows so 0,N+1 is always southeast from
	// 0,N.
	col = (col + (row / 2)) % gr.Width
	x = float32(col+1) * gr.hexWidth
	if row&1 == 0 {
		x -= gr.hexWidth / 2
	}
	y = gr.hexHeight * ((3 * float32(row)) + 2) / 4
	return (x + gr.ox) * scale, (y + gr.oy) * scale
}

func (gr *HexGrid) CellAt(x, y int) (l ILoc, c *Cell) {
	x, y = x-int(gr.ox), y-int(gr.oy)
	xInt, xOffset := math.Modf(float32(x) / gr.hexWidth)
	yInt, yOffset := math.Modf(float32(y) / gr.perHexHeight)
	xOffset -= 0.5
	xAway := math.Abs(xOffset) / 0.5
	//	fmt.Printf("%d, %d => %.0f [%.3f] [+%.3f], %.0f [%.3f]", x, y, xInt, xOffset, xAway, yInt, yOffset)
	x, y = int(xInt), int(yInt)
	if y%2 == 1 {
		if yOffset < 0.33 && (1-xAway) > yOffset*3 {
			y--
		} else {
			if xOffset < 0 {
				x--
			}
		}
	} else {
		if xAway > yOffset*3 {
			y--
			if xOffset < 0 {
				x--
			}
		}
	}
	//	fmt.Printf("=> %d, %d\n", x, y)
	if x >= 0 && x < gr.Width && y >= 0 && y < gr.Height {
		// handle the column offsets, coerce back into range
		x -= y / 2
		if x < 0 {
			x = (x % gr.Width) + gr.Width
		}
		return gr.Cell(x, y)
	} else {
		return ILoc{X: x, Y: y}, nil
	}
}

func (gr *HexGrid) Cell(x, y int) (ILoc, *Cell) {
	x, y = x%gr.Width, y%gr.Height
	if x < 0 {
		x += gr.Width
	}
	if y < 0 {
		y += gr.Height
	}
	l := ILoc{X: x, Y: y}
	return l, &gr.Cells[l.X][l.Y]
}

func (gr *HexGrid) Draw(target *ebiten.Image, scale float32) {
	textureSetup()
	CreateHexTextures()

	op := &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter, Filter: ebiten.FilterLinear}

	radius := gr.hexHeight * scale
	baseMatrix := IdentityAffine()
	baseMatrix.Rotate(math.Pi / 2)
	baseMatrix.Scale(radius, radius)
	offset := 0
	hd := hexDests[gr.render]
	for col, colCells := range gr.Cells {
		for row, cell := range colCells {
			tri := gr.vertices[offset : offset+3]
			r, g, b, _ := gr.Palette.Float32(cell.P)
			a := cell.Alpha
			var aff Affine
			if cell.Theta != 0 || cell.Scale != 1 {
				aff = IdentityAffine()
				aff.Rotate(cell.Theta + math.Pi/2)
				aff.Scale(cell.Scale*radius, cell.Scale*radius)
			} else {
				aff = baseMatrix
			}
			aff.E, aff.F = gr.center(row, col, scale)
			for j := 0; j < 3; j++ {
				tri[j].ColorR, tri[j].ColorG, tri[j].ColorB, tri[j].ColorA = r, g, b, a
				tri[j].DstX, tri[j].DstY = aff.Project(hd[j][0], hd[j][1])
			}
			offset += 3
		}
	}
	target.DrawTriangles(gr.vertices, gr.indices, hexTexture, op)
}

// Iterate runs fn on the entire grid.
func (gr *HexGrid) Iterate(fn GridFunc) {
	for i, col := range gr.Cells {
		for j := range col {
			fn(gr, ILoc{X: i, Y: j}, 1, &col[j])
		}
	}
}

func (gr *HexGrid) At(l ILoc) *Cell {
	return &gr.Cells[l.X][l.Y]
}

func (gr *HexGrid) IncP(l ILoc, n int) {
	sq := &gr.Cells[l.X][l.Y]
	sq.P = gr.Palette.Inc(sq.P, n)
}

func (gr *HexGrid) IncAlpha(l ILoc, a float32) {
	gr.Cells[l.X][l.Y].IncAlpha(a)
}

func (gr *HexGrid) IncTheta(l ILoc, t float32) {
	gr.Cells[l.X][l.Y].IncTheta(t)
}

// Splash does splashes in rings of radius min..max.
func (gr *HexGrid) Splash(l ILoc, min, max int, fn GridFunc) {
	if min < 0 {
		min = 0
	}
	if min == 0 {
		fn(gr, l, 0, &gr.Cells[l.X][l.Y])
		min++
	}
	for depth := min; depth <= max; depth++ {
		for idx, vec := range hexDirections {
			loc := gr.Add(l, vec.Times(depth))
			right := hexDirections[(idx+2)%len(hexDirections)]
			fn(gr, loc, depth, &gr.Cells[loc.X][loc.Y])
			for i := 1; i < depth; i++ {
				loc = gr.Add(loc, right)
				fn(gr, loc, depth, &gr.Cells[loc.X][loc.Y])
			}
		}
	}
}

func (gr *HexGrid) Neighbors(l ILoc, fn GridFunc) {
	gr.Splash(l, 1, 1, fn)
}

func (gr *HexGrid) Add(l ILoc, v IVec) ILoc {
	x, y := (l.X+v.X)%gr.Width, (l.Y+v.Y)%gr.Height
	if x < 0 {
		x += gr.Width
	}
	if y < 0 {
		y += gr.Height
	}
	return ILoc{X: x, Y: y}
}
