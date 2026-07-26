package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// battleTestBase 把測試裡的小座標平移進 15×15 戰場：(5,5) 落在正中央。
//
// 戰場的合法座標是 6–20（`docs/re/36`），測試寫 0–8 比較好讀，
// 所以統一在這裡加上偏移，斷言那邊也用 tx()／ty() 換算。
const battleTestBase = BattleCentreX - 5

func tx(n int) int { return n + battleTestBase }
func ty(n int) int { return n + battleTestBase }

// gridBattle 擺一個玩家與一隻怪，位置與面向可指定（座標會平移進戰場）。
func gridBattle(px, py, pf, mx, my int) (*Battle, *Unit, *Unit) {
	px, py, mx, my = tx(px), ty(py), tx(mx), ty(my)
	player := &Unit{Slot: PlayerSlotStart, Name: "玩家", X: px, Y: py, Facing: pf,
		Speed: 12, Skill: 10, HP: 30, MaxHP: 30, IsPlayer: true}
	monster := &Unit{Slot: 0, Name: "怪物", X: mx, Y: my,
		Speed: 1, Skill: 5, HP: 20, MaxHP: 20}
	b := NewBattle(rng.NewWithSeed(1), []*Unit{player, monster})
	b.BeginRound()
	return b, player, monster
}

// 攻擊沒有選目標的步驟，面向決定打誰。
func TestTargetInFront_FollowsFacing(t *testing.T) {
	cases := []struct {
		facing   Facing
		mx, my   int
		wantHit  bool
		facingZh string
	}{
		{North, 5, 4, true, "北"},
		{East, 6, 5, true, "東"},
		{South, 5, 6, true, "南"},
		{West, 4, 5, true, "西"},
		{North, 5, 6, false, "北（怪在南邊）"},
		{East, 7, 5, false, "東（隔了一格）"},
	}
	for _, c := range cases {
		b, player, monster := gridBattle(5, 5, int(c.facing), c.mx, c.my)
		got := b.TargetInFront(player)
		if c.wantHit && got != monster {
			t.Errorf("面向%s、怪在 (%d,%d)：應打得到，得到 %v",
				c.facingZh, c.mx, c.my, got)
		}
		if !c.wantHit && got != nil {
			t.Errorf("面向%s、怪在 (%d,%d)：不該打得到", c.facingZh, c.mx, c.my)
		}
	}
}

// 正前方是友軍時不算目標 —— 免得打到自己人。
func TestTargetInFront_IgnoresAllies(t *testing.T) {
	a := &Unit{Slot: PlayerSlotStart, Name: "甲", X: 5, Y: 5, Facing: int(North),
		Speed: 10, HP: 10, MaxHP: 10, IsPlayer: true}
	ally := &Unit{Slot: PlayerSlotStart + 1, Name: "乙", X: 5, Y: 4,
		Speed: 5, HP: 10, MaxHP: 10, IsPlayer: true}
	b := NewBattle(rng.NewWithSeed(1), []*Unit{a, ally})
	b.BeginRound()

	if got := b.TargetInFront(a); got != nil {
		t.Errorf("正前方是友軍不該成為攻擊目標，得到 %v", got.Name)
	}
}

// 已死的單位不佔位置，也不是目標。
func TestUnitAt_SkipsDead(t *testing.T) {
	b, _, monster := gridBattle(5, 5, int(North), 5, 4)
	b.Kill(monster)

	if got := b.UnitAt(tx(5), ty(4)); got != nil {
		t.Errorf("已死單位不該再佔格子，得到 %v", got.Name)
	}
}

func TestStep_MovesAndSpends(t *testing.T) {
	b, player, _ := gridBattle(5, 5, int(East), 0, 0)

	if !b.Step(player) {
		t.Fatal("前方空地應該走得動")
	}
	if player.X != tx(6) || player.Y != ty(5) {
		t.Errorf("走到 (%d,%d)，預期 (%d,%d)", player.X, player.Y, tx(6), ty(5))
	}
	if got := b.Points(); got != 10 {
		t.Errorf("前進應花 2 點，剩 %d，預期 10", got)
	}
}

// 前方有人就走不過去，而且不能扣點。
func TestStep_BlockedByUnit(t *testing.T) {
	b, player, _ := gridBattle(5, 5, int(East), 6, 5)

	if b.Step(player) {
		t.Error("前方有單位不該走得動")
	}
	if player.X != tx(5) {
		t.Errorf("被擋下時不該移動，位置 (%d,%d)", player.X, player.Y)
	}
	if got := b.Points(); got != 12 {
		t.Errorf("被擋下時不該扣點，剩 %d，預期 12", got)
	}
}

