package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func mkUnit(slot int, player bool, speed, hp int) *Unit {
	return &Unit{Slot: slot, X: 5, Y: 5, Speed: speed, HP: hp, MaxHP: hp,
		Skill: 25, Strength: 7, IsPlayer: player, WeaponIndex: 1}
}

// 行動順序把玩家與怪物混在一起依速度排，不是「玩家全體行動完再換敵人」。
func TestBattle_TurnOrderInterleavesSides(t *testing.T) {
	b := NewBattle(rng.NewWithSeed(1), []*Unit{
		mkUnit(0, false, 30, 10), // 怪物 快
		mkUnit(1, false, 5, 10),  // 怪物 慢
		mkUnit(7, true, 20, 10),  // 玩家 中
		mkUnit(8, true, 10, 10),  // 玩家 中慢
	})
	b.BeginRound()

	var got []int
	for u := b.Current(); u != nil; u = b.Current() {
		got = append(got, u.Slot)
		b.EndTurn()
	}

	want := []int{0, 7, 8, 1} // 30, 20, 10, 5
	if len(got) != len(want) {
		t.Fatalf("行動順序 = %v，預期 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("行動順序 = %v，預期 %v（兩陣營應交錯）", got, want)
		}
	}
}

func TestBattle_RoundAdvances(t *testing.T) {
	b := NewBattle(rng.NewWithSeed(1), []*Unit{mkUnit(0, false, 10, 10)})

	if b.Round() != 0 {
		t.Errorf("開打前回合數應為 0，得到 %d", b.Round())
	}
	b.BeginRound()
	if b.Round() != 1 {
		t.Errorf("第一回合應為 1，得到 %d", b.Round())
	}
	if b.RoundFinished() {
		t.Error("還有單位可行動，不該算回合結束")
	}
	b.EndTurn()
	if !b.RoundFinished() {
		t.Error("單位都行動完了，應算回合結束")
	}
	b.BeginRound()
	if b.Round() != 2 {
		t.Errorf("第二回合應為 2，得到 %d", b.Round())
	}
}

// 順序是回合開始時算好的，中途死亡的單位不該再行動。
func TestBattle_DeadUnitSkippedMidRound(t *testing.T) {
	victim := mkUnit(1, false, 20, 10)
	b := NewBattle(rng.NewWithSeed(1), []*Unit{
		mkUnit(0, false, 30, 10),
		victim,
		mkUnit(7, true, 10, 10),
	})
	b.BeginRound()

	// 第一個單位行動後，把排在第二的 victim 殺掉。
	if u := b.Current(); u.Slot != 0 {
		t.Fatalf("第一個應是槽 0，得到 %d", u.Slot)
	}
	b.EndTurn()
	b.Kill(victim)

	if u := b.Current(); u == nil || u.Slot != 7 {
		t.Fatalf("已死的槽 1 應被跳過，下一個該是槽 7，得到 %v", u)
	}
}

func TestBattle_Outcome(t *testing.T) {
	// 雙方都活著 → 進行中。
	b := NewBattle(rng.NewWithSeed(1), []*Unit{
		mkUnit(0, false, 10, 10),
		mkUnit(7, true, 10, 10),
	})
	if got := b.Outcome(); got != Ongoing {
		t.Errorf("雙方存活應為 Ongoing，得到 %d", got)
	}

	// 怪物全滅 → 勝利。
	b.Kill(b.Unit(0))
	if got := b.Outcome(); got != Victory {
		t.Errorf("怪物全滅應為 Victory，得到 %d", got)
	}

	// 兩邊都死 → 隊伍全滅優先判為戰敗。
	b.Kill(b.Unit(7))
	if got := b.Outcome(); got != Defeat {
		t.Errorf("隊伍全滅應為 Defeat，得到 %d", got)
	}
}

// 召喚物屬玩家陣營但不算隊伍成員：只剩召喚物存活時仍算全滅。
func TestBattle_SummonsDoNotPreventDefeat(t *testing.T) {
	summon := mkUnit(12, true, 10, 10)
	b := NewBattle(rng.NewWithSeed(1), []*Unit{
		mkUnit(0, false, 10, 10),
		mkUnit(7, true, 10, 10),
		summon,
	})

	b.Kill(b.Unit(7))
	if got := b.Outcome(); got != Defeat {
		t.Errorf("隊伍成員全滅時應判 Defeat（召喚物不算），得到 %d", got)
	}
	if !summon.Alive() {
		t.Error("召喚物本身應該還活著")
	}
}

// 死亡結算要同時清 X/Y 與設狀態。
func TestBattle_KillClearsPosition(t *testing.T) {
	u := mkUnit(0, false, 10, 10)
	b := NewBattle(rng.NewWithSeed(1), []*Unit{u})

	b.Kill(u)

	if u.HP != 0 || u.Status != StatusDead {
		t.Errorf("死亡後 HP 應為 0、狀態為死亡，得到 HP %d 狀態 %d", u.HP, u.Status)
	}
	if u.X != 0 || u.Y != 0 {
		t.Errorf("死亡後座標應清零，得到 (%d,%d)", u.X, u.Y)
	}
	if len(TurnOrder(b.Units())) != 0 {
		t.Error("已死單位不該出現在行動順序裡")
	}
}

