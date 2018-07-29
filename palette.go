package main

import (
	"fmt"
	"image/color"
	"github.com/hajimehoshi/ebiten"
)

type Paint struct {
	palette *Palette
	paint int
}

type Palette struct {
	raw []color.RGBA
	Paints []Paint
	Colors []ebiten.ColorM
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
	p.Paints = make([]Paint, 0, len(p.raw))
	p.Colors = make([]ebiten.ColorM, 0, len(p.raw))
	for idx, c := range p.raw {
		cm := ebiten.ColorM{}
		cm.Scale(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 1.0)
		p.Colors = append(p.Colors, cm)
		p.Paints = append(p.Paints, Paint{palette: p, paint: idx})
	}
}

func (p Palette) Paint(idx int) Paint {
	return p.Paints[idx % len(p.Paints)]
}

func (p Palette) Color(pt Paint) ebiten.ColorM {
	return p.Colors[pt.paint % len(p.Paints)]
}

func (p *Paint) Inc(n int) {
	p.paint = (p.paint + n) % len(p.palette.Paints)
}

func (p *Paint) Set(n int) {
	p.paint = n % len(p.palette.Paints)
}
