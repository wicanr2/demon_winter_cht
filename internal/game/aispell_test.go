package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// fixedRolls 依序回傳指定的擲點結果，用完就重複最後一個。
// 拿來把「挑法術」的邏輯與亂數本身分開驗。
type fixedRolls struct {
	vals []int
	i    int
}

func (f *fixedRolls) Roll(n int) int {
	v := f.vals[len(f.vals)-1]
	if f.i < len(f.vals) {
		v = f.vals[f.i]
	}
	f.i++
	if v > n {
		v = n
	}
	if v < 1 {
		v = 1
	}
	return v
}

// 五個符文系的法術 id 區間必須連續、不重疊，而且剛好接到召喚／幻術。
//
// 這條驗的是區間表本身：T = {0,0,4,10,14,19,24,27}。表抄錯一格，
// AI 就會抽到別系的法術，而且完全不會報錯。
func TestAISpellRange_ContiguousAndMeetsSummon(t *testing.T) {
	prev := aiSpellRange[1]
	if prev != 0 {
		t.Errorf("第 1 系從 %d 起算，預期 0", prev)
	}
	for s := 1; s <= AISpellSchools; s++ {
		lo, hi := aiSpellRange[s], aiSpellRange[s+1]
		if lo != prev {
			t.Errorf("第 %d 系從 %d 起算，與前一系的結尾 %d 不連續", s, lo, prev)
		}
		if hi <= lo {
			t.Errorf("第 %d 系的區間 [%d,%d) 是空的", s, lo, hi)
		}
		prev = hi
	}
	// 五系涵蓋 0–23，下一段從 24 起 —— 召喚 0x18=24、幻術 0x19=25 正好在那裡。
	if prev != 24 {
		t.Errorf("五系結束在 %d，預期 24（召喚 0x18 的位置）", prev)
	}
	if aiSpellRange[7] != 27 {
		t.Errorf("召喚／幻術那一段結束在 %d，預期 27", aiSpellRange[7])
	}
}

// 符文系 1–5 對應技能 id 17–21（火焰／金屬／風／寒冰／靈魂）。
func TestAISchoolSkill(t *testing.T) {
	for school, want := range map[int]int{1: 17, 2: 18, 3: 19, 4: 20, 5: 21} {
		if got := aiSchoolSkill(school); got != want {
			t.Errorf("第 %d 系 → 技能 %d，預期 %d", school, got, want)
		}
	}
}

// 挑出來的法術一定落在被抽中那一系的區間裡。
func TestAISpellChoice_StaysInSchool(t *testing.T) {
	tb := loadTables(t)
	for school := 1; school <= AISpellSchools; school++ {
		lo, hi := aiSpellRange[school], aiSpellRange[school+1]
		for pick := 1; pick <= hi-lo; pick++ {
			r := &fixedRolls{vals: []int{school, pick}}
			id, ok := AISpellChoice(r, tb, 999, nil)
			if !ok {
				continue // 該格是空記錄，跳過
			}
			if id < lo || id >= hi {
				t.Errorf("第 %d 系第 %d 個挑出 id %d，超出區間 [%d,%d)",
					school, pick, id, lo, hi)
			}
		}
	}
}

// 法力不夠就不能挑到那個法術 —— 原版是回頭重挑（0x7bed）。
func TestAISpellChoice_RespectsSPCost(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(9)

	for i := 0; i < 500; i++ {
		id, ok := AISpellChoice(r, tb, 3, nil)
		if !ok {
			continue
		}
		sp, err := tb.Spell(id)
		if err != nil {
			t.Fatal(err)
		}
		if sp.M > 3 {
			t.Fatalf("法力只有 3，卻挑到需要 %d 點的法術 %d", sp.M, id)
		}
	}
}

// 完全沒有法力時挑不到任何法術，而且不能卡在重挑迴圈裡。
//
// 原版的重挑是無界迴圈；這裡有重試上限，回 ok=false 讓呼叫端改做別的事。
func TestAISpellChoice_GivesUpWhenBroke(t *testing.T) {
	tb := loadTables(t)
	if _, ok := AISpellChoice(rng.NewWithSeed(3), tb, 0, nil); ok {
		t.Error("法力 0 卻挑到法術")
	}
}

// 不會該系就不挑那一系。
func TestAISpellChoice_ChecksSkill(t *testing.T) {
	tb := loadTables(t)
	// 只會第 3 系（技能 19）。
	knows := func(id int) bool { return id == aiSchoolSkill(3) }

	r := rng.NewWithSeed(11)
	found := false
	for i := 0; i < 300; i++ {
		id, ok := AISpellChoice(r, tb, 999, knows)
		if !ok {
			continue
		}
		found = true
		if id < aiSpellRange[3] || id >= aiSpellRange[4] {
			t.Fatalf("只會第 3 系，卻挑到 id %d", id)
		}
	}
	if !found {
		t.Error("一次都沒挑到，測試沒驗到東西")
	}
}

func TestAISpellChoice_NilArgs(t *testing.T) {
	tb := loadTables(t)
	if _, ok := AISpellChoice(nil, tb, 10, nil); ok {
		t.Error("沒有擲點來源卻挑得到")
	}
	if _, ok := AISpellChoice(rng.NewWithSeed(1), nil, 10, nil); ok {
		t.Error("沒有法術表卻挑得到")
	}
}
