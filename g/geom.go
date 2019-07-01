// Package g provides either Graphics or Geometry depending on my mood.
// It's intended to provide the infrastructure necessary for the
// Miracle Modus, using ebiten for the rendering backend, but also
// providing useful data structures and an API for working with them.
package g

import (
	"fmt"
	"math"
	"math/rand"
)

// General geometry functions.

// A MovingPoint is a point which has a velocity, and can bounce off the
// edges of a screen or something.
type MovingPoint struct {
	Loc      Point
	Velocity Vec
	Bounds   Region
}

// A Region represents a rectangle with diagonal between two points.
type Region struct {
	Min, Max Point
}

// IVec represents a vector of motion within a grid. (Contrast time.Duration.)
type IVec struct {
	X, Y int
}

// Times multiplies a vector by a scalar.
func (v IVec) Times(n int) IVec {
	return IVec{X: v.X * n, Y: v.Y * n}
}

// ILoc represents a location within a grid. (Contrast time.Time.)
type ILoc struct {
	X, Y int
}

func (i ILoc) FLoc() FLoc {
	return FLoc{X: float32(i.X), Y: float32(i.Y)}
}

// FLoc is a float version of ILoc, used for things that aren't precisely
// on the grid.
type FLoc struct {
	X, Y float32
}

// FVec is a float version of IVec, used for things that aren't precisely
// on the grid.
type FVec struct {
	X, Y float32
}

// moveCoordinate moves x by dx, returning new x, new dx, and whether or
// not a bounce happened.
func moveCoordinate(x, dx float32, min, max float32) (float32, float32, bool) {
	bounce := false
	x += dx
	var dabs, dsign, base float32
	if x < min {
		dabs = min - x
		dsign = -1
		base = min
		bounce = true
	}
	if x > max {
		dabs = x - max
		dsign = 1
		base = max
		bounce = true
	}
	if bounce {
		scale := max - min
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

// SetBounds sets the bounds of a point to range from min to max.
func (m *MovingPoint) SetBounds(min, max Point) {
	m.Bounds.Min = min
	m.Bounds.Max = max
}

// PerturbVelocity randomly increments or decrements the velocity
// components.
func (m *MovingPoint) PerturbVelocity() {
	switch rand.Intn(3) {
	case 0:
		m.Velocity.X += 0.001
	case 1:
		m.Velocity.X -= 0.001
	}
	switch rand.Intn(3) {
	case 0:
		m.Velocity.Y += 0.001
	case 1:
		m.Velocity.Y -= 0.001
	}
}

func (m *MovingPoint) Update() bool {
	var bounceX, bounceY bool
	m.Loc.X, m.Velocity.X, bounceX = moveCoordinate(m.Loc.X, m.Velocity.X, m.Bounds.Min.X, m.Bounds.Max.X)
	m.Loc.Y, m.Velocity.Y, bounceY = moveCoordinate(m.Loc.Y, m.Velocity.Y, m.Bounds.Min.Y, m.Bounds.Max.Y)
	return bounceX || bounceY
}

func (m MovingPoint) String() string {
	return fmt.Sprintf("@%g,%g +%g,%g, >%g,%g <%g,%g",
		m.Loc.X, m.Loc.Y,
		m.Velocity.X, m.Velocity.Y,
		m.Bounds.Min.X, m.Bounds.Min.Y,
		m.Bounds.Max.X, m.Bounds.Max.Y)
}

// Affine is a trivial affine matrix
// { a, c, e }
// { b, d, f }
type Affine struct {
	A, B, C, D, E, F float32
}

// Vec represents motion (contrast time.Duration).
type Vec struct {
	X, Y float32
}

// Point represents a location (contrast time.Time).
type Point struct {
	X, Y float32
}

// Project projects a given vector through an affine matrix. Because
// vectors represent motion, not position, the translation of the matrix
// is ignored.
func (v Vec) Project(a *Affine) Vec {
	return Vec{X: a.A*v.X + a.C*v.Y, Y: a.B*v.X + a.D*v.Y}
}

// Project projects a given point through an affine matrix.
func (p Point) Project(a *Affine) Point {
	return Point{X: a.A*p.X + a.C*p.Y + a.E, Y: a.B*p.X + a.D*p.Y + a.F}
}

// Project applies the affine matrix, including translation.
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
func (a *Affine) Scale(x, y float32) *Affine {
	a.A, a.C, a.E = a.A*x, a.C*x, a.E*x
	a.B, a.D, a.F = a.B*y, a.D*y, a.F*y
	return a
}

func (a *Affine) String() string {
	return fmt.Sprintf("a: %g, c: %g, e: %g\nb: %g, d: %g, f: %g\n",
		a.A, a.C, a.E, a.B, a.D, a.F)
}

// Translate translates by X and Y
func (a *Affine) Translate(x, y float32) *Affine {
	a.E = a.E + x
	a.F = a.F + y
	return a
}

// Rotate rotates by an angle.
func (a *Affine) Rotate(theta float32) *Affine {
	s64, c64 := math.Sincos(float64(theta))
	s, c := float32(s64), float32(c64)
	a.A, a.B, a.C, a.D = a.A*c+a.C*s, a.B*c+a.D*s, a.C*c-a.A*s, a.D*c-a.B*s
	return a
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
