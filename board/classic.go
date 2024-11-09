package grid

import (
	"github.com/kybin/tiled"
)

func ClassicBoard(width, height int) *tiled.Board {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	board := &tiled.Board{
		Width:   width,
		Height:  Height,
		Tiles:   make([]*tiled.Tile, width*height),
		AllWays: []string{"N", "W", "S", "E"},
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			t := g.Tiles[y*width+x]
			t.Pos = tiled.Pos{x, y}
			t.ways = make([]*tiled.Way, 0)
			if x != 0 {
				t.Ways = append(t.Ways, &Way{Name: "W", From: t, To: tiles[idx-1], Cost: 1})
			}
			if x != width-1 {
				t.Ways = append(t.Ways, &Way{Name: "E", From: t, To: tiles[idx+1], Cost: 1})
			}
			if y != 0 {
				t.Ways = append(t.Ways, &Way{Name: "S", From: t, To: tiles[idx-width], Cost: 1})
			}
			if y != height-1 {
				t.Ways = append(t.Ways, &Way{Name: "N", From: t, To: tiles[idx+width], Cost: 1})
			}
		}
	}
}