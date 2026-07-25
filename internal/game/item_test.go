package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 武器與護甲只有「已裝備的那一格」才進得了 Use 選單。
//
// 語意是「Use 拿來觸發已裝備武器／護甲的特殊能力」，
// 不是拿背包裡沒裝備的武器當道具用。
func TestUsableItems_OnlyEquippedWeaponAndArmor(t *testing.T) {
	var c Character
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: 0xff}
	}
	c.Inventory[0] = scenario.InventorySlot{Type: 5}  // 闊劍（已裝備）
	c.Inventory[1] = scenario.InventorySlot{Type: 7}  // 戰斧（沒裝備）
	c.Inventory[2] = scenario.InventorySlot{Type: 10} // 鎖子甲（已裝備）
	c.Inventory[3] = scenario.InventorySlot{Type: 8}  // 布甲（沒裝備）
	c.Inventory[4] = scenario.InventorySlot{Type: 20} // 消耗品
	c.EquippedWeapon = 0
	c.EquippedArmor = 2

	got := map[int]bool{}
	for _, u := range c.UsableItems() {
		got[u.Slot] = true
	}

	for _, want := range []int{0, 2, 4} {
		if !got[want] {
			t.Errorf("第 %d 格應該可用", want)
		}
	}
	for _, notWant := range []int{1, 3} {
		if got[notWant] {
			t.Errorf("第 %d 格沒裝備，不該出現在選單", notWant)
		}
	}
}

// 消耗品不受裝備限制，永遠可選。
func TestUsableItems_ConsumablesAlwaysAvailable(t *testing.T) {
	var c Character
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: 0xff}
	}
	for i := 0; i < 3; i++ {
		c.Inventory[i] = scenario.InventorySlot{Type: byte(consumableFirstType + i)}
	}
	// 裝備欄指向空槽。
	c.EquippedWeapon, c.EquippedArmor = 9, 9

	if got := len(c.UsableItems()); got != 3 {
		t.Errorf("三件消耗品應全部可選，得到 %d", got)
	}
}

func TestUsableItems_SkipsEmpty(t *testing.T) {
	var c Character
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: 0xff}
	}
	if got := c.UsableItems(); len(got) != 0 {
		t.Errorf("道具欄全空時不該有可用道具，得到 %v", got)
	}
}

// 真實存檔的角色都有裝備，Use 選單至少列得出那一件。
func TestUsableItems_RealSave(t *testing.T) {
	save := loadParty(t)
	total := 0
	for i := range save.Characters {
		c := FromSave(save.Characters[i])
		total += len(c.UsableItems())
	}
	if total == 0 {
		t.Error("五名角色一件可用道具都沒有，過濾規則大概太嚴")
	}
}
