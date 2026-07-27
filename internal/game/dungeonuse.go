package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

// `U` 使用：拿手上一件對另一件用（原版 `222f:2088(0)`，`docs/re/95` §3.10）
//
// 兩段式。第一段選手上那件，第二段是三選一的小選單
// （`Use on: Character / Room / Quit`，`122f:1bf4`）：
//
//	Character  再選一件**身上的**（param 2）
//	Room       選一件**腳下的**（`222f:2da5`）
//
// 兩條路的比對規則一樣：目標的 `+4` 欄要等於手上那件的名字。
// 差別在**動作的後果**（`docs/re/95` §3.1 的對照表）。
//
// **地城解謎的全部內容在這裡** —— 18 條 `+4` 把 50 件串成主線的線索鏈。

// DungeonUseOutcome 是比對之後要做什麼。
type DungeonUseOutcome int

const (
	// DungeonUseNothing 是沒對上（或目標的 `+4` 空著）。
	// 原版 ds:0x2443 `Nothing happens`。**什麼都不消耗。**
	DungeonUseNothing DungeonUseOutcome = iota
	// DungeonUseDescribe 是動作碼 `D`：印一段敘述。多半是主線提示。
	DungeonUseDescribe
	// DungeonUseBecome 是動作碼 `N`：目標變成另一件。
	DungeonUseBecome
	// DungeonUseTeleport 是動作碼 `T`：隊伍傳送。
	DungeonUseTeleport
	// DungeonUsePassage 是動作碼 `P`：改一格地圖 tile。
	DungeonUsePassage
	// DungeonUseStory 是動作碼 `S`：主線劇情（`S1` 結局抉擇、`S2` 光之環）。
	DungeonUseStory
)

// DungeonUseResult 是 `U` 的結果。哪幾個欄位有意義由 Outcome 決定。
type DungeonUseResult struct {
	Outcome DungeonUseOutcome

	// Text 是 `D` 要印的那段敘述（動作碼後面的內容）。
	Text string
	// NewName 是 `N` 變成的那件道具的名字。
	NewName string
	// X／Y 是 `T` 的目的地或 `P` 要改的那一格。
	X, Y int
	// MapID 是 `T` 的目標子地圖。
	MapID byte
	// Tile 是 `P` 要寫進去的值。
	Tile byte
	// Story 是 `S` 的參數（1 或 2）。
	Story int
}

// dungeonUseWildcard 是 `+4` 的萬用字元（原版 ds:0x2441 的 `=`，`0x1824c`）。
//
// **出貨資料沒有用到** —— 18 個 `+4` 全部是別件道具的名字。
// 記在這裡是為了別把它當成一般名字比對：`=` 出現時任何東西都對得上。
const dungeonUseWildcard = "="

// UseDungeonItem 算出「拿 sourceName 對第 target 件用」的結果。
//
// **不改任何狀態。** 消耗、清位置表、放新道具、傳送、改 tile 全部是
// 呼叫端的事 —— 因為 `Use on: Room` 與 `Use on: Character` 的後果不同
// （`docs/re/95` §3.1 的對照表），而那個差別屬於介面那一層。
//
// 比對順序照原版（`0x18232`–`0x18263`）：
//
//  1. 目標的 `+4` 空著 → `Nothing happens`
//  2. `+4` 等於手上那件的名字 → 跑 `+5`
//  3. `+4` 是 `=` → 也跑 `+5`（萬用字元）
//  4. 否則 → `Nothing happens`
func UseDungeonItem(items gamedata.DungeonItems, sourceName string, target int) DungeonUseResult {
	nothing := DungeonUseResult{Outcome: DungeonUseNothing}
	if target < 0 || target >= len(items) {
		return nothing
	}
	it := items[target]
	if it.UseWith == "" {
		return nothing
	}
	if it.UseWith != sourceName && it.UseWith != dungeonUseWildcard {
		return nothing
	}

	param := it.ActionParam()
	switch it.Action() {
	case gamedata.ActionDescribe:
		return DungeonUseResult{Outcome: DungeonUseDescribe, Text: param}
	case gamedata.ActionBecome:
		return DungeonUseResult{Outcome: DungeonUseBecome, NewName: param}
	case gamedata.ActionTeleport:
		// `T564603` ＝ (56,46) 子地圖 3。
		//
		// > 原版是三組各取兩個字元丟 `atoi`（`0x183f8`–`0x18441`），
		// > 但**實際資料裡有五個字元的**：`T07203` ＝ (07,20) 子地圖 3。
		// > 那一筆的第三組只有 `3`，原版第二個字元讀到的是字串結尾的 NUL，
		// > `atoi("3")` 一樣是 3。所以「三組都是兩位數」是錯的判讀 ——
		// > 正確的描述是「前兩組固定兩位，第三組讀到結尾」，
		// > 與 `P`／`+3` 同一套（`parseTileSpec`）。
		x, y, m, ok := parseTileSpec(param)
		if !ok {
			return nothing
		}
		return DungeonUseResult{Outcome: DungeonUseTeleport, X: x, Y: y, MapID: m}
	case gamedata.ActionPassage:
		// `P04480` ＝ (04,48) → tile 0，同樣是五個字元。與 `T` 共用解析。
		x, y, tile, ok := parseTileSpec(param)
		if !ok {
			return nothing
		}
		return DungeonUseResult{Outcome: DungeonUsePassage, X: x, Y: y, Tile: tile}
	case gamedata.ActionStory:
		n, ok := storyNumber(param)
		if !ok {
			return nothing
		}
		return DungeonUseResult{Outcome: DungeonUseStory, Story: n}
	}
	return nothing
}

// storyNumber 取 `S1`／`S2` 的那一位數字（原版 `0x185b5` 只比第一個字元）。
func storyNumber(s string) (int, bool) {
	if len(s) < 1 || s[0] < '0' || s[0] > '9' {
		return 0, false
	}
	return int(s[0] - '0'), true
}
