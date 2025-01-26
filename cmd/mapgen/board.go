package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/image/draw"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

type ImageLayer []SrcPos

type SrcPos struct {
	T, I, J int
}

type TileSet struct {
	Reference string
	Link      string
	Unzip     string
	Author    string
	License   string
	File      string
	TileSize  int
}

func loadImage(name string) (image.Image, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

type Board struct {
	node.LeafEmbed
	TileCount   image.Point
	TileSets    []*TileSet
	TileSize    image.Point
	Layers      []ImageLayer // from bg to fg
	ImgAt       map[SrcPos]image.Image
	CursorImg   image.Image
	HoverImg    image.Image
	CursorPos   image.Point
	HoverPos    image.Point
	CursorColor color.NRGBA
	HoverColor  color.NRGBA
	DragFrom    image.Point
	Dragging    bool
	Offset      image.Point
}

func NewBoard() *Board {
	f, err := os.Open("asset/tileset.toml")
	if err != nil {
		log.Fatal(err)
	}
	d := toml.NewDecoder(f)
	tilesets := make(map[string]*TileSet)
	_, err = d.Decode(&tilesets)
	if err != nil {
		log.Fatal(err)
	}
	w := &Board{
		TileCount: image.Pt(3, 4),
		TileSize:  image.Pt(32, 32),
		TileSets: []*TileSet{
			tilesets["basictiles"],
			tilesets["overworld"],
		},
		Layers: []ImageLayer{
			ImageLayer{
				{T: 0, I: 0, J: 0},
				{T: 0, I: 1, J: 0},
				{T: 0, I: 2, J: 0},
				{T: 0, I: 0, J: 1},
				{T: 0, I: 1, J: 1},
				{T: 0, I: 2, J: 1},
				{T: 0, I: 0, J: 2},
				{T: 0, I: 1, J: 2},
				{T: 0, I: 2, J: 2},
				{T: 1, I: 0, J: 0},
				{T: 1, I: 0, J: 1},
				{T: 1, I: 0, J: 2},
			},
			ImageLayer{
				{T: 1, I: 0, J: 0},
				{T: 1, I: 1, J: 0},
				{T: 1, I: 2, J: 0},
				{T: 1, I: 0, J: 1},
				{T: 1, I: 1, J: 1},
				{T: 1, I: 2, J: 1},
				{T: 1, I: 0, J: 2},
				{T: 1, I: 1, J: 2},
				{T: 1, I: 2, J: 2},
				{T: 0, I: 0, J: 0},
				{T: 0, I: 0, J: 1},
				{T: 0, I: 0, J: 2},
			},
		},
		HoverPos:    image.Pt(-1, -1),
		CursorColor: color.NRGBA{64, 192, 0, 255},
		HoverColor:  color.NRGBA{192, 192, 192, 255},
	}
	w.Wrapper = w
	return w
}

func (w *Board) Setup() error {
	if w.TileCount.X < 0 || w.TileCount.Y < 0 {
		return fmt.Errorf("stage size couldn't be negative: got %v", w.TileCount)
	}
	if w.TileSize.X <= 0 || w.TileSize.Y <= 0 {
		return fmt.Errorf("tile size needs 1 or more pixels : got %v", w.TileSize)
	}
	for i, layer := range w.Layers {
		if w.TileCount.X*w.TileCount.Y != len(layer) {
			return fmt.Errorf("tex layer %d size different from stage size", i)
		}
	}
	baseImgs := make([]image.Image, len(w.TileSets))
	for i, atx := range w.TileSets {
		img, err := loadImage(filepath.Join("asset", atx.File))
		if err != nil {
			log.Println(err)
			img = image.White
		}
		baseImgs[i] = img
	}
	posMap := make(map[SrcPos]bool)
	for _, layer := range w.Layers {
		for _, pos := range layer {
			posMap[pos] = true
		}
	}
	w.ImgAt = make(map[SrcPos]image.Image)
	for pos := range posMap {
		t, i, j := pos.T, pos.I, pos.J
		if t >= len(w.TileSets) {
			return fmt.Errorf("invalid tileset index: %v, only %v exists", t, len(w.TileSets))
		}
		ts := w.TileSets[t]
		sz := ts.TileSize
		src := baseImgs[t]
		srcBound := image.Rect(i*sz, j*sz, (i+1)*sz, (j+1)*sz)
		tileImg := image.NewNRGBA(image.Rect(0, 0, w.TileSize.X, w.TileSize.Y))
		draw.NearestNeighbor.Scale(tileImg, tileImg.Rect, src, srcBound, draw.Src, nil)
		w.ImgAt[pos] = tileImg
	}
	cursorPoints := make([]image.Point, 0)
	for y := 0; y < w.TileSize.Y; y++ {
		for x := 0; x < w.TileSize.X; x++ {
			if x == 0 || y == 0 || x == w.TileSize.X-1 || y == w.TileSize.Y-1 {
				cursorPoints = append(cursorPoints, image.Pt(x, y))
			}
		}
	}
	cs := image.NewNRGBA(image.Rect(0, 0, w.TileSize.X, w.TileSize.Y))
	for _, p := range cursorPoints {
		cs.SetNRGBA(p.X, p.Y, w.CursorColor)
	}
	w.CursorImg = cs
	hv := image.NewNRGBA(image.Rect(0, 0, w.TileSize.X, w.TileSize.Y))
	for _, p := range cursorPoints {
		hv.SetNRGBA(p.X, p.Y, w.HoverColor)
	}
	w.HoverImg = hv
	return nil
}

func (w *Board) TileImgs(p image.Point) []image.Image {
	idx := p.Y*w.TileCount.X + p.X
	imgs := make([]image.Image, 0, len(w.Layers))
	for _, layer := range w.Layers {
		ip := layer[idx]
		imgs = append(imgs, w.ImgAt[ip])
	}
	return imgs
}

func (w *Board) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	r := w.Rect.Add(origin)
	topLeft := image.Pt(w.Offset.X-w.Rect.Max.X/2+w.TileSize.X/2*w.TileCount.X, w.Offset.Y-w.Rect.Max.Y/2+w.TileSize.Y/2*w.TileCount.Y)
	var wg sync.WaitGroup
	for y := -(topLeft.Y & 0x7f); y < w.Rect.Max.Y; y += w.TileSize.Y {
		for x := -(topLeft.X & 0x7f); x < w.Rect.Max.X; x += w.TileSize.X {
			wg.Add(1)
			go func(x, y int) {
				defer wg.Done()
				ip := image.Point{
					(x + topLeft.X) / w.TileSize.X,
					(y + topLeft.Y) / w.TileSize.Y,
				}
				if ip.X < 0 || ip.X >= w.TileCount.X || ip.Y < 0 || ip.Y >= w.TileCount.Y {
					return
				}
				dr := r.Add(image.Point{x, y})
				imgs := w.TileImgs(ip)
				for _, m := range imgs {
					draw.Draw(ctx.Dst, dr, m, image.Point{}, draw.Over)
				}
			}(x, y)
		}
	}
	wg.Wait()
	if w.HoverPos.X != -1 {
		dr := r.Add(image.Point{-topLeft.X + w.HoverPos.X*w.TileSize.X, -topLeft.Y + w.HoverPos.Y*w.TileSize.Y})
		draw.Draw(ctx.Dst, dr, w.HoverImg, image.Point{}, draw.Over)
	}
	dr := r.Add(image.Point{-topLeft.X + w.CursorPos.X*w.TileSize.X, -topLeft.Y + w.CursorPos.Y*w.TileSize.Y})
	draw.Draw(ctx.Dst, dr, w.CursorImg, image.Point{}, draw.Over)
	return nil
}

