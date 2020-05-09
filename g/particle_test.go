package g_test

import (
	"testing"

	"seebs.net/modus/g"
)

func BenchmarkParticleOp(b *testing.B) {
	c := g.NewContext(1280, 960, false)
	params := g.ParticleParams{
		State: g.ParticleState{
			ParticlePos: g.ParticlePos{X: 23, Y: 23, Theta: 1},
		},
		Delta: g.ParticlePos{
			X:     1,
			Y:     1,
			Theta: 1,
		},
		Delay: 3,
	}
	for i := 0; i < b.N; i++ {
		m := &g.MovingParticles{}
		ps := c.NewParticleSystem(20, 1, g.Palettes["rainbow"], m)
		ps.Anim, _ = m.Animation("splasher", 30)
		for j := 0; j < 10000; j++ {
			params.Delay = (params.Delay + 1) & 7
			err := ps.Add(params)
			if err != nil {
				b.Fatalf("unexpected error adding particle: %v", err)
			}
		}
		for !ps.Tick() {
			// do nothing
		}
	}
}
