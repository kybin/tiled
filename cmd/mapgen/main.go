package main

import (
	"image"
	"image/color"
	"log"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/image/draw"
)

func stretch(n node.Node, alongWeight int) node.Node {
	return widget.WithLayoutData(n, widget.FlowLayoutData{
		AlongWeight:  alongWeight,
		ExpandAlong:  true,
		ShrinkAlong:  true,
		ExpandAcross: true,
		ShrinkAcross: true,
	})
}

// Uniform is a shell widget that paints a uniform color, analogous to an
// image.Uniform.
type Uniform struct {
	node.ShellEmbed
	Color color.Color
}

// NewUniform returns a new Uniform widget of the given color.
func NewUniform(c color.Color, inner node.Node) *Uniform {
	w := &Uniform{
		Color: c,
	}
	w.Wrapper = w
	if inner != nil {
		w.Insert(inner, nil)
	}
	return w
}

func (w *Uniform) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	r := w.Rect.Add(origin)
	img := image.NewUniform(w.Color)
	draw.Draw(ctx.Dst, r, img, image.Point{}, draw.Src)
	if c := w.FirstChild; c != nil {
		return c.Wrapper.PaintBase(ctx, origin.Add(w.Rect.Min))
	}
	return nil
}

func main() {
	driver.Main(func(s screen.Screen) {
		side := widget.NewSizer(unit.DIPs(400), unit.Value{},
			NewUniform(color.RGBA{255, 128, 128, 255},
				widget.NewPadder(widget.AxisBoth, unit.Ems(0.5),
					widget.NewLabel("hi"),
				),
			),
		)
		board := NewBoard()
		err := board.Setup()
		if err != nil {
			log.Fatal(err)
		}
		body := NewUniform(color.RGBA{255, 255, 255, 255},
			board,
		)
		main := widget.NewFlow(widget.AxisHorizontal,
			stretch(widget.NewSheet(side), 0),
			stretch(widget.NewSheet(body), 1),
		)
		if err := widget.RunWindow(s, main, &widget.RunWindowOptions{
			NewWindowOptions: screen.NewWindowOptions{
				Title: "Hello",
			},
		}); err != nil {
			log.Fatal(err)
		}
	})
}
