package modes

import (
	"fmt"
	"math/rand"

	math "github.com/chewxy/math32"
	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/keys"
	"seebs.net/modus/sound"
)

// vectorMode is one of the internal modes based on vector graphics
type vectorMode struct {
	name        string
	cycleTime   int // number of ticks to go by between updates
	compute     func(*vectorScene, keys.Map) string
	computeInit func(*vectorScene)
}

const vectorCycleTime = 1

const shieldSegments = 30

const weaveInterpolate = 3

var vectorModes = []vectorMode{
	{name: "Test", cycleTime: vectorCycleTime, compute: simpleDemo, computeInit: simpleDemoInit},
}

func init() {
	for _, mode := range vectorModes {
		allModes = append(allModes, mode)
	}
}

func (m vectorMode) Name() string {
	return fmt.Sprintf("vector%s", m.name)
}

func (m vectorMode) Description() string {
	return fmt.Sprintf("vector: %s", m.name)
}

func (m vectorMode) New(gctx *g.Context, detail int, p *g.Palette) (Scene, error) {
	return newVectorScene(m, gctx, detail, p)
}

type knotProto struct {
	pts []g.LinePoint
}

var sampleKnots = map[string]knotProto{
	"ship": knotProto{
		pts: []g.LinePoint{
			{X: 0.5, Y: 0, Open: true},
			{X: 0, Y: .25},
			{X: -0.25, Y: 0},
			{X: 0, Y: -0.25},
			{X: 0.5, Y: 0, Close: true},
			{X: 0, Y: .125, Skip: true, P: 9},
			{X: 0, Y: -.125, P: 12},
		},
	},
}

type polarPos struct{ x, y, theta float32 }

var shieldPositions []polarPos
var shieldSegmentLen float32

func (b *bouncer) setShield() {
	for i := 0; i < shieldSegments; i++ {
		var s, c float32
		x, y, theta := shieldPositions[i].x, shieldPositions[i].y, shieldPositions[i].theta
		if i%2 == 0 {
			s, c = math.Sincos(theta + b.shieldSpin)
		} else {
			s, c = math.Sincos(theta - b.shieldSpin)
		}
		s, c = s*shieldSegmentLen/2, c*shieldSegmentLen/2
		b.shield.Points[i*2].X, b.shield.Points[i*2].Y = x+c, y+s
		b.shield.Points[i*2+1].X, b.shield.Points[i*2+1].Y = x-c, y-s
		b.shield.Points[i*2].P = (b.shield.Points[i*2].P + 1) % (6 * weaveInterpolate)
		b.shield.Points[i*2+1].P = (b.shield.Points[i*2+1].P + 1) % (6 * weaveInterpolate)
	}
}

func (b *bouncer) initShield() {
	for i := 0; i < shieldSegments; i++ {
		b.shield.Points[i*2].Skip = true
		b.shield.Points[i*2].P = g.Paint(i*weaveInterpolate) % (6 * weaveInterpolate)
		b.shield.Points[i*2+1].P = g.Paint(i*weaveInterpolate+3) % (6 * weaveInterpolate)
	}
}

// simpleDemo is just a trivial test case
func simpleDemoInit(s *vectorScene) {
	shieldSegmentLen = (math.Pi * 2) / shieldSegments
	for i := 0; i < shieldSegments; i++ {
		theta := shieldSegmentLen * float32(i)
		pp := polarPos{theta: theta + math.Pi/2}
		pp.y, pp.x = math.Sincos(theta)
		shieldPositions = append(shieldPositions, pp)
	}
	for i := 0; i < 1; i++ {
		proto := sampleKnots["ship"]
		k1 := s.wv.NewKnot(len(proto.pts))
		b := bouncer{ship: k1, pOffset: i}
		b.shield = s.wv.NewKnot(shieldSegments * 2)
		b.shield.Size = 0.6
		b.initShield()
		b.setShield()
		b.shield.Dirty()
		// fmt.Printf("shield points: %v, %v\n", b.shield.Points[0], b.shield.Points[1])
		copy(k1.Points, proto.pts)
		k1.Dirty()
		k1.Size = float32(1.0) / float32(i+1)
		k1.X, k1.Y = -0.5+float32(i&1), -0.5+float32(i>>1)
		for j := range k1.Points {
			k1.Points[j].P = s.palette.Inc(k1.Points[j].P, b.pOffset)
		}
		b.pt = g.MovingPoint{Loc: g.Point{X: k1.X, Y: k1.Y}, Velocity: g.Vec{X: -k1.X * .002 * (float32(i) + 1), Y: -k1.Y * .002 * (float32(i) + 1)}, Bounds: s.bounds}
		b.shield.X, b.shield.Y = b.pt.Loc.X, b.pt.Loc.Y
		s.bouncers = append(s.bouncers, b)
	}
}

