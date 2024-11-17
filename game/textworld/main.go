package main

import (
	"fmt"
	"time"
)

// Game is text based single user rpg.
type Game struct {
	Player *Player
	Fields map[string]*Field
	Drawer *Drawer
	Ticker *time.Ticker
}

func (g *Game) Name() string {
	return "텍스트월드"
}

func (g *Game) Run() {
	fmt.Println("텍스트월드에 오신것을 환영합니다!")
	g.Player.Enter(g.Fields["성문 앞"])
	for {
		select {
		case <-g.Ticker.C:
			g.Tick()
		}
	}
}

func (g *Game) Tick() {
	for _, e := range g.Player.Field.Events {
		fmt.Println(e)
	}
	g.Player.Field.Events = g.Player.Field.PendingEvents
	g.Player.Field.PendingEvents = make([]*Event, 0)
}

type Player struct {
	Character *Character
	Field     *Field
}

func (p *Player) Enter(f *Field) {
	p.Field = f
	p.Field.Events = append(p.Field.Events, &Event{Src: p.Character, Dest: p.Field, Action: ActionEnter})
	for _, npc := range f.NPCs {
		p.Field.PendingEvents = append(p.Field.PendingEvents, &Event{Src: npc, Dest: p.Character, Action: ActionLook})
	}
}

type Drawer struct{}

type Field struct {
	name          string
	Events        []*Event
	PendingEvents []*Event
	NPCs          []*Character
}

func (f *Field) Name() string {
	return f.name
}

type Character struct {
	name        string
	Personality *Personality
}

func (c *Character) Name() string {
	return c.name
}

type Personality struct {
	Ego      int
	Friendly int // <-> Aggressive
	Qurious  int
	Free     int
}

type Event struct {
	Src      Namer
	Dest     Namer
	Action   Action
	Age      time.Duration
	Lifetime time.Duration
}

type Namer interface {
	Name() string
}

func (e *Event) String() string {
	return fmt.Sprintf("%s이 %s을 %s", e.Src.Name(), e.Dest.Name(), e.Action)
}

type Action int

const (
	ActionEnter = Action(iota)
	ActionNotice
	ActionLook
	ActionSmile
	ActionQurious
)

func (a Action) String() string {
	if a == ActionEnter {
		return "들어왔습니다"
	}
	if a == ActionNotice {
		return "눈치챕니다"
	}
	if a == ActionLook {
		return "바라봅니다"
	}
	if a == ActionSmile {
		return "미소짓습니다"
	}
	if a == ActionQurious {
		return "갸우뚱 거립니다"
	}
	return ""
}

func main() {
	game := &Game{}
	pc := &Character{
		name: "당신",
	}
	player := &Player{
		Character: pc,
	}
	guard1 := &Character{
		name: "왼쪽 경비병",
	}
	guard2 := &Character{
		name: "오른쪽 경비병",
	}
	npcs := []*Character{
		guard1, guard2,
	}
	field := &Field{
		name: "성문 앞",
		NPCs: npcs,
	}
	fields := make(map[string]*Field)
	fields[field.name] = field
	game.Fields = fields
	game.Player = player
	game.Ticker = time.NewTicker(time.Second)
	game.Run()
}
