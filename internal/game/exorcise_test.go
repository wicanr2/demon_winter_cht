package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func exorcist() *Character {
	c := campChar("薩滿")
	c.Skills[SkillShaman] = true
	return c
}

// cursedTarget 造一名戴著詛咒武器、兩項技能被封住的角色。
func cursedTarget(resist int) *Character {
	c := campChar("受害者", 3)
	c.Inventory[0].ExorciseResist = resist
	c.EquippedWeapon = 0
	c.CursedSkills[SkillShaman] = true
	c.CursedSkills[4] = true
	return c
}

func TestExorcise_SuccessFreesSkillsAndUnequips(t *testing.T) {
	caster, target := exorcist(), cursedTarget(100) // 必成

	res := Exorcise(rng.NewWithSeed(1), caster, target, 0)
	if !res.OK || !res.Success {
		t.Fatalf("成功率 100 卻沒成：%+v", res)
	}
	if res.Freed != 2 {
		t.Errorf("解開 %d 項技能，預期 2", res.Freed)
	}
	if len(target.CursedSkillList()) != 0 {
		t.Error("還有技能被封著")
	}
	if target.EquippedWeapon == 0 {
		t.Error("那一件應該脫下來")
	}
	if target.Inventory[0].Empty() {
		t.Error("驅邪不會弄丟道具，只是脫下來")
	}
}

// 失敗也用掉今天的機會。
func TestExorcise_FailureStillCostsTheDay(t *testing.T) {
	caster, target := exorcist(), cursedTarget(0) // 必敗

	res := Exorcise(rng.NewWithSeed(1), caster, target, 0)
	if !res.OK {
		t.Fatalf("應該動手了：%s", res.Reason)
	}
	if res.Success {
		t.Error("成功率 0 不該成功")
	}
	if !caster.ExorcisedToday {
		t.Error("失敗也要用掉今天的機會")
	}
	if len(target.CursedSkillList()) != 2 {
		t.Error("失敗不該解開技能")
	}
}

func TestExorcise_Refusals(t *testing.T) {
	stunned := exorcist()
	stunned.Status = 2

	done := exorcist()
	done.ExorcisedToday = true

	// 詛咒物在包包裡、沒穿在身上。
	inBag := campChar("受害者", 3)

	cases := []struct {
		name           string
		caster, target *Character
		slot           int
		reason         string
	}{
		{"沒有這個人", nil, cursedTarget(50), 0, "reason.member.invalid"},
		{"狀態太差", stunned, cursedTarget(50), 0, "reason.exorcise.unavailable"},
		{"不會驅邪", campChar("路人"), cursedTarget(50), 0, "reason.exorcise.no_skill"},
		{"今天驅過了", done, cursedTarget(50), 0, "reason.exorcise.used_today"},
		{"沒有這一格", exorcist(), cursedTarget(50), 99, "reason.slot.invalid"},
		{"空格", exorcist(), campChar("空手"), 0, "reason.slot.empty"},
		{"沒穿在身上", exorcist(), inBag, 0, "reason.exorcise.not_equipped"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, why := CanExorcise(tc.caster, tc.target, tc.slot)
			if ok {
				t.Fatal("預期擋下來")
			}
			if why != tc.reason {
				t.Errorf("理由 %q，預期 %q", why, tc.reason)
			}
		})
	}
}

// 司祭也驅得了 —— 兩種教團技能任一即可。
func TestExorcise_PriesthoodAlsoWorks(t *testing.T) {
	caster := campChar("司祭")
	caster.Skills[SkillPriesthood] = true
	if ok, why := CanExorcise(caster, cursedTarget(50), 0); !ok {
		t.Fatalf("司祭應該也會驅邪：%s", why)
	}
}

// 技能旗標值 2 存回去還是 2 —— 不能被存檔悄悄解掉。
func TestCharacter_CursedSkillSurvivesRoundTrip(t *testing.T) {
	var rec scenario.Character
	rec.SkillFlags[SkillShaman] = 2
	rec.SkillFlags[3] = 1

	c := FromSave(rec)
	if c.Skills[SkillShaman] {
		t.Error("被封住的技能不該算成會用")
	}
	if !c.CursedSkills[SkillShaman] {
		t.Error("旗標 2 應該記成被詛咒")
	}

	var back scenario.Character
	c.ApplyTo(&back)
	if back.SkillFlags[SkillShaman] != 2 {
		t.Errorf("寫回去是 %d，預期 2（詛咒不能憑空消失）", back.SkillFlags[SkillShaman])
	}
	if back.SkillFlags[3] != 1 {
		t.Errorf("一般技能寫回去是 %d，預期 1", back.SkillFlags[3])
	}
}
