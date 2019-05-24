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

// NewHexGrid returns a grid of hexes with width "w"
// across its wider dimension.
func (c *Context) NewHexGrid(w int, r RenderType, p *Palette) *HexGrid {
	return newHexGrid(w, r, p, c.w, c.h)
}

// NewDotGrid returns a grid of dots with width "w" across its wider
// dimension.
func (c *Context) NewDotGrid(w int, thickness float32, depth int, r RenderType, p *Palette) *DotGrid {
	return newDotGrid(w, thickness, depth, r, p, c.w, c.h)
}

// NewSpiral returns a spiral for the given Context.
func (c *Context) NewSpiral(depth int, r RenderType, points int, p *Palette, cycles int, offset int) *Spiral {
	return newSpiral(depth, r, points, p, cycles, offset)
}

func (c *Context) NewPolyline(thickness int, r RenderType, p *Palette) *PolyLine {
	return newPolyLine(thickness, r, p)
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
		c.fsaa.Fill(color.RGBA{0, 0, 0, 0})
		fn(c.fsaa, 2)
		screen.DrawImage(c.fsaa, c.fsaaOp)
	} else {
		fn(screen, 1)
	}
}
