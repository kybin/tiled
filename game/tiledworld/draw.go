package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

func drawOutline(img *ebiten.Image, b image.Rectangle, c color.Color) {
	o := b.Min
	s := b.Size()
	outline := ebiten.NewImage(1, 1)
	outline.Set(0, 0, c)
	// vertical lines
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1, float64(s.Y))
	op.GeoM.Translate(float64(o.X), float64(o.Y))
	img.DrawImage(outline, op)
	op.GeoM.Translate(float64(s.X)-1, 0)
	img.DrawImage(outline, op)
	// horizontal lines
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(s.X)-2, 1)
	op.GeoM.Translate(float64(o.X)+1, float64(o.Y))
	img.DrawImage(outline, op)
	op.GeoM.Translate(0, float64(s.Y)-1)
	img.DrawImage(outline, op)
}
