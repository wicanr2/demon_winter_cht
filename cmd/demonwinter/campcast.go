package main

import (
	"fmt"
	"strings"

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

// worshipScreen 是敬拜的狀態：選祈求者 → 選法術落在誰身上。
type worshipScreen struct {
	caster, target int
	// picked 為 true 代表已經選完祈求者，正在選目標。
	picked bool
}

func (a *app) openWorship() {
	if len(a.members) == 0 {
		a.camp.message = a.tr.UI("campcast.worship.empty")
		return
	}
	a.camp.message = ""
	a.camp.worship = &worshipScreen{}
}

func (a *app) updateWorship() error {
	w := a.camp.worship
	cursor := &w.caster
	if w.picked {
		cursor = &w.target
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		if w.picked {
			w.picked = false
			return nil
		}
		a.camp.worship = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		*cursor = (*cursor + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		*cursor = (*cursor - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if !w.picked {
			if ok, why := game.CanWorship(&a.members[w.caster]); !ok {
				a.camp.message = a.reasonText(why)
				return nil
			}
			w.target, w.picked = w.caster, true
			return nil
		}
		a.doWorship(w)
	}
	return nil
}

func (a *app) doWorship(w *worshipScreen) {
	caster, target := &a.members[w.caster], &a.members[w.target]
	res := game.Worship(a.rng, a.tables, caster, target)
	if !res.OK {
		a.camp.message = a.reasonText(res.Reason)
		return
	}
	if !res.Answered {
		a.camp.message = fmt.Sprintf(a.tr.UI("campcast.worship.noanswer"),
			caster.Name, res.Chance)
		a.camp.worship = nil
		return
	}

	name := fmt.Sprintf(a.tr.UI("campcast.worship.spellname"), res.SpellID)
	if n, err := a.strings.SpellName(res.SpellID); err == nil {
		name = a.tr.Event(spellSourceFile, res.SpellID, n)
	}
	msg := fmt.Sprintf(a.tr.UI("campcast.worship.answered")+"\n",
		a.deityName(caster.Deity), caster.Name, name)
	switch {
	case !res.Cast.OK:
		msg += "　" + a.reasonText(res.Cast.Reason)
	case res.Cast.Delta > 0:
		msg += fmt.Sprintf(a.tr.UI("campcast.worship.heal"), target.Name, res.Cast.Delta)
	case res.Cast.Delta < 0:
		msg += fmt.Sprintf(a.tr.UI("campcast.worship.damage"), target.Name, -res.Cast.Delta)
	case res.Cast.Released:
		msg += a.tr.UI("campcast.worship.released")
	case res.Cast.Withered:
		msg += a.tr.UI("campcast.worship.withered")
	}
	a.camp.message = msg + fmt.Sprintf("\n"+a.tr.UI("campcast.pray.chancedrop"), caster.PrayChance)
	a.camp.worship = nil
}

func (a *app) drawWorship(line func(string)) {
	w := a.camp.worship

	if !w.picked {
		line(a.tr.UI("campcast.worship.who"))
		line("")
		a.drawMemberList(line, w.caster, func(i int) string {
			c := a.members[i]
			if ok, why := game.CanWorship(&c); !ok {
				return "（" + a.reasonText(why) + "）"
			}
			return fmt.Sprintf("%s　%d%%", a.deityName(c.Deity), c.PrayChance)
		})
		line("")
		line(a.tr.UI("campcast.worship.keys1"))
	} else {
		line(fmt.Sprintf(a.tr.UI("campcast.worship.target"), a.members[w.caster].Name))
		line("")
		a.drawMemberList(line, w.target, func(i int) string {
			m := a.members[i]
			return fmt.Sprintf(a.tr.UI("campcast.worship.hp"), m.CurrentHP, m.MaxHP)
		})
		line("")
		line(a.tr.UI("campcast.worship.keys2"))
	}

	if a.camp.message != "" {
		line("")
		for _, l := range strings.Split(a.camp.message, "\n") {
			line(l)
		}
	}
}

// castScreen 是營地施法的狀態。
// plotAction 標記選單裡那兩個**不是法術**的主線動作（`docs/re/59`–`61`）。
//
// UNCURSE／IMPRISON 不在 `FILES.DTT` 的 43 筆法術表裡，原版是把它們
// 換進同一支施法選單當另一組選項（熱鍵 U／I）。
type plotAction int

const (
	plotNone plotAction = iota
	plotUncurse
	plotImprison
)

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
		a.camp.message = a.tr.UI("campcast.worship.empty")
		return
	}
	c := &castScreen{}
	a.rebuildCampSpells(c)
	a.camp.message = ""
	a.camp.cast = c
}

// rebuildCampSpells 依**目前選中的施法者**重列法術。
//
// 手冊：「角色學會的符文與吟唱法術會顯示在畫面底部」（`part-3.md`）——
// 所以清單是跟著人變的，換人要重列。原本是在開選單時列一次、
// 而且不看人，等於五系符文與三個吟唱學不學都一樣。
func (a *app) rebuildCampSpells(c *castScreen) {
	c.entries = nil
	c.spell = 0
	var caster *game.Character
	if c.caster >= 0 && c.caster < len(a.members) {
		caster = &a.members[c.caster]
	}
	for _, i := range game.CampCastCandidates(a.tables) {
		sp, err := a.tables.Spell(i)
		if err != nil || !game.CanCast(caster, i, sp) {
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
	c.entries = append(c.entries, a.plotCastEntries()...)
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
			a.rebuildCampSpells(c)
			if len(c.entries) == 0 {
				a.camp.message = fmt.Sprintf(a.tr.UI("campcast.cast.nospell"),
					a.members[c.caster].Name)
				return nil
			}
			a.camp.message = ""
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
			// 主線動作是固定花費、目標就是腳下那個符印 ——
			// 沒有「投入多少」也沒有目標可選，直接執行。
			if c.entries[c.spell].plot != plotNone {
				a.doPlotCast(c)
				return nil
			}
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
			effect := c.entries[c.spell].spell.Effect
			if effect == game.EffectLight || effect == game.EffectWindWalk {
				a.doCampCast(c)
			} else {
				c.step = castPickTarget
			}
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
	if a.doPlotCast(c) {
		return
	}
	caster, target := &a.members[c.caster], &a.members[c.target]
	e := c.entries[c.spell]

	res := game.CampCast(a.rng, caster, target, e.spell, c.power)
	if !res.OK {
		a.camp.message = a.reasonText(res.Reason)
		return
	}
	msg := fmt.Sprintf(a.tr.UI("campcast.cast.cast"), caster.Name, target.Name, e.name)
	switch {
	case res.Light > 0:
		a.torch = byte(res.Light)
		a.save.LightSource = byte(res.Light)
		msg = fmt.Sprintf(a.tr.UI("campcast.cast.light"),
			caster.Name, e.name, res.Light)
	case res.WindWalk:
		if a.save.ShardShattered == 0 {
			a.changeMap(34, 28, 50)
		} else {
			a.changeMap(43, 39, 35)
		}
		msg = a.tr.UI("campcast.cast.windwalk")
	case res.Resurrected:
		msg += a.tr.UI("campcast.cast.resurrected")
	case res.Cured:
		msg += a.tr.UI("campcast.cast.cured")
	case res.Reason != "":
		msg += "　" + a.reasonText(res.Reason)
	case res.Delta > 0:
		msg += fmt.Sprintf(a.tr.UI("campcast.cast.heal"), res.Delta)
	case res.Delta < 0:
		msg += fmt.Sprintf(a.tr.UI("campcast.cast.damage"), -res.Delta)
	case res.Released:
		msg += a.tr.UI("campcast.worship.released")
	case res.Withered:
		msg += a.tr.UI("campcast.worship.withered")
	default:
		msg += a.tr.UI("campcast.cast.noeffect")
	}
	if res.Died {
		msg += "　" + target.Name + a.tr.UI("campcast.cast.died")
	}
	a.camp.message = msg
	a.camp.cast = nil
}

func (a *app) drawCampCast(line func(string)) {
	c := a.camp.cast

	switch c.step {
	case castPickCaster:
		line(a.tr.UI("campcast.cast.who"))
		line("")
		a.drawMemberList(line, c.caster, func(i int) string {
			return fmt.Sprintf(a.tr.UI("campcast.cast.sp"), a.members[i].CurrentSP, a.members[i].MaxSP)
		})
		line("")
		line(a.tr.UI("campcast.cast.keys1"))

	case castPickSpell:
		line(fmt.Sprintf(a.tr.UI("campcast.cast.which"), a.members[c.caster].Name))
		line("")
		for i, e := range c.entries {
			mark := "   "
			if i == c.spell {
				mark = " > "
			}
			cost := e.spell.M
			if e.plot != plotNone {
				// 主線動作是固定花費，不走法術表的最低法力
				cost = plotCost(e.plot)
			}
			line(fmt.Sprintf(a.tr.UI("campcast.cast.entry"),
				mark, textlayout.PadCells(e.name, 14), cost))
		}
		line("")
		line(a.tr.UI("campcast.worship.keys2"))

	case castPickPower:
		e := c.entries[c.spell]
		line(fmt.Sprintf(a.tr.UI("campcast.cast.power_ask"), e.name))
		line("")
		line(fmt.Sprintf(a.tr.UI("campcast.cast.power"),
			c.power, e.spell.M, a.members[c.caster].CurrentSP))
		line("")
		line(a.tr.UI("campcast.cast.keys2"))

	default:
		line(fmt.Sprintf(a.tr.UI("campcast.cast.target"),
			a.members[c.caster].Name, c.entries[c.spell].name))
		line("")
		a.drawMemberList(line, c.target, func(i int) string {
			m := a.members[i]
			return fmt.Sprintf(a.tr.UI("campcast.cast.hpsp"),
				m.CurrentHP, m.MaxHP, m.CurrentSP, m.MaxSP)
		})
		line("")
		line(a.tr.UI("campcast.cast.keys3"))
	}

	if a.camp.message != "" {
		line("")
		for _, l := range strings.Split(a.camp.message, "\n") {
			line(l)
		}
	}
}

// plotCastEntries 依隊伍目前的位置，決定要不要在施法選單裡放主線動作。
//
// 原版把 UNCURSE／IMPRISON 換進同一支施法選單（`docs/re/61` §3），
// **但切換條件還沒反組譯出來**（掃過 `0x6300`–`0x6a14` 的近跳全部沒命中，
// 可能來自跳表）。所以這裡的條件是本專案依機制推導的設計決定：
//
//   - 腳下是符印圖塊（`0x63`）且在三張符印子地圖之一 → 給 UNCURSE
//   - 在禁錮判定過得了的地方 → 給 IMPRISON
//
// 兩者都對得上攻略「趕到每個符印上紮營，然後施展解咒」的玩法。
// 等切換條件解出來再換掉這一段。
func (a *app) plotCastEntries() []spellEntry {
	var out []spellEntry
	tile, err := a.tiles.TileAt(a.party.X(), a.party.Y())
	if err == nil && tile == game.GlyphTile && game.GlyphIndexFor(a.mapID) >= 0 {
		out = append(out, spellEntry{name: a.tr.UI("plot.uncurse"), plot: plotUncurse})
	}
	// 禁錮的出現條件是「三個符印都解完」，**不是「這裡施放得會成功」** ——
	// 後者會讓 "The spell fizzles..." 永遠走不到，等於幫玩家擋掉原版
	// 「在錯地方施放白扣 100 點」的損失，那是改遊戲不是移植。
	if game.CircleOfLightOpen(a.save.GlyphFlags) {
		out = append(out, spellEntry{name: a.tr.UI("plot.imprison"), plot: plotImprison})
	}
	return out
}

// plotCost 是主線動作的固定花費（`docs/re/59`、`60`）。
func plotCost(p plotAction) int {
	switch p {
	case plotUncurse:
		return game.UncurseCost
	case plotImprison:
		return game.ImprisonCost
	}
	return 0
}

// doPlotCast 執行主線動作。回傳 false 代表這一格不是主線動作。
//
// UNCURSE 的法力檢查在扣費之前，IMPRISON 則是**先扣再檢查地點** ——
// 兩者不一致是原版的行為，照抄（`docs/re/60` §2）。
func (a *app) doPlotCast(c *castScreen) bool {
	e := c.entries[c.spell]
	caster := &a.members[c.caster]

	switch e.plot {
	case plotUncurse:
		tile, err := a.tiles.TileAt(a.party.X(), a.party.Y())
		if err != nil {
			return false
		}
		switch game.Uncurse(caster, tile, a.mapID, &a.save.GlyphFlags) {
		case game.GlyphNoGlyph:
			a.camp.message = a.tr.UI("plot.noglyph")
		case game.GlyphAlreadyDone:
			a.camp.message = a.tr.UI("plot.inactive")
		case game.GlyphNotEnoughSP:
			a.camp.message = fmt.Sprintf(a.tr.UI("plot.needsp"), game.UncurseCost)
		case game.GlyphDestroyed:
			a.camp.message = a.tr.UI("plot.destroyed")
		}

	case plotImprison:
		switch game.Imprison(caster, a.mapID, a.party.Y()) {
		case game.ImprisonNotEnoughSP:
			a.camp.message = fmt.Sprintf(a.tr.UI("plot.needsp"), game.ImprisonCost)
		case game.ImprisonFizzles:
			a.camp.message = a.tr.UI("plot.fizzles")
		case game.ImprisonWon:
			a.won = true
			a.camp.message = a.tr.UI("plot.won")
		}

	default:
		return false
	}
	a.camp.cast = nil
	return true
}
