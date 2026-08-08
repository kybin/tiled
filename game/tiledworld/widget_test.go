package main

import (
	"image"
	"testing"
)

func TestCalcBounds(t *testing.T) {
	cases := []struct {
		parentBounds image.Rectangle
		pin          WidgetPin
		off          image.Point
		size         image.Point
		want         image.Rectangle
	}{
		{
			parentBounds: image.Rect(0, 0, 640, 480),
			pin:          WidgetPinTopLeft,
			size:         image.Pt(-100, -100),
			want:         image.Rect(0, 0, 540, 380),
		},
		{
			parentBounds: image.Rect(0, 0, 640, 480),
			pin:          WidgetPinCenter,
			size:         image.Pt(-100, -100),
			want:         image.Rect(50, 50, 590, 430),
		},
		{
			parentBounds: image.Rect(0, 0, 640, 480),
			pin:          WidgetPinBottomRight,
			size:         image.Pt(-100, -100),
			want:         image.Rect(100, 100, 640, 480),
		},
	}
	for _, c := range cases {
		got := calcBounds(c.parentBounds, c.pin, c.off, c.size)
		if got != c.want {
			t.Fatalf("unexpected result of calcBounds(%v, %v, %v, %v): got %v, want %v)", c.parentBounds, c.pin, c.off, c.size, got, c.want)
		}
	}
}
