package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 地城道具的規則層（`docs/spec/10-dungeon-items.md`）
//
// 手冊「物品」一節那六個指令裡的 `T` 拾取與 `D` 丟棄。兩件事都只動兩份狀態：
//
//	scenario.ItemLocTable   東西在地圖上的哪一格（子地圖 0 ＝ 在某人身上）
//	角色的道具槽             型別 0xfe ＋ 16 bytes 的名字
//
// **索引就是身分**：位置表第 i 筆 ↔ `gamedata.DungeonItems[i]`，
// 兩張表都是固定 50 格（`docs/re/95` §1 的 `cmp [bp-4], 0x32`）。
// 所以拿走時**不能刪除記錄**，只能把子地圖寫 0 —— 刪掉會讓後面每一件換身分。

// DungeonSpot 是腳下這一格的一件地城道具。
type DungeonSpot struct {
	// Index 是它在位置表／內容表裡的索引（兩張表共用）。
	Index int
	Item  gamedata.DungeonItem
}

// ItemsUnderfoot 列出隊伍腳下這一格有什麼（原版 `222f:2da5`）。
//
// 掃描順序照原版：由索引 0 往上，**掃滿 50 筆或湊滿 9 件才停**。
// 上限 9 是原版明寫的（手冊寫 10，以執行檔為準，見 spec 的未解表）。
func ItemsUnderfoot(t *scenario.ItemLocTable, items gamedata.DungeonItems,
	x, y, mapID byte) []DungeonSpot {

	if t == nil || mapID == scenario.ItemLocTaken {
		return nil
	}
	var out []DungeonSpot
	for i, r := range t.Records {
		if len(out) >= gamedata.DungeonItemsPerTile {
			break
		}
		if r.MapID != mapID || r.X != x || r.Y != y {
			continue
		}
		spot := DungeonSpot{Index: i}
		if i < len(items) {
			spot.Item = items[i]
		}
		out = append(out, spot)
	}
	return out
}

// DungeonRefusal 是拿不走的三種理由。
//
// **分成三種而不是回一串字**，因為三者的文字來源不同：
// `TakeFromData` 那句在 `FILES.DTT` 裡（要走 `tr.Event` 翻譯、對得上索引），
// 另外兩句是引擎自己的（走 `tr.UI` 的語意化 key）。
// 混成一個字串就沒辦法分開翻，而且會把資料原文寫死在程式裡。
type DungeonRefusal int

const (
	// TakeAllowed 代表拿得走。
	TakeAllowed DungeonRefusal = iota
	// TakeSilent 是 `+1` 欄只有一個 `*`。原版 ds:0x2793 `You can't`。
	TakeSilent
	// TakeFromData 是 `+1` 欄裡那句話，內容在 DungeonTakeResult.Message。
	TakeFromData
	// TakeNoRoom 是道具欄滿了。原版 ds:0x27b6 `No more room.`。
	TakeNoRoom
	// TakeGone 是那一筆已經不在地圖上了（重複觸發時的保險，原版沒有）。
	TakeGone
)

// DungeonTakeResult 是拾取的結果。
type DungeonTakeResult struct {
	OK bool
	// Refusal 是拿不走的理由，OK 時為 TakeAllowed。
	Refusal DungeonRefusal
	// Message 只有 TakeFromData 時有值 —— 就是 `+1` 欄的原文。
	Message string
	// Slot 是收下它的道具欄索引，失敗時為 −1。
	Slot int
}

// TakeDungeonItem 把位置表第 index 件收進 c 的道具欄（原版 `25be:0077`）。
//
// 原版的順序照抄（`0x19891`–`0x19a40`）：
//
//  1. `+1` 欄 == `*` → 印 `You can't`（那個星號是佔位，不是台詞）。
//  2. `+1` 欄非空 → 印那句話（`It is too heavy`）。
//  3. 掃 10 格找第一個 `0xff`，找不到 → 印 `No more room.`。
//  4. 找到 → 槽寫 `0xfe` ＋ 名字（NUL 結尾），位置表那一筆**三個 byte 全清**。
//
// 第 1、2 步的先後在原版就是這樣：`*` 先被 `strcmp` 攔下來，所以
// 它印的是通用句不是空字串。
//
// > 拿不起來時**選角色那一步已經走過了** —— 原版是先問「Character to take」
// > 再檢查有沒有空格。介面照這個順序，規則層不管。
//
// **位置表與道具欄要嘛一起改、要嘛都不改。** 中間失敗會造出「東西同時在
// 地上又在身上」或「憑空消失」——後者玩家完全看不出原因。
func TakeDungeonItem(c *Character, t *scenario.ItemLocTable,
	items gamedata.DungeonItems, index int) DungeonTakeResult {

	fail := func(r DungeonRefusal, msg string) DungeonTakeResult {
		return DungeonTakeResult{Refusal: r, Message: msg, Slot: -1}
	}
	if c == nil || t == nil || index < 0 || index >= len(items) || t.Taken(index) {
		return fail(TakeGone, "")
	}
	if r, msg := DungeonTakeRefusal(items[index]); r != TakeAllowed {
		return fail(r, msg)
	}
	it := items[index]
	slot := c.FreeSlot()
	if slot < 0 {
		return fail(TakeNoRoom, "")
	}
	c.Inventory[slot] = scenario.NewDungeonSlot(it.Name)
	t.Take(index)
	return DungeonTakeResult{OK: true, Slot: slot}
}

