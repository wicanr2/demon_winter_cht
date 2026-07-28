package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 買賣與裝備。
//
// 價格公式在 economy.go（早就解出來了）；這裡補的是「買下去之後會發生什麼」
// —— 扣錢、找空格、放進去，以及換裝。
//
// # 買到的是什麼樣的道具
//
// 效果索引與強度一律 0，也就是**沒有可用效果的平凡裝備**。理由不是偷懶：
// 原版的效果是**掉寶時生成**的（`docs/re/25` §3 的兩層擲點），店裡賣的
// 30 種基本道具在 `ITEMS.DAT` 裡沒有帶效果值。而且起始存檔那五個人身上
// 每一件的 `+0x05`–`+0x08` 都是 0 —— 那就是原版平凡裝備的樣子。
//
// 掉寶生成那條路徑本專案還沒實作（`docs/re/25` §3 有一角未解），
// 所以現在買不到魔法道具。
//
// # 裝備
//
// 角色記錄的 `+0x100`／`+0x101` 存的是「裝備在哪一格」的**槽位索引**，
// 不是道具型別（`docs/re/16` §4.3 的過濾規則就是拿槽位索引去比對的）。
// 所以換裝只要改那兩個索引，道具本身不動。

// TradeResult 說明一次買賣為什麼沒成。
type TradeResult struct {
	// OK 為 true 代表成交。
	OK bool
	// Reason 是沒成的原因。
	Reason string
	// Gold 是成交後的金幣。
	Gold int
	// Slot 是買到的道具放進哪一格（賣出時是被清掉的那一格）。
	Slot int
	// Member 是哪一名隊員的道具欄。
	Member int
}

// FreeSlot 回傳這名角色第一個空格，沒有空格回 −1。
func (c *Character) FreeSlot() int {
	for i := range c.Inventory {
		if c.Inventory[i].Empty() {
			return i
		}
	}
	return -1
}

// Buy 讓隊伍買下一件道具。
//
// price 是議價後的實付價（呼叫端從 TownVisit.Price 拿）。
// 道具會放進**第一個有空格的隊員**身上。
func Buy(members []Character, gold, price, itemType int) TradeResult {
	if price > gold {
		return TradeResult{Reason: "金幣不夠", Gold: gold}
	}
	for i := range members {
		slot := members[i].FreeSlot()
		if slot < 0 {
			continue
		}
		members[i].Inventory[slot] = scenario.InventorySlot{
			Type: byte(itemType),
			// 店裡的貨都是平凡裝備，沒有效果、沒有附魔、已鑑定。
			Identified: true,
		}
		return TradeResult{OK: true, Gold: gold - price, Slot: slot, Member: i}
	}
	return TradeResult{Reason: "全隊的道具欄都滿了", Gold: gold}
}

// Sell 讓某名隊員賣掉一格道具。
//
// price 是賣價（`Economy.SellPrice`，原版是底價的一半，不受議價影響）。
// **裝備中的那一件不能賣** —— 賣掉會讓裝備索引指向空格。
func Sell(members []Character, gold, member, slot, price int) TradeResult {
	if member < 0 || member >= len(members) {
		return TradeResult{Reason: "沒有這名隊員", Gold: gold}
	}
	c := &members[member]
	if slot < 0 || slot >= InventorySlots || c.Inventory[slot].Empty() {
		return TradeResult{Reason: "那一格是空的", Gold: gold}
	}
	if slot == c.EquippedWeapon || slot == c.EquippedArmor {
		return TradeResult{Reason: "身上正在用的東西不能賣", Gold: gold}
	}
	c.Inventory[slot] = scenario.InventorySlot{Type: slotEmptyType}
	return TradeResult{OK: true, Gold: gold + price, Slot: slot, Member: member}
}

// slotEmptyType 是空格的道具型別。與 scenario 的 slotEmpty 同值，
// 那邊沒有匯出，這裡自己定一個並由測試釘住兩者一致。
const slotEmptyType = 0xff

// 裝備分類的邊界。與 item.go 的 armorFirstIndex／consumableFirstType 同源
// （`docs/re/16` §4.3：型別 0–7 武器、8–12 護甲、>= 13 消耗品）。

// CanEquipAsWeapon 回報這件道具能不能當武器裝備。
func CanEquipAsWeapon(it scenario.InventorySlot) bool {
	return !it.Empty() && int(it.Type) < armorFirstIndex
}

// CanEquipAsArmor 回報這件道具能不能當護甲裝備。
func CanEquipAsArmor(it scenario.InventorySlot) bool {
	t := int(it.Type)
	return !it.Empty() && t >= armorFirstIndex && t < consumableFirstType
}

// Equip 把某一格裝備到武器手或護甲位，回傳是否成功與原因。
//
// 換裝只改角色記錄的槽位索引（`+0x100`／`+0x101`），道具本身不動。
func (c *Character) Equip(slot int) (bool, string) {
	if slot < 0 || slot >= InventorySlots {
		return false, "沒有這一格"
	}
	it := c.Inventory[slot]
	switch {
	case it.Empty():
		return false, "那一格是空的"
	case CanEquipAsWeapon(it):
		c.EquippedWeapon = slot
		return true, ""
	case CanEquipAsArmor(it):
		// 型別過了還要過職業（`1000:283d`）。原版的訊息是
		// `You're the wrong class.`（`ds:0x044a`）。
		if !ClassCanWear(c.Class, it) {
			return false, "你的職業穿不了這麼重的護甲"
		}
		c.EquippedArmor = slot
		return true, ""
	}
	return false, "這件不是武器也不是護甲"
}
