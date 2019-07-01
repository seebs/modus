package g

import (
	"fmt"
	"os"
	"sync"

	math "github.com/chewxy/math32"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

// A PolyLine represents a series of line segments, each
// with a distinct start/end color. In the absence of per-vertex
// colors, we use the end color for each line segment.
//
// A PolyLine is intended to be rendered onto a given display
// by scaling/rotating quads, because things like ebiten (or
// Corona) have limited line-drawing capabilities, so abstracting
// that away and presenting a polyline interface is more
// convenient.
type PolyLine struct {
	Points           []LinePoint
	Thickness        float32
	scale            float32
	offsetX, offsetY float32 // screen space conversion
	render           RenderType
	Palette          *Palette
	Blend            bool
	Joined           bool // one segment per point past the first, rather than each pair a segment
	DebugColor       bool // use debug colors
	debug            *PolyLine
	vertices         []ebiten.Vertex
	indices          []uint16
	dirty            bool
	glowing          bool
	status           string // debug status message if any
}

// A LinePoint is one point in a PolyLine, containing both
// a location and a Paint corresponding to the PolyLine's Palette.
type LinePoint struct {
	X, Y float32
	P    Paint
	Skip bool
	Glow bool // unimplemented
}

var (
	initLineData sync.Once
	debugColors  [][3]float32
)

func lineSetup() {
	textureSetup()
	debugColors = make([][3]float32, 6)
	rb := Palettes["rainbow"]
	for i := 0; i < 6; i++ {
		r, g, b, _ := rb.Float32(Paint(i))
		debugColors[i] = [3]float32{r, g, b}
	}
}

// NewPolyLine creates a new PolyLine using the specified sprite and palette.
func newPolyLine(thickness int, r RenderType, p *Palette, scale, offsetX, offsetY float32) *PolyLine {
	initLineData.Do(lineSetup)
	if r > 3 {
		r = 3
	}
	pl := &PolyLine{
		Palette:   p,
		render:    r,
		Blend:     true,
		Thickness: float32(thickness),
		scale:     scale,
		offsetX:   offsetX,
		offsetY:   offsetY,
	}
	return pl
}

func (pl *PolyLine) Debug(enable bool) {
	if enable {
		pl.debug = newPolyLine(2, 1, pl.Palette, pl.scale, pl.offsetX, pl.offsetY)
		pl.debug.Thickness = 2
	} else {
		pl.debug = nil
	}
}

func (pl *PolyLine) SetGlow(enable bool) {
	pl.glowing = enable
}

func (pl *PolyLine) SetStatus(status string) {
	pl.status = status
}

type LineBits struct {
	dx, dy float32 // delta x, delta y
	ux, uy float32 // unit x/y: x/y adjusted to a length of 1
	l      float32 // length
	nx, ny float32 // normal vector, normalized to unit length
	theta  float32 // angle, if applicable
}

func linebits(x0, y0, x1, y1 float32) (lb LineBits) {
	lb.dx, lb.dy = x1-x0, y1-y0
	lb.l = math.Sqrt(lb.dx*lb.dx + lb.dy*lb.dy)
	if lb.l == 0 {
		return lb
	}
	lb.ux, lb.uy = lb.dx/lb.l, lb.dy/lb.l
	lb.nx, lb.ny = lb.uy, -lb.ux
	lb.theta = math.Atan2(lb.dx, lb.dy)
	return lb
}

// The hard case: We compute six vertices per segment, and draw
// four triangles using them, plus we also have a bevel between
// line segments most of the time.
//
// P0 +----------------------------------------+ P2
//    |                                        |
//    |                                        |
// P4 +----------------------------------------+ P5
//    |                                        |
//    |                                        |
// P1 +----------------------------------------+ P3
//
// Triangles are 4-0-2, 2-5-4, 1-4-5, 5-3-1
//
// if the next segment bends right, we also have a triangle of P5-P2-nP0,
// and if it bends left, we also have one of P5-nP1-P3.
//
// for the first and last segment, the first or last set of vertexes
// is just the end of the line, +/- the normal to the line segment times
// the half-thickness. this is also used if the previous segment's length
// is zero, and for the "outer" side of a bend.
//
// For the inside of a bend, we want to compute a point which is somewhere
// inside the edge. For angles from "straight" to a right angle, we want
// the inside edge of the corner not to overlap. For acute angles, we want
// some overlap again. As the angle between two consecutive lines goes
// from 180 degrees down to 90, the relative angle of the inner bevel goes
// up to 45 degrees; as it goes from 90 back up, it goes back down to 0.
// The point should always still be on the line parallel to the line
// segment, and halfthick distance out from the line segment. So, for a
// precise right angle, the point is the original location it would have
// had, plus halfthick * the direction of the line segment.
//
//
var prevTheta float32

// please inline me
func adjust(v *ebiten.Vertex, ux, uy, scale float32) {
	v.DstX += ux * scale
	v.DstY += uy * scale
}

// Dirty marks that a line's been changed in a way it may not easily
// detect, such as modifying a point returned by pl.Point(). Otherwise
// it won't recompute its vertex buffer.
func (pl *PolyLine) Dirty() {
	pl.dirty = true
}

func (pl *PolyLine) vsPerSegment() int {
	switch {
	case pl.glowing && pl.Joined:
		return 12
	case pl.glowing && !pl.Joined:
		return 8
	case !pl.glowing && pl.Joined:
		return 6
	case !pl.glowing && !pl.Joined:
		return 4
	}
	return 0
}

func (pl *PolyLine) idxsPerSegment() int {
	switch {
	case pl.glowing && pl.Joined:
		return 30
	case pl.glowing && !pl.Joined:
		return 12
	case !pl.glowing && pl.Joined:
		return 15
	case !pl.glowing && !pl.Joined:
		return 6
	}
	return 0
}

func populateJoinedRGB(v []ebiten.Vertex, r0, g0, b0, r1, g1, b1, alpha float32) {
	if len(v) == 12 {
		alpha *= 0.75
		v[6].ColorR, v[6].ColorG, v[6].ColorB, v[6].ColorA = 1, 1, 1, alpha
		v[7].ColorR, v[7].ColorG, v[7].ColorB, v[7].ColorA = 1, 1, 1, alpha
		v[10].ColorR, v[10].ColorG, v[10].ColorB, v[10].ColorA = 1, 1, 1, alpha
		v[8].ColorR, v[8].ColorG, v[8].ColorB, v[8].ColorA = 1, 1, 1, alpha
		v[9].ColorR, v[9].ColorG, v[9].ColorB, v[9].ColorA = 1, 1, 1, alpha
		v[11].ColorR, v[11].ColorG, v[11].ColorB, v[11].ColorA = 1, 1, 1, alpha
	}
	v[0].ColorR, v[0].ColorG, v[0].ColorB, v[0].ColorA = r0, g0, b0, alpha
	v[1].ColorR, v[1].ColorG, v[1].ColorB, v[1].ColorA = r0, g0, b0, alpha
	v[4].ColorR, v[4].ColorG, v[4].ColorB, v[4].ColorA = r0, g0, b0, alpha
	v[2].ColorR, v[2].ColorG, v[2].ColorB, v[2].ColorA = r1, g1, b1, alpha
	v[3].ColorR, v[3].ColorG, v[3].ColorB, v[3].ColorA = r1, g1, b1, alpha
	v[5].ColorR, v[5].ColorG, v[5].ColorB, v[5].ColorA = r1, g1, b1, alpha
}

func populateJoinedVs(v []ebiten.Vertex, px, py, nx, ny float32, lb LineBits, halfthick, scale float32) {
	v[0].DstX = float32(px+lb.nx*halfthick) * scale
	v[0].DstY = float32(py+lb.ny*halfthick) * scale
	v[1].DstX = float32(px-lb.nx*halfthick) * scale
	v[1].DstY = float32(py-lb.ny*halfthick) * scale
	v[2].DstX = float32(nx+lb.nx*halfthick) * scale
	v[2].DstY = float32(ny+lb.ny*halfthick) * scale
	v[3].DstX = float32(nx-lb.nx*halfthick) * scale
	v[3].DstY = float32(ny-lb.ny*halfthick) * scale
	v[4].DstX, v[4].DstY = float32(px)*scale, float32(py)*scale
	v[5].DstX, v[5].DstY = float32(nx)*scale, float32(ny)*scale
	if len(v) == 12 {
		halfthick /= 4
		v[6].DstX = float32(px+lb.nx*halfthick) * scale
		v[6].DstY = float32(py+lb.ny*halfthick) * scale
		v[7].DstX = float32(px-lb.nx*halfthick) * scale
		v[7].DstY = float32(py-lb.ny*halfthick) * scale
		v[8].DstX = float32(nx+lb.nx*halfthick) * scale
		v[8].DstY = float32(ny+lb.ny*halfthick) * scale
		v[9].DstX = float32(nx-lb.nx*halfthick) * scale
		v[9].DstY = float32(ny-lb.ny*halfthick) * scale
		v[10].DstX, v[10].DstY = float32(px)*scale, float32(py)*scale
		v[11].DstX, v[11].DstY = float32(nx)*scale, float32(ny)*scale
	}
}

func (pl *PolyLine) computeJoinedVertices(halfthick, alpha, scale float32) (vertices, indices int) {
	segments := len(pl.Points) - 1
	if segments < 1 {
		// fail
		fmt.Fprintf(os.Stderr, "polyline of %d segments can't be drawn\n", segments)
		return
	}
	vsPerSegment := pl.vsPerSegment()
	// populate with the SrcX, SrcY values.
	if len(pl.vertices) < segments*vsPerSegment {
		fv := lineData.vsByR[pl.render]
		pl.vertices = make([]ebiten.Vertex, 0, segments*vsPerSegment)
		for i := 0; i < segments; i++ {
			pl.vertices = append(pl.vertices, fv...)
		}
		if pl.glowing {
			for i := 0; i < segments; i++ {
				pl.vertices = append(pl.vertices, fv...)
			}
		}
	}
	// indices can never change, conveniently!
	idxsPerSegment := pl.idxsPerSegment()
	if len(pl.indices) < segments*idxsPerSegment {
		for i := len(pl.indices) / 15; i < segments*idxsPerSegment; i++ {
			offset := uint16(i * 6)
			// Triangles are 4-0-2, 2-5-4, 1-4-5, 5-3-1
			// bezel is a special case: it has to be altered
			// later.
			pl.indices = append(pl.indices,
				offset+4, offset+0, offset+2,
				offset+2, offset+5, offset+4,
				offset+1, offset+4, offset+5,
				offset+5, offset+3, offset+1,
				offset+0, offset+0, offset+0)
		}
	}
	prev := pl.Points[0]
	r0, g0, b0, _ := pl.Palette.Float32(prev.P)
	count := 0

	if pl.debug != nil {
		pl.debug.Reset()
	}
	// Joined: We will draw one segment for each point past the first.
	var plb LineBits
	px, py := (prev.X*pl.scale)+pl.offsetX, (prev.Y*pl.scale)+pl.offsetY
	for idx, next := range pl.Points[1:] {
		nx, ny := (next.X*pl.scale)+pl.offsetX, (next.Y*pl.scale)+pl.offsetY
		if next.Skip {
			// update things so the next point is the new previous point
			prev = next
			r0, g0, b0, _ = pl.Palette.Float32(next.P)
			// we didn't compute the LineBits, but we want to act
			// as though this one had length zero
			plb.l = 0
			px, py = nx, ny
			// NOTE: This does not "fix up" a previous line's
			// end points, which would normally be done while
			// processing this line. That's probably correct
			// when this line isn't drawn.
			continue
		}
		// compute normal x/y values, scaled to unit length
		lb := linebits(prev.X, prev.Y, next.X, next.Y)
		if lb.l == 0 {
			// avoid division by zero
			// update things so the next point is the new previous point
			prev = next
			r0, g0, b0, _ = pl.Palette.Float32(next.P)
			plb = lb
			px, py = nx, ny
			// NOTE: This does not "fix up" a previous line's
			// end points, which would normally be done while
			// processing this line. That's probably correct
			// when this line isn't drawn.
			continue
		}
		var bezel, bezel2 []uint16
		bezel = pl.indices[count*idxsPerSegment+12 : count*idxsPerSegment+15]
		// make it a degenerate triangle so it gets ignored unless we use it later
		bezel[0], bezel[1], bezel[2] = 0, 0, 0
		if pl.glowing {
			bezel2 = pl.indices[count*idxsPerSegment+27 : count*idxsPerSegment+30]
			bezel2[0], bezel2[1], bezel2[2] = 0, 0, 0
		}
		r1, g1, b1, _ := pl.Palette.Float32(next.P)
		offset := uint16(count * vsPerSegment)
		v := pl.vertices[offset : offset+uint16(vsPerSegment)]
		// populate these with default values, which we'd use without the fancy algorithm
		populateJoinedVs(v, px, py, nx, ny, lb, halfthick, scale)

		if plb.l > 0 {
			// fix up the overlap between these lines
			theta := lb.theta
			if theta < plb.theta {
				theta += math.Pi * 2
			}
			dt := theta - plb.theta
			// are we turning "left"?
			// left = our P1, previous segment's P3
			// right = our P0, previous segment's P2
			left := false
			if dt > math.Pi {
				// our P1, previous segment's P3
				dt -= math.Pi
				left = true
			}

			sharp := math.Pi/2 - (math.Abs(dt - (math.Pi / 2)))
			scale := math.Tan(sharp / 2)
			if idx == 1 && dt != prevTheta {
				prevTheta = dt
			}
			// create bezel:
			if left {
				adjust(&v[1], lb.ux, lb.uy, scale*halfthick)
				adjust(&pl.vertices[offset-3], plb.ux, plb.uy, -scale*halfthick)
				if pl.glowing {
					adjust(&v[7], lb.ux, lb.uy, scale*halfthick/2)
					// undo half of the adjustment that was incorrect
					adjust(&pl.vertices[offset-3], plb.ux, plb.uy, +scale*halfthick/2)
					adjust(&pl.vertices[offset-9], plb.ux, plb.uy, -scale*halfthick)
					bezel[0] = offset - 1 - 6
					bezel[1] = offset - 4 - 6
					bezel[2] = offset - 6
					bezel2[0] = offset - 1
					bezel2[1] = offset - 4
					bezel2[2] = offset
				} else {
					bezel[0] = offset - 1
					bezel[1] = offset - 4
					bezel[2] = offset
				}
			} else {
				adjust(&v[0], lb.ux, lb.uy, scale*halfthick)
				adjust(&pl.vertices[offset-4], plb.ux, plb.uy, -scale*halfthick)
				if pl.glowing {
					adjust(&v[6], lb.ux, lb.uy, scale*halfthick/2)
					// undo half of the adjustment that was incorrect
					adjust(&pl.vertices[offset-4], plb.ux, plb.uy, +scale*halfthick/2)
					adjust(&pl.vertices[offset-10], plb.ux, plb.uy, -scale*halfthick)
					bezel[0] = offset - 1 - 6
					bezel[1] = offset + 1
					bezel[2] = offset - 3 - 6
					bezel2[0] = offset - 1
					bezel2[1] = offset + 1 + 6
					bezel2[2] = offset - 3
				} else {
					bezel[0] = offset - 1
					bezel[1] = offset + 1
					bezel[2] = offset - 3
				}
			}
		}
		if pl.Blend {
			populateJoinedRGB(v, r0, g0, b0, r1, g1, b1, alpha)
		} else {
			populateJoinedRGB(v, r1, g1, b1, r1, g1, b1, alpha)
		}

		if pl.DebugColor {
			for i := 0; i < 6; i++ {
				v[i].ColorR, v[i].ColorG, v[i].ColorB, v[i].ColorA = debugColors[i][0], debugColors[i][1], debugColors[i][2], 1.0
			}
		}

		// rotate colors
		r0, g0, b0 = r1, g1, b1
		// rotate points
		prev = next
		plb = lb
		px, py = nx, ny
		// bump count since we drew a segment
		count++
	}
	return count * vsPerSegment, count * idxsPerSegment
}

func populateUnjoinedRGB(v []ebiten.Vertex, r0, g0, b0, r1, g1, b1, alpha float32) {
	if len(v) == 8 {
		alpha *= 0.75
		v[4].ColorR, v[4].ColorG, v[4].ColorB, v[4].ColorA = 1, 1, 1, alpha
		v[5].ColorR, v[5].ColorG, v[5].ColorB, v[5].ColorA = 1, 1, 1, alpha
		v[6].ColorR, v[6].ColorG, v[6].ColorB, v[6].ColorA = 1, 1, 1, alpha
		v[7].ColorR, v[7].ColorG, v[7].ColorB, v[7].ColorA = 1, 1, 1, alpha
	}
	v[0].ColorR, v[0].ColorG, v[0].ColorB, v[0].ColorA = r0, g0, b0, alpha
	v[1].ColorR, v[1].ColorG, v[1].ColorB, v[1].ColorA = r0, g0, b0, alpha
	v[2].ColorR, v[2].ColorG, v[2].ColorB, v[2].ColorA = r1, g1, b1, alpha
	v[3].ColorR, v[3].ColorG, v[3].ColorB, v[3].ColorA = r1, g1, b1, alpha
}

func populateUnjoinedVs(v []ebiten.Vertex, px, py, nx, ny float32, lb LineBits, halfthick, scale float32) {
	v[0].DstX = float32(px+lb.nx*halfthick) * scale
	v[0].DstY = float32(py+lb.ny*halfthick) * scale
	v[1].DstX = float32(px-lb.nx*halfthick) * scale
	v[1].DstY = float32(py-lb.ny*halfthick) * scale
	v[2].DstX = float32(nx+lb.nx*halfthick) * scale
	v[2].DstY = float32(ny+lb.ny*halfthick) * scale
	v[3].DstX = float32(nx-lb.nx*halfthick) * scale
	v[3].DstY = float32(ny-lb.ny*halfthick) * scale
	if len(v) == 8 {
		halfthick /= 4
		v[4].DstX = float32(px+lb.nx*halfthick) * scale
		v[4].DstY = float32(py+lb.ny*halfthick) * scale
		v[5].DstX = float32(px-lb.nx*halfthick) * scale
		v[5].DstY = float32(py-lb.ny*halfthick) * scale
		v[6].DstX = float32(nx+lb.nx*halfthick) * scale
		v[6].DstY = float32(ny+lb.ny*halfthick) * scale
		v[7].DstX = float32(nx-lb.nx*halfthick) * scale
		v[7].DstY = float32(ny-lb.ny*halfthick) * scale
	}
}

// The easy case: We compute four vertices per segment,
// and draw two triangles using them, giving us an easy quad.
func (pl *PolyLine) computeUnjoinedVertices(halfthick, alpha, scale float32) (vertices, indices int) {
	segments := len(pl.Points) / 2
	if segments < 1 {
		// fail
		fmt.Fprintf(os.Stderr, "polyline of %d segments can't be drawn\n", segments)
		return
	}
	// populate with the SrcX, SrcY values.
	vsPerSegment := pl.vsPerSegment()
	if len(pl.vertices) < segments*vsPerSegment {
		fv := lineData.vsByR[pl.render]
		pl.vertices = make([]ebiten.Vertex, 0, segments*vsPerSegment)
		for i := 0; i < segments; i++ {
			pl.vertices = append(pl.vertices, fv[0:4]...)
		}
		if pl.glowing {
			for i := 0; i < segments; i++ {
				pl.vertices = append(pl.vertices, fv[0:4]...)
			}
		}
	}
	// indices can never change, conveniently!
	idxsPerSegment := pl.idxsPerSegment()
	if len(pl.indices) < segments*idxsPerSegment {
		// note: we need 6 indexes per 4 vertexes, no matter whether
		// there's 4 or 8 vertexes per segment.
		offset := uint16(len(pl.indices)/6) * 4
		for i := len(pl.indices); i < segments*idxsPerSegment; i += 6 {
			pl.indices = append(pl.indices,
				offset+1, offset+0, offset+2,
				offset+2, offset+3, offset+1)
			offset += 4
		}
	}
	prev := pl.Points[0]
	r0, g0, b0, _ := pl.Palette.Float32(prev.P)
	count := 0

	// Unjoined: We draw one segment for each pair.
	px, py := (prev.X*pl.scale)+pl.offsetX, (prev.Y*pl.scale)+pl.offsetY
	for idx, next := range pl.Points[1:] {
		nx, ny := (next.X*pl.scale)+pl.offsetX, (next.Y*pl.scale)+pl.offsetY
		if next.Skip {
			prev = next
			r0, g0, b0, _ = pl.Palette.Float32(next.P)
			px, py = nx, ny
			continue
		}
		lb := linebits(prev.X, prev.Y, next.X, next.Y)
		if (lb.l == 0) || ((idx % 2) == 1) {
			// don't draw 0-length line, don't divide by zero, but
			// do update the point so we use the right color to draw
			// the next segment.
			prev = next
			r0, g0, b0, _ = pl.Palette.Float32(next.P)
			px, py = nx, ny
			continue
		}
		// compute normal x/y values, scaled to unit length
		lb.nx, lb.ny = lb.dy/lb.l, -lb.dx/lb.l
		r1, g1, b1, _ := pl.Palette.Float32(next.P)
		offset := uint16(count * vsPerSegment)
		v := pl.vertices[offset : offset+uint16(vsPerSegment)]
		populateUnjoinedVs(v, px, py, nx, ny, lb, halfthick, scale)
		if pl.Blend {
			populateUnjoinedRGB(v, r0, g0, b0, r1, g1, b1, alpha)
		} else {
			populateUnjoinedRGB(v, r1, g1, b1, r1, g1, b1, alpha)
		}

		// rotate colors
		r0, g0, b0 = r1, g1, b1
		// rotate points
		prev = next
		px, py = nx, ny
		// bump count since we drew a segment
		count++
	}
	return count * vsPerSegment, count * idxsPerSegment
}

// Draw renders the line on the target, using the sprite's drawimage
// options modified by color and location of line segments.
func (pl *PolyLine) Draw(target *ebiten.Image, alpha float32, scale float32) {
	thickness := pl.Thickness
	// no invisible lines plz
	if thickness == 0 {
		thickness = 0.7
	}
	halfthick := thickness / 2
	var vCount, iCount int
	if pl.dirty {
		if pl.Joined {
			vCount, iCount = pl.computeJoinedVertices(halfthick, alpha, scale)
		} else {
			vCount, iCount = pl.computeUnjoinedVertices(halfthick, alpha, scale)
		}
		// trim to actually returned length
		pl.vertices = pl.vertices[:vCount]
		pl.indices = pl.indices[:iCount]
		pl.dirty = false
	}

	// draw the triangles
	target.DrawTriangles(pl.vertices, pl.indices, lineData.img, &ebiten.DrawTrianglesOptions{CompositeMode: ebiten.CompositeModeLighter})
	if pl.debug != nil {
		pl.debug.Draw(target, alpha, scale)
	}
	ebitenutil.DebugPrint(target, pl.status)
}

// Length yields the number of points in the line.
func (pl *PolyLine) Length() int {
	return len(pl.Points)
}

// Reset removes all points.
func (pl *PolyLine) Reset() {
	if pl.Points != nil {
		pl.Points = pl.Points[:0]
	}
	pl.Dirty()
}

// Point yields a given point within the line.
// If you modify the point, it's on you to call pl.Dirty().
func (pl *PolyLine) Point(i int) *LinePoint {
	if i < 0 || i >= len(pl.Points) {
		return nil
	}
	return &pl.Points[i]
}

// Add adds a new point to the line.
func (pl *PolyLine) Add(x, y float32, p Paint) {
	pt := LinePoint{X: x, Y: y, P: p}
	pl.Points = append(pl.Points, pt)
	pl.Dirty()
}
