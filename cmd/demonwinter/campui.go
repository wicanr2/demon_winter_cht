package main

import (
	"fmt"
	"strings"

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
// Party／Reorder／Sleep／Identify／Worship／Xorcise／View land／Trade／
// Drop／Equip／Use／Hunt／Cast／Quit。**十四項全部接上了** ——
// 對照表與各自的出處見 `docs/re/33` §1。
//
// **進入鍵是 `C`，與原版一致**（手冊 §407：「只要不在城鎮內隨時都可按 `C`
// 紮營休息，但戰鬥中無效」；DOSBox 實跑驗過，見 `docs/re/81` §4）。
//
// 這裡原本是 `R`，理由寫「原版用哪個鍵沒查（手冊那句『按 P 鍵』指的是紮營
// 畫面裡的另一個功能，不是進入鍵）」—— **手冊是查了，但查到錯的那一句**。
// §305 的「按 P 鍵」確實不是進入鍵，可是 §407「紮營」那一節開頭就寫著 `C`。
// 手冊還有一節叫「控制方法」，開宗明義說「選單指令一律鍵入該選項的第一個
// 英文字母」—— Camp 的第一個字母。**查到一句不合就停，比沒查更容易下錯結論。**
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

	// party 是角色卡的狀態；nil 代表沒在看（見 campparty.go）。
	party *partyScreen

	// cast 是營地施法的狀態；nil 代表沒在進行（見 campcast.go）。
	cast *castScreen

	// worship 是敬拜的狀態；nil 代表沒在進行（見 campcast.go）。
	worship *worshipScreen
}

var campMenuLabels = []uiLabel{
	{"camp.menu.party"},
	{"camp.menu.reorder"},
	{"camp.menu.sleep"},
	{"camp.menu.identify"},
	{"camp.menu.worship"},
	{"camp.menu.exorcise"},
	{"camp.menu.viewland"},
	{"camp.menu.trade"},
	{"camp.menu.drop"},
	{"camp.menu.equip"},
	{"camp.menu.use"},
	{"camp.menu.hunt"},
	{"camp.menu.cast"},
	{"camp.menu.quit"},
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
	if c.party != nil {
		if a.updatePartySheet(c.party) {
			c.party = nil
		}
		return nil
	}
	if c.cast != nil {
		return a.updateCampCast()
	}
	if c.worship != nil {
		return a.updateWorship()
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.camp = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyS):
		a.campSleep()
	case inpututil.IsKeyJustPressed(ebiten.KeyH):
		if len(a.members) == 0 {
			c.message = a.tr.UI("camp.empty")
			return nil
		}
		c.hunter = 0
	case inpututil.IsKeyJustPressed(ebiten.KeyE):
		if len(a.members) == 0 {
			c.message = a.tr.UI("camp.empty")
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
	case inpututil.IsKeyJustPressed(ebiten.KeyU):
		a.openItemAction(itemActionUse)
	case inpututil.IsKeyJustPressed(ebiten.KeyP):
		a.openPartySheet()
	case inpututil.IsKeyJustPressed(ebiten.KeyX):
		a.openItemAction(itemActionExorcise)
	case inpututil.IsKeyJustPressed(ebiten.KeyC):
		a.openCampCast()
	case inpututil.IsKeyJustPressed(ebiten.KeyW):
		a.openWorship()
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
		c.message = "" // 換一格就把上一格的理由收掉
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		c.equipSlot = (c.equipSlot - 1 + game.InventorySlots) % game.InventorySlots
		c.message = ""
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		m := &a.members[c.equipMember]
		ok, why := m.Equip(c.equipSlot)
		if !ok {
			c.message = why
			return nil
		}
		c.message = fmt.Sprintf(a.tr.UI("camp.equip.done"), m.Name,
			a.itemLabel(m.Inventory[c.equipSlot]))
		c.equipMember, c.equipSlot = -1, -1
	}
	return nil
}

// campSleep 在野外睡一晚。
func (a *app) campSleep() {
	c := a.camp
	if !a.clock.CanSleep() {
		c.message = a.tr.UI("camp.sleep.blocked")
		return
	}
	rations := int(a.save.Rations)
	res := game.Rest(a.rng, game.RestCamp, a.members, a.clock, &rations)
	a.save.Rations = byte(rations)

	// 紮營會把光源重設成火把（原版 2aed:040c）。
	a.torch = game.RestCampTorch
	// 隊伍層級的「每日一次」旗標也由睡覺清掉（`2aed:03f1` 那一段）。
	a.save.ViewedLandToday = false
	// 治療水池的額度也在同一段補回 7（原版 `0x1eee6`，`docs/re/90` §2）。
	game.ResetPoolDrinks(a.save)
	// 兩個靈視技能的每日次數也在同一段清 0（`0x1ef68`–`0x1ef7c`）。
	game.ResetPsychicUses(a.save)

	msg := fmt.Sprintf(a.tr.UI("camp.sleep.result"),
		res.Hours, a.clock.Day(), a.clock.Hour())
	switch {
	case res.Starved:
		msg += a.tr.UI("camp.sleep.starved")
	case res.AteFood:
		msg += fmt.Sprintf(a.tr.UI("camp.sleep.ate"), rations)
	}
	for _, i := range res.Died {
		msg += "　" + a.members[i].Name + a.tr.UI("camp.sleep.died")
	}
	c.message = msg

	// 睡著之後才輪到劇情 —— 冬之魔是在玩家睡覺的時候降臨的（`docs/re/80`）。
	a.advancePlot()
}

