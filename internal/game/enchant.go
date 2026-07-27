package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 矮人大師的附魔工坊（地點劇情 case 6，地圖 2 的 (28,5)，`docs/re/102`）
//
// `docs/re/55` 從攻略的 80 個點反推出「附魔費用 ＝ `35 × (20−c) × n^1.7`」，
// 但當時**找不到原版哪裡提供這個服務**，也不敢確定 `n^1.7` 是機制還是
// 一張表湊出來的（那一篇 §2(2) 明寫「非整數次方要 log／exp，
// 而我們目前在這個執行檔裡看到的只有整數次方」）。
//
// 兩件事都定案了：入口是地圖 2 上的一格劇情事件（不是城鎮設施），
// 而執行檔裡**真的有一支手寫的 pow**（`0x1b4dc`：log → 乘 → exp，
// 開頭一道 `n == 0 → 0.0` 的守衛）。
//
// # 費用怎麼算（`0x0feb1`–`0x0ff4b`）
//
//	新價 = ItemValue(把附魔改成新的值)
//	舊價 = ItemValue(現在的樣子)
//	費用 = (新價 − 舊價) × (20 − 材質類別) ÷ 10        ; 整數除法
//
// **`35 × (20−c) × n^1.7` 是這條式子的展開**：估價的附魔項是
// `350 × n^1.7`（武器），所以 `350 / 10 = 35`。`docs/re/55` 反推對了，
// 而且反推的是**可觀察行為**，這一輪補上的是機制。
//
// 這也解釋了為什麼費用要用「估價差」而不是一張表：**附魔會漲估價**，
// 而工坊收的就是那個差額打材質折扣 —— 越好的材質折扣越多
// （每高一級便宜 1/19，攻略說「大約每高一級折扣 5%」）。

const (
	// enchantMax 是附魔上限（原版 `0x0f9fa` 的 `cmp …,0xa`：
	// 超過 10 就印 `Enchantment beyond plus 10 / is not possible`）。
	enchantMax = 10

	// enchantDiscountBase／enchantDiscountDiv 是材質折扣：
	// `(20 − 材質類別) ÷ 10`（原版 `0x0ff17` 的 `mov ax,0x14` 與
	// `0x0ff40` 的 `mov cx,0xa` ＋ 32 位元除法）。
	enchantDiscountBase = 20
	enchantDiscountDiv  = 10
)

// EnchantMax 是玩家看得到的附魔上限。
const EnchantMax = enchantMax

// Enchantable 回報一件道具能不能附魔。
//
// 原版兩道閘門：
//
//	0x0f7e8  型別 > 0x1b        → "Only weapons and armor may be enchanted"
//	         （其實這一道是 `<= 0x1b` 才繼續，擋掉的是空槽 0xff 與地城道具）
//	0x0f7fd  `+0x10 == 0`       → "Only identified items may be enchanted."
//	0x0f954  型別 > 12          → "Only weapons and armor may be enchanted"
//
// 所以真正能附魔的是**武器（型別 0–7）與護甲（8–12）而且已鑑定**。
func Enchantable(slot scenario.InventorySlot) bool {
	if slot.Empty() || slot.Dungeon() {
		return false
	}
	if int(slot.Type) > valueArmourMax {
		return false
	}
	return slot.Identified
}

// EnchantCost 是把一件道具從現在的附魔值改成 plus 要花多少錢。
//
// basePrice 是 `ITEMS.DAT` 的底價（`ItemValue` 的第一個參數）。
// plus 不合法（<= 現值、或超過上限）時回 0 —— 原版分別印
// 「It is already +%d」與「Enchantment beyond plus 10 is not possible」。
func EnchantCost(basePrice int, slot scenario.InventorySlot, plus int) int {
	if plus > enchantMax || plus <= slot.Enchant || !Enchantable(slot) {
		return 0
	}
	after := slot
	after.Enchant = plus
	diff := ItemValue(basePrice, after) - ItemValue(basePrice, slot)
	if diff <= 0 {
		return 0
	}
	return diff * (enchantDiscountBase - slot.MaterialClass) / enchantDiscountDiv
}
