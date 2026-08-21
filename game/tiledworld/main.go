package main

import (
	"encoding/gob"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	tileSize  = 16
	zoomScale = 8
	// maxSteps defines how many visual steps the cursor will have when its position changed.
	maxSteps = 3
)

var (
	faceSource *text.GoTextFaceSource
)

func init() {
	f, err := os.Open("data/font/PixelOperator.ttf")
	if err != nil {
		log.Fatal(err)
	}
	s, err := text.NewGoTextFaceSource(f)
	if err != nil {
		log.Fatal(err)
	}
	faceSource = s
}

type Point3 struct {
	X, Y, Z int
}

type SaveData struct {
	WorldData *WorldData
}

type WorldData struct {
	Map       map[Point3]int
	TileImage map[int]*image.RGBA
}

type World struct {
	Layers []*Layer
	Camera *Camera
}

func NewWorld() *World {
	w := &World{
		Layers: []*Layer{NewLayer()},
	}
	return w
}

// NewLayer creates a new layer then returns it's index.
func (w *World) AddLayer() int {
	w.Layers = append(w.Layers, NewLayer())
	return len(w.Layers) - 1
}

func (w *World) RemoveLayer(i int) error {
	if i < 0 {
		return fmt.Errorf("don't accept negative index: %v", i)
	}
	if i >= len(w.Layers) {
		return fmt.Errorf("index exceeded number of layers: %v", i)
	}
	if len(w.Layers) <= 1 {
		return fmt.Errorf("world should have at lease 1 layer")
	}
	w.Layers = append(w.Layers[:i], w.Layers[i+1:]...)
	return nil
}

func (w *World) ToData() *WorldData {
	d := &WorldData{
		Map:       make(map[Point3]int),
		TileImage: make(map[int]*image.RGBA),
	}
	tileID := make(map[*Tile]int)
	for i, l := range w.Layers {
		for p, tile := range l.Map {
			if tile == nil {
				log.Fatal("should not have nil in world map")
			}
			id := tileID[tile]
			if id == 0 {
				// unknown tile
				id = len(tileID) + 1
				tileID[tile] = id
				img := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
				tile.Image.ReadPixels(img.Pix)
				d.TileImage[id] = img
			}
			p3 := Point3{X: p.X, Y: p.Y, Z: i}
			d.Map[p3] = id
		}
	}
	return d
}

func (w *World) FromData(d *WorldData) {
	getTile := make(map[int]*Tile)
	for p3, id := range d.Map {
		for len(w.Layers)-1 < p3.Z {
			w.Layers = append(w.Layers, NewLayer())
		}
		t := getTile[id]
		if t == nil {
			img := d.TileImage[id]
			t = &Tile{}
			t.Image = ebiten.NewImage(tileSize, tileSize)
			t.Image.WritePixels(img.Pix)
			getTile[id] = t
		}
		p := image.Point{p3.X, p3.Y}
		w.Layers[p3.Z].Map[p] = t
	}
}

type Layer struct {
	Map map[image.Point]*Tile
}

func NewLayer() *Layer {
	return &Layer{
		Map: make(map[image.Point]*Tile),
	}
}

func (l *Layer) NewTile(p image.Point) *Tile {
	l.ClearTile(p)
	tile := &Tile{}
	tile.Image = ebiten.NewImage(tileSize, tileSize)
	l.Map[p] = tile
	return tile
}

func (l *Layer) ClearTile(p image.Point) {
	delete(l.Map, p)
	// TODO: clear the tile when all its references are gone
}

func (l *Layer) PutTile(p image.Point, t *Tile) {
	if t == nil {
		l.ClearTile(p)
		return
	}
	l.Map[p] = t
}

func (l *Layer) DuplicateTile(from image.Point, to image.Point) {
	tile, ok := l.Map[from]
	if !ok {
		l.ClearTile(to)
		return
	}
	l.Map[to] = tile
}

func (l *Layer) MakeTileUnique(p image.Point) {
	old, ok := l.Map[p]
	if !ok {
		return
	}
	tile := l.NewTile(p)
	draw.Draw(tile.Image, tile.Image.Bounds(), old.Image, image.Pt(0, 0), draw.Src)
}

func (l *Layer) TileAt(p image.Point) *Tile {
	return l.Map[p]
}

func (l *Layer) TilePoses(tile *Tile) []image.Point {
	pts := make([]image.Point, 0)
	for pt, t := range l.Map {
		if tile == t {
			pts = append(pts, pt)
		}
	}
	return pts
}

type Tile struct {
	Image *ebiten.Image
}

func keyDirection(k ebiten.Key) image.Point {
	switch k {
	case ebiten.KeyArrowUp:
		return image.Pt(0, -1)
	case ebiten.KeyArrowDown:
		return image.Pt(0, 1)
	case ebiten.KeyArrowLeft:
		return image.Pt(-1, 0)
	case ebiten.KeyArrowRight:
		return image.Pt(1, 0)
	}
	return image.Pt(0, 0)
}

type Mover struct {
	Pos    image.Point
	OldPos image.Point
	steps  int
}

func (m *Mover) MoveTo(p image.Point) {
	if m.steps != 0 {
		return
	}
	m.OldPos = m.Pos
	m.Pos = p
}

func (m *Mover) Step() {
	if m.Pos == m.OldPos {
		return
	}
	m.steps += 1
	if m.steps >= maxSteps {
		m.OldPos = m.Pos
		m.steps = 0
	}
}

