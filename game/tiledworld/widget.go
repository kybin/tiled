package main

import (
	"image"
)

type Widget struct {
	Pin    WidgetPin
	Offset image.Point
	Size   image.Point
	// If Block returns true, the Widget and the chlidren should stop operate.
	// If Block is nil, the Widget will always operate (if one of the ancestors didn't block).
	Block func(g *Game) bool
	// Update is a given function to the Widget,
	// so it can handle inputs and update the game status.
	// If Update is nil, the Widget will not update the game.
	Update func(g *Game, bounds image.Rectangle) error
	// Draw is a given function to the Widget,
	// so it can draw on the game screen.
	// If Draw is nil, the Widget will not draw on the screen.
	Draw func(g *Game, bounds image.Rectangle)
	// Children is child Widgets of this Widget.
	Children []*Widget
}

func (w *Widget) UpdateRecursive(g *Game, bounds image.Rectangle) error {
	if w.Block != nil && w.Block(g) {
		// don't evaluate the branch
		return nil
	}
	bounds = calcBounds(bounds, w.Pin, w.Offset, w.Size)
	if w.Update != nil {
		err := w.Update(g, bounds)
		if err != nil {
			return err
		}
	}
	for _, c := range w.Children {
		err := c.UpdateRecursive(g, bounds)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Widget) DrawRecursive(g *Game, bounds image.Rectangle) {
	if w.Block != nil && w.Block(g) {
		// don't evaluate the branch
		return
	}
	bounds = calcBounds(bounds, w.Pin, w.Offset, w.Size)
	if w.Draw != nil {
		w.Draw(g, bounds)
	}
	for _, c := range w.Children {
		c.DrawRecursive(g, bounds)
	}
}

func calcBounds(parentBounds image.Rectangle, pin WidgetPin, off, size image.Point) image.Rectangle {
	if size.X == 0 {
		size.X = parentBounds.Size().X
	}
	if size.Y == 0 {
		size.Y = parentBounds.Size().Y
	}
	toEnd := image.Point{
		(parentBounds.Size().X - size.X),
		(parentBounds.Size().Y - size.Y),
	}
	toCenter := toEnd.Div(2)
	b := image.Rect(0, 0, size.X, size.Y)
	dir := pinDirection(pin)
	switch dir.X {
	case 0:
		b = b.Add(image.Pt(toCenter.X, 0))
	case 1:
		b = b.Add(image.Pt(toEnd.X, 0))
	}
	switch dir.Y {
	case 0:
		b = b.Add(image.Pt(0, toCenter.Y))
	case 1:
		b = b.Add(image.Pt(0, toEnd.Y))
	}
	b = b.Add(off)
	return parentBounds.Intersect(b)
}

type WidgetPin int

const (
	WidgetPinTopLeft = WidgetPin(iota)
	WidgetPinTop
	WidgetPinTopRight
	WidgetPinLeft
	WidgetPinCenter
	WidgetPinRight
	WidgetPinBottomLeft
	WidgetPinBottom
	WidgetPinBottomRight
)

func pinDirection(p WidgetPin) image.Point {
	switch p {
	case WidgetPinTopLeft:
		return image.Pt(-1, -1)
	case WidgetPinTop:
		return image.Pt(0, -1)
	case WidgetPinTopRight:
		return image.Pt(1, -1)
	case WidgetPinLeft:
		return image.Pt(-1, 0)
	case WidgetPinCenter:
		return image.Pt(0, 0)
	case WidgetPinRight:
		return image.Pt(1, 0)
	case WidgetPinBottomLeft:
		return image.Pt(-1, 1)
	case WidgetPinBottom:
		return image.Pt(0, 1)
	case WidgetPinBottomRight:
		return image.Pt(1, 1)
	}
	return image.Pt(0, 0)
}
