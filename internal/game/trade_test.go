package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func tradeParty() []Character {
	p := make([]Character, 2)
	for i := range p {
		for j := range p[i].Inventory {
			p[i].Inventory[j] = scenario.InventorySlot{Type: slotEmptyType}
		}
		p[i].EquippedWeapon, p[i].EquippedArmor = -1, -1
	}
	p[0].Name, p[1].Name = "甲", "乙"
	return p
}

// 空格常數要與 scenario 那邊一致 —— 兩邊各定一個，靠這條綁在一起。
func TestSlotEmptyTypeMatchesScenario(t *testing.T) {
	var s scenario.InventorySlot
	s.Type = slotEmptyType
	if !s.Empty() {
		t.Errorf("slotEmptyType = 0x%02x，scenario 不認為那是空格", slotEmptyType)
	}
}

// 買東西：扣錢、放進第一個有空格的隊員。
func TestBuy(t *testing.T) {
	p := tradeParty()
	res := Buy(p, 100, 30, 5)
	if !res.OK || res.Gold != 70 || res.Member != 0 || res.Slot != 0 {
		t.Fatalf("買一件 30 金：%+v", res)
	}
	if p[0].Inventory[0].Type != 5 {
		t.Errorf("第 0 格是 %d，預期 5", p[0].Inventory[0].Type)
	}
	// 店裡的貨沒有效果 —— 有效果的道具是掉寶生成的。
	got := p[0].Inventory[0]
	if got.Effect != 0 || got.Power != 0 || got.Enchant != 0 {
		t.Errorf("買到的道具帶了效果／附魔：%+v", got)
	}
	if !got.Identified {
		t.Error("店裡買的應該是已鑑定的")
	}
}

func TestBuy_NotEnoughGold(t *testing.T) {
	p := tradeParty()
	res := Buy(p, 10, 30, 5)
	if res.OK || res.Gold != 10 {
		t.Errorf("錢不夠卻買成了：%+v", res)
	}
	if !p[0].Inventory[0].Empty() {
		t.Error("沒買成卻放了東西進去")
	}
}

// 第一個人滿了就換下一個。
func TestBuy_FallsThroughToNextMember(t *testing.T) {
	p := tradeParty()
	for j := range p[0].Inventory {
		p[0].Inventory[j] = scenario.InventorySlot{Type: 1}
	}
	res := Buy(p, 100, 10, 7)
	if !res.OK || res.Member != 1 {
		t.Errorf("第一個人滿了應該放到第二個，得到 %+v", res)
	}
}

func TestBuy_EveryoneFull(t *testing.T) {
	p := tradeParty()
	for i := range p {
		for j := range p[i].Inventory {
			p[i].Inventory[j] = scenario.InventorySlot{Type: 1}
		}
	}
	if res := Buy(p, 100, 10, 7); res.OK {
		t.Error("全隊道具欄都滿了還買得成")
	}
}

// 賣東西：加錢、清空那一格。
func TestSell(t *testing.T) {
	p := tradeParty()
	p[0].Inventory[3] = scenario.InventorySlot{Type: 6}

	res := Sell(p, 50, 0, 3, 15)
	if !res.OK || res.Gold != 65 {
		t.Fatalf("賣一件 15 金：%+v", res)
	}
	if !p[0].Inventory[3].Empty() {
		t.Error("賣掉之後那一格應該是空的")
	}
}

// **身上正在用的不能賣** —— 賣掉會讓裝備索引指向空格。
func TestSell_RefusesEquipped(t *testing.T) {
	p := tradeParty()
	p[0].Inventory[2] = scenario.InventorySlot{Type: 6}
	p[0].Inventory[4] = scenario.InventorySlot{Type: 10}
	p[0].EquippedWeapon = 2
	p[0].EquippedArmor = 4

	for _, slot := range []int{2, 4} {
		if res := Sell(p, 50, 0, slot, 15); res.OK {
			t.Errorf("第 %d 格是裝備中的，不該賣得掉", slot)
		}
	}
	if p[0].Inventory[2].Empty() || p[0].Inventory[4].Empty() {
		t.Error("拒絕之後道具不該消失")
	}
}

