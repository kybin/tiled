package main

import (
	"fmt"
	"math/rand"
	"time"
)

// Game is text based single user rpg.
type Game struct {
	Player       *Player
	Fields       map[string]*Field
	Drawer       Drawer
	Ticker       *time.Ticker
	TickInterval time.Duration
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
		if g.Player.Character.Dead() {
			fmt.Println("당신은 사망했습니다.")
			return
		}
	}
}

func (g *Game) Tick() {
	if len(g.Player.Field.Events) != 0 {
		// Stop The World
		for _, e := range g.Player.Field.Events {
			fmt.Println(e)
		}
		g.Player.Field.Events = g.Player.Field.PendingEvents
		g.Player.Field.PendingEvents = make([]*Event, 0)
		return
	}
	cs := append(g.Player.Field.NPCs, g.Player.Character)
	for _, c := range cs {
		for _, s := range c.States {
			if s.Tick != nil {
				s.Tick(c)
			}
		}
	}
	for _, c := range cs {
		g.Drawer.DrawCharacter(c)
	}
	for _, c := range cs {
		for name, s := range c.States {
			if s.Temp || s.End != nil && s.End(c) {
				delete(c.States, name)
			}
		}
	}
}

type Drawer interface {
	DrawCharacter(c *Character)
}

type textDrawer struct{}

func (t *textDrawer) DrawCharacter(c *Character) {
	if c.States["bleed"] != nil {
		fmt.Printf("%s의 몸에서 피가 흘러 나옵니다.\n", c.Name())
		if c.States["bleed-damage"] != nil {
			fmt.Printf("%s은 %d 만큼의 데미지를 입었습니다. (남은체력 %d)\n", c.Name(), c.States["bleed-damage"].Value, c.States["HP"].Value)
		}
	}
	if c.States["stop-bleed"] != nil {
		fmt.Printf("%s의 몸에서 피가 흘러 나오기를 멈추었습니다.\n", c.Name())
	}
	if c.States["attack"] != nil {
		a := c.States["attack"]
		fmt.Printf("%s이 %s를 공격해 %d 만큼의 데미지를 입혔습니다. (남은체력 %d)\n", c.Name(), a.Target.Name(), a.Value, a.Target.States["HP"].Value)
	}
	if c.States["dead"] != nil {
		fmt.Printf("%s이 죽었습니다.\n", c.Name())
	}
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
	States      map[string]*State
	Personality *Personality
}

func (c *Character) Name() string {
	return c.name
}

func (c *Character) Dead() bool {
	return c.States["HP"].Value <= 0
}

func (c *Character) Attack(target *Character) {
	c.States["attack"] = &State{
		Name:   "attack",
		Target: target,
		Tick: func(c *Character) {
			damage := 1 + rand.Intn(6)
			c.States["attack"].Value = damage
			target.States["HP"].Value -= damage
			target.States["attacked"] = &State{
				Name:   "attacked",
				Target: c,
				Value:  damage,
				Temp:   true,
			}
		},
		End: func(c *Character) bool {
			if target.States["HP"].Value <= 0 {
				target.States["dead"] = &State{Temp: true}
				return true
			}
			return false
		},
	}
}

type Personality struct {
	Ego      int
	Friendly int // <-> Aggressive
	Qurious  int
	Free     int
}

type Namer interface {
	Name() string
}

type Event struct {
	Src      Namer
	Dest     Namer
	Action   Action
	Age      time.Duration
	Lifetime time.Duration
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
	if a == ActionQurious {
		return "갸우뚱 거립니다"
	}
	return ""
}

type State struct {
	Temp   bool
	Name   string
	Target *Character
	Value  int
	End    func(c *Character) bool
	Tick   func(c *Character)
}

func main() {
	game := &Game{
		Drawer: &textDrawer{},
	}
	pc := &Character{
		name:   "당신",
		States: make(map[string]*State),
	}
	stats := []State{
		State{Name: "HP", Value: 10},
		State{Name: "AttackSpeed", Value: 10},
		State{Name: "AttackDamage", Value: 10},
		State{Name: "DefenceDamage", Value: 10},
		State{Name: "bleed", Value: 1,
			Tick: func(c *Character) {
				d := rand.Intn(3) + 1
				c.States["bleed-damage"] = &State{Name: "bleed-damage", Value: d, Temp: true}
				c.States["HP"].Value -= d
				c.States["bleed"].Value -= 1
			},
			End: func(c *Character) bool {
				if c.States["bleed"].Value <= 0 {
					c.States["stop-bleed"] = &State{Temp: true}
					return true
				}
				return false
			},
		},
	}
	for _, stat := range stats {
		pc.States[stat.Name] = &stat
	}
	player := &Player{
		Character: pc,
	}
	guard1 := &Character{
		name:   "왼쪽 경비병",
		States: make(map[string]*State),
	}
	for _, stat := range stats {
		guard1.States[stat.Name] = &stat
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
	pc.Attack(guard1)
	fields := make(map[string]*Field)
	fields[field.name] = field
	game.Fields = fields
	game.Player = player
	game.TickInterval = 2 * time.Second
	game.Ticker = time.NewTicker(game.TickInterval)
	game.Run()
}