// DungeonTakeRefusal 回報這件東西拿不拿得走。
//
// **原版在問「哪個角色來拿」之前就先問這一句**（`0x19891` 的 `strcmp`
// 排在 `0x198ed` 的選人之前），所以介面也要能單獨問一次 ——
// 不能等 TakeDungeonItem 回來才知道。兩邊叫的是這同一支，
// 別在介面層另外抄一份 `Immovable == ""` 的判斷。
func DungeonTakeRefusal(it gamedata.DungeonItem) (DungeonRefusal, string) {
	switch it.Immovable {
	case "":
		return TakeAllowed, ""
	case dungeonItemSilentImmovable:
		return TakeSilent, ""
	}
	return TakeFromData, it.Immovable
}

// dungeonItemSilentImmovable 是「拿不走，但沒有話要說」的 `+1` 欄內容。
//
// 25 件拿不走的裡面有 18 件只放了一個 `*`。那不是台詞，是佔位 ——
// 原版明文對它做 `strcmp`（`0x19891`），命中就改印 `You can't`。
const dungeonItemSilentImmovable = "*"

// DungeonDropResult 是丟棄的結果。
//
// 三種失敗都是「介面不該讓你走到這裡」的保險（不是地城道具、名字認不出、
// 座標不合法），所以只有一個布林 —— 沒有需要分開翻譯的玩家訊息。
type DungeonDropResult struct {
	OK bool
	// Index 是它在位置表裡的那一筆，失敗時為 −1。
	Index int
}

// DropDungeonItem 把 c 的第 slot 格（必須是地城道具）丟在目前這一格。
//
// 原版是 `222f:2088(1)` → `122f:2845`（`0x18735`），三行就講完：
//
//	18748  逐 6 條掃字串表，strcmp(槽裡的名字, 表[j])   ; ← **靠名字回推索引**
//	18785  itemloc[i×3 + 0] = ds:0x50f0                ; 目前 X
//	18795  itemloc[i×3 + 1] = ds:0x50ee                ; 目前 Y
//	187a5  itemloc[i×3 + 2] = party[+0xa3]             ; 目前子地圖
//
// 手冊「丟棄地城道具，之後一定能在原地撿回」的機制就是這個：**改寫原本
// 那一筆**，不是新增（50 格是固定陣列，索引就是身分）。
//
// **靠名字回推索引是原版的作法，不是本專案的權宜。** 代價也照單全收：
// 名字不是唯一的（`Bed` 在第 1 與第 41 件各一個），原版取第一個命中的。
// 那兩件都拿不走，所以這個歧義在實際遊玩中碰不到 —— 但**別把它當成
// 「名字可以當主鍵」**，`U` 之後如果要加新的查找，先想清楚重名。
//
// 名字對不上就拒絕。原版沒有這道檢查，掃完 50 件會拿 `i == 51` 去寫
// `itemloc[153..155]` —— 寫在有效區之外，看不出症狀但是壞的。
//
// 這與營地的 `DropItem` 是**兩回事**：那一支明文拒絕地城道具
// （原版 `> 0xfd`，`docs/re/33` §3），因為地城道具只能在移動途中丟。
func DropDungeonItem(c *Character, t *scenario.ItemLocTable,
	items gamedata.DungeonItems, slot int, x, y, mapID byte) DungeonDropResult {

	fail := DungeonDropResult{Index: -1}
	if c == nil || t == nil || slot < 0 || slot >= InventorySlots {
		return fail
	}
	if mapID == scenario.ItemLocTaken {
		return fail
	}
	it := c.Inventory[slot]
	if !it.Dungeon() {
		return fail
	}
	index, ok := items.ByName(it.DungeonName)
	if !ok {
		return fail
	}
	if !t.Drop(index, x, y, mapID) {
		return fail
	}
	c.Inventory[slot] = scenario.InventorySlot{Type: scenario.SlotEmpty}
	return DungeonDropResult{OK: true, Index: index}
}

// ExamineDungeonItem 是 `E` 檢視：回傳 `+2` 欄那段敘述（原版 `222f:2088(3)`）。
//
// **檢視的是身上那件，不是腳下那件。** `Use`／`Drop`／`Examine` 三個
// 共用 `122f:1d23` 選「哪個角色的第幾格」，`Take:`／`Move:` 才是掃腳下
// （`docs/re/95` §3.8）。規格原本把 `E` 寫成「選一件腳下的」，那是推的。
//
// 第二個回傳值是「有沒有話可說」。`+2` 欄空著的 48／50 件之外那兩件，
// 原版印的是 `You see nothing special about the %s` —— 那句話由介面組，
// 因為它要塞譯名進去。
func ExamineDungeonItem(items gamedata.DungeonItems, name string) (string, bool) {
	i, ok := items.ByName(name)
	if !ok || items[i].Look == "" {
		return "", false
	}
	return items[i].Look, true
}

// CarriedDungeonItem 是某人身上的一件地城道具。
type CarriedDungeonItem struct {
	// Member 是隊伍索引，Slot 是道具欄索引。
	Member, Slot int
	Name         string
	// Index 是它在內容表裡的索引；名字對不上時為 −1
	// （存檔被改過、或名字被截斷都會走到這裡）。
	Index int
}

// CarriedDungeonItems 列出全隊身上的地城道具。
//
// `U`（用一件對另一件）與 `D`（丟棄）的選單都從這裡來 ——
// **兩邊共用同一份列舉**，不要各自掃一次道具欄。
func CarriedDungeonItems(party []Character, items gamedata.DungeonItems) []CarriedDungeonItem {
	var out []CarriedDungeonItem
	for m := range party {
		for s := 0; s < InventorySlots; s++ {
			it := party[m].Inventory[s]
			if !it.Dungeon() {
				continue
			}
			idx, ok := items.ByName(it.DungeonName)
			if !ok {
				idx = -1
			}
			out = append(out, CarriedDungeonItem{
				Member: m, Slot: s, Name: it.DungeonName, Index: idx,
			})
		}
	}
	return out
}
