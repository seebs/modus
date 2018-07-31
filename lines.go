package main

import (
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
	sp        *Sprite
	Thickness float64
	Palette   *Palette
	sx, sy    float64
}

// A LinePoint is one point in a PolyLine, containing both
// a location and a Paint corresponding to the PolyLine's Palette.
type LinePoint struct {
	X, Y float64
	P    Paint
}

// NewPolyLine creates a new PolyLine using the specified sprite and palette.
func NewPolyLine(sp *Sprite, p *Palette) *PolyLine {
	pl := &PolyLine{sp: sp, Palette: p}
	return pl
}

// Draw renders the line on the target, using the sprite's drawimage
// options modified by color and location of line segments.
func (pl PolyLine) Draw(target *ebiten.Image, alpha float64) {
	// can't draw without an image
	if pl.sp == nil {
		return
	}
	thickness := pl.Thickness
	// no invisible lines plz
	if thickness == 0 {
		thickness = 0.7
	}
	prev := pl.Points[0]
	count := 0
	op := pl.sp.Op
	op.Filter = ebiten.FilterLinear
	baseG := op.GeoM
	for _, next := range pl.Points[1:] {
		cx, cy := (prev.X+next.X)/2, (prev.Y+next.Y)/2
		dx, dy := (next.X - prev.X), (next.Y - prev.Y)
		l := math.Sqrt(dx*dx + dy*dy)
		theta := math.Atan2(dy, dx)
		g := baseG
		g2 := baseG
		g.Scale(l, thickness)
		g2.Scale(l, (thickness + 0.5))
		g.Rotate(theta)
		g2.Rotate(theta)
		g.Translate(cx, cy)
		g2.Translate(cx, cy)
		op.ColorM = pl.Palette.Color(next.P)
		op.ColorM.Scale(1, 1, 1, 0.5*alpha)
		op.GeoM = g
		target.DrawImage(pl.sp.Image, &op)
		op.GeoM = g2
		target.DrawImage(pl.sp.Image, &op)
		prev = next
		count++
	}
}

// Length yields the number of points in the line.
func (pl PolyLine) Length() int {
	return len(pl.Points)
}

// Point yields a given point within the line.
func (pl PolyLine) Point(i int) *LinePoint {
	if i < 0 || i >= len(pl.Points) {
		return nil
	}
	return &pl.Points[i]
}

// Add adds a new point to the line.
func (pl *PolyLine) Add(x, y float64, p Paint) {
	pt := LinePoint{X: x, Y: y, P: p}
	pl.Points = append(pl.Points, pt)
}
