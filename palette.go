package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten"
)

// A Paint represents a selection-of-color from a Palette.
type Paint int

// A Palette represents a collection of indexed colors.
type Palette struct {
	RGBA    []color.RGBA
	ColorMs []ebiten.ColorM
	F32     [][3]float32
	Length  int
}

// Palettes is, perhaps surprisingly, the set of known Palettes.
var Palettes = map[string]*Palette{
	"rainbow": {
		RGBA: []color.RGBA{
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

// Initialize converts a palettes RGBA colors to ebiten.ColorM objects.
func (p *Palette) Initialize() {
	p.Length = len(p.RGBA)
	p.ColorMs = make([]ebiten.ColorM, 0, p.Length)
	p.F32 = make([][3]float32, 0, p.Length)
	for _, c := range p.RGBA {
		r, g, b := float64(c.R) / 255, float64(c.G)/255, float64(c.B)/255
		cm := ebiten.ColorM{}
		cm.Scale(r, g, b, 1.0)
		p.ColorMs = append(p.ColorMs, cm)
		p.F32 = append(p.F32, [3]float32{float32(r), float32(g), float32(b)})
	}
}

func interpolate(into []color.RGBA, from, to color.RGBA) {
	r0, g0, b0, a0 := int(from.R), int(from.G), int(from.B), int(from.A)
	r1, g1, b1, a1 := int(to.R), int(to.G), int(to.B), int(to.A)
	n := len(into)

	for i := 0; i < n; i++ {
		inv := n - i
		r := (r0 * inv + r1 * i) / n
		g := (g0 * inv + g1 * i) / n
		b := (b0 * inv + b1 * i) / n
		a := (a0 * inv + a1 * i) / n
		
		into[i] = color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
		// fmt.Printf("%d/%d from #%02x%02x%02x to #%02x%02x%02x: #%02x%02x%02x\n", i, n, r0, g0, b0, r1, g1, b1, r, g, b)
	}
}

func (p *Palette) Interpolate(n int) (*Palette) {
	np := &Palette{Length: p.Length * n, RGBA: make([]color.RGBA, p.Length * n)}

	prev := p.RGBA[0]
	for idx, next := range p.RGBA[1:] {
		offset := idx * n
		interpolate(np.RGBA[offset:offset+n], prev, next)
		prev = next
	}
	interpolate(np.RGBA[(p.Length - 1) * n:], prev, p.RGBA[0])
	np.Initialize()
	return np
}

// Paint yields the idx'th Paint in a given Palette, coercing into range.
func (p Palette) Paint(idx int) Paint {
	return Paint(idx % p.Length)
}

// Color yields the corresponding ebiten.ColorM, coercing into range.
func (p Palette) ColorM(pt Paint) ebiten.ColorM {
	return p.ColorMs[int(pt)%p.Length]
}

// Float32 yields RGBA float32 values
func (p Palette) Float32(pt Paint) (float32, float32, float32, float32) {
	f := p.F32[int(pt)%p.Length]
	return f[0], f[1], f[2], 1.0
}

// Inc yields the nth Paint after the given Paint. n may be negative,
// but not negative and greater in magnitude than the length of the palette.
func (p Palette) Inc(pt Paint, n int) Paint {
	return Paint((int(pt) + n) % p.Length)
}
