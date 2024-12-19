package asset

import "image"

type TileSet struct {
	Author   string
	License  string
	File     string
	TileSize int
	BaseImg  image.Image
}

var basictiles = &TileSet{
	Author:   "Lanea Zimmerman",
	License:  "anti-DRM clause of CC-BY 3.0",
	File:     "asset/basictiles.png",
	TileSize: 16,
}

var TileSets = map[string]*TileSet{
	"basictiles": basictiles,
}
