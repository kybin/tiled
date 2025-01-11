package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/draw"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

type Widget interface {
	SetRect(image.Rectangle)
	HandleEvent(any)
	Draw(screen.Window, screen.Screen)
}

type Area struct {
	Name   string
	Rect   image.Rectangle
	Split  *AreaSplit
	Widget Widget
}

func resize(a *Area, rect image.Rectangle) {
	a.Rect = rect
	if a.Widget != nil {
		a.Widget.SetRect(rect)
	}
	if a.Split == nil {
		return
	}
	s := a.Split
	switch a.Split.Dir {
	case SplitLeft:
		x := min(rect.Min.X+s.Dist, rect.Max.X)
		resize(s.A, image.Rect(rect.Min.X, rect.Min.Y, x, rect.Max.Y))
		resize(s.B, image.Rect(x, rect.Min.Y, rect.Max.X, rect.Max.Y))
	case SplitRight:
		x := max(rect.Min.X, rect.Max.X-s.Dist)
		resize(s.A, image.Rect(rect.Min.X, rect.Min.Y, x, rect.Max.Y))
		resize(s.B, image.Rect(x, rect.Min.Y, rect.Max.X, rect.Max.Y))
	case SplitTop:
		y := min(rect.Min.Y+s.Dist, rect.Max.Y)
		resize(s.A, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, y))
		resize(s.B, image.Rect(rect.Min.X, y, rect.Max.X, rect.Max.Y))
	case SplitBottom:
		y := max(rect.Min.Y, rect.Max.Y-s.Dist)
		resize(s.A, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, y))
		resize(s.B, image.Rect(rect.Min.X, y, rect.Max.X, rect.Max.Y))
	}
}

func named(a *Area, m map[string]*Area) {
	if a.Name != "" {
		m[a.Name] = a
	}
	if a.Split == nil {
		return
	}
	if a.Split.A != nil {
		named(a.Split.A, m)
	}
	if a.Split.B != nil {
		named(a.Split.B, m)
	}
}

func inside(r image.Rectangle, pt image.Point) bool {
	if pt.X < r.Min.X || pt.X > r.Max.X {
		return false
	}
	if pt.Y < r.Min.Y || pt.Y > r.Max.Y {
		return false
	}
	return true
}

func WidgetAt(a *Area, pt image.Point) Widget {
	if a == nil {
		return nil
	}
	if !inside(a.Rect, pt) {
		return nil
	}
	// area rect contains pt
	if a.Split == nil {
		return a.Widget
	}
	if w := WidgetAt(a.Split.A, pt); w != nil {
		return w
	}
	if w := WidgetAt(a.Split.B, pt); w != nil {
		return w
	}
	return a.Widget
}

type AreaSplit struct {
	Dir  SplitDir
	Dist int
	A    *Area // Top or Left according to SplitDir
	B    *Area // Bottom or Right according to SplitDir
}

type SplitDir int

const (
	SplitLeft = SplitDir(iota)
	SplitRight
	SplitTop
	SplitBottom
)

func drawColor(w screen.Window, s screen.Screen, rect image.Rectangle, c color.Color) {
	size := rect.Max.Sub(rect.Min)
	tex, err := s.NewTexture(size)
	if err != nil {
		log.Println(err)
		return
	}
	buf, err := s.NewBuffer(size)
	if err != nil {
		tex.Release()
		log.Println(err)
		return
	}
	img := image.NewUniform(c)
	draw.Copy(buf.RGBA(), image.Point{}, img, buf.Bounds(), draw.Src, nil)
	tex.Upload(image.Point{}, buf, tex.Bounds())
	buf.Release()
	w.Copy(rect.Min, tex, tex.Bounds(), screen.Src, nil)
}

type board struct {
	rect  image.Rectangle
	dirty bool
	size  image.Point
	bg    color.Color
}

func (b *board) SetRect(r image.Rectangle) {
	if b.rect == r {
		return
	}
	b.rect = r
	b.dirty = true
}

func (b *board) HandleEvent(e any) {
	fmt.Println(e)
}

func (b *board) Draw(w screen.Window, s screen.Screen) {
	if !b.dirty {
		return
	}
	drawColor(w, s, b.rect, b.bg)
	b.dirty = false
}

type position struct {
	X float32
	Y float32
}

func pos(x, y float32) position {
	return position{X: x, Y: y}
}

func main() {
	area := &Area{
		Name: "bg",
		Split: &AreaSplit{
			Dir:  SplitLeft,
			Dist: 200, // px
			A:    &Area{Name: "side"},
			B:    &Area{Name: "playground"},
		},
	}
	namedArea := make(map[string]*Area)
	named(area, namedArea)

	playground := &board{dirty: true, bg: color.NRGBA{255, 255, 0, 255}}
	namedArea["playground"].Widget = playground

	side := &board{dirty: true, bg: color.NRGBA{0, 255, 255, 255}}
	namedArea["side"].Widget = side

	widgets := make([]Widget, 0)
	for _, a := range namedArea {
		if a.Widget != nil {
			widgets = append(widgets, a.Widget)
		}
	}
	winSize := image.Pt(400, 300)
	resize(area, image.Rect(0, 0, winSize.X, winSize.Y))
	mousePos := image.Pt(-1, -1)
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(&screen.NewWindowOptions{
			Title:  "Board GUI",
			Width:  winSize.X,
			Height: winSize.Y,
		})
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()
		paintPending := true
		w.Send(paint.Event{})
		for {
			switch e := w.NextEvent().(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
			case key.Event:
				if e.Code == key.CodeEscape {
					return
				}
				wg := WidgetAt(area, mousePos)
				if wg != nil {
					wg.HandleEvent(e)
				}
				if !paintPending {
					paintPending = true
					w.Send(paint.Event{})
				}
			case paint.Event:
				resize(area, image.Rect(0, 0, winSize.X, winSize.Y))
				for _, wg := range widgets {
					wg.Draw(w, s)
				}
				w.Publish()
				paintPending = false
			case mouse.Event:
				mousePos = image.Pt(int(e.X), int(e.Y))
			case size.Event:
				winSize = image.Pt(e.WidthPx, e.HeightPx)
				resize(area, image.Rect(0, 0, winSize.X, winSize.Y))
				if !paintPending {
					paintPending = true
					w.Send(paint.Event{})
				}
			case error:
				log.Print(e)
			}
		}
	})
}
