package main

import "github.com/hajimehoshi/ebiten"

type Spiral struct {
	pl     *PolyLine
	sprite *Sprite
}

func NewSpiral() *Spiral {
	s := &Spiral{pl: NewPolyLine(Sprites["white"], Palettes["rainbow"])}
	return s
}

func (s *Spiral) Draw(target *ebiten.Image) {
	s.pl.Draw(target, 1.0)
}
