package g

import (
	"fmt"

	"github.com/hajimehoshi/ebiten"
)

// Weave represents a vector-graphics display, which is implemented on top
// of polylines. A Weave display has objects, called Knots, which are represented
// as a series of line segments. Each Knot is assumed to have its own local
// coordinate space; it has a position, size, and rotation. Within the Knot,
// you use coordinates in the -1..+1 space to describe where line segments
// go; these are then scaled such that a line from [-1,-1] to [+1,+1] would
// be a diagonal across a square with sides equal to the knot's scale, centered
// around its position in screen space. The Weave's space as a whole is
// similarly scaled for [-1,-1] to [1,1], so a Knot with a size of "0.25"
// would be about 1/4 the width-or-height of the screen. (Actual screen space
// is usually larger in one dimension because of aspect ratios.)
//
// The first point in a Weave should always have Skip set.
//
//

type Weave struct {
	pl    *PolyLine
	knots []*Knot
	dirty bool
}

type Knot struct {
	X, Y      float32
	Size      float32
	Theta     float32
	Alpha     float32
	Points    []LinePoint
	offset    int // offset of this knot's points in the weave's polyline's points.
	length    int
	weave     *Weave
	dirty     bool
	rawPoints []LinePoint
}

func newWeave(thickness int, p *Palette, scale, offsetX, offsetY float32) *Weave {
	w := &Weave{pl: newPolyLine(thickness, 3, p, scale, offsetX, offsetY)}
	w.pl.Joined = true
	w.pl.SetGlow(true)
	return w
}

func (w *Weave) Dirty() {
	w.dirty = true
}

func (w *Weave) Draw(target *ebiten.Image, alpha float32, scale float32) {
	if w.dirty {
		count := 0
		for _, k := range w.knots {
			if k.dirty {
				count++
				k.apply()
				w.pl.Dirty()
			}
		}
		w.SetStatus(fmt.Sprintf("applied %d", count))
		w.dirty = false
	}
	w.pl.Draw(target, alpha, scale)
}

// updateSlices points the slices for the knots at the right segments of the
// line after a copy-or-move operation.
func (w *Weave) updateSlices() {
	offset := 0
	for i := range w.knots {
		w.knots[i].rawPoints = w.pl.Points[offset : offset+w.knots[i].length]
		offset += w.knots[i].length
	}
}

// NewKnot allocates a new knot in the given weave. It adds points to the weave's
// line, and returns a knot with those points in its slice.
func (w *Weave) NewKnot(points int) *Knot {
	k := &Knot{length: points, weave: w, Points: make([]LinePoint, points)}
	k.Points[0].Skip = true
	if len(w.knots) == 0 {
		k.offset = 0
	} else {
		lastK := w.knots[len(w.knots)-1]
		k.offset = lastK.offset + lastK.length
	}
	for w.pl.Length() < k.offset+points {
		w.pl.Add(0, 0, 0)

	}
	w.knots = append(w.knots, k)
	w.updateSlices()
	return k
}

func (w *Weave) SetStatus(status string) {
	w.pl.SetStatus(status)
}

// Delete deletes the knot from the parent weave.
func (k *Knot) Delete() {
	if k.weave == nil {
		return
	}
	w := k.weave
	offset := 0
	for i := range w.knots {
		if w.knots[i] == k {
			// move all the lines up the polyline
			copy(w.pl.Points[offset:], w.pl.Points[offset+w.knots[i].length:])
			// move all the knots down a slot
			copy(w.knots[i:], w.knots[i+1:])
			// and lower their offsets
			for _, k2 := range w.knots[i:] {
				k2.offset = offset
				offset += k2.length
			}
			break
		}
		offset += w.knots[i].length
	}
	// Trim the polyline to the points we still care about.
	offset = 0
	if len(w.knots) > 0 {
		lastK := w.knots[len(w.knots)-1]
		offset = lastK.offset + lastK.length
	}
	w.pl.Points = w.pl.Points[:offset]
	w.updateSlices()
	// no longer associated with that weave
	k.weave = nil
}

// Dirty indicates that the values in Points have been changed, and the
// corresponding points in the parent line should be updated.
func (k *Knot) Dirty() {
	k.dirty = true
	if k.weave != nil {
		k.weave.Dirty()
	}
}

// apply copies the user-visible points into the underlying points, translating
// as necessary.
func (k *Knot) apply() {
	if !k.dirty {
		return
	}
	aff := IdentityAffine()
	aff.Scale(k.Size/2, k.Size/2)
	aff.Rotate(k.Theta)
	aff.Translate(k.X, k.Y)
	for i := range k.Points {
		k.rawPoints[i] = k.Points[i]
		k.rawPoints[i].X, k.rawPoints[i].Y = aff.Project(k.Points[i].X, k.Points[i].Y)
	}
	// we never draw a line to the starting point of the thing.
	k.rawPoints[0].Skip = true
	k.dirty = false
}

func (k *Knot) Affine() Affine {
	aff := IdentityAffine()
	aff.Scale(k.Size/2, k.Size/2)
	aff.Rotate(k.Theta)
	aff.Translate(k.X, k.Y)
	return aff
}
