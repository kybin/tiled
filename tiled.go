package tiled

type Pos [2]int

type Board struct {
	Width   int
	Height  int
	Tiles   []*Tile
	AllWays []string
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
