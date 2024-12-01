package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"sync"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

var (
	crossColor  = color.RGBA{0x7f, 0x00, 0x00, 0xff}
	crossPoints = []image.Point{
		{0x00, 0x7e},
		{0x00, 0x7f},
		{0x7e, 0x00},
		{0x7f, 0x00},
		{0x00, 0x00},
		{0x01, 0x00},
		{0x02, 0x00},
		{0x00, 0x01},
		{0x00, 0x02},

		{0x40, 0x3f},
		{0x3f, 0x40},
		{0x40, 0x40},
		{0x41, 0x40},
		{0x40, 0x41},

		{0x40, 0x00},

		{0x00, 0x40},
	}
	cursorColor  = color.RGBA{0x7f, 0x7f, 0x00, 0xff}
	hoverColor   = color.RGBA{192, 192, 192, 255}
	cursorPoints = []image.Point{}

	generation int

	tileSize   = image.Point{64, 64}
	tileBounds = image.Rectangle{Max: tileSize}
	boardSize  = [2]int{10, 10}
)

func main() {
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(&screen.NewWindowOptions{
			Title: "Board GUI",
		})
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		var (
			pool = &tilePool{
				screen:         s,
				drawTileRGBA:   drawTileRGBA,
				drawCursorRGBA: drawCursorRGBA,
				m:              map[image.Point]*tilePoolEntry{},
			}
			dragging     bool
			paintPending bool
			drag         image.Point
			dragOffset   image.Point
			offset       image.Point
			topLeft      image.Point
			sz           size.Event
		)
		cursorPos := [2]int{0, 0}
		hoverPos := [2]int{-1, -1}
		for y := 0; y < tileSize.Y; y++ {
			for x := 0; x < tileSize.X; x++ {
				if x == 0 || y == 0 || x == tileSize.X-1 || y == tileSize.Y-1 {
					cursorPoints = append(cursorPoints, image.Pt(x, y))
				}
			}
		}
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
				if e.Direction == key.DirPress {
					if e.Code == key.CodeLeftArrow {
						cursorPos[0]--
						if cursorPos[0] < 0 {
							cursorPos[0] = 0
						}
					}
					if e.Code == key.CodeRightArrow {
						cursorPos[0]++
						if cursorPos[0] >= boardSize[0] {
							cursorPos[0] = boardSize[0] - 1
						}
					}
					if e.Code == key.CodeUpArrow {
						cursorPos[1]--
						if cursorPos[1] < 0 {
							cursorPos[1] = 0
						}
					}
					if e.Code == key.CodeDownArrow {
						cursorPos[1]++
						if cursorPos[1] >= boardSize[1] {
							cursorPos[1] = boardSize[1] - 1
						}
					}
					if !paintPending {
						paintPending = true
						w.Send(paint.Event{})
					}
				}

			case mouse.Event:
				p := image.Point{X: int(e.X), Y: int(e.Y)}
				winSize := image.Pt(sz.WidthPx, sz.HeightPx)
				topLeft = image.Pt(offset.X-winSize.X/2+tileSize.X/2*boardSize[0], offset.Y-winSize.Y/2+tileSize.Y/2*boardSize[1])
				x := (p.X + topLeft.X) / tileSize.X
				y := (p.Y + topLeft.Y) / tileSize.Y
				hx := x
				hy := y
				if hx < 0 || hx >= boardSize[0] || hy < 0 || hy >= boardSize[1] {
					hx = -1
					hy = -1
				}
				if hoverPos[0] != hx || hoverPos[1] != hy {
					hoverPos[0] = hx
					hoverPos[1] = hy
					if !paintPending {
						paintPending = true
						w.Send(paint.Event{})
					}
				}
				if e.Button == mouse.ButtonLeft && e.Direction == mouse.DirRelease {
					xoff := dragOffset.X
					if xoff < 0 {
						xoff = -xoff
					}
					yoff := dragOffset.Y
					if yoff < 0 {
						yoff = -yoff
					}
					if xoff < 3 && yoff < 3 {
						x := (p.X + topLeft.X) / tileSize.X
						y := (p.Y + topLeft.Y) / tileSize.Y
						if x < 0 || x >= boardSize[0] || y < 0 || y >= boardSize[1] {
							dragging = false
							break
						}
						cursorPos[0] = x
						cursorPos[1] = y
						if !paintPending {
							paintPending = true
							w.Send(paint.Event{})
						}
						dragging = false
						break
					}
					dragging = false
					dragOffset = image.Point{}
					break
				}
				if e.Button == mouse.ButtonLeft && e.Direction != mouse.DirNone {
					dragging = e.Direction == mouse.DirPress
					drag = p
					dragOffset = image.Point{}
				}
				if !dragging {
					break
				}
				offset = offset.Sub(p.Sub(drag))
				dragOffset = dragOffset.Sub(p.Sub(drag))
				drag = p
				if !paintPending {
					paintPending = true
					w.Send(paint.Event{})
				}

			case paint.Event:
				generation++
				var wg sync.WaitGroup
				winSize := image.Pt(sz.WidthPx, sz.HeightPx)
				topLeft = image.Pt(offset.X-winSize.X/2+tileSize.X/2*boardSize[0], offset.Y-winSize.Y/2+tileSize.Y/2*boardSize[1])
				for y := -(topLeft.Y & 0x7f); y < winSize.Y; y += tileSize.Y {
					for x := -(topLeft.X & 0x7f); x < winSize.X; x += tileSize.X {
						wg.Add(1)
						go drawTile(&wg, w, pool, topLeft, x, y)
					}
				}
				wg.Wait()
				if hoverPos[0] != -1 {
					wg.Add(1)
					go drawHover(&wg, w, pool, topLeft, -topLeft.X+hoverPos[0]*tileSize.X, -topLeft.Y+hoverPos[1]*tileSize.Y)
					wg.Wait()
				}
				wg.Add(1)
				go drawCursor(&wg, w, pool, topLeft, -topLeft.X+cursorPos[0]*tileSize.X, -topLeft.Y+cursorPos[1]*tileSize.Y)
				wg.Wait()
				w.Publish()
				paintPending = false
				pool.releaseUnused()

			case size.Event:
				sz = e

			case error:
				log.Print(e)
			}
		}
	})
}

