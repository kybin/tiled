package class

import (
	"github.com/kybin/tiled"
	"github.com/kybin/tiled/game/example"
)

type Knockback struct {
	Caster *tiled.Character
}

func (a *Knockback) Origin(cursor *tiled.Tile) tiled.Tile {
	return a.Caster.At()
}

func (a *Knockback) SelectableArea() tiled.Area {
	return example.AxisArea
}

func (a *Knockback) CastArea(sel tiled.Pos) tiled.Area {
	return tiled.CreateArea([]tiled.Pos{sel})
}

func (a *Knockback) Cast(tiles []*tiled.Tile) {
	for _, t := range tiles {
		ch := t.Occupier
		if ch.Party.IsAlly(a.Caster.Party) {
			continue
		}
		if ch.At()[0] < a.Caster.At()[0] {
			ch.Pos[0] = ch.Pos[0] - 1
		}
		if ch.At()[0] > a.Caster.At()[0] {
			ch.Pos[0] = ch.Pos[0] + 1
		}
		if ch.At()[1] < a.Caster.At()[1] {
			ch.Pos[1] = ch.Pos[1] - 1
		}
		if ch.At()[1] > a.Caster.At()[1] {
			ch.Pos[1] = ch.Pos[1] + 1
		}
		ch.HP -= a.Caster.HP * a.Caster.AttackPower / 3
	}
}

func NewSwordman(ch *tiled.Character) *tiled.Class {
	cls := &tiled.Class{}
	cls.Skill["attack"] = tiled.Skill(&SwordAttack{Caster: ch})
	cls.Skill["knockback"] = tiled.Skill(&Knockback{Caster: ch})
}

type SwordAttack struct {
	Caster *tiled.Character
}

func (a *SwordAttack) Origin(cursor *tiled.Tile) tiled.Tile {
	return a.Caster.At()
}

func (a *SwordAttack) SelectableArea() tiled.Area {
	return example.AxisArea
}

func (a *SwordAttack) CastArea(sel tiled.Pos) tiled.Area {
	return tiled.CreateArea([]tiled.Pos{sel})
}

func (a *SwordAttack) Cast(tiles []*tiled.Tile) {
	for _, t := range tiles {
		ch := t.Occupier
		if ch.Party.IsAlly(a.Caster.Party) {
			continue
		}
		ch.HP -= a.Caster.HP * a.Caster.AttackPower
	}
}

func NewSpearman(ch *tiled.Character) *tiled.Class {
	cls := &tiled.Class{}
	cls.Skill["attack"] = tiled.Skill(&SpearAttack{Caster: ch})
	cls.Skill["knockback"] = tiled.Skill(&Knockback{Caster: ch})
}

type SpearAttack struct {
	Caster *tiled.Character
}

func (a *SpearAttack) Origin(cursor *tiled.Tile) tiled.Tile {
	return a.Caster.At()
}

func (a *SpearAttack) SelectableArea() tiled.Area {
	return example.AxisArea.Add(example.AxisArea2)
}

func (a *SpearAttack) CastArea(sel tiled.Pos) tiled.Area {
	return tiled.CreateArea([]tiled.Pos{sel})
}

func (a *SpearAttack) Cast(tiles []*tiled.Tile) {
	for _, t := range tiles {
		ch := t.Occupier
		if ch.Party.IsAlly(a.Caster.Party) {
			continue
		}
		k := 0.5
		if tile.Distance(ch.At(), a.Caster.At()) > 1 {
			k := 1
		}
		ch.HP -= a.Caster.HP * a.Caster.AttackPower * k
	}
}
