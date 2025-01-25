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

type Stage struct {
	node.LeafEmbed
	TileCount   image.Point
	TileSets    []*TileSet
	TileSize    image.Point
	Layers      []ImageLayer // from bg to fg
	ImgAt       map[SrcPos]image.Image
	CursorImg   image.Image
	HoverImg    image.Image
	CursorPos   [2]int
	HoverPos    [2]int
	CursorColor color.NRGBA
	HoverColor  color.NRGBA
	DragFrom    image.Point
	Dragging    bool
	Offset      image.Point
}

func NewStage() *Stage {
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
	s := &Stage{
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
		HoverPos:    [2]int{-1, -1},
		CursorColor: color.NRGBA{64, 192, 0, 255},
		HoverColor:  color.NRGBA{192, 192, 192, 255},
	}
	s.Wrapper = s
	return s
}

func (s *Stage) Setup() error {
	if s.TileCount.X < 0 || s.TileCount.Y < 0 {
		return fmt.Errorf("stage size couldn't be negative: got %v", s.TileCount)
	}
	if s.TileSize.X <= 0 || s.TileSize.Y <= 0 {
		return fmt.Errorf("tile size needs 1 or more pixels : got %v", s.TileSize)
	}
	for i, layer := range s.Layers {
		if s.TileCount.X*s.TileCount.Y != len(layer) {
			return fmt.Errorf("tex layer %d size different from stage size", i)
		}
	}
	baseImgs := make([]image.Image, len(s.TileSets))
	for i, atx := range s.TileSets {
		img, err := loadImage(filepath.Join("asset", atx.File))
		if err != nil {
			log.Println(err)
			img = image.White
		}
		baseImgs[i] = img
	}
	posMap := make(map[SrcPos]bool)
	for _, layer := range s.Layers {
		for _, pos := range layer {
			posMap[pos] = true
		}
	}
	s.ImgAt = make(map[SrcPos]image.Image)
	for pos := range posMap {
		t, i, j := pos.T, pos.I, pos.J
		if t >= len(s.TileSets) {
			return fmt.Errorf("invalid tileset index: %v, only %v exists", t, len(s.TileSets))
		}
		ts := s.TileSets[t]
		sz := ts.TileSize
		src := baseImgs[t]
		srcBound := image.Rect(i*sz, j*sz, (i+1)*sz, (j+1)*sz)
		tileImg := image.NewNRGBA(image.Rect(0, 0, s.TileSize.X, s.TileSize.Y))
		draw.NearestNeighbor.Scale(tileImg, tileImg.Rect, src, srcBound, draw.Src, nil)
		s.ImgAt[pos] = tileImg
	}
	cursorPoints := make([]image.Point, 0)
	for y := 0; y < s.TileSize.Y; y++ {
		for x := 0; x < s.TileSize.X; x++ {
			if x == 0 || y == 0 || x == s.TileSize.X-1 || y == s.TileSize.Y-1 {
				cursorPoints = append(cursorPoints, image.Pt(x, y))
			}
		}
	}
	cs := image.NewNRGBA(image.Rect(0, 0, s.TileSize.X, s.TileSize.Y))
	for _, p := range cursorPoints {
		cs.SetNRGBA(p.X, p.Y, s.CursorColor)
	}
	s.CursorImg = cs
	hv := image.NewNRGBA(image.Rect(0, 0, s.TileSize.X, s.TileSize.Y))
	for _, p := range cursorPoints {
		hv.SetNRGBA(p.X, p.Y, s.HoverColor)
	}
	s.HoverImg = hv
	return nil
}

func (s *Stage) TileImgs(p image.Point) []image.Image {
	idx := p.Y*s.TileCount.X + p.X
	imgs := make([]image.Image, 0, len(s.Layers))
	for _, layer := range s.Layers {
		ip := layer[idx]
		imgs = append(imgs, s.ImgAt[ip])
	}
	return imgs
}