func drawTile(wg *sync.WaitGroup, w screen.Window, pool *tilePool, topLeft image.Point, x, y int) {
	defer wg.Done()
	tp := image.Point{
		(x + topLeft.X) / tileSize.X,
		(y + topLeft.Y) / tileSize.Y,
	}
	tex, err := pool.get(tp)
	if err != nil {
		log.Println(err)
		return
	}
	w.Copy(image.Point{x, y}, tex, tileBounds, screen.Src, nil)
}

func drawTileRGBA(m *image.RGBA, tp image.Point) {
	draw.Draw(m, m.Bounds(), image.White, image.Point{}, draw.Src)
	for _, p := range crossPoints {
		m.SetRGBA(p.X, p.Y, crossColor)
	}
	d := font.Drawer{
		Dst:  m,
		Src:  image.Black,
		Face: inconsolata.Regular8x16,
		Dot: fixed.Point26_6{
			Y: inconsolata.Regular8x16.Metrics().Ascent,
		},
	}
	d.DrawString(fmt.Sprint(tp))
}

func drawCursor(wg *sync.WaitGroup, w screen.Window, pool *tilePool, topLeft image.Point, x, y int) {
	defer wg.Done()
	tp := image.Point{
		(x + topLeft.X) / tileSize.X,
		(y + topLeft.Y) / tileSize.Y,
	}
	tex, err := pool.screen.NewTexture(tileSize)
	if err != nil {
		log.Println(err)
		return
	}
	buf, err := pool.screen.NewBuffer(tileSize)
	if err != nil {
		tex.Release()
		log.Println(err)
		return
	}
	pool.drawCursorRGBA(buf.RGBA(), tp)
	tex.Upload(image.Point{}, buf, tileBounds)
	buf.Release()
	w.Copy(image.Point{x, y}, tex, tileBounds, screen.Over, nil)
	tex.Release()
}

func drawCursorRGBA(m *image.RGBA, tp image.Point) {
	draw.Draw(m, m.Bounds(), image.NewUniform(color.RGBA{}), image.Point{}, draw.Src)
	for _, p := range cursorPoints {
		m.SetRGBA(p.X, p.Y, cursorColor)
	}
}

func drawHover(wg *sync.WaitGroup, w screen.Window, pool *tilePool, topLeft image.Point, x, y int) {
	defer wg.Done()
	tp := image.Point{
		(x + topLeft.X) / tileSize.X,
		(y + topLeft.Y) / tileSize.Y,
	}
	tex, err := pool.screen.NewTexture(tileSize)
	if err != nil {
		log.Println(err)
		return
	}
	buf, err := pool.screen.NewBuffer(tileSize)
	if err != nil {
		tex.Release()
		log.Println(err)
		return
	}
	drawHoverRGBA(buf.RGBA(), tp)
	tex.Upload(image.Point{}, buf, tileBounds)
	buf.Release()
	w.Copy(image.Point{x, y}, tex, tileBounds, screen.Over, nil)
	tex.Release()
}

func drawHoverRGBA(m *image.RGBA, tp image.Point) {
	draw.Draw(m, m.Bounds(), image.NewUniform(color.RGBA{}), image.Point{}, draw.Src)
	for _, p := range cursorPoints {
		m.SetRGBA(p.X, p.Y, hoverColor)
	}
}

type tilePoolEntry struct {
	tex screen.Texture
	gen int
}

type tilePool struct {
	screen         screen.Screen
	drawTileRGBA   func(*image.RGBA, image.Point)
	drawCursorRGBA func(*image.RGBA, image.Point)

	mu sync.Mutex
	m  map[image.Point]*tilePoolEntry
}

func (p *tilePool) get(tp image.Point) (screen.Texture, error) {
	p.mu.Lock()
	v, ok := p.m[tp]
	if v != nil {
		v.gen = generation
	}
	p.mu.Unlock()

	if ok {
		return v.tex, nil
	}
	tex, err := p.screen.NewTexture(tileSize)
	if err != nil {
		return nil, err
	}
	buf, err := p.screen.NewBuffer(tileSize)
	if err != nil {
		tex.Release()
		return nil, err
	}
	if tp.X >= 0 && tp.X < boardSize[0] && tp.Y >= 0 && tp.Y < boardSize[1] {
		p.drawTileRGBA(buf.RGBA(), tp)
	}
	tex.Upload(image.Point{}, buf, tileBounds)
	buf.Release()

	p.mu.Lock()
	p.m[tp] = &tilePoolEntry{
		tex: tex,
		gen: generation,
	}
	n := len(p.m)
	p.mu.Unlock()

	fmt.Printf("%4d textures; created  %v\n", n, tp)
	return tex, nil
}

func (p *tilePool) releaseUnused() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for tp, v := range p.m {
		if v.gen == generation {
			continue
		}
		v.tex.Release()
		delete(p.m, tp)
		fmt.Printf("%4d textures; released %v\n", len(p.m), tp)
	}
}
