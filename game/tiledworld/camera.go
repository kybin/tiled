package main

import "image"

type Camera struct {
	Origin       image.Point // top-left corner position
	Size         image.Point
	FollowMargin int
	Bounds       *image.Rectangle
}

func NewCamera(origin, size image.Point) *Camera {
	c := &Camera{
		Origin: origin,
	}
	c.SetSize(size)
	return c
}

func (c *Camera) SetSize(s image.Point) {
	if s.X < 1 {
		s.X = 1
	}
	if s.Y < 1 {
		s.Y = 1
	}
	c.Size = s
}

func (c *Camera) Rect() image.Rectangle {
	End := c.Origin.Add(c.Size)
	return image.Rect(c.Origin.X, c.Origin.Y, End.X, End.Y)
}

func (c *Camera) Follow(p image.Point) {
	ir := c.Rect().Inset(c.FollowMargin) // inner rect
	if p.In(ir) {
		return
	}
	tr := image.Point{}
	if p.X < ir.Min.X {
		tr.X = p.X - ir.Min.X
	} else if p.X > ir.Max.X {
		tr.X = p.X - ir.Max.X
	}
	if p.Y < ir.Min.Y {
		tr.Y = p.Y - ir.Min.Y
	} else if p.Y > ir.Max.Y {
		tr.Y = p.Y - ir.Max.Y
	}
	c.Origin = c.Origin.Add(tr)
	// but don't go outside of camera bounds
	r := c.Rect()
	b := c.Bounds
	if b == nil {
		return
	}
	tr = image.Point{}
	if r.Min.X < b.Min.X {
		tr.X = b.Min.X - r.Min.X
	} else if r.Max.X > b.Max.X {
		tr.X = b.Max.X - r.Max.X
	}
	if r.Min.Y < b.Min.Y {
		tr.Y = b.Min.Y - r.Min.Y
	} else if r.Max.Y > b.Max.Y {
		tr.Y = b.Max.Y - r.Max.Y
	}
	c.Origin = c.Origin.Add(tr)
}
