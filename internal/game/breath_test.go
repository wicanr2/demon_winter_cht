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
		{Slot: 0, Name: "龍", X: BattleCentreX, Y: BattleCentreY + 2, HP: 30, MaxHP: 30, Facing: int(North), RaceOrElement: 9},
	}
	// 三個玩家排在龍的正前方一直線上。
	for i := 0; i < 3; i++ {
		units = append(units, &Unit{
			Slot: PlayerSlotStart + i, Name: "我" + string(rune('A'+i)),
			X: BattleCentreX, Y: BattleCentreY + 1 - i, HP: 10, MaxHP: 10, IsPlayer: true,
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

// 傷害 = rnd(施放者當前HP ÷ 3)，整數除法。
//
// 這條的重點是「只看施放者剩多少血」—— 受傷的龍噴得比較弱，
// 與投入、等級、目標護甲都無關。
func TestBreathDamage(t *testing.T) {
	for _, c := range []struct{ hp, roll, want int }{
		{30, 1, 1}, {30, 10, 10},
		{29, 9, 9}, // 29/3 = 9
		{5, 1, 1},  // 5/3 = 1
		{2, 1, 0},  // 不到 3：商 0，原版會拿 0 去擲
		{0, 1, 0},
	} {
		got := BreathDamage(&fixedRolls{vals: []int{c.roll}}, c.hp)
		if got != c.want {
			t.Errorf("HP %d、擲 %d：傷害 %d，預期 %d", c.hp, c.roll, got, c.want)
		}
	}
	// 上界：不會超過 HP/3。
	r := rng.NewWithSeed(7)
	for i := 0; i < 500; i++ {
		if d := BreathDamage(r, 31); d < 1 || d > 10 {
			t.Fatalf("HP 31 的傷害 %d 落在 [1,10] 之外", d)
		}
	}
}

// 種族 → 元素：8→6、9→7、10→8、11→rnd(3)+5，12 沒有對應。
func TestBreathElement(t *testing.T) {
	for race, want := range map[int]int{8: 6, 9: 7, 10: 8} {
		got, ok := BreathElement(nil, race)
		if !ok || got != want {
			t.Errorf("種族 %d → (%d, %v)，預期 (%d, true)", race, got, ok, want)
		}
	}
	// 種族 11 三種都可能，而且只會是 6／7／8。
	seen := map[int]bool{}
	r := rng.NewWithSeed(5)
	for i := 0; i < 300; i++ {
		got, ok := BreathElement(r, 11)
		if !ok || got < 6 || got > 8 {
			t.Fatalf("種族 11 → (%d, %v)，預期落在 6–8", got, ok)
		}
		seen[got] = true
	}
	if len(seen) != 3 {
		t.Errorf("種族 11 只擲出 %v，三個值應該都會出現", seen)
	}
	// 種族 12 落在 AI 的噴吐區間（8–12）裡卻沒有對應分支，不過 MONSTER.DAT
	// 落在區間內的只有四隻龍（8/9/10/11），12 打不到。
	if _, ok := BreathElement(rng.NewWithSeed(1), 12); ok {
		t.Error("種族 12 不該有元素")
	}
}

// 否決：一個敵人都沒打到，或己方命中 × 2 > 敵方命中。
func TestBreathPlan_Veto(t *testing.T) {
	for _, c := range []struct {
		own, enemy int
		veto       bool
	}{
		{0, 0, true}, {1, 0, true},
		{0, 1, false}, {1, 1, true}, {1, 2, false}, {1, 3, false},
		{2, 3, true}, {2, 4, false},
	} {
		got := BreathPlan{Own: c.own, Enemy: c.enemy}.Veto()
		if got != c.veto {
			t.Errorf("己方 %d 敵方 %d：否決 = %v，預期 %v",
				c.own, c.enemy, got, c.veto)
		}
	}
}

// 打到自己人太多就整個放棄，一滴血都不扣。
func TestBreathe_VetoLeavesEveryoneAlone(t *testing.T) {
	b := breathBattle(t)
	dragon := b.Unit(0)
	// 塞兩個同伴進錐形裡：敵方 3、己方 2 → 2×2 = 4 > 3 → 否決。
	for i, y := range []int{BattleCentreY, BattleCentreY - 1} {
		b.units[1+i] = &Unit{Slot: 1 + i, Name: "同伴", X: BattleCentreX - 1 + i, Y: y,
			HP: 10, MaxHP: 10, Side: SideMonster}
	}
	before := map[*Unit]int{}
	for _, u := range b.Units() {
		before[u] = u.HP
	}
	if hits := b.Breathe(dragon); hits != nil {
		t.Fatalf("誤傷太多應該放棄，卻打了 %d 人", len(hits))
	}
	for u, hp := range before {
		if u.HP != hp {
			t.Errorf("%s 的 HP 從 %d 變成 %d", u.Name, hp, u.HP)
		}
	}
}

// 正常情形：錐形內三人都扣血。
func TestBreathe_DamagesEveryoneInCone(t *testing.T) {
	b := breathBattle(t)
	hits := b.Breathe(b.Unit(0))
	if len(hits) != 3 {
		t.Fatalf("打到 %d 人，預期 3", len(hits))
	}
	for _, h := range hits {
		if h.Damage < 1 || h.Damage > 10 { // 龍 30 HP → rnd(10)
			t.Errorf("%s 受到 %d 點傷害，超出 rnd(30/3) 的範圍", h.Unit.Name, h.Damage)
		}
		if !h.Killed && h.Unit.HP != 10-h.Damage {
			t.Errorf("%s 的 HP 是 %d，預期 %d", h.Unit.Name, h.Unit.HP, 10-h.Damage)
		}
	}
}

// 傷害 >= HP 就死（原版 0x1bb5 的 JG：HP > 傷害 才活）。
func TestBreathe_ExactDamageKills(t *testing.T) {
	b := breathBattle(t)
	dragon := b.Unit(0)
	dragon.HP = 3 // rnd(1) 恆為 1
	victim := b.Unit(PlayerSlotStart)
	victim.HP = 1

	hits := b.Breathe(dragon)
	if len(hits) == 0 {
		t.Fatal("沒有打到任何人")
	}
	if hits[0].Unit != victim || !hits[0].Killed {
		t.Errorf("HP 1 吃 1 點傷害應該倒下，得到 %+v", hits[0])
	}
}

// 噴吐錐形：形狀與 inBreathCone 一致、順序是由近而遠。
func TestBreathCone(t *testing.T) {
	b := breathBattle(t)
	caster := b.Units()[0]
	caster.X, caster.Y = BattleCentreX, BattleCentreY
	caster.Facing = int(East)

	cone := b.BreathCone(caster)
	if len(cone) == 0 {
		t.Fatal("錐形是空的")
	}

	// 每一格都要真的在錐形裡（跟命中判定同一個定義）。
	dx, dy := East.Delta()
	for _, c := range cone {
		if !inBreathCone(caster.X, caster.Y, dx, dy, c.X, c.Y) {
			t.Errorf("(%d,%d) 不在錐形內", c.X, c.Y)
		}
	}
	// 起點那一格排第一 —— 動畫要從嘴邊開始擴。
	if cone[0].X != caster.X || cone[0].Y != caster.Y {
		t.Errorf("第一格是 (%d,%d)，預期起點 (%d,%d)",
			cone[0].X, cone[0].Y, caster.X, caster.Y)
	}
	// 沿噴吐方向的距離不遞減。
	prev := -1
	for _, c := range cone {
		if along := (c.X - caster.X) * dx; along < prev {
			t.Fatalf("順序不是由近而遠：(%d,%d) 的距離 %d < %d", c.X, c.Y, along, prev)
		} else {
			prev = along
		}
	}
	// 沒有面向就沒有錐形。
	caster.Facing = -1
	if got := b.BreathCone(caster); got != nil {
		t.Errorf("沒有方向卻回了 %d 格", len(got))
	}
}
