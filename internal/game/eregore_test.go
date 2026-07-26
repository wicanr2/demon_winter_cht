package game

import "testing"

// 七條出口，六條開打 —— 這個比例是原版的（`docs/re/83` §3）。
//
// 會想「這麼多條都打，是不是我把樹接錯了」，所以把它釘成測試：
// 走完全部葉節點，數 EregoreFight 的個數。
func TestEregoreAllExits(t *testing.T) {
	type path struct {
		name    string
		choices []int
		want    EregoreOutcome
		pages   []int
	}
	paths := []path{
		{"開場就嗆回去", []int{1}, EregoreFight, []int{1}},
		{"卡雷辛是白騎士 → 白騎士都死了", []int{2, 1, 1}, EregoreFight, nil},
		{"卡雷辛是白騎士 → 談成", []int{2, 1, 2}, EregoreDone, []int{7, 8, 9}},
		{"卡雷辛是白騎士 → 提遠古者", []int{2, 1, 3}, EregoreFight, []int{6}},
		{"卡雷辛是古神", []int{2, 2}, EregoreFight, []int{3}},
		{"卡雷辛是巨龍", []int{2, 3}, EregoreFight, []int{4}},
	}

	fights := 0
	for _, p := range paths {
		step := StartEregore(false)
		if step.Outcome != EregoreAsk || step.Choices != 2 {
			t.Fatalf("%s：開場應該是兩選一，得到 %+v", p.name, step)
		}
		for _, c := range p.choices {
			if step.Outcome != EregoreAsk {
				t.Fatalf("%s：還沒問完就結束了", p.name)
			}
			step = EregoreAnswer(step.Next, c)
		}
		if step.Outcome != p.want {
			t.Errorf("%s：結局 = %d，預期 %d", p.name, step.Outcome, p.want)
		}
		if len(step.Pages) != len(p.pages) {
			t.Errorf("%s：頁 = %v，預期 %v", p.name, step.Pages, p.pages)
		}
		for i := range p.pages {
			if i < len(step.Pages) && step.Pages[i] != p.pages[i] {
				t.Errorf("%s：頁 = %v，預期 %v", p.name, step.Pages, p.pages)
				break
			}
		}
		if step.Outcome == EregoreFight {
			fights++
		}
	}
	if fights != 5 {
		t.Errorf("開打的路徑數 = %d，預期 5（六條葉節點裡只有一條談得成）", fights)
	}
}

// 第二次見面跳過全部問答。
func TestEregoreSecondVisit(t *testing.T) {
	step := StartEregore(true)
	if step.Outcome != EregoreDone {
		t.Errorf("第二次見面應該直接結束，得到 %d", step.Outcome)
	}
	if len(step.Pages) != 1 || step.Pages[0] != EregoreFinale {
		t.Errorf("第二次見面應該只播頁 %d，得到 %v", EregoreFinale, step.Pages)
	}
}

// 不合法的選項要重問同一頁，不能往前走。
func TestEregoreRejectsBadChoice(t *testing.T) {
	for _, c := range []int{0, 3, 99, -1} {
		step := EregoreAnswer(eregoreNodeOpening, c)
		if step.Outcome != EregoreAsk || step.Next != eregoreNodeOpening {
			t.Errorf("選項 %d 應該重問開場那一頁，得到 %+v", c, step)
		}
	}
	// 三選一的節點，3 是合法的。
	if step := EregoreAnswer(eregoreNodeWhoIs, 3); step.Outcome != EregoreFight {
		t.Errorf("節點 %d 的選項 3 應該開打，得到 %+v", eregoreNodeWhoIs, step)
	}
}
