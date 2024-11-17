package hex

import (
	"github.com/kybin/tiled"
)

// TODO: write tests

// NewBoard creates a new hex board.
func NewBoard(width, height int) *tiled.Board {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	// note that dirs are not having same distance.
	// it makes somewhat squeezed board, but we have integer positions instead.
	dirs := make(map[string]tiled.Pos)
	dirs["N"] = tiled.Pos{0, -2}
	dirs["NW"] = tiled.Pos{1, -1}
	dirs["SW"] = tiled.Pos{1, 1}
	dirs["S"] = tiled.Pos{0, 2}
	dirs["SE"] = tiled.Pos{-1, 1}
	dirs["NE"] = tiled.Pos{-1, -1}
	tileAt := make(map[tiled.Pos]*tiled.Tile)
	for y := 0; y <= height+1; y++ {
		for x := 0; x <= width+2; x += 2 {
			if x-2 <= width && y-1 <= height {
				tileAt[tiled.Pos{x, y}] = &tiled.Tile{}
			}
		}
	}
	board := &tiled.Board{
		Width:  width,
		Height: Height,
		TileAt: tileAt,
		Ways:   []string{"N", "NW", "SW", "S", "SE", "NE"},
	}
	// find ways to nearby tiles
	for y := 0; y <= height+1; y++ {
		for x := 0; x <= width+2; x += 2 {
			pos := tiled.Pos{x, y}
			t := g.TileAt[pos]
			t.ways = make([]*tiled.Way, 0)
			for name, dir := range dirs {
				at := pos.Add(dir)
				if g.TileAt[at] != nil {
					t.Ways = append(t.Ways, &Way{Name: name, From: t, To: g.TileAt[at], Cost: 1})
				}
			}
		}
	}
}

var AroundArea = tiled.CreateArea([]tiled.Pos{{0, -2}, {1, -1}, {1, 1}, {0, 2}, {-1, 1}, {-1, -1}})
