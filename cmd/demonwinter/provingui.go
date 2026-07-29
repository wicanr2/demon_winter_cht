package main

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/i18n"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
)

// 十間試煉室（地點劇情 case 9，地圖 2，`docs/re/101` §2）。
//
// 一個職業一間。走進去房間被顏色的光灌滿、一個聲音說話，
// 然後**只有那個職業的人上場** —— 陣型被借走，其他人站在旁邊看。
//
// 流程分三條：
//
//	隊伍裡沒有那個職業      → 「既然你沒有…我就讓你過。」直接過關
//	有但全都倒了            → 「等你的…好了再來。」**趕出去**，不給旗標
//	有能上場的              → 照 Trial 走（七間打一場、三間只給一句話）
//
// 打的那三分之七要等**戰勝**才算過（旗標在 battleui 的收尾裡寫）。

// enterProvingRoom 是 case 9 的入口。
func (a *app) enterProvingRoom() {
	idx := game.ProvingRoomAt(a.party.X(), a.party.Y())
	if idx < 0 {
		// 座標不在那十格裡 —— 原版此時 idx 留 −1，`+0xab` 會被寫成 0，
		// 等於什麼都沒發生。這裡明確不做事。
		a.trace.note("試煉室：(%d,%d) 不在十格裡", a.party.X(), a.party.Y())
		return
	}
	room := game.ProvingRooms[idx]

	entry, fighters := game.EnterProvingRoom(idx, a.members)
	classLabel := a.label(className, int(room.Class))

	// 開場敘述：房間名 ＋ 顏色的光 ＋ 那個聲音。三段都是原版的字串。
	intro := fmt.Sprintf(a.tr.UI("proving.room"), classLabel) + "\n" +
		fmt.Sprintf(a.tr.UI("proving.colour"),
			provingColour(a.tr, room.Colour)) + "\n" +
		a.tr.UI("proving.flood")

	switch entry {
	case game.ProvingFreePass:
		// 原版 `0x1f56`：「Since you have no %s / I will let you pass.」
		a.box = ui.NewMixedTextBox(intro + "\n" +
			fmt.Sprintf(a.tr.UI("proving.nopass"), classLabel))
		a.passProvingRoom(idx)
		a.trace.note("試煉室 %d（%s）：隊伍裡沒有這個職業，直接過", idx, classLabel)
		return

	case game.ProvingComeBackWhenWell:
		// 原版 `0x1ed1`：「Return when your %s is well.」→ 趕到 (42,19)。
		a.box = ui.NewMixedTextBox(intro + "\n" +
			fmt.Sprintf(a.tr.UI("proving.unwell"), classLabel))
		a.party.TeleportTo(game.ProvingEjectTo.X, game.ProvingEjectTo.Y)
		a.save.ProvingRoom = 0
		a.trace.note("試煉室 %d（%s）：人都倒了，趕到 (%d,%d)",
			idx, classLabel, game.ProvingEjectTo.X, game.ProvingEjectTo.Y)
		return
	}

	// 有人能上場。原版此時已經把陣型借走了。
	a.save.ProvingRoom = byte(idx + 1)

	switch room.Trial {
	case game.ProvingTaunt:
		// 盜賊那間：`Let the %s / Be ashamed if he fell into that trap!`
		a.box = ui.NewMixedTextBox(intro + "\n" +
			fmt.Sprintf(a.tr.UI("proving.taunt"), classLabel))
		a.passProvingRoom(idx)
		a.trace.note("試煉室 %d（%s）：只有一句話", idx, classLabel)

	case game.ProvingBlessing:
		// 靈視者那間：`Let the %s / prove his worth / in the times ahead.`
		a.box = ui.NewMixedTextBox(intro + "\n" +
			fmt.Sprintf(a.tr.UI("proving.blessing"), classLabel))
		a.passProvingRoom(idx)
		a.trace.note("試煉室 %d（%s）：只有一句話", idx, classLabel)

	case game.ProvingLore:
		// 學者那間：`Let the %s / know this.` ＋ 一段符文密語。
		// **提示本身不翻**（worklist D2）—— 那是要玩家自己解的符文。
		a.box = ui.NewMixedTextBox(intro + "\n" +
			fmt.Sprintf(a.tr.UI("proving.lore"), classLabel))
		a.pendingRunes = game.ProvingLoreHint
		a.passProvingRoom(idx)
		a.trace.note("試煉室 %d（%s）：給了符文提示", idx, classLabel)

	default:
		// 打一場。**陣型先備份再借走** —— 只有那幾個人上場。
		a.save.FormationBackup = a.save.Formation
		a.save.Formation = game.ProvingFormation(fighters)
		a.box = ui.NewMixedTextBox(intro + "\n" +
			fmt.Sprintf(a.tr.UI("proving.fight"), classLabel))
		a.pendingIDs = room.Monsters
		a.trace.note("試煉室 %d（%s）：%d 人上場，怪 %v",
			idx, classLabel, len(fighters), room.Monsters)
	}
}

