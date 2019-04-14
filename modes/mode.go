package modes

import (
	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
)

// Mode represents one of the modus's display modes.
type Mode interface {
	Name() string
	Description() string
	New(ctx *g.Context, detail int, p *g.Palette) (Scene, error)
}

// Scene represents a specific instantiation of a display mode.
type Scene interface {
	// Mode indicates what mode this scene is for.
	Mode() Mode
	// Display sets up any needed graphical objects.
	Display() error
	// Hide unreferences any graphical objects, but preserves the
	// mode's state.
	Hide() error
	// Reset re-initializes the mode's state. The detail parameter
	// specifies an approximate level of detail -- for instance, the
	// number of squares across that a grid should be.
	Reset(detail int, p *g.Palette) error
	// Tick updates the mode's internal state, such as moving objects
	// around the screen.
	Tick() error
	// Draw renders the mode's internal state graphically to the provided
	// screen.
	Draw(screen *ebiten.Image) error
}

var allModes []Mode

// ListModes provides a list of the available modes
func ListModes() []Mode {
	return allModes
}
