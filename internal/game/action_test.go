package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// twoUnitBattle 建一場「一個玩家、一隻怪」的戰鬥，玩家速度可指定。
func twoUnitBattle(t *testing.T, playerSpeed int) *Battle {
	t.Helper()
	player := &Unit{Slot: PlayerSlotStart, Name: "玩家", X: 8, Y: 1,
		Speed: playerSpeed, Skill: 10, HP: 30, MaxHP: 30, IsPlayer: true}
	monster := &Unit{Slot: 0, Name: "怪物", X: 2, Y: 1,
		Speed: 1, Skill: 5, HP: 20, MaxHP: 20}

	b := NewBattle(rng.NewWithSeed(1), []*Unit{player, monster})
	b.BeginRound()
	if b.Current() != player {
		t.Fatalf("速度 %d 的玩家應該先行動，實際是 %v", playerSpeed, b.Current().Name)
	}
	return b
}

// 行動點初值 = 速度屬性。
//
// 這一條是**手冊補上反組譯缺口**的地方：`[0x5190]` 的初始化點落在
// Ghidra 的分析間隙裡沒追出來，初值來自手冊「移動點數等於其速度值」。
func TestPoints_InitialisedToSpeed(t *testing.T) {
	for _, speed := range []int{3, 7, 12, 20} {
		b := twoUnitBattle(t, speed)
		if got := b.Points(); got != speed {
			t.Errorf("速度 %d 的單位起手 %d 點，預期 %d", speed, got, speed)
		}
	}
}

// 攻擊 3 點且**不結束回合**：12 點速度可以攻擊兩次再閃避。
//
// 手冊的移動點數表在 C／U／T／P／L 標星號代表結束回合，攻擊沒有星號 ——
// 這是原版刻意的攻守配置設計，把攻擊也做成結束回合會讓戰鬥完全變樣。
func TestSpend_AttackDoesNotEndTurn(t *testing.T) {
	b := twoUnitBattle(t, 12)
	player := b.Current()

	for i := 1; i <= 2; i++ {
		spent, ok := b.Spend(ActionAttack)
		if !ok {
			t.Fatalf("第 %d 次攻擊應該付得起", i)
		}
		if spent != 3 {
			t.Errorf("第 %d 次攻擊花了 %d 點，預期 3", i, spent)
		}
		if b.Current() != player {
			t.Fatalf("第 %d 次攻擊後不該換人", i)
		}
	}
	if got := b.Points(); got != 6 {
		t.Errorf("攻擊兩次後剩 %d 點，預期 6", got)
	}
}

// 施法／使用道具／驅散不死／祈禱／汲取法力做完就換人，不管還剩幾點。
func TestSpend_TurnEndingActions(t *testing.T) {
	for _, a := range []Action{
		ActionCast, ActionUseItem, ActionTurnUndead, ActionPray, ActionLeech,
	} {
		b := twoUnitBattle(t, 20)
		player := b.Current()

		if _, ok := b.Spend(a); !ok {
			t.Fatalf("%s 在 20 點時應該付得起", ActionName(a))
		}
		if b.Current() == player {
			t.Errorf("%s 之後應該換人（剩 %d 點也一樣）", ActionName(a), b.Points())
		}
	}
}

// 點數扣光就自動換人，不必呼叫端記得 EndTurn。
func TestSpend_ExhaustingPointsEndsTurn(t *testing.T) {
	b := twoUnitBattle(t, 6)
	player := b.Current()

	b.Spend(ActionAttack)
	if b.Current() != player {
		t.Fatal("剩 3 點時不該換人")
	}
	b.Spend(ActionAttack)
	if b.Current() == player {
		t.Error("點數歸零後應自動換人")
	}
}

// 點數不足時什麼都不會發生 —— 呼叫端不該執行該動作的效果。
func TestSpend_RefusesWhenTooExpensive(t *testing.T) {
	b := twoUnitBattle(t, 2)

	if _, ok := b.Spend(ActionAttack); ok {
		t.Error("2 點不該付得起 3 點的攻擊")
	}
	if got := b.Points(); got != 2 {
		t.Errorf("失敗的動作不該扣點，剩 %d 點", got)
	}
	// 但 2 點還轉得動向。
	if _, ok := b.Spend(ActionTurnCW); !ok {
		t.Error("2 點應該付得起 1 點的轉向")
	}
}

