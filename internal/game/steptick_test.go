package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func TestStepHPTick_Regen(t *testing.T) {
	party := []Character{
		{Race: gamedata.Troll, CurrentHP: 10, MaxHP: 20},
		{Race: gamedata.Troll, CurrentHP: 20, MaxHP: 20}, // 滿血不再生
		{Race: gamedata.Human, CurrentHP: 10, MaxHP: 20}, // 非巨魔不再生
	}
	res := StepHPTick(party, StepHPRegen)
	if !res.Changed {
		t.Error("巨魔沒滿血應該回一點")
	}
	if party[0].CurrentHP != 11 {
		t.Errorf("巨魔 HP %d，預期 11", party[0].CurrentHP)
	}
	if party[1].CurrentHP != 20 {
		t.Errorf("滿血巨魔被改成 %d", party[1].CurrentHP)
	}
	if party[2].CurrentHP != 10 {
		t.Errorf("人類 HP 被改成 %d —— 再生是巨魔專屬", party[2].CurrentHP)
	}
	if len(res.Died) != 0 || res.AllDead {
		t.Error("再生模式不該死人")
	}
}

// TestStepHPTick_DrainHitsTrollToo 釘住「符印區連巨魔都受傷」。
//
// 原版的條件是 `0x7f < mode || 不是巨魔`，0x80 > 0x7f 直接成立，
// 種族判斷被 || 短路 —— 這是讀對 if-else 鏈順序才看得出來的。
func TestStepHPTick_DrainHitsTrollToo(t *testing.T) {
	party := []Character{
		{Race: gamedata.Troll, CurrentHP: 10, MaxHP: 20},
		{Race: gamedata.Human, CurrentHP: 10, MaxHP: 20},
	}
	StepHPTick(party, StepHPDrain)
	if party[0].CurrentHP != 9 {
		t.Errorf("巨魔在符印區 HP %d，預期 9 —— 不該免疫", party[0].CurrentHP)
	}
	if party[1].CurrentHP != 9 {
		t.Errorf("人類 HP %d，預期 9", party[1].CurrentHP)
	}
}

func TestStepHPTick_Death(t *testing.T) {
	party := []Character{
		{Race: gamedata.Human, CurrentHP: 1, MaxHP: 20},
		{Race: gamedata.Human, CurrentHP: 5, MaxHP: 20},
	}
	res := StepHPTick(party, StepHPDrain)
	if len(res.Died) != 1 || res.Died[0] != 0 {
		t.Errorf("死亡清單 %v，預期只有 0 號", res.Died)
	}
	if party[0].Status != scenario.StatusDead {
		t.Errorf("HP 歸零卻沒標成死亡，狀態 %v", party[0].Status)
	}
	if res.AllDead {
		t.Error("還有人活著卻回報全滅")
	}

	// 已死的人不再扣血，只計入人數
	res = StepHPTick(party, StepHPDrain)
	if party[0].CurrentHP != 0 {
		t.Errorf("死人 HP 被繼續扣到 %d", party[0].CurrentHP)
	}
	if party[1].CurrentHP != 3 {
		t.Errorf("活人 HP %d，預期 3", party[1].CurrentHP)
	}
}

func TestStepHPTick_AllDead(t *testing.T) {
	party := []Character{{Race: gamedata.Human, CurrentHP: 1, MaxHP: 20}}
	if res := StepHPTick(party, StepHPDrain); !res.AllDead {
		t.Error("唯一的隊員死了卻沒回報全滅")
	}
	// 空隊伍不算全滅（避免把「還沒建角」誤判成 game over）
	if res := StepHPTick(nil, StepHPDrain); res.AllDead {
		t.Error("空隊伍不該回報全滅")
	}
}

func TestGlyphDrainMode(t *testing.T) {
	none := [3]byte{}
	done := [3]byte{GlyphDone, GlyphDone, GlyphDone}

	// 符印還在的子地圖 → 流血
	for _, sm := range []int{55, 56, 66} {
		if GlyphDrainMode(none, sm) != StepHPDrain {
			t.Errorf("子地圖 %d 符印還在，應該流血", sm)
		}
		if GlyphDrainMode(done, sm) != StepHPRegen {
			t.Errorf("子地圖 %d 符印解了，不該再流血", sm)
		}
	}
	// 其他地方一律再生模式
	for _, sm := range []int{11, 44, 57, 65, 5} {
		if GlyphDrainMode(none, sm) != StepHPRegen {
			t.Errorf("子地圖 %d 不該流血", sm)
		}
	}
}
