package g

import (
	"math"

	"github.com/hajimehoshi/ebiten"
)

type HexGrid struct {
	Width, Height   int
	hWidth, hHeight float32 // scaling size of
	Palette         *Palette
	Cells           [][]Cell
	render          RenderType
	vertices        []ebiten.Vertex
	indices         []uint16
	ox, oy          float32 // offset to draw grid at for centering
}

// make a new hex grid. since hexes aren't interchangeable, we can't
// just flip X and Y...
//
// we start with the easy one: we use the flat ends, so the width of
// the row is trivial, except we need an extra half-hex, because a second
// row of hexes will be half a hex offset.
func newHexGrid(w int, r RenderType, sx, sy int) *HexGrid {
	gr := &HexGrid{render: r, Width: w}
	hexWidth := math.Floor(float64(sx) / (float64(w) + 0.5))
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
	vHexes := math.Floor((float64(sy)*4/hexHeight - 1) / 3)

	gr.hWidth = float32(hexWidth)
	gr.hHeight = float32(hexHeight)
	totalHeight := float32((3*vHexes + 1) * hexHeight / 4)
	totalWidth := float32(hexWidth * (float64(w) + 0.5))
	gr.ox, gr.oy = (float32(sx)-totalWidth)/2, (float32(sy)-totalHeight)/2
	gr.Height = int(vHexes)

	// fmt.Printf("sx %d, w %d, hexWidth %.1f\n", sx, w, hexWidth)
	// fmt.Printf("hexHeight %.1f, sy %d, vHexes %f, total %f\n", hexHeight, sy, vHexes, totalHeight)
	// fmt.Printf("ox %.1f, oy %.1f\n", gr.ox, gr.oy)

	gr.Cells = make([][]Cell, gr.Height)
	gr.vertices = make([]ebiten.Vertex, 0, 3*gr.Width*gr.Height)
	gr.indices = make([]uint16, 3*gr.Width*gr.Height)
	for row := range gr.Cells {
		r := make([]Cell, gr.Width)
		for col := range r {
			r[col] = Cell{Alpha: 1, Scale: .95}
			offset := uint16(len(gr.vertices))
			gr.vertices = append(gr.vertices, hexVerticesByDepth[gr.render]...)
			gr.indices = append(gr.indices, offset+0, offset+1, offset+2)
		}
		gr.Cells[row] = r
	}
	return gr
}

//
func (gr *HexGrid) center(row, col int) (x, y float32) {
	x = float32(col+1) * gr.hWidth
	if row&1 != 0 {
		x -= gr.hWidth / 2
	}
	y = gr.hHeight * ((3 * float32(row)) + 2) / 4
	return x + gr.ox, y + gr.oy
}

func (gr *HexGrid) Draw(target *ebiten.Image, scale float64) {
	textureSetup()
	CreateHexTextures()

	op := &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter, Filter: ebiten.FilterLinear}

	radius := gr.hHeight
	baseMatrix := IdentityAffine()
	baseMatrix.Rotate(math.Pi / 2)
	baseMatrix.Scale(radius, radius)
	offset := 0
	hd := hexDests[gr.render]
	for row, rowCells := range gr.Cells {
		for col, cell := range rowCells {
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
			aff.E, aff.F = gr.center(row, col)
			for j := 0; j < 3; j++ {
				tri[j].ColorR, tri[j].ColorG, tri[j].ColorB, tri[j].ColorA = r, g, b, a
				tri[j].DstX, tri[j].DstY = aff.Project(hd[j][0], hd[j][1])
			}
			offset += 3
		}
	}
	target.DrawTriangles(gr.vertices, gr.indices, hexTexture, op)
}
