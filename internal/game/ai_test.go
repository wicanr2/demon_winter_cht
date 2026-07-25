package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func aiBattle(t *testing.T) *Battle {
	t.Helper()
	var units []*Unit
	for i := 0; i < 3; i++ { // 怪物槽 0–2
		units = append(units, &Unit{
			Slot: i, Name: "怪" + string(rune('A'+i)),
			X: 2, Y: i + 1, HP: 10, MaxHP: 10, Speed: 5,
		})
	}
	for i := 0; i < 3; i++ { // 玩家槽 7–9
		units = append(units, &Unit{
			Slot: PlayerSlotStart + i, Name: "我" + string(rune('A'+i)),
			X: 7, Y: i + 1, HP: 10, MaxHP: 10, Speed: 5, IsPlayer: true,
		})
	}
	return NewBattle(rng.NewWithSeed(42), units)
}

// 目標一旦選定就要記住，不能每回合重挑。
//
// 這是原版的行為（目標槽位存在戰鬥單位記錄的 unit+0x1e）。少了記憶，
// 整群怪物每回合都會鎖定同一個「第一個敵人」，玩家的隊形完全失去意義。
func TestAITarget_RemembersTarget(t *testing.T) {
	b := aiBattle(t)
	m := b.Unit(0)

	first := b.AITarget(m)
	if first == nil {
		t.Fatal("挑不到目標")
	}
	for i := 0; i < 20; i++ {
		if got := b.AITarget(m); got != first {
			t.Fatalf("第 %d 次改挑 %s，應該記住 %s", i, got.Name, first.Name)
		}
	}
}

// 目標死了就要換人，而且換到的必須還活著。
func TestAITarget_RetargetsWhenDead(t *testing.T) {
	b := aiBattle(t)
	m := b.Unit(0)

	first := b.AITarget(m)
	if first == nil {
		t.Fatal("挑不到目標")
	}
	b.Kill(first)

	next := b.AITarget(m)
	if next == nil {
		t.Fatal("還有活著的敵人，卻挑不到目標")
	}
	if next == first {
		t.Error("目標已死，卻沒有換人")
	}
	if !next.Alive() {
		t.Error("換到的目標是死的")
	}
}

// 只能挑敵對陣營 —— 挑到同伴就會出現怪物打怪物。
func TestAITarget_NeverPicksAlly(t *testing.T) {
	b := aiBattle(t)

	for _, slot := range []int{0, 1, 2} {
		m := b.Unit(slot)
		for i := 0; i < 50; i++ {
			got := b.AITarget(m)
			if got == nil {
				t.Fatal("挑不到目標")
			}
			if got.IsPlayer == m.IsPlayer {
				t.Fatalf("%s 挑到同陣營的 %s", m.Name, got.Name)
			}
			m.AITargetSlot = noAITarget // 逼它重挑
		}
	}
}

// 敵人全滅時回 nil，不能卡在重挑迴圈裡。
//
// 原版的重挑是「隨機挑、驗證、不合格再挑」的無界迴圈 —— 它只在還有敵人
// 的前提下被呼叫，所以沒事；這裡多一道保護，不然全滅那一瞬間會當掉。
func TestAITarget_NilWhenNoEnemies(t *testing.T) {
	b := aiBattle(t)
	m := b.Unit(0)

	for i := 0; i < 3; i++ {
		b.Kill(b.Unit(PlayerSlotStart + i))
	}
	if got := b.AITarget(m); got != nil {
		t.Errorf("敵人全滅，卻挑到 %s", got.Name)
	}
}

// 新戰鬥的單位不能「一開始就記得」打 0 號。
//
// AITargetSlot 的零值 0 剛好是合法槽位，忘了初始化的話每隻怪物開場都會
// 直奔 0 號槽 —— 而 0 號是怪物自己那一側，行為會整個亂掉。
func TestNewBattle_ClearsAITarget(t *testing.T) {
	b := aiBattle(t)
	for slot := 0; slot < CombatSlots; slot++ {
		u := b.Unit(slot)
		if u == nil {
			continue
		}
		if u.AITargetSlot != noAITarget {
			t.Errorf("槽位 %d 開場的目標是 %d，應該是「還沒有目標」",
				slot, u.AITargetSlot)
		}
	}
}

// 目標分散：三隻怪物各自挑，不該全部擠在同一個人身上。
func TestAITarget_SpreadsAcrossParty(t *testing.T) {
	b := aiBattle(t)
	seen := map[string]bool{}
	for _, slot := range []int{0, 1, 2} {
		m := b.Unit(slot)
		for i := 0; i < 30; i++ {
			seen[b.AITarget(m).Name] = true
			m.AITargetSlot = noAITarget
		}
	}
	if len(seen) < 2 {
		t.Errorf("三隻怪物重挑 90 次只挑中 %d 個目標 —— 看起來沒有隨機性", len(seen))
	}
}
