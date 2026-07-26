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
// > **這裡原本寫著「原版還有兩條過濾條件沒有實作」，那是誤判。**
// > 當時把反組譯裡的 `+0x11`／`+0x12` 當成「超出 17 bytes 槽長、
// > 應該是載入後的記憶體結構」。其實那兩個位移是**相對角色記錄**的
// > （`角色 + 槽×17` 起算，槽本身從 `+0x0c` 開始），減掉 `0x0c` 就是
// > `+0x05`／`+0x06` —— 也就是次數上限與已用次數。
// >
// > 紮營選單的 Use（`1000:2262`，見 `docs/re/39`）把三個欄位並排寫出來，
// > 一看就清楚：`+0x13` = 效果、`+0x14` = 強度、`+0x11`/`+0x12` = 兩個次數。
// > 所謂「沒實作的兩條」就是 `強度 != 0` 與 `上限 != 已用` ——
// > `InventorySlot.Usable()` 早就實作了。
// >
// > 教訓：位移是相對誰算的，要跟著它的基底走。少了那一步，
// > 一個已經解出來的欄位會被誤判成「還沒定位到的東西」。
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
		if !it.Usable() {
			continue // 強度 0 或次數用完（原版 17c5:1976／1981）
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
