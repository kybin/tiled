package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	tileSize     = 16
	layoutWidth  = 320
	layoutHeight = 240
	zoomScale    = 8
	// maxSteps defines how many visual steps the cursor will have when its position changed.
	maxSteps = 3
)

var (
	faceSource *text.GoTextFaceSource
)

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	faceSource = s
}

type Point3 struct {
	X, Y, Z int
}

func what(vs ...any) {
	e, _ := os.Create("what")
	e.WriteString(fmt.Sprintf("%v", vs))
	e.Close()
}

type SaveData struct {
	WorldData *WorldData
}

type WorldData struct {
	Map     map[Point3]int
	GetTile map[int]*Tile
}

type World struct {
	Layers []*Layer
	Camera *Camera
}

type point struct {
	X, Y float32
}

func pt(x, y float32) point {
	return point{X: x, Y: y}
}

func (p point) Add(q point) point {
	return point{p.X + q.X, p.Y + q.Y}
}

func (p point) Mul(a float32) point {
	return point{p.X * a, p.Y * a}
}

func (p point) In(r rectangle) bool {
	if p.X < r.Min.X || p.X >= r.Max.X {
		return false
	}
	if p.Y < r.Min.Y || p.Y >= r.Max.Y {
		return false
	}
	return true
}

type rectangle struct {
	Min, Max point
}

func rect(xmin, ymin, xmax, ymax float32) rectangle {
	return rectangle{
		Min: point{X: xmin, Y: ymin},
		Max: point{X: xmax, Y: ymax},
	}
}

func (r rectangle) Inset(a float32) rectangle {
	if r.Max.X-r.Min.X < a {
		r.Min.X = (r.Min.X + r.Max.X) / 2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += a
		r.Max.X -= a
	}
	if r.Max.Y-r.Min.Y < a {
		r.Min.Y = (r.Min.Y + r.Max.Y) / 2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += a
		r.Max.Y -= a
	}
	return r
}

func (r rectangle) ImageRectangle() image.Rectangle {
	return image.Rect(int(r.Min.X), int(r.Min.Y), int(r.Max.X), int(r.Max.Y))
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
		Map:     make(map[Point3]int),
		GetTile: make(map[int]*Tile),
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
				d.GetTile[id] = tile
			}
			p3 := Point3{X: p.X, Y: p.Y, Z: i}
			d.Map[p3] = id
		}
	}
	return d
}

