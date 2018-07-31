package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten"
)

type Paint int

type Palette struct {
	raw    []color.RGBA
	Colors []ebiten.ColorM
	Length int
}

var Palettes = map[string]*Palette{
	"rainbow": {
		raw: []color.RGBA{
			{255, 0, 0, 255},
			color.RGBA{240, 90, 0, 255},
			color.RGBA{220, 220, 0, 255},
			color.RGBA{0, 200, 0, 255},
			color.RGBA{0, 0, 255, 255},
			color.RGBA{180, 0, 200, 255},
		},
	},
}

func init() {
	for _, p := range Palettes {
		p.Initialize()
	}
}

func (p *Palette) Initialize() {
	p.Length = len(p.raw)
	p.Colors = make([]ebiten.ColorM, 0, p.Length)
	for _, c := range p.raw {
		cm := ebiten.ColorM{}
		cm.Scale(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 1.0)
		p.Colors = append(p.Colors, cm)
	}
}

func (p Palette) Paint(idx int) Paint {
	return Paint(idx % p.Length)
}

func (p Palette) Color(pt Paint) ebiten.ColorM {
	return p.Colors[int(pt)%p.Length]
}

func (p Palette) Inc(pt Paint, n int) Paint {
	return Paint((int(pt) + n) % p.Length)
}
