package main

type Camera struct {
	Origin       point // top-left corner position
	Size         point
	FollowMargin float32
	// Bounds defines boundary that camera can navigate around.
	// eg. world bounds
	Bounds *rectangle
}

func NewCamera(origin, size point) *Camera {
	c := &Camera{
		Origin: origin,
	}
	c.SetSize(size)
	return c
}

func (c *Camera) SetSize(s point) {
	if s.X < 1 {
		s.X = 1
	}
	if s.Y < 1 {
		s.Y = 1
	}
	c.Size = s
}

func (c *Camera) Rect() rectangle {
	End := c.Origin.Add(c.Size)
	return rect(c.Origin.X, c.Origin.Y, End.X, End.Y)
}

func (c *Camera) Follow(p point) {
	ir := c.Rect().Inset(c.FollowMargin) // inner rect
	if p.In(ir) {
		return
	}
	tr := point{}
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
	tr = point{}
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
