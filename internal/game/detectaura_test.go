package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func elf(st scenario.CombatStatus) Character {
	return Character{Name: "精靈", Race: gamedata.Elf, Status: st, CurrentHP: 10}
}

// 數的是**種族**不是技能旗標，而且死人也算（原版 count++ 排在狀態檢查之前）。
func TestElvesInParty(t *testing.T) {
	if n := ElvesInParty([]Character{human(), human()}); n != 0 {
		t.Errorf("沒有精靈卻數出 %d 個", n)
	}
	got := ElvesInParty([]Character{
		human(),
		elf(scenario.StatusNormal),
		elf(scenario.StatusDead),
	})
	if got != 2 {
		t.Errorf("數出 %d 個精靈，預期 2（死掉的也算）", got)
	}
}

// 四道或條件，而且**沒有查附帶法術 B 與特效值 B**。
func TestHasAura(t *testing.T) {
	for _, c := range []struct {
		slot  scenario.InventorySlot
		want  bool
		label string
	}{
		{scenario.InventorySlot{Type: 3}, false, "什麼都沒有"},
		{scenario.InventorySlot{Type: 3, SpellAPower: 4}, true, "附帶法術 A 的強度"},
		{scenario.InventorySlot{Type: 3, Power: 6}, true, "效果強度"},
		{scenario.InventorySlot{Type: 3, EffectAByte: 12}, true, "特效值 A"},
		{scenario.InventorySlot{Type: 3, Enchant: 1}, true, "附魔"},
		{scenario.InventorySlot{Type: 3, Enchant: -1}, true, "負附魔也算"},
		// 只長 B 那一組的偵測不到 —— 原版就沒查，照抄。
		{scenario.InventorySlot{Type: 3, SpellBPower: 4}, false, "只有附帶法術 B"},
		{scenario.InventorySlot{Type: 3, EffectBByte: 12}, false, "只有特效值 B"},
	} {
		if got := HasAura(c.slot); got != c.want {
			t.Errorf("%s：%v，預期 %v", c.label, got, c.want)
		}
	}
}

// 沒有精靈就完全不會亮，而且**連骰都不擲**。
func TestDetectAuraNeedsAnElf(t *testing.T) {
	magic := scenario.InventorySlot{Type: 3, Enchant: 2}
	noElf := []Character{human(), human()}
	r := rng.NewWithSeed(1)
	before := r.Next()
	_ = before
	for i := 0; i < 20; i++ {
		if DetectAura(r, noElf, magic) {
			t.Fatal("隊伍裡沒有精靈卻偵測到靈光")
		}
	}
	// 擲點次數：沒有精靈那條路一次都不該擲。
	r2 := rng.NewWithSeed(7)
	s0 := r2.Next()
	DetectAura(rng.NewWithSeed(7), noElf, magic)
	r3 := rng.NewWithSeed(7)
	DetectAura(r3, noElf, magic)
	if r3.Next() != s0 {
		t.Error("沒有精靈時不該動到亂數序列")
	}
}

// 機率是 `Roll(9) <= 精靈人數 + 4`：一個精靈 5/9、五個必中。
func TestDetectAuraOdds(t *testing.T) {
	magic := scenario.InventorySlot{Type: 3, Enchant: 2}
	count := func(elves, trials int) int {
		party := []Character{human()}
		for i := 0; i < elves; i++ {
			party = append(party, elf(scenario.StatusNormal))
		}
		r := rng.NewWithSeed(20260728)
		hits := 0
		for i := 0; i < trials; i++ {
			if DetectAura(r, party, magic) {
				hits++
			}
		}
		return hits
	}
	const trials = 900
	one := count(1, trials)
	// 5/9 ≈ 500；容忍 ±60 的抽樣誤差。
	if one < 440 || one > 560 {
		t.Errorf("一個精靈命中 %d/%d，預期約 500（5/9）", one, trials)
	}
	if got := count(5, trials); got != trials {
		t.Errorf("五個精靈命中 %d/%d，預期必中（9/9）", got, trials)
	}
	// 擲點過了但道具沒有靈光 → 還是不亮。
	plain := scenario.InventorySlot{Type: 3}
	party := []Character{elf(scenario.StatusNormal), elf(scenario.StatusNormal),
		elf(scenario.StatusNormal), elf(scenario.StatusNormal), elf(scenario.StatusNormal)}
	r := rng.NewWithSeed(3)
	for i := 0; i < 20; i++ {
		if DetectAura(r, party, plain) {
			t.Fatal("普通道具不該有靈光")
		}
	}
}