func (w *World) FromData(d *WorldData) {
	for p3, id := range d.Map {
		for len(w.Layers)-1 < p3.Z {
			w.Layers = append(w.Layers, NewLayer())
		}
		t := d.GetTile[id]
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
	tile.Image = image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
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
	Image *image.RGBA
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
	if p == m.Pos && p == m.OldPos {
		return
	}
	if m.steps == 0 {
		m.OldPos = m.Pos
		m.Pos = p
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
		float32(m.OldPos.X) + float32(dir.X)*float32(m.steps)/maxSteps,
		float32(m.OldPos.Y) + float32(dir.Y)*float32(m.steps)/maxSteps,
	}
}

type UpdateDrawer interface {
	Update() error
	Draw(*ebiten.Image)
	// SetBounds let parent UpdateDrawer set bounds of this UpdateDrawer.
	// SetBounds should be called in Update.
	SetBounds(image.Rectangle)
	// Bounds let parent UpdateDrawer know bounds of this UpdateDrawer,
	// So the parent can pass that portion of *ebiten.Image inside of Draw.
	Bounds() image.Rectangle
	SubUpdateDrawers() []UpdateDrawer
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

func (m *NormalMode) Update() error {
	keys := inpututil.AppendPressedKeys(nil)
	alt := false
	for _, k := range keys {
		if k == ebiten.KeyAlt {
			alt = true
			break
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
		if k == ebiten.KeyMinus {
			if !inpututil.IsKeyJustPressed(k) {
				continue
			}
			if m.CurLayer != 0 {
				m.CurLayer--
			}
			continue
		}
		if k == ebiten.KeyEqual {
			if !inpututil.IsKeyJustPressed(k) {
				continue
			}
			if m.CurLayer == len(m.World.Layers)-1 {
				m.World.AddLayer()
			}
			m.CurLayer++
			continue
		}
		if k == ebiten.KeyE {
			if inpututil.IsKeyJustPressed(ebiten.KeyE) {
				m.ExclusiveMode = !m.ExclusiveMode
			}
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
			break
		}
		if k == ebiten.KeyX {
			m.ClearTile()
			*m.Dirty = true
			continue
		}
		if k == ebiten.KeyC {
			m.CopyPos()
			continue
		}
		if k == ebiten.KeyV {
			m.PastePos()
			*m.Dirty = true
			continue
		}
		if k == ebiten.KeyD {
			m.MakeTileUnique()
			continue
		}
		if k == ebiten.KeyP {
			r := m.WorldView.Camera.Rect()
			screenshot := image.NewRGBA(image.Rect(int(r.Min.X), int(r.Min.Y), int(r.Max.X)*tileSize, int(r.Max.Y)*tileSize))
			f, err := os.Create("screenshot.png")
			if err != nil {
				what(err)
				panic(err)
			}
			for p, t := range m.Layer().Map {
				tmin := p.Mul(tileSize)
				tmax := p.Add(image.Pt(1, 1)).Mul(tileSize)
				draw.Draw(screenshot, image.Rect(tmin.X, tmin.Y, tmax.X, tmax.Y), t.Image, image.Pt(0, 0), draw.Src)
			}
			err = png.Encode(f, screenshot)
			if err != nil {
				what(err)
				panic(err)
			}
			continue
		}
		if m.steps == 0 {
			d := keyDirection(k)
			if d != image.Pt(0, 0) {
				dest = dest.Add(d)
			}
		}
	}
	m.MoveTo(dest)
	m.WorldView.Camera.Follow(m.VisualPos())
	for _, ud := range m.SubUpdateDrawers() {
		err := ud.Update()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *NormalMode) Draw(screen *ebiten.Image) {
	for _, ud := range m.SubUpdateDrawers() {
		subScreen := screen.SubImage(ud.Bounds()).(*ebiten.Image)
		ud.Draw(subScreen)
	}
	// draw slots at lower center
	slotPad := 10
	slotWidth := (tileSize+2)*len(m.PosSlots) + slotPad*len(m.PosSlots) // +2 for outline
	slotHeight := tileSize + 2
	mid := layoutWidth/2 + 1
	slotOrigin := image.Pt(mid-slotWidth/2, layoutHeight-slotHeight-25)
	slotImage := ebiten.NewImage((tileSize*2)+2, (tileSize*2)+2)
	c := color.RGBA{R: 192, G: 192, B: 192, A: 255}
	op := &ebiten.DrawImageOptions{}
	at := image.Pt(slotOrigin.X, slotOrigin.Y)
	for _, pos := range m.PosSlots {
		op.GeoM.Reset()
		op.GeoM.Translate(float64(at.X*2), float64(at.Y*2))
		slotImage.Clear()
		draw.Draw(slotImage, image.Rect(1, 1, tileSize+1, tileSize+1), image.Black, image.Pt(0, 0), draw.Src)
		if pos != nil {
			for _, t := range m.TilesAt(*pos) {
				if t != nil {
					draw.Draw(slotImage, image.Rect(1, 1, tileSize+1, tileSize+1), t.Image, image.Pt(0, 0), draw.Over)
				}
			}
		}
		drawOutline(slotImage, slotImage.Bounds(), c)
		screen.DrawImage(slotImage, op)
		at = at.Add(image.Pt(tileSize+2+slotPad, 0))
	}
	op = &ebiten.DrawImageOptions{}
	at = image.Pt(slotOrigin.X, slotOrigin.Y)
	for i := range m.PosSlots {
		top := &text.DrawOptions{}
		top.GeoM.Translate(float64(at.X*2)+2, float64(at.Y*2))
		top.ColorM.Scale(1, 1, 1, 0.5)
		text.Draw(
			screen,
			strconv.Itoa(i+1),
			&text.GoTextFace{
				Source: faceSource,
				Size:   16,
			},
			top,
		)
		at = at.Add(image.Pt(tileSize+2+slotPad, 0))
	}
	top := &text.DrawOptions{
		LayoutOptions: text.LayoutOptions{
			PrimaryAlign: text.AlignEnd,
		},
	}
	width, _ := screen.Size()
	top.ColorM.Scale(1, 1, 1, 0.5)
	top.GeoM.Translate(float64(width)-10, 10)
	text.Draw(
		screen,
		fmt.Sprintf("Exclusive View (E): %v", m.ExclusiveMode),
		&text.GoTextFace{
			Source: faceSource,
			Size:   16,
		},
		top,
	)
	top.GeoM.Translate(0, 20)
	text.Draw(
		screen,
		fmt.Sprintf("Current Layer: %v", m.CurLayer),
		&text.GoTextFace{
			Source: faceSource,
			Size:   16,
		},
		top,
	)
}

func (m *NormalMode) SubUpdateDrawers() []UpdateDrawer {
	return []UpdateDrawer{m.WorldView}
}

// WorldView implements UpdateDrawer
type WorldView struct {
	normalMode *NormalMode
	bounds     image.Rectangle
	Camera     *Camera
	hoverPos   *image.Point
}

func (v *WorldView) Update() error {
	cx, cy := ebiten.CursorPosition()
	if !image.Pt(cx, cy).In(v.bounds) {
		v.hoverPos = nil
	} else {
		relP := image.Pt(cx, cy).Sub(v.bounds.Min)
		rx := relP.X / tileSize / 2
		ry := relP.Y / tileSize / 2
		v.hoverPos = &image.Point{X: int(v.Camera.Origin.X) + int(rx), Y: int(v.Camera.Origin.Y) + int(ry)}
	}
	return nil
}

func (v *WorldView) Draw(screen *ebiten.Image) {
	m := v.normalMode
	if m == nil {
		return
	}
	origin := screen.Bounds().Min
	camRect := m.WorldView.Camera.Rect()
	camSize := m.WorldView.Camera.Size
	tileImage := ebiten.NewImage(tileSize, tileSize)
	var layers []*Layer
	if m.ExclusiveMode {
		layers = []*Layer{m.Layer()}
	} else {
		layers = m.World.Layers
	}
	for _, l := range layers {
		for j := int(camRect.Min.Y); j < int(camRect.Max.Y)+1; j++ {
			for i := int(camRect.Min.X); i < int(camRect.Max.X)+1; i++ {
				tile, ok := l.Map[image.Pt(i, j)]
				if ok {
					tileImage.WritePixels(tile.Image.Pix)
				} else {
					tileImage.Clear()
				}
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate((float64(i)-float64(camRect.Min.X))*tileSize, (float64(j)-float64(camRect.Min.Y))*tileSize)
				op.GeoM.Scale(2, 2)
				op.GeoM.Translate(float64(origin.X), float64(origin.Y))
				screen.DrawImage(tileImage, op)
			}
		}
	}
	c := color.RGBA{R: 192, G: 192, B: 192, A: 255}
	drawOutline(screen, image.Rect(origin.X, origin.Y, origin.X+int(camSize.X)*tileSize*2, origin.Y+int(camSize.Y)*tileSize*2), c)
	// draw cursor
	cursorImage := ebiten.NewImage(tileSize, tileSize)
	c = color.RGBA{R: 192, G: 192, B: 64, A: 128}
	drawOutline(cursorImage, cursorImage.Bounds(), c)
	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	vp := m.VisualPos()
	op.GeoM.Translate(float64(vp.X-camRect.Min.X)*tileSize, float64(vp.Y-camRect.Min.Y)*tileSize)
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(screen.Bounds().Min.X), float64(screen.Bounds().Min.Y))
	screen.DrawImage(cursorImage, op)
	// draw hover cursor
	if v.hoverPos != nil {
		cursorImage := ebiten.NewImage(tileSize, tileSize)
		c = color.RGBA{R: 192, G: 192, B: 192, A: 64}
		drawOutline(cursorImage, cursorImage.Bounds(), c)
		op := &ebiten.DrawImageOptions{}
		op.Blend = ebiten.BlendSourceOver
		p := v.hoverPos
		op.GeoM.Translate((float64(p.X)-float64(camRect.Min.X))*tileSize, (float64(p.Y)-float64(camRect.Min.Y))*tileSize)
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(screen.Bounds().Min.X), float64(screen.Bounds().Min.Y))
		screen.DrawImage(cursorImage, op)
	}
	// draw copy cursor
	if m.copyTilePos.In(camRect.ImageRectangle()) {
		cursorImage.Clear()
		c = color.RGBA{R: 64, G: 64, B: 192, A: 128}
		drawOutline(cursorImage, cursorImage.Bounds(), c)
		op = &ebiten.DrawImageOptions{}
		op.Blend = ebiten.BlendSourceOver
		op.GeoM.Translate((float64(m.copyTilePos.X)-float64(camRect.Min.X))*tileSize, (float64(m.copyTilePos.Y)-float64(camRect.Min.Y))*tileSize)
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(screen.Bounds().Min.X), float64(screen.Bounds().Min.Y))
		screen.DrawImage(cursorImage, op)
	}
	// draw all matching cursor
	cursorImage.Clear()
	c = color.RGBA{R: 32, G: 32, B: 32, A: 32}
	drawOutline(cursorImage, cursorImage.Bounds(), c)
	op = &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	for _, p := range m.Layer().TilePoses(m.ActionTile()) {
		if !p.In(camRect.ImageRectangle()) {
			continue
		}
		op.GeoM.Reset()
		op.GeoM.Translate((float64(p.X)-float64(camRect.Min.X))*tileSize, (float64(p.Y)-float64(camRect.Min.Y))*tileSize)
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(screen.Bounds().Min.X), float64(screen.Bounds().Min.Y))
		screen.DrawImage(cursorImage, op)
	}
}

