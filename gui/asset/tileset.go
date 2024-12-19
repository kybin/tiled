package asset

import "image"

type TileSet struct {
	Reference string
	Author    string
	License   string
	File      string
	TileSize  int
	BaseImg   image.Image
}

var basictiles = &TileSet{
	Reference: "https://opengameart.org/content/tiny-16-basic",
	Author:    "Lanea Zimmerman",
	License:   "anti-DRM clause of CC-BY 3.0",
	File:      "asset/basictiles.png",
	TileSize:  16,
}

var overworld = &TileSet{
	Reference: "https://opengameart.org/content/zelda-like-tilesets-and-sprites",
	Author:    "ArMM1998",
	License:   "CC0",
	File:      "asset/overworld.png",
	TileSize:  16,
}

var TileSets = map[string]*TileSet{
	"basictiles": basictiles,
	"overworld":  overworld,
}
