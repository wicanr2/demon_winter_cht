package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func casterChar(sp int) *Character {
	c := campChar("法師")
	c.MaxSP, c.CurrentSP = 30, sp
	c.MaxHP, c.CurrentHP = 30, 30
	return c
}

func woundedChar(hp int) *Character {
	c := campChar("傷兵")
	c.MaxHP, c.CurrentHP = 30, hp
	c.MaxSP, c.CurrentSP = 10, 10
	return c
}

// 治療：K 為正的 EffectHP。
func healSpell() gamedata.Spell {
	return gamedata.Spell{Effect: EffectHP, K: 2, M: 1}
}

func TestCampCast_HealsAndCostsSP(t *testing.T) {
	caster, target := casterChar(20), woundedChar(5)

	res := CampCast(rng.NewWithSeed(1), caster, target, healSpell(), 6)
	if !res.OK {
		t.Fatalf("應該放得出來：%s", res.Reason)
	}
	if res.Delta <= 0 {
		t.Errorf("治療量 %d，預期為正", res.Delta)
	}
	if target.CurrentHP != 5+res.Delta {
		t.Errorf("目標 HP %d，預期 %d", target.CurrentHP, 5+res.Delta)
	}
	if caster.CurrentSP != 14 {
		t.Errorf("施法者剩 %d 法力，預期 14", caster.CurrentSP)
	}
}

// 治療不會超過上限。
func TestCampCast_HealClampsToMax(t *testing.T) {
	caster, target := casterChar(30), woundedChar(29)
	CampCast(rng.NewWithSeed(2), caster, target, healSpell(), 20)
	if target.CurrentHP > target.MaxHP {
		t.Errorf("HP %d 超過上限 %d", target.CurrentHP, target.MaxHP)
	}
}

// 四種「只在戰鬥中有意義」的增減要擋下來，而且不能扣法力。
func TestCampCast_RefusesCombatOnlyBuffs(t *testing.T) {
	for _, e := range []int{EffectSkillMod, EffectStrengthMod, EffectSpeedMod, EffectArmorMod} {
		caster, target := casterChar(20), woundedChar(10)
		res := CampCast(rng.NewWithSeed(1), caster, target,
			gamedata.Spell{Effect: e, K: 1, M: 1}, 5)
		if res.OK {
			t.Errorf("效果 %d 不該在營地放得出來", e)
		}
		if caster.CurrentSP != 20 {
			t.Errorf("效果 %d 被擋下卻扣了法力（剩 %d）", e, caster.CurrentSP)
		}
	}
}

func TestCampCast_RefusesBattleOnlyEffects(t *testing.T) {
	for _, e := range []int{EffectAOE, EffectInstantDeath, EffectBindApply} {
		caster, target := casterChar(20), woundedChar(10)
		if res := CampCast(rng.NewWithSeed(1), caster, target,
			gamedata.Spell{Effect: e, K: 1, M: 1}, 5); res.OK {
			t.Errorf("效果 %d 需要戰場，不該放得出來", e)
		}
	}
}

func TestCampCast_Refusals(t *testing.T) {
	stunned := casterChar(20)
	stunned.Status = 2

	cases := []struct {
		name   string
		caster *Character
		sp     int
		reason string
	}{
		{"沒有這個人", nil, 5, "沒有這個人"},
		{"狀態太差", stunned, 5, "現在沒辦法施法"},
		{"沒投法力", casterChar(20), 0, "要投入法力才放得出來"},
		{"法力不夠", casterChar(3), 5, "法力不夠"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := CampCast(rng.NewWithSeed(1), tc.caster, woundedChar(10), healSpell(), tc.sp)
			if res.OK {
				t.Fatal("預期擋下來")
			}
			if res.Reason != tc.reason {
				t.Errorf("理由 %q，預期 %q", res.Reason, tc.reason)
			}
		})
	}
}

// 解束縛：狀態要真的回到正常。
func TestCampCast_BindRelease(t *testing.T) {
	caster, target := casterChar(30), woundedChar(20)
	target.Status = 3 // 被 3 號符文系束縛

	s := gamedata.Spell{Effect: EffectBindRelease, School: 3, K: 1, M: 1}
	res := CampCast(rng.NewWithSeed(1), caster, target, s, 30)
	if !res.OK {
		t.Fatalf("應該放得出來：%s", res.Reason)
	}
	if res.Released && target.Status != 0 {
		t.Errorf("解開之後狀態是 %d，預期 0", target.Status)
	}
}

func TestCampCast_CuresPoison(t *testing.T) {
	caster := woundedChar(10)
	caster.CurrentSP = 30
	target := woundedChar(10)
	target.Status = scenario.StatusPoison
	s := gamedata.Spell{Effect: EffectPoison, K: 60, M: 9}

	res := CampCast(rng.NewWithSeed(1), caster, target, s, 15)
	if !res.OK || !res.Cured || target.Status != scenario.StatusNormal {
		t.Fatalf("解毒結果 = %+v，目標狀態 = %d", res, target.Status)
	}
	if caster.CurrentSP != 15 {
		t.Errorf("施法後法力 = %d，預期 15", caster.CurrentSP)
	}
}

func TestCampCast_ResurrectsWithOneHP(t *testing.T) {
	caster := woundedChar(10)
	caster.CurrentSP = 120
	target := woundedChar(10)
	target.CurrentHP = 0
	target.Status = scenario.StatusDead
	target.BindLevel = 4
	s := gamedata.Spell{Effect: EffectResurrect, K: 25, M: 25}

	res := CampCast(rng.NewWithSeed(1), caster, target, s, 100)
	if !res.OK || !res.Resurrected {
		t.Fatalf("復活結果 = %+v", res)
	}
	if target.CurrentHP != 1 || target.Status != scenario.StatusNormal || target.BindLevel != 0 {
		t.Errorf("復活後 = HP %d、狀態 %d、束縛 %d", target.CurrentHP, target.Status, target.BindLevel)
	}
}

func TestCampCast_LightAndWindWalk(t *testing.T) {
	caster := woundedChar(10)
	caster.CurrentSP = 30
	target := woundedChar(10)

	light := CampCast(rng.NewWithSeed(1), caster, target,
		gamedata.Spell{Effect: EffectLight, K: 2, M: 2}, 4)
	if !light.OK || light.Light != int(MaxLightLevel) {
		t.Errorf("光源結果 = %+v，預期鉗在 %d", light, MaxLightLevel)
	}

	wind := CampCast(rng.NewWithSeed(1), caster, target,
		gamedata.Spell{Effect: EffectWindWalk, K: 1, M: 10}, 10)
	if !wind.OK || !wind.WindWalk {
		t.Errorf("御風而行結果 = %+v", wind)
	}
}

func TestCampItemCast_UsesPowerWithoutSpendingCharacterSP(t *testing.T) {
	caster := casterChar(0)
	target := woundedChar(20)
	target.CurrentHP = 1
	spell := gamedata.Spell{Effect: EffectHP, K: 3, M: 1}

	res := CampItemCast(rng.NewWithSeed(1), caster, target, spell, 5)
	if !res.OK || res.Delta <= 0 {
		t.Fatalf("魔法物品應以 Power 施放：%+v", res)
	}
	if caster.CurrentSP != 0 {
		t.Fatalf("魔法物品不應扣角色 SP，得到 %d", caster.CurrentSP)
	}
}
