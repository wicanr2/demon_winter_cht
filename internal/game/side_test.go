package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 六個陣營值分成兩側，門檻是 10。
func TestSideIsPlayer(t *testing.T) {
	for side, want := range map[int]bool{
		SideMonster: false, SideCharmedPlayer: false,
		SideEnemyIllusion: false, SideEnemySummon: false,
		SidePlayer: true, SideCharmedMonster: true,
		SideIllusion: true, SideSummon: true,
	} {
		if got := SideIsPlayer(side); got != want {
			t.Errorf("陣營值 %d 判成玩家側 = %v，預期 %v", side, got, want)
		}
	}
}

// 魅惑的對翻表是個對合：翻兩次回到原點，而且一定會換邊。
//
// 這條是 side.go 那套推導的自我檢查 —— 原版那串比對（0x0d6a–0x0d88）
// 若我抄錯任何一格，對合性或換邊性就會破。
func TestSideFlip_IsInvolutionAndSwapsSides(t *testing.T) {
	for _, side := range []int{
		SideMonster, SideCharmedPlayer, SideEnemySummon,
		SidePlayer, SideCharmedMonster, SideSummon,
	} {
		flipped, ok := SideFlip(side)
		if !ok {
			t.Fatalf("陣營值 %d 不在對翻表裡", side)
		}
		if SideIsPlayer(flipped) == SideIsPlayer(side) {
			t.Errorf("陣營值 %d → %d，沒有換邊", side, flipped)
		}
		back, ok := SideFlip(flipped)
		if !ok || back != side {
			t.Errorf("陣營值 %d → %d → %d，不是對合", side, flipped, back)
		}
	}
}

// 表上沒有的值原樣回傳（原版 0x0d88 的 default 什麼都不做）。
func TestSideFlip_UnknownIsNoop(t *testing.T) {
	// 3／13 是幻化生物：原版的比對鏈就是沒有這兩個值 —— 幻影不能被附身。
	for _, side := range []int{0, SideEnemyIllusion, 5, 10, SideIllusion, 99} {
		got, ok := SideFlip(side)
		if ok || got != side {
			t.Errorf("陣營值 %d → (%d, %v)，預期原樣回傳", side, got, ok)
		}
	}
}

// 被魅惑的玩家角色仍然是玩家角色（IsPlayer 不動），只是換邊打。
//
// 原版就是靠這個區分：槽位決定有沒有 PC 記錄，`+0x20` 決定替誰打。
// AI 施法前查符文系技能旗標那一支（`== 2`）正是踩在這個區分上。
func TestCharmedPlayerKeepsRecord(t *testing.T) {
	u := &Unit{Slot: PlayerSlotStart, IsPlayer: true, Side: SidePlayer}
	side, _ := SideFlip(u.Side)
	u.Side = side

	if !u.IsPlayer {
		t.Error("被魅惑之後 IsPlayer 被改掉了 —— PC 記錄不該跟著換邊")
	}
	if u.OnPlayerSide() {
		t.Error("被魅惑之後還算在玩家側")
	}
	if u.Side != SideCharmedPlayer {
		t.Errorf("被魅惑的玩家角色陣營值 %d，預期 %d", u.Side, SideCharmedPlayer)
	}
}

// NewBattle 會替沒設過的單位補上預設陣營值。
func TestNewBattle_FillsDefaultSide(t *testing.T) {
	monster := &Unit{Slot: 0, X: 1, Y: 1, HP: 5, MaxHP: 5}
	player := &Unit{Slot: PlayerSlotStart, X: 2, Y: 2, HP: 5, MaxHP: 5, IsPlayer: true}
	kept := &Unit{Slot: 1, X: 3, Y: 3, HP: 5, MaxHP: 5, Side: SideCharmedMonster}
	NewBattle(rng.NewWithSeed(1), []*Unit{monster, player, kept})

	if monster.Side != SideMonster {
		t.Errorf("怪物預設陣營值 %d，預期 %d", monster.Side, SideMonster)
	}
	if player.Side != SidePlayer {
		t.Errorf("玩家預設陣營值 %d，預期 %d", player.Side, SidePlayer)
	}
	if kept.Side != SideCharmedMonster {
		t.Errorf("已經設過的陣營值被覆蓋成 %d", kept.Side)
	}
}

// 附身術成功率：投入 × 300 / (2×HP + 法力)，整數除法。
func TestPossessionChance(t *testing.T) {
	for _, c := range []struct{ invested, hp, sp, want int }{
		{10, 10, 10, 100}, // 3000 / 30
		{5, 10, 10, 50},
		{1, 30, 0, 5},    // 300 / 60
		{20, 1, 0, 3000}, // 沒有鉗制，超過 100 就是必中
		{10, 0, 0, 0},    // 分母 0：原版會當掉，這裡回 0
	} {
		if got := PossessionChance(c.invested, c.hp, c.sp); got != c.want {
			t.Errorf("投入 %d、HP %d、法力 %d：成功率 %d，預期 %d",
				c.invested, c.hp, c.sp, got, c.want)
		}
	}
}

// 附身成功就換邊，失敗什麼都不動。
func TestPossess(t *testing.T) {
	target := &Unit{Slot: PlayerSlotStart, IsPlayer: true, Side: SidePlayer,
		HP: 10, MaxHP: 10, CurrentSP: 10}

	// 成功率 100，擲 1 必中。
	if !Possess(&fixedRolls{vals: []int{1}}, target, 10) {
		t.Fatal("成功率 100 卻失敗")
	}
	if target.Side != SideCharmedPlayer {
		t.Errorf("附身後陣營值 %d，預期 %d", target.Side, SideCharmedPlayer)
	}

	// 成功率 50（投入 5），擲 51 落空。
	target.Side = SidePlayer
	if Possess(&fixedRolls{vals: []int{51}}, target, 5) {
		t.Error("成功率 50、擲 51 應該失敗")
	}
	if target.Side != SidePlayer {
		t.Error("附身失敗卻換了邊")
	}
}