func (m *Mover) VisualPos() point {
	dir := m.Pos.Sub(m.OldPos)
	return point{
		float64(m.OldPos.X) + float64(dir.X)*float64(m.steps)/maxSteps,
		float64(m.OldPos.Y) + float64(dir.Y)*float64(m.steps)/maxSteps,
	}
}

type NormalMode struct {
	Mover
	bounds        image.Rectangle
	World         *World
	WorldView     *WorldView
	CurLayer      int
	ExclusiveMode bool
	copyTilePos   image.Point
	PosSlots      [5]*image.Point
	Dirty         *bool
}

func (m *NormalMode) SetBounds(b image.Rectangle) {
	m.bounds = b
}

func (m *NormalMode) Bounds() image.Rectangle {
	return m.bounds
}

func (m *NormalMode) ActiveLayers() []*Layer {
	if m.ExclusiveMode {
		return []*Layer{m.Layer()}
	}
	return m.World.Layers
}

func (m *NormalMode) Layer() *Layer {
	// World should have at least 1 layer.
	return m.World.Layers[m.CurLayer]
}

func (m *NormalMode) NewTile() *Tile {
	return m.Layer().NewTile(m.Pos)
}

func (m *NormalMode) ActionTile() *Tile {
	return m.Layer().TileAt(m.Pos)
}

func (m *NormalMode) ClearTile() {
	for _, l := range m.ActiveLayers() {
		l.ClearTile(m.Pos)
	}
}

func (m *NormalMode) TilesAt(p image.Point) []*Tile {
	tiles := make([]*Tile, 0, len(m.World.Layers))
	for _, l := range m.World.Layers {
		t := l.TileAt(p)
		tiles = append(tiles, t)
	}
	return tiles
}

func (m *NormalMode) LayerUp() {
	m.CurLayer++
	n := len(m.World.Layers) - 1
	if m.CurLayer > n {
		m.CurLayer = n
	}
}

func (m *NormalMode) LayerDown() {
	m.CurLayer--
	if m.CurLayer < 0 {
		m.CurLayer = 0
	}
}

func (m *NormalMode) CopyPos() {
	m.copyTilePos = m.Pos
}

func (m *NormalMode) PastePos() {
	for _, l := range m.ActiveLayers() {
		l.DuplicateTile(m.copyTilePos, m.Pos)
	}
}

func (m *NormalMode) PasteTile(t *Tile) {
	for _, l := range m.ActiveLayers() {
		l.PutTile(m.Pos, t)
	}
}

func (m *NormalMode) MakeTileUnique() {
	for _, l := range m.ActiveLayers() {
		l.MakeTileUnique(m.Pos)
	}
}

func normalModeUpdate(g *Game, w *Widget) error {
	m := g.NormalMode
	keys := inpututil.AppendPressedKeys(nil)
	ctrl := false
	alt := false
	for _, k := range keys {
		if k == ebiten.KeyControl {
			ctrl = true
		}
		if k == ebiten.KeyAlt {
			alt = true
		}
	}
	slotKeys := []ebiten.Key{
		ebiten.Key1, // slot0
		ebiten.Key2,
		ebiten.Key3,
		ebiten.Key4,
		ebiten.Key5,
	}
	dest := m.Pos
	for _, k := range keys {
		if m.steps == 0 {
			d := keyDirection(k)
			if d != image.Pt(0, 0) {
				dest = dest.Add(d)
			}
		}
	}
	// move doesn't comsume update
	m.MoveTo(dest)
	// handle other operations
	for _, k := range keys {
		if k == ebiten.KeyMinus {
			if !inpututil.IsKeyJustPressed(k) {
				continue
			}
			if m.CurLayer != 0 {
				m.CurLayer--
			}
			return UpdateHandled
		}
		if k == ebiten.KeyEqual {
			if !inpututil.IsKeyJustPressed(k) {
				continue
			}
			if m.CurLayer == len(m.World.Layers)-1 {
				m.World.AddLayer()
			}
			m.CurLayer++
			return UpdateHandled
		}
		if k == ebiten.KeyE {
			if inpututil.IsKeyJustPressed(ebiten.KeyE) {
				m.ExclusiveMode = !m.ExclusiveMode
			}
			return UpdateHandled
		}
		for i, sk := range slotKeys {
			if k != sk {
				continue
			}
			if alt {
				p := m.Pos
				m.PosSlots[i] = &p
			} else {
				from := m.PosSlots[i]
				if from != nil {
					tiles := m.TilesAt(*from)
					for i := range m.World.Layers {
						t := tiles[i]
						if t == nil {
							delete(m.World.Layers[i].Map, m.Pos)
							continue
						}
						m.World.Layers[i].Map[m.Pos] = t
					}
				}
			}
			return UpdateHandled
		}
		if k == ebiten.KeyX {
			m.ClearTile()
			*m.Dirty = true
			return UpdateHandled
		}
		if k == ebiten.KeyC {
			m.CopyPos()
			return UpdateHandled
		}
		if k == ebiten.KeyV {
			m.PastePos()
			*m.Dirty = true
			return UpdateHandled
		}
		if k == ebiten.KeyD {
			m.MakeTileUnique()
			return UpdateHandled
		}
		if k == ebiten.KeyP {
			err := os.Mkdir("screenshot", 0755)
			if err != nil && !os.IsExist(err) {
				return fmt.Errorf("make screenshot directory: %v", err)
			}
			root, err := os.OpenRoot("screenshot")
			if err != nil {
				return fmt.Errorf("open screenshot directory: %v", err)
			}
			if ctrl {
				err := func() error {
					r := m.WorldView.Camera.Rect()
					screenshot := image.NewRGBA(image.Rect(int(r.Min.X), int(r.Min.Y), int(r.Max.X)*tileSize, int(r.Max.Y)*tileSize))
					f, err := root.Create("camera.png")
					if err != nil {
						return fmt.Errorf("create screenshot file: %v", err)
					}
					f.Close()
					for p, t := range m.Layer().Map {
						tmin := p.Mul(tileSize)
						tmax := p.Add(image.Pt(1, 1)).Mul(tileSize)
						draw.Draw(screenshot, image.Rect(tmin.X, tmin.Y, tmax.X, tmax.Y), t.Image, image.Pt(0, 0), draw.Src)
					}
					err = png.Encode(f, screenshot)
					if err != nil {
						return fmt.Errorf("png encode: %v", err)
					}
					return nil
				}()
				if err != nil {
					return err
				}
				return UpdateHandled
			}
			err = func() error {
				f, err := root.Create("tile.png")
				if err != nil {
					return fmt.Errorf("create tile image: %v", err)
				}
				defer f.Close()
				b := image.Rect(0, 0, tileSize, tileSize)
				tileImg := image.NewRGBA(b)
				for _, t := range m.TilesAt(m.Pos) {
					if t != nil {
						draw.Draw(tileImg, b, t.Image, image.Pt(0, 0), draw.Over)
					}
				}
				err = png.Encode(f, tileImg)
				if err != nil {
					return fmt.Errorf("png encode: %v", err)
				}
				return nil
			}()
			if err != nil {
				return err
			}
			return UpdateHandled
		}
	}
	return nil
}

