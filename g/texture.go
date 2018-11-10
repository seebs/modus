package g

import (
	"image"
	"log"
	"math"
	"sync"

	"image/color"

	"github.com/hajimehoshi/ebiten"
)

// Here, we create textures for other parts of g to use.

// hexDepth represents one of the rings of a hex, which are
// drawn as opaque values out-to-in, allowing us to make rings.
// radius ranges from 0 to 1, value from 0 to 255.
type hexDepth struct {
	radius float32
	value  uint8
}

const (
	hexRadius = 72
)

var (
	// line textures: each line gets a 32x32 box, which is a pixel-doubled
	// 16x16 box, although only the middle 14x14 (28x28) are supposed to be
	// used as the texture. The idea is to have boundaries around the part of
	// the texture we use to keep the edges/ends from being rendered darker
	// due when rendered with FilterLinear, even though actually I don't plan
	// to use FilterLinear anymore anyway.
	lineDepths = [4][16]byte{
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		{127, 127, 127, 127, 127, 255, 255, 255, 255, 255, 255, 127, 127, 127, 127, 127},
		{85, 85, 85, 127, 127, 127, 255, 255, 255, 255, 127, 127, 127, 85, 85, 85},
		{63, 63, 63, 127, 127, 191, 191, 255, 255, 191, 191, 127, 127, 63, 63, 63},
	}
	// squares store a series of rings around the central point.
	squareDepths = [4][32]byte{
		// white
		{
			255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
			255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
		},
		// the original: faded grey, with a bright line and a dimmer line at the edge
		{
			224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224, 224,
			224, 224, 224, 224, 224, 224, 224, 224, 240, 240, 240, 240, 192, 192, 192, 192,
		},
		// fancy!
		{
			96, 96, 96, 96, 128, 128, 192, 192, 200, 200, 128, 200, 200, 200, 200, 224,
			224, 224, 224, 240, 240, 240, 224, 224, 200, 192, 192, 192, 64, 0, 0, 0,
		},
	}
	baseVertices = []ebiten.Vertex{
		// basic quad vertex defaults:
		{SrcX: 0, SrcY: 0, ColorA: 1.0}, // prev + nx,ny
		{SrcX: 0, SrcY: 1, ColorA: 1.0}, // prev - nx,ny
		{SrcX: 1, SrcY: 0, ColorA: 1.0}, // next + nx,ny
		{SrcX: 1, SrcY: 1, ColorA: 1.0}, // next - nx,ny
		// for line segments in fancy mode:
		{SrcX: 0, SrcY: 0.5, ColorA: 1.0}, // prev
		{SrcX: 1, SrcY: 0.5, ColorA: 1.0}, // next
	}
	hexDepths = [][]hexDepth{
		[]hexDepth{
			{radius: 1, value: 255},
		},
		[]hexDepth{
			{radius: 1, value: 192},
			{radius: 0.875, value: 220},
		},
		[]hexDepth{
			{radius: 1, value: 220},
			{radius: 0.875, value: 0},
			{radius: 0.75, value: 128},
			{radius: 0.625, value: 96},
		},
		[]hexDepth{
			{radius: 1, value: 128},
		},
		[]hexDepth{
			{radius: 1, value: 192},
			{radius: 0.875, value: 220},
		},
		[]hexDepth{
			{radius: 1, value: 220},
			{radius: 0.875, value: 192},
			{radius: 0.75, value: 128},
			{radius: 0.625, value: 96},
		},
	}
	baseHexDests = [2][][2]float32{
		{
			{1.5, -hexHeightScale},
			{0, -2 * hexHeightScale},
			{-1.5, -hexHeightScale},
		},
		{
			{-1.5, hexHeightScale},
			{0, -2 * hexHeightScale},
			{1.5, hexHeightScale},
		},
	}
	hexDests [][][2]float32
	// the height of the flat side of the hex, from the center
	hexHeightScale = float32(math.Sqrt(3) / 2)
	// hexHeight is the offset we'll actually use, just so it's a consistent
	// integer range for things using integer-ish points.
	hexHeight        = int(math.Sqrt(3) / 2 * hexRadius)
	hexRows, hexCols int

	triVerticesByDepth    [][]ebiten.Vertex
	squareVerticesByDepth [][]ebiten.Vertex
	hexVerticesByDepth    [][]ebiten.Vertex
	lineTexture           *ebiten.Image
	squareTexture         *ebiten.Image
	hexTexture            *ebiten.Image
	solidTexture          *ebiten.Image
)

