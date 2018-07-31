package main

import (
	"image"

	"github.com/hajimehoshi/ebiten"
)

type Sprite struct {
	Image *ebiten.Image
	Op    ebiten.DrawImageOptions
	w, h  int
}

var Sprites map[string]*Sprite = make(map[string]*Sprite)

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