func normalModeTick(g *Game, w *Widget) {
	m := g.NormalMode
	m.Step()
	v := g.NormalMode.WorldView
	// Points in World actually have size 1x1.
	// Which the point represents top-left corner.
	// Follow bottom-right corner as well.
	v.Camera.Follow(g.NormalMode.VisualPos())
	v.Camera.Follow(g.NormalMode.VisualPos().Add(pt(1, 1)))
}

func normalModeSlotsDraw(g *Game, w *Widget) {
	m := g.NormalMode
	screen := g.screen.SubImage(w.Bounds).(*ebiten.Image)
	toScreen := ebiten.GeoM{}
	toScreen.Translate(float64(w.Bounds.Min.X), float64(w.Bounds.Min.Y))

	slotPad := 20
	slotSize := tileSize*2 + 2 // +2 for outline
	slotsWidth := slotSize*len(m.PosSlots) + slotPad*(len(m.PosSlots)-1)
	sz := w.Bounds.Size()
	midX := sz.X/2 + 1
	midY := sz.Y/2 + 1
	slotsOrigin := image.Pt(midX-slotsWidth/2, midY-slotSize/2)
	slotImage := ebiten.NewImage(slotSize, slotSize)
	c := color.RGBA{R: 192, G: 192, B: 192, A: 255}
	at := image.Pt(slotsOrigin.X, slotsOrigin.Y)
	for i, pos := range m.PosSlots {
		// draw slot image itself
		slotImage.Clear()
		if pos != nil {
			for _, t := range m.TilesAt(*pos) {
				if t != nil {
					op := ebiten.DrawImageOptions{}
					op.GeoM.Scale(2, 2)
					op.GeoM.Translate(1, 1) // for outline
					slotImage.DrawImage(t.Image, &op)
				}
			}
		}
		drawOutline(slotImage, slotImage.Bounds(), 1, c)

		// draw slot image at an appropriate position
		op := ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(at.X)+1, float64(at.Y)+1)
		op.GeoM.Concat(toScreen)
		screen.DrawImage(slotImage, &op)

		// draw slot number
		top := text.DrawOptions{}
		top.GeoM.Translate(float64(at.X)+2, float64(at.Y)+1)
		top.GeoM.Concat(toScreen)
		top.ColorM.Scale(1, 1, 1, 0.5)
		text.Draw(
			screen,
			strconv.Itoa(i+1),
			&text.GoTextFace{
				Source: faceSource,
				Size:   16,
			},
			&top,
		)
		at = at.Add(image.Pt(slotSize+slotPad, 0))
	}
}

func normalModeAnalyzerDraw(g *Game, w *Widget) {
	m := g.NormalMode
	screen := g.screen.SubImage(w.Bounds).(*ebiten.Image)
	toScreen := ebiten.GeoM{}
	toScreen.Translate(float64(w.Bounds.Min.X), float64(w.Bounds.Min.Y))
	top := text.DrawOptions{
		LayoutOptions: text.LayoutOptions{
			LineSpacing:  20,
			PrimaryAlign: text.AlignEnd,
		},
	}
	top.ColorM.Scale(1, 1, 1, 0.5)
	top.GeoM.Translate(float64(w.Bounds.Size().X), 0) // to match text.AlignEnd
	top.GeoM.Concat(toScreen)
	text.Draw(
		screen,
		strings.Join([]string{
			fmt.Sprintf("Exclusive View (E): %v", m.ExclusiveMode),
			fmt.Sprintf("Current Layer: %v", m.CurLayer),
			fmt.Sprintf("Position: (%v, %v)", m.Pos.X, m.Pos.Y),
		}, "\n"),
		&text.GoTextFace{
			Source: faceSource,
			Size:   16,
		},
		&top,
	)
}

