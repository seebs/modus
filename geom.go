package main

import "image"

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

func (m *MovingPoint) SetBounds(x, y int) {
	m.Bounds.Min = image.Point{X: 0, Y: 0}
	m.Bounds.Max = image.Point{X: x, Y: y}
}

func (m *MovingPoint) Update() bool {
	var bounceX, bounceY bool
	m.Loc.X, m.Velocity.X, bounceX = moveCoordinate(m.Loc.X, m.Velocity.X, m.Bounds.Min.X, m.Bounds.Max.X)
	m.Loc.Y, m.Velocity.Y, bounceY = moveCoordinate(m.Loc.Y, m.Velocity.Y, m.Bounds.Min.Y, m.Bounds.Max.Y)
	return bounceX || bounceY
}
