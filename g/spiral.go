package g

import (
	math "github.com/chewxy/math32"

	"github.com/hajimehoshi/ebiten"
)

// A Spiral represents one or more PolyLines ranging from a center point to
// a target point.
type Spiral struct {
	Center, Target MovingPoint
	Theta          float32
	Depth          int
	render         RenderType
	Length         int
	Step           int // step 1 = draw every line, step 2 = draw every other line
	Palette        *Palette
	Ripples        []int
	pl             []*PolyLine
	thetas         []float32
	sprite         *Sprite
	scaleTheta     float32
	thetaRatio     float32
}

// the ripple pattern is used to perturb the radius of a spiral to make it look
// like it's bouncing.
var ripplePattern = []int{-1, -2, 0, 2, 1, 0, -1, 0, 1}
var defaultThetaRatio = float32(4.0)

// newSpiral creates a new spiral.
func newSpiral(depth int, r RenderType, points int, p *Palette, cycles int, offset int, scale, offsetX, offsetY float32) *Spiral {
	s := &Spiral{Depth: depth, render: r, Length: points, Step: 1}
	// we want to make it through the palette cycles times; for instance,
	// if cycles is 3, we want a total of 18 color shifts, divided among
	// s.Length segments, so that's the interpolation scale.
	paletteScale := s.Length / (p.Length * cycles)
	s.Palette = p.Interpolate(paletteScale)
	// an offset of 1 is "one color"
	offset *= paletteScale
	// scale theta: inner points get thetaRatio times as much theta as outer points
	s.thetas = make([]float32, s.Length)
	s.SetThetaRatio(defaultThetaRatio)
	for i := 0; i < s.Depth; i++ {
		l := newPolyLine(3, r, s.Palette, scale, offsetX, offsetY)
		l.Joined = true
		l.Blend = true
		l.Points = make([]LinePoint, s.Length)
		for j := 0; j < s.Length; j++ {
			l.Points[j].P = s.Palette.Paint(j + offset)
		}
		s.pl = append(s.pl, l)
	}
	return s
}

// SetThetaRatio recomputes theta values for a 1:N theta ratio,
// meaning that the innermost segment will be about N times as
// large an angle as the outermost.
func (s *Spiral) SetThetaRatio(ratio float32) {
	s.thetaRatio = ratio
	subscale := (s.thetaRatio - 1.0) / float32(s.Length)
	s.scaleTheta = 0
	for i := 1; i < s.Length; i++ {
		s.scaleTheta += (subscale * float32(s.Length-i)) + 1
		s.thetas[i] = s.scaleTheta
	}
}

// Draw draws the spiral on the specified image.
func (s *Spiral) Draw(target *ebiten.Image, scale float32) {
	for i := 0; i < s.Depth; i += s.Step {
		s.pl[i].Draw(target, (float32(i)+1)/float32(s.Depth), scale)
	}
}

func (s *Spiral) Compute(pl *PolyLine) {
	dx, dy := s.Target.Loc.X-s.Center.Loc.X, s.Target.Loc.Y-s.Center.Loc.Y
	baseTheta := math.Atan2(dy, dx)
	baseR := math.Sqrt(dx*dx + dy*dy)
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
	pt.P = s.Palette.Inc(pt.P, 1)

	pt = pl.Point(s.Length - 1)
	pt.X, pt.Y = s.Target.Loc.X, s.Target.Loc.Y
	pt.P = s.Palette.Inc(pt.P, 1)
	for i := 1; i < s.Length-1; i++ {
		pt := pl.Point(i)
		sin, cos := math.Sincos((s.thetas[i]/s.scaleTheta)*s.Theta + baseTheta)
		r := float32(i) / float32(s.Length-1) * baseR
		if ripples[i] != 0 {
			r *= 1 + (0.03 * float32(ripples[i]))
		}
		x, y := (cos*r)+s.Center.Loc.X, (sin*r)+s.Center.Loc.Y
		pt.X, pt.Y = x, y
		pt.P = s.Palette.Inc(pt.P, 1)
	}
	pl.Dirty()
}

// Update moves the target according to its velocity, possibly adding a ripple.
func (s *Spiral) Update() (bounced bool, note int, l Point) {
	if s.Target.Update() {
		s.Ripples = append(s.Ripples, s.Length)
		s.Target.PerturbVelocity()
		bounced = true
	}
	// emit note/color even if we won't play it
	note = (int(s.pl[0].Point(0).P) * 6) / (s.Palette.Length)
	line := s.pl[0]
	for i := 0; i < s.Depth-1; i++ {
		s.pl[i] = s.pl[i+1]
	}
	s.pl[s.Depth-1] = line
	s.Compute(line)
	return bounced, note, s.Target.Loc
}