type MenuBar struct {
	Tiles []*Tile
	Idx   int
}

func menuBarUpdate(g *Game, w *Widget) error {
	m := g.MenuBar
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		m.Idx -= 1
		if m.Idx < 0 {
			m.Idx = 0
		}
		return UpdateHandled
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		m.Idx += 1
		if m.Idx >= len(m.Tiles) {
			m.Idx = len(m.Tiles) - 1
		}
		return UpdateHandled
	}
	return nil
}

func menuBarDraw(g *Game, w *Widget) {
	m := g.MenuBar
	outlineScreen := g.screen.SubImage(w.Bounds.Inset(-2)).(*ebiten.Image)
	c := color.RGBA{R: 192, G: 192, B: 192, A: 128}
	drawOutline(outlineScreen, w.Bounds.Inset(-2), 2, c)
	screen := g.screen.SubImage(w.Bounds).(*ebiten.Image)
	toScreen := ebiten.GeoM{}
	toScreen.Scale(2, 2)
	toScreen.Translate(float64(w.Bounds.Min.X), float64(w.Bounds.Min.Y))
	for i, t := range g.MenuBar.Tiles {
		op := ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(i)*tileSize, 0)
		op.GeoM.Concat(toScreen)
		screen.DrawImage(t.Image, &op)
	}
	if len(m.Tiles) == 0 {
		return
	}
	b := image.Rect(m.Idx*tileSize*2, 0, (m.Idx+1)*tileSize*2, tileSize*2).Add(w.Bounds.Min)
	c = color.RGBA{R: 192, G: 192, B: 64, A: 128}
	drawOutline(screen, b, 2, c)
}

// WorldView implements UpdateDrawer
type WorldView struct {
	Camera    *Camera
	cursorPos *image.Point
}

func worldViewUpdate(g *Game, w *Widget) error {
	v := g.NormalMode.WorldView
	cx, cy := ebiten.CursorPosition()
	if !image.Pt(cx, cy).In(w.Bounds) {
		v.cursorPos = nil
	} else {
		relP := image.Pt(cx, cy).Sub(w.Bounds.Min)
		rx := relP.X / tileSize / 2
		ry := relP.Y / tileSize / 2
		v.cursorPos = &image.Point{X: int(rx) + int(math.Round(v.Camera.Origin.X)), Y: int(ry) + int(math.Round(v.Camera.Origin.Y))}
	}
	if v.cursorPos != nil && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.NormalMode.MoveTo(*v.cursorPos)
		return UpdateHandled
	}
	return nil
}

func worldViewDraw(g *Game, w *Widget) {
	v := g.NormalMode.WorldView
	m := g.NormalMode
	outlineScreen := g.screen.SubImage(w.Bounds.Inset(-2)).(*ebiten.Image)
	c := color.RGBA{R: 192, G: 192, B: 192, A: 255}
	drawOutline(outlineScreen, w.Bounds.Inset(-2), 2, c)
	screen := g.screen.SubImage(w.Bounds).(*ebiten.Image)
	toScreen := ebiten.GeoM{}
	toScreen.Scale(2, 2)
	toScreen.Translate(float64(w.Bounds.Min.X), float64(w.Bounds.Min.Y))
	camRect := v.Camera.Rect()
	var layers []*Layer
	if m.ExclusiveMode {
		layers = []*Layer{m.Layer()}
	} else {
		layers = m.World.Layers
	}
	for _, l := range layers {
		for j := int(camRect.Min.Y) - 1; j < int(camRect.Max.Y)+1; j++ {
			for i := int(camRect.Min.X) - 1; i < int(camRect.Max.X)+1; i++ {
				tile, ok := l.Map[image.Pt(i, j)]
				if !ok {
					continue
				}
				op := ebiten.DrawImageOptions{}
				op.GeoM.Translate((float64(i)-float64(camRect.Min.X))*tileSize, (float64(j)-float64(camRect.Min.Y))*tileSize)
				op.GeoM.Concat(toScreen)
				screen.DrawImage(tile.Image, &op)
			}
		}
	}
	// draw cursor
	cursorImage := ebiten.NewImage(tileSize, tileSize)
	c = color.RGBA{R: 192, G: 192, B: 64, A: 128}
	drawOutline(cursorImage, cursorImage.Bounds(), 1, c)
	op := ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	vp := m.VisualPos()
	op.GeoM.Translate(float64(vp.X-camRect.Min.X)*tileSize, float64(vp.Y-camRect.Min.Y)*tileSize)
	op.GeoM.Concat(toScreen)
	screen.DrawImage(cursorImage, &op)
	// draw hover cursor
	if v.cursorPos != nil {
		cursorImage := ebiten.NewImage(tileSize, tileSize)
		c = color.RGBA{R: 192, G: 192, B: 192, A: 32}
		drawOutline(cursorImage, cursorImage.Bounds(), 1, c)
		op := ebiten.DrawImageOptions{}
		op.Blend = ebiten.BlendSourceOver
		// cursorPos is a relative position to camRect
		p := v.cursorPos
		op.GeoM.Translate((float64(p.X)-float64(camRect.Min.X))*tileSize, (float64(p.Y)-float64(camRect.Min.Y))*tileSize)
		op.GeoM.Concat(toScreen)
		screen.DrawImage(cursorImage, &op)
	}
	// draw copy cursor
	if m.copyTilePos.In(camRect.ImageRectangle()) {
		cursorImage.Clear()
		c = color.RGBA{R: 64, G: 64, B: 192, A: 128}
		drawOutline(cursorImage, cursorImage.Bounds(), 1, c)
		op := ebiten.DrawImageOptions{}
		op.Blend = ebiten.BlendSourceOver
		op.GeoM.Translate((float64(m.copyTilePos.X)-float64(camRect.Min.X))*tileSize, (float64(m.copyTilePos.Y)-float64(camRect.Min.Y))*tileSize)
		op.GeoM.Concat(toScreen)
		screen.DrawImage(cursorImage, &op)
	}
	// draw all matching cursor
	cursorImage.Clear()
	c = color.RGBA{R: 32, G: 32, B: 32, A: 32}
	drawOutline(cursorImage, cursorImage.Bounds(), 1, c)
	for _, p := range m.Layer().TilePoses(m.ActionTile()) {
		if !p.In(camRect.ImageRectangle()) {
			continue
		}
		op := ebiten.DrawImageOptions{}
		op.Blend = ebiten.BlendSourceOver
		op.GeoM.Translate((float64(p.X)-float64(camRect.Min.X))*tileSize, (float64(p.Y)-float64(camRect.Min.Y))*tileSize)
		op.GeoM.Concat(toScreen)
		screen.DrawImage(cursorImage, &op)
	}
}

