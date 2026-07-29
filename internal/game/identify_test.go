package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// loreChar 造一名帶指定學識技能、身上有一件未鑑定道具的角色。
func loreChar(itemType byte, skills ...gamedata.SkillID) *Character {
	c := campChar("研究者", itemType)
	c.Traits[gamedata.Intellect] = 10
	for _, s := range skills {
		c.Skills[s] = true
	}
	return c
}

// 0–25 每一型都要落在某一種學識底下，而且分界要與原版一致。
func TestLoreSkillFor_PartitionsEveryType(t *testing.T) {
	want := map[byte]gamedata.SkillID{}
	for v := byte(0); v <= 12; v++ {
		want[v] = SkillWeaponLore
	}
	want[13] = SkillItemLore
	want[14] = SkillPotionLore
	for v := byte(15); v <= 23; v++ {
		want[v] = SkillItemLore
	}
	want[24] = SkillPotionLore
	want[25] = SkillPotionLore

	for v, w := range want {
		if got := LoreSkillFor(v); got != w {
			t.Errorf("型別 %d 需要技能 %d，預期 %d", v, got, w)
		}
	}
}

func TestIdentifyChance_IsIntellectTimesNineHalves(t *testing.T) {
	cases := map[int]int{0: 0, 10: 45, 11: 49, 22: 99, 30: 135}
	for intel, want := range cases {
		if got := IdentifyChance(intel); got != want {
			t.Errorf("智力 %d 的成功率 %d%%，預期 %d%%", intel, got, want)
		}
	}
}

func TestIdentify_SuccessMarksTheSlot(t *testing.T) {
	c := loreChar(3, SkillWeaponLore)
	c.Traits[gamedata.Intellect] = 30 // 成功率 135%，必中

	res := Identify(rng.NewWithSeed(1), c, 0)
	if !res.OK || !res.Success {
		t.Fatalf("必中的情況卻沒成功：%+v", res)
	}
	if !c.Inventory[0].Identified {
		t.Error("成功之後那一格應該標成已鑑定")
	}
}

// 失敗也用掉一天 —— 這是原版的順序（先設旗標再擲骰）。
func TestIdentify_FailureStillCostsTheDay(t *testing.T) {
	c := loreChar(3, SkillWeaponLore)
	c.Traits[gamedata.Intellect] = 0 // 成功率 0%，必敗

	res := Identify(rng.NewWithSeed(1), c, 0)
	if !res.OK {
		t.Fatalf("應該動手了才對：%s", res.Reason)
	}
	if res.Success || c.Inventory[0].Identified {
		t.Error("成功率 0% 不該鑑定成功")
	}
	if !c.IdentifiedToday {
		t.Error("失敗也要用掉今天的機會")
	}
	if _, why := CanIdentify(c, 0); why != "reason.identify.used_today" {
		t.Errorf("第二次的理由是 %q", why)
	}
}

func TestIdentify_Refusals(t *testing.T) {
	poisoned := loreChar(3, SkillWeaponLore)
	poisoned.Status = 3

	done := loreChar(3, SkillWeaponLore)
	done.IdentifiedToday = true

	known := loreChar(3, SkillWeaponLore)
	known.Inventory[0].Identified = true

	cases := []struct {
		name   string
		c      *Character
		slot   int
		reason string
	}{
		{"狀態太差", poisoned, 0, "reason.identify.unavailable"},
		{"今天研究過了", done, 0, "reason.identify.used_today"},
		{"空格", loreChar(3, SkillWeaponLore), 5, "reason.slot.empty"},
		{"已鑑定", known, 0, "reason.identify.already"},
		{"沒有那種學識", loreChar(3), 0, "reason.identify.unsupported"},
		{"藥劑要藥劑學識", loreChar(14, SkillWeaponLore, SkillItemLore), 0, "reason.identify.unsupported"},
		{"沒有這一格", loreChar(3, SkillWeaponLore), 99, "reason.slot.invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, why := CanIdentify(tc.c, tc.slot)
			if ok {
				t.Fatal("預期擋下來，卻放行了")
			}
			if why != tc.reason {
				t.Errorf("理由 %q，預期 %q", why, tc.reason)
			}
			if res := Identify(rng.NewWithSeed(1), tc.c, tc.slot); res.OK {
				t.Error("被擋下來還是動手了")
			}
		})
	}
}

// 睡一晚就能再研究一次。
func TestRest_ClearsIdentifiedToday(t *testing.T) {
	party := restParty(10, 10)
	party[0].IdentifiedToday = true
	food := 5
	Rest(&fixedRolls{vals: []int{3}}, RestCamp, party, NewClock(), &food)
	if party[0].IdentifiedToday {
		t.Error("睡一晚應該清掉「本日已研究」")
	}
}

func TestIdentifiableSlots_SkipsWhatHeCannotRead(t *testing.T) {
	c := campChar("A", 3, 14, 20)
	c.Skills[SkillWeaponLore] = true // 只懂武器
	c.Inventory[2].Identified = true

	got := c.IdentifiableSlots()
	if len(got) != 1 || got[0] != 0 {
		t.Errorf("可鑑定的格子 %v，預期只有第 0 格", got)
	}
}
