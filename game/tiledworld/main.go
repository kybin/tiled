package main

import (
	"encoding/gob"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	tileSize     = 16
	layoutWidth  = 320
	layoutHeight = 240
	zoomScale    = 8
	maxStepTicks = 3
)

type SaveData struct {
	WorldData *WorldData
}

type WorldData struct {
	Bound   image.Rectangle
	Map     map[image.Point]int
	GetTile map[int]*Tile
}

type World struct {
	Bound image.Rectangle
	Map   map[image.Point]*Tile
}

func NewWorld() *World {
	w := &World{
		Bound: image.Rect(0, 0, layoutWidth/tileSize, layoutHeight/tileSize),
		Map:   make(map[image.Point]*Tile),
	}
	return w
}

func (w *World) ToData() *WorldData {
	d := &WorldData{
		Bound:   w.Bound,
		Map:     make(map[image.Point]int),
		GetTile: make(map[int]*Tile),
	}
	tileID := make(map[*Tile]int)
	for p, tile := range w.Map {
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
		d.Map[p] = id
	}
	return d
}

func (w *World) FromData(d *WorldData) {
	w.Bound = d.Bound
	for p, id := range d.Map {
		t := d.GetTile[id]
		w.Map[p] = t
	}
}

func (w *World) NewTile(p image.Point) *Tile {
	w.ClearTile(p)
	tile := &Tile{}
	tile.Image = image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
	w.Map[p] = tile
	return tile
}

func (w *World) ClearTile(p image.Point) {
	delete(w.Map, p)
	// TODO: clear the tile when all its references are gone
}

func (w *World) DuplicateTile(from image.Point, to image.Point) {
	tile, ok := w.Map[from]
	if !ok {
		w.ClearTile(to)
		return
	}
	w.Map[to] = tile
}

func (w *World) MakeTileUnique(p image.Point) {
	old, ok := w.Map[p]
	if !ok {
		return
	}
	tile := w.NewTile(p)
	draw.Draw(tile.Image, tile.Image.Bounds(), old.Image, image.Pt(0, 0), draw.Src)
}

func (w *World) TileAt(p image.Point) *Tile {
	return w.Map[p]
}

