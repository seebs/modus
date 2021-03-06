package g

import (
	"image/color"

	"github.com/hajimehoshi/ebiten"
)

// Context represents a graphics context, basically providing a cache of
// screen size for now.
type Context struct {
	w, h        int
	fsaa        *ebiten.Image
	fsaaOp      *ebiten.DrawImageOptions
	multisample bool
}

// RenderType represents the way a thing is drawn; for instance, which
// of several square textures to use.
type RenderType int

// NewContext creates a new context, corresponding to a window with
// the specified width and height. If multisample is set, it scales
// everything by 2x internally.
func NewContext(w, h int, multisample bool) *Context {
	ctx := &Context{w: w, h: h, multisample: multisample}
	if multisample {
		ctx.fsaa, _ = ebiten.NewImage(w*2, h*2, ebiten.FilterLinear)
		ctx.fsaaOp = &ebiten.DrawImageOptions{}
		ctx.fsaaOp.GeoM.Scale(0.5, 0.5)
	}
	return ctx
}

// NewSquareGrid returns a grid of squares with width "w"
// across its wider dimension.
func (c *Context) NewSquareGrid(w int, r RenderType, p *Palette) *SquareGrid {
	return newSquareGrid(w, r, p, c.w, c.h)
}

// NewParticles returns a particle-emitter source.
func (c *Context) NewParticleSystem(w int, r RenderType, p *Palette, particles Particles) *ParticleSystem {
	scale, ox, oy, _, _ := c.Centered()
	return newParticles(float32(c.w)/float32(w), r, p, scale, ox, oy, particles)
}

// NewHexGrid returns a grid of hexes with width "w"
// across its wider dimension.
func (c *Context) NewHexGrid(w int, r RenderType, p *Palette) *HexGrid {
	return newHexGrid(w, r, p, c.w, c.h)
}

// NewDotGrid returns a grid of dots with width "w" across its wider
// dimension.
func (c *Context) NewDotGrid(w int, thickness float32, depth int, r RenderType, p *Palette) *DotGrid {
	scale, ox, oy, cx, cy := c.Centered()
	return newDotGrid(w, thickness, depth, r, p, scale, ox, oy, cx, cy)
}

// NewSpiral returns a spiral for the given Context.
func (c *Context) NewSpiral(depth int, r RenderType, points int, p *Palette, cycles int, offset int) *Spiral {
	scale, ox, oy, _, _ := c.Centered()
	return newSpiral(depth, r, points, p, cycles, offset, scale, ox, oy)
}

func (c *Context) NewPolyline(thickness int, r RenderType, p *Palette) *PolyLine {
	scale, ox, oy, _, _ := c.Centered()
	return newPolyLine(thickness, r, p, scale, ox, oy)
}

func (c *Context) NewWeave(thickness int, p *Palette) *Weave {
	scale, ox, oy, _, _ := c.Centered()
	return newWeave(thickness, p, scale, ox, oy)
}

func (c *Context) NewText(fontName string, size int, p *Palette) (*Text, error) {
	scale, ox, oy, _, _ := c.Centered()
	return newText(fontName, size, p, scale, ox, oy)
}

// Centered yields a scale factor and X/Y offsets for converting X/Y values in
// a -1..+1 range to screen coordinates for the ebiten.Image the context uses
// to render to. The entire [-1,-1] to [1,1] space is on-screen; if the screen
// is not square, the wider coordinate's visible range will be broader. The
// coordinate offsets represent the additional visible area on each side of
// the screen; one of them (corresponding to the smaller dimension) is always
// zero.
//
// Example: If the context is 1200x800, scale should be 400, and offsetY should be
// 400, so Y -1 converts to 0, and Y 1 converts to 800. offsetX should be 600.
// Meanwhile, coordX will be 0.5; the lowest X coordinate should be -1.5, and
// the highest 1.5.
func (c *Context) Centered() (scale, offsetX, offsetY, coordX, coordY float32) {
	if c.w > c.h {
		scale = float32(c.h) / 2
		offsetY = scale
		offsetX = float32(c.w) / 2
		coordX = (float32(c.w) - float32(c.h)) / float32(c.h)
	} else {
		scale = float32(c.w) / 2
		offsetX = scale
		offsetY = float32(c.h) / 2
		coordY = (float32(c.h) - float32(c.w)) / float32(c.w)
	}
	return scale, offsetX, offsetY, coordX, coordY
}

func (c *Context) DrawSize() (int, int) {
	if c.multisample {
		return c.w * 2, c.h * 2
	} else {
		return c.w, c.h
	}
}

func (c *Context) Render(screen *ebiten.Image, fn func(*ebiten.Image, float32)) {
	if c.multisample {
		// should probably be checking these errors
		c.fsaa.Fill(color.RGBA{0, 0, 0, 0})
		fn(c.fsaa, 2)
		screen.DrawImage(c.fsaa, c.fsaaOp)
	} else {
		fn(screen, 1)
	}
}
