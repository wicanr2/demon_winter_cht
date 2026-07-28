package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 紮營選單的 Drop 與 Trade（規則在 internal/game/camp.go，出處 docs/re/33）。
//
// 兩者的流程只差最後一步：
//
//	Drop     選人 → 選道具 → 丟掉
//	Identify 選人 → 選道具 → 研究（每人每天一次）
//	Use      選人 → 選道具 → 用掉（目前只走得通光源那一條）
//	Xorcise  選施術者 → 選對象 → 選身上的裝備 → 驅邪
//	Trade    選人 → 選道具 → 選收方 → 交出去
//
// 所以共用同一組游標，用 itemAction 區分。

type itemAction int

const (
	itemActionNone itemAction = iota
	itemActionDrop
	itemActionTrade
	itemActionIdentify
	itemActionUse
	itemActionExorcise
)

// itemPicker 是 Drop／Trade 共用的三層游標。
type itemPicker struct {
	action itemAction
	// member 是持有人；slot 是道具欄索引，−1 代表還在選人。
	member, slot int
	// target 是 Trade 的收方，−1 代表還沒進到選收方那一步。
	target int
	// caster 是 Xorcise 的施術者，−1 代表還在選他。
	caster int
}

func (a *app) openItemAction(act itemAction) {
	if len(a.members) == 0 {
		a.camp.message = a.tr.UI("campitem.msg.no_party", "隊伍是空的")
		return
	}
	a.camp.message = ""
	p := &itemPicker{action: act, member: 0, slot: -1, target: -1, caster: -1}
	if act != itemActionExorcise {
		p.caster = 0 // 只有驅邪要先選施術者
	}
	a.camp.items = p
}

func (a *app) updateItemPicker() error {
	c := a.camp
	p := c.items

	// 驅邪的順序與其他三項相反：先選施術者、再選對象，最後才選裝備。
	if p.action == itemActionExorcise {
		switch {
		case p.caster < 0:
			return a.updateExorciseCaster(p)
		case p.slot < 0:
			return a.updateItemOwner(p)
		default:
			return a.updateItemSlot(p)
		}
	}

	switch {
	case p.slot < 0:
		return a.updateItemOwner(p)
	case p.target < 0:
		return a.updateItemSlot(p)
	default:
		return a.updateItemTarget(p)
	}
}

// updateExorciseCaster 選施術者（只有 Xorcise 會走到）。
func (a *app) updateExorciseCaster(p *itemPicker) error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.camp.items = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		p.caster = (p.caster + 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		p.caster = (p.caster - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if p.caster < 0 {
			p.caster = 0
		}
		p.member = 0
	}
	return nil
}

// updateItemOwner 選持有人。
func (a *app) updateItemOwner(p *itemPicker) error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		if p.action == itemActionExorcise {
			p.caster = -1
			return nil
		}
		a.camp.items = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		p.member = (p.member + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		p.member = (p.member - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		p.slot = 0
	}
	return nil
}

// updateItemSlot 選道具欄。
func (a *app) updateItemSlot(p *itemPicker) error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		p.slot = -1
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		p.slot = (p.slot + 1) % game.InventorySlots
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		p.slot = (p.slot - 1 + game.InventorySlots) % game.InventorySlots
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if p.action == itemActionTrade {
			// 原版在**選道具這一步**就擋掉空格與裝備中的東西（印訊息、
			// 回頭再問一次），不是先問「給誰」再來拒絕。照做。
			if why := a.tradableReason(p); why != "" {
				a.camp.message = why
				return nil
			}
			// 收方預設指向下一個人，免得游標一開就停在自己身上。
			p.target = (p.member + 1) % len(a.members)
			return nil
		}
		switch p.action {
		case itemActionIdentify:
			a.identifySelected(p)
		case itemActionUse:
			a.useSelected(p)
		case itemActionExorcise:
			a.exorciseSelected(p)
		default:
			a.dropSelected(p)
		}
	}
	return nil
}

// tradableReason 回傳這一格不能交出去的原因，可以交出去回空字串。
// 只看「與收方無關」的那兩條 —— 收方滿了要等選完人才知道。
func (a *app) tradableReason(p *itemPicker) string {
	m := a.members[p.member]
	switch {
	case m.Inventory[p.slot].Empty():
		return a.tr.UI("campitem.trade.slot_empty", "這一格是空的")
	case p.slot == m.EquippedWeapon || p.slot == m.EquippedArmor:
		return a.tr.UI("campitem.trade.equipped", "那件裝備還在身上")
	}
	return ""
}

// updateItemTarget 選收方（只有 Trade 會走到）。
func (a *app) updateItemTarget(p *itemPicker) error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		p.target = -1
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		p.target = (p.target + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		p.target = (p.target - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.giveSelected(p)
	}
	return nil
}

