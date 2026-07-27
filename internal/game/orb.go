package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 恆世寶珠（地點劇情 case 8，地圖 2 的 (42,28)，`docs/re/101` §3）
//
// 十間試煉室全過之後才拿得到。它是主線的引爆點：拿到手的那一刻
// 劇情階段從 0 推到 1，**下一次睡覺就會播馬利馮的預言**
//（`docs/re/80`／`81`；那段預言的第一行正是 `The Orb of Evertime now is yours`）。
//
// # 為什麼不用劇情道具旗標
//
// 原版的閘門與旗標都是 `+0xb9`，而那個 byte 同時就是劇情階段 ——
// 「送道具」常式寫的 `party[0xb3 + param] = 1` 在 param 6 剛好落在那裡。
// 那不是溢位，是刻意共用。但引擎**不跟著共用**：同一個 byte 有兩個名字
// 才是真正會漂的東西，所以這裡直接讀寫 `PlotStage`。

// OrbItemType 是恆世寶珠在 `ITEMS.DAT` 的索引（`param + 0x17` ＝ 29）。
const OrbItemType = 29

// orbOfEvertime 是那顆寶珠。共用前置段只給它「附魔 0、已鑑定」，
// param 6 的分支只寫型別 —— 沒有效果、沒有材質，它是劇情道具。
var orbOfEvertime = scenario.InventorySlot{
	Type:       OrbItemType,
	Identified: true,
}

// OrbResult 是走上那一格的結果。
type OrbResult int

const (
	// OrbAlreadyTaken：拿過了（劇情階段已經不是 0）。原版只放個音效。
	OrbAlreadyTaken OrbResult = iota
	// OrbNotYet：十間試煉室還沒全過。原版也只放個音效，**什麼都不說**。
	OrbNotYet
	// OrbTaken：拿到了，劇情階段推到 1。
	OrbTaken
	// OrbNoRoom：那名角色的道具欄滿了 —— 階段不推進，還能再來。
	OrbNoRoom
)

// OrbAvailable 回報這一格現在給不給寶珠。
//
// 原版兩道閘門（`0x1a267`、`0x1a2af`）：階段不是 0 就當拿過了；
// 十間試煉室數出任何一間沒過就不給。
func OrbAvailable(s *scenario.SaveGame) OrbResult {
	if s == nil {
		return OrbAlreadyTaken
	}
	if s.PlotStage != PlotBeforeArrival {
		return OrbAlreadyTaken
	}
	if !ProvingRoomsCleared(s) {
		return OrbNotYet
	}
	return OrbTaken
}

// TakeOrbOfEvertime 把寶珠交給 c 並把劇情階段推到 1。
//
// **道具與階段要嘛一起寫、要嘛都不寫**（同 TakePlotGift 的理由）：
// 先推階段再發現放不下的話，那顆寶珠就永遠拿不到了，
// 而畫面上只會顯示「放不下」，看不出主線已經被推進去了。
func TakeOrbOfEvertime(s *scenario.SaveGame, c *Character) (OrbResult, int) {
	if r := OrbAvailable(s); r != OrbTaken {
		return r, -1
	}
	if c == nil {
		return OrbAlreadyTaken, -1
	}
	slot := c.FreeSlot()
	if slot < 0 {
		return OrbNoRoom, -1
	}
	c.Inventory[slot] = orbOfEvertime
	s.PlotStage = PlotArrivalDue
	return OrbTaken, slot
}