func (w *World) TilePoses(tile *Tile) []image.Point {
	pts := make([]image.Point, 0)
	for pt, t := range w.Map {
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
	Pos       image.Point
	IsMoving  bool
	MovingDir image.Point
	stepTicks int
}

func (m *Mover) ActionPos() image.Point {
	return m.Pos.Add(m.MovingDir)
}

type Mode interface {
	Update() error
	Draw(*ebiten.Image)
}

type NormalMode struct {
	Mover
	World       *World
	copyTilePos image.Point
}

func (m *NormalMode) Move(dir image.Point) {
	if dir == image.Pt(0, 0) {
		return
	}
	p := m.Pos
	p = p.Add(dir)
	if !p.In(m.World.Bound) {
		return
	}
	m.Pos = p
}

func (m *NormalMode) NewTile() *Tile {
	return m.World.NewTile(m.ActionPos())
}

func (m *NormalMode) ActionTile() *Tile {
	return m.World.TileAt(m.ActionPos())
}

func (m *NormalMode) ClearTile() {
	m.World.ClearTile(m.ActionPos())
}

func (m *NormalMode) CopyTile() {
	m.copyTilePos = m.ActionPos()
}

func (m *NormalMode) PasteTile() {
	m.World.DuplicateTile(m.copyTilePos, m.ActionPos())
}

func (m *NormalMode) MakeTileUnique() {
	m.World.MakeTileUnique(m.ActionPos())
}

func (m *NormalMode) Update() error {
	keys := inpututil.AppendPressedKeys(nil)
	for _, k := range keys {
		if k == ebiten.KeyX {
			m.ClearTile()
			continue
		}
		if k == ebiten.KeyC {
			m.CopyTile()
			continue
		}
		if k == ebiten.KeyV {
			m.PasteTile()
			continue
		}
		if k == ebiten.KeyD {
			m.MakeTileUnique()
			continue
		}
		if !m.IsMoving {
			d := keyDirection(k)
			if d != image.Pt(0, 0) {
				m.IsMoving = true
				m.MovingDir = d
				continue
			}
		}
	}
	if m.MovingDir == image.Pt(0, 0) {
		m.stepTicks = 0
	} else {
		m.stepTicks += 1
	}
	if m.stepTicks >= maxStepTicks {
		m.Move(m.MovingDir)
		m.IsMoving = false
		m.MovingDir = image.Pt(0, 0)
		m.stepTicks = 0
	}
	return nil
}

func (m *NormalMode) Draw(screen *ebiten.Image) {
	camRect := image.Rect(0, 0, layoutWidth, layoutHeight)
	tileImage := ebiten.NewImage(tileSize, tileSize)
	minPos := image.Pt(camRect.Min.X/tileSize, camRect.Min.Y/tileSize)
	maxPos := image.Pt(camRect.Max.X/tileSize, camRect.Max.Y/tileSize)
	for j := minPos.Y; j <= maxPos.Y; j++ {
		for i := minPos.X; i <= maxPos.X; i++ {
			tile, ok := m.World.Map[image.Pt(i, j)]
			if ok {
				tileImage.WritePixels(tile.Image.Pix)
			} else {
				tileImage.Clear()
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(i)*tileSize, float64(j)*tileSize)
			screen.DrawImage(tileImage, op)
		}
	}
	// draw cursor
	cursorImage := ebiten.NewImage(tileSize, tileSize)
	c := color.RGBA{R: 192, G: 192, B: 64, A: 128}
	drawOutline(cursorImage, cursorImage.Bounds(), c)
	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	x := float64(m.Pos.X) + float64(m.MovingDir.X)*float64(m.stepTicks)/maxStepTicks
	y := float64(m.Pos.Y) + float64(m.MovingDir.Y)*float64(m.stepTicks)/maxStepTicks
	op.GeoM.Translate(x*tileSize, y*tileSize)
	screen.DrawImage(cursorImage, op)
	// draw copy cursor
	cursorImage.Clear()
	c = color.RGBA{R: 64, G: 64, B: 192, A: 128}
	drawOutline(cursorImage, cursorImage.Bounds(), c)
	op = &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	op.GeoM.Translate(float64(m.copyTilePos.X)*tileSize, float64(m.copyTilePos.Y)*tileSize)
	screen.DrawImage(cursorImage, op)
	// draw all matching cursor
	cursorImage.Clear()
	c = color.RGBA{R: 32, G: 32, B: 32, A: 32}
	drawOutline(cursorImage, cursorImage.Bounds(), c)
	op = &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	for _, p := range m.World.TilePoses(m.ActionTile()) {
		op.GeoM.Reset()
		op.GeoM.Translate(float64(p.X)*tileSize, float64(p.Y)*tileSize)
		screen.DrawImage(cursorImage, op)
	}
}

type ZoomMode struct {
	NormalMode *NormalMode
	Mover
	Hue        int
	Saturation int
	Lightness  int
}

func (m *ZoomMode) Move(dir image.Point) {
	if dir == image.Pt(0, 0) {
		return
	}
	p := m.Pos
	p = p.Add(dir)
	if p.In(image.Rect(0, 0, tileSize, tileSize)) {
		m.Pos = p
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
	if !np.In(m.NormalMode.World.Bound) {
		if p.X < 0 {
			p.X = 0
		}
		if p.X >= tileSize {
			p.X = tileSize - 1
		}
		if p.Y < 0 {
			p.Y = 0
		}
		if p.Y >= tileSize {
			p.Y = tileSize - 1
		}
	} else {
		// moved to a new tile
		m.NormalMode.Pos = np
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
}

func (m *ZoomMode) Update() error {
	keys := inpututil.AppendPressedKeys(nil)
	alt := false
	for _, k := range keys {
		if k == ebiten.KeyAlt {
			alt = true
			break
		}
	}
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
	} else {
		for _, k := range keys {
			if !m.IsMoving {
				d := keyDirection(k)
				if d != image.Pt(0, 0) {
					m.IsMoving = true
					m.MovingDir = d
				}
			}
			if k == ebiten.KeyX {
				tile := m.NormalMode.ActionTile()
				if tile != nil {
					p := m.ActionPos()
					tile.Image.Set(p.X, p.Y, color.RGBA{})
				}
			}
			if k == ebiten.KeyC {
				tile := m.NormalMode.ActionTile()
				if tile == nil {
					tile = m.NormalMode.NewTile()
				}
				p := m.ActionPos()
				c, _ := tile.Image.At(p.X, p.Y).(color.RGBA)
				if c.A != 0 {
					h, s, l := RGBToHSL(c)
					m.Hue = int(h * 255)
					m.Saturation = int(s * 255)
					m.Lightness = int(l * 255)
				}
			}
			if k == ebiten.KeyV {
				tile := m.NormalMode.ActionTile()
				if tile == nil {
					tile = m.NormalMode.NewTile()
				}
				p := m.ActionPos()
				c := HSLToRGB(float64(m.Hue)/255, float64(m.Saturation)/255, float64(m.Lightness)/255)
				tile.Image.Set(p.X, p.Y, c)
			}
		}
	}
	if m.MovingDir == image.Pt(0, 0) {
		m.stepTicks = 0
	} else {
		m.stepTicks += 1
	}
	if m.stepTicks >= maxStepTicks {
		m.Move(m.MovingDir)
		m.IsMoving = false
		m.MovingDir = image.Pt(0, 0)
		m.stepTicks = 0
	}
	return nil
}

func (m *ZoomMode) Draw(screen *ebiten.Image) {
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
	x := float64(m.Pos.X) + float64(m.MovingDir.X)*float64(m.stepTicks)/maxStepTicks
	y := float64(m.Pos.Y) + float64(m.MovingDir.Y)*float64(m.stepTicks)/maxStepTicks
	op.GeoM.Translate(float64(origin.X)+x*zoomScale, float64(origin.Y)+y*zoomScale)
	screen.DrawImage(cursorImage, op)
	// draw outline
	c = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	b := image.Rectangle{}
	b.Min = origin.Sub(image.Pt(1, 1))
	b.Max = origin.Add(image.Pt(zoomedTileSize+1, zoomedTileSize+1))
	drawOutline(screen, b, c)
}

type Character struct {
	Mode       Mode
	NormalMode *NormalMode
	ZoomMode   *ZoomMode
}

type Game struct {
	Char     *Character
	SaveFile string
}

func (g *Game) Update() error {
	keys := inpututil.AppendPressedKeys(nil)
	ctrl := false
	for _, k := range keys {
		if k == ebiten.KeyControl {
			ctrl = true
		}
	}
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		f, err := os.Create(g.SaveFile)
		if err == nil {
			enc := gob.NewEncoder(f)
			data := &SaveData{
				WorldData: g.Char.NormalMode.World.ToData(),
			}
			if err := enc.Encode(data); err != nil {
				// couldn't print in wsl with GOOS=windows
				e, _ := os.Create("err")
				e.WriteString(err.Error())
				e.Close()
			}
			f.Close()
		}
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if g.Char.Mode == g.Char.NormalMode {
			g.Char.Mode = g.Char.ZoomMode
		} else {
			g.Char.Mode = g.Char.NormalMode
		}
	}
	return g.Char.Mode.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Clear()
	g.Char.Mode.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return layoutWidth, layoutHeight
}

func main() {
	game := &Game{
		SaveFile: "save",
	}
	// get World from save data if exists
	world := NewWorld()
	gob.Register(SaveData{})
	saved := &SaveData{}
	f, err := os.Open(game.SaveFile)
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
	normalMode := &NormalMode{
		World: world,
	}
	ch := &Character{
		Mode:       normalMode,
		NormalMode: normalMode,
		ZoomMode: &ZoomMode{
			NormalMode: normalMode,
			Saturation: 255,
			Lightness:  128,
		},
	}
	game.Char = ch
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Tiled World")
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetTPS(20)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
