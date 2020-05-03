package g_test

import (
	"testing"

	"seebs.net/modus/g"
)

func BenchmarkParticleOp(b *testing.B) {
	c := g.NewContext(1280, 960, false)
	ps := c.NewParticles(20, 1, g.Palettes["rainbow"])
	for i := 0; i < b.N; i++ {
		_ = ps.Add(g.SecondSplasher, 0, 230, 240)
	}
	for !ps.Tick() {
		// do nothing
	}
}
