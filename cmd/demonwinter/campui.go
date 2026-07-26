package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 紮營畫面。
//
// 原版的紮營選單有 **14 個**選項（`docs/re/33` §1）：
// Reorder／Sleep／Identify／Worship／Xorcise／View land／Trade／Drop／
// Equip／Use／Hunt／Cast／Quit。**這裡只做了規則已經解出來的那幾個** ——
// 睡覺與打獵。其餘標成「尚未實作」列出來，不假裝沒有。
//
// **進入鍵是本作自己選的 `R`**：原版用哪個鍵沒查（手冊那句「按 P 鍵」指的是
// 紮營畫面裡的另一個功能，不是進入鍵），而 `P` 在本作已經是隊伍名冊。
type campScreen struct {
	// message 是最近一次操作的結果。
	message string
	// hunter 是打獵選單的游標；-1 代表選單沒開。
	hunter int

	// equipMember／equipSlot 是換裝選單的兩層游標；
	// equipMember 為 -1 代表選單沒開，equipSlot 為 -1 代表還在選人。
	equipMember, equipSlot int

	// items 是 Drop／Trade 共用的游標；nil 代表沒在進行（見 campitems.go）。
	items *itemPicker

	// reorder 是排陣型的進行狀態；nil 代表沒在進行（見 campreorder.go）。
	reorder *reorderScreen

	// viewLand 是觀地檢視的狀態；nil 代表沒在進行（見 campviewland.go）。
	viewLand *viewLandScreen
}

// openCamp 進入紮營畫面。
func (a *app) openCamp() {
	a.camp = &campScreen{hunter: -1, equipMember: -1, equipSlot: -1}
}

func (a *app) updateCamp() error {
	c := a.camp

	// 打獵要先選人。
	if c.hunter >= 0 {
		return a.updateHuntPicker()
	}
	if c.equipMember >= 0 {
		return a.updateEquipPicker()
	}
	if c.items != nil {
		return a.updateItemPicker()
	}
	if c.reorder != nil {
		return a.updateReorder()
	}
	if c.viewLand != nil {
		return a.updateViewLand()
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.camp = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyS):
		a.campSleep()
	case inpututil.IsKeyJustPressed(ebiten.KeyH):
		if len(a.members) == 0 {
			c.message = "隊伍是空的"
			return nil
		}
		c.hunter = 0
	case inpututil.IsKeyJustPressed(ebiten.KeyE):
		if len(a.members) == 0 {
			c.message = "隊伍是空的"
			return nil
		}
		c.equipMember, c.equipSlot = 0, -1
	case inpututil.IsKeyJustPressed(ebiten.KeyD):
		a.openItemAction(itemActionDrop)
	case inpututil.IsKeyJustPressed(ebiten.KeyT):
		a.openItemAction(itemActionTrade)
	case inpututil.IsKeyJustPressed(ebiten.KeyR):
		a.openReorder()
	case inpututil.IsKeyJustPressed(ebiten.KeyI):
		a.openItemAction(itemActionIdentify)
	case inpututil.IsKeyJustPressed(ebiten.KeyV):
		a.openViewLand()
	}
	return nil
}

// updateEquipPicker 處理換裝的兩層選單：先選人、再選要裝哪一格。
func (a *app) updateEquipPicker() error {
	c := a.camp
	if c.equipSlot < 0 {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			c.equipMember = -1
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			c.equipMember = (c.equipMember + 1) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			c.equipMember = (c.equipMember - 1 + len(a.members)) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			c.equipSlot = 0
		}
		return nil
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		c.equipSlot = -1
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		c.equipSlot = (c.equipSlot + 1) % game.InventorySlots
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		c.equipSlot = (c.equipSlot - 1 + game.InventorySlots) % game.InventorySlots
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		m := &a.members[c.equipMember]
		ok, why := m.Equip(c.equipSlot)
		if !ok {
			c.message = why
			return nil
		}
		c.message = fmt.Sprintf("%s 換上%s", m.Name,
			a.itemLabel(m.Inventory[c.equipSlot]))
		c.equipMember, c.equipSlot = -1, -1
	}
	return nil
}