// 閃避吃掉全部剩餘點數，每 3 點換 1 點狀態計數。
func TestDoDodge_ConvertsRemainingPoints(t *testing.T) {
	cases := []struct {
		speed     int
		attacks   int
		wantBonus int
	}{
		{12, 0, 4}, // 12/3
		{12, 2, 2}, // 攻擊兩次剩 6 → 6/3
		{7, 1, 1},  // 剩 4 → 4/3 無條件捨去
		{3, 1, 0},  // 剩 0
	}
	for _, c := range cases {
		b := twoUnitBattle(t, c.speed)
		player := b.Current()
		for i := 0; i < c.attacks; i++ {
			b.Spend(ActionAttack)
		}
		before := player.StatusCount

		got := b.DoDodge()
		if got != c.wantBonus {
			t.Errorf("速度 %d 攻擊 %d 次後閃避加成 %d，預期 %d",
				c.speed, c.attacks, got, c.wantBonus)
		}
		if player.StatusCount != before+c.wantBonus {
			t.Errorf("速度 %d：狀態計數 = %d，預期 %d",
				c.speed, player.StatusCount, before+c.wantBonus)
		}
	}
}

// 閃避永遠按得下去，就算一點都不剩 —— 它是「結束回合」而不是要付費的動作。
func TestDoDodge_AlwaysAllowed(t *testing.T) {
	b := twoUnitBattle(t, 3)
	player := b.Current()
	b.Spend(ActionAttack) // 剩 0 點，已自動換人

	if b.Current() == player {
		t.Fatal("前置條件不成立：攻擊完應已換人")
	}
	// 換到怪物身上，閃避仍要能執行。
	if !b.CanAct(ActionDodge) {
		t.Error("閃避應永遠可執行")
	}
}

// 檢視不耗點數，也不結束回合。
func TestSpend_FreeActions(t *testing.T) {
	for _, a := range []Action{ActionExamine, ActionSound} {
		b := twoUnitBattle(t, 10)
		player := b.Current()

		spent, ok := b.Spend(a)
		if !ok || spent != 0 {
			t.Errorf("%s 應該免費，得到 spent=%d ok=%v", ActionName(a), spent, ok)
		}
		if b.Points() != 10 {
			t.Errorf("%s 之後剩 %d 點，預期 10", ActionName(a), b.Points())
		}
		if b.Current() != player {
			t.Errorf("%s 之後不該換人", ActionName(a))
		}
	}
}

// 換人時重新配點，換回同一個人不會重配。
func TestPoints_ResetPerUnit(t *testing.T) {
	fast := &Unit{Slot: PlayerSlotStart, Name: "快", X: 8, Y: 1,
		Speed: 9, HP: 10, MaxHP: 10, IsPlayer: true}
	slow := &Unit{Slot: 0, Name: "慢", X: 2, Y: 1,
		Speed: 4, HP: 10, MaxHP: 10}

	b := NewBattle(rng.NewWithSeed(1), []*Unit{fast, slow})
	b.BeginRound()

	if b.Current() != fast || b.Points() != 9 {
		t.Fatalf("先手應是快（9 點），實際 %s（%d 點）", b.Current().Name, b.Points())
	}
	b.Spend(ActionEndTurn)

	if b.Current() != slow || b.Points() != 4 {
		t.Fatalf("次手應是慢（4 點），實際 %s（%d 點）", b.Current().Name, b.Points())
	}

	// 下一回合重新配點。
	b.Spend(ActionEndTurn)
	b.BeginRound()
	if b.Current() != fast || b.Points() != 9 {
		t.Errorf("新回合應重新配點，實際 %s（%d 點）", b.Current().Name, b.Points())
	}
}

// 主動結束回合不耗點。
func TestSpend_EndTurn(t *testing.T) {
	b := twoUnitBattle(t, 10)
	player := b.Current()
	if _, ok := b.Spend(ActionEndTurn); !ok {
		t.Fatal("結束回合應永遠可執行")
	}
	if b.Current() == player {
		t.Error("結束回合後應換人")
	}
}