func (a *app) dropSelected(p *itemPicker) {
	m := &a.members[p.member]
	label := a.itemLabel(m.Inventory[p.slot])
	res := game.DropItem(m, p.slot, a.save.EndingOffered)
	if !res.OK {
		a.camp.message = res.Reason
		return
	}
	a.camp.message = fmt.Sprintf(a.tr.UI("campitem.msg.dropped", "%s 丟掉了%s"), m.Name, label)
	a.camp.items = nil
}

func (a *app) identifySelected(p *itemPicker) {
	m := &a.members[p.member]
	label := a.itemLabel(m.Inventory[p.slot])
	res := game.Identify(a.rng, m, p.slot)
	if !res.OK {
		a.camp.message = res.Reason
		return
	}
	if res.Success {
		a.camp.message = fmt.Sprintf(a.tr.UI("campitem.msg.identified", "%s 看懂了%s（成功率 %d%%）"),
			m.Name, label, res.Chance)
	} else {
		a.camp.message = fmt.Sprintf(a.tr.UI("campitem.msg.identify_failed", "%s 研究了一整天也看不出名堂（成功率 %d%%）"),
			m.Name, res.Chance)
	}
	a.camp.items = nil
}

func (a *app) useSelected(p *itemPicker) {
	m := &a.members[p.member]
	label := a.itemLabel(m.Inventory[p.slot])
	res := game.UseInCamp(m, p.slot)
	if !res.OK {
		a.camp.message = res.Reason
		return
	}
	a.torch = byte(res.Light)
	a.save.LightSource = byte(res.Light)
	a.camp.message = fmt.Sprintf(a.tr.UI("campitem.msg.lit", "%s 點起了%s（光源 %d）"), m.Name, label, res.Light)
	a.camp.items = nil
}

func (a *app) exorciseSelected(p *itemPicker) {
	caster, target := &a.members[p.caster], &a.members[p.member]
	label := a.itemLabel(target.Inventory[p.slot])
	res := game.Exorcise(a.rng, caster, target, p.slot)
	if !res.OK {
		a.camp.message = res.Reason
		return
	}
	if res.Success {
		msg := fmt.Sprintf(a.tr.UI("campitem.msg.exorcised", "%s 解開了%s的詛咒（成功率 %d%%）"),
			caster.Name, label, res.Chance)
		if res.Freed > 0 {
			msg += fmt.Sprintf(a.tr.UI("campitem.msg.skills_freed", "　%s 找回 %d 項技能"), target.Name, res.Freed)
		}
		a.camp.message = msg
	} else {
		a.camp.message = fmt.Sprintf(a.tr.UI("campitem.msg.exorcise_failed", "%s 驅不動%s（成功率 %d%%）"),
			caster.Name, label, res.Chance)
	}
	a.camp.items = nil
}

func (a *app) giveSelected(p *itemPicker) {
	if p.member == p.target {
		a.camp.message = a.tr.UI("campitem.trade.self", "不能給自己")
		return
	}
	from, to := &a.members[p.member], &a.members[p.target]
	label := a.itemLabel(from.Inventory[p.slot])
	res := game.GiveItem(from, to, p.slot)
	if !res.OK {
		// 原版撞到「裝備中」會退回選道具那一步，不是整個取消 —— 照做。
		a.camp.message = res.Reason
		p.target = -1
		return
	}
	a.camp.message = fmt.Sprintf(a.tr.UI("campitem.msg.given", "%s 把%s交給 %s"), from.Name, label, to.Name)
	a.camp.items = nil
}

