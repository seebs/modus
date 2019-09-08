package modes

import (
	"strings"

	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/keys"
	"seebs.net/modus/sound"
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
	Tick(voice *sound.Voice, km keys.Map) (bool, error)
	// Draw renders the mode's internal state graphically to the provided
	// screen.
	Draw(screen *ebiten.Image) error
}

// ModeFilter defines the way we're currently filtering modes. The set
// of modes is whitelist (or all modes if no whitelist is present) minus
// blacklist.
type ModeFilter struct {
	whitelist map[string]struct{}
	blacklist map[string]struct{}
}

// Whitelist whitelists the given mode.
func (mf *ModeFilter) Whitelist(add string) {
	if mf.whitelist == nil {
		mf.whitelist = map[string]struct{}{add: struct{}{}}
		return
	}
	mf.whitelist[add] = struct{}{}
}

// Blacklist blacklists the given mode.
func (mf *ModeFilter) Blacklist(remove string) {
	if mf.blacklist == nil {
		mf.blacklist = map[string]struct{}{remove: struct{}{}}
		return
	}
	mf.blacklist[remove] = struct{}{}
}

func (mf *ModeFilter) Apply(ml ModeList) (results []Mode) {
	if len(mf.whitelist) == 0 {
		for _, mode := range ml.list {
			name := mode.Name()
			if _, ok := mf.blacklist[name]; !ok {
				results = append(results, mode)
			}
		}
		return results
	}
	for _, mode := range ml.list {
		name := mode.Name()
		if _, ok := mf.whitelist[name]; ok {
			if _, ok := mf.blacklist[name]; !ok {
				results = append(results, mode)
			}
		}
	}
	return results
}

// ApplyList applies a given list, in the form +mode,-mode,...
// Bare modes are whitelisted, +mode is whitelisted, -mode is blacklisted.
func (mf *ModeFilter) ApplyList(list string) {
	if list == "" {
		return
	}
	mods := strings.Split(list, ",")
	for _, m := range mods {
		if m == "" {
			continue
		}
		if m[0] == '-' {
			Blacklist(m[1:])
		} else if m[0] == '+' {
			Whitelist(m[1:])
		} else {
			Whitelist(m)
		}
	}
}

type ModeList struct {
	list []Mode
}

func (ml *ModeList) Add(m Mode) {
	ml.list = append(ml.list, m)
}

var defaultFilter ModeFilter
var defaultList ModeList

// ListModes provides a list of the available modes.
func ListModes() []Mode {
	return defaultFilter.Apply(defaultList)
}

// Whitelist modifies the default filter.
func Whitelist(add string) {
	defaultFilter.Whitelist(add)
}

// Blacklist modifies the default filter.
func Blacklist(remove string) {
	defaultFilter.Blacklist(remove)
}

// ApplyList modifies the default filter.
func ApplyList(list string) {
	defaultFilter.ApplyList(list)
}
