package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten"
)

// A PolyLine represents a series of line segments, each
// with a distinct start/end color. In the absence of per-vertex
// colors, we use the end color for each line segment.
//
// A PolyLine is intended to be rendered onto a given display
// by scaling/rotating quads, because things like ebiten (or
// Corona) have limited line-drawing capabilities, so abstracting
// that away and presenting a polyline interface is more
// convenient.
type PolyLine struct {
	Points    []LinePoint
	image     *ebiten.Image
	sr        *image.Rectangle
	Thickness float64
	Palette   *Palette
	sx, sy    float64
	w, h      float64
	cw, ch    float64
}

type LinePoint struct {
	X, Y float64
	P    Paint
}

func NewPolyLine(source *ebiten.Image, p *Palette) *PolyLine {
	pl := &PolyLine{image: source, Palette: p}
	if pl.image != nil {
		// TODO: Don't hard-code this, also figure out how to specify
		// the sourcerect cleanly.
		pl.w, pl.h = 32, 32
		pl.cw, pl.ch = pl.w/2, pl.h/2
		pl.sx, pl.sy = 1/pl.w, 1/pl.h
		pl.sr = &image.Rectangle{
			Min: image.Point{X: 32, Y: 0},
			Max: image.Point{X: 64, Y: 32},
		}
	}
	return pl
}

// Draw renders the line on the target, using drawimage options modified by
// color and location of line segments.
func (pl PolyLine) Draw(target *ebiten.Image, alpha float64) {
	// can't draw without an image
	if pl.image == nil {
		return
	}
	thickness := pl.Thickness
	// no invisible lines plz
	if thickness == 0 {
		thickness = 0.7
	}
	prev := pl.Points[0]
	count := 0
	op := ebiten.DrawImageOptions{}
	op.SourceRect = pl.sr
	op.Filter = ebiten.FilterLinear
	for _, next := range pl.Points[1:] {
		var g ebiten.GeoM
		cx, cy := (prev.X+next.X)/2, (prev.Y+next.Y)/2
		dx, dy := (next.X - prev.X), (next.Y - prev.Y)
		l := math.Sqrt(dx*dx + dy*dy)
		theta := math.Atan2(dy, dx)
		g.Translate(-pl.cw, -pl.ch)
		g2 := g
		g.Scale(l*pl.sx, thickness*pl.sy)
		g2.Scale(l*pl.sx, (thickness+0.5)*pl.sy)
		g.Rotate(theta)
		g2.Rotate(theta)
		g.Translate(cx, cy)
		g2.Translate(cx, cy)
		op.ColorM = pl.Palette.Color(next.P)
		op.ColorM.Scale(1, 1, 1, 0.5 * alpha)
		op.GeoM = g
		target.DrawImage(pl.image, &op)
		op.GeoM = g2
		target.DrawImage(pl.image, &op)
		prev = next
		count++
	}
}

func (pl PolyLine) Length() int {
	return len(pl.Points)
}

func (pl PolyLine) Point(i int) *LinePoint {
	if i < 0 || i >= len(pl.Points) {
		return nil
	}
	return &pl.Points[i]
}

func (pl *PolyLine) Add(x, y float64, p Paint) {
	pt := LinePoint{X: x, Y: y, P: p}
	pl.Points = append(pl.Points, pt)
}