func hexTextureWidth(cols int) int {
	if cols == 0 {
		return 2
	}
	if cols < 0 {
		cols *= -1
	}
	return int(math.Ceil(3+1.5*(float64(cols)-1))*hexRadius) + 2*(cols+1)
}

func hexTextureXOffset(col int) int {
	return ((3*hexRadius/2)+2)*col + 2
}

func hexTextureYOffset(row int) int {
	return 3*(hexHeight+2)*row + 2
}

func hexTextureHeight(rows int) int {
	if rows == 0 {
		return 2
	}
	if rows < 0 {
		rows *= -1
	}
	return int(math.Ceil(float64(hexHeightScale)*4+(float64(hexHeightScale)*3)*(float64(rows)-1))*hexRadius) + 2*(rows+1)
}

func createTextures() {
	img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 64, Y: 64}})
	for depth := 0; depth < 4; depth++ {
		offsetX := (depth%2)*32 + 2
		offsetY := (depth/2)*32 + 2
		offsetXf := float32(offsetX)
		offsetYf := float32(offsetY)
		scalef := float32(28)
		for r := 0; r < 32; r++ {
			v := lineDepths[depth][r/2]
			col := color.RGBA{v, v, v, v}
			for c := 0; c < 14; c++ {
				img.Set(offsetX+c*2, offsetY+r-1, col)
				img.Set(offsetX+c*2+1, offsetY+r-1, col)
			}
		}
		triVertices := make([]ebiten.Vertex, 6)
		for i := 0; i < 6; i++ {
			triVertices[i] = baseVertices[i]
			// pull X in from the ends, so it doesn't dim at the ends.
			triVertices[i].SrcX = offsetXf + 2 + triVertices[i].SrcX*(scalef-4)
			triVertices[i].SrcY = offsetYf + triVertices[i].SrcY*scalef
		}
		triVerticesByDepth = append(triVerticesByDepth, triVertices)
	}
	var err error
	lineTexture, err = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	if err != nil {
		log.Fatalf("couldn't create image: %s", err)
	}
	img = image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 128, Y: 128}})
	for depth := 0; depth < 4; depth++ {
		offsetX := (depth % 2) * 64
		offsetY := (depth / 2) * 64
		offsetXf := float32(offsetX)
		offsetYf := float32(offsetY)
		scalef := float32(60)
		c := 32
		for r := 0; r < 32; r++ {
			// zero value is transparent black
			var col color.RGBA
			v := squareDepths[depth][r]
			if v != 0 {
				// leave 0 values transparent
				col = color.RGBA{v, v, v, 255}
			}
			// radius 0 = the points immediately adjacent to center square,
			// thus, [c-r-1, c-r-1] through [c+r][c+r], inclusive
			min := c - r - 1
			max := c + r
			for i := min; i <= max; i++ {
				img.Set(offsetX+i, offsetY+min, col)
				img.Set(offsetX+i, offsetY+max, col)
				img.Set(offsetX+min, offsetY+i, col)
				img.Set(offsetX+max, offsetY+i, col)
			}
		}
		squareVertices := make([]ebiten.Vertex, 4)
		for i := 0; i < 4; i++ {
			squareVertices[i] = baseVertices[i]
			squareVertices[i].SrcX = offsetXf + 2 + squareVertices[i].SrcX*scalef
			squareVertices[i].SrcY = offsetYf + 2 + squareVertices[i].SrcY*scalef
		}
		squareVerticesByDepth = append(squareVerticesByDepth, squareVertices)
	}
	squareTexture, err = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	if err != nil {
		log.Fatalf("couldn't create image: %s", err)
	}
	// hexes alternate between triangle-up and triangle-down. given a hex of
	// radius 1, the leftmost/rightmost point of its triangle will be 1.5
	// from its center, the flat side of its triangle will be (sqrt(3)/2)
	// above its center, and the point of its triangle will be sqrt(3)
	// below its center.
	// the next hex should have a center (sqrt(3)/2) down, and 1.5 to the
	// right. so, the bottom right of the triangle bounding the second hex
	// is a touch over 2*sqrt(3) down, and 4.5 to the right, from the
	// starting corner. each additional hex to the right adds 1.5.
	// if there's another row of hexes, they should be parallel, with
	// centers just over (3*sqrt(3)/2) below the previous centers.
	//
	// so, we start with a 2sqrt3 height, 4.5 width, for two hexes. each
	// additional hex adds 1.5 width. each additional row adds 1.5sqrt3
	// height.

	rows := 1
	cols := 2
	for n := 2; n < len(hexDepths); n++ {
		// if this already fits, we don't need to do anything
		if n <= (rows * cols) {
			continue
		}
		// compute area if we add a column:
		nRows := int(math.Ceil(float64(n) / float64(cols+1)))
		aCol := npo2(hexTextureWidth(cols+1)+2) * npo2(hexTextureHeight(nRows)+2)
		// but what if adding a row is better?
		nCols := int(math.Ceil(float64(n) / float64(rows+1)))
		aRow := npo2(hexTextureWidth(nCols)+2) * npo2(hexTextureHeight(rows+1)+2)
		// fmt.Printf("n = %d: aRow [%d+2x%d+2]: %d, aCol [%d+2x%d+2]: %d", n, nCols, rows+1, aRow, cols+1, nRows, aCol)
		if aRow == aCol {
			// favor squares as tiebreaker
			if float32(rows)*hexHeightScale > float32(cols) {
				aRow++
			} else {
				aCol++
			}
		}
		if aRow >= aCol {
			cols++
			rows = nRows
			// fmt.Printf(", adding col [%dx%d]\n", cols, rows)
		} else {
			rows++
			cols = nCols
			// fmt.Printf(", adding row [%dx%d]\n", cols, rows)
		}
	}
	hexRows, hexCols = rows, cols
	// +2px offset per
	hexW := npo2(hexTextureWidth(cols) + 2)
	hexH := npo2(hexTextureHeight(rows) + 2)
	// fmt.Printf("total texture size: %d x %d => %d x %d\n", cols, rows, hexW, hexH)

	// create hexTexture now, but it won't actually be populated yet.
	hexTexture, err = ebiten.NewImage(hexW, hexH, ebiten.FilterLinear)

	if err != nil {
		log.Fatalf("couldn't create image: %s", err)
	}
	// Populate base vertices. actual rendering happens later when the user
	// calls CreateHexTextures after ebiten is up.
	hexVerticesByDepth = make([][]ebiten.Vertex, len(hexDepths))

	for depth := 0; depth < len(hexDepths); depth++ {
		var hd [][2]float32
		vs := make([]ebiten.Vertex, 0, 3)
		row, col := depth/hexCols, depth%hexCols
		// the top-left point of a down-pointing triangle around the hex
		ox, oy := hexTextureXOffset(col), hexTextureYOffset(row)
		if col&1 != 0 {
			hd = baseHexDests[1]
			// up-pointing triangle
			vs = append(vs, ebiten.Vertex{DstX: hd[0][0], DstY: hd[0][1], SrcX: float32(ox), SrcY: float32(oy + 3*hexHeight), ColorA: 1})
			vs = append(vs, ebiten.Vertex{DstX: hd[1][0], DstY: hd[1][1], SrcX: float32(ox + (3*hexRadius)/2), SrcY: float32(oy), ColorA: 1})
			vs = append(vs, ebiten.Vertex{DstX: hd[2][0], DstY: hd[2][1], SrcX: float32(ox + 3*hexRadius), SrcY: float32(oy + 3*hexHeight), ColorA: 1})
		} else {
			hd = baseHexDests[0]
			// down-pointing triangle
			vs = append(vs, ebiten.Vertex{DstX: hd[0][0], DstY: hd[0][1], SrcX: float32(ox), SrcY: float32(oy), ColorA: 1})
			vs = append(vs, ebiten.Vertex{DstX: hd[1][0], DstY: hd[1][1], SrcX: float32(ox + (3*hexRadius)/2), SrcY: float32(oy + 3*hexHeight), ColorA: 1})
			vs = append(vs, ebiten.Vertex{DstX: hd[2][0], DstY: hd[2][1], SrcX: float32(ox + 3*hexRadius), SrcY: float32(oy), ColorA: 1})

		}
		// fmt.Printf("hex depth %d [@%d,%d]: %d, %d\n", depth, col, row, ox, oy)
		hexVerticesByDepth[depth] = vs
		hexDests = append(hexDests, hd)
	}

	solidTexture, err = ebiten.NewImage(16, 16, ebiten.FilterDefault)
	if err != nil {
		log.Fatalf("couldn't create image: %s", err)
	}
	solidTexture.Fill(color.RGBA{255, 255, 255, 255})
}

