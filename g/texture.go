package g

import (
	"image"
	"log"
	"sync"

	"image/color"

	math "github.com/chewxy/math32"
	"github.com/hajimehoshi/ebiten"
)

// Here, we create textures for other parts of g to use.

// hexRender represents one of the rings of a hex, which are
// drawn as opaque values out-to-in, allowing us to make rings.
// radius ranges from 0 to 1, value from 0 to 255.
type hexRender struct {
	radius float32
	value  uint8
}

const (
	hexRadius = 72
)

var (
	// line textures: each line gets a 32x32 box, although only the middle
	// 28x28 are supposed to be used as the texture. The idea is to have
	// boundaries around the part of the texture we use to keep the
	// edges/ends from being rendered darker when rendered with
	// FilterLinear, even though actually I don't plan to use FilterLinear
	// anymore anyway.
	lineDepths = [4][32]byte{
		{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		{127, 127, 127, 127, 127, 127, 127, 127, 127, 127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127, 127, 127, 127, 127, 127, 127, 127, 127, 127},
		{63, 63, 63, 78, 93, 107, 122, 137, 152, 166, 181, 196, 211, 225, 240, 255, 255, 240, 225, 211, 196, 181, 166, 152, 137, 122, 107, 93, 78, 63, 63, 63},
		{15, 15, 15, 33, 52, 70, 89, 107, 126, 144, 163, 181, 200, 218, 237, 255, 255, 237, 218, 200, 181, 163, 144, 126, 107, 89, 70, 52, 33, 15, 15, 15},
	}
	// squareRenders defines the brightness/intensity of rings around the central point
	// of a square.
	squareRenders = [4][32]byte{
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
	dotRenders = []func(x, y float32) uint8{
		func(x, y float32) uint8 {
			dx, dy := x, y
			grey := 1 - math.Sqrt((dx*dx)+(dy*dy))
			if grey < 0 {
				grey = 0
			}
			if grey > 1 {
				grey = 1
			}
			return uint8(grey * 255)
		},
		func(x, y float32) uint8 {
			dx, dy := math.Abs(x), math.Abs(y)
			dx2, dy2 := dx*dx, dy*dy
			m := dx
			if dy < m {
				m = dy
			}
			// We multiply in an extra copy of whichever of dx/dy is smaller, to generate a sort of concave
			// look.
			grey := (1 - math.Sqrt(dx2+dy2)) * (1.05 - math.Sqrt(m)) * (1.05 - math.Sqrt(dx)) * (1.05 - math.Sqrt(dy))

			if grey < 0 {
				grey = 0
			}
			if grey > 1 {
				grey = 1
			}
			return uint8(grey * 255)
		},
	}
	hexRenders = [][]hexRender{
		[]hexRender{
			{radius: 1, value: 255},
		},
		[]hexRender{
			{radius: 1, value: 192},
			{radius: 0.875, value: 220},
		},
		[]hexRender{
			{radius: 1, value: 192},
			{radius: 0.875, value: 220},
			{radius: 0.75, value: 192},
			{radius: 0.625, value: 128},
			{radius: 0.5, value: 64},
		},
		[]hexRender{
			{radius: 1, value: 64},
			{radius: 0.875, value: 96},
			{radius: 0.75, value: 128},
			{radius: 0.625, value: 160},
			{radius: 0.5, value: 192},
			{radius: 0.375, value: 224},
			{radius: 0.25, value: 225},
		},
		[]hexRender{
			{radius: 1, value: 192},
			{radius: 0.875, value: 220},
		},
		[]hexRender{
			{radius: 1, value: 220},
			{radius: 0.875, value: 192},
			{radius: 0.75, value: 128},
			{radius: 0.625, value: 96},
		},
	}
	baseHexDests = [2][][2]float32{
		{
			{1.5, -hexHeightScale},
			{0, 2 * hexHeightScale},
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

	squareData *textureWithVertices
	hexData    *textureWithVertices
	lineData   *textureWithVertices
	dotData    *textureWithVertices
	solidData  *textureWithVertices
)

// textureWithVertices holds an image representing multiple render types,
// and corresponding vertex values by render type. types == len(vsByR).
type textureWithVertices struct {
	types  int
	img    *ebiten.Image
	vsByR  [][]ebiten.Vertex
	scales []float32
}

func hexTextureWidth(cols int) int {
	if cols == 0 {
		return 2
	}
	if cols < 0 {
		cols *= -1
	}
	return int(math.Ceil(3+1.5*(float32(cols)-1))*hexRadius) + 2*(cols+1)
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
	return int(math.Ceil(float32(hexHeightScale)*4+(float32(hexHeightScale)*3)*(float32(rows)-1))*hexRadius) + 2*(rows+1)
}

func createSquareTextures() (*textureWithVertices, error) {
	twv := &textureWithVertices{}
	img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 128, Y: 128}})
	for render := 0; render < 4; render++ {
		offsetX := (render % 2) * 64
		offsetY := (render / 2) * 64
		offsetXf := float32(offsetX)
		offsetYf := float32(offsetY)
		scalef := float32(60)
		c := 32
		for r := 0; r < 32; r++ {
			// zero value is transparent black
			var col color.RGBA
			v := squareRenders[render][r]
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
		twv.vsByR = append(twv.vsByR, squareVertices)
	}
	twv.types = len(twv.vsByR)
	var err error
	twv.img, err = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	if err != nil {
		return nil, err
	}
	return twv, nil
}

func createLineTextures() (*textureWithVertices, error) {
	twv := &textureWithVertices{}
	img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 64, Y: 64}})
	for depth := 0; depth < 4; depth++ {
		offsetX := (depth%2)*32 + 2
		offsetY := (depth/2)*32 + 2
		offsetXf := float32(offsetX)
		offsetYf := float32(offsetY)
		scalef := float32(28)
		for r := 0; r < 32; r++ {
			v := lineDepths[depth][r]
			col := color.RGBA{v, v, v, v}
			for c := 0; c < 14; c++ {
				img.Set(offsetX+c*2, offsetY+r-1, col)
				img.Set(offsetX+c*2+1, offsetY+r-1, col)
			}
		}
		lineVertices := make([]ebiten.Vertex, 6)
		for i := 0; i < 6; i++ {
			lineVertices[i] = baseVertices[i]
			// pull X in from the ends, so it doesn't dim at the ends.
			lineVertices[i].SrcX = offsetXf + 2 + lineVertices[i].SrcX*(scalef-4)
			lineVertices[i].SrcY = offsetYf + lineVertices[i].SrcY*scalef
		}
		twv.vsByR = append(twv.vsByR, lineVertices)
	}
	var err error
	twv.img, err = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	if err != nil {
		return nil, err
	}
	return twv, err
}