// passProvingRoom 記下過關並把那一格消耗掉。
//
// 原版 `0x0f66f`：寫 `+0x8a + idx` → 清 `+0xab` → 查回那筆 `nSS.DAT`
// 記錄把 attr 清成 0（也就是 A8 的 `Consume`）→ 還原陣型。
func (a *app) passProvingRoom(idx int) {
	game.PassProvingRoom(a.save, idx)
	a.save.ProvingRoom = 0
	a.consumeSpecialHere()
}

// finishProvingRoom 是戰勝之後的收尾（原版 `0x0e1bc`）。
//
// **只有勝利才走這裡。** 陣型的還原是另一支（勝敗都要做），見
// restoreProvingFormation。
func (a *app) finishProvingRoom() {
	if a.save.ProvingRoom == 0 {
		return
	}
	idx := int(a.save.ProvingRoom) - 1
	a.passProvingRoom(idx)
	if idx >= 0 && idx < len(game.ProvingRooms) {
		a.trace.note("試煉室 %d（%s）：戰勝過關",
			idx, a.label(className, int(game.ProvingRooms[idx].Class)))
	}
}

// restoreProvingFormation 把借走的陣型還回去。
//
// **勝敗都要做**（原版 `0x0e004` 在收尾常式的第一行，還沒判勝負）。
// 漏掉的話打完之後隊伍只剩那幾個人上場，而畫面上完全看不出來。
func (a *app) restoreProvingFormation() {
	if a.save.ProvingRoom == 0 {
		return
	}
	a.save.Formation = a.save.FormationBackup
	a.trace.note("試煉室：陣型還原成 %v", a.save.Formation)
}

// provingColour 是房間顏色的譯名（原版 `ds:0x0c86` 的十個字串）。
//
// 顏色與職業一對一，所以它是**房間的識別**而不只是裝飾 ——
// 玩家拿攻略對畫面時是靠顏色認房間的。
func provingColour(tr *i18n.Translator, c string) string {
	if _, ok := provingColourNames[c]; ok {
		return tr.UI("proving.colour." + c)
	}
	return c
}

// ui:dynamic proving.colour. —— 由上面那一支用 `"proving.colour."+c` 查表。
var provingColourNames = map[string]string{
	"green": "綠色", "silver": "銀色", "brown": "棕色", "violet": "紫色",
	"white": "白色", "grey": "灰色", "crimson": "緋紅", "black": "黑色",
	"blue": "藍色", "beige": "米色",
}

// consumeSpecialHere 把腳下那一筆 `nSS.DAT` 記錄的屬性清 0。
//
// 這就是 A8 的 `Consume` 在**事件表那一側**的第一個呼叫端
// （原版 `0x0f6a3`／`0x0e1f4` 的 `[0x52f8]->attr = 0`）——
// 那一格從此不再是事件，重讀地圖也不會回來，因為它改的是記憶體裡的
// `nSS` 表而不是地圖。
func (a *app) consumeSpecialHere() {
	st := a.special[a.mapID]
	if st == nil {
		return
	}
	hit := st.Lookup(byte(a.party.X()), byte(a.party.Y()))
	if hit == nil {
		return
	}
	st.Consume(hit.Index)
}
