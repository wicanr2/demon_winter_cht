package scenario

// 道具槽的已解欄位（17 bytes，見 docs/formats/game-data-tables.md §1.3）。
//
//	+0x00  道具型別 = ITEMS.DAT 索引，0xFF 空槽
//	+0x09  條件旗標 A，值 0x15 時啟用 +0x0a
//	+0x0a  武器特效值 A（以 +10 偏移儲存）
//	+0x0b  條件旗標 B，值 0x15 時啟用 +0x0c
//	+0x0c  武器特效值 B（以 +10 偏移儲存）
//	+0x0e  附魔加成（以 +10 偏移儲存，10 = 無附魔）
//	+0x10  已鑑定旗標
const (
	slotType       = 0x00
	slotCondA      = 0x09
	slotEffectA    = 0x0a
	slotCondB      = 0x0b
	slotEffectB    = 0x0c
	slotEnchant    = 0x0e
	slotIdentified = 0x10

	// slotEmpty 是空槽的型別值。
	slotEmpty = 0xff

	// effectCondEnabled 是「下一個位元組有效」的條件值。
	effectCondEnabled = 0x15

	// storedOffset 是特效值與附魔共用的儲存偏移：存的是實際值 +10。
	storedOffset = 10
)

// InventorySlot 是一格裝備／道具。
type InventorySlot struct {
	// Type 是 ITEMS.DAT 的記錄索引。空槽是 0xFF。
	Type byte
	// Enchant 是附魔加成（已扣掉 +10 的儲存偏移）。
	Enchant int
	// WeaponEffect 是武器特效值（已扣掉 +10）。兩個條件旗標都沒啟用時為 0。
	WeaponEffect int
	// Identified 是已鑑定旗標。
	Identified bool
}

// Empty 回報這格是不是空的。
func (s InventorySlot) Empty() bool { return s.Type == slotEmpty }

// parseInventorySlot 解出一格的已解欄位。
//
// **特效值有兩組（A/B），各自有一個條件旗標。** 原版是
// `if (slot[+0x09] == 0x15) 讀 slot[+0x0a]`，B 組同理。
// 兩組都啟用時相加 —— 一件武器可以帶兩個特效。
func parseInventorySlot(raw []byte) InventorySlot {
	if len(raw) < inventorySlotLen {
		return InventorySlot{Type: slotEmpty}
	}
	out := InventorySlot{
		Type:       raw[slotType],
		Identified: raw[slotIdentified] != 0,
	}
	// 空槽的其餘欄位沒有意義，不要解讀成「附魔 −10」。
	if out.Empty() {
		return out
	}
	out.Enchant = int(raw[slotEnchant]) - storedOffset
	if raw[slotCondA] == effectCondEnabled {
		out.WeaponEffect += int(raw[slotEffectA]) - storedOffset
	}
	if raw[slotCondB] == effectCondEnabled {
		out.WeaponEffect += int(raw[slotEffectB]) - storedOffset
	}
	return out
}
