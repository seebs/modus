// Package g provides either Graphics or Geometry depending on my mood.
// It's intended to provide the infrastructure necessary for the
// Miracle Modus, using ebiten for the rendering backend, but also
// providing useful data structures and an API for working with them.
package g

import (
	"image"
	"math"
	"math/rand"
)

// General geometry functions.

// A MovingPoint is a point which has a velocity, and can bounce off the
// edges of a screen or something.
type MovingPoint struct {
	Loc      Point
	Velocity Vec
	Bounds   image.Rectangle
}

// IVec represents a vector of motion within a grid. (Contrast time.Duration.)
type IVec struct {
	X, Y int
}

func (v IVec) Times(n int) IVec {
	return IVec{X: v.X * n, Y: v.Y * n}
}

// ILoc represents a location within a grid. (Contrast time.Time.)
type ILoc struct {
	X, Y int
}

// moveCoordinate moves x by dx, returning new x, new dx, and whether or
// not a bounce happened.
func moveCoordinate(x, dx float32, min, max int) (float32, float32, bool) {
	bounce := false
	x += dx
	var dabs, dsign, base float32
	if x < float32(min) {
		dabs = float32(min) - x
		dsign = -1
		base = float32(min)
		bounce = true
	}
	if x > float32(max) {
		dabs = x - float32(max)
		dsign = 1
		base = float32(max)
		bounce = true
	}
	if bounce {
		scale := float32(max - min)
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

type Vec struct {
	X, Y float32
}

type Point struct {
	X, Y float32
}

func (v Vec) Project(a *Affine) Vec {
	return Vec{X: a.A*v.X + a.C*v.Y, Y: a.B*v.X + a.D*v.Y}
}

func (p Point) Project(a *Affine) Point {
	return Point{X: a.A*p.X + a.C*p.Y + a.E, Y: a.B*p.X + a.D*p.Y + a.F}
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
func (a *Affine) Scale(xs, ys float32) *Affine {
	a.A, a.C, a.E = a.A*xs, a.C*xs, a.E*xs
	a.B, a.D, a.F = a.B*ys, a.D*ys, a.F*ys
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