// 走出可站範圍要被擋下。
//
// 可站範圍是 6–20（外面就是那圈牆），這裡直接用絕對座標，不走 tx()／ty()。
func TestStep_BlockedByEdge(t *testing.T) {
	cases := []struct {
		x, y   int
		facing Facing
		where  string
	}{
		{BattleFieldMin, BattleCentreY, West, "西"},
		{BattleCentreX, BattleFieldMin, North, "北"},
		{BattleFieldMax, BattleCentreY, East, "東"},
		{BattleCentreX, BattleFieldMax, South, "南"},
	}
	for _, c := range cases {
		b, player, _ := gridBattle(c.x-battleTestBase, c.y-battleTestBase,
			int(c.facing), 0, 0)
		if b.Step(player) {
			t.Errorf("從 (%d,%d) 往%s走出邊界應被擋下", c.x, c.y, c.where)
		}
		if got := b.Points(); got != 12 {
			t.Errorf("從 (%d,%d) 往%s被擋下時不該扣點，剩 %d", c.x, c.y, c.where, got)
		}
	}
}

// 站在 X=0 的單位根本不會被排進行動順序 —— 釘住這個哨兵語意，
// 免得日後有人「順手」把最小 X 改成 0。
func TestBattleGridMinX_ZeroIsSentinel(t *testing.T) {
	ghost := &Unit{Slot: PlayerSlotStart, Name: "在零欄", X: 0, Y: 5,
		Speed: 99, HP: 10, MaxHP: 10, IsPlayer: true}
	real := &Unit{Slot: 0, Name: "怪物", X: 5, Y: 5,
		Speed: 1, HP: 10, MaxHP: 10}

	b := NewBattle(rng.NewWithSeed(1), []*Unit{ghost, real})
	b.BeginRound()
	if got := b.Current(); got != real {
		t.Errorf("X=0 的單位不該進行動順序，Current() = %v", got.Name)
	}
}

// 點數不足時走不動，且位置不變。
func TestStep_RefusedWhenOutOfPoints(t *testing.T) {
	b, player, _ := gridBattle(5, 5, int(East), 0, 0)
	// 12 點：攻擊四次會扣完並換人，改用轉向把點數磨到剩 1。
	for i := 0; i < 11; i++ {
		b.Spend(ActionTurnCW)
	}
	if got := b.Points(); got != 1 {
		t.Fatalf("前置條件不成立：剩 %d 點，預期 1", got)
	}
	if b.Step(player) {
		t.Error("剩 1 點不該走得動（前進要 2 點）")
	}
	if player.X != tx(5) {
		t.Errorf("失敗的前進不該移動，位置 (%d,%d)", player.X, player.Y)
	}
}

func TestTurnTo(t *testing.T) {
	cases := []struct {
		from Facing
		act  Action
		want Facing
	}{
		{North, ActionTurnCW, East},
		{North, ActionTurnCCW, West},
		{North, ActionAboutFace, South},
		{West, ActionTurnCW, North},
		{West, ActionTurnCCW, South},
		{East, ActionAboutFace, West},
	}
	for _, c := range cases {
		b, player, _ := gridBattle(5, 5, int(c.from), 0, 0)
		if !b.TurnTo(player, c.act) {
			t.Fatalf("%s 應該轉得動", ActionName(c.act))
		}
		if Facing(player.Facing) != c.want {
			t.Errorf("面向 %d 做 %s → %d，預期 %d",
				c.from, ActionName(c.act), player.Facing, c.want)
		}
		if got := b.Points(); got != 11 {
			t.Errorf("轉向應花 1 點，剩 %d", got)
		}
	}
}

// 轉向之後前方的目標跟著換 —— 這是「先轉再打」能成立的前提。
func TestTurnThenTarget(t *testing.T) {
	b, player, monster := gridBattle(5, 5, int(North), 6, 5)

	if b.TargetInFront(player) != nil {
		t.Fatal("面向北時不該打得到東邊的怪")
	}
	b.TurnTo(player, ActionTurnCW)
	if got := b.TargetInFront(player); got != monster {
		t.Error("轉向東之後應該打得到")
	}
}
