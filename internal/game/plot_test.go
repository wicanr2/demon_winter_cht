package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

// TestAdvancePlotOnSleep_Order 釘住「每晚只推進一段，而且順序固定」。
//
// 原版每一段都以 return 結束（`docs/re/80` §1）。一晚跑兩段的話，
// 玩家會在同一次睡覺裡看到冬之魔降臨又看到諸神已死 —— 兩場夢疊在一起，
// 而且中間那一晚的鋪陳整個消失。
func TestAdvancePlotOnSleep_Order(t *testing.T) {
	// 月份 > 3 且還沒做過第一場夢 → 只走第一段。
	p := &PlotState{Month: 4, Stage: PlotArrivalDue}
	got, wipe := AdvancePlotOnSleep(p)
	if got != DreamWarning || wipe {
		t.Fatalf("第一晚 = (%d, %v)，預期 (DreamWarning, false)", got, wipe)
	}
	if p.FirstDream != 1 {
		t.Error("第一場夢應該把 FirstDream 立起來")
	}
	if p.Stage != PlotArrivalDue {
		t.Error("第一場夢不該同時推進階段 —— 原版那一段直接 return")
	}

	// 第二晚才輪到降臨。
	got, wipe = AdvancePlotOnSleep(p)
	if got != DreamArrival || wipe {
		t.Fatalf("第二晚 = (%d, %v)，預期 (DreamArrival, false)", got, wipe)
	}
	if p.Stage != PlotArrived {
		t.Errorf("階段 = %d，預期 %d", p.Stage, PlotArrived)
	}
	if p.TempleRuin != 0 {
		t.Error("降臨那一晚神殿還沒毀 —— 那是下一晚的事")
	}

	// 第三晚神殿全毀、信仰歸零。
	got, wipe = AdvancePlotOnSleep(p)
	if got != DreamGodsDead || !wipe {
		t.Fatalf("第三晚 = (%d, %v)，預期 (DreamGodsDead, true)", got, wipe)
	}
	if p.TempleRuin != TempleRuinsValue {
		t.Errorf("TempleRuin = 0x%02x，預期 0x%02x", p.TempleRuin, TempleRuinsValue)
	}

	// 之後就沒事了。
	if got, wipe = AdvancePlotOnSleep(p); got != NoDream || wipe {
		t.Errorf("第四晚 = (%d, %v)，預期 (NoDream, false)", got, wipe)
	}
}

// 月份不夠就不做第一場夢 —— 但階段條件仍然照走。
func TestAdvancePlotOnSleep_MonthGate(t *testing.T) {
	p := &PlotState{Month: firstDreamMonth, Stage: PlotArrivalDue}
	got, _ := AdvancePlotOnSleep(p)
	if got != DreamArrival {
		t.Errorf("月份 %d 不該觸發第一場夢，應該直接輪到降臨，得到 %d",
			firstDreamMonth, got)
	}
	if p.FirstDream != 0 {
		t.Error("第一場夢沒播就不該立旗標")
	}
}

// 起始狀態（階段 0）什麼都不會發生。
//
// ⚠ 這正是 `docs/re/81` 記的那個未解問題：把 `+0xb9` 從 0 推到 1 的寫入
// 還沒找到，所以照目前的實作，劇情鏈在引擎裡也推不動。
// **這條測試釘住現況，不是釘住「正確行為」** —— 找到寫入端之後要改。
func TestAdvancePlotOnSleep_Stage0DoesNothing(t *testing.T) {
	p := &PlotState{Month: 10, FirstDream: 1, Stage: PlotBeforeArrival}
	if got, _ := AdvancePlotOnSleep(p); got != NoDream {
		t.Errorf("階段 0 目前不該有動作，得到 %d", got)
	}
}

func TestWipeFaith(t *testing.T) {
	c := &Character{Deity: 5}
	c.Skills[gamedata.SkillShaman] = true
	c.Skills[gamedata.SkillPriesthood] = true
	c.Skills[gamedata.SkillHunting] = true

	WipeFaith(c)

	if c.Skills[gamedata.SkillShaman] || c.Skills[gamedata.SkillPriesthood] {
		t.Error("薩滿與司祭技能應該被清掉")
	}
	if c.Deity != 0 {
		t.Errorf("信奉的神祇 = %d，預期 0", c.Deity)
	}
	if !c.Skills[gamedata.SkillHunting] {
		t.Error("其他技能不該被牽連 —— 原版只清那兩個")
	}
}
