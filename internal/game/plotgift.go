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

// 跳表七個 param 的分派（`cs:0x139a`，`docs/re/99` §2）。
//
// **0–4 已經有呼叫端、規格也對上了；5／6 還沒有。** 前四個是兵器庫的
// 四座台座（case 3 用座標算出 0–3），第 5 個是鐵匠鋪（case 4）。
// param 5／6 共用同一段程式（型別 ＝ `param + 0x17` ＝ 28／29），
// 掛在還沒讀的 case 6／7／8／9 上，**沒有呼叫端就先不填**。
const (
	// PlotGiftArmoryChain 是兵器庫的鏈甲（Silver Suit of Chain Mail）。
	PlotGiftArmoryChain PlotGiftID = 0
	// PlotGiftArmoryMace 是兵器庫的釘頭錘（a Mace）。
	PlotGiftArmoryMace PlotGiftID = 1
	// PlotGiftArmoryDagger 是兵器庫的水晶匕首（a Crystal Dagger）。
	PlotGiftArmoryDagger PlotGiftID = 2
	// PlotGiftArmorySword 是兵器庫的冰藍短劍（an Icy Blue Short Sword）。
	PlotGiftArmorySword PlotGiftID = 3
	// PlotGiftBlacksmith 是地點劇情 case 4（鐵匠鋪）送的那把闊劍。
	PlotGiftBlacksmith PlotGiftID = 4
	// PlotGiftDemonCrystal 是地點劇情 case 7（地圖 4 的 (7,4)）送的惡魔水晶
	// （`docs/re/101` §4）。分支只寫型別 ＝ `param + 0x17` ＝ 28，
	// 其餘欄位留共用前置段的值。
	PlotGiftDemonCrystal PlotGiftID = 5
)

// plotGiftTypeBase 是 param 5／6 那一段共用的型別基底
// （`0x1ab6c` 的 `mov ax,[bp+6] / add ax,0x17`）。
const plotGiftTypeBase = 0x17

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

// armoryGifts 是兵器庫四座台座的道具（`0x1aadc`／`0x1aaee`／`0x1ab08`／`0x1ab1a`）。
//
// 共用的前置段（`0x1aaa2`–`0x1aacb`）先把 17 bytes 清 0，
// 再寫 `+0x0e = 0x0a`（附魔 ＝ 0）與 `+0x10 = 1`（已鑑定），
// 各 param 的分支只覆寫自己那幾格。所以**沒被覆寫的欄位就是 0**，
// 包含型別 —— param 2 的匕首正是靠這個拿到 `ITEMS.DAT` 第 0 件。
//
// # 名字自己招供
//
// 這四件的顯示字串就在跳表附近（`ds:0x27cc` 的四個遠指標）。
// 把型別、材質類別、附帶法術三樣分別對回去，**六處零誤差**：
//
//	param 0  型別 10 ＝ chain        材質 3 ＝ Silver    → "Silver Suit of Chain Mail"
//	param 1  型別  3 ＝ mace         材質 0 ＝（無前綴）  → "a Mace"
//	param 2  型別  0 ＝ dagger       材質 5 ＝ Crystal   → "a Crystal Dagger"
//	param 3  型別  2 ＝ short sword  附帶法術 15 ＝ 寒顫（Ice）→ "an Icy Blue Short Sword"
//
// 材質前綴表見 `docs/re/48` §3、法術表見 `docs/re/15`。
// `docs/re/65` §3.2 當時只讀出這些原始位元組，並寫著「沒有呼叫端就確認不了
// 誰是誰」—— 呼叫端是 case 3，四座台座各對一格。
var armoryGifts = map[PlotGiftID]scenario.InventorySlot{
	// 銀鏈甲。除了材質拉到 Silver 之外沒有任何附加效果。
	PlotGiftArmoryChain: {
		Type:          0x0a,
		MaterialClass: 3,
		Identified:    true,
	},
	// 釘頭錘。材質是普通的，特別之處在那組**特效條件旗標**。
	// `CondA`／`EffectAByte` 照抄原始位元組 —— 條件 0x12 不是
	// 估價那條路認得的 0x15，語意還沒定案（`docs/re/46` §4），
	// 所以不推導成 `WeaponEffect`。
	PlotGiftArmoryMace: {
		Type:        0x03,
		CondA:       0x12,
		EffectAByte: 0x0c,
		Identified:  true,
	},
	// 水晶匕首。**型別 0 是「沒被覆寫」的結果，不是漏寫。**
	PlotGiftArmoryDagger: {
		Type:          0x00,
		MaterialClass: 5,
		Enchant:       1, // 分支覆寫 +0x0e = 0x0b → 11 − 10
		Identified:    true,
	},
	// 冰藍短劍。附帶法術 15（寒顫，Ice 系）強度 4 —— 名字裡的
	// 「Icy Blue」就是這個，不是材質前綴。
	PlotGiftArmorySword: {
		Type:        0x02,
		SpellA:      0x0f,
		SpellAPower: 0x04,
		Identified:  true,
	},
}

// demonCrystal 是 case 7 送的惡魔水晶（`0x1ab6c`）。
//
// 那一段只寫一個欄位：`[+0x00] = param + 0x17`。param 5 → 型別 28，
// 而 `ITEMS.DAT` 第 28 件就是 `Demon Crystal`。其餘欄位是共用前置段的值
// （附魔 0、已鑑定），**沒有附帶法術也沒有效果** —— 它是劇情道具不是裝備。
//
// param 6（永恆之寶珠，型別 29）走同一段程式，但**不放進這張表** ——
// 它的「拿過了」旗標落在 `+0xb9`，那個 byte 已經是劇情階段
// （`docs/re/101` §3.2），要由 case 8 直接推進階段，不要多一格旗標。
var demonCrystal = scenario.InventorySlot{
	Type:       plotGiftTypeBase + byte(PlotGiftDemonCrystal),
	Identified: true,
}

// plotGiftSpec 是 id → 道具規格。param 6 刻意不在表內（見 demonCrystal）。
func plotGiftSpec(id PlotGiftID) (scenario.InventorySlot, bool) {
	switch id {
	case PlotGiftBlacksmith:
		return blacksmithSword, true
	case PlotGiftDemonCrystal:
		return demonCrystal, true
	}
	s, ok := armoryGifts[id]
	return s, ok
}
