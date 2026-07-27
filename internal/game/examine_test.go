package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func examineParty(skills ...gamedata.SkillID) []Character {
	c := Character{Name: "學者", CurrentHP: 10, MaxHP: 10}
	for _, s := range skills {
		c.Skills[s] = true
	}
	return []Character{c}
}

func examineBattle() *Battle {
	return NewBattle(rng.NewWithSeed(1), []*Unit{
		{Slot: 0, Name: "狗頭人", Strength: 8, Skill: 9, Speed: 7, Armor: 3,
			WeaponIndex: 2, HP: 6, MaxHP: 6, AITargetSlot: PlayerSlotStart},
		nil,
		{Slot: PlayerSlotStart, Name: "甲", IsPlayer: true,
			Strength: 14, Skill: 12, Speed: 11, Armor: 5, WeaponIndex: 1,
			HP: 20, MaxHP: 20},
		{Slot: SummonSlotStart, Name: "棕熊", IsPlayer: true,
			Strength: 16, Skill: 10, Speed: 9, Armor: 4,
			HP: 18, MaxHP: 24, CurrentSP: 3, MaxSP: 8},
	})
}

// 沒有怪物學識就看不到怪物的屬性 —— 手冊：「除非隊伍中有人具備怪物學識，
// 否則不會顯示怪物的屬性」。**隊伍成員自己一律看得到。**
func TestExamineMonsterStatsNeedMonsterLore(t *testing.T) {
	b := examineBattle()

	blind := ExamineUnit(b, 0, examineParty())
	if blind.Stats {
		t.Error("沒有怪物學識卻看得到怪物屬性")
	}
	if blind.Name != "狗頭人" {
		t.Errorf("名稱 = %q —— 名字任何時候都看得到", blind.Name)
	}

	lore := ExamineUnit(b, 0, examineParty(SkillMonsterLore))
	if !lore.Stats || lore.Strength != 8 || lore.Armor != 3 {
		t.Errorf("有怪物學識卻讀不到屬性：%+v", lore)
	}

	// 隊伍成員不受這個閘門影響。
	self := ExamineUnit(b, PlayerSlotStart, examineParty())
	if !self.Stats || self.Strength != 14 {
		t.Errorf("看自己人也被怪物學識擋住了：%+v", self)
	}
}

// 戰術才顯示「牠打算攻擊誰」。
func TestExamineTargetNeedsTactics(t *testing.T) {
	b := examineBattle()

	// **開場沒有目標**：`NewBattle` 把每個單位的 AITargetSlot 設成
	// noAITarget，怪物要到第一回合才真的挑人。所以先確認這時候是空的，
	// 再手動指一個目標模擬「已經挑過了」。
	if got := ExamineUnit(b, 0, examineParty(SkillTactics)).TargetName; got != "" {
		t.Errorf("開場就顯示攻擊目標 %q —— 那時候還沒挑", got)
	}
	b.Unit(0).AITargetSlot = PlayerSlotStart

	if got := ExamineUnit(b, 0, examineParty()).TargetName; got != "" {
		t.Errorf("沒有戰術卻顯示攻擊目標 %q", got)
	}
	if got := ExamineUnit(b, 0, examineParty(SkillTactics)).TargetName; got != "甲" {
		t.Errorf("有戰術時攻擊目標 = %q，預期「甲」", got)
	}
}

// 只有召喚／幻術生物（槽位 >= 12）多一行 HP／SP。
func TestExamineHPSPOnlyForSummons(t *testing.T) {
	b := examineBattle()

	if ExamineUnit(b, PlayerSlotStart, examineParty()).ShowHPSP {
		t.Error("隊伍成員不該顯示 HP／SP 那一行（原版條件是槽位 > 11）")
	}
	if ExamineUnit(b, 0, examineParty(SkillMonsterLore)).ShowHPSP {
		t.Error("怪物不該顯示 HP／SP 那一行")
	}
	sum := ExamineUnit(b, SummonSlotStart, examineParty())
	if !sum.ShowHPSP || sum.HP != 18 || sum.SP != 3 || sum.MaxSP != 8 {
		t.Errorf("召喚生物的 HP／SP 不對：%+v", sum)
	}
}

// 束縛才在狀態後面帶等級（原版 `1 < 狀態 < 5`）。中毒與死亡都不帶。
func TestExamineBindLevelOnlyForBinds(t *testing.T) {
	cases := []struct {
		st   UnitStatus
		want bool
	}{
		{StatusNormal, false},
		{StatusPoison, false},
		{StatusBindLow, true},
		{3, true},
		{StatusBindHigh, true},
		{StatusDead, false},
	}
	for _, c := range cases {
		bb := NewBattle(rng.NewWithSeed(1), []*Unit{
			{Slot: PlayerSlotStart, Name: "甲", IsPlayer: true,
				Status: c.st, BindRounds: 3}})
		got := ExamineUnit(bb, PlayerSlotStart, examineParty()).ShowBindLevel
		if got != c.want {
			t.Errorf("狀態 %d：ShowBindLevel = %v，預期 %v", c.st, got, c.want)
		}
	}
}

// 死掉的隊員不提供技能。
func TestExaminePartySkillIgnoresTheDead(t *testing.T) {
	p := examineParty(SkillMonsterLore)
	p[0].Status = scenario.StatusDead
	if PartyHasSkill(p, SkillMonsterLore) {
		t.Error("死人還在提供怪物學識")
	}
}

// 走訪順序是槽位由小到大，跳過空槽與死亡單位。
func TestExamineOrderSkipsEmptyAndDead(t *testing.T) {
	b := examineBattle()
	// 補一具屍體到 1 號槽（那一格本來是空的）。
	b.units[1] = &Unit{Slot: 1, Name: "屍體", Status: StatusDead}

	got := ExamineOrder(b)
	want := []int{0, PlayerSlotStart, SummonSlotStart}
	if len(got) != len(want) {
		t.Fatalf("走訪順序 = %v，預期 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("走訪順序 = %v，預期 %v", got, want)
		}
	}
}

// 越界或空槽不 panic，回一張空卡。
func TestExamineOutOfRange(t *testing.T) {
	b := examineBattle()
	for _, i := range []int{-1, 1, 99} {
		if got := ExamineUnit(b, i, examineParty()); got.Name != "" {
			t.Errorf("索引 %d 回了一張非空的卡：%+v", i, got)
		}
	}
}
