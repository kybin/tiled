package main

import (
	"image/color"
	"math"
)

func HSLToRGB(h, s, l float64) color.RGBA {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h*6, 2)-1))
	m := l - c/2
	var rr, gg, bb float64
	switch int(h * 6) {
	case 0:
		rr, gg, bb = c, x, 0
	case 1:
		rr, gg, bb = x, c, 0
	case 2:
		rr, gg, bb = 0, c, x
	case 3:
		rr, gg, bb = 0, x, c
	case 4:
		rr, gg, bb = x, 0, c
	case 5:
		rr, gg, bb = c, 0, x
	default:
		rr, gg, bb = c, 0, x
	}
	r := uint8((rr + m) * 255)
	g := uint8((gg + m) * 255)
	b := uint8((bb + m) * 255)
	return color.RGBA{r, g, b, 255}
}
