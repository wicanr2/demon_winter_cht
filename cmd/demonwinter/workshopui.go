package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 矮人大師的附魔工坊（地點劇情 case 6，地圖 2 的 (28,5)，`docs/re/102`）。
//
// **這是 worklist C2 找了好幾輪的那個服務。** 它不在城鎮八設施裡、
// 也不在市集 12 選項裡 —— 是地圖 2 上一格劇情事件。
//
// 三頁：選人 → 選道具 → 選附魔等級並確認。費用是
// 「附魔漲的估價 × 材質折扣」，見 `game.EnchantCost`。

// workshopStage 是目前停在哪一頁。
type workshopStage int

const (
	workshopPickMember workshopStage = iota
	workshopPickItem
	workshopPickPlus
)

// workshopScreen 是附魔工坊的狀態。
type workshopScreen struct {
	stage  workshopStage
	member int
	slot   int
	// plus 是要附魔到 +幾，從現值 +1 起算。
	plus int
	// msg 是這一頁下方的一行提示（原版那幾句拒絕理由）。
	msg string
}

// openWorkshop 是 case 6 的入口。
func (a *app) openWorkshop() {
	a.box = ui.NewMixedTextBox(a.tr.UI("workshop.scene"))
	a.workshop = &workshopScreen{}
	a.trace.note("附魔工坊：開場")
}

func (a *app) updateWorkshop() error {
	w := a.workshop
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch w.stage {
		case workshopPickMember:
			a.workshop = nil
			a.trace.note("附魔工坊：離開")
		case workshopPickItem:
			w.stage, w.msg = workshopPickMember, ""
		default:
			w.stage, w.msg = workshopPickItem, ""
		}
		return nil
	}

	switch w.stage {
	case workshopPickMember:
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			w.member = (w.member + 1) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			w.member = (w.member - 1 + len(a.members)) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			w.stage, w.slot, w.msg = workshopPickItem, 0, ""
		}

	case workshopPickItem:
		inv := a.members[w.member].Inventory
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			w.slot = (w.slot + 1) % len(inv)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			w.slot = (w.slot - 1 + len(inv)) % len(inv)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			a.workshopPickItem()
		}

	case workshopPickPlus:
		cur := a.members[w.member].Inventory[w.slot].Enchant
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyRight):
			if w.plus < game.EnchantMax {
				w.plus++
			} else {
				// 原版：`Enchantment beyond plus 10 / is not possible`
				w.msg = a.tr.UI("workshop.max")
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyLeft):
			if w.plus > cur+1 {
				w.plus--
			} else {
				// 原版：`It is already +%d`
				w.msg = fmt.Sprintf(a.tr.UI("workshop.already"), cur)
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			a.workshopConfirm()
		}
	}
	return nil
}

// workshopPickItem 是選完道具那一步的兩道閘門。
func (a *app) workshopPickItem() {
	w := a.workshop
	it := a.members[w.member].Inventory[w.slot]
	switch {
	case it.Empty():
		w.msg = a.tr.UI("workshop.empty")
	case !it.Identified:
		// 原版 `0x0f804`：`Only identified items / may be enchanted.`
		w.msg = a.tr.UI("workshop.unidentified")
	case !game.Enchantable(it):
		// 原版 `0x0f95a`：`Only weapons and armor / may be enchanted`
		w.msg = a.tr.UI("workshop.wrongtype")
	case it.Enchant >= game.EnchantMax:
		w.msg = fmt.Sprintf(a.tr.UI("workshop.already"), it.Enchant)
	default:
		w.stage, w.plus, w.msg = workshopPickPlus, it.Enchant+1, ""
	}
}

// workshopConfirm 收費並把附魔加上去。
//
// **錢不夠就什麼都不做**（原版 `0x0ff84` 比完金幣才決定，
// 不足時印 ds:0x12cc 那句）。
func (a *app) workshopConfirm() {
	w := a.workshop
	c := &a.members[w.member]
	it := c.Inventory[w.slot]
	cost := a.enchantCost(it, w.plus)
	if cost <= 0 {
		w.msg = a.tr.UI("workshop.nocost")
		return
	}
	if a.gold() < cost {
		w.msg = a.tr.UI("workshop.nogold")
		a.trace.note("附魔工坊：錢不夠（要 %d，有 %d）", cost, a.gold())
		return
	}
	// **名字要在改附魔之前取。** `itemLabel` 會自己附上 `+n`，
	// 改完再取就會印出「釘頭鎚+3 附魔到 +3」。
	label := a.itemLabel(it)
	a.setGold(a.gold() - cost)
	it.Enchant = w.plus
	c.Inventory[w.slot] = it
	a.message = fmt.Sprintf(a.tr.UI("workshop.done"),
		c.Name, label, w.plus, cost)
	a.trace.note("附魔工坊：%s 第 %d 格 → +%d，花 %d 金",
		c.Name, w.slot, w.plus, cost)
	a.workshop = nil
}

// enchantCost 查底價再算費用。認不出型別就回 0（＝不做事）。
func (a *app) enchantCost(it scenario.InventorySlot, plus int) int {
	item, err := a.items.ByIndex(int(it.Type))
	if err != nil {
		return 0
	}
	return game.EnchantCost(item.Price, it, plus)
}

func (a *app) drawWorkshop(dst *ebiten.Image) {
	w := a.workshop
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	switch w.stage {
	case workshopPickMember:
		line(a.tr.UI("workshop.who"))
		line("")
		for i := range a.members {
			line(memberMark(w.member, i) + a.members[i].Name)
		}

	case workshopPickItem:
		line(a.tr.UI("workshop.which"))
		line("")
		inv := a.members[w.member].Inventory
		for i := range inv {
			// itemLabel 本身就會附上 `+n`（battleui.go），不要再加一次。
			label := a.tr.UI("workshop.slotempty")
			if !inv[i].Empty() {
				label = a.itemLabel(inv[i])
			}
			line(memberMark(w.slot, i) + label)
		}

	case workshopPickPlus:
		it := a.members[w.member].Inventory[w.slot]
		line(fmt.Sprintf(a.tr.UI("workshop.raise"),
			a.itemLabel(it), w.plus))
		line("")
		line(fmt.Sprintf(a.tr.UI("workshop.cost"),
			a.enchantCost(it, w.plus), a.gold()))
		line("")
		line(a.tr.UI("workshop.pluskeys"))
	}

	if w.msg != "" {
		line("")
		line(w.msg)
	}
	if w.stage != workshopPickPlus {
		line("")
		line(a.tr.UI("dungeon.keys"))
	}
}