func activeTileLayersDraw(g *Game, w *Widget) {
	m := g.NormalMode
	outlineScreen := g.screen.SubImage(w.Bounds.Inset(-2)).(*ebiten.Image)
	c := color.RGBA{R: 192, G: 192, B: 192, A: 255}
	drawOutline(outlineScreen, w.Bounds.Inset(-2), 2, c)
	screen := g.screen.SubImage(w.Bounds).(*ebiten.Image)
	toScreen := ebiten.GeoM{}
	toScreen.Scale(2, 2)
	toScreen.Translate(float64(w.Bounds.Min.X), float64(w.Bounds.Min.Y))
	for i, l := range m.World.Layers {
		p := m.VisualPos()
		// get top-left, and bottom-right corners
		p1 := p.Sub(pt(-0.5, -0.5))
		p2 := p.Add(pt(0.5, 0.5))
		x1, y1 := int(math.Floor(p1.X)), int(math.Floor(p1.Y))
		x2, y2 := int(math.Floor(p2.X)), int(math.Floor(p2.Y))
		tps := []image.Point{
			{X: x1, Y: y1},
			{X: x2, Y: y1},
			{X: x1, Y: y2},
			{X: x2, Y: y2},
		}
		for _, tp := range tps {
			t := l.TileAt(tp)
			if t == nil {
				continue
			}
			op := ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, float64((7-i)*tileSize))
			op.GeoM.Translate((float64(x2)-p.X)*tileSize, (float64(y2)-p.Y)*tileSize)
			op.GeoM.Concat(toScreen)
			screen.DrawImage(t.Image, &op)
		}
	}
	// draw cursor
	cursorBounds := image.Rect(0, 0, tileSize*2, tileSize*2).Add(image.Pt(0, (7-m.CurLayer)*tileSize*2)).Add(image.Pt(w.Bounds.Min.X, w.Bounds.Min.Y))
	c = color.RGBA{R: 192, G: 192, B: 64, A: 128}
	drawOutline(screen, cursorBounds, 2, c)
}

type ZoomMode struct {
	Dirty      *bool
	NormalMode *NormalMode
	Mover
	Hue        int
	Saturation int
	Lightness  int
}

func (m *ZoomMode) MoveTo(dest image.Point) {
	p := m.Pos
	m.Mover.MoveTo(dest)
	if p.In(image.Rect(0, 0, tileSize, tileSize)) {
		return
	}
	// user go outside of the tile
	np := m.NormalMode.Pos
	if p.X < 0 {
		np = np.Add(image.Pt(-1, 0))
	}
	if p.X >= tileSize {
		np = np.Add(image.Pt(1, 0))
	}
	if p.Y < 0 {
		np = np.Add(image.Pt(0, -1))
	}
	if p.Y >= tileSize {
		np = np.Add(image.Pt(0, 1))
	}
	// moved to a new tile if needed
	m.NormalMode.Pos = np
	m.NormalMode.OldPos = np
	m.NormalMode.steps = 0
	if p.X < 0 {
		p.X = tileSize - 1
	}
	if p.X >= tileSize {
		p.X = 0
	}
	if p.Y < 0 {
		p.Y = tileSize - 1
	}
	if p.Y >= tileSize {
		p.Y = 0
	}
	m.Pos = p
}

