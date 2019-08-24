package g

import (
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/text"
)

type Text struct {
	fontName                string
	face                    font.Face
	P                       Paint
	size                    int
	palette                 *Palette
	Text                    string
	X, Y                    float32
	scale, offsetX, offsetY float32
}

func newText(fontName string, size int, palette *Palette, scale, offsetX, offsetY float32) (*Text, error) {
	f, err := truetype.Parse(fonts.ArcadeN_ttf)
	if err != nil {
		return nil, err
	}
	t := &Text{fontName: fontName, size: size, palette: palette, scale: scale, offsetX: offsetX, offsetY: offsetY}
	t.face = truetype.NewFace(f, &truetype.Options{
		Size:    float64(t.size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	return t, nil
}

func (t *Text) Draw(target *ebiten.Image, alpha float32, scale float32) {
	if target != nil {
		text.Draw(target, t.Text, t.face, int(t.X*t.scale+t.offsetX), int(t.Y*t.scale+t.offsetY), t.palette.Color(t.P))
	}
}
