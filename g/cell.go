package g

import "math"

// Cell represents a single cell, which has a color
// understood in terms of the parent grid's palette, also
// values for alpha, theta, and scale. Scale of 1 represents
// cells which are directly touching.
type Cell struct {
	P     Paint
	Alpha float32
	Theta float32
	Scale float32
}

func (c *Cell) IncTheta(t float32) {
	t += c.Theta
	if t < 0 {
		x := math.Ceil(math.Abs(float64(t)) / (math.Pi * 2))
		t += float32(math.Pi * 2 * x)
	}
	if t > (math.Pi * 2) {
		x := math.Floor(float64(t) / (math.Pi * 2))
		t -= float32(math.Pi * 2 * x)
	}
	c.Theta = t
}

func (c *Cell) IncAlpha(a float32) {
	a += c.Alpha
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	c.Alpha = a
}
