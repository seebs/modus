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

func (km Map) Pressed(ks ...ebiten.Key) bool {
	for _, k := range ks {
		if km.State(k) == PRESS {
			return true
		}
	}
	return false
}

func (km Map) Released(ks ...ebiten.Key) bool {
	for _, k := range ks {
		if km.State(k) == RELEASE {
			return true
		}
	}
	return false
}

func (km Map) Held(ks ...ebiten.Key) bool {
	for _, k := range ks {
		if km.State(k) == HOLD {
			return true
		}
	}
	return false
}

func (km Map) Down(ks ...ebiten.Key) bool {
	for _, k := range ks {
		if km.State(k)&PRESS != 0 {
			return true
		}
	}
	return false
}

func (km Map) AllDown(ks ...ebiten.Key) bool {
	for _, k := range ks {
		if km.State(k)&PRESS == 0 {
			return false
		}
	}
	return true
}

func (km Map) Up(ks ...ebiten.Key) bool {
	for _, k := range ks {
		if km.State(k)&PRESS == 0 {
			return true
		}
	}
	return false
}

func (km Map) AllUp(ks ...ebiten.Key) bool {
	for _, k := range ks {
		if km.State(k)&PRESS != 0 {
			return false
		}
	}
	return true
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