func (a *app) drawItemPicker(line func(string)) {
	p := a.camp.items

	if p.caster < 0 {
		line(a.tr.UI("campitem.exorcise.who", "誰來驅邪？"))
		line("")
		a.drawMemberList(line, 0, func(i int) string {
			c := a.members[i]
			if !c.HasSkill(game.SkillShaman) && !c.HasSkill(game.SkillPriesthood) {
				return a.tr.UI("campitem.exorcise.no_skill", "（不會驅邪）")
			}
			if c.ExorcisedToday {
				return a.tr.UI("campitem.exorcise.used_today", "（今天驅過了）")
			}
			return ""
		})
		line("")
		line(a.tr.UI("campitem.keys.confirm_cancel", "↑↓：選擇　Enter：確定　Esc：取消"))
		if a.camp.message != "" {
			line("")
			line(a.camp.message)
		}
		return
	}

	m := a.members[p.member]

	// 拒絕的理由要留在原畫面 —— 只印在主選單的話，按下去像是沒反應。
	defer func() {
		if a.camp.message == "" {
			return
		}
		line("")
		for _, l := range strings.Split(a.camp.message, "\n") {
			line(l)
		}
	}()

	switch {
	case p.slot < 0:
		verb := a.tr.UI("campitem.verb.drop", "丟東西")
		switch p.action {
		case itemActionTrade:
			verb = a.tr.UI("campitem.verb.trade", "交出東西")
		case itemActionIdentify:
			verb = a.tr.UI("campitem.verb.identify", "研究道具")
		case itemActionUse:
			verb = a.tr.UI("campitem.verb.use", "用東西")
		case itemActionExorcise:
			verb = a.tr.UI("campitem.verb.exorcise", "被驅邪")
		}
		line(a.tr.UI("campitem.owner.who_prefix", "誰要") + verb + "？")
		line("")
		a.drawMemberList(line, p.member, nil)
		line("")
		line(a.tr.UI("campitem.keys.confirm_cancel", "↑↓：選擇　Enter：確定　Esc：取消"))

	case p.target < 0:
		head := a.tr.UI("campitem.slot.head_drop", "要丟掉哪一件？")
		switch p.action {
		case itemActionTrade:
			head = a.tr.UI("campitem.slot.head_trade", "要交出哪一件？")
		case itemActionIdentify:
			head = a.tr.UI("campitem.slot.head_identify", "要研究哪一件？")
		case itemActionUse:
			head = a.tr.UI("campitem.slot.head_use", "要用哪一件？")
		case itemActionExorcise:
			head = a.tr.UI("campitem.slot.head_exorcise", "要驅哪一件？")
		}
		line(fmt.Sprintf("%s %s", m.Name, head))
		line("")
		for i := 0; i < game.InventorySlots; i++ {
			mark := "   "
			if i == p.slot {
				mark = " > "
			}
			it := m.Inventory[i]
			name, note := a.tr.UI("campitem.slot.empty", "（空）"), ""
			if !it.Empty() {
				name = a.itemLabel(it)
				switch {
				case i == m.EquippedWeapon:
					note = a.tr.UI("campitem.slot.weapon", "（武器）")
				case i == m.EquippedArmor:
					note = a.tr.UI("campitem.slot.armor", "（護甲）")
				case it.Type == game.ItemTypeDungeon:
					note = a.tr.UI("campitem.slot.dungeon_item", "（地城道具）")
				}
				// 鑑定時把「看不懂」與「已鑑定」先標出來 ——
				// 一天只能試一次，讓人白白浪費一天不厚道。
				if p.action == itemActionExorcise {
					switch i {
					case m.EquippedWeapon:
						note = a.tr.UI("campitem.slot.weapon_exorcisable", "（武器，可驅）")
					case m.EquippedArmor:
						note = a.tr.UI("campitem.slot.armor_exorcisable", "（護甲，可驅）")
					default:
						note = a.tr.UI("campitem.slot.not_worn", "（沒穿在身上）")
					}
				}
				if p.action == itemActionUse {
					if lv := game.LightSourceLevel(it.Type); lv != 0 {
						note = fmt.Sprintf(a.tr.UI("campitem.slot.light_source", "（點起來，光源 %d）"), lv)
					} else if !it.Usable() {
						note = a.tr.UI("campitem.slot.unusable", "（用不了）")
					} else {
						note = a.tr.UI("campitem.slot.battle_only", "（戰鬥中才能用）")
					}
				}
				if p.action == itemActionIdentify {
					switch {
					case it.Identified:
						note = a.tr.UI("campitem.slot.identified", "（已鑑定）")
					case !m.HasSkill(game.LoreSkillFor(it.Type)):
						note = a.tr.UI("campitem.slot.unreadable", "（看不懂）")
					default:
						note = a.tr.UI("campitem.slot.unidentified", "（未鑑定）")
					}
				}
			}
			line(fmt.Sprintf("%s%s%s", mark, textlayout.PadCells(name, 12), note))
		}
		line("")
		line(a.tr.UI("campitem.keys.confirm_back", "↑↓：選擇　Enter：確定　Esc：返回"))

	default:
		line(fmt.Sprintf(a.tr.UI("campitem.target.who", "%s 的%s要給誰？"), m.Name,
			a.itemLabel(m.Inventory[p.slot])))
		line("")
		a.drawMemberList(line, p.target, func(i int) string {
			switch {
			case i == p.member:
				return a.tr.UI("campitem.target.self", "（本人）")
			case a.members[i].FreeSlot() < 0:
				return a.tr.UI("campitem.target.full", "（道具欄滿了）")
			}
			return ""
		})
		line("")
		line(a.tr.UI("campitem.keys.confirm_give", "↑↓：選擇　Enter：交出　Esc：返回"))
	}
}

// drawMemberList 畫一份帶游標的隊員清單，note 可以為每一列補一段說明。
func (a *app) drawMemberList(line func(string), cursor int, note func(i int) string) {
	for i, m := range a.members {
		mark := "   "
		if i == cursor {
			mark = " > "
		}
		extra := ""
		if note != nil {
			extra = note(i)
		}
		line(fmt.Sprintf("%s%s%s", mark, textlayout.PadCells(m.Name, 10), extra))
	}
}
