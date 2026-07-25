package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 錐形的形狀：起點那一格只有正中命中，往前每深一格橫向就寬一格。
//
// 對應原版四個方向各自寫死的那段（0x8d95 起）：`|across| <= along`。
func TestBreathCone_Shape(t *testing.T) {
	// 朝北：起點 (5,5)，方向 (0,-1)。
	const ox, oy, dx, dy = 5, 5, 0, -1

	for _, c := range []struct {
		x, y int
		want bool
		why  string
	}{
		{5, 5, true, "起點本身"},
		{4, 5, false, "起點同列但偏一格：along=0 時只容得下 across=0"},
		{5, 4, true, "正前方一格"},
		{4, 4, true, "前一格、偏一格：across=1 <= along=1"},
		{6, 4, true, "前一格、另一側"},
		{3, 4, false, "前一格、偏兩格：across=2 > along=1"},
		{5, 2, true, "最遠一格（along=3）"},
		{2, 2, true, "最遠一格、偏三格：across=3 <= along=3"},
		{5, 1, false, "超過深度（along=4）"},
		{5, 6, false, "在起點後面"},
	} {
		if got := inBreathCone(ox, oy, dx, dy, c.x, c.y); got != c.want {
			t.Errorf("(%d,%d) 命中 = %v，預期 %v —— %s", c.x, c.y, got, c.want, c.why)
		}
	}
}

// 四個方向要是同一個形狀轉過去，不能有哪一邊寫歪。
//
// 原版是四段寫死展開的程式碼，正是最容易某一段抄錯的寫法；
// 這條用旋轉不變性一次涵蓋四段。
func TestBreathCone_SameShapeAllDirections(t *testing.T) {
	dirs := [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

	// 以「沿方向 a 格、橫向 c 格」描述，四個方向的命中結果應相同。
	for a := 0; a < BreathDepth+1; a++ {
		for c := -BreathDepth; c <= BreathDepth; c++ {
			var first bool
			for i, d := range dirs {
				dx, dy := d[0], d[1]
				// 垂直方向的單位向量。
				px, py := -dy, dx
				tx := 5 + dx*a + px*c
				ty := 5 + dy*a + py*c
				got := inBreathCone(5, 5, dx, dy, tx, ty)
				if i == 0 {
					first = got
					continue
				}
				if got != first {
					t.Errorf("沿 %d 橫 %d：方向 %d 得到 %v，方向 0 得到 %v",
						a, c, i, got, first)
				}
			}
		}
	}
}

func breathBattle(t *testing.T) *Battle {
	t.Helper()
	units := []*Unit{
		{Slot: 0, Name: "龍", X: 4, Y: 6, HP: 30, MaxHP: 30, Facing: int(North), RaceOrElement: 9},
	}
	// 三個玩家排在龍的正前方一直線上。
	for i := 0; i < 3; i++ {
		units = append(units, &Unit{
			Slot: PlayerSlotStart + i, Name: "我" + string(rune('A'+i)),
			X: 4, Y: 5 - i, HP: 10, MaxHP: 10, IsPlayer: true,
		})
	}
	return NewBattle(rng.NewWithSeed(1), units)
}

// 噴吐從「面前那一格」起算，所以站在龍正前方的人會被打到。
func TestBreathTargets_HitsUnitsInFront(t *testing.T) {
	b := breathBattle(t)
	hits := b.BreathTargets(b.Unit(0))

	if len(hits) != 3 {
		names := []string{}
		for _, h := range hits {
			names = append(names, h.Name)
		}
		t.Fatalf("打到 %d 人 %v，預期正前方三人都中", len(hits), names)
	}
}

// 背後的人打不到。
func TestBreathTargets_MissesBehind(t *testing.T) {
	b := breathBattle(t)
	dragon := b.Unit(0)
	dragon.Facing = int(South) // 轉身背對

	for _, u := range b.BreathTargets(dragon) {
		if u.IsPlayer {
			t.Errorf("轉身之後還打到背後的 %s", u.Name)
		}
	}
}

// 同族免疫：施放者類型 10 時，類型 4／7／10 的目標整個跳過。
func TestBreathTargets_SameElementImmunity(t *testing.T) {
	b := breathBattle(t)
	dragon := b.Unit(0)
	dragon.RaceOrElement = breathImmuneCaster

	for race, immune := range map[int]bool{4: true, 7: true, 10: true, 9: false, 0: false} {
		for i := 0; i < 3; i++ {
			b.Unit(PlayerSlotStart + i).RaceOrElement = race
		}
		hits := b.BreathTargets(dragon)
		if immune && len(hits) != 0 {
			t.Errorf("施放者類型 %d、目標類型 %d 應該免疫，卻打到 %d 人",
				breathImmuneCaster, race, len(hits))
		}
		if !immune && len(hits) == 0 {
			t.Errorf("目標類型 %d 不該免疫，卻一個都沒打到", race)
		}
	}

	// 施放者不是類型 10 時，免疫不成立。
	dragon.RaceOrElement = 9
	for i := 0; i < 3; i++ {
		b.Unit(PlayerSlotStart + i).RaceOrElement = 10
	}
	if len(b.BreathTargets(dragon)) == 0 {
		t.Error("施放者不是類型 10，免疫不該生效")
	}
}

// 死掉的單位不算在內（原版看的是 X == 0 的哨兵）。
func TestBreathTargets_SkipsDead(t *testing.T) {
	b := breathBattle(t)
	b.Kill(b.Unit(PlayerSlotStart))

	for _, u := range b.BreathTargets(b.Unit(0)) {
		if !u.Alive() {
			t.Errorf("打到已經倒下的 %s", u.Name)
		}
	}
}
