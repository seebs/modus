package g

import "math"

// Cell represents a single cell, which has a color
// understood in terms of the parent grid's palette, also
// values for alpha, theta, and scale. Scale of 1 represents
// cells which are directly touching.
type Cell struct {
	P     Paint
	R     RenderType
	Alpha float32
	Theta float32
	Scale float32
}

type FloatingCell interface {
	C() *Cell
	Loc() *FLoc
	X() *float32
	Y() *float32
	Z() *float32
}

// FloatingCell represents a single floating cell, which may be rendered
// in a non-integer location over a grid.
type FloatingCellBase struct {
	Cell
	loc FLoc
}

func (f *FloatingCellBase) C() *Cell {
	return &f.Cell
}

func (f *FloatingCellBase) Loc() *FLoc {
	return &f.loc
}

func (f *FloatingCellBase) X() *float32 {
	return &f.loc.X
}

func (f *FloatingCellBase) Y() *float32 {
	return &f.loc.Y
}

func (f *FloatingCellBase) Z() *float32 {
	return nil
}

// IncTheta rotates the given cell by t.
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

// IncAlpha increments the given cell's alpha by a, clamping to 0/1.
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
