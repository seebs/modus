package keys

import (
	"github.com/hajimehoshi/ebiten"
)

// handle keypresses
const (
	PRESS   = 0x01
	RELEASE = 0x02
	HOLD    = 0x03
)

type Map map[ebiten.Key]byte

func NewMap(keys ...ebiten.Key) Map {
	m := make(Map, len(keys))
	for _, k := range keys {
		m[k] = 0
	}
	return m
}

// State returns the current state of a key
func (km Map) State(k ebiten.Key) byte {
	if _, ok := km[k]; !ok {
		km[k] = 0
	}
	return km[k] & HOLD
}

func (km Map) Pressed(k ebiten.Key) bool {
	return km.State(k) == PRESS
}

func (km Map) Released(k ebiten.Key) bool {
	return km.State(k) == RELEASE
}

func (km Map) Held(k ebiten.Key) bool {
	return km.State(k) == HOLD
}

func (km Map) Down(k ebiten.Key) bool {
	return (km.State(k) & PRESS) != 0
}

func (km Map) Up(k ebiten.Key) bool {
	return (km.State(k) & PRESS) == 0
}

func (km Map) Update() {
	for i := range km {
		state := byte(0)
		if ebiten.IsKeyPressed(i) {
			state = 1
		}
		km[i] = ((km[i] & 0x1) << 1) | state
	}
}
