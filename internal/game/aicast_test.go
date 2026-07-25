package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 跳表裡有實作的那十個效果，以及走 default 的那七個。
//
// 表是從 DEMON.INT 的 138d:0c65 直接讀出來的 17 個 word；這條把讀出來的
// 結果釘住，免得日後憑印象改。
func TestAIEffectHandled(t *testing.T) {
	handled := map[int]bool{
		1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true,
		0x0b: true, 0x0e: true, 0x10: true,
	}
	for effect := 0; effect <= 0x10; effect++ {
		if got := AIEffectHandled(effect); got != handled[effect] {
			t.Errorf("效果 %d（0x%x）有實作 = %v，預期 %v",
				effect, effect, got, handled[effect])
		}
	}
}

// K 的正負只在效果 3–7 起作用，其餘一律打玩家側。
//
// 這是「怪物法師對玩家放增益」那個 bug 的直接對應條件。
func TestAISpellTargetsOwnSide(t *testing.T) {
	for effect := 0; effect <= 0x10; effect++ {
		inRange := effect >= 3 && effect <= 7
		for _, k := range []int{-5, 0, 3} {
			want := inRange && k > 0
			if got := AISpellTargetsOwnSide(effect, k); got != want {
				t.Errorf("效果 %d、K %d：打自己人 = %v，預期 %v",
					effect, k, got, want)
			}
		}
	}
}

// 兩隻怪物、兩個玩家，四個人擠在同一格附近。
func areaBattle(t *testing.T) (*Battle, *Unit) {
	t.Helper()
	caster := &Unit{Slot: 0, Name: "法師", X: 1, Y: 1, HP: 10, MaxHP: 10}
	units := []*Unit{caster}
	for i := 0; i < 3; i++ {
		units = append(units, &Unit{
			Slot: PlayerSlotStart + i, Name: "我" + string(rune('A'+i)),
			X: 5, Y: 5 + i, HP: 10, MaxHP: 10, IsPlayer: true,
		})
	}
	return NewBattle(rng.NewWithSeed(2), units), caster
}

// 方框是 5×5、中心含在內；框外的不算。
func TestAIAreaCountAt_BoxShape(t *testing.T) {
	b, caster := areaBattle(t)
	// 中心放在 (5,5)：三個玩家在 (5,5)(5,6)(5,7)，都在框內。
	c := b.AIAreaCountAt(caster, 5, 5, 0)
	if c.Enemy != 3 || c.Own != 0 {
		t.Fatalf("中心 (5,5)：己方 %d 敵方 %d，預期 0／3", c.Own, c.Enemy)
	}
	// 中心移到 (5,3)：只剩 (5,5) 那一個在 |Δy| <= 2 內。
	c = b.AIAreaCountAt(caster, 5, 3, 0)
	if c.Enemy != 1 {
		t.Errorf("中心 (5,3)：敵方 %d，預期 1", c.Enemy)
	}
	// 剛好差一格：|Δ| = 3 應該落在框外。
	c = b.AIAreaCountAt(caster, 5, 2, 0)
	if c.Enemy != 0 {
		t.Errorf("中心 (5,2)：敵方 %d，預期 0（|Δy| = 3 已出框）", c.Enemy)
	}
}

// 施法者自己落在框內時要算成一個己方命中。
func TestAIAreaCountAt_CountsCasterItself(t *testing.T) {
	b, caster := areaBattle(t)
	caster.X, caster.Y = 5, 5 // 自己站進去

	c := b.AIAreaCountAt(caster, 5, 5, 0)
	if c.Own != 1 {
		t.Errorf("施法者站在中心，己方命中 %d，預期 1", c.Own)
	}
}