func simpleDemo(s *vectorScene, km keys.Map) string {
	b := &s.bouncers[0]
	sin, cos := math.Sincos(b.ship.Theta)
	if s.keysReady {
		if km.Down(ebiten.KeyW, ebiten.KeyUp) {
			b.pt.Velocity.X += cos * .0001
			b.pt.Velocity.Y += sin * .0001
			p := s.pt.Add(g.SecondSplasher, g.Paint(b.pOffset+1), -0.01, 0)
			p.Alpha = 0
			p.Scale = rand.Float32()/16 + 0.125
			p.DX = -(0.0625 + (rand.Float32() / 8))
			p.DY = (rand.Float32() - 0.5) / 8
			if math.Abs(p.DY) > 0.05 {
				awayFrom05 := math.Abs(p.DY) - 0.05
				// gets us a range of about 0 to 0.0125, which we want to convert to about 1 palette's
				// worth of shift
				p.P = s.palette.Inc(p.P, -int(awayFrom05*80*weaveInterpolate))
			}
			if math.Abs(p.DY) < 0.01 {
				// exactly 0.01: we get 0.01
				// exactly 0: we get 0.02
				// multiplying by 75 gets us a range from .75 to 1.5, so "around 1" palette's worth of shift
				awayFrom01 := 0.02 - math.Abs(p.DY)
				p.P = s.palette.Inc(p.P, int(awayFrom01*100*weaveInterpolate))
			}
			p.DTheta = p.DY * 2
		}
		if km.Down(ebiten.KeyA, ebiten.KeyLeft) {
			b.ship.Theta -= .05
		}
		if km.Down(ebiten.KeyD, ebiten.KeyRight) {
			b.ship.Theta += 0.05
		}
	} else {
		if km.AllUp(ebiten.KeyW, ebiten.KeyUp, ebiten.KeyA, ebiten.KeyLeft, ebiten.KeyD, ebiten.KeyRight) {
			s.keysReady = true
		}
	}
	s.pt.X, s.pt.Y = b.pt.Loc.X-cos*.1, b.pt.Loc.Y-sin*.1
	s.pt.Theta = b.ship.Theta
	for idx := range s.bouncers {
		b := &s.bouncers[idx]
		b.shieldSpin += .1
		b.pt.Update()
		b.ship.X, b.ship.Y = b.pt.Loc.X, b.pt.Loc.Y
		b.shield.X, b.shield.Y = b.pt.Loc.X, b.pt.Loc.Y
		b.ship.Dirty()
		b.setShield()
		b.shield.Dirty()
	}
	return ""
}

type vectorScene struct {
	palette   *g.Palette
	gctx      *g.Context
	mode      vectorMode
	wv        *g.Weave
	pt        *g.Particles
	tx        *g.Text
	detail    int
	cycle     int
	t0        float32
	bouncers  []bouncer
	bounds    g.Region
	keysReady bool
}

type bouncer struct {
	pt         g.MovingPoint
	pOffset    int
	ship       *g.Knot
	shield     *g.Knot
	shieldSpin float32
}

func newVectorScene(m vectorMode, gctx *g.Context, detail int, p *g.Palette) (*vectorScene, error) {
	sc := &vectorScene{mode: m, gctx: gctx, detail: detail, palette: p.Interpolate(weaveInterpolate)}
	_, _, _, cx, cy := gctx.Centered()
	sc.bounds = g.Region{
		Min: g.Point{X: -1 - cx, Y: -1 - cy},
		Max: g.Point{X: 1 + cx, Y: 1 + cy},
	}
	err := sc.Reset(detail, p)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *vectorScene) Mode() Mode {
	return s.mode
}

func (s *vectorScene) Reset(detail int, p *g.Palette) error {
	_ = s.Hide()
	s.palette = p.Interpolate(weaveInterpolate)
	err := s.Display()
	if err != nil {
		return err
	}
	// and then reset the grid, and the knights, i guess?
	return nil
}

func (s *vectorScene) Display() error {
	s.wv = s.gctx.NewWeave(16, s.palette)
	s.pt = s.gctx.NewParticles(16, 1, s.palette)
	var err error
	s.tx, err = s.gctx.NewText("arcade", 24, s.palette)
	if s.mode.computeInit != nil {
		s.mode.computeInit(s)
	}
	return err
}

func (s *vectorScene) Hide() error {
	s.wv = nil
	s.pt = nil
	s.tx = nil
	return nil
}

func (s *vectorScene) Tick(voice *sound.Voice, km keys.Map) (bool, error) {
	s.cycle = (s.cycle + 1) % s.mode.cycleTime
	s.t0++
	if s.cycle != 0 {
		return false, nil
	}
	if s.mode.compute != nil {
		s.wv.SetStatus(s.mode.compute(s, km))
		s.pt.Tick()
	}
	s.tx.P = s.palette.Inc(s.tx.P, 1)
	s.tx.Text = "got here"
	return true, nil
}

func (s *vectorScene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		// s.tx.Draw(t, 1.0, scale)
		s.wv.Draw(t, 1.0, scale)
		s.pt.Draw(t, scale)
	})
	return nil
}
