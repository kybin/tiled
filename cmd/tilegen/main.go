package main

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
)

type Area struct {
	Name  string
	Rect  image.Rectangle
	Split *AreaSplit
}

func resize(a *Area, rect image.Rectangle) {
	a.Rect = rect
	if a.Split == nil {
		return
	}
	s := a.Split
	switch a.Split.Dir {
	case SplitLeft:
		x := min(rect.Min.X+s.Dist, rect.Max.X)
		resize(s.A, image.Rect(rect.Min.X, rect.Min.Y, x, rect.Max.Y))
		resize(s.B, image.Rect(x, rect.Min.Y, rect.Max.X, rect.Max.Y))
	case SplitRight:
		x := max(rect.Min.X, rect.Max.X-s.Dist)
		resize(s.A, image.Rect(rect.Min.X, rect.Min.Y, x, rect.Max.Y))
		resize(s.B, image.Rect(x, rect.Min.Y, rect.Max.X, rect.Max.Y))
	case SplitTop:
		y := min(rect.Min.Y+s.Dist, rect.Max.Y)
		resize(s.A, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, y))
		resize(s.B, image.Rect(rect.Min.X, y, rect.Max.X, rect.Max.Y))
	case SplitBottom:
		y := max(rect.Min.Y, rect.Max.Y-s.Dist)
		resize(s.A, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, y))
		resize(s.B, image.Rect(rect.Min.X, y, rect.Max.X, rect.Max.Y))
	}
}

type AreaSplit struct {
	Dir  SplitDir
	Dist int
	A    *Area // Top or Left according to SplitDir
	B    *Area // Bottom or Right according to SplitDir
}

type SplitDir int

const (
	SplitLeft = SplitDir(iota)
	SplitRight
	SplitTop
	SplitBottom
)

func main() {
	area := &Area{
		Name: "bg",
		Split: &AreaSplit{
			Dir:  SplitLeft,
			Dist: 100, // px
			A: &Area{
				Name: "side",
				Split: &AreaSplit{
					Dir:  SplitBottom,
					Dist: 100,
					A:    &Area{Name: "tileset"},
					B:    &Area{Name: "drawset"},
				},
			},
			B: &Area{Name: "playground"},
		},
	}
	rect := image.Rect(0, 0, 600, 400)
	resize(area, rect)
	js, err := json.MarshalIndent(area, "", "	")
	if err != nil {
		log.Println(err)
	}
	fmt.Println(string(js))
}