func zoomModeUpdate(g *Game, w *Widget) error {
	m := g.ZoomMode
	keys := inpututil.AppendPressedKeys(nil)
	alt := false
	shift := false
	for _, k := range keys {
		if k == ebiten.KeyAlt {
			alt = true
		}
		if k == ebiten.KeyShift {
			shift = true
		}
	}
	dest := m.Pos
	if alt {
		for _, k := range keys {
			if k == ebiten.KeyArrowLeft {
				m.Hue = max(m.Hue-8, 1)
			}
			if k == ebiten.KeyArrowRight {
				m.Hue = min(m.Hue+8, 255)
			}
			if k == ebiten.KeyMinus {
				m.Saturation = max(m.Saturation-16, 1)
			}
			if k == ebiten.KeyEqual {
				m.Saturation = min(m.Saturation+16, 255)
			}
			if k == ebiten.KeyArrowDown {
				m.Lightness = max(m.Lightness-16, 1)
			}
			if k == ebiten.KeyArrowUp {
				m.Lightness = min(m.Lightness+16, 255)
			}
		}
	} else if shift {
		tile := g.NormalMode.ActionTile()
		if tile != nil {
			for _, k := range keys {
				if k == ebiten.KeyArrowLeft {
					cutImage := ebiten.NewImage(1, tileSize)
					draw.Draw(cutImage, cutImage.Bounds(), tile.Image, image.Pt(0, 0), draw.Src)
					draw.Draw(tile.Image, image.Rect(0, 0, tileSize-1, tileSize), tile.Image, image.Pt(1, 0), draw.Src)
					draw.Draw(tile.Image, image.Rect(tileSize-1, 0, tileSize, tileSize), cutImage, image.Pt(0, 0), draw.Src)
				}
				if k == ebiten.KeyArrowRight {
					cutImage := ebiten.NewImage(1, tileSize)
					draw.Draw(cutImage, cutImage.Bounds(), tile.Image, image.Pt(tileSize-1, 0), draw.Src)
					draw.Draw(tile.Image, image.Rect(1, 0, tileSize, tileSize), tile.Image, image.Pt(0, 0), draw.Src)
					draw.Draw(tile.Image, image.Rect(0, 0, 1, tileSize), cutImage, image.Pt(0, 0), draw.Src)
				}
				if k == ebiten.KeyArrowUp {
					cutImage := ebiten.NewImage(tileSize, 1)
					draw.Draw(cutImage, cutImage.Bounds(), tile.Image, image.Pt(0, 0), draw.Src)
					draw.Draw(tile.Image, image.Rect(0, 0, tileSize, tileSize-1), tile.Image, image.Pt(0, 1), draw.Src)
					draw.Draw(tile.Image, image.Rect(0, tileSize-1, tileSize, tileSize), cutImage, image.Pt(0, 0), draw.Src)
				}
				if k == ebiten.KeyArrowDown {
					cutImage := ebiten.NewImage(tileSize, 1)
					draw.Draw(cutImage, cutImage.Bounds(), tile.Image, image.Pt(0, tileSize-1), draw.Src)
					draw.Draw(tile.Image, image.Rect(0, 1, tileSize, tileSize), tile.Image, image.Pt(0, 0), draw.Src)
					draw.Draw(tile.Image, image.Rect(0, 0, tileSize, 1), cutImage, image.Pt(0, 0), draw.Src)
				}
			}
		}
	} else {
		for _, k := range keys {
			if m.steps == 0 {
				d := keyDirection(k)
				if d != image.Pt(0, 0) {
					dest = dest.Add(d)
				}
			}
		}
		// move doesn't consume update
		m.MoveTo(dest)

		for _, k := range keys {
			if k == ebiten.KeyX {
				tile := g.NormalMode.ActionTile()
				if tile != nil {
					p := m.Pos
					tile.Image.Set(p.X, p.Y, color.RGBA{})
				}
				*m.Dirty = true
				return UpdateHandled
			}
			if k == ebiten.KeyC {
				tile := g.NormalMode.ActionTile()
				if tile == nil {
					tile = g.NormalMode.NewTile()
				}
				p := m.Pos
				c, _ := tile.Image.At(p.X, p.Y).(color.RGBA)
				if c.A != 0 {
					h, s, l := RGBToHSL(c)
					m.Hue = int(h * 255)
					m.Saturation = int(s * 255)
					m.Lightness = int(l * 255)
				}
				return UpdateHandled
			}
			if k == ebiten.KeyV {
				*m.Dirty = true
				tile := g.NormalMode.ActionTile()
				if tile == nil {
					tile = g.NormalMode.NewTile()
				}
				p := m.Pos
				c := HSLToRGB(float64(m.Hue)/255, float64(m.Saturation)/255, float64(m.Lightness)/255)
				tile.Image.Set(p.X, p.Y, c)
				return UpdateHandled
			}
		}
	}
	return nil
}

func zoomModeTick(g *Game, w *Widget) {
	m := g.ZoomMode
	m.Step()
}

