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
	Img image.RGBA
}

type Character struct {
	World      *World
	Level      int
	Pos        image.Point
	CopiedTile *Tile
}

func (ch *Character) Move(pt image.Point) {
	ch.Pos.Add(pt)
}

func (ch *Character) SetLevel(l int) {
	ch.Level = l
}

func (ch *Character) StandingTile() *Tile {
	return ch.World.Map[ch.Pos]
}

func (ch *Character) CopyTile() {
	ch.CopiedTile = ch.World.Map[ch.Pos]
}

func (ch *Character) PasteTile() {
	ch.World.Map[ch.Pos] = ch.CopiedTile
}

type Game struct {
	input      bool
	introShown bool
	World      *World
	Char       *Character
}

func (g *Game) Update() error {
	g.input = len(inpututil.AppendPressedKeys(nil)) > 0
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return ebiten.Termination
	}
	p := g.Char.Pos
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		p = p.Add(image.Pt(-1, 0))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		p = p.Add(image.Pt(1, 0))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		p = p.Add(image.Pt(0, -1))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		p = p.Add(image.Pt(0, 1))
	}
	if p.In(g.World.Bound) {
		g.Char.Pos = p
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.input && g.introShown {
		return
	}
	g.introShown = true
	camRect := image.Rect(0, 0, layoutWidth, layoutHeight)
	tile := ebiten.NewImage(tileSize, tileSize)
	c := color.RGBA{R: 255, G: 128, B: 128, A: 255}
	for i := 0; i < tileSize; i++ {
		tile.Set(i, 0, c)
		tile.Set(i, tileSize-1, c)
	}
	for j := 0; j < tileSize; j++ {
		tile.Set(0, j, c)
		tile.Set(tileSize-1, j, c)
	}
	minPos := image.Pt(camRect.Min.X/tileSize, camRect.Min.Y/tileSize)
	maxPos := image.Pt(camRect.Max.X/tileSize, camRect.Max.Y/tileSize)
	for j := minPos.Y; j <= maxPos.Y; j++ {
		for i := minPos.X; i <= maxPos.X; i++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(i)*tileSize, float64(j)*tileSize)
			screen.DrawImage(tile, op)
		}
	}
	tile = ebiten.NewImage(tileSize, tileSize)
	c = color.RGBA{R: 255, G: 255, B: 64, A: 255}
	for i := 0; i < tileSize; i++ {
		tile.Set(i, 0, c)
		tile.Set(i, tileSize-1, c)
	}
	for j := 0; j < tileSize; j++ {
		tile.Set(0, j, c)
		tile.Set(tileSize-1, j, c)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(g.Char.Pos.X)*tileSize, float64(g.Char.Pos.Y)*tileSize)
	screen.DrawImage(tile, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return layoutWidth, layoutHeight
}

func main() {
	world := NewWorld()
	ch := &Character{
		World: world,
	}
	game := &Game{}
	game.World = world
	game.Char = ch
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Tiled World")
	ebiten.SetScreenClearedEveryFrame(false)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
