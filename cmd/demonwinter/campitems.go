package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 紮營選單的 Drop 與 Trade（規則在 internal/game/camp.go，出處 docs/re/33）。
//
// 兩者的流程只差最後一步：
//
//	Drop  選人 → 選道具 → 丟掉
//	Trade 選人 → 選道具 → 選收方 → 交出去
//
// 所以共用同一組游標，用 itemAction 區分。

type itemAction int

const (
	itemActionNone itemAction = iota
	itemActionDrop
	itemActionTrade
)

// itemPicker 是 Drop／Trade 共用的三層游標。
type itemPicker struct {
	action itemAction
	// member 是持有人；slot 是道具欄索引，−1 代表還在選人。
	member, slot int
	// target 是 Trade 的收方，−1 代表還沒進到選收方那一步。
	target int
}

func (a *app) openItemAction(act itemAction) {
	if len(a.members) == 0 {
		a.camp.message = "隊伍是空的"
		return
	}
	a.camp.message = ""
	a.camp.items = &itemPicker{action: act, member: 0, slot: -1, target: -1}
}

func (a *app) updateItemPicker() error {
	c := a.camp
	p := c.items

	switch {
	case p.slot < 0:
		return a.updateItemOwner(p)
	case p.target < 0:
		return a.updateItemSlot(p)
	default:
		return a.updateItemTarget(p)
	}
}

// updateItemOwner 選持有人。
func (a *app) updateItemOwner(p *itemPicker) error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
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
		a.dropSelected(p)
	}
	return nil
}

// tradableReason 回傳這一格不能交出去的原因，可以交出去回空字串。
// 只看「與收方無關」的那兩條 —— 收方滿了要等選完人才知道。
func (a *app) tradableReason(p *itemPicker) string {
	m := a.members[p.member]
	switch {
	case m.Inventory[p.slot].Empty():
		return "這一格是空的"
	case p.slot == m.EquippedWeapon || p.slot == m.EquippedArmor:
		return "那件裝備還在身上"
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
	res := game.DropItem(m, p.slot, a.save.UnknownC1)
	if !res.OK {
		a.camp.message = res.Reason
		return
	}
	a.camp.message = fmt.Sprintf("%s 丟掉了%s", m.Name, label)
	a.camp.items = nil
}

func (a *app) giveSelected(p *itemPicker) {
	if p.member == p.target {
		a.camp.message = "不能給自己"
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
	a.camp.message = fmt.Sprintf("%s 把%s交給 %s", from.Name, label, to.Name)
	a.camp.items = nil
}

func (a *app) drawItemPicker(line func(string)) {
	p := a.camp.items
	m := a.members[p.member]

	// 拒絕的理由要留在原畫面 —— 只印在主選單的話，按下去像是沒反應。
	defer func() {
		if a.camp.message != "" {
			line("")
			line(a.camp.message)
		}
	}()

	switch {
	case p.slot < 0:
		verb := "丟東西"
		if p.action == itemActionTrade {
			verb = "交出東西"
		}
		line("誰要" + verb + "？")
		line("")
		a.drawMemberList(line, p.member, nil)
		line("")
		line("↑↓：選擇　Enter：確定　Esc：取消")

	case p.target < 0:
		head := "要丟掉哪一件？"
		if p.action == itemActionTrade {
			head = "要交出哪一件？"
		}
		line(fmt.Sprintf("%s %s", m.Name, head))
		line("")
		for i := 0; i < game.InventorySlots; i++ {
			mark := "   "
			if i == p.slot {
				mark = " > "
			}
			it := m.Inventory[i]
			name, note := "（空）", ""
			if !it.Empty() {
				name = a.itemLabel(it)
				switch {
				case i == m.EquippedWeapon:
					note = "（武器）"
				case i == m.EquippedArmor:
					note = "（護甲）"
				case it.Type == game.ItemTypeDungeon:
					note = "（地城道具）"
				}
			}
			line(fmt.Sprintf("%s%s%s", mark, textlayout.PadCells(name, 12), note))
		}
		line("")
		line("↑↓：選擇　Enter：確定　Esc：返回")

	default:
		line(fmt.Sprintf("%s 的%s要給誰？", m.Name,
			a.itemLabel(m.Inventory[p.slot])))
		line("")
		a.drawMemberList(line, p.target, func(i int) string {
			switch {
			case i == p.member:
				return "（本人）"
			case a.members[i].FreeSlot() < 0:
				return "（道具欄滿了）"
			}
			return ""
		})
		line("")
		line("↑↓：選擇　Enter：交出　Esc：返回")
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
