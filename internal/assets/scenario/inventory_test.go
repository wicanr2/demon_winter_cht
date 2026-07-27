package scenario

import "testing"

// 地城道具的槽是**另一種解讀**：`[0]` 型別 `0xfe`、`[1..16]` NUL 結尾的名字
// （`docs/re/95` §3.1，寫入端 `0x182ea`–`0x18382`，讀取端 `0x199b4`）。
func TestDungeonSlotParsesTheNameNotTheEffectFields(t *testing.T) {
	raw := make([]byte, inventorySlotLen)
	raw[0] = slotDungeon
	copy(raw[1:], "Iron key")

	got := parseInventorySlot(raw)
	if !got.Dungeon() {
		t.Fatalf("型別 %#x 沒被認成地城道具", got.Type)
	}
	if got.Empty() {
		t.Error("0xfe 被當成空格了 —— 那是 0xff")
	}
	if got.DungeonName != "Iron key" {
		t.Errorf("名字 = %q，預期 Iron key", got.DungeonName)
	}
	// 名字的位元組不可以被當成效果／強度／附魔讀。
	if got.Effect != 0 || got.Power != 0 || got.Enchant != 0 {
		t.Errorf("名字被解讀成效果欄了：effect=%d power=%d enchant=%d",
			got.Effect, got.Power, got.Enchant)
	}
}

// 地城道具**不能出現在戰鬥的使用道具選單**。
func TestDungeonSlotIsNeverUsable(t *testing.T) {
	s := NewDungeonSlot("Mallet")
	s.Power, s.Total = 5, 3 // 就算有人硬塞了效果欄
	if s.Usable() {
		t.Error("地城道具進了使用道具選單")
	}
}

// 逐位元組往返：解出來再寫回去，**一個 byte 都不能變**。
func TestDungeonSlotRoundTripsByteForByte(t *testing.T) {
	raw := make([]byte, inventorySlotLen)
	raw[0] = slotDungeon
	copy(raw[1:], "Bag/red dust")
	raw[15], raw[16] = 0x7f, 0x2a // NUL 之後的殘值，原版不清

	out := make([]byte, inventorySlotLen)
	copy(out, raw)
	parseInventorySlot(raw).encodeInto(out)

	for i := range raw {
		if raw[i] != out[i] {
			t.Fatalf("第 %d 個 byte %#x → %#x", i, raw[i], out[i])
		}
	}
}

// 名字放得下 16 bytes（最長的 `Serpent pillar` 只有 14）。
// 超過就截斷 —— 原版的槽就這麼大，寫過去會蓋到下一格。
func TestDungeonSlotNameIsCappedAtSixteen(t *testing.T) {
	long := "0123456789abcdefGHIJ"
	s := NewDungeonSlot(long)
	if len(s.DungeonName) != SlotDungeonNameMax {
		t.Fatalf("名字長度 = %d，預期截到 %d", len(s.DungeonName), SlotDungeonNameMax)
	}

	raw := make([]byte, inventorySlotLen)
	s.encodeInto(raw)
	if got := parseInventorySlot(raw); got.DungeonName != long[:SlotDungeonNameMax] {
		t.Errorf("寫回再讀出 = %q", got.DungeonName)
	}
}

// 空槽仍然只有 `0xff`。`> 0xfd` 那種寫法是原版營地丟棄的隨手複用，
// 不是「空格」的定義（`docs/re/33` §4 找空格用的是 `== 0xff`）。
func TestEmptyIsStillOnlyFF(t *testing.T) {
	if (InventorySlot{Type: slotDungeon}).Empty() {
		t.Error("0xfe 被算成空格 —— 拿到的地城道具會被別處當空位覆蓋掉")
	}
	if !(InventorySlot{Type: slotEmpty}).Empty() {
		t.Error("0xff 不是空格？")
	}
}
