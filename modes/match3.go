package modes

import (
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/keys"
	"seebs.net/modus/sound"
)

const match3GridScale = 0.8

// match3Mode is one of the internal modes based on match3 thing
type match3Mode struct {
	cycleTime int // number of ticks to go by between updates
}

const match3CycleTime = 1

var match3Modes = []match3Mode{
	{cycleTime: match3CycleTime},
}

func init() {
	for _, mode := range match3Modes {
		defaultList.Add(mode)
	}
}

func (m match3Mode) Name() string {
	return "match3"
}

func (m match3Mode) Description() string {
	return "match3 thing"
}

func (m match3Mode) New(gctx *g.Context, detail int, p *g.Palette) (Scene, error) {
	scale, ox, oy, _, _ := gctx.Centered()
	return newMatch3Scene(m, gctx, detail, p, scale, ox, oy)
}

type locCell struct {
	g.ILoc
	*g.HexCell
}
type match3Scene struct {
	nextMatch    g.Paint
	matchCount   int
	fading       []locCell
	erased       []locCell
	moving       []locCell
	matching     [][]bool
	fallSpeed    float32
	fadeDir      float32
	palette      *g.Palette
	gctx         *g.Context
	detail       int
	gr           *g.HexGrid
	cycle        int
	mode         match3Mode
	explode      bool
	splashy      *g.Particles
	toneOffset   int
	particleShim g.Affine
}

func newMatch3Scene(m match3Mode, gctx *g.Context, detail int, p *g.Palette, scale, offsetX, offsetY float32) (*match3Scene, error) {
	sc := &match3Scene{mode: m, gctx: gctx, detail: detail, palette: p}
	sc.particleShim = g.IdentityAffine()
	sc.particleShim.Translate(-offsetX, -offsetY)
	sc.particleShim.Scale(1/scale, 1/scale)
	err := sc.Reset(detail, p)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *match3Scene) Mode() Mode {
	return s.mode
}

func (s *match3Scene) Reset(detail int, p *g.Palette) error {
	_ = s.Hide()
	s.palette = p
	err := s.Display()
	if err != nil {
		return err
	}
	// and then reset the grid, and the knights, i guess?
	return nil
}

func (s *match3Scene) Display() error {
	s.gr = s.gctx.NewHexGrid(s.detail, 3, s.palette)
	s.matching = make([][]bool, len(s.gr.Cells))
	for i := range s.gr.Cells {
		s.matching[i] = make([]bool, len(s.gr.Cells[i]))
	}
	s.gr.Iterate(func(generic g.Grid, l g.ILoc, n int, c *g.Cell) {
		c.P = g.Paint(rand.Int31n(6))
		c.Scale = match3GridScale
		c.Alpha = 1.0
	})
	return nil
}

func (s *match3Scene) Hide() error {
	s.gr = nil
	return nil
}

// getMatches returns all sets of 3+ of the current match color that are in a line.
func (s *match3Scene) getMatches(update bool) int {
	matches := 0
	cells := s.gr.Cells
	for i := 0; i < len(cells); i++ {
		for j := 0; j < len(cells[i]); j++ {
			start := g.ILoc{X: i, Y: j}
			if cells[i][j].P != s.nextMatch || s.matching[i][j] {
				continue
			}
			for d := 0; d < 3; d++ {
				run := 1
				dir1 := g.HexDir(d)
				dir2 := g.HexDir(d + 3)
				for c1, l1 := s.gr.Neighbor(start, dir1, false); c1 != nil && c1.P == s.nextMatch; c1, l1 = s.gr.Neighbor(l1, dir1, false) {
					run++
				}
				for c2, l2 := s.gr.Neighbor(start, dir2, false); c2 != nil && c2.P == s.nextMatch; c2, l2 = s.gr.Neighbor(l2, dir2, false) {
					run++
				}
				if run >= 3 {
					// without update, we actually just don't care
					if !update {
						return 1
					}
					matches++
					if !s.matching[i][j] {
						s.matching[i][j] = true
						s.fading = append(s.fading, locCell{HexCell: &cells[i][j], ILoc: g.ILoc{X: i, Y: j}})
					}
					for c1, l1 := s.gr.Neighbor(start, dir1, false); c1 != nil && c1.P == s.nextMatch; c1, l1 = s.gr.Neighbor(l1, dir1, false) {
						if !s.matching[l1.X][l1.Y] {
							s.matching[l1.X][l1.Y] = true
							s.fading = append(s.fading, locCell{HexCell: c1, ILoc: g.ILoc{X: l1.X, Y: l1.Y}})

						}
					}
					for c2, l2 := s.gr.Neighbor(start, dir2, false); c2 != nil && c2.P == s.nextMatch; c2, l2 = s.gr.Neighbor(l2, dir2, false) {
						if !s.matching[l2.X][l2.Y] {
							s.matching[l2.X][l2.Y] = true
							s.fading = append(s.fading, locCell{HexCell: c2, ILoc: g.ILoc{X: l2.X, Y: l2.Y}})

						}
					}
				}
			}
		}
	}
	return matches
}