func createTextures() {
	var err error
	squareData, err = createSquareTextures()
	if err != nil {
		log.Fatalf("couldn't make square textures: %v", err)
	}
	lineData, err = createLineTextures()
	if err != nil {
		log.Fatalf("couldn't make line textures: %v", err)
	}
	hexData, err = createHexTextures()
	if err != nil {
		log.Fatalf("couldn't make hex textures: %v", err)
	}
	dotData, err = createDotTextures()
	if err != nil {
		log.Fatalf("couldn't make dot textures: %v", err)
	}
	solidData, err = createSolidTexture()
	if err != nil {
		log.Fatalf("couldn't make solid texture: %v", err)
	}
}

func createHexTextures() (*textureWithVertices, error) {
	twv := &textureWithVertices{}
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
	for n := 2; n < len(hexRenders); n++ {
		// if this already fits, we don't need to do anything
		if n <= (rows * cols) {
			continue
		}
		// compute area if we add a column:
		nRows := int(math.Ceil(float32(n) / float32(cols+1)))
		aCol := npo2(hexTextureWidth(cols+1)+2) * npo2(hexTextureHeight(nRows)+2)
		// but what if adding a row is better?
		nCols := int(math.Ceil(float32(n) / float32(rows+1)))
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
	var err error
	twv.img, err = ebiten.NewImage(hexW, hexH, ebiten.FilterLinear)

	if err != nil {
		return nil, err
	}
	// Populate base vertices. actual rendering happens later when the user
	// calls CreateHexTextures after ebiten is up.
	twv.vsByR = make([][]ebiten.Vertex, len(hexRenders))

	for render := 0; render < len(hexRenders); render++ {
		var hd [][2]float32
		vs := make([]ebiten.Vertex, 0, 3)
		row, col := render/hexCols, render%hexCols
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
		twv.vsByR[render] = vs
		hexDests = append(hexDests, hd)
	}
	return twv, nil
}

func createSolidTexture() (*textureWithVertices, error) {
	img, err := ebiten.NewImage(8, 8, ebiten.FilterDefault)
	if err != nil {
		return nil, err
	}
	_ = img.Fill(color.RGBA{255, 255, 255, 255})
	// special case: since it's a solid texture, just use the same point repeatedly.
	v := ebiten.Vertex{SrcX: 0.5, SrcY: 0.5, ColorR: 1.0, ColorG: 1.0, ColorB: 1.0, ColorA: 1.0}
	vs := []ebiten.Vertex{v, v, v}
	twv := &textureWithVertices{img: img, vsByR: [][]ebiten.Vertex{vs}, types: 1}
	return twv, nil
}

func createDotTextures() (*textureWithVertices, error) {
	twv := &textureWithVertices{}
	// we make a square-ish thing: compute sqrt of the dotRenders set, then
	// use that as our width-in-tiles.
	tilesAcross := int(math.Sqrt(float32(len(dotRenders))))
	tilesDown := (len(dotRenders) + tilesAcross - 1) / tilesAcross
	img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 64 * tilesAcross, Y: 64 * tilesDown}})
	scalef := float32(62)
	sums := make([]int, len(dotRenders))
	maxSum := 0

	for render := 0; render < len(dotRenders); render++ {
		fn := dotRenders[render]
		offsetX := (render % tilesAcross) * 64
		offsetY := (render / tilesAcross) * 64
		offsetXf := float32(offsetX)
		offsetYf := float32(offsetY)
		sum := 0
		// fill in the insides, roughly:
		for i := 1; i < 63; i++ {
			for j := 1; j < 63; j++ {
				v := fn((float32(i)-31.5)/30.5, (float32(j)-31.5)/30.5)
				sum += int(v)
				col := color.RGBA{v, v, v, v}
				img.Set(i+offsetX, j+offsetY, col)
			}
		}
		sums[render] = sum
		if sum > maxSum {
			maxSum = sum
		}
		dotVertices := make([]ebiten.Vertex, 4)
		for i := 0; i < 4; i++ {
			dotVertices[i] = baseVertices[i]
			dotVertices[i].SrcX = offsetXf + 2 + dotVertices[i].SrcX*scalef
			dotVertices[i].SrcY = offsetYf + 2 + dotVertices[i].SrcY*scalef
		}
		twv.vsByR = append(twv.vsByR, dotVertices)
	}
	twv.scales = make([]float32, len(dotRenders))
	for i := 0; i < len(dotRenders); i++ {
		if sums[i] > 0 {
			twv.scales[i] = math.Sqrt(float32(maxSum) / float32(sums[i]))
		} else {
			twv.scales[i] = 1
		}
	}
	twv.types = len(twv.vsByR)
	var err error
	twv.img, err = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	if err != nil {
		return nil, err
	}
	return twv, nil
}

// CreateHexTextures will actually create hex textures. It must be called when
// ebiten is actually running, such as from a hex grid's draw loop.
func CreateHexTextures() {
	createHexesOnce.Do(renderHexTextures)
}

func renderHexTextures() {
	w, h := hexData.img.Size()
	hexTex2x, _ := ebiten.NewImage(w*2, h*2, ebiten.FilterLinear)
	for depth, depths := range hexRenders {
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
	_ = hexData.img.DrawImage(hexTex2x, op)
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
	target.DrawTriangles(vertices, indices, solidData.img, nil)
}

var (
	createTexturesOnce sync.Once
	createHexesOnce    sync.Once
)

func textureSetup() {
	createTexturesOnce.Do(createTextures)
}
