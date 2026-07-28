package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 防護點數表逐格釘住（`31f0:16c5` = `00 01 02 04 05 06 03`）。
//
// 重點是**鎖子甲以上不是等差**：布 1、皮 2 之後跳到 4／5／6。
func TestArmorPointsTable(t *testing.T) {
	for _, c := range []struct {
		typ   byte
		want  int
		label string
	}{
		{7, 0, "雙手劍（佔位）"},
		{8, 1, "布甲"},
		{9, 2, "皮甲"},
		{10, 4, "鎖子甲"},
		{11, 5, "鱗甲"},
		{12, 6, "板甲"},
		{13, 3, "王冠"},
		{14, 0, "表外"},
	} {
		got := ArmorPoints(scenario.InventorySlot{Type: c.typ})
		if got != c.want {
			t.Errorf("%s（型別 %d）= %d，預期 %d", c.label, c.typ, got, c.want)
		}
	}
	if got := ArmorPoints(scenario.InventorySlot{Type: slotEmptyType}); got != 0 {
		t.Errorf("空格 = %d，預期 0", got)
	}
}

// 附魔直接加在表值上（`17c5:0bd4` 的 `ADD AX,0xfff6` 扣掉 +10 偏移）。
func TestArmorPointsAddsEnchant(t *testing.T) {
	chain := scenario.InventorySlot{Type: 10}
	if got := ArmorPoints(chain); got != 4 {
		t.Fatalf("鎖子甲 = %d，預期 4", got)
	}
	chain.Enchant = 2
	if got := ArmorPoints(chain); got != 6 {
		t.Errorf("鎖子甲+2 = %d，預期 6（與板甲同級）", got)
	}
	chain.Enchant = -1
	if got := ArmorPoints(chain); got != 3 {
		t.Errorf("鎖子甲−1 = %d，預期 3", got)
	}
}

// 硬化皮膚加的 2 點 = 皮甲那一格，手冊說的「皮甲等級的防護」。
func TestBarkskinMatchesLeather(t *testing.T) {
	leather := ArmorPoints(scenario.InventorySlot{Type: 9})
	if BarkskinArmor != leather {
		t.Errorf("硬化皮膚 %d ≠ 皮甲 %d —— 手冊說兩者同級", BarkskinArmor, leather)
	}
}

// 職業上限表：三格與手冊對得上，其餘照表釘住。
func TestClassArmorMax(t *testing.T) {
	for _, c := range []struct {
		class gamedata.Class
		want  int
		label string
	}{
		{gamedata.Ranger, 14, "遊俠（實質不限）"},
		{gamedata.Paladin, 12, "聖騎士 板甲"},
		{gamedata.Barbarian, 12, "蠻族 板甲"},
		{gamedata.Monk, 9, "武僧 皮甲"},
		{gamedata.Cleric, 10, "司祭 鎖子甲"},
		{gamedata.Thief, 10, "盜賊 鎖子甲"},
		{gamedata.Wizard, 8, "巫師 布甲"},
		{gamedata.Sorcerer, 8, "術士 布甲"},
		{gamedata.Visionary, 10, "靈視者 鎖子甲"},
		{gamedata.Scholar, 9, "學者 皮甲"},
	} {
		if got := ClassArmorMax(c.class); got != c.want {
			t.Errorf("%s = %d，預期 %d", c.label, got, c.want)
		}
	}
	// 認不出的職業放行，不要把資料問題變成缺功能。
	if got := ClassArmorMax(gamedata.Class(99)); got != armorLastIndex {
		t.Errorf("職業 id 99 = %d，預期放行到 %d", got, armorLastIndex)
	}
}

// 換裝要同時過型別與職業兩道；原版的第二道印 `You're the wrong class.`。
func TestEquipRespectsClassArmorMax(t *testing.T) {
	plate := scenario.InventorySlot{Type: 12}
	cloth := scenario.InventorySlot{Type: 8}

	wizard := &Character{Name: "巫師", Class: gamedata.Wizard, EquippedArmor: -1}
	wizard.Inventory[0] = plate
	wizard.Inventory[1] = cloth
	if ok, _ := wizard.Equip(0); ok {
		t.Error("巫師穿上板甲了")
	}
	if wizard.EquippedArmor != -1 {
		t.Error("擋下來卻還是改了護甲槽")
	}
	if ok, why := wizard.Equip(1); !ok {
		t.Errorf("巫師穿不上布甲：%s", why)
	}

	// 靈視者的上限剛好卡在鎖子甲與鱗甲之間 —— 相鄰兩格配成一對測。
	seer := &Character{Name: "靈視者", Class: gamedata.Visionary, EquippedArmor: -1}
	seer.Inventory[0] = scenario.InventorySlot{Type: 10}
	seer.Inventory[1] = scenario.InventorySlot{Type: 11}
	if ok, why := seer.Equip(0); !ok {
		t.Errorf("靈視者穿不上鎖子甲：%s", why)
	}
	if ok, _ := seer.Equip(1); ok {
		t.Error("靈視者穿上鱗甲了")
	}
}
