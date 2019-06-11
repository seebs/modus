package g

import (
	"fmt"
	"math/rand"

	math "github.com/chewxy/math32"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

type HexDir int

func (h HexDir) IVec() IVec {
	if h < 0 {
		return IVec{X: 0, Y: 0}
	}
	return hexDirections[h%6]
}

func (h HexDir) FVec() FVec {
	if h < 0 {
		return FVec{X: 0, Y: 0}
	}
	return hexFloatDirections[h%6]
}

func (h HexDir) Right() HexDir {
	return (h + 5) % 6
}

func (h HexDir) Left() HexDir {
	return (h + 1) % 6
}

var hexDirections = []IVec{
	{X: 1, Y: 0},
	{X: 1, Y: -1}, // +Z
	{X: 0, Y: -1},
	{X: -1, Y: 0},
	{X: -1, Y: 1}, // -Z
	{X: 0, Y: 1},
}

var hexFloatDirections = []FVec{
	{X: 1, Y: 0},
	{X: 0.5, Y: -1}, // +Z
	{X: -0.5, Y: -1},
	{X: -1, Y: 0},
	{X: -0.5, Y: 1}, // -Z
	{X: .5, Y: 1},
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
	palette             *Palette
	Cells               [][]HexCell
	ExtraCells          []*FloatingHexCell
	render              RenderType
	vertices            []ebiten.Vertex
	indices             []uint16
	hexDirs             [6][2]float32
	ox, oy              float32 // offset to draw grid at for centering
	Status              string
}

type HexCell struct {
	Cell
	HexMotion
}

// HexMotion reflects the motion of a thing on a hex grid.
type HexMotion struct {
	Dir  HexDir
	Dist float32
}

// NewDir yields a random hex direction
func (gr *HexGrid) NewDir() HexDir {
	return HexDir(rand.Int31n(6))
}

func (gr *HexGrid) Palette() *Palette {
	return gr.palette
}

// RandRow yields a random valid row.
func (gr *HexGrid) RandRow() int {
	return int(rand.Int31n(int32(gr.Height)))
}

// RandCol yields a random valid column.
func (gr *HexGrid) RandCol() int {
	return int(rand.Int31n(int32(gr.Width)))
}

func (gr *HexGrid) NewLoc() ILoc {
	return ILoc{X: gr.RandCol(), Y: gr.RandRow()}
}

// make a new hex grid. since hexes aren't interchangeable, we can't
// just flip X and Y...
//
// we start with the easy one: we use the flat ends, so the width of
// the row is trivial, except we need an extra half-hex, because a second
// row of hexes will be half a hex offset.
func newHexGrid(w int, r RenderType, p *Palette, sx, sy int) *HexGrid {
	textureSetup()

	gr := &HexGrid{render: r, Width: w, palette: p}
	var hexWidth float32
	var hexHeight float32
	var vHexes float32

	for {
		hexWidth = math.Floor(float32(sx) / (float32(gr.Width) + 0.5))
		// make it an even number, so the half-hex offset rows don't
		// look funny
		if int(hexWidth)&1 == 1 {
			hexWidth--
		}
		// the full height of a hex is 2/sqrt(3) times the width, but
		// each additional row costs only 3/4 that much.
		hexHeight = math.Floor(2 / math.Sqrt(3) * hexWidth)
		// the first row costs a full hexHeight. Every row after it costs
		// 3/4 of that. So:
		// h = (3n+1) * x/4
		// 4h/x = 3n + 1
		// (4h/x - 1) = 3n
		// (4h/x - 1)/3 = n
		vHexes = math.Floor((float32(sy)*4/hexHeight - 1) / 3)
		if gr.Width*int(vHexes)*3 < ebiten.MaxIndicesNum {
			break
		}
		gr.Width--
	}

	gr.hexWidth = float32(hexWidth)
	gr.hexHeight = float32(hexHeight)
	gr.perHexHeight = 3 * hexHeight / 4
	totalHeight := (3*vHexes + 1) * hexHeight / 4
	totalWidth := hexWidth * (float32(gr.Width) + 0.5)
	gr.ox, gr.oy = (float32(sx)-totalWidth)/2, (float32(sy)-totalHeight)/2
	gr.Height = int(vHexes)
	fmt.Printf("%dx%d => %d [*3]\n", gr.Width, gr.Height, gr.Width*gr.Height)

	// fmt.Printf("sx %d, w %d, hexWidth %.1f\n", sx, gr.Width, hexWidth)
	// fmt.Printf("hexHeight %.1f, sy %d, vHexes %f, total %f\n", hexHeight, sy, vHexes, totalHeight)
	// fmt.Printf("ox %.1f, oy %.1f\n", gr.ox, gr.oy)

	for i := 0; i < 6; i++ {
		ivec := HexDir(i).FVec()
		gr.hexDirs[i][0] = float32(ivec.X) * gr.hexWidth
		gr.hexDirs[i][1] = float32(ivec.Y) * gr.hexHeight * 3 / 4
	}

	gr.Cells = make([][]HexCell, gr.Width)
	gr.vertices = make([]ebiten.Vertex, 0, 3*gr.Width*gr.Height)
	gr.indices = make([]uint16, 0, 3*gr.Width*gr.Height)
	for col := range gr.Cells {
		r := make([]HexCell, gr.Height)
		for row := range r {
			r[row] = HexCell{Cell: Cell{Alpha: 1, Scale: .95}}
			offset := uint16(len(gr.vertices))
			gr.vertices = append(gr.vertices, hexData.vsByR[gr.render]...)
			gr.indices = append(gr.indices, offset+0, offset+1, offset+2)
		}
		gr.Cells[col] = r
	}
	fmt.Printf("indices: %d\n", len(gr.indices))
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

func (gr *HexGrid) CellAt(x, y int) (l ILoc, c *HexCell) {
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

func (gr *HexGrid) Cell(x, y int) (ILoc, *HexCell) {
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

func (gr *HexGrid) CenterFor(x, y int) (x1, y1 float32) {
	return gr.center(x, y, 1.0)
}

func (gr *HexGrid) Draw(target *ebiten.Image, scale float32) {
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
			r, g, b, _ := gr.palette.Float32(cell.P)
			a := cell.Alpha
			var aff Affine
			aff = baseMatrix
			if cell.Theta != 0 {
				aff.Rotate(cell.Theta)
			}
			if cell.Scale != 1 {
				aff.Scale(cell.Scale, cell.Scale)
			}
			aff.E, aff.F = gr.center(row, col, scale)
			if cell.Dist != 0 {
				aff.E += gr.hexDirs[cell.Dir][0] * cell.Dist
				aff.F += gr.hexDirs[cell.Dir][1] * cell.Dist
			}
			adj := make([]string, 6)
			for i := 0; i < 6; i++ {
				c, _ := gr.Neighbor(ILoc{X: col, Y: row}, HexDir(i), false)
				if c != nil {
					adj[i] = "Y"
				} else {
					adj[i] = "N"
				}
			}
			if false {
				ebitenutil.DebugPrintAt(target, fmt.Sprintf("  %s  %s\n%s %2d,%2d %s\n  %s  %s",
					adj[2], adj[1], adj[3],
					col, row,
					adj[0], adj[4], adj[5]), int(aff.E-25), int(aff.F-25))
			}
			for j := 0; j < 3; j++ {
				tri[j].ColorR, tri[j].ColorG, tri[j].ColorB, tri[j].ColorA = r, g, b, a
				tri[j].DstX, tri[j].DstY = aff.Project(hd[j][0], hd[j][1])
			}
			offset += 3
		}
	}
	target.DrawTriangles(gr.vertices, gr.indices, hexData.img, op)
	ebitenutil.DebugPrint(target, gr.Status)
}

// Iterate runs fn on the entire grid.
func (gr *HexGrid) Iterate(fn GridFunc) {
	for i, col := range gr.Cells {
		for j := range col {
			fn(gr, ILoc{X: i, Y: j}, 1, &col[j].Cell)
		}
	}
}

func (gr *HexGrid) At(l ILoc) *Cell {
	return &gr.Cells[l.X][l.Y].Cell
}

func (gr *HexGrid) IncP(l ILoc, n int) Paint {
	sq := &gr.Cells[l.X][l.Y]
	sq.P = gr.palette.Inc(sq.P, n)
	return sq.P
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
		fn(gr, l, 0, &gr.Cells[l.X][l.Y].Cell)
		min++
	}
	for depth := min; depth <= max; depth++ {
		for idx, vec := range hexDirections {
			loc, _ := gr.Add(l, vec.Times(depth))
			right := hexDirections[(idx+2)%len(hexDirections)]
			fn(gr, loc, depth, &gr.Cells[loc.X][loc.Y].Cell)
			for i := 1; i < depth; i++ {
				loc, _ = gr.Add(loc, right)
				fn(gr, loc, depth, &gr.Cells[loc.X][loc.Y].Cell)
			}
		}
	}
}

func (gr *HexGrid) Neighbors(l ILoc, fn GridFunc) {
	gr.Splash(l, 1, 1, fn)
}

// Add attempts to determine not only what the new location
// would be, but whether it wrapped. This is difficult because
// of the strange offset grid compromise; as you move down the
// screen (positive Y), X coordinate 0 gradually drifts to the
// right, but wrapping stays at the screen edges.
//
// We assume the location starts out in the normalized range.
func (gr *HexGrid) Add(l ILoc, v IVec) (loc ILoc, wrapped bool) {
	loc.X, loc.Y = (l.X+v.X)%gr.Width, (l.Y+v.Y)%gr.Height
	if loc.X < 0 {
		// this may not constitute "wrapping" if we're in a
		// line where 0 is somewhere in the mid-screen.
		loc.X += gr.Width
	}
	// negative Y is always a wrap at the top edge
	if loc.Y < 0 {
		loc.Y += gr.Height
		return loc, true
	}
	// Y should have increased, but is now smaller; that also wrapped.
	if loc.Y < l.Y && v.Y > 0 {
		return loc, true
	}
	// X is effectively incremented by Y/2. v.X * 2 + v.Y has the same
	// sign as effective-X.
	sx := (v.X * 2) + v.Y

	tx1 := (l.X + l.Y/2) % gr.Width
	tx2 := (loc.X + loc.Y/2) % gr.Width
	if (tx2-tx1)*sx < 0 {
		return loc, true
	}
	return loc, false
}

// Neighbor yields the neighbor in the given direction, wrapping if wrap is
// true, otherwise returning nil for edge cases.
func (gr *HexGrid) Neighbor(old ILoc, d HexDir, wrap bool) (c *HexCell, loc ILoc) {
	loc, wrapped := gr.Add(old, hexDirections[d])
	if wrapped && !wrap {
		return nil, loc
	}
	return &gr.Cells[loc.X][loc.Y], loc
}
