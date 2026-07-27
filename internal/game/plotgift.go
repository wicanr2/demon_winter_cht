package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 劇情送的道具（原版 `25be:11ff` ＝ `0x1a9df`，`docs/re/65` §3.2）
//
// 好幾個地點劇情 case 共用這一支：印一句話 → 問「哪個角色來拿」→
// 把一件**寫死規格的道具**塞進第一個空槽 → 在 trailer `+0xb3 + id`
// 記一格「拿過了」。
//
// 道具不是從 `ITEMS.DAT` 抄一筆，而是**逐欄位寫死**（型別、附帶法術、
// 材質類別、附魔、驅邪率、已鑑定旗標）。所以它們是獨一無二的劇情裝備，
// 不是商店買得到的東西。

// PlotGiftID 是送道具常式的參數，同時也是旗標陣列的索引。
type PlotGiftID int

// PlotGiftBlacksmith 是地點劇情 case 4（鐵匠鋪）送的那把武器。
//
// **只認出這一格。** 那支常式的跳表有 7 個 param，其餘六個掛在還沒接的
// 劇情 case 上（`docs/re/65` §3 的 case 6／7／8／9／12／13），
// 沒有呼叫端就沒辦法確認誰是誰 —— 所以不先把表填滿。
const PlotGiftBlacksmith PlotGiftID = 4

// blacksmithSword 是鐵匠那把武器的完整規格（`0x1ab33`–`0x1ab61`）。
//
//	[+0x00] = 0x05   ITEMS.DAT 第 5 件 ＝ broad sword
//	[+0x01] = 0x94   附帶法術 A（最高位 0x80 是旗標 → 法術 0x14）
//	[+0x02] = 0x04   附帶法術 A 的強度
//	[+0x0d] = 0x14   驅邪成功率 20
//	[+0x0e] = 0x09   附魔 ＝ 9 − 10 ＝ **−1**
//	[+0x0f] = 0x08   材質類別 8（估價倍率 ×75，最高一級）
//	[+0x10] = 0x01   已鑑定
//
// **附魔是 −1，不是筆誤。** 共用那一段先把 `+0x0e` 寫成 `0x0a`（＝0），
// param 4 的分支再覆寫成 `0x09`。一把 −1 的闊劍配一個強度 4 的附帶法術、
// 材質類別拉到頂 —— 台詞說它是「為了要殺 Xeres 的人打造的」，
// 那個 −1 大概是代價。**照抄，不要「修正」成 +1。**
var blacksmithSword = scenario.InventorySlot{
	Type:        0x05,
	SpellA:      0x94,
	SpellAPower: 0x04,

	ExorciseResist: 0x14,
	Enchant:        -1,
	MaterialClass:  0x08,
	Identified:     true,
}

// PlotGiftTaken 回報這件劇情道具拿過了沒。
func PlotGiftTaken(s *scenario.SaveGame, id PlotGiftID) bool {
	if s == nil || id < 0 || int(id) >= scenario.PlotGiftCount {
		return true // 不知道就當拿過了 —— 寧可少給，不要無限複製
	}
	return s.PlotGifts[id] != 0
}

// PlotGiftResult 是領取的結果。
type PlotGiftResult struct {
	OK bool
	// Slot 是收下它的道具欄索引，失敗時為 −1。
	Slot int
	// Full 為真代表那名角色的道具欄滿了。
	Full bool
}

// TakePlotGift 把劇情道具交給 c，並記下「拿過了」。
//
// **旗標與道具要嘛一起寫、要嘛都不寫。** 欄位滿了就整件事不算 ——
// 先記旗標再發現放不下的話，那件道具就永遠拿不到了，
// 而畫面上只會顯示「道具欄滿了」，看不出進度已經被燒掉。
func TakePlotGift(s *scenario.SaveGame, c *Character, id PlotGiftID) PlotGiftResult {
	if s == nil || c == nil || PlotGiftTaken(s, id) {
		return PlotGiftResult{Slot: -1}
	}
	spec, ok := plotGiftSpec(id)
	if !ok {
		return PlotGiftResult{Slot: -1}
	}
	slot := c.FreeSlot()
	if slot < 0 {
		return PlotGiftResult{Slot: -1, Full: true}
	}
	c.Inventory[slot] = spec
	s.PlotGifts[id] = 1
	return PlotGiftResult{OK: true, Slot: slot}
}

// plotGiftSpec 是 id → 道具規格。**只有一格** —— 見 PlotGiftBlacksmith。
func plotGiftSpec(id PlotGiftID) (scenario.InventorySlot, bool) {
	if id == PlotGiftBlacksmith {
		return blacksmithSword, true
	}
	return scenario.InventorySlot{}, false
}