func zoomModePalleteDraw(g *Game, w *Widget) {
	m := g.ZoomMode
	screen := g.screen.SubImage(w.Bounds).(*ebiten.Image)
	toScreen := ebiten.GeoM{}
	toScreen.Scale(2, 2)
	toScreen.Translate(float64(w.Bounds.Min.X), float64(w.Bounds.Min.Y))

	// draw colorpallete
	colorPickerSize := 32
	colorPalette := ebiten.NewImage(colorPickerSize, colorPickerSize)
	for h := range colorPickerSize {
		for l := range colorPickerSize {
			rgb := HSLToRGB(float64(h)/32, float64(m.Saturation)/256, float64(l)/32)
			colorPalette.Set(h, colorPickerSize-l-1, rgb)
		}
	}
	op := ebiten.DrawImageOptions{}
	op.GeoM.Concat(toScreen)
	screen.DrawImage(colorPalette, &op)

	// draw focus on color palette
	focus := ebiten.NewImage(5, 5)
	focusPts := []image.Point{{2, 0}, {2, 1}, {2, 3}, {2, 4}, {0, 2}, {1, 2}, {3, 2}, {4, 2}}
	for _, pt := range focusPts {
		focus.Set(pt.X, pt.Y, color.RGBA{R: 255, G: 255, A: 255})
	}
	op = ebiten.DrawImageOptions{}
	op.GeoM.Translate(-2, -2) // focus to origin
	op.GeoM.Translate(float64(m.Hue)/8, float64(255-m.Lightness-1)/8)
	op.GeoM.Concat(toScreen)
	screen.DrawImage(focus, &op)

	// draw picked color
	colorPicker := ebiten.NewImage(colorPickerSize, colorPickerSize)
	c := HSLToRGB(float64(m.Hue)/255, float64(m.Saturation)/255, float64(m.Lightness)/255)
	for h := 0; h < colorPickerSize; h += 1 {
		for s := 0; s < colorPickerSize; s += 1 {
			colorPicker.Set(h, colorPickerSize-s-1, c)
		}
	}
	op = ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 64)
	op.GeoM.Concat(toScreen)
	screen.DrawImage(colorPicker, &op)
}

func zoomModeTileDraw(g *Game, w *Widget) {
	m := g.ZoomMode
	screen := g.screen.SubImage(w.Bounds).(*ebiten.Image)
	toScreen := ebiten.GeoM{}
	toScreen.Scale(2, 2)
	toScreen.Translate(float64(w.Bounds.Min.X), float64(w.Bounds.Min.Y))
	toScreen.Translate(1, 1) // shift for outline

	// draw zoomed tile pixels
	tile := g.NormalMode.ActionTile()
	if tile != nil {
		op := ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoomScale, zoomScale)
		op.GeoM.Concat(toScreen)
		screen.DrawImage(tile.Image, &op)
	}

	// draw cursor
	cursorImage := ebiten.NewImage(zoomScale, zoomScale)
	c := color.RGBA{R: 192, G: 192, B: 64, A: 128}
	drawOutline(cursorImage, cursorImage.Bounds(), 1, c)
	op := ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	dir := m.Pos.Sub(m.OldPos)
	x := float64(m.OldPos.X) + float64(dir.X)*float64(m.steps)/maxSteps
	y := float64(m.OldPos.Y) + float64(dir.Y)*float64(m.steps)/maxSteps
	op.GeoM.Translate(x*zoomScale, y*zoomScale)
	op.GeoM.Concat(toScreen)
	screen.DrawImage(cursorImage, &op)

	// draw outline of zoomed tile
	drawOutline(screen, w.Bounds, 1, color.White)
}

type Game struct {
	ModeWidget                   *Widget
	MenuBar                      *MenuBar
	NormalMode                   *NormalMode
	ZoomMode                     *ZoomMode
	SaveFile                     string
	Dirty                        *bool
	askingQuitWithUnsavedChanges bool
	worldStopped                 bool
	Bounds                       image.Rectangle
	Widget                       *Widget
	screen                       *ebiten.Image
}

func (g *Game) Update() error {
	err := g.Widget.UpdateRecursive(g)
	if err != nil && err != UpdateHandled {
		return err
	}
	g.Widget.TickRecursive(g)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Clear()
	g.screen = screen
	g.Widget.DrawRecursive(g)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}

func (g *Game) save() {
	f, err := os.Create("save")
	if err == nil {
		enc := gob.NewEncoder(f)
		data := &SaveData{
			WorldData: g.NormalMode.World.ToData(),
		}
		if err := enc.Encode(data); err != nil {
			log.Fatalf("save data: %v", err)
		}
		f.Close()
	}
}

func gameUpdate(g *Game, w *Widget) error {
	keys := inpututil.AppendPressedKeys(nil)
	ctrl := false
	for _, k := range keys {
		if k == ebiten.KeyControl {
			ctrl = true
		}
	}
	if g.askingQuitWithUnsavedChanges {
		if inpututil.IsKeyJustPressed(ebiten.KeyY) {
			return ebiten.Termination
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.askingQuitWithUnsavedChanges = false
			return UpdateHandled
		}
		return UpdateHandled
	}
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		if *g.Dirty {
			g.askingQuitWithUnsavedChanges = true
			return UpdateHandled
		}
		return ebiten.Termination
	}
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.save()
		*g.Dirty = false
		return UpdateHandled
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		normalModeWidget := g.Widget.Child("body/normal")
		zoomModeWidget := g.Widget.Child("body/zoom")
		if g.ModeWidget == normalModeWidget {
			g.ModeWidget = zoomModeWidget
		} else {
			g.ModeWidget = normalModeWidget
		}
		return UpdateHandled
	}
	return nil
}

