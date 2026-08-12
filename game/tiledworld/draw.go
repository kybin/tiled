package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

func drawOutline(img *ebiten.Image, b image.Rectangle, width float64, c color.Color) {
	o := b.Min
	s := b.Size()
	outline := ebiten.NewImage(1, 1)
	outline.Set(0, 0, c)
	// vertical lines
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(width, float64(s.Y))
	op.GeoM.Translate(float64(o.X), float64(o.Y))
	img.DrawImage(outline, op)
	op.GeoM.Translate(float64(s.X)-width, 0)
	img.DrawImage(outline, op)
	// horizontal lines
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(s.X)-width*2, width)
	op.GeoM.Translate(float64(o.X)+width, float64(o.Y))
	img.DrawImage(outline, op)
	op.GeoM.Translate(0, float64(s.Y)-width)
	img.DrawImage(outline, op)
}