func (v *WorldView) SetBounds(b image.Rectangle) {
	v.bounds = b
}

func (v *WorldView) Bounds() image.Rectangle {
	return v.bounds
}

func (v *WorldView) SubUpdateDrawers() []UpdateDrawer {
	return nil
}

type ZoomMode struct {
	bounds     image.Rectangle
	Dirty      *bool
	NormalMode *NormalMode
	Mover
	Hue        int
	Saturation int
	Lightness  int
}

func (m *ZoomMode) SetBounds(b image.Rectangle) {
	m.bounds = b
}

func (m *ZoomMode) Bounds() image.Rectangle {
	return m.bounds
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

func (m *ZoomMode) Update() error {
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
		tile := m.NormalMode.ActionTile()
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
			if k == ebiten.KeyX {
				tile := m.NormalMode.ActionTile()
				if tile != nil {
					p := m.Pos
					tile.Image.Set(p.X, p.Y, color.RGBA{})
				}
				*m.Dirty = true
			}
			if k == ebiten.KeyC {
				tile := m.NormalMode.ActionTile()
				if tile == nil {
					tile = m.NormalMode.NewTile()
				}
				p := m.Pos
				c, _ := tile.Image.At(p.X, p.Y).(color.RGBA)
				if c.A != 0 {
					h, s, l := RGBToHSL(c)
					m.Hue = int(h * 255)
					m.Saturation = int(s * 255)
					m.Lightness = int(l * 255)
				}
			}
			if k == ebiten.KeyV {
				*m.Dirty = true
				tile := m.NormalMode.ActionTile()
				if tile == nil {
					tile = m.NormalMode.NewTile()
				}
				p := m.Pos
				c := HSLToRGB(float64(m.Hue)/255, float64(m.Saturation)/255, float64(m.Lightness)/255)
				tile.Image.Set(p.X, p.Y, c)
			}
		}
	}
	m.MoveTo(dest)
	return nil
}