// CreateHexTextures will actually create hex textures. It must be called when
// ebiten is actually running, such as from a hex grid's draw loop.
func CreateHexTextures() {
	createHexesOnce.Do(createHexTextures)
}

func createHexTextures() {
	w, h := hexTexture.Size()
	hexTex2x, _ := ebiten.NewImage(w*2, h*2, ebiten.FilterLinear)
	for depth, depths := range hexDepths {
		row, col := depth/hexCols, depth%hexCols
		// the top-left point of a down-pointing triangle around the hex
		ox, oy := hexTextureXOffset(col)*2, hexTextureYOffset(row)*2
		if col&1 != 0 {
			oy += int(hexHeight) * 2
		}
		for _, hd := range depths {
			drawHexAround(ox+(hexRadius*3), oy+hexHeight*2, hd.radius, hd.value, hexTex2x)
		}
		// fmt.Printf("hex depth %d [@%d,%d]: %d, %d\n", depth, col, row, ox, oy)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.5, 0.5)
	hexTexture.DrawImage(hexTex2x, op)
	hexTex2x.Dispose()
}

var hexOffsets = [][2]float32{
	{hexRadius, 0},
	{hexRadius / 2, float32(hexHeight)},
	{-hexRadius / 2, float32(hexHeight)},
	{-hexRadius, 0},
	{-hexRadius / 2, -float32(hexHeight)},
	{hexRadius / 2, -float32(hexHeight)},
}

func drawHexAround(xI, yI int, radius float32, value uint8, target *ebiten.Image) {
	x := float32(xI)
	y := float32(yI)
	v := float32(value) / 255
	vertices := make([]ebiten.Vertex, 0, 7)
	indices := make([]uint16, 0, 18)

	for i := 0; i < 6; i++ {
		sx := 0
		if i > 0 {
			sx = 1
		}
		sy := 0
		if i%2 == 1 {
			sy = 1
		}
		vertices = append(vertices, ebiten.Vertex{
			SrcX: float32(sx), SrcY: float32(sy),
			DstX: x + (hexOffsets[i][0] * radius), DstY: y + (hexOffsets[i][1] * radius),
			ColorR: v,
			ColorB: v,
			ColorG: v,
			ColorA: 1.0,
		})
		next := (i + 1) % 6
		indices = append(indices, 6, uint16(i), uint16(next))
	}
	vertices = append(vertices, ebiten.Vertex{
		SrcX: 0, SrcY: 0,
		DstX: x, DstY: y,
		ColorR: v,
		ColorB: v,
		ColorG: v,
		ColorA: 1.0,
	})
	target.DrawTriangles(vertices, indices, solidTexture, nil)
}

var (
	createTexturesOnce sync.Once
	createHexesOnce    sync.Once
)

func textureSetup() {
	createTexturesOnce.Do(createTextures)
}
