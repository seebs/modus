package modes

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"seebs.net/modus/g"
)

func benchmarkOneModeTick(b *testing.B, scene Scene) {
	for i := 0; i < b.N; i++ {
		_, err := scene.Tick(nil, nil)
		if err != nil {
			b.Fatalf("error in tick: %v", err)
		}
	}
}

func benchmarkOneModeTickDraw(b *testing.B, scene Scene) {
	for i := 0; i < b.N; i++ {
		_, err := scene.Tick(nil, nil)
		if err != nil {
			b.Fatalf("error in tick: %v", err)
		}
		err = scene.Draw(nil)
		if err != nil {
			b.Fatalf("error in draw: %v", err)
		}
	}
}

var detailLevels = []int{5, 10, 20, 30}

func TestMain(m *testing.M) {
	ApplyList(os.Getenv("MODUS_MODES"))
	modes := ListModes()
	names := make([]string, len(modes))
	for i, m := range modes {
		names[i] = m.Name()
	}
	fmt.Printf("Testing modes: %s\n", strings.Join(names, ", "))
	os.Exit(m.Run())
}

func Benchmark_ModeTick(b *testing.B) {
	c := g.NewContext(1280, 960, false)
	p := g.Palettes["rainbow"]
	for _, mode := range ListModes() {
		for _, detail := range detailLevels {
			scene, err := mode.New(c, detail, p)
			if err != nil {
				b.Fatalf("failed to initialize scene %s@%d: %v", mode.Name(), detail, err)
			}
			err = scene.Display()
			if err != nil {
				b.Fatalf("failed to display scene %s@%d: %v", mode.Name(), detail, err)
			}
			b.Run(fmt.Sprintf("Tick/%s@%d", mode.Name(), detail), func(b *testing.B) {
				benchmarkOneModeTick(b, scene)
			})
		}
	}
}

func Benchmark_ModeDraw(b *testing.B) {
	c := g.NewContext(1280, 960, false)
	p := g.Palettes["rainbow"]
	for _, mode := range ListModes() {
		for _, detail := range detailLevels {
			scene, err := mode.New(c, detail, p)
			if err != nil {
				b.Fatalf("failed to initialize scene %s@%d: %v", mode.Name(), detail, err)
			}
			err = scene.Display()
			if err != nil {
				b.Fatalf("failed to display scene %s@%d: %v", mode.Name(), detail, err)
			}
			// note: you can't really just benchmark the draw, because often if
			// no ticks have happened it'll use cached results.
			b.Run(fmt.Sprintf("Draw/%s@%d", mode.Name(), detail), func(b *testing.B) {
				benchmarkOneModeTickDraw(b, scene)
			})
		}
	}
}