func (m *ZoomMode) Draw(fullscreen *ebiten.Image) {
	screen := ebiten.NewImage(layoutWidth, layoutHeight)
	colorPickerSize := 32
	colorPalette := ebiten.NewImage(colorPickerSize, colorPickerSize)
	for h := range colorPickerSize {
		for l := range colorPickerSize {
			rgb := HSLToRGB(float64(h)/32, float64(m.Saturation)/256, float64(l)/32)
			colorPalette.Set(h, colorPickerSize-l-1, rgb)
		}
	}
	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(colorPalette, op)
	focus := ebiten.NewImage(5, 5)
	focusPts := []image.Point{{2, 0}, {2, 1}, {2, 3}, {2, 4}, {0, 2}, {1, 2}, {3, 2}, {4, 2}}
	for _, pt := range focusPts {
		focus.Set(pt.X, pt.Y, color.RGBA{R: 255, G: 255, A: 255})
	}
	op.GeoM.Translate(float64(m.Hue)/8-2, float64(255-m.Lightness-1)/8-2)
	screen.DrawImage(focus, op)
	colorPicker := ebiten.NewImage(colorPickerSize, colorPickerSize)
	c := HSLToRGB(float64(m.Hue)/255, float64(m.Saturation)/255, float64(m.Lightness)/255)
	for h := 0; h < colorPickerSize; h += 1 {
		for s := 0; s < colorPickerSize; s += 1 {
			colorPicker.Set(h, colorPickerSize-s-1, c)
		}
	}
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 64)
	screen.DrawImage(colorPicker, op)
	// draw zoomed tile
	zoomedTileSize := zoomScale * tileSize
	center := image.Pt(layoutWidth/2+1, layoutHeight/2+1)
	origin := image.Pt(center.X-zoomedTileSize/2, center.Y-zoomedTileSize/2)
	tileImage := ebiten.NewImage(tileSize, tileSize)
	tile := m.NormalMode.ActionTile()
	if tile != nil {
		tileImage.WritePixels(tile.Image.Pix)
	}
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(zoomScale, zoomScale)
	op.GeoM.Translate(float64(origin.X), float64(origin.Y))
	screen.DrawImage(tileImage, op)
	// draw cursor
	cursorImage := ebiten.NewImage(zoomScale, zoomScale)
	c = color.RGBA{R: 192, G: 192, B: 64, A: 128}
	drawOutline(cursorImage, cursorImage.Bounds(), c)
	op = &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	dir := m.Pos.Sub(m.OldPos)
	x := float64(m.OldPos.X) + float64(dir.X)*float64(m.steps)/maxSteps
	y := float64(m.OldPos.Y) + float64(dir.Y)*float64(m.steps)/maxSteps
	op.GeoM.Translate(float64(origin.X)+x*zoomScale, float64(origin.Y)+y*zoomScale)
	screen.DrawImage(cursorImage, op)
	// draw outline
	c = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	b := image.Rectangle{}
	b.Min = origin.Sub(image.Pt(1, 1))
	b.Max = origin.Add(image.Pt(zoomedTileSize+1, zoomedTileSize+1))
	drawOutline(screen, b, c)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	fullscreen.DrawImage(screen, op)
}

