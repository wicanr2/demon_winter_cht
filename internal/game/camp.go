package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 紮營選單的道具管理：丟棄與轉手（`docs/re/33`）。
//
// 原版紮營選單有 **14 項**（不是 13）：
// Party／Reorder／Sleep／Identify／Worship／Xorcise／View land／
// Trade／Drop／Equip／Use／Hunt／Cast／Quit，快捷鍵 `PRSIWXVTDEUHCQ`。
// 這一輪接上 Drop 與 Trade —— 兩者的規則已經完整讀出來，沒有留白。

// 特殊道具型別。一般道具是 0–25（`ITEMS.DAT` 的 26 種），
// 26 以上是劇情／地城物品，各有各的限制。
const (
	// ItemTypeDungeon 是地城道具。**在營地丟不掉**（原版
	// "Dungeon items cannot be dropped in camp"），要到地城裡才行。
	ItemTypeDungeon = 0xfe

	// itemTypeNoDrop 這一型無論如何都不能丟（原版 `1000:21f2`）。
	itemTypeNoDrop = 0x1c
	// itemTypeGatedDrop 這一型要 `party+0xc1` 不為 0 才丟得掉
	// （原版 `1000:21f8`）。那個旗標的語意還沒解，起始存檔是 0。
	itemTypeGatedDrop = 0x1d
)

// ItemMoveResult 是一次丟棄或轉手的結果。
type ItemMoveResult struct {
	OK bool
	// Reason 是沒成功的原因。
	Reason string
	// Slot 是收方收下的那一格（轉手成功時才有意義），其餘情況為 −1。
	Slot int
}

// DropItem 丟掉某一格的道具。
//
// 原版的四道關卡（`1000:2191`–`0x222e`），順序照抄：
//
//  1. 型別 > 0xfd → 拒絕。這一條同時蓋住**空格（0xff）與地城道具（0xfe）**，
//     兩者印同一句 "Dungeon items cannot be dropped in camp"。本專案把兩者
//     分開講 —— 對空格說「地城道具丟不掉」只會讓人困惑，那是原版的隨手複用。
//  2. 這一格正裝備著（武器或護甲）→ "That item is equipped!"
//  3. 型別 0x1c，或型別 0x1d 且 `flagC1 == 0` → "I don't think you want to do that!"
//  4. 都過了 → 型別寫成 0xff，**其餘 16 bytes 不動**
//
// 第 4 點是原版的作法，本專案照做：`saveenc` 對「型別沒變」的槽只覆蓋已解欄位，
// 清空的那一格因此留著前一件的殘值 —— 與原版存檔的樣子一致。
func DropItem(c *Character, slot int, flagC1 byte) ItemMoveResult {
	if slot < 0 || slot >= InventorySlots {
		return ItemMoveResult{Reason: "沒有這一格", Slot: -1}
	}
	it := c.Inventory[slot]
	switch {
	case it.Empty():
		return ItemMoveResult{Reason: "這一格是空的", Slot: -1}
	case it.Type == ItemTypeDungeon:
		return ItemMoveResult{Reason: "地城道具不能在營地丟棄", Slot: -1}
	case slot == c.EquippedWeapon || slot == c.EquippedArmor:
		return ItemMoveResult{Reason: "那件裝備還在身上", Slot: -1}
	case it.Type == itemTypeNoDrop,
		it.Type == itemTypeGatedDrop && flagC1 == 0:
		return ItemMoveResult{Reason: "我看你不會想這麼做", Slot: -1}
	}
	c.Inventory[slot] = scenario.InventorySlot{Type: scenario.SlotEmpty}
	return ItemMoveResult{OK: true, Slot: -1}
}

// GiveItem 把 from 的第 slot 格交給 to。
//
// 原版（`1000:628`–`0x7ed`）與 Drop 的差別有三個，都不是細節：
//
//   - **裝備中只是不給選，不是放棄。** Drop 撞到裝備品就結束，
//     Trade 印完 "That item is equipped" 會回到選道具那一步。
//     這裡的介面差異交給呼叫端，規則層一律回傳原因。
//   - **地城道具沒有限制。** 丟不掉但給得出去 —— 原版這一段完全沒有型別檢查。
//   - **收方滿了就失敗**（"%s is full"），道具留在原處。
//
// 搬的是整格 17 bytes，來源那一格只把型別寫成 0xff（同 Drop）。
func GiveItem(from, to *Character, slot int) ItemMoveResult {
	if slot < 0 || slot >= InventorySlots {
		return ItemMoveResult{Reason: "沒有這一格", Slot: -1}
	}
	if from == to {
		return ItemMoveResult{Reason: "不能給自己", Slot: -1}
	}
	it := from.Inventory[slot]
	switch {
	case it.Empty():
		return ItemMoveResult{Reason: "這一格是空的", Slot: -1}
	case slot == from.EquippedWeapon || slot == from.EquippedArmor:
		return ItemMoveResult{Reason: "那件裝備還在身上", Slot: -1}
	}
	dst := to.FreeSlot()
	if dst < 0 {
		return ItemMoveResult{Reason: to.Name + " 的道具欄滿了", Slot: -1}
	}
	to.Inventory[dst] = it
	from.Inventory[slot] = scenario.InventorySlot{Type: scenario.SlotEmpty}

	// 交出去的那一格若是裝備索引指向的位置，索引要跟著失效 ——
	// 原版不會發生（裝備中的格子選不到），但規則層擋不住呼叫端亂傳。
	from.unequipIfSlot(slot)
	return ItemMoveResult{OK: true, Slot: dst}
}

// unequipIfSlot 在指定格被清空時解除裝備索引。
func (c *Character) unequipIfSlot(slot int) {
	if c.EquippedWeapon == slot {
		c.EquippedWeapon = -1
	}
	if c.EquippedArmor == slot {
		c.EquippedArmor = -1
	}
}
