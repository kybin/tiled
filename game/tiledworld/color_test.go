package main

import (
	"image/color"
	"math"
	"testing"
)

func TestHSLToRGB(t *testing.T) {
	cases := []struct {
		h, s, l float64
		want    color.RGBA
	}{
		{
			h:    0,
			s:    1,
			l:    0.5,
			want: color.RGBA{R: 255, A: 255},
		},
		{
			h:    float64(1) / float64(6),
			s:    1,
			l:    0.5,
			want: color.RGBA{R: 255, G: 255, A: 255},
		},
		{
			h:    float64(2) / float64(6),
			s:    1,
			l:    0.5,
			want: color.RGBA{G: 255, A: 255},
		},
		{
			h:    float64(3) / float64(6),
			s:    1,
			l:    0.5,
			want: color.RGBA{G: 255, B: 255, A: 255},
		},
		{
			h:    float64(4) / float64(6),
			s:    1,
			l:    0.5,
			want: color.RGBA{B: 255, A: 255},
		},
		{
			h:    float64(5) / float64(6),
			s:    1,
			l:    0.5,
			want: color.RGBA{R: 255, B: 255, A: 255},
		},
		{
			h:    1,
			s:    1,
			l:    0.5,
			want: color.RGBA{R: 255, A: 255},
		},
	}
	for _, c := range cases {
		got := HSLToRGB(c.h, c.s, c.l)
		if got != c.want {
			t.Fatalf("HSLToRGBA(%v, %v, %v) got %v, want %v", c.h, c.s, c.l, got, c.want)
		}
	}
	for _, c := range cases {
		h, s, l := RGBToHSL(c.want)
		if h != math.Mod(c.h, 1) || s != c.s || l != c.l {
			t.Logf("RGBToHSL(%v) got (%v, %v, %v), want (%v, %v, %v)", c.want, h, s, l, c.h, c.s, c.l)
		}
	}
}
