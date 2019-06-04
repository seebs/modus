package modes

import (
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"seebs.net/modus/g"
	"seebs.net/modus/keys"
	"seebs.net/modus/sound"
)

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
		allModes = append(allModes, mode)
	}
}

func (m match3Mode) Name() string {
	return "match3"
}

func (m match3Mode) Description() string {
	return "match3 thing"
}

func (m match3Mode) New(gctx *g.Context, detail int, p *g.Palette) (Scene, error) {
	return newMatch3Scene(m, gctx, detail, p)
}

type locCell struct {
	g.ILoc
	*g.HexCell
}
type match3Scene struct {
	nextMatch g.Paint
	fading    []locCell
	erased    []locCell
	moving    []locCell
	matching  [][]bool
	fallSpeed float32
	fadeDir   float32
	palette   *g.Palette
	gctx      *g.Context
	detail    int
	gr        *g.HexGrid
	cycle     int
	mode      match3Mode
}

func newMatch3Scene(m match3Mode, gctx *g.Context, detail int, p *g.Palette) (*match3Scene, error) {
	sc := &match3Scene{mode: m, gctx: gctx, detail: detail, palette: p}
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
	s.gr = s.gctx.NewHexGrid(s.detail, 1, s.palette)
	s.matching = make([][]bool, len(s.gr.Cells))
	for i := range s.gr.Cells {
		s.matching[i] = make([]bool, len(s.gr.Cells[i]))
	}
	s.gr.Iterate(func(generic g.Grid, l g.ILoc, n int, c *g.Cell) {
		c.P = g.Paint(rand.Int31n(6))
		c.Alpha = 0.75
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
				loc1 := start
				dir1 := g.HexDir(d)
				loc2 := start
				dir2 := g.HexDir(d + 3)
				for c1, l1 := s.gr.Neighbor(loc1, dir1, false); c1 != nil && c1.P == s.nextMatch; c1, l1 = s.gr.Neighbor(loc1, dir1, false) {
					run++
					loc1 = l1
				}
				for c2, l2 := s.gr.Neighbor(loc2, dir2, false); c2 != nil && c2.P == s.nextMatch; c2, l2 = s.gr.Neighbor(loc2, dir2, false) {
					run++
					loc2 = l2
				}
				if run >= 3 {
					matches++
					if update && !s.matching[i][j] {
						s.matching[i][j] = true
						s.fading = append(s.fading, locCell{HexCell: &cells[i][j], ILoc: g.ILoc{X: i, Y: j}})
					}
					for c1, l1 := s.gr.Neighbor(loc1, dir1, false); c1 != nil && c1.P == s.nextMatch; c1, l1 = s.gr.Neighbor(loc1, dir1, false) {
						if update && !s.matching[loc1.X][loc1.Y] {
							s.matching[loc1.X][loc1.Y] = true
							s.fading = append(s.fading, locCell{HexCell: c1, ILoc: g.ILoc{X: loc1.X, Y: loc1.Y}})

						}
						loc1 = l1
					}
					for c2, l2 := s.gr.Neighbor(loc2, dir2, false); c2 != nil && c2.P == s.nextMatch; c2, l2 = s.gr.Neighbor(loc2, dir2, false) {
						if update && !s.matching[loc2.X][loc2.Y] {
							s.matching[loc2.X][loc2.Y] = true
							s.fading = append(s.fading, locCell{HexCell: c2, ILoc: g.ILoc{X: loc2.X, Y: loc2.Y}})

						}
						loc2 = l2
					}
				}
			}
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
	switch {
	case len(s.fading) > 0: // fade things first
		n := 0
		for _, c := range s.fading {
			c.Alpha += s.fadeDir
			if c.Alpha <= 0 {
				s.erased = append(s.erased, c)
				c.Alpha = 0
			} else {
				s.fading[n] = c
				n++
			}
		}
		if s.fading[0].Alpha >= 1 {
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
			s.gr.Status = fmt.Sprintf("fading %d hexes, alpha %.2f", n, s.fading[0].Alpha)
		}
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
					gone.HexCell.Alpha = 0.75
					s.moving = append(s.moving, gone)
					gone.HexCell, gone.ILoc = s.gr.Neighbor(gone.ILoc, dir, false)
				}
			}
			for gone.HexCell != nil {
				gone.HexCell.Dir = dir
				gone.HexCell.Dist = float32(skipped)
				gone.HexCell.P = g.Paint(rand.Int31n(6))
				gone.HexCell.Alpha = 0.75
				addedCells = append(addedCells, gone)
				s.moving = append(s.moving, gone)
				gone.HexCell, gone.ILoc = s.gr.Neighbor(gone.ILoc, dir, false)
			}
		}
		// check whether we'd find matches:
		s.nextMatch = (s.nextMatch + 1) % 6
		counter := 0
		// fmt.Printf("trying to create matches for color %d after adding %d\n", s.nextMatch, len(addedCells))
		for s.getMatches(false) == 0 {
			counter++
			for _, c := range addedCells {
				c.P = g.Paint(rand.Int31n(6))
			}
			// give up for now
			if counter > 1000 {
				fmt.Printf("couldn't create match. cells:\n")
				for _, c := range addedCells {
					fmt.Printf(" %d,%d\n", c.ILoc.X, c.ILoc.Y)
				}
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
		n := s.getMatches(true)
		if n > 0 {
			s.gr.Status = fmt.Sprintf("found %d matches", n)
			s.fadeDir = float32(1) / 32
		} else {
			s.gr.Status = fmt.Sprintf("found no matches")
			s.nextMatch = (s.nextMatch + 1) % 6
		}
	}
	return true, nil
}

func (s *match3Scene) Draw(screen *ebiten.Image) error {
	s.gctx.Render(screen, func(t *ebiten.Image, scale float32) {
		s.gr.Draw(t, scale)
	})
	return nil
}
