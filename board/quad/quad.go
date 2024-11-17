package quad

import (
	"github.com/kybin/tiled"
)

// NewBoard creates a new quad board.
func NewBoard(width, height int) *tiled.Board {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	dirs := make(map[string]tiled.Pos)
	dirs["N"] = tiled.Pos{0, -1}
	dirs["W"] = tiled.Pos{1, 0}
	dirs["S"] = tiled.Pos{0, 1}
	dirs["E"] = tiled.Pos{-1, 0}
	board := &tiled.Board{
		Width:  width,
		Height: Height,
		Tiles:  make([]*tiled.Tile, width*height),
		Ways:   []string{"N", "W", "S", "E"},
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pos := tiled.Pos{x, y}
			t := board.TileAt[pos]
			t.Ways = make([]*tiled.Way, 0)
			for name, dir := range dirs {
				at := pos.Add(dir)
				if board.TileAt[at] != nil {
					t.Ways = append(t.Ways, &Way{Name: name, From: t, To: board.TileAt[at], Cost: 1})
				}
			}
		}
	}
}

var AxisArea = tiled.CreateArea([]tiled.Pos{{0, 1}, {1, 0}, {0, -1}, {-1, 0}})

var Axis2Area = tiled.CreateArea([]tiled.Pos{{0, 2}, {2, 0}, {0, -2}, {-2, 0}})

var DiagArea = tiled.CreateArea([]tiled.Pos{{1, 1}, {1, -1}, {-1, -1}, {-1, 1}})

var Diag2Area = tiled.CreateArea([]tiled.Pos{{2, 2}, {2, -2}, {-2, -2}, {-2, 2}})

var AroundArea = tiled.CreateArea([]tiled.Pos{{0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}})

var Around2Area = tiled.CreateArea([]tiled.Pos{
	{0, 2}, {1, 2}, {2, 2}, {2, 1}, {2, 0}, {2, -1}, {2, -2}, {1, -2},
	{0, -2}, {-1, -2}, {-2, -2}, {-2, -1}, {-2, 0}, {-2, 1}, {-2, 2}, {-1, 2},
})
