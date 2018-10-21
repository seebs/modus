package g

import (
	"image"

	"github.com/hajimehoshi/ebiten"
)

// A Sprite combines an image with a source rectangle, embedded in an
// ebiten.DrawImageOptions containing a SourceRect and a GeoM.
// The DrawImageOptions scales the image to a 1x1 image centered on {0,0}.
type Sprite struct {
	Image *ebiten.Image
	Op    ebiten.DrawImageOptions
	w, h  int
}

// Sprites represents the set of known sprites.
var Sprites = make(map[string]*Sprite)

// NewSprite creates a new sprite (unless one with that name already
// exists) using the given source rectangle and image.
func NewSprite(name string, img *ebiten.Image, sr image.Rectangle) *Sprite {
	if sp, ok := Sprites[name]; ok {
		return sp
	}
	sp := &Sprite{Image: img, Op: ebiten.DrawImageOptions{SourceRect: &sr}}
	sp.w = sr.Max.X - sr.Min.X
	sp.h = sr.Max.Y - sr.Min.Y
	// start out translated to center and scaled to 1x1
	sp.Op.GeoM.Scale(1/float64(sp.w), 1/float64(sp.h))
	sp.Op.GeoM.Translate(-0.5, -0.5)
	Sprites[name] = sp
	return sp
}
