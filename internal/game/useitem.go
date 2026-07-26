package game

// 紮營選單的 Use（`1000:2262`，見 `docs/re/39`）。
//
// 選單裡的道具分成兩類，走的是完全不同的兩條路：
//
//   - **光源**（火把 26、提燈 27）：用掉就消失，把隊伍的光源等級設成 2／3
//   - **有效果的道具**：拿 `+0x07` 的效果索引與 `+0x08` 的強度去施法，
//     道具留著、次數 +1
//
// 第二條路本專案還沒有 —— 施法目前只跑得起來在戰鬥裡（見 `CanUseInCamp`）。

// 兩種光源道具的型別。名稱來自 `ITEMS.DAT` 的型別名表：26 火把、27 提燈。
const (
	ItemTypeTorch   = 26
	ItemTypeLantern = 27
)

// 用掉之後的光源等級。**提燈比火把亮一級。**
//
// 原版 `1000:2379` 起：型別 0x1b 設 3、否則設 2，寫進 `party+0xa7`。
// 這正好補上 `docs/re/26` §5 記的缺口 —— 那時說「本專案還沒有點火把這些
// 會改動光源的機制」，機制就在這裡。
const (
	TorchLight   = 2
	LanternLight = 3
)

// LightSourceLevel 回傳這件光源道具點起來的等級，不是光源回 0。
func LightSourceLevel(itemType byte) int {
	switch itemType {
	case ItemTypeTorch:
		return TorchLight
	case ItemTypeLantern:
		return LanternLight
	default:
		return 0
	}
}

// useStatusLimit 是能用道具的狀態上限（原版 `<= 1`：正常與中毒可以）。
const useStatusLimit = 2

// UseResult 是一次「在營地用道具」的結果。
type UseResult struct {
	OK bool
	// Reason 是沒用成的原因。
	Reason string
	// Light 是用完之後的光源等級（只有光源道具會設）。
	Light int
	// Consumed 為 true 代表道具用掉消失了。
	Consumed bool
}

// CanUseInCamp 回報這一格在營地能不能用，不能的話給原因。
//
// 原版的過濾與戰鬥裡那條一樣（`強度 != 0` 且 `上限 != 已用`），
// 光源則是另外一條路、不看那兩個欄位。
func CanUseInCamp(c *Character, slot int) (bool, string) {
	if c == nil {
		return false, "沒有這個人"
	}
	if int(c.Status) >= useStatusLimit {
		return false, "現在沒辦法用東西"
	}
	if slot < 0 || slot >= InventorySlots {
		return false, "沒有這一格"
	}
	it := c.Inventory[slot]
	switch {
	case it.Empty():
		return false, "這一格是空的"
	case LightSourceLevel(it.Type) != 0:
		return true, ""
	case !it.Usable():
		return false, "這件現在用不了"
	}
	// 有效果的道具走施法那條路 —— 本專案的施法還離不開戰鬥。
	return false, "這件要在戰鬥中才用得出來"
}

// UseInCamp 在營地用掉一件道具。
//
// 目前只走得通光源那一條：**用掉即消失**（型別寫成 0xff，其餘 bytes 不動，
// 與 Drop 一樣），並回報新的光源等級由呼叫端寫進存檔。
func UseInCamp(c *Character, slot int) UseResult {
	ok, why := CanUseInCamp(c, slot)
	if !ok {
		return UseResult{Reason: why}
	}
	level := LightSourceLevel(c.Inventory[slot].Type)
	if level == 0 {
		// CanUseInCamp 已經擋掉其餘情形，走到這裡代表判定與行為脫節了。
		return UseResult{Reason: "這件要在戰鬥中才用得出來"}
	}
	c.Inventory[slot].Type = itemSlotEmpty
	c.unequipIfSlot(slot)
	return UseResult{OK: true, Light: level, Consumed: true}
}

// itemSlotEmpty 是空槽的型別值，與 scenario.SlotEmpty 相同。
const itemSlotEmpty = 0xff