func TestSell_BadArgs(t *testing.T) {
	p := tradeParty()
	for _, c := range []struct{ member, slot int }{
		{-1, 0}, {5, 0}, {0, -1}, {0, 99}, {0, 0}, // 最後一個是空格
	} {
		if res := Sell(p, 50, c.member, c.slot, 15); res.OK {
			t.Errorf("隊員 %d 第 %d 格不該賣得掉", c.member, c.slot)
		}
	}
}

// 裝備：武器進武器位、護甲進護甲位、消耗品兩邊都不行。
func TestEquip(t *testing.T) {
	p := tradeParty()
	c := &p[0]
	c.Inventory[0] = scenario.InventorySlot{Type: 5}  // 闊劍
	c.Inventory[1] = scenario.InventorySlot{Type: 10} // 鎖子甲
	c.Inventory[2] = scenario.InventorySlot{Type: 20} // 消耗品

	if ok, why := c.Equip(0); !ok {
		t.Errorf("武器該裝得上：%s", why)
	}
	if c.EquippedWeapon != 0 {
		t.Errorf("武器索引 %d，預期 0", c.EquippedWeapon)
	}
	if ok, why := c.Equip(1); !ok {
		t.Errorf("護甲該裝得上：%s", why)
	}
	if c.EquippedArmor != 1 {
		t.Errorf("護甲索引 %d，預期 1", c.EquippedArmor)
	}
	if ok, _ := c.Equip(2); ok {
		t.Error("消耗品不該裝得上")
	}
	if ok, _ := c.Equip(3); ok {
		t.Error("空格不該裝得上")
	}
	if ok, _ := c.Equip(99); ok {
		t.Error("超出範圍的格不該裝得上")
	}
	// 裝備只改索引，道具本身不動。
	if c.Inventory[0].Type != 5 || c.Inventory[1].Type != 10 {
		t.Error("換裝不該動到道具本身")
	}
}

// 換武器就是把索引指到另一格。
func TestEquip_Swaps(t *testing.T) {
	p := tradeParty()
	c := &p[0]
	c.Inventory[0] = scenario.InventorySlot{Type: 0} // 匕首
	c.Inventory[1] = scenario.InventorySlot{Type: 7} // 雙手劍
	c.Equip(0)
	c.Equip(1)
	if c.EquippedWeapon != 1 {
		t.Errorf("換武器後索引 %d，預期 1", c.EquippedWeapon)
	}
}

