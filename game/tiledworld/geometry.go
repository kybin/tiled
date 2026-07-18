package main

import "image"

type point struct {
	X, Y float64
}

func pt(x, y float64) point {
	return point{X: x, Y: y}
}

func (p point) Add(q point) point {
	return point{p.X + q.X, p.Y + q.Y}
}

func (p point) Mul(a float64) point {
	return point{p.X * a, p.Y * a}
}

func (p point) In(r rectangle) bool {
	if p.X < r.Min.X || p.X >= r.Max.X {
		return false
	}
	if p.Y < r.Min.Y || p.Y >= r.Max.Y {
		return false
	}
	return true
}

type rectangle struct {
	Min, Max point
}

func rect(xmin, ymin, xmax, ymax float64) rectangle {
	return rectangle{
		Min: point{X: xmin, Y: ymin},
		Max: point{X: xmax, Y: ymax},
	}
}

func (r rectangle) Add(q point) rectangle {
	return rect(r.Min.X+q.X, r.Min.Y+q.Y, r.Max.X+q.X, r.Max.Y+q.Y)
}

func (r rectangle) Scale(s float64) rectangle {
	return rect(r.Min.X*s, r.Min.Y*s, r.Max.X*s, r.Max.Y*s)
}

func (r rectangle) Inset(a float64) rectangle {
	if r.Max.X-r.Min.X < a {
		r.Min.X = (r.Min.X + r.Max.X) / 2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += a
		r.Max.X -= a
	}
	if r.Max.Y-r.Min.Y < a {
		r.Min.Y = (r.Min.Y + r.Max.Y) / 2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += a
		r.Max.Y -= a
	}
	return r
}

func (r rectangle) ImageRectangle() image.Rectangle {
	return image.Rect(int(r.Min.X), int(r.Min.Y), int(r.Max.X), int(r.Max.Y))
}