func (m *ZoomMode) SubUpdateDrawers() []UpdateDrawer {
	return nil
}

type Game struct {
	Mode                         UpdateDrawer
	NormalMode                   *NormalMode
	ZoomMode                     *ZoomMode
	SaveFile                     string
	Dirty                        *bool
	askingQuitWithUnsavedChanges bool
	bounds                       image.Rectangle
}

func (g *Game) Update() error {
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
			return nil
		}
		return nil
	}
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		if *g.Dirty {
			g.askingQuitWithUnsavedChanges = true
			return nil
		}
		return ebiten.Termination
	}
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.save()
		*g.Dirty = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if g.Mode == g.NormalMode {
			g.Mode = g.ZoomMode
		} else {
			g.Mode = g.NormalMode
		}
	}
	for _, ud := range g.SubUpdateDrawers() {
		err := ud.Update()
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	defer func() {
		r := recover()
		if r != nil {
			f, err := os.Create("err")
			if err != nil {
				// nothing I can do
				return
			}
			defer f.Close()
			f.WriteString(fmt.Sprintf("%s\n", debug.Stack()))
			f.WriteString(fmt.Sprintf("%v", r))
		}
	}()
	screen.Clear()
	for _, ud := range g.SubUpdateDrawers() {
		ud.Draw(screen)
	}
	_, height := screen.Size()
	if g.askingQuitWithUnsavedChanges {
		top := &text.DrawOptions{
			LayoutOptions: text.LayoutOptions{
				SecondaryAlign: text.AlignEnd,
			},
		}
		top.GeoM.Translate(10, float64(height)-10)
		top.ColorM.Scale(1, 1, 1, 0.5)
		text.Draw(
			screen,
			fmt.Sprintf("Want to quit without save your changes? (y/n)"),
			&text.GoTextFace{
				Source: faceSource,
				Size:   16,
			},
			top,
		)
	}
}

