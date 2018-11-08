package g

import (
	"image"
	"math"
	"math/rand"
)

// A Point represents a location on-screen.
type Point struct {
	X, Y float64
}

// A MovingPoint is a point which has a velocity, and can bounce off the
// edges of a screen or something.
type MovingPoint struct {
	Loc      Point
	Velocity Point
	Bounds   image.Rectangle
}

// moveCoordinate moves x by dx, returning new x, new dx, and whether or
// not a bounce happened.
func moveCoordinate(x, dx float64, min, max int) (float64, float64, bool) {
	bounce := false
	x += dx
	var dabs, dsign, base float64
	if x < float64(min) {
		dabs = float64(min) - x
		dsign = -1
		base = float64(min)
		bounce = true
	}
	if x > float64(max) {
		dabs = x - float64(max)
		dsign = 1
		base = float64(max)
		bounce = true
	}
	if bounce {
		scale := float64(max - min)
		// if moving too fast, slow down
		if dabs > scale/2 {
			dabs = scale / 2
			dx /= 2
		}
		x = (dabs * dsign) + base
		dx *= -1
	}
	return x, dx, bounce
}

// SetBounds sets the bounds of a point to range from {0, 0}
// to {x, y}.
func (m *MovingPoint) SetBounds(x, y int) {
	m.Bounds.Min = image.Point{X: 0, Y: 0}
	m.Bounds.Max = image.Point{X: x, Y: y}
}

// PerturbVelocity randomly increments or decrements the velocity
// components.
func (m *MovingPoint) PerturbVelocity() {
	switch rand.Intn(3) {
	case 0:
		m.Velocity.X++
	case 1:
		m.Velocity.X--
	}
	switch rand.Intn(3) {
	case 0:
		m.Velocity.Y++
	case 1:
		m.Velocity.Y--
	}
}

func (m *MovingPoint) Update() bool {
	var bounceX, bounceY bool
	m.Loc.X, m.Velocity.X, bounceX = moveCoordinate(m.Loc.X, m.Velocity.X, m.Bounds.Min.X, m.Bounds.Max.X)
	m.Loc.Y, m.Velocity.Y, bounceY = moveCoordinate(m.Loc.Y, m.Velocity.Y, m.Bounds.Min.Y, m.Bounds.Max.Y)
	return bounceX || bounceY
}

// Affine is a trivial affine matrix
// { a, c, e }
// { b, d, f }
type Affine struct {
	A, B, C, D, E, F float32
}

// Project applies the affine matrix.
func (a Affine) Project(x0, y0 float32) (x1, y1 float32) {
	return a.A*x0 + a.C*y0 + a.E, a.B*x0 + a.D*y0 + a.F
}

// Unproject reverses projection.
func (a Affine) Unproject(x1, y1 float32) (x0, y0 float32) {
	// subtract translation, multiply by inverse of upper left 2x2
	d := (a.A * a.D) - (a.B * a.C)
	x1, y1 = (x1-a.E)/d, (y1-a.F)/d
	return x1*a.D - y1*a.B, y1*a.A - x1*a.C
}

// Scale scales by X and Y.
func (a *Affine) Scale(xs, ys float32) {
	a.A, a.C, a.E = a.A*xs, a.C*xs, a.E*xs
	a.B, a.D, a.F = a.B*ys, a.D*ys, a.F*ys
}

// Rotate rotates by an angle.
func (a *Affine) Rotate(theta float32) {
	s64, c64 := math.Sincos(float64(theta))
	s, c := float32(s64), float32(c64)
	a.A, a.B, a.C, a.D, a.E, a.F = a.A*c+a.C*s, a.B*c+a.D*s, a.C*c-a.A*s, a.D*c-a.B*s, a.E, a.F
}

// IdentityAffine yields the identity matrix.
func IdentityAffine() Affine {
	return Affine{A: 1, D: 1}
}

// finds next power of 2, but only if n < 2^31, because
// this is for texture sizes
func npo2(n int) int {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}
