package modes

import (
	"fmt"
	"os"
	"strings"
	"testing"

	math "github.com/chewxy/math32"
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

var detailLevels = []int{30}

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

type gravityTestWrapper struct {
	scene          *dotGridScene
	modeName       string
	base           g.DotGridBase
	state0, state1 g.DotGridState
	compute        func(int, int, *dotGridScene, g.DotGridBase, g.DotGridState, g.DotGridState) string
	computeInit    func(int, int, *dotGridScene, g.DotGridBase, g.DotGridState)
}

const gravityTestScale = 16
const gravityTestN = gravityTestScale * gravityTestScale

func fuzzyComp(x, y float32) error {
	if x == y {
		return nil
	}
	diff := math.Abs(x - y)
	xa := math.Abs(x)
	ya := math.Abs(y)
	if (xa/diff < 1000) || (ya/diff < 1000) {
		return fmt.Errorf("expected %f, got %f", x, y)
	}
	return nil
}

func TestGravityModes(t *testing.T) {
	var wrappers []gravityTestWrapper
	for _, mode := range ListModes() {
		if strings.HasPrefix(mode.Name(), "gravity") {
			if mode, ok := mode.(dotGridMode); ok {
				s := &dotGridScene{
					mode: mode,
				}
				wrapper := gravityTestWrapper{
					scene:       s,
					modeName:    mode.Name(),
					compute:     mode.compute,
					computeInit: mode.computeInit,
					base: g.DotGridBase{
						Locs: make([]g.FLoc, gravityTestN),
						Vecs: make([]g.FVec, gravityTestN),
					},
					state0: g.DotGridState{
						Locs: make([]g.FLoc, gravityTestN),
						P:    make([]g.Paint, gravityTestN),
						A:    make([]float32, gravityTestN),
						S:    make([]float32, gravityTestN),
					},
					state1: g.DotGridState{
						Locs: make([]g.FLoc, gravityTestN),
						P:    make([]g.Paint, gravityTestN),
						A:    make([]float32, gravityTestN),
						S:    make([]float32, gravityTestN),
					},
				}
				wrappers = append(wrappers, wrapper)
			}
		}
	}
	if len(wrappers) < 2 {
		t.Skip("can't test gravity modes without at least two existing")
	}
	for i := range wrappers[0].base.Locs {
		x, y := i%gravityTestScale, i/gravityTestScale
		fx, fy := float32(x-16)/gravityTestScale, float32(y-16)/gravityTestScale
		for j := range wrappers {
			wrappers[j].base.Locs[i] = g.FLoc{X: fx, Y: fy}
		}
	}
	for j := range wrappers {
		wrappers[j].computeInit(gravityTestScale, gravityTestScale, wrappers[j].scene, wrappers[j].base, wrappers[j].state0)
	}
	closeCount := 0
	for i := 0; i < 5; i++ {
		for j := range wrappers {
			wrappers[j].compute(gravityTestScale, gravityTestScale, wrappers[j].scene, wrappers[j].base, wrappers[j].state0, wrappers[j].state1)
			wrappers[j].state1, wrappers[j].state0 = wrappers[j].state0, wrappers[j].state1
		}
		referenceLocs := wrappers[0].state0.Locs
		referenceVecs := wrappers[0].base.Vecs
		for _, wrapper := range wrappers[1:] {
			subjectLocs := wrapper.state0.Locs
			subjectVecs := wrapper.base.Vecs
			for j, loc := range referenceLocs {
				if subjectLocs[j] != loc {
					err := fuzzyComp(subjectLocs[j].X, loc.X)
					if err == nil {
						err = fuzzyComp(subjectLocs[j].Y, loc.Y)
						if err != nil {
							t.Fatalf("wrapper %s: iteration %d, loc index %d, expected %f,%f, got %f,%f",
								wrapper.modeName, i, j, loc.X, loc.Y, subjectLocs[j].X, subjectLocs[j].Y)
						}
						closeCount++
					}
				}
			}
			for j, vec := range referenceVecs {
				if subjectVecs[j] != vec {
					err := fuzzyComp(subjectVecs[j].X, vec.X)
					if err == nil {
						err = fuzzyComp(subjectVecs[j].Y, vec.Y)
						if err != nil {
							t.Fatalf("wrapper %s: iteration %d, vec index %d, expected %f,%f, got %f,%f",
								wrapper.modeName, i, j, vec.X, vec.Y, subjectVecs[j].X, subjectVecs[j].Y)
						}
						closeCount++
					}
				}
			}
		}
	}
}
