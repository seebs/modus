package g

import (
	"github.com/hajimehoshi/ebiten"
)

type HexGrid struct {
	Depth int
}

func NewHexGrid(depth int) *HexGrid {
	return &HexGrid{Depth: depth}
}

func (h *HexGrid) Draw(screen *ebiten.Image) {
	textureSetup()
	CreateHexTextures()
	o2 := &ebiten.DrawImageOptions{}
	o2.GeoM.Reset()
	o2.GeoM.Translate(128, 32)
	screen.DrawImage(hexTexture, o2)
	op := &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter, Filter: ebiten.FilterLinear}
	vs := make([]ebiten.Vertex, 0)
	ix := make([]uint16, 0)
	for i := 0; i < 6; i++ {
		ox, oy := float32(i*64)+32, float32(i*32)+(64*float32(h.Depth))
		r, g, b, _ := Palettes["rainbow"].Float32(Paint(i))
		oldL := uint16(len(vs))
		vs = append(vs, hexVerticesByDepth[h.Depth]...)
		tri := vs[oldL : oldL+3]
		for j := 0; j < 3; j++ {
			tri[j].ColorR, tri[j].ColorG, tri[j].ColorB = r, g, b
			tri[j].DstX = tri[j].DstX*24 + ox
			tri[j].DstY = tri[j].DstY*24 + oy
		}
		ix = append(ix, oldL, oldL+1, oldL+2)
	}
	// fmt.Printf("vs[0:3]: %#v\n", vs[0:3])

	screen.DrawTriangles(vs, ix, hexTexture, op)

}