func (g *Game) SubUpdateDrawers() []UpdateDrawer {
	// Only a Mode in the game is active.
	return []UpdateDrawer{g.Mode}
}

func (g *Game) SetBounds(b image.Rectangle) {
	g.bounds = b
}

func (g *Game) Bounds() image.Rectangle {
	return g.bounds
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func (g *Game) save() {
	f, err := os.Create(g.SaveFile)
	if err == nil {
		enc := gob.NewEncoder(f)
		data := &SaveData{
			WorldData: g.NormalMode.World.ToData(),
		}
		if err := enc.Encode(data); err != nil {
			// couldn't print in wsl with GOOS=windows
			e, _ := os.Create("err")
			e.WriteString(err.Error())
			e.Close()
		}
		f.Close()
	}
}

func main() {
	// get World from save data if exists
	windowBounds := image.Rect(0, 0, 640, 480)
	saveFile := "save"
	world := NewWorld()
	gob.Register(SaveData{})
	saved := &SaveData{}
	f, err := os.Open(saveFile)
	if err == nil {
		defer f.Close()
		dec := gob.NewDecoder(f)
		err = dec.Decode(saved)
		if err != nil {
			// couldn't print in wsl with GOOS=windows
			e, _ := os.Create("err")
			defer e.Close()
			e.WriteString(err.Error())
			return
		}
		world.FromData(saved.WorldData)
	}
	dirty := false
	normalMode := &NormalMode{
		bounds: windowBounds,
		World:  world,
		Dirty:  &dirty,
	}
	normalMode.WorldView = &WorldView{
		bounds:     image.Rect(10, 10, 10+8*tileSize*2, 10+6*tileSize*2),
		normalMode: normalMode,
		Camera:     NewCamera(pt(0, 0), pt(8, 6)),
	}
	normalMode.WorldView.Camera.FollowMargin = 2
	game := &Game{
		bounds:     windowBounds,
		Mode:       normalMode,
		NormalMode: normalMode,
		ZoomMode: &ZoomMode{
			bounds:     windowBounds,
			NormalMode: normalMode,
			Saturation: 255,
			Lightness:  128,
			Dirty:      &dirty,
		},
		Dirty:    &dirty,
		SaveFile: saveFile,
	}
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