// campSleep 在野外睡一晚。
func (a *app) campSleep() {
	c := a.camp
	if !a.clock.CanSleep() {
		c.message = "現在睡不著（原版：You are restless）"
		return
	}
	rations := int(a.save.Rations)
	res := game.Rest(a.rng, game.RestCamp, a.members, a.clock, &rations)
	a.save.Rations = byte(rations)

	// 紮營會把光源重設成火把（原版 2aed:040c）。
	a.torch = game.RestCampTorch
	// 隊伍層級的「每日一次」旗標也由睡覺清掉（`2aed:03f1` 那一段）。
	a.save.ViewedLandToday = false

	msg := fmt.Sprintf("睡了 %d 個時辰，%d 日 %d 時醒來",
		res.Hours, a.clock.Day(), a.clock.Hour())
	switch {
	case res.Starved:
		msg += "　沒有糧食，全隊受餓"
	case res.AteFood:
		msg += fmt.Sprintf("　吃掉一份糧食（剩 %d）", rations)
	}
	for _, i := range res.Died {
		msg += "　" + a.members[i].Name + " 沒有醒來"
	}
	c.message = msg
}

func (a *app) updateHuntPicker() error {
	c := a.camp
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		c.hunter = -1
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		c.hunter = (c.hunter + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		c.hunter = (c.hunter - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		rations := int(a.save.Rations)
		res := game.Hunt(a.rng, &a.members[c.hunter], &rations)
		a.save.Rations = byte(rations)
		c.hunter = -1

		switch {
		case res.Reason != "":
			c.message = res.Reason
		case res.Gained == 0:
			c.message = "這趟打獵一無所獲"
		default:
			c.message = fmt.Sprintf("打到 %d 份糧食（共 %d 份）", res.Gained, rations)
		}
	}
	return nil
}

func (a *app) drawCamp(dst *ebiten.Image) {
	c := a.camp
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	// 觀地攤開地圖時整個畫面讓給地圖 —— 紮營抬頭會壓在右欄的字上。
	if c.viewLand != nil && c.viewLand.member < 0 {
		a.drawViewLand(dst, line)
		return
	}

	line(fmt.Sprintf("紮營　%s月 %d 日 %d 時　糧食 %d 份",
		a.monthName(), a.clock.Day(), a.clock.Hour(), a.save.Rations))
	line("")

	if c.hunter >= 0 {
		line("誰去打獵？")
		line("")
		for i, m := range a.members {
			mark := "   "
			if i == c.hunter {
				mark = " > "
			}
			note := ""
			if !m.Skills[game.SkillHunting] {
				note = "（不會狩獵）"
			}
			line(fmt.Sprintf("%s%s%s", mark,
				textlayout.PadCells(m.Name, 10), note))
		}
		line("")
		line("↑↓：選擇　Enter：出發　Esc：取消")
		return
	}

	if c.equipMember >= 0 {
		a.drawEquipPicker(line)
		return
	}

	if c.items != nil {
		a.drawItemPicker(line)
		return
	}

	if c.reorder != nil {
		a.drawReorder(line)
		return
	}

	if c.viewLand != nil {
		a.drawViewLand(dst, line)
		return
	}

	if a.clock.CanSleep() {
		line("  S 睡覺")
	} else {
		line("  S 睡覺（現在睡不著，要 15–24 時）")
	}
	line("  H 打獵")
	line("  E 換裝")
	line("  D 丟棄　T 交給隊友")
	line("  R 排列陣型　I 鑑定道具　V 觀地")
	line("")
	line("Esc：收帳篷")
	line("")
	if c.message != "" {
		line(c.message)
		line("")
	}
	line("※ 原版的紮營選單有 14 項，這裡只做了規則已解出的")
	line("　 睡覺、打獵、換裝、丟棄、轉手、陣型、鑑定、觀地，")
	line("　 其餘見 docs/re/33")
}

func (a *app) drawEquipPicker(line func(string)) {
	c := a.camp
	if c.equipSlot < 0 {
		line("誰要換裝？")
		line("")
		for i, m := range a.members {
			mark := "   "
			if i == c.equipMember {
				mark = " > "
			}
			line(fmt.Sprintf("%s%s%s　護甲 %d", mark,
				textlayout.PadCells(m.Name, 10),
				textlayout.PadCells(a.weaponLabel(m), 8), m.ArmorRating()))
		}
		line("")
		line("↑↓：選擇　Enter：確定　Esc：取消")
		return
	}

	m := a.members[c.equipMember]
	line(fmt.Sprintf("%s 要換上哪一件？", m.Name))
	line("")
	for i := 0; i < game.InventorySlots; i++ {
		mark := "   "
		if i == c.equipSlot {
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
			case !game.CanEquipAsWeapon(it) && !game.CanEquipAsArmor(it):
				note = "（不能裝備）"
			}
		}
		line(fmt.Sprintf("%s%s%s", mark, textlayout.PadCells(name, 12), note))
	}
	line("")
	line("↑↓：選擇　Enter：換上　Esc：返回")
}