func TestBattle_Enemies(t *testing.T) {
	b := NewBattle(rng.NewWithSeed(1), []*Unit{
		mkUnit(0, false, 10, 10),
		mkUnit(1, false, 10, 10),
		mkUnit(7, true, 10, 10),
		mkUnit(12, true, 10, 10), // 召喚物也算玩家陣營
	})

	player := b.Unit(7)
	got := b.Enemies(player)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Errorf("玩家的敵人 = %v，預期 [0 1]", got)
	}

	monster := b.Unit(0)
	got = b.Enemies(monster)
	if len(got) != 2 || got[0] != 7 || got[1] != 12 {
		t.Errorf("怪物的敵人 = %v，預期 [7 12]（召喚物屬玩家陣營）", got)
	}

	// 死掉的不列入。
	b.Kill(b.Unit(1))
	if got := b.Enemies(player); len(got) != 1 || got[0] != 0 {
		t.Errorf("死亡的敵人不該列入，得到 %v", got)
	}
}

func TestBattle_FreeSummonSlot(t *testing.T) {
	b := NewBattle(rng.NewWithSeed(1), nil)

	// 三格全空 → 第一格。
	if got := b.FreeSummonSlot(); got != SummonSlotStart {
		t.Errorf("空戰場的召喚槽 = %d，預期 %d", got, SummonSlotStart)
	}

	tb := loadTables(t)
	e, err := tb.Summon(0)
	if err != nil {
		t.Fatalf("Summon(0): %v", err)
	}
	for s := SummonSlotStart; s < SummonSlotEnd; s++ {
		b.PlaceSummon(s, e, KindSummon, 3, 3)
	}
	if got := b.FreeSummonSlot(); got != -1 {
		t.Errorf("三格全滿應回傳 −1，得到 %d", got)
	}

	// 死掉一隻就空出來 —— 對應手冊「最多同時存在三隻」。
	b.Kill(b.Unit(13))
	if got := b.FreeSummonSlot(); got != 13 {
		t.Errorf("槽 13 死亡後應可重用，得到 %d", got)
	}
}

// 幻術把法力歸零，召喚保留 —— 這是幻化生物不能施法的原因。
func TestBattle_IllusionZeroesSP(t *testing.T) {
	tb := loadTables(t)
	b := NewBattle(rng.NewWithSeed(1), nil)

	// 找一個表值法力不為 0 的生物，否則這條測不出差別。
	var idx = -1
	for i := 0; i < tb.NumSpells() && i < tb.NumSummons(); i++ {
		e, err := tb.Summon(i)
		if err != nil {
			t.Fatalf("Summon(%d): %v", i, err)
		}
		if e.Word(9) != 0 {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Skip("召喚表裡沒有法力不為 0 的生物，這條無從驗證")
	}
	e, _ := tb.Summon(idx)

	summoned := b.PlaceSummon(12, e, KindSummon, 3, 3)
	if summoned.MaxSP == 0 {
		t.Fatal("召喚版應保留表值法力")
	}

	illusion := b.PlaceSummon(13, e, KindIllusion, 3, 3)
	if illusion.MaxSP != 0 || illusion.CurrentSP != 0 {
		t.Errorf("幻術版法力應歸零，得到 %d/%d", illusion.CurrentSP, illusion.MaxSP)
	}
}

// 召喚成本 = 附魔基數 ×4、幻術 ×2。
func TestSummonCost(t *testing.T) {
	tb := loadTables(t)
	for i := 0; i < tb.NumSummons(); i++ {
		e, err := tb.Summon(i)
		if err != nil {
			t.Fatalf("Summon(%d): %v", i, err)
		}
		if got, want := SummonCost(e, KindSummon), e.PowerBase()*4; got != want {
			t.Errorf("生物 %d 召喚成本 = %d，預期 %d", i, got, want)
		}
		if got, want := SummonCost(e, KindIllusion), e.PowerBase()*2; got != want {
			t.Errorf("生物 %d 幻術成本 = %d，預期 %d", i, got, want)
		}
	}
}

// 怪物進場擲點：速度落在基礎值的 [0.7, 1.3)、生命 [0.6, 1.4)，各有下限。
func TestRollMonsterStats_Ranges(t *testing.T) {
	r := rng.NewWithSeed(31337)

	const baseSpeed, baseHP = 100, 100
	loS, hiS := 1<<30, 0
	loH, hiH := 1<<30, 0
	for i := 0; i < 200000; i++ {
		s, h := RollMonsterStats(r, baseSpeed, baseHP)
		if s < loS {
			loS = s
		}
		if s > hiS {
			hiS = s
		}
		if h < loH {
			loH = h
		}
		if h > hiH {
			hiH = h
		}
	}

	if loS < 70 || hiS > 129 {
		t.Errorf("速度範圍 = [%d, %d]，預期落在 [70, 129]", loS, hiS)
	}
	if loH < 60 || hiH > 139 {
		t.Errorf("生命範圍 = [%d, %d]，預期落在 [60, 139]", loH, hiH)
	}
}

func TestRollMonsterStats_Floors(t *testing.T) {
	r := rng.NewWithSeed(4)
	for i := 0; i < 20000; i++ {
		s, h := RollMonsterStats(r, 1, 1)
		if s < 3 {
			t.Fatalf("速度 = %d，下限應為 3", s)
		}
		if h < 1 {
			t.Fatalf("生命 = %d，下限應為 1", h)
		}
	}
}