func (w *Board) OnInputEvent(e any, origin image.Point) node.EventHandled {
	switch e := e.(type) {
	case node.KeyEvent:
		if e.Direction == key.DirPress {
			if e.Code == key.CodeLeftArrow {
				w.CursorPos.X--
				if w.CursorPos.X < 0 {
					w.CursorPos.X = 0
				}
				w.Mark(node.MarkNeedsPaintBase)
				break
			}
			if e.Code == key.CodeRightArrow {
				w.CursorPos.X++
				if w.CursorPos.X >= w.TileCount.X {
					w.CursorPos.X = w.TileCount.X - 1
				}
				w.Mark(node.MarkNeedsPaintBase)
				break
			}
			if e.Code == key.CodeUpArrow {
				w.CursorPos.Y--
				if w.CursorPos.Y < 0 {
					w.CursorPos.Y = 0
				}
				w.Mark(node.MarkNeedsPaintBase)
				break
			}
			if e.Code == key.CodeDownArrow {
				w.CursorPos.Y++
				if w.CursorPos.Y >= w.TileCount.Y {
					w.CursorPos.Y = w.TileCount.Y - 1
				}
				w.Mark(node.MarkNeedsPaintBase)
				break
			}
		}
	case mouse.Event:
		p := image.Point{X: int(e.X), Y: int(e.Y)}.Sub(origin)
		topLeft := image.Pt(w.Offset.X-w.Rect.Max.X/2+w.TileSize.X/2*w.TileCount.X, w.Offset.Y-w.Rect.Max.Y/2+w.TileSize.Y/2*w.TileCount.Y)
		x := (p.X + topLeft.X) / w.TileSize.X
		y := (p.Y + topLeft.Y) / w.TileSize.Y
		hx := x
		hy := y
		if hx < 0 || hx >= w.TileCount.X || hy < 0 || hy >= w.TileCount.Y {
			hx = -1
			hy = -1
		}
		if hx != w.HoverPos.X || hy != w.HoverPos.Y {
			w.HoverPos.X = hx
			w.HoverPos.Y = hy
			w.Mark(node.MarkNeedsPaintBase)
		}
		if e.Button == mouse.ButtonLeft && e.Direction == mouse.DirRelease {
			if x < 0 || x >= w.TileCount.X || y < 0 || y >= w.TileCount.Y {
				break
			}
			if x != w.CursorPos.X || y != w.CursorPos.Y {
				w.CursorPos.X = x
				w.CursorPos.Y = y
				w.Mark(node.MarkNeedsPaintBase)
				break
			}
		}
		if e.Button == mouse.ButtonMiddle && e.Direction != mouse.DirNone {
			w.Dragging = e.Direction == mouse.DirPress
			w.DragFrom = p
			break
		}
		if !w.Dragging {
			break
		}
		w.Offset = w.Offset.Sub(p.Sub(w.DragFrom))
		w.DragFrom = p
		w.Mark(node.MarkNeedsPaintBase)
	}
	return node.Handled
}
