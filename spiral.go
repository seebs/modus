package main

import (
	"math"

	"github.com/hajimehoshi/ebiten"
)

// A Spiral represents one or more PolyLines ranging from a center point to
// a target point.
type Spiral struct {
	Center, Target MovingPoint
	Theta          float64
	Depth          int
	Length         int
	Step           int // step 1 = draw every line, step 2 = draw every other line
	Palette        *Palette
	Ripples        []int
	pl             []*PolyLine
	sprite         *Sprite
}

// the ripple pattern is used to perturb the radius of a spiral to make it look
// like it's bouncing.
var ripplePattern = []int{-1, -2, 0, 2, 1, 0, -1, 0, 1}

// NewSpiral creates a new spiral.
func NewSpiral(depth int, points int, p *Palette, cycles int) *Spiral {
	s := &Spiral{Depth: depth, Length: points}
	s.Palette = p.Interpolate(s.Length / (p.Length * cycles))
	for i := 0; i < depth; i++ {
		l := NewPolyLine(s.Palette, 3)
		l.Thickness = 3
		l.Points = make([]LinePoint, s.Length)
		for j := 0; j < s.Length; j++ {
			l.Points[j].P = s.Palette.Paint(j)
		}
		s.pl = append(s.pl, l)
	}
	return s
}

// Draw draws the spiral on the specified image.
func (s *Spiral) Draw(target *ebiten.Image) {
	for i := 0; i < s.Depth; i += s.Step {
		s.pl[i].Draw(target, (float64(i)+1)/float64(s.Depth))
		// s.pl[i].Draw(target, 1.0)
	}
}

func (s *Spiral) Compute(pl *PolyLine) {
	dx, dy := s.Target.Loc.X-s.Center.Loc.X, s.Target.Loc.Y-s.Center.Loc.Y
	baseTheta := math.Atan2(dy, dx)
	baseR := math.Sqrt(dx*dx + dy*dy)
	scaleR := s.Theta
	ripples := make([]int, s.Length)
	drop := 0
	for idx, rip := range s.Ripples {
		for i, p := range ripplePattern {
			if i+rip < s.Length && i+rip >= 0 {
				ripples[rip+i] += p
			}
		}
		s.Ripples[idx] -= 2
		if s.Ripples[idx] < 0 {
			drop = idx + 1
		}
	}
	s.Ripples = s.Ripples[drop:]
	// degenerate cases
	pt := pl.Point(0)
	pt.X, pt.Y = s.Center.Loc.X, s.Center.Loc.Y
	pt = pl.Point(s.Length - 1)
	pt.X, pt.Y = s.Target.Loc.X, s.Target.Loc.Y
	for i := 1; i < s.Length-1; i++ {
		pt := pl.Point(i)
		sin, cos := math.Sincos(float64(i)/float64(s.Length-1)*scaleR + baseTheta)
		r := float64(i) / float64(s.Length-1) * baseR
		if ripples[i] != 0 {
			r *= 1 + (0.03 * float64(ripples[i]))
		}
		x, y := (cos*r)+s.Center.Loc.X, (sin*r)+s.Center.Loc.Y
		pt.X, pt.Y = x, y
	}

}

// Update moves the target according to its velocity, possibly adding a ripple.
func (s *Spiral) Update() {
	if s.Target.Update() {
		s.Ripples = append(s.Ripples, s.Length)
	}
	line := s.pl[0]
	for i := 0; i < s.Depth-1; i++ {
		s.pl[i] = s.pl[i+1]
	}
	s.pl[s.Depth-1] = line
	s.Compute(line)
}
