package tiled

import (
	"sort"
	"time"
)

type Pos [2]int
type Dir [2]int
type Size [2]int

type World struct {
	// Time will only passed on playing.
	// eg. Opening an UI will stop the world.
	Time time.Time
	// FPS controls speed of the game. When not defined, default FPS is 1.
	FPS    int
	Player *Player
	Field  map[string]*Field
}

func (w *World) ListenEvent() {
	if w.FPS <= 0 {
		w.FPS = 1
	}
	d := time.Duration(time.Second / w.FPS)
	ticker := time.NewTicker(d)
	for {
		select {
		case t := <-ticker.C:
			for _, t := range w.Player.Field.Board.Tiles {
				Tile.Occupier.Tick(d)
			}
		}
	}
}

type Field struct {
	Board *Board
	PC    *Character
	NPCs  []*Character
}

type Stage struct {
	// Board have Tiles with it's style, axis and sizes defined.
	Board *Board
	// Parties are all parties in the stage.
	Parties []*Party
	// ActiveParty is the party currently acting, either by user or AI.
	ActiveParty *Party
	// WaitedParties will act before the next party of ActiveParty.
	WaitedParties []*Party
	// Allies are ally parties to Party.
	Allies map[*Party][]*Party
	// DefaultStrategy is the default AI strategy for NPC.
	// It should be defined so it can be used when an NPC doesn't have distinctive strategy.
	DefaultStrategy *Strategy
}

func Win(*Player) bool      {}
func Defeated(*Player) bool {}
func NoMorePlayer() bool    {}
func CurrentTurn() int      {}
func MaxTurns() int         {}

func (s *Stage) TurnOver() {
	if len(s.WaitedParties) != 0 {
		w.ActiveParty = w.WaitedParties[0]
		w.WaitedParty = w.WaitedParty[1:]
	} else {
		nextPartyIdx := -1
		for i, p := range Parties {
			if p == w.ActiveParty {
				nextPartyIdx = i + 1
				break
			}
		}
		if nextPartyIdx == len(w.Parties) {
			nextPartyIdx = 0
		}
		w.ActiveParty = w.Parties[nextPartyIdx]
	}
}

type Board struct {
	Width  int
	Height int
	Ways   []Dir
	TileAt map[Pos]*Tile
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