func (s *Stage) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	s.Marks.UnmarkNeedsPaintBase()
	r := s.Rect.Add(origin)
	topLeft := image.Pt(s.Offset.X-s.Rect.Max.X/2+s.TileSize.X/2*s.TileCount.X, s.Offset.Y-s.Rect.Max.Y/2+s.TileSize.Y/2*s.TileCount.Y)
	var wg sync.WaitGroup
	for y := -(topLeft.Y & 0x7f); y < s.Rect.Max.Y; y += s.TileSize.Y {
		for x := -(topLeft.X & 0x7f); x < s.Rect.Max.X; x += s.TileSize.X {
			wg.Add(1)
			go func(x, y int) {
				defer wg.Done()
				ip := image.Point{
					(x + topLeft.X) / s.TileSize.X,
					(y + topLeft.Y) / s.TileSize.Y,
				}
				if ip.X < 0 || ip.X >= s.TileCount.X || ip.Y < 0 || ip.Y >= s.TileCount.Y {
					return
				}
				dr := r.Add(image.Point{x, y})
				imgs := s.TileImgs(ip)
				for _, m := range imgs {
					draw.Draw(ctx.Dst, dr, m, image.Point{}, draw.Over)
				}
			}(x, y)
		}
	}
	wg.Wait()
	if s.HoverPos[0] != -1 {
		dr := r.Add(image.Point{-topLeft.X + s.HoverPos[0]*s.TileSize.X, -topLeft.Y + s.HoverPos[1]*s.TileSize.Y})
		draw.Draw(ctx.Dst, dr, s.HoverImg, image.Point{}, draw.Over)
	}
	dr := r.Add(image.Point{-topLeft.X + s.CursorPos[0]*s.TileSize.X, -topLeft.Y + s.CursorPos[1]*s.TileSize.Y})
	draw.Draw(ctx.Dst, dr, s.CursorImg, image.Point{}, draw.Over)
	return nil
}

func (s *Stage) OnInputEvent(e any, origin image.Point) node.EventHandled {
	switch e := e.(type) {
	case node.KeyEvent:
		if e.Direction == key.DirPress {
			if e.Code == key.CodeLeftArrow {
				s.CursorPos[0]--
				if s.CursorPos[0] < 0 {
					s.CursorPos[0] = 0
				}
				s.Mark(node.MarkNeedsPaintBase)
				break
			}
			if e.Code == key.CodeRightArrow {
				s.CursorPos[0]++
				if s.CursorPos[0] >= s.TileCount.X {
					s.CursorPos[0] = s.TileCount.X - 1
				}
				s.Mark(node.MarkNeedsPaintBase)
				break
			}
			if e.Code == key.CodeUpArrow {
				s.CursorPos[1]--
				if s.CursorPos[1] < 0 {
					s.CursorPos[1] = 0
				}
				s.Mark(node.MarkNeedsPaintBase)
				break
			}
			if e.Code == key.CodeDownArrow {
				s.CursorPos[1]++
				if s.CursorPos[1] >= s.TileCount.Y {
					s.CursorPos[1] = s.TileCount.Y - 1
				}
				s.Mark(node.MarkNeedsPaintBase)
				break
			}
		}
	case mouse.Event:
		p := image.Point{X: int(e.X), Y: int(e.Y)}.Sub(origin)
		topLeft := image.Pt(s.Offset.X-s.Rect.Max.X/2+s.TileSize.X/2*s.TileCount.X, s.Offset.Y-s.Rect.Max.Y/2+s.TileSize.Y/2*s.TileCount.Y)
		x := (p.X + topLeft.X) / s.TileSize.X
		y := (p.Y + topLeft.Y) / s.TileSize.Y
		hx := x
		hy := y
		if hx < 0 || hx >= s.TileCount.X || hy < 0 || hy >= s.TileCount.Y {
			hx = -1
			hy = -1
		}
		if hx != s.HoverPos[0] || hy != s.HoverPos[1] {
			s.HoverPos[0] = hx
			s.HoverPos[1] = hy
			s.Mark(node.MarkNeedsPaintBase)
		}
		if e.Button == mouse.ButtonLeft && e.Direction == mouse.DirRelease {
			if x < 0 || x >= s.TileCount.X || y < 0 || y >= s.TileCount.Y {
				break
			}
			if x != s.CursorPos[0] || y != s.CursorPos[1] {
				s.CursorPos[0] = x
				s.CursorPos[1] = y
				s.Mark(node.MarkNeedsPaintBase)
				break
			}
		}
		if e.Button == mouse.ButtonMiddle && e.Direction != mouse.DirNone {
			s.Dragging = e.Direction == mouse.DirPress
			s.DragFrom = p
			break
		}
		if !s.Dragging {
			break
		}
		s.Offset = s.Offset.Sub(p.Sub(s.DragFrom))
		s.DragFrom = p
		s.Mark(node.MarkNeedsPaintBase)
	}
	return node.Handled
}
