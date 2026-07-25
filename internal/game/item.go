package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 戰鬥中「使用道具」的可用清單。
//
// 過濾規則來自 docs/re/16 §4.3（已驗證）：
//
//	skip = 空槽
//	    || 武器類（type < 8）且不是目前裝備的那一格
//	    || 護甲類（type 8–12）且不是目前裝備的那一格
//
// 語意是「Use 拿來觸發**已裝備**武器／護甲的特殊能力」，
// 不是拿沒裝備的武器當道具用；一般消耗品（type >= 13）不受限制。
//
// **原版還有兩條過濾條件沒有實作**：`usable_flag == 0` 與
// 「兩個充能欄位相等」。那兩個欄位在 17 bytes 的存檔槽裡沒有定位到
// （反組譯的 `0x11`/`0x12` 超出槽長，應該是載入後的記憶體結構），
// 而且原文自己標了「假設，語意不明」。少實作會讓清單**偏寬**——
// 多出幾個實際上不能用的道具，比漏掉可用的好查。
const (
	// consumableFirstType 是「一般消耗品」的型別下界。
	// 8–12 是護甲，13 以上是藥水／卷軸一類。
	consumableFirstType = 13
)

// UsableItem 是「使用道具」選單上的一項。
type UsableItem struct {
	// Slot 是道具欄索引 0–9。
	Slot int
	Item scenario.InventorySlot
}

// UsableItems 列出這名角色在戰鬥中選得到的道具。
func (c *Character) UsableItems() []UsableItem {
	var out []UsableItem
	for i := 0; i < InventorySlots; i++ {
		it := c.Inventory[i]
		if it.Empty() {
			continue
		}
		t := int(it.Type)
		switch {
		case t < armorFirstIndex && i != c.EquippedWeapon:
			continue // 武器類：只有已裝備的那一把能用
		case t >= armorFirstIndex && t < consumableFirstType && i != c.EquippedArmor:
			continue // 護甲類：同理
		}
		out = append(out, UsableItem{Slot: i, Item: it})
	}
	return out
}
