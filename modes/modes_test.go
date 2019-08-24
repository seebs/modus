package modes

import (
	"fmt"
	"testing"

	"seebs.net/modus/g"
)

func benchmarkOneMode(b *testing.B, scene Scene) {
	for i := 0; i < b.N; i++ {
		_, err := scene.Tick(nil, nil)
		if err != nil {
			b.Fatalf("error in tick: %v", err)
		}
	}
}

var detailLevels = []int{5, 10, 20}

func Benchmark_Modes(b *testing.B) {
	c := g.NewContext(1280, 960, false)
	p := g.Palettes["rainbow"]
	for _, mode := range allModes {
		for _, detail := range detailLevels {
			scene, err := mode.New(c, detail, p)
			if err != nil {
				b.Fatalf("failed to initialize scene %s@%d: %v", mode.Name(), detail, err)
			}
			err = scene.Display()
			if err != nil {
				b.Fatalf("failed to display scene %s@%d: %v", mode.Name(), detail, err)
			}
			b.Run(fmt.Sprintf("%s@%d", mode.Name(), detail), func(b *testing.B) {
				benchmarkOneMode(b, scene)
			})
		}
	}
}
