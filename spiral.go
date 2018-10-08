package main

import (
	"math"

	"github.com/hajimehoshi/ebiten"
)

// A Spiral represents one or more PolyLines ranging from a center point to
// a target point.
type Spiral struct {
	Center, Target Point
	Theta          float64
	Depth          int
	pl             []*PolyLine
	sprite         *Sprite
}

// NewSpiral creates a new spiral.
func NewSpiral(depth int, points int, p *Palette) *Spiral {
	s := &Spiral{Depth: depth}
	for i := 0; i < depth; i++ {
		l := NewPolyLine(p)
		l.Points = make([]LinePoint, points)
		for j := 0; j < points; j++ {
			l.Points[j].P = p.Paint(j)
		}
		s.pl = append(s.pl, l)
	}
	return s
}

// Draw draws the spiral on the specified image.
func (s *Spiral) Draw(target *ebiten.Image) {
	for i := 0; i < s.Depth; i++ {
		s.pl[i].Draw(target, (float64(i)+1)/float64(s.Depth))
	}
}

func (s *Spiral) spiralTo(pl *PolyLine, tx, ty float64) {
	dx, dy := tx-s.Center.X, ty-s.Center.Y
	baseTheta := math.Atan2(dy, dx)
	baseR := math.Sqrt(dx*dx + dy*dy)
	scaleR := s.Theta
	l := len(pl.Points)
	for i := 0; i < l; i++ {
		pt := pl.Point(i)
		sin, cos := math.Sincos(float64(i)/float64(l-1)*scaleR + baseTheta)
		r := float64(i) / float64(l-1) * baseR
		x, y := (cos*r)+s.Center.X, (sin*r)+s.Center.Y
		pt.X, pt.Y = x, y
	}
}

// UpdateTarget sets a new location for target x/y, and replaces the oldest
// line with a new spiral pointing to that location.
func (s *Spiral) UpdateTarget(x, y float64) {
	line := s.pl[0]
	for i := 0; i < s.Depth-1; i++ {
		s.pl[i] = s.pl[i+1]
	}
	s.pl[s.Depth-1] = line
	s.spiralTo(line, x, y)
}