// advancePlot 推進一段劇情並把夢播出來（原版 `1000:0206`）。
//
// 原版每晚只走一段，所以這裡也只播一場夢。第三場夢連帶把神殿變成廢墟、
// 把全隊的信仰清空 —— **那是永久剝奪，不能「順手修掉」**（`docs/re/79` §3）。
func (a *app) advancePlot() {
	st := game.PlotState{
		Month:      byte(a.clock.Month()),
		Stage:      a.save.PlotStage,
		FirstDream: a.save.FirstDream,
		TempleRuin: a.save.TempleRuins,
	}
	dream, wipeFaith := game.AdvancePlotOnSleep(&st)
	a.save.PlotStage = st.Stage
	a.save.FirstDream = st.FirstDream
	a.save.TempleRuins = st.TempleRuin
	if dream == game.NoDream {
		return
	}
	if wipeFaith {
		for i := range a.members {
			game.WipeFaith(&a.members[i])
		}
		// 神殿變成廢墟是畫面上看得見的：地圖要重畫（`docs/re/79` §2）。
		a.drawTiles = ditheredTiles(a.tiles, uint16(a.ditherSeed), a.save.TempleRuins)
	}
	a.openDream(int(dream))
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
			c.message = a.tr.UI("camp.hunt.empty")
		default:
			c.message = fmt.Sprintf(a.tr.UI("camp.hunt.gained"), res.Gained, rations)
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

	// 營地各層共用地圖視窗的面板骨架；先畫框，再讓標題壓在上框線。
	cellW, cellH, _ := a.tileMetrics()
	drawMapFrame(dst, cellW, cellH)

	line(fmt.Sprintf(a.tr.UI("camp.header"),
		a.monthName(), a.clock.Day(), a.clock.Hour(), a.save.Rations))
	line("")

	if c.hunter >= 0 {
		line(a.tr.UI("camp.hunt.who"))
		line("")
		for i, m := range a.members {
			mark := "   "
			if i == c.hunter {
				mark = " > "
			}
			note := ""
			if !m.Skills[game.SkillHunting] {
				note = a.tr.UI("camp.hunt.cannot")
			}
			line(fmt.Sprintf("%s%s%s", mark,
				textlayout.PadCells(m.Name, 10), note))
		}
		line("")
		line(a.tr.UI("camp.hunt.keys"))
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

	if c.party != nil {
		a.drawPartySheet(c.party, line)
		return
	}

	if c.cast != nil {
		a.drawCampCast(line)
		return
	}

	if c.worship != nil {
		a.drawWorship(line)
		return
	}

	items := make([]ui.MenuItem, len(campMenuLabels))
	for i, l := range campMenuLabels {
		items[i] = ui.MenuItem{Label: a.tr.UI(l.key), Enabled: true}
	}
	items[2].Enabled = a.clock.CanSleep()
	if len(a.members) == 0 {
		for _, i := range []int{0, 1, 3, 4, 5, 7, 8, 9, 10, 11, 12} {
			items[i].Enabled = false
		}
	}
	byKey := make(map[string]ui.MenuItem, len(items))
	for i, label := range campMenuLabels {
		byKey[label.key] = items[i]
	}
	a.drawOperationMenu(dst, "camp", byKey)

	if c.message != "" {
		for _, l := range strings.Split(c.message, "\n") {
			line(l)
		}
		line("")
	}
}

func (a *app) drawEquipPicker(line func(string)) {
	c := a.camp
	if c.equipSlot < 0 {
		line(a.tr.UI("camp.equip.who"))
		line("")
		for i, m := range a.members {
			mark := "   "
			if i == c.equipMember {
				mark = " > "
			}
			line(fmt.Sprintf(a.tr.UI("camp.equip.row"), mark,
				textlayout.PadCells(m.Name, 10),
				textlayout.PadCells(a.weaponLabel(m), 8), m.ArmorRating()))
		}
		line("")
		line(a.tr.UI("camp.equip.keys1"))
		return
	}

	m := a.members[c.equipMember]
	line(fmt.Sprintf(a.tr.UI("camp.equip.which"), m.Name))
	line("")
	for i := 0; i < game.InventorySlots; i++ {
		mark := "   "
		if i == c.equipSlot {
			mark = " > "
		}
		it := m.Inventory[i]
		name, note := a.tr.UI("camp.equip.empty"), ""
		if !it.Empty() {
			name = a.itemLabel(it)
			switch {
			case i == m.EquippedWeapon:
				note = a.tr.UI("camp.equip.weapon")
			case i == m.EquippedArmor:
				note = a.tr.UI("camp.equip.armor")
			case !game.CanEquipAsWeapon(it) && !game.CanEquipAsArmor(it):
				note = a.tr.UI("camp.equip.cannot")
			// 型別上是護甲、但這個職業穿不上（原版：You're the wrong class.）。
			case game.CanEquipAsArmor(it) && !game.ClassCanWear(m.Class, it):
				note = a.tr.UI("camp.equip.heavy")
			}
		}
		line(fmt.Sprintf("%s%s%s", mark, textlayout.PadCells(name, 12), note))
	}
	line("")
	line(a.tr.UI("camp.equip.keys2"))
	// 換不上時的理由要在這裡印 —— 原本只寫進 c.message，而這一層
	// 沒有印它，玩家按下去只會看到「什麼都沒發生」。
	if c.message != "" {
		line("")
		line(c.message)
	}
}