// explodeColor matches everything of the given color.
func (s *match3Scene) explodeColor(exploding g.Paint) int {
	matches := 0
	cells := s.gr.Cells
	for i := 0; i < len(cells); i++ {
		for j := 0; j < len(cells[i]); j++ {
			if cells[i][j].P != exploding {
				continue
			}
			matches++
			s.matching[i][j] = true
			s.fading = append(s.fading, locCell{HexCell: &cells[i][j], ILoc: g.ILoc{X: i, Y: j}})
		}
	}
	return matches
}

//
func (s *match3Scene) Tick(voice *sound.Voice, km keys.Map) (bool, error) {
	s.cycle = (s.cycle + 1) % s.mode.cycleTime
	if s.cycle != 0 {
		return false, nil
	}
	if s.splashy != nil {
		if s.splashy.Tick() {
			s.splashy = nil
		}
	}
	switch {
	case len(s.fading) > 0: // fade things first
		n := 0
		for _, c := range s.fading {
			c.Scale += s.fadeDir
			if s.fadeDir < 0 {
				c.Alpha = c.Scale
			}
			if c.Scale <= 0 {
				s.erased = append(s.erased, c)
				c.Alpha = 0
				c.Scale = match3GridScale
			} else {
				s.fading[n] = c
				n++
			}
		}
		if s.fading[0].Scale >= 1 {
			s.fading[0].Scale = 1
			s.fadeDir *= -1
		}
		s.fading = s.fading[:n]
		if n == 0 {
			s.gr.Status = "done fading"
			for i := range s.matching {
				for j := range s.matching[i] {
					s.matching[i][j] = false
				}
			}
		} else {
			s.gr.Status = fmt.Sprintf("fading %d hexes for %d matches, alpha %.2f", n, s.matchCount, s.fading[0].Alpha)
		}
	case s.splashy != nil:
		// just wait for the splash animation to complete.
	case len(s.erased) > 0:
		s.gr.Status = "things to move!"
		dir := g.HexDir(s.nextMatch)
		var goneCells []locCell
		// from each cell, verify that the rest of the direction it's supposed to fall in hasn't
		// got any erased cells, so we're dropping all but the furthest-down cell in each set.
	outer:
		for _, locCell := range s.erased {
			for c, l := s.gr.Neighbor(locCell.ILoc, dir, false); c != nil; c, l = s.gr.Neighbor(l, dir, false) {
				if c.Alpha == 0 {
					continue outer
				}
			}
			goneCells = append(goneCells, locCell)
		}
		s.moving = nil
		dir = dir.Right().Right().Right()
		// we now have a list of cells each of which is the furthest nextMatchwards of its row/column/something
		var addedCells []locCell
		for _, gone := range goneCells {
			skipped := 1
			for c, l := s.gr.Neighbor(gone.ILoc, dir, false); c != nil; c, l = s.gr.Neighbor(l, dir, false) {
				if c.Alpha == 0 {
					skipped++
				} else {
					// copy cell into the next open cell
					*gone.HexCell = *c
					gone.HexCell.Dir = dir
					gone.HexCell.Dist = float32(skipped)
					gone.HexCell.Alpha = 1.0
					s.moving = append(s.moving, gone)
					gone.HexCell, gone.ILoc = s.gr.Neighbor(gone.ILoc, dir, false)
				}
			}
			for gone.HexCell != nil {
				gone.HexCell.Dir = dir
				gone.HexCell.Dist = float32(skipped)
				gone.HexCell.P = g.Paint(rand.Int31n(6))
				gone.HexCell.Alpha = 1.0
				addedCells = append(addedCells, gone)
				s.moving = append(s.moving, gone)
				gone.HexCell, gone.ILoc = s.gr.Neighbor(gone.ILoc, dir, false)
			}
		}
		// check whether we'd find matches:
		s.nextMatch = (s.nextMatch + 1) % 6
		s.toneOffset = (s.toneOffset + 1) % 15
		counter := 0
		// fmt.Printf("trying to create matches for color %d after adding %d\n", s.nextMatch, len(addedCells))
		for s.getMatches(false) == 0 {
			counter++
			for _, c := range addedCells {
				c.P = g.Paint(rand.Int31n(6))
			}
			// give up for now.
			if counter > 1000 {
				s.explode = true
				break
			}
		}
		s.gr.Status = fmt.Sprintf("had %d gone cells, now have %d falling", len(goneCells), len(s.moving))
		s.erased = nil
		s.fallSpeed = 0
		// we start falling the same frame
		fallthrough
	case len(s.moving) > 0:
		if s.fallSpeed < 0.5 {
			s.fallSpeed += float32(1) / 32
		}
		//s.gr.Status = fmt.Sprintf("%d cells falling, first cell distance %.2f", len(s.moving))
		n := 0
		for i := range s.moving {
			c := s.moving[i]
			c.Dist -= s.fallSpeed
			if c.Dist > 0 {
				s.moving[n] = c
				n++
			} else {
				c.Dist = 0
			}
		}
		s.moving = s.moving[:n]
		if n == 0 {
			s.gr.Status = "done moving"
		} else {
			s.gr.Status = fmt.Sprintf("moving %d hexes, dist %.2f", n, s.moving[0].Dist)
		}
	default:
		s.matchCount = s.getMatches(true)
		if s.matchCount > 0 {
			s.gr.Status = fmt.Sprintf("found %d matches", s.matchCount)
			s.fadeDir = float32(1) / 32
			for i := 0; i < s.matchCount && i < 3; i++ {
				voice.Play((int(s.nextMatch)+s.toneOffset+(i*3))%15, 75-(20*i))
			}
		} else {
			if s.explode {
				s.explodeColor(s.nextMatch)
				s.explode = false
				voice.Play((int(s.nextMatch)+s.toneOffset)%15, 75)
			} else {
				s.gr.Status = fmt.Sprintf("found no matches")
				s.nextMatch = (s.nextMatch + 1) % 6
			}
		}
		s.splashy = s.gctx.NewParticles(s.gr.Width*4, 1, s.palette)
		for _, c := range s.fading {
			x0, y0 := s.particleShim.Project(s.gr.CenterFor(c.ILoc.Y, c.ILoc.X))
			// add a particle animation for each c
			for i := 0; i < 5; i++ {
				p := s.splashy.Add(g.SecondSplasher, c.P, x0, y0)
				p.Alpha = 0
				p.Scale = rand.Float32()/2 + 0.5
				p.Delay = int(rand.Int31n(6))
				p.X = (rand.Float32() - 0.5) / 2
				p.Y = (rand.Float32() - 0.5) / 2
				p.DX = (rand.Float32() - 0.5) / 4
				p.DY = (rand.Float32() - 0.5) / 4
			}
			for i := 0; i < 3; i++ {
				p := s.splashy.Add(g.SecondSplasher, c.P+g.Paint(rand.Int31n(5))+1, x0, y0)
				p.Alpha = 0
				p.Scale = rand.Float32()/2 + 0.5
				p.Delay = int(rand.Int31n(6))
				p.X = (rand.Float32() - 0.5) / 2
				p.Y = (rand.Float32() - 0.5) / 2
				p.DX = (rand.Float32() - 0.5) / 4
				p.DY = (rand.Float32() - 0.5) / 4
			}
		}
	}
	return true, nil
}

func (s *match3Scene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.gr.Draw(t, scale)
		if s.splashy != nil {
			s.splashy.Draw(t, scale)
		}

	})
	return nil
}
