package main

import (
	"errors"
	"image"
)

var UpdateHandled error = errors.New("update handled")

type Widget struct {
	Pin    WidgetPin
	Offset image.Point
	Size   image.Point
	// Bounds will be cacluated from Pin, Offset and Size.
	Bounds image.Rectangle
	Focus  bool
	// If Block returns true, the Widget and the chlidren should stop operate.
	// If Block is nil, the Widget will always operate (if one of the ancestors didn't block).
	Block func(g *Game) bool
	// Tick updates the status without any input.
	// It will be run even if the Widget has blocked.
	// World stop can prevent it to be run.
	Tick func(g *Game, w *Widget)
	// Update is a given function to the Widget,
	// so it can handle inputs and update the game status.
	// If Update is nil, the Widget will not update the game.
	Update func(g *Game, w *Widget) error
	// Draw is a given function to the Widget,
	// so it can draw on the game screen.
	// If Draw is nil, the Widget will not draw on the screen.
	Draw func(g *Game, w *Widget)
	// Parent is a parent Widget of this Widget.
	Parent *Widget
	// Children is child Widgets of this Widget.
	Children []*Widget
}

// Build builds Widget hierarchy.
// Some operations will not be run correctly before it.
func (w *Widget) Build(parent *Widget) {
	w.Parent = parent
	for _, c := range w.Children {
		c.Build(w)
	}
}

// Root returns Root Widget of the tree Widget w is belong to.
func (w *Widget) Root() *Widget {
	wg := w
	for true {
		if wg.Parent == nil {
			return wg
			break
		}
		wg = wg.Parent
	}
	return nil
}

// SetFocus set focus on the Widget and it's ancesters.
// Focus on all other widgets will be dismissed.
func (w *Widget) SetFocus() {
	// dismiss all focus
	w.Root().FuncRecursive(func(w *Widget) {
		w.Focus = false
	})
	wg := w
	// set focus
	for true {
		wg.Focus = true
		if wg.Parent == nil {
			break
		}
		wg = wg.Parent
	}
}

// FuncRecusive recusively run func f on the Widget and it's Children.
func (w *Widget) FuncRecursive(f func(w *Widget)) {
	if f == nil {
		return
	}
	f(w)
	for _, c := range w.Children {
		c.FuncRecursive(f)
	}
}

func (w *Widget) UpdateRecursive(g *Game) error {
	if w.Block != nil && w.Block(g) {
		// don't evaluate the branch
		return nil
	}
	bounds := g.Bounds
	if w.Parent != nil {
		bounds = w.Parent.Bounds
	}
	w.Bounds = calcBounds(bounds, w.Pin, w.Offset, w.Size)
	if w.Update != nil {
		err := w.Update(g, w)
		if err != nil {
			return err
		}
	}
	for _, c := range w.Children {
		err := c.UpdateRecursive(g)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Widget) TickRecursive(g *Game) {
	if g.worldStopped {
		return
	}
	if w.Tick != nil {
		w.Tick(g, w)
	}
	for _, c := range w.Children {
		c.TickRecursive(g)
	}
}

func (w *Widget) DrawRecursive(g *Game) {
	if w.Block != nil && w.Block(g) {
		// don't evaluate the branch
		return
	}
	if w.Draw != nil {
		w.Draw(g, w)
	}
	for _, c := range w.Children {
		c.DrawRecursive(g)
	}
}

func calcBounds(parentBounds image.Rectangle, pin WidgetPin, off, size image.Point) image.Rectangle {
	if size.X <= 0 {
		size.X = max(parentBounds.Size().X+size.X, 0)
	}
	if size.Y <= 0 {
		size.Y = max(parentBounds.Size().Y+size.Y, 0)
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
