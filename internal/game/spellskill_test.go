package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

// 五系符文照 `16 + 系別`，與 AI 那一側的 aiSchoolSkill 同一條。
func TestSpellSkillForRuneSchools(t *testing.T) {
	for school := 1; school <= AISpellSchools; school++ {
		got, ok := SpellSkillFor(0, gamedata.Spell{School: school})
		if !ok {
			t.Fatalf("系別 %d 查不到技能", school)
		}
		if int(got) != aiSchoolSkill(school) {
			t.Errorf("系別 %d → 技能 %d，AI 那側是 %d —— 兩邊必須一致",
				school, got, aiSchoolSkill(school))
		}
	}
}

// 吟唱那一系三筆共用 school 6，只能靠法術索引分。
func TestSpellSkillForChants(t *testing.T) {
	chant := gamedata.Spell{School: chantSchool}
	for _, c := range []struct {
		index int
		want  gamedata.SkillID
		label string
	}{
		{24, gamedata.SkillSummoning, "召喚"},
		{25, gamedata.SkillIllusion, "幻術"},
		{26, gamedata.SkillPossession, "附身"},
	} {
		got, ok := SpellSkillFor(c.index, chant)
		if !ok || got != c.want {
			t.Errorf("%s（索引 %d）→ 技能 %d（ok=%v），預期 %d",
				c.label, c.index, got, ok, c.want)
		}
	}
	// 不在那三格的吟唱記錄查不到 —— 呼叫端要放行不是擋下。
	if _, ok := SpellSkillFor(99, chant); ok {
		t.Error("不認得的吟唱索引不該回 ok")
	}
}

// 學過才施得出來；認不出系別的放行。
func TestCanCast(t *testing.T) {
	c := &Character{Name: "巫師"}
	c.Skills[gamedata.SkillFireRunes] = true
	c.Skills[gamedata.SkillPossession] = true

	fire := gamedata.Spell{School: 1}
	ice := gamedata.Spell{School: 4}
	chant := gamedata.Spell{School: chantSchool}

	if !CanCast(c, 0, fire) {
		t.Error("學過火焰符文卻施不出火系")
	}
	if CanCast(c, 14, ice) {
		t.Error("沒學過寒冰符文卻施得出冰系")
	}
	if !CanCast(c, 26, chant) {
		t.Error("學過附身卻施不出附身")
	}
	if CanCast(c, 24, chant) {
		t.Error("沒學過召喚卻施得出召喚")
	}
	// 佔位記錄（系別 0）認不出來 → 放行，不要把資料問題變成缺功能。
	if !CanCast(c, 38, gamedata.Spell{}) {
		t.Error("認不出的系別應該放行")
	}
	// 被詛咒封住的技能在 Skills 裡就是 false，施不出來。
	cursed := &Character{Name: "被詛咒的"}
	if CanCast(cursed, 0, fire) {
		t.Error("沒學過就不該施得出來")
	}
}
