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
	maxStepTicks = 4
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

type Character struct {
	World      *World
	Level      int
	Pos        image.Point
	CopiedTile *Tile
	InZoomMode bool
	ZoomPos    image.Point // cannot exceed (tileSize, tileSize)
	IsMoving   bool
	MovingDir  image.Point
	stepTicks  int
}

func (ch *Character) Move(dir image.Point) {
	if dir == image.Pt(0, 0) {
		return
	}
	p := ch.Pos
	p = p.Add(dir)
	if !p.In(ch.World.Bound) {
		return
	}
	ch.Pos = p
}

func (ch *Character) ZoomMove(dir image.Point) {
	if dir == image.Pt(0, 0) {
		return
	}
	zp := ch.ZoomPos
	zp = zp.Add(dir)
	if zp.In(image.Rect(0, 0, tileSize, tileSize)) {
		ch.ZoomPos = zp
		return
	}
	// user go outside of the tile
	p := ch.Pos
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
	if !p.In(ch.World.Bound) {
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
		ch.Pos = p
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
		ch.ZoomPos = zp
	}
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
	World *World
	Char  *Character
}

func (g *Game) Update() error {
	keys := inpututil.AppendPressedKeys(nil)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.Char.InZoomMode = !g.Char.InZoomMode
	}
	if !g.Char.IsMoving {
		for _, k := range keys {
			d := keyDirection(k)
			if d == image.Pt(0, 0) {
				// not a key related with movement
				continue
			}
			g.Char.IsMoving = true
			g.Char.MovingDir = d
			break
		}
	}
	if g.Char.MovingDir == image.Pt(0, 0) {
		g.Char.stepTicks = 0
	} else {
		g.Char.stepTicks += 1
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
		if g.Char.stepTicks >= maxStepTicks {
			g.Char.ZoomMove(g.Char.MovingDir)
			g.Char.IsMoving = false
			g.Char.MovingDir = image.Pt(0, 0)
			g.Char.stepTicks = 0
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
		if g.Char.stepTicks >= maxStepTicks {
			g.Char.Move(g.Char.MovingDir)
			g.Char.IsMoving = false
			g.Char.MovingDir = image.Pt(0, 0)
			g.Char.stepTicks = 0
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
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
		x := float64(g.Char.ZoomPos.X) + float64(g.Char.MovingDir.X)*float64(g.Char.stepTicks)/maxStepTicks
		y := float64(g.Char.ZoomPos.Y) + float64(g.Char.MovingDir.Y)*float64(g.Char.stepTicks)/maxStepTicks
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
		x := float64(g.Char.Pos.X) + float64(g.Char.MovingDir.X)*float64(g.Char.stepTicks)/maxStepTicks
		y := float64(g.Char.Pos.Y) + float64(g.Char.MovingDir.Y)*float64(g.Char.stepTicks)/maxStepTicks
		op.GeoM.Translate(x*tileSize, y*tileSize)
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
	ebiten.SetTPS(20)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
