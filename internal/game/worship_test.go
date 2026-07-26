package game

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func worshipTables(t *testing.T) *gamedata.Tables {
	t.Helper()
	tb, err := gamedata.LoadTables(filepath.Join(repoRoot(t),
		"workplace", "orig", "demwin", "DEM_DATA", "FILES.DAT"))
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	return tb
}

func believer(deity, chance int) *Character {
	c := campChar("信徒")
	c.Skills[SkillPriesthood] = true
	c.Deity = deity
	c.PrayChance = chance
	c.MaxHP, c.CurrentHP = 40, 20
	c.MaxSP, c.CurrentSP = 10, 10
	return c
}

// 成功一次就把祈禱成功率永久扣 5，而且不吃施術者的法力。
func TestWorship_SuccessCostsChanceNotSP(t *testing.T) {
	tb := worshipTables(t)
	c := believer(1, 100) // 神祇 1 賜法術 0（EffectHP）

	res := Worship(rng.NewWithSeed(1), tb, c, c)
	if !res.OK || !res.Answered {
		t.Fatalf("成功率 100 應該必成：%+v", res)
	}
	if c.PrayChance != 95 {
		t.Errorf("祈禱成功率剩 %d，預期 95", c.PrayChance)
	}
	if c.CurrentSP != 10 {
		t.Errorf("施術者法力剩 %d，預期沒被扣（10）", c.CurrentSP)
	}
	if !c.WorshipedToday {
		t.Error("應該用掉今天的機會")
	}
}

// 失敗也用掉今天的機會，但不扣成功率。
func TestWorship_FailureCostsTheDayOnly(t *testing.T) {
	tb := worshipTables(t)
	c := believer(1, 0)

	res := Worship(rng.NewWithSeed(1), tb, c, c)
	if !res.OK {
		t.Fatalf("祈求本身應該成立：%s", res.Reason)
	}
	if res.Answered {
		t.Error("成功率 0 不該被回應")
	}
	if c.PrayChance != 0 {
		t.Errorf("失敗不該扣成功率，剩 %d", c.PrayChance)
	}
	if !c.WorshipedToday {
		t.Error("失敗也要用掉今天的機會")
	}
}

// 第 11 位神祇不賜法術 —— 擲贏了也沒有回應。
func TestWorship_LastDeityGrantsNothing(t *testing.T) {
	tb := worshipTables(t)
	c := believer(DeityAncient, 100)

	res := Worship(rng.NewWithSeed(1), tb, c, c)
	if res.Answered {
		t.Error("這位神祇不賜法術，不該有回應")
	}
	if c.PrayChance != 100 {
		t.Errorf("沒有回應就不該扣成功率，剩 %d", c.PrayChance)
	}
}

func TestWorship_Refusals(t *testing.T) {
	tb := worshipTables(t)

	stunned := believer(1, 50)
	stunned.Status = 2
	done := believer(1, 50)
	done.WorshipedToday = true
	noSkill := believer(1, 50)
	noSkill.Skills[SkillPriesthood] = false
	noFaith := believer(0, 50)

	cases := []struct {
		name   string
		c      *Character
		reason string
	}{
		{"沒有這個人", nil, "沒有這個人"},
		{"狀態太差", stunned, "現在沒辦法祈求"},
		{"不懂祈求", noSkill, "不懂得如何祈求"},
		{"今天求過了", done, "今天已經祈求過了"},
		{"沒有信仰", noFaith, "沒有信奉的神祇"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := Worship(rng.NewWithSeed(1), tb, tc.c, tc.c)
			if res.OK {
				t.Fatal("預期擋下來")
			}
			if res.Reason != tc.reason {
				t.Errorf("理由 %q，預期 %q", res.Reason, tc.reason)
			}
		})
	}
}

// 睡一晚清掉三個每日旗標。
func TestRest_ClearsAllDailyFlags(t *testing.T) {
	party := restParty(10, 10)
	party[0].IdentifiedToday = true
	party[0].WorshipedToday = true
	party[0].ExorcisedToday = true
	food := 5
	Rest(&fixedRolls{vals: []int{3}}, RestCamp, party, NewClock(), &food)
	if party[0].IdentifiedToday || party[0].WorshipedToday || party[0].ExorcisedToday {
		t.Error("睡一晚應該把三個每日旗標都清掉")
	}
}
