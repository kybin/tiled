package main

import (
	"image"
	"image/color"
	"log"

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

type World struct {
	Bound image.Rectangle
	Map   Map
}

func NewWorld() *World {
	w := &World{
		Bound: image.Rect(0, 0, layoutWidth/tileSize, layoutHeight/tileSize),
		Map:   make(map[image.Point]*Tile),
	}
	return w
}

type Map map[image.Point]*Tile

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
	World      *World
	CopiedTile *Tile
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

func (m *NormalMode) CurrentTile() *Tile {
	return m.World.Map[m.ActionPos()]
}

func (m *NormalMode) CopyTile() {
	m.CopiedTile = m.CurrentTile()
}

func (m *NormalMode) PasteTile() {
	m.World.Map[m.ActionPos()] = m.CopiedTile
}

func (m *NormalMode) Update() error {
	keys := inpututil.AppendPressedKeys(nil)
	for _, k := range keys {
		if k == ebiten.KeyC {
			m.CopiedTile = m.CurrentTile()
			continue
		}
		if k == ebiten.KeyV {
			if m.CopiedTile == nil {
				delete(m.World.Map, m.Pos)
			} else {
				m.PasteTile()
			}
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
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		m.CopiedTile = m.CurrentTile()
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
			tile := m.World.Map[image.Pt(i, j)]
			if tile != nil {
				tileImage.WritePixels(tile.Image.Pix)
			} else {
				tileImage.Clear()
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(i)*tileSize, float64(j)*tileSize)
			screen.DrawImage(tileImage, op)
		}
	}
	cursorImage := ebiten.NewImage(tileSize, tileSize)
	c := color.RGBA{R: 192, G: 192, B: 64, A: 128}
	for i := 0; i < tileSize; i++ {
		cursorImage.Set(i, 0, c)
		cursorImage.Set(i, tileSize-1, c)
	}
	for j := 0; j < tileSize; j++ {
		cursorImage.Set(0, j, c)
		cursorImage.Set(tileSize-1, j, c)
	}
	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	x := float64(m.Pos.X) + float64(m.MovingDir.X)*float64(m.stepTicks)/maxStepTicks
	y := float64(m.Pos.Y) + float64(m.MovingDir.Y)*float64(m.stepTicks)/maxStepTicks
	op.GeoM.Translate(x*tileSize, y*tileSize)
	screen.DrawImage(cursorImage, op)
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
			if k == ebiten.KeyArrowDown {
				m.Saturation = max(m.Saturation-16, 1)
			}
			if k == ebiten.KeyArrowUp {
				m.Saturation = min(m.Saturation+16, 255)
			}
			if k == ebiten.KeyMinus {
				m.Lightness = max(m.Lightness-16, 1)
			}
			if k == ebiten.KeyEqual {
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
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		tile := m.NormalMode.CurrentTile()
		if tile == nil {
			tile = &Tile{}
			tile.Image = image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
			m.NormalMode.World.Map[m.NormalMode.Pos] = tile
		}
		p := m.ActionPos()
		c := HSLToRGB(float64(m.Hue)/255, float64(m.Saturation)/255, float64(m.Lightness)/255)
		tile.Image.Set(p.X, p.Y, c)
	}
	return nil
}

func (m *ZoomMode) Draw(screen *ebiten.Image) {
	colorPickerSize := 32
	colorPicker := ebiten.NewImage(colorPickerSize, colorPickerSize)
	c := HSLToRGB(float64(m.Hue)/255, float64(m.Saturation)/255, float64(m.Lightness)/255)
	for h := 0; h < colorPickerSize; h += 1 {
		for s := 0; s < colorPickerSize; s += 1 {
			colorPicker.Set(h, colorPickerSize-s-1, c)
		}
	}
	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(colorPicker, op)
	// draw zoomed tile
	zoomedTileSize := zoomScale * tileSize
	center := image.Pt(layoutWidth/2+1, layoutHeight/2+1)
	origin := image.Pt(center.X-zoomedTileSize/2, center.Y-zoomedTileSize/2)
	tileImage := ebiten.NewImage(tileSize, tileSize)
	tile := m.NormalMode.CurrentTile()
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
	for i := 0; i < zoomScale; i++ {
		cursorImage.Set(i, 0, c)
		cursorImage.Set(i, zoomScale-1, c)
	}
	for j := 0; j < zoomScale; j++ {
		cursorImage.Set(0, j, c)
		cursorImage.Set(zoomScale-1, j, c)
	}
	op = &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	x := float64(m.Pos.X) + float64(m.MovingDir.X)*float64(m.stepTicks)/maxStepTicks
	y := float64(m.Pos.Y) + float64(m.MovingDir.Y)*float64(m.stepTicks)/maxStepTicks
	op.GeoM.Translate(float64(origin.X)+x*zoomScale, float64(origin.Y)+y*zoomScale)
	screen.DrawImage(cursorImage, op)
	// draw outline
	outlineImage := ebiten.NewImage(1, 1)
	c = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	outlineImage.Set(0, 0, c)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1, float64(zoomedTileSize)+2)
	op.GeoM.Translate(float64(origin.X)-1, float64(origin.Y)-1)
	screen.DrawImage(outlineImage, op)
	op.GeoM.Translate(float64(zoomedTileSize)+1, 0)
	screen.DrawImage(outlineImage, op)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(zoomedTileSize)+2, 1)
	op.GeoM.Translate(float64(origin.X)-1, float64(origin.Y)-1)
	screen.DrawImage(outlineImage, op)
	op.GeoM.Translate(0, float64(zoomedTileSize)+1)
	screen.DrawImage(outlineImage, op)
}

type Character struct {
	Mode       Mode
	NormalMode *NormalMode
	ZoomMode   *ZoomMode
}

type Game struct {
	Char *Character
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
	world := NewWorld()
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
	game := &Game{}
	game.Char = ch
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Tiled World")
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetTPS(20)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
