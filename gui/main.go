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
	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/image/draw"
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
	cursorColor  = color.RGBA{64, 192, 0, 255}
	hoverColor   = color.RGBA{192, 192, 192, 255}
	cursorPoints = []image.Point{}
)

type TextureLayer []TexPos

type Stage struct {
	Screen    screen.Screen
	Size      image.Point
	TileSets  []*TileSet
	TileSize  image.Point
	TexLayers []TextureLayer // from bg to fg
	TexAt     map[TexPos]screen.Texture
}

func (s *Stage) TileTexs(p image.Point) []screen.Texture {
	idx := p.Y*s.Size.X + p.X
	texs := make([]screen.Texture, 0, len(s.TexLayers))
	for _, layer := range s.TexLayers {
		texp := layer[idx]
		texs = append(texs, s.TexAt[texp])
	}
	return texs
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

func (s *Stage) Setup() error {
	if s.Screen == nil {
		return fmt.Errorf("stage doesn't have screen")
	}
	if s.Size.X < 0 || s.Size.Y < 0 {
		return fmt.Errorf("stage size couldn't be negative: got %v", s.Size)
	}
	if s.TileSize.X <= 0 || s.TileSize.Y <= 0 {
		return fmt.Errorf("tile size needs 1 or more pixels : got %v", s.TileSize)
	}
	for i, layer := range s.TexLayers {
		if s.Size.X*s.Size.Y != len(layer) {
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
	posMap := make(map[TexPos]bool)
	for _, layer := range s.TexLayers {
		for _, pos := range layer {
			posMap[pos] = true
		}
	}
	s.TexAt = make(map[TexPos]screen.Texture)
	for pos := range posMap {
		t, i, j := pos.T, pos.I, pos.J
		if t >= len(s.TileSets) {
			return fmt.Errorf("invalid tileset index: %v, only %v exists", t, len(s.TileSets))
		}
		ts := s.TileSets[t]
		sz := ts.TileSize
		src := baseImgs[t]
		srcBound := image.Rect(i*sz, j*sz, (i+1)*sz, (j+1)*sz)
		tile := image.NewNRGBA(image.Rect(0, 0, s.TileSize.X, s.TileSize.Y))
		draw.NearestNeighbor.Scale(tile, tile.Rect, src, srcBound, draw.Src, nil)
		tileTex, err := toScreenTex(s.Screen, tile)
		if err != nil {
			return err
		}
		s.TexAt[pos] = tileTex
	}
	return nil
}

func toScreenTex(s screen.Screen, img image.Image) (screen.Texture, error) {
	size := img.Bounds().Max
	tex, err := s.NewTexture(size)
	if err != nil {
		return nil, err
	}
	buf, err := s.NewBuffer(size)
	if err != nil {
		tex.Release()
		return nil, err
	}
	draw.Copy(buf.RGBA(), image.Point{}, img, img.Bounds(), draw.Src, nil)
	tex.Upload(image.Point{}, buf, buf.Bounds())
	buf.Release()
	return tex, nil
}

type Tile struct {
	Tex TexPos
}

type TexPos struct {
	T, I, J int
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
			dragging     bool
			paintPending bool
			drag         image.Point
			dragOffset   image.Point
			offset       image.Point
			topLeft      image.Point
			sz           size.Event
			winSize      image.Point
		)
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
		stg := &Stage{
			Screen:   s,
			Size:     image.Pt(3, 4),
			TileSize: image.Pt(32, 32),
			TileSets: []*TileSet{
				tilesets["basictiles"],
				tilesets["overworld"],
			},
			TexLayers: []TextureLayer{
				TextureLayer{
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
				TextureLayer{
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
		}
		err = stg.Setup()
		if err != nil {
			log.Fatal(err)
		}
		cursorPos := [2]int{0, 0}
		hoverPos := [2]int{-1, -1}
		for y := 0; y < stg.TileSize.Y; y++ {
			for x := 0; x < stg.TileSize.X; x++ {
				if x == 0 || y == 0 || x == stg.TileSize.X-1 || y == stg.TileSize.Y-1 {
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
						if cursorPos[0] >= stg.Size.X {
							cursorPos[0] = stg.Size.X - 1
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
						if cursorPos[1] >= stg.Size.Y {
							cursorPos[1] = stg.Size.Y - 1
						}
					}
					if !paintPending {
						paintPending = true
						w.Send(paint.Event{})
					}
				}

			case mouse.Event:
				p := image.Point{X: int(e.X), Y: int(e.Y)}
				topLeft = image.Pt(offset.X-winSize.X/2+stg.TileSize.X/2*stg.Size.X, offset.Y-winSize.Y/2+stg.TileSize.Y/2*stg.Size.Y)
				x := (p.X + topLeft.X) / stg.TileSize.X
				y := (p.Y + topLeft.Y) / stg.TileSize.Y
				hx := x
				hy := y
				if hx < 0 || hx >= stg.Size.X || hy < 0 || hy >= stg.Size.Y {
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
					x := (p.X + topLeft.X) / stg.TileSize.X
					y := (p.Y + topLeft.Y) / stg.TileSize.Y
					if x < 0 || x >= stg.Size.X || y < 0 || y >= stg.Size.Y {
						break
					}
					cursorPos[0] = x
					cursorPos[1] = y
					if !paintPending {
						paintPending = true
						w.Send(paint.Event{})
					}
					break
				}
				if e.Button == mouse.ButtonMiddle && e.Direction == mouse.DirRelease {
					dragging = false
					dragOffset = image.Point{}
					break
				}
				if e.Button == mouse.ButtonMiddle && e.Direction != mouse.DirNone {
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
				var wg sync.WaitGroup
				drawBg(w, s, winSize)
				topLeft = image.Pt(offset.X-winSize.X/2+stg.TileSize.X/2*stg.Size.X, offset.Y-winSize.Y/2+stg.TileSize.Y/2*stg.Size.Y)
				for y := -(topLeft.Y & 0x7f); y < winSize.Y; y += stg.TileSize.Y {
					for x := -(topLeft.X & 0x7f); x < winSize.X; x += stg.TileSize.X {
						wg.Add(1)
						go drawTile(&wg, w, stg, topLeft, x, y)
					}
				}
				wg.Wait()
				if hoverPos[0] != -1 {
					drawHover(w, stg, topLeft, -topLeft.X+hoverPos[0]*stg.TileSize.X, -topLeft.Y+hoverPos[1]*stg.TileSize.Y)
				}
				drawCursor(w, stg, topLeft, -topLeft.X+cursorPos[0]*stg.TileSize.X, -topLeft.Y+cursorPos[1]*stg.TileSize.Y)
				w.Publish()
				paintPending = false

			case size.Event:
				sz = e
				winSize = image.Pt(sz.WidthPx, sz.HeightPx)

			case error:
				log.Print(e)
			}
		}
	})
}

func drawBg(w screen.Window, s screen.Screen, winSize image.Point) {
	tex, err := s.NewTexture(winSize)
	if err != nil {
		log.Println(err)
		return
	}
	buf, err := s.NewBuffer(winSize)
	if err != nil {
		tex.Release()
		log.Println(err)
		return
	}
	draw.Copy(buf.RGBA(), image.Point{}, image.White, buf.Bounds(), draw.Src, nil)
	tex.Upload(image.Point{}, buf, tex.Bounds())
	buf.Release()
	w.Copy(image.Point{}, tex, tex.Bounds(), screen.Src, nil)
}

func drawTile(wg *sync.WaitGroup, w screen.Window, stg *Stage, topLeft image.Point, x, y int) {
	defer wg.Done()
	tp := image.Point{
		(x + topLeft.X) / stg.TileSize.X,
		(y + topLeft.Y) / stg.TileSize.Y,
	}
	if tp.X < 0 || tp.X >= stg.Size.X || tp.Y < 0 || tp.Y >= stg.Size.Y {
		return
	}
	texs := stg.TileTexs(tp)
	for _, tex := range texs {
		w.Copy(image.Point{x, y}, tex, tex.Bounds(), screen.Over, nil)
	}
}

func drawCursor(w screen.Window, stg *Stage, topLeft image.Point, x, y int) {
	img := image.NewRGBA(image.Rect(0, 0, stg.TileSize.X, stg.TileSize.Y))
	for _, p := range cursorPoints {
		img.SetRGBA(p.X, p.Y, cursorColor)
	}
	tex, err := toScreenTex(stg.Screen, img)
	if err != nil {
		log.Println(err)
		return
	}
	w.Copy(image.Point{x, y}, tex, tex.Bounds(), screen.Over, nil)
	tex.Release()
}

func drawHover(w screen.Window, stg *Stage, topLeft image.Point, x, y int) {
	img := image.NewRGBA(image.Rect(0, 0, stg.TileSize.X, stg.TileSize.Y))
	for _, p := range cursorPoints {
		img.SetRGBA(p.X, p.Y, hoverColor)
	}
	tex, err := toScreenTex(stg.Screen, img)
	if err != nil {
		log.Println(err)
		return
	}
	w.Copy(image.Point{x, y}, tex, tex.Bounds(), screen.Over, nil)
	tex.Release()
}
