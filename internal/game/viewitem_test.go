package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func psychic() []Character {
	c := Character{Name: "靈視者", CurrentHP: 10, MaxHP: 10}
	c.Skills[SkillViewItem] = true
	return []Character{c}
}

// viewItemRNG 造一個不會落在同一面的骰子來源。
//
// **不要用 NewWithSeed(1..n)** —— LCG 的 state 太小時 `Roll(n)` 恆為 1，
// 三條測試會一起假綠（`traps_test.go` 踩過同一個坑）。
func viewItemRNG(i, samples int) *rng.RNG {
	return rng.NewWithSeed(uint32(1 + i*(rng.Modulus/samples)))
}

// 沒有人會鑑物 → 什麼都不做，**而且不消耗額度**。
func TestViewItemNeedsTheSkill(t *testing.T) {
	uses := byte(0)
	res := BeginViewItem(viewItemRNG(1, 4), []Character{{Name: "路人", CurrentHP: 5}}, &uses)
	if !res.NoSkill {
		t.Error("沒人會鑑物卻跑了")
	}
	if uses != 0 {
		t.Errorf("沒人會卻消耗了額度：%d", uses)
	}
}

// 死掉的靈視者不提供技能（與觀室共用 partyHasSkill）。
func TestViewItemIgnoresTheDead(t *testing.T) {
	p := psychic()
	p[0].Status = scenario.StatusDead
	uses := byte(0)
	if !BeginViewItem(viewItemRNG(1, 4), p, &uses).NoSkill {
		t.Error("死掉的靈視者還在提供技能")
	}
}

// 一天三次，與觀室**各自獨立計數**。
func TestViewItemThreeTimesADay(t *testing.T) {
	uses := byte(PsychicUsesPerDay)
	if !BeginViewItem(viewItemRNG(1, 4), psychic(), &uses).Exhausted {
		t.Errorf("第 %d 次還用得到", PsychicUsesPerDay+1)
	}
}

// **失敗照樣扣額度** —— 原版的 `inc` 排在擲點之前（`0x1943e`）。
// 少了這一條，玩家可以一直重試到成功為止。
func TestViewItemSpendsAUseEvenOnFailure(t *testing.T) {
	const samples = 60
	failed, ready := 0, 0
	for i := 0; i < samples; i++ {
		uses := byte(0)
		res := BeginViewItem(viewItemRNG(i, samples), psychic(), &uses)
		if uses != 1 {
			t.Fatalf("第 %d 次沒扣額度（結果 %+v）", i, res)
		}
		switch {
		case res.Failed:
			failed++
		case res.Ready:
			ready++
		}
	}
	if failed == 0 || ready == 0 {
		t.Fatalf("%d 次裡失敗 %d、成功 %d —— 應該兩種都出現（1/3 失敗率）",
			samples, failed, ready)
	}
}

// `+4` 欄就是「要搭配哪一件」。
func TestViewItemHintReadsUseWith(t *testing.T) {
	items := gamedata.DungeonItems{
		{Name: "Cage", UseWith: "Iron key"},
		{Name: "Mallet"}, // `+4` 空著
	}
	if got, ok := ViewItemHint(items, 0); !ok || got != "Iron key" {
		t.Errorf("鑑物得到 %q／%v，預期 Iron key", got, ok)
	}
	if _, ok := ViewItemHint(items, 1); ok {
		t.Error("`+4` 空著卻回了搭配對象")
	}
	if _, ok := ViewItemHint(items, 99); ok {
		t.Error("越界索引卻回了搭配對象")
	}
}
