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
	// Power 非 0 代表這件有可用效果 —— 沒有效果的裝備原版根本不列（見下）。
	eq := func(t byte) scenario.InventorySlot {
		return scenario.InventorySlot{Type: t, Power: 5, Total: 3}
	}
	c.Inventory[0] = eq(5)  // 闊劍（已裝備）
	c.Inventory[1] = eq(7)  // 戰斧（沒裝備）
	c.Inventory[2] = eq(10) // 鎖子甲（已裝備）
	c.Inventory[3] = eq(8)  // 布甲（沒裝備）
	c.Inventory[4] = eq(20) // 消耗品
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
		c.Inventory[i] = scenario.InventorySlot{
			Type: byte(consumableFirstType + i), Power: 5, Total: 3}
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

// 沒有效果的道具選不到 —— 強度 0 或次數用完（原版 17c5:1976／1981）。
func TestUsableItems_NeedsEffect(t *testing.T) {
	var c Character
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: 0xff}
	}
	c.EquippedWeapon, c.EquippedArmor = 9, 9

	c.Inventory[0] = scenario.InventorySlot{Type: 20, Power: 0, Total: 3}
	c.Inventory[1] = scenario.InventorySlot{Type: 21, Power: 5, Used: 3, Total: 3}
	c.Inventory[2] = scenario.InventorySlot{Type: 22, Power: 5, Used: 1, Total: 3}

	got := c.UsableItems()
	if len(got) != 1 || got[0].Slot != 2 {
		t.Errorf("只有第 2 格該可用，得到 %v", got)
	}
}

// 原版起始存檔的裝備**一件都選不到** —— 全是 +0x07/+0x08 為 0 的平凡武具。
//
// ⚠ 這條原本寫的是「至少列得出一件，否則過濾規則太嚴」。**那個預期是錯的**，
// 它釘住的是還沒解出 +0x08 時的不完整規則。實際 dump 過起始存檔：五名角色
// 全部 10 格的 `+0x05`–`+0x08` 都是 0，原版的 Use 選單在開局本來就是空的
// —— 要拿到有效果的道具才選得到東西。
func TestUsableItems_RealSaveHasNoUsableStartingGear(t *testing.T) {
	save := loadParty(t)
	for i := range save.Characters {
		c := FromSave(save.Characters[i])
		if got := c.UsableItems(); len(got) != 0 {
			t.Errorf("%s 的起始裝備不該有可用道具，得到 %v", c.Name, got)
		}
	}
}
