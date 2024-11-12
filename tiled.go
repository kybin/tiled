package tiled

import "sort"

type Pos [2]int
type Dir [2]int

type Board struct {
	Width  int
	Height int
	Axis   []Dir
	Tiles  []*Tile
	Tile   map[Pos]*Tile
}

type Tile struct {
	Pos      Pos
	Base     *BaseTile
	Occupier *Character
	Ways     []*Way
}

type Way struct {
	Name string
	From *Tile
	To   *Tile
	Cost int
}

func (t *Tile) Way(name string) *Way {
	for _, w := range t.Ways {
		if w.Name == name {
			return w
		}
	}
	return nil
}

// Area is composed with Poses in tiled world.
// Adding an existing Pos do nothing to the area.
// Removing a non-existing Pos do nothing as well.
type Area struct {
	pos map[Pos]bool
}

func CreateArea(poses []Pos) Area {
	a := &Area{}
	for _, p := range poses {
		a.pos[m] = true
	}
	return a
}

func (a Area) Add(poses []Pos) Area {
	for _, p := range poses {
		a.pos[m] = true
	}
	return a
}

func (a Area) Sub(poses []Pos) Area {
	for _, p := range poses {
		delete(a.pos, p)
	}
	return a
}

func (a Area) Poses() []Pos {
	poses := make([]Pos, 0, len(a.pos))
	for _, p := range a.pos {
		poses = append(poses, p)
	}
	sort.Slice(poses, func(i, j int) bool {
		if poses[i][0] < poses[j][0] {
			return true
		}
		if poses[i][0] > poses[j][0] {
			return false
		}
		if poses[i][1] < poses[j][1] {
			return true
		}
		return false
	})
	return poses
}
