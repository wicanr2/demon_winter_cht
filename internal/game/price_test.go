package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func TestMaterialMultiplier_Table(t *testing.T) {
	want := []int{0, 1, 2, 5, 20, 35, 50, 60, 75}
	for i, w := range want {
		if got := MaterialMultiplier(i); got != w {
			t.Errorf("類別 %d 的倍率 %d，預期 %d", i, got, w)
		}
	}
	if MaterialMultiplier(-1) != 0 || MaterialMultiplier(MaterialClassCount) != 0 {
		t.Error("越界的類別應該回 0")
	}
}

// 起始存檔的裝備都是類別 1（×1）—— 估價就等於底價。
func TestItemValueBase_PlainGearIsBasePrice(t *testing.T) {
	slot := scenario.InventorySlot{Type: 3, MaterialClass: 1}
	if got := ItemValueBase(30, slot); got != 30 {
		t.Errorf("平凡裝備估價 %d，預期 30", got)
	}
}

func TestItemValueBase_ScalesWithClass(t *testing.T) {
	cases := map[int]int{0: 0, 1: 30, 3: 150, 8: 2250}
	for class, want := range cases {
		slot := scenario.InventorySlot{Type: 3, MaterialClass: class}
		if got := ItemValueBase(30, slot); got != want {
			t.Errorf("類別 %d 的估價 %d，預期 %d", class, got, want)
		}
	}
}

// 原版存檔裡每一件實物的材質類別都是 1 —— 這條同時釘住讀檔路徑。
func TestItemValueBase_RealSaveIsAllClassOne(t *testing.T) {
	save := loadParty(t)
	seen := 0
	for i := range save.Characters {
		for _, it := range save.Characters[i].Inventory {
			if it.Empty() {
				continue
			}
			seen++
			if it.MaterialClass != 1 {
				t.Errorf("%s 身上有材質類別 %d 的東西（型別 %d）",
					save.Characters[i].Name, it.MaterialClass, it.Type)
			}
		}
	}
	if seen == 0 {
		t.Fatal("原版存檔應該有裝備")
	}
}

// 已鑑定的加價 = (+0x02 + +0x04) × 270。225 × 1.2 剛好是整數。
func TestItemValue_IdentifiedBonus(t *testing.T) {
	slot := scenario.InventorySlot{
		Type: 3, MaterialClass: 1, Identified: true,
		Unknown02: 2, Unknown04: 3,
	}
	got, exact := ItemValue(30, slot)
	if !exact {
		t.Error("強度 0 的道具應該算得出確切售價")
	}
	if want := 30 + 5*270; got != want {
		t.Errorf("估價 %d，預期 %d", got, want)
	}
}

// 未鑑定就沒有第二項。
func TestItemValue_UnidentifiedSkipsBonus(t *testing.T) {
	slot := scenario.InventorySlot{Type: 3, MaterialClass: 1, Unknown02: 2, Unknown04: 3}
	if got, _ := ItemValue(30, slot); got != 30 {
		t.Errorf("未鑑定的估價 %d，預期 30", got)
	}
}

// 強度不為 0 就缺第三項 —— 一定要回報 exact=false。
func TestItemValue_PoweredItemIsNotExact(t *testing.T) {
	slot := scenario.InventorySlot{Type: 3, MaterialClass: 1, Power: 5}
	if _, exact := ItemValue(30, slot); exact {
		t.Error("有強度的道具還缺第三項，不該說算得準")
	}
}