// School 為 4 時，己方裡種族／元素 4／7／10 的不算誤傷。
func TestAIAreaCountAt_ElementImmunity(t *testing.T) {
	b, caster := areaBattle(t)
	friend := &Unit{Slot: 1, Name: "同伴", X: 5, Y: 5, HP: 10, MaxHP: 10,
		RaceOrElement: 7, Side: SideMonster}
	b.units[1] = friend

	if c := b.AIAreaCountAt(caster, 5, 5, 0); c.Own != 1 {
		t.Errorf("School 0：己方命中 %d，預期 1（沒有免疫）", c.Own)
	}
	if c := b.AIAreaCountAt(caster, 5, 5, aiAreaElement); c.Own != 0 {
		t.Errorf("School %d：己方命中 %d，預期 0（種族 7 免疫）", aiAreaElement, c.Own)
	}
	friend.RaceOrElement = 9
	if c := b.AIAreaCountAt(caster, 5, 5, aiAreaElement); c.Own != 1 {
		t.Errorf("種族 9 不在免疫組，卻沒算成誤傷")
	}
}

// 否決條件：己方 × 2 > 敵方。
//
// 邊界要對 —— 己方 1 敵方 2 剛好不否決，己方 1 敵方 1 就否決。
func TestAIAreaCount_Veto(t *testing.T) {
	for _, c := range []struct {
		own, enemy int
		veto       bool
	}{
		{0, 0, false}, {0, 3, false},
		{1, 1, true}, {1, 2, false}, {1, 3, false},
		{2, 3, true}, {2, 4, false},
	} {
		got := AIAreaCount{Own: c.own, Enemy: c.enemy}.Veto()
		if got != c.veto {
			t.Errorf("己方 %d 敵方 %d：否決 = %v，預期 %v",
				c.own, c.enemy, got, c.veto)
		}
	}
}

// 挑目標會挑對邊。
func TestAIPickTarget_Side(t *testing.T) {
	b, caster := areaBattle(t)
	friend := &Unit{Slot: 1, Name: "同伴", X: 2, Y: 2, HP: 10, MaxHP: 10,
		Side: SideMonster}
	b.units[1] = friend

	for i := 0; i < 50; i++ {
		if u := b.AIPickTarget(caster, false, -1); u == nil || !u.IsPlayer {
			t.Fatalf("打敵人卻挑到 %v", u)
		}
		if u := b.AIPickTarget(caster, true, -1); u == nil || u.IsPlayer {
			t.Fatalf("加持自己人卻挑到 %v", u)
		}
	}
}

// 沒有合格對象時回 nil，不會卡在重挑迴圈裡。
//
// 原版是無界重挑，只靠「呼叫時一定還有敵人」這個前提撐著；效果 0x0b 那一支
// 自己也是先掃一遍才進迴圈（0x0b29），所以這個保護不算加料。
func TestAIPickTarget_NoCandidate(t *testing.T) {
	b, caster := areaBattle(t)
	for i := 0; i < 3; i++ {
		b.Kill(b.Unit(PlayerSlotStart + i))
	}
	if u := b.AIPickTarget(caster, false, -1); u != nil {
		t.Errorf("玩家全倒了卻挑到 %s", u.Name)
	}
	// 施法者自己是合格的己方對象 —— 原版的隨機挑也會挑到自己那一格。
	if u := b.AIPickTarget(caster, true, -1); u != caster {
		t.Errorf("己方只剩自己，卻挑到 %v", u)
	}
	if u := b.AIPickTarget(nil, false, -1); u != nil {
		t.Error("沒有施法者卻挑得到目標")
	}
}

// 增益不會往狀態 > 1 的自己人身上放（原版 0x0a7c）。
func TestAIPickTarget_MaxStatus(t *testing.T) {
	b, caster := areaBattle(t)
	friend := &Unit{Slot: 1, Name: "同伴", X: 2, Y: 2, HP: 10, MaxHP: 10,
		Side: SideMonster, Status: StatusPoison}
	b.units[1] = friend

	// 施法者自己狀態 0，同伴狀態 1 —— 兩個都合格。
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		if u := b.AIPickTarget(caster, true, 1); u != nil {
			seen[u.Name] = true
		}
	}
	if !seen["同伴"] {
		t.Error("狀態 1 的同伴應該可以被加持")
	}

	friend.Status = StatusPoison + 1
	for i := 0; i < 50; i++ {
		if u := b.AIPickTarget(caster, true, 1); u == friend {
			t.Fatal("狀態 > 1 的同伴不該被挑到")
		}
	}
}