func TestEquip_RecomputesPassiveEffects(t *testing.T) {
	c := Character{EquippedWeapon: -1, EquippedArmor: -1, MaxSP: 20}
	c.Traits[gamedata.Speed] = 8
	c.Traits[gamedata.Strength] = 9
	c.Traits[gamedata.Skill] = 10
	c.TraitsWithBonus.MaxSP = 20
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: scenario.SlotEmpty}
	}
	c.Inventory[0] = scenario.InventorySlot{
		Type:             3,
		EffectTypeA:      scenario.EquipmentEffectSpeed,
		EffectValueAByte: 12, // +2
		EffectTypeB:      scenario.EquipmentEffectStrength,
		EffectValueBByte: 14, // +4
	}
	c.Inventory[1] = scenario.InventorySlot{
		Type:             10,
		EffectTypeA:      scenario.EquipmentEffectSkill,
		EffectValueAByte: 13, // +3
		EffectTypeB:      scenario.EquipmentEffectMaxSP,
		EffectValueBByte: 15, // +5
	}
	c.Inventory[2] = scenario.InventorySlot{
		Type:             5,
		EffectTypeA:      scenario.EquipmentEffectSpeed,
		EffectValueAByte: 7, // −3
	}

	if ok, why := c.Equip(0); !ok {
		t.Fatalf("裝武器失敗：%s", why)
	}
	if ok, why := c.Equip(1); !ok {
		t.Fatalf("穿護甲失敗：%s", why)
	}
	if got := int(c.TraitsWithBonus.Speed); got != 10 {
		t.Errorf("速度 = %d，預期 8+2=10", got)
	}
	if got := int(c.TraitsWithBonus.Strength); got != 13 {
		t.Errorf("力量 = %d，預期 9+4=13", got)
	}
	if got := int(c.TraitsWithBonus.Skill); got != 13 {
		t.Errorf("技巧 = %d，預期 10+3=13", got)
	}
	if c.MaxSP != 25 {
		t.Errorf("MaxSP = %d，預期 20+5=25", c.MaxSP)
	}

	// 換武器必須移除舊武器效果，再由天生值重算；護甲效果仍保留。
	if ok, why := c.Equip(2); !ok {
		t.Fatalf("換武器失敗：%s", why)
	}
	if got := int(c.TraitsWithBonus.Speed); got != 5 {
		t.Errorf("換裝後速度 = %d，預期 8−3=5", got)
	}
	if got := int(c.TraitsWithBonus.Strength); got != 9 {
		t.Errorf("舊武器力量效果未移除：%d", got)
	}
	if int(c.TraitsWithBonus.Skill) != 13 || c.MaxSP != 25 {
		t.Errorf("護甲效果不應消失：技巧 %d MaxSP %d",
			c.TraitsWithBonus.Skill, c.MaxSP)
	}

	u := c.CombatUnit(PlayerSlotStart, 0, 0, North)
	if u.Speed != 5 || u.Strength != 9 || u.Skill != 13 || u.MaxSP != 25 {
		t.Errorf("戰鬥沒有使用有效值：%+v", u)
	}

	var rec scenario.Character
	c.ApplyTo(&rec)
	if rec.SpeedBonus != 5 || rec.StrengthBonus != 9 ||
		rec.SkillBonus != 13 || rec.MaxSPBonus != 25 || rec.MaxSPNatural != 20 {
		t.Errorf("存檔常駐效果欄位錯誤：%+v", rec)
	}
}

// 買到的東西要一路寫得回存檔記錄。
//
// 規則層與存檔層各有一份道具表示，中間靠 ApplyTo 接起來。**這一段斷掉的話
// 買賣與換裝在畫面上全部正常，只有存檔時悄悄退回舊值** —— 玩家會發現
// 錢花了、東西沒了。scenario 那一層另有一條測試釘住「解析後的道具要寫進
// 原始 bytes」，兩條合起來才涵蓋完整的路徑。
func TestApplyTo_WritesInventoryAndEquipment(t *testing.T) {
	var rec scenario.Character
	for i := range rec.Inventory {
		rec.Inventory[i] = scenario.InventorySlot{Type: slotEmptyType}
	}
	rec.WeaponSlotIndex, rec.ArmorSlotIndex = 0xff, 0xff

	c := FromSave(rec)
	res := Buy([]Character{c}, 100, 30, 5)
	if !res.OK {
		t.Fatalf("買不成：%+v", res)
	}
	c.Inventory[0] = scenario.InventorySlot{Type: 5, Identified: true}
	c.Inventory[1] = scenario.InventorySlot{Type: 10, Identified: true}
	if ok, why := c.Equip(1); !ok {
		t.Fatalf("護甲裝不上：%s", why)
	}
	c.Status = scenario.StatusPoison

	c.ApplyTo(&rec)

	if rec.Inventory[0].Type != 5 || rec.Inventory[1].Type != 10 {
		t.Errorf("道具沒寫回：%+v", rec.Inventory[:2])
	}
	if rec.ArmorSlotIndex != 1 {
		t.Errorf("護甲槽索引 %d，預期 1", rec.ArmorSlotIndex)
	}
	// 還沒配武器 —— 存檔用 0xFF 表示「沒有」，不是 0（0 是第一格）。
	if rec.WeaponSlotIndex != 0xff {
		t.Errorf("沒配武器時武器槽索引 %d，預期 255", rec.WeaponSlotIndex)
	}
	if rec.CombatStatus != scenario.StatusPoison {
		t.Errorf("戰鬥狀態 %d，預期中毒", rec.CombatStatus)
	}
}
