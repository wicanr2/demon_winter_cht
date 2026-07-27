package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func seerParty() []Character {
	c := Character{Name: "靈視者", CurrentHP: 10, MaxHP: 10}
	c.Skills[SkillViewRoom] = true
	return []Character{c}
}

// specialAt 建一張只有指定座標有記錄的清單。cls 是類別、v 是低 5 bit。
func specialAt(x, y int, cls, v byte) *scenario.SpecialTiles {
	return &scenario.SpecialTiles{Tiles: []scenario.SpecialTile{
		{X: byte(x), Y: byte(y), Attr: cls<<5 | v},
	}}
}

// 沒有人會觀室 → 什麼都不做，**而且不消耗次數**。
func TestViewRoomNeedsTheSkill(t *testing.T) {
	uses := byte(0)
	res := ViewRoom([]Character{{Name: "路人", CurrentHP: 5}},
		specialAt(1, 5, scenario.SpecialClassEventA, 0),
		corridor{wallX: 10}, 0, 5, East, &uses)

	if !res.NoSkill {
		t.Error("沒人會觀室卻跑了")
	}
	if uses != 0 {
		t.Errorf("沒人會卻消耗了次數：%d", uses)
	}
}

// 一天三次（**不是手冊寫的一次**）。第四次拒絕。
func TestViewRoomThreeTimesADay(t *testing.T) {
	st := specialAt(1, 5, scenario.SpecialClassEventA, 0)
	uses := byte(0)
	for i := 1; i <= PsychicUsesPerDay; i++ {
		if res := ViewRoom(seerParty(), st, corridor{wallX: 10}, 0, 5, East, &uses); res.Exhausted {
			t.Fatalf("第 %d 次就說耗盡了（上限應該是 %d）", i, PsychicUsesPerDay)
		}
	}
	if uses != PsychicUsesPerDay {
		t.Errorf("次數 = %d，預期 %d", uses, PsychicUsesPerDay)
	}
	if !ViewRoom(seerParty(), st, corridor{wallX: 10}, 0, 5, East, &uses).Exhausted {
		t.Errorf("第 %d 次還用得到", PsychicUsesPerDay+1)
	}
}

// **看不到東西也算用掉一次** —— 原版的 inc 排在掃描之前。
func TestViewRoomSpendsAUseEvenWhenNothingFound(t *testing.T) {
	uses := byte(0)
	res := ViewRoom(seerParty(), &scenario.SpecialTiles{},
		corridor{wallX: 10}, 0, 5, East, &uses)
	if res.Hit != nil {
		t.Fatal("空清單卻看到東西")
	}
	if uses != 1 {
		t.Errorf("什麼都沒看到卻沒扣次數：%d", uses)
	}
}

// 只看三格。
func TestViewRoomRangeIsThree(t *testing.T) {
	for _, tc := range []struct {
		x    int
		want bool
	}{
		{1, true}, {2, true}, {3, true}, {4, false},
	} {
		uses := byte(0)
		res := ViewRoom(seerParty(), specialAt(tc.x, 5, scenario.SpecialClassEventA, 0),
			corridor{wallX: 20}, 0, 5, East, &uses)
		if got := res.Hit != nil; got != tc.want {
			t.Errorf("x=%d 看得到 = %v，預期 %v（範圍 %d 格）",
				tc.x, got, tc.want, ViewRoomRange)
		}
	}
}

// 類別 0（預設文字）與類別 4（傳送）都跳過，繼續往後看。
func TestViewRoomSkipsPlainAndTeleport(t *testing.T) {
	st := &scenario.SpecialTiles{Tiles: []scenario.SpecialTile{
		{X: 1, Y: 5, Attr: 0}, // 類別 0
		{X: 2, Y: 5, Attr: scenario.SpecialClassTeleport << 5}, // 類別 4
		{X: 3, Y: 5, Attr: scenario.SpecialClassTrap<<5 | 1},   // 類別 3
	}}
	uses := byte(0)
	res := ViewRoom(seerParty(), st, corridor{wallX: 20}, 0, 5, East, &uses)
	if res.Hit == nil {
		t.Fatal("三格之內有一個陷阱卻沒看到")
	}
	if res.X != 3 {
		t.Errorf("看到 x=%d，預期 3（前兩格是類別 0 與 4，要跳過）", res.X)
	}
}

// 觀室**不改寫** `nSS.DAT` —— 手冊：「用觀室技巧也能看到陷阱，
// 但不會標記為『已注意』」。這一條單獨釘，因為 `L` 那條路會改寫，
// 兩者共用同一份清單，很容易在重構時被合成一條。
func TestViewRoomDoesNotMarkAnything(t *testing.T) {
	st := specialAt(1, 5, scenario.SpecialClassTrap, 2)
	before := st.Tiles[0].Attr
	uses := byte(0)

	ViewRoom(seerParty(), st, corridor{wallX: 10}, 0, 5, East, &uses)

	if st.Tiles[0].Attr != before {
		t.Errorf("觀室改寫了記錄：%#x → %#x（不該標記已注意）",
			before, st.Tiles[0].Attr)
	}
}

// 撞牆就停。
func TestViewRoomStopsAtWall(t *testing.T) {
	uses := byte(0)
	res := ViewRoom(seerParty(), specialAt(2, 5, scenario.SpecialClassEventA, 0),
		corridor{wallX: 2}, 0, 5, East, &uses)
	if res.Hit != nil {
		t.Error("牆在 x=2 卻看到了 x=2 的記錄")
	}
}

// 睡覺把兩個靈視計數清 0。
func TestResetPsychicUses(t *testing.T) {
	s := &scenario.SaveGame{ViewRoomUses: 3, ViewItemUses: 2}
	ResetPsychicUses(s)
	if s.ViewRoomUses != 0 || s.ViewItemUses != 0 {
		t.Errorf("睡覺之後 = %d／%d，預期 0／0", s.ViewRoomUses, s.ViewItemUses)
	}
	ResetPsychicUses(nil) // 不 panic
}

// 死掉的靈視者不提供技能。
func TestViewRoomIgnoresTheDead(t *testing.T) {
	p := seerParty()
	p[0].Status = scenario.StatusDead
	uses := byte(0)
	if !ViewRoom(p, specialAt(1, 5, scenario.SpecialClassEventA, 0),
		corridor{wallX: 10}, 0, 5, East, &uses).NoSkill {
		t.Error("死掉的靈視者還在提供技能")
	}
}