func gameNotifierDraw(g *Game, w *Widget) {
	screen := g.screen.SubImage(w.Bounds).(*ebiten.Image)
	toScreen := ebiten.GeoM{}
	toScreen.Translate(float64(w.Bounds.Min.X), float64(w.Bounds.Min.Y))
	if g.askingQuitWithUnsavedChanges {
		screen.Fill(color.RGBA{R: 32, G: 32, B: 32, A: 255})
		top := text.DrawOptions{
			LayoutOptions: text.LayoutOptions{
				SecondaryAlign: text.AlignEnd,
			},
		}
		top.GeoM.Translate(10, float64(w.Bounds.Size().Y)-10)
		top.GeoM.Concat(toScreen)
		top.ColorM.Scale(1, 1, 1, 1)
		text.Draw(
			screen,
			fmt.Sprintf("Want to quit without save your changes? (y/n)"),
			&text.GoTextFace{
				Source: faceSource,
				Size:   16,
			},
			&top,
		)
	}
}

func decodePng(fname string) (image.Image, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("open file: %v", err)
	}
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode png: %v", err)
	}
	return img, nil
}

func main() {
	// get World from save data if exists
	dirty := false
	menuBar := &MenuBar{
		Tiles: make([]*Tile, 0),
	}
	icons := []string{
		"data/icon/map.png",
		"data/icon/self.png",
		"data/icon/portal.png",
	}
	for _, fname := range icons {
		icon, err := decodePng(fname)
		if err != nil {
			log.Fatalf("load icon: %v", err)
		}
		menuBar.Tiles = append(menuBar.Tiles, &Tile{
			Image: ebiten.NewImageFromImage(icon),
		})
	}
	normalMode := &NormalMode{
		Dirty: &dirty,
	}
	camBound := rect(0, 0, 12, 8)
	normalMode.WorldView = &WorldView{
		Camera: NewCamera(camBound.Min, camBound.Max),
	}
	normalMode.WorldView.Camera.FollowMargin = 2
	game := &Game{
		Bounds:     image.Rect(0, 0, 640, 480),
		MenuBar:    menuBar,
		NormalMode: normalMode,
		ZoomMode: &ZoomMode{
			NormalMode: normalMode,
			Saturation: 255,
			Lightness:  128,
			Dirty:      &dirty,
		},
		Dirty: &dirty,
	}
	game.NormalMode.World = NewWorld()
	gob.Register(SaveData{})
	saved := &SaveData{}
	f, err := os.Open("save")
	if err == nil {
		defer f.Close()
		dec := gob.NewDecoder(f)
		err = dec.Decode(saved)
		if err != nil {
			log.Fatalf("load data: %v", err)
		}
		game.NormalMode.World.FromData(saved.WorldData)
	}
	game.Widget = &Widget{
		Update: gameUpdate,
		Children: []*Widget{
			&Widget{
				Name: "body",
				Pin:  WidgetPinTop,
				Size: image.Pt(0, -32),
				Children: []*Widget{
					&Widget{
						Name:   "menu",
						Update: menuBarUpdate,
						Draw:   menuBarDraw,
						Pin:    WidgetPinTopLeft,
						Offset: image.Pt(10, 10),
						Size:   image.Pt(12*tileSize, tileSize).Mul(2),
					},
					&Widget{
						Name: "normal",
						Block: func(g *Game, w *Widget) bool {
							return g.ModeWidget != w
						},
						Update: normalModeUpdate,
						Tick:   normalModeTick,
						Children: []*Widget{
							&Widget{
								Name:   "world",
								Update: worldViewUpdate,
								Draw:   worldViewDraw,
								Pin:    WidgetPinTopLeft,
								Offset: image.Pt(10, tileSize*2+25),
								Size:   image.Pt(12*tileSize, 8*tileSize).Mul(2),
							},
							&Widget{
								Name:   "activetile",
								Draw:   activeTileLayersDraw,
								Pin:    WidgetPinTopLeft,
								Offset: image.Pt(12*tileSize, 0).Mul(2).Add(image.Pt(25, tileSize*2+25)),
								Size:   image.Pt(tileSize, 8*tileSize).Mul(2),
							},
							&Widget{
								Name:   "analyzer",
								Draw:   normalModeAnalyzerDraw,
								Pin:    WidgetPinTopRight,
								Offset: image.Pt(-10, 10),
								Size:   image.Pt(200, 200),
							},
							&Widget{
								Name:   "slots",
								Draw:   normalModeSlotsDraw,
								Pin:    WidgetPinBottom,
								Offset: image.Pt(0, -10),
								Size:   image.Pt(0, 100),
							},
						},
					},
					&Widget{
						Name: "zoom",
						Block: func(g *Game, w *Widget) bool {
							return g.ModeWidget != w
						},
						Update: zoomModeUpdate,
						Tick:   zoomModeTick,
						Children: []*Widget{
							&Widget{
								Name:   "pallete",
								Draw:   zoomModePalleteDraw,
								Pin:    WidgetPinTopLeft,
								Offset: image.Pt(10, tileSize*2+25),
							},
							&Widget{
								Name: "tile",
								Draw: zoomModeTileDraw,
								Pin:  WidgetPinCenter,
								Size: image.Pt(zoomScale*tileSize, zoomScale*tileSize).Mul(2).Add(image.Pt(2, 2)),
							},
						},
					},
				},
			}, &Widget{
				Name: "notifier",
				Draw: gameNotifierDraw,
				Pin:  WidgetPinBottom,
				Size: image.Pt(0, 32),
			},
		},
	}
	game.Widget.Build(nil)
	game.ModeWidget = game.Widget.Child("body/normal")
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Tiled World")
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetScreenFilterEnabled(false)
	ebiten.SetTPS(20)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
