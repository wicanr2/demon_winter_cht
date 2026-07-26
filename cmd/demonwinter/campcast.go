package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 紮營選單的 Cast（規則在 internal/game/campcast.go，出處 docs/re/42）。
//
// 四步：選施術者 → 選法術 → 投入法力 → 選目標。
// 只列得出**在戰鬥外有意義**的法術（治療、回魔、解束縛、枯萎）——
// 那四種「戰鬥用增減」在營地放等於什麼都沒做，不放進清單。

// castScreen 是營地施法的狀態。
type castScreen struct {
	// step 是目前在哪一步。
	step castStep
	// caster／target 是施術者與目標的索引，spell 是法術清單的游標。
	caster, target, spell int
	// entries 是這一次選得到的法術。
	entries []spellEntry
	// power 是投入的法力，用 ↑↓ 調 —— 與紮營其餘畫面同一套操作。
	power int
}

type castStep int

const (
	castPickCaster castStep = iota
	castPickSpell
	castPickPower
	castPickTarget
)

func (a *app) openCampCast() {
	if len(a.members) == 0 {
		a.camp.message = "隊伍是空的"
		return
	}
	c := &castScreen{}
	for _, i := range game.CampCastCandidates(a.tables) {
		sp, err := a.tables.Spell(i)
		if err != nil {
			continue
		}
		name, err := a.strings.SpellName(i)
		if err != nil {
			continue
		}
		c.entries = append(c.entries, spellEntry{
			index: i, name: a.tr.Event(spellSourceFile, i, name), spell: sp,
		})
	}
	if len(c.entries) == 0 {
		a.camp.message = "沒有能在營地放的法術"
		return
	}
	a.camp.message = ""
	a.camp.cast = c
}

func (a *app) updateCampCast() error {
	c := a.camp.cast

	switch c.step {
	case castPickCaster:
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			a.camp.cast = nil
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			c.caster = (c.caster + 1) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			c.caster = (c.caster - 1 + len(a.members)) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			c.step = castPickSpell
		}

	case castPickSpell:
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			c.step = castPickCaster
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			c.spell = (c.spell + 1) % len(c.entries)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			c.spell = (c.spell - 1 + len(c.entries)) % len(c.entries)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			// 預設就投最低需求 —— 一按 Enter 就能放，不必先加半天。
			c.power = c.entries[c.spell].spell.M
			c.step = castPickPower
		}

	case castPickPower:
		maxSP := a.members[c.caster].CurrentSP
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			c.step = castPickSpell
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			if c.power < maxSP {
				c.power++
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			if c.power > 0 {
				c.power--
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			c.target = c.caster
			c.step = castPickTarget
		}

	case castPickTarget:
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			c.step = castPickPower
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			c.target = (c.target + 1) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			c.target = (c.target - 1 + len(a.members)) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			a.doCampCast(c)
		}
	}
	return nil
}

func (a *app) doCampCast(c *castScreen) {
	caster, target := &a.members[c.caster], &a.members[c.target]
	e := c.entries[c.spell]

	res := game.CampCast(a.rng, caster, target, e.spell, c.power)
	if !res.OK {
		a.camp.message = res.Reason
		return
	}
	msg := fmt.Sprintf("%s 對 %s 施放%s", caster.Name, target.Name, e.name)
	switch {
	case res.Delta > 0:
		msg += fmt.Sprintf("　回復 %d", res.Delta)
	case res.Delta < 0:
		msg += fmt.Sprintf("　造成 %d", -res.Delta)
	case res.Released:
		msg += "　束縛解開了"
	case res.Withered:
		msg += "　枯萎生效"
	default:
		msg += "　沒有明顯的變化"
	}
	if res.Died {
		msg += "　" + target.Name + " 倒下了"
	}
	a.camp.message = msg
	a.camp.cast = nil
}

func (a *app) drawCampCast(line func(string)) {
	c := a.camp.cast

	switch c.step {
	case castPickCaster:
		line("誰來施法？")
		line("")
		a.drawMemberList(line, c.caster, func(i int) string {
			return fmt.Sprintf("法力 %d/%d", a.members[i].CurrentSP, a.members[i].MaxSP)
		})
		line("")
		line("↑↓：選擇　Enter：確定　Esc：取消")

	case castPickSpell:
		line(fmt.Sprintf("%s 要放什麼？", a.members[c.caster].Name))
		line("")
		for i, e := range c.entries {
			mark := "   "
			if i == c.spell {
				mark = " > "
			}
			line(fmt.Sprintf("%s%s最低 %d 點",
				mark, textlayout.PadCells(e.name, 14), e.spell.M))
		}
		line("")
		line("↑↓：選擇　Enter：確定　Esc：返回")

	case castPickPower:
		e := c.entries[c.spell]
		line(fmt.Sprintf("%s：投入多少法力？", e.name))
		line("")
		line(fmt.Sprintf("　%d（最低 %d，最多 %d）",
			c.power, e.spell.M, a.members[c.caster].CurrentSP))
		line("")
		line("↑↓：加減　Enter：確定　Esc：返回")

	default:
		line(fmt.Sprintf("%s 的%s要放在誰身上？",
			a.members[c.caster].Name, c.entries[c.spell].name))
		line("")
		a.drawMemberList(line, c.target, func(i int) string {
			m := a.members[i]
			return fmt.Sprintf("生命 %d/%d　法力 %d/%d",
				m.CurrentHP, m.MaxHP, m.CurrentSP, m.MaxSP)
		})
		line("")
		line("↑↓：選擇　Enter：施放　Esc：返回")
	}

	if a.camp.message != "" {
		line("")
		line(a.camp.message)
	}
}
