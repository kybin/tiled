package main

import (
	"fmt"
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

type Character struct {
	World      *World
	Level      int
	Pos        image.Point
	CopiedTile *Tile
	InZoomMode bool
	ZoomPos    image.Point // cannot exceed (tileSize, tileSize)
	MovingDir  image.Point
}

func (ch *Character) Move(pt image.Point) {
	ch.Pos.Add(pt)
}

func (ch *Character) SetLevel(l int) {
	ch.Level = l
}

func (ch *Character) CurrentTile() *Tile {
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
	zero := image.Point{}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.Char.InZoomMode = !g.Char.InZoomMode
		g.Char.MovingDir = zero
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyArrowUp) {
		g.Char.MovingDir = zero
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyArrowDown) {
		g.Char.MovingDir = zero
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyArrowLeft) {
		g.Char.MovingDir = zero
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyArrowRight) {
		g.Char.MovingDir = zero
	}
	if g.Char.InZoomMode {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			tile := g.Char.CurrentTile()
			if tile == nil {
				tile = &Tile{}
				tile.Image = image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
				g.World.Map[g.Char.Pos] = tile
			}
			zp := g.Char.ZoomPos
			tile.Image.Set(zp.X, zp.Y, color.RGBA{R: 255, A: 255})
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyG) {
			tile := g.Char.CurrentTile()
			if tile == nil {
				tile = &Tile{}
				tile.Image = image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
				g.World.Map[g.Char.Pos] = tile
			}
			zp := g.Char.ZoomPos
			tile.Image.Set(zp.X, zp.Y, color.RGBA{G: 255, A: 255})
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyB) {
			tile := g.Char.CurrentTile()
			if tile == nil {
				tile = &Tile{}
				tile.Image = image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
				g.World.Map[g.Char.Pos] = tile
			}
			zp := g.Char.ZoomPos
			tile.Image.Set(zp.X, zp.Y, color.RGBA{B: 255, A: 255})
			return nil
		}
		zp := g.Char.ZoomPos
		dir := image.Point{}
		if g.Char.MovingDir != zero {
			dir = g.Char.MovingDir
		} else {
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				dir = image.Pt(-1, 0)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				dir = image.Pt(1, 0)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				dir = image.Pt(0, -1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				dir = image.Pt(0, 1)
			}
		}
		zp = zp.Add(dir)
		if zp.In(image.Rect(0, 0, tileSize, tileSize)) {
			g.Char.MovingDir = dir
			g.Char.ZoomPos = zp
			return nil
		}
		// user go outside of the tile
		p := g.Char.Pos
		if zp.X < 0 {
			p = p.Add(image.Pt(-1, 0))
		}
		if zp.X >= tileSize {
			p = p.Add(image.Pt(1, 0))
		}
		if zp.Y < 0 {
			p = p.Add(image.Pt(0, -1))
		}
		if zp.Y >= tileSize {
			p = p.Add(image.Pt(0, 1))
		}
		if !p.In(g.World.Bound) {
			if zp.X < 0 {
				zp.X = 0
			}
			if zp.X >= tileSize {
				zp.X = tileSize - 1
			}
			if zp.Y < 0 {
				zp.Y = 0
			}
			if zp.Y >= tileSize {
				zp.Y = tileSize - 1
			}
		} else {
			// moved to a new tile
			g.Char.Pos = p
			if zp.X < 0 {
				zp.X = tileSize - 1
			}
			if zp.X >= tileSize {
				zp.X = 0
			}
			if zp.Y < 0 {
				zp.Y = tileSize - 1
			}
			if zp.Y >= tileSize {
				zp.Y = 0
			}
			g.Char.ZoomPos = zp
		}
	} else {
		// in normal mode
		if inpututil.IsKeyJustPressed(ebiten.KeyC) {
			g.Char.CopiedTile = g.Char.CurrentTile()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyV) {
			if g.Char.CopiedTile == nil {
				delete(g.World.Map, g.Char.Pos)
			} else {
				g.World.Map[g.Char.Pos] = g.Char.CopiedTile
			}
		}
		p := g.Char.Pos
		dir := image.Point{}
		if g.Char.MovingDir != zero {
			dir = g.Char.MovingDir
		} else {
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				dir = image.Pt(-1, 0)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				dir = image.Pt(1, 0)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				dir = image.Pt(0, -1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				dir = image.Pt(0, 1)
			}
		}
		p = p.Add(dir)
		if p.In(g.World.Bound) {
			g.Char.MovingDir = dir
			g.Char.Pos = p
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.input && g.introShown {
		return
	}
	g.introShown = true
	if g.Char.InZoomMode {
		screen.Clear()
		// draw zoomed tile
		zoomedTileSize := zoomScale * tileSize
		center := image.Pt(layoutWidth/2+1, layoutHeight/2+1)
		origin := image.Pt(center.X-zoomedTileSize/2, center.Y-zoomedTileSize/2)
		tileImage := ebiten.NewImage(tileSize, tileSize)
		tile := g.Char.CurrentTile()
		if tile != nil {
			fmt.Println(tile.Image.Pix)
			tileImage.WritePixels(tile.Image.Pix)
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoomScale, zoomScale)
		op.GeoM.Translate(float64(origin.X), float64(origin.Y))
		screen.DrawImage(tileImage, op)
		// draw cursor
		cursorImage := ebiten.NewImage(zoomScale, zoomScale)
		c := color.RGBA{R: 255, G: 255, B: 64, A: 128}
		for i := 0; i < zoomScale; i++ {
			cursorImage.Set(i, 0, c)
			cursorImage.Set(i, zoomScale-1, c)
		}
		for j := 0; j < zoomScale; j++ {
			cursorImage.Set(0, j, c)
			cursorImage.Set(zoomScale-1, j, c)
		}
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(origin.X)+float64(g.Char.ZoomPos.X)*zoomScale, float64(origin.Y)+float64(g.Char.ZoomPos.Y)*zoomScale)
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
	} else {
		screen.Clear()
		camRect := image.Rect(0, 0, layoutWidth, layoutHeight)
		tileImage := ebiten.NewImage(tileSize, tileSize)
		minPos := image.Pt(camRect.Min.X/tileSize, camRect.Min.Y/tileSize)
		maxPos := image.Pt(camRect.Max.X/tileSize, camRect.Max.Y/tileSize)
		for j := minPos.Y; j <= maxPos.Y; j++ {
			for i := minPos.X; i <= maxPos.X; i++ {
				tile := g.World.Map[image.Pt(i, j)]
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
		c := color.RGBA{R: 255, G: 255, B: 64, A: 255}
		for i := 0; i < tileSize; i++ {
			cursorImage.Set(i, 0, c)
			cursorImage.Set(i, tileSize-1, c)
		}
		for j := 0; j < tileSize; j++ {
			cursorImage.Set(0, j, c)
			cursorImage.Set(tileSize-1, j, c)
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(g.Char.Pos.X)*tileSize, float64(g.Char.Pos.Y)*tileSize)
		screen.DrawImage(cursorImage, op)
	}
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
