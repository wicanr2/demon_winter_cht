package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 武器骰表索引 1–8 必須與手冊附錄 E 的最大傷害逐項吻合。
func TestWeaponDamageDice_MatchesManual(t *testing.T) {
	want := map[int]int{
		1: 3,  // 匕首 1-3
		2: 4,  // 小斧 1-4
		3: 6,  // 短劍 1-6
		4: 6,  // 矛鎚 1-6
		5: 7,  // 晨星 1-7
		6: 8,  // 寬刀 1-8
		7: 10, // 戰斧 1-10
		8: 12, // 雙手刃 1-12
	}
	for idx, w := range want {
		if got := WeaponDamageDie(idx); got != w {
			t.Errorf("武器索引 %d：骰面 %d，手冊最大傷害 %d", idx, got, w)
		}
	}

	// 表共 14 項，索引 0 是徒手、9–13 是天生攻擊。
	if got := WeaponDamageDie(0); got != 2 {
		t.Errorf("徒手（索引 0）骰面 = %d，預期 2", got)
	}
	if got := WeaponDamageDie(13); got != 3 {
		t.Errorf("索引 13 骰面 = %d，預期 3", got)
	}
	if got := WeaponDamageDie(14); got != 0 {
		t.Errorf("超出表範圍應回傳 0，得到 %d", got)
	}
	// 負數是毒武器，取絕對值後查表。
	if WeaponDamageDie(-8) != WeaponDamageDie(8) {
		t.Error("負的武器索引（毒武器）應取絕對值查表")
	}
}

func TestStyleFor(t *testing.T) {
	has := func(set ...gamedata.SkillID) func(gamedata.SkillID) bool {
		m := map[gamedata.SkillID]bool{}
		for _, s := range set {
			m[s] = true
		}
		return func(s gamedata.SkillID) bool { return m[s] }
	}

	cases := []struct {
		name   string
		weapon int
		skills func(gamedata.SkillID) bool
		want   CombatStyle
	}{
		{"徒手 + 空手道", 0, has(gamedata.SkillKarate), StyleKarate},
		{"徒手 + 功夫", 0, has(gamedata.SkillKungFu), StyleKungFu},
		{"徒手 + 兩者", 0, has(gamedata.SkillKarate, gamedata.SkillKungFu), StyleBoth},
		{"徒手無技能", 0, has(), StyleNone},
		{"短劍 + 劍擊", 3, has(gamedata.SkillFencing), StyleFencing},
		{"寬刀 + 劍擊", 6, has(gamedata.SkillFencing), StyleFencing},
		{"雙手刃 + 劍擊", 8, has(gamedata.SkillFencing), StyleFencing},
		{"矛鎚 + 劍擊", 4, has(gamedata.SkillFencing), StyleNone},
		{"戰斧 + 劍擊", 7, has(gamedata.SkillFencing), StyleNone},
		{"短劍無劍擊", 3, has(), StyleNone},
		{"持劍但只有空手道", 3, has(gamedata.SkillKarate), StyleNone},
	}
	for _, c := range cases {
		if got := StyleFor(c.weapon, c.skills); got != c.want {
			t.Errorf("%s：得到 0x%02x，預期 0x%02x", c.name, got, c.want)
		}
	}
}

// 行動順序：三個排除條件 + 速度降冪 + 同速維持槽位順序。
func TestTurnOrder_ExclusionsAndStability(t *testing.T) {
	units := []*Unit{
		{Slot: 0, X: 5, Speed: 10},                        // 正常
		{Slot: 1, X: 5, Speed: 20},                        // 最快
		{Slot: 2, X: 0, Speed: 99},                        // X=0 排除
		{Slot: 3, X: 5, Speed: 30, Status: StatusBindLow}, // 束縛排除
		{Slot: 4, X: 5, Speed: 40, Status: StatusDead},    // 死亡排除
		{Slot: 5, X: 5, Speed: 10},                        // 與槽 0 同速
		{Slot: 6, X: 5, Speed: 50, StatusCount: -1},       // 暈眩排除
		{Slot: 7, X: 5, Speed: 15, Status: StatusPoison},  // 中毒仍可行動
	}

	order := TurnOrder(units)

	want := []int{1, 7, 0, 5} // 20, 15, 10(槽0), 10(槽5)
	if len(order) != len(want) {
		t.Fatalf("行動順序 = %v，預期 %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("行動順序 = %v，預期 %v", order, want)
		}
	}
}

// 被暈眩的單位這回合不能動，但計數每回合 +1，數回合後恢復。
func TestTurnOrder_StunCountsUpEachRound(t *testing.T) {
	u := &Unit{Slot: 0, X: 5, Speed: 10, StatusCount: -2}
	units := []*Unit{u}

	if got := TurnOrder(units); len(got) != 0 {
		t.Fatalf("第 1 回合應被排除，得到 %v", got)
	}
	if u.StatusCount != -1 {
		t.Errorf("計數應 +1 到 −1，得到 %d", u.StatusCount)
	}

	if got := TurnOrder(units); len(got) != 0 {
		t.Fatalf("第 2 回合仍應被排除，得到 %v", got)
	}
	if u.StatusCount != 0 {
		t.Errorf("計數應 +1 到 0，得到 %d", u.StatusCount)
	}

	if got := TurnOrder(units); len(got) != 1 {
		t.Fatalf("第 3 回合應可行動，得到 %v", got)
	}
}

// 傷害 < 1 時整條扣血路徑被跳過，不是鉗制為 0 再套用。
// 差別在於後者仍會走命中後的連鎖判定（中毒、暈眩），原版是直接中止。
func TestAttack_NoEffectSkipsEntireChain(t *testing.T) {
	r := rng.NewWithSeed(1)

	// 技巧 25 → 命中率 100，必中。武器 1（骰面 3）、力量 7（修正 0）、
	// 目標護甲 99 → 傷害必定 < 1。武器索引設負數（附毒）驗證毒也不會生效。
	atk := &Unit{Skill: 25, Strength: 7, WeaponIndex: -1, IsPlayer: true}
	tgt := &Unit{HP: 50, Armor: 99}

	noEffect := 0
	for i := 0; i < 200; i++ {
		res := Attack(r, atk, tgt, 0)
		if !res.Hit {
			continue
		}
		if !res.NoEffect {
			continue
		}
		noEffect++
		if res.Poisoned || res.Stunned || res.Killed {
			t.Fatal("傷害 < 1 時不應觸發中毒／暈眩／死亡")
		}
	}
	if noEffect == 0 {
		t.Fatal("這個情境應該一直落在「無效果」分支")
	}
	if tgt.HP != 50 {
		t.Errorf("目標 HP = %d，傷害 < 1 時不應改變（預期 50）", tgt.HP)
	}
	if tgt.Status != StatusNormal {
		t.Errorf("目標狀態 = %d，不應被附毒", tgt.Status)
	}
}

// 爆擊門檻：一般 10%、狂暴 25%、狂暴 + 鬥劍再多 8%。
func TestAttack_CriticalThresholds(t *testing.T) {
	measure := func(u *Unit) float64 {
		r := rng.NewWithSeed(4242)
		const iters = 200000
		crit := 0
		for i := 0; i < iters; i++ {
			tgt := &Unit{HP: 1 << 20, Armor: 0}
			res := Attack(r, u, tgt, 0)
			if res.Hit && res.Critical {
				crit++
			}
		}
		return float64(crit) / float64(iters)
	}

	base := &Unit{Skill: 25, Strength: 7, WeaponIndex: 1, IsPlayer: true}
	if got := measure(base); got < 0.09 || got > 0.11 {
		t.Errorf("一般爆擊率 = %.4f，預期約 0.10", got)
	}

	berserk := &Unit{Skill: 25, Strength: 7, WeaponIndex: 1, IsPlayer: true, Berserking: true}
	if got := measure(berserk); got < 0.24 || got > 0.26 {
		t.Errorf("狂暴爆擊率 = %.4f，預期約 0.25", got)
	}

	// 鬥劍再 −8 門檻 → 門檻 75−8 = 67 → 33%。
	both := &Unit{Skill: 25, Strength: 7, WeaponIndex: 3, IsPlayer: true,
		Berserking: true, Style: StyleFencing}
	if got := measure(both); got < 0.32 || got > 0.34 {
		t.Errorf("狂暴＋鬥劍爆擊率 = %.4f，預期約 0.33", got)
	}

	// 怪物不吃這些加成，一律 10%。
	monster := &Unit{Skill: 25, Strength: 7, WeaponIndex: 1, Berserking: true, Style: StyleFencing}
	if got := measure(monster); got < 0.09 || got > 0.11 {
		t.Errorf("怪物爆擊率 = %.4f，預期約 0.10（不吃玩家加成）", got)
	}
}

// 索引 13 的天生攻擊不加力量修正。
func TestRawDamage_NativeAttackIgnoresStrengthModifier(t *testing.T) {
	r := rng.NewWithSeed(77)
	atk := &Unit{Strength: 20, WeaponIndex: 13}
	tgt := &Unit{}

	// Roll(20/2) = Roll(10) → 1..10，不會有 (20−7)/2 = 6 的加成。
	for i := 0; i < 5000; i++ {
		d := rawDamage(r, atk, tgt)
		if d < 1 || d > 10 {
			t.Fatalf("天生攻擊傷害 = %d，應落在 1..10（不加力量修正）", d)
		}
	}
}

// 空手道改用技巧算傷害，且只對玩家生效。
func TestRawDamage_KarateUsesSkill(t *testing.T) {
	r := rng.NewWithSeed(88)

	player := &Unit{Skill: 20, Strength: 7, WeaponIndex: 0, IsPlayer: true, Style: StyleKarate}
	tgt := &Unit{}
	maxSeen := 0
	for i := 0; i < 20000; i++ {
		if d := rawDamage(r, player, tgt); d > maxSeen {
			maxSeen = d
		}
	}
	// Roll(|20−5|) = Roll(15) → 上界 15，超過雙手劍的 12。
	if maxSeen != 15 {
		t.Errorf("空手道傷害上界 = %d，預期 15", maxSeen)
	}

	// 同樣設定但不是玩家 → 走一般徒手骰表（索引 0，骰面 2）。
	monster := &Unit{Skill: 20, Strength: 7, WeaponIndex: 0, Style: StyleKarate}
	maxSeen = 0
	for i := 0; i < 20000; i++ {
		if d := rawDamage(r, monster, tgt); d > maxSeen {
			maxSeen = d
		}
	}
	if maxSeen != 2 {
		t.Errorf("怪物徒手傷害上界 = %d，預期 2（骰表索引 0）", maxSeen)
	}
}

// 對種族 7／10 的目標，武器特效再加一次。
func TestRawDamage_BonusVersusSpecificRaces(t *testing.T) {
	r := rng.NewWithSeed(99)
	atk := &Unit{Skill: 10, Strength: 7, WeaponIndex: 1, IsPlayer: true, WeaponEffect: 5}

	minAgainst := func(race int) int {
		tgt := &Unit{RaceOrElement: race}
		lo := 1 << 30
		for i := 0; i < 5000; i++ {
			if d := rawDamage(r, atk, tgt); d < lo {
				lo = d
			}
		}
		return lo
	}

	normal := minAgainst(0)
	special := minAgainst(7)
	if special-normal != 5 {
		t.Errorf("對種族 7 的傷害下界差 = %d，預期比一般多一次武器特效 5", special-normal)
	}
	if minAgainst(10) != special {
		t.Error("種族 10 應與種族 7 同樣觸發額外特效")
	}
}

func TestTurnUndead(t *testing.T) {
	r := rng.NewWithSeed(11)

	// 智力 20、等級 5 → (18×15 + 18)/5 = 288/5 = 57（整數除法）。
	caster := &Unit{Intellect: 20, Level: 5}
	hits := 0
	const iters = 100000
	for i := 0; i < iters; i++ {
		tgt := &Unit{HP: 10}
		if TurnUndead(r, caster, tgt) {
			hits++
			if tgt.HP != 0 {
				t.Fatal("成功時目標 HP 應歸零")
			}
		}
	}
	got := float64(hits) / float64(iters)
	if got < 0.55 || got > 0.59 {
		t.Errorf("驅散成功率 = %.4f，預期約 0.57", got)
	}
}

func TestDodge(t *testing.T) {
	u := &Unit{}
	if got := Dodge(u, 9); got != 3 {
		t.Errorf("行動點 9 的閃避計數 = %d，預期 3", got)
	}
	if u.StatusCount != 3 {
		t.Errorf("計數應寫回單位，得到 %d", u.StatusCount)
	}
	if got := DodgeHitModifier(3); got != -12 {
		t.Errorf("閃避 3 的命中修正 = %d，預期 −12", got)
	}
}

// 祈禱成功後 chance 永久 −5，失敗不變。
func TestPray_ChanceDecaysOnSuccess(t *testing.T) {
	r := rng.NewWithSeed(21)

	chance := 20
	successes := 0
	for i := 0; i < 200; i++ {
		ok, next := Pray(r, chance)
		if ok {
			successes++
			if next != chance-5 {
				t.Fatalf("成功後 chance = %d，預期 %d", next, chance-5)
			}
		} else if next != chance {
			t.Fatalf("失敗後 chance = %d，不應改變（預期 %d）", next, chance)
		}
		chance = next
		if chance <= 0 {
			break
		}
	}
	if successes == 0 {
		t.Fatal("初始 20% 跑 200 次應該至少成功一次")
	}
}

func TestLeech(t *testing.T) {
	r := rng.NewWithSeed(31)

	// 智力 50 → 成功率 100%，必成功。
	caster := &Unit{Intellect: 50}
	ok, left := Leech(r, caster, 21)
	if !ok {
		t.Fatal("成功率 100% 時應必定成功")
	}
	// 損失當前 SP 的一半（整數除法）：21 − 10 = 11。
	if left != 11 {
		t.Errorf("剩餘 SP = %d，預期 11", left)
	}

	// 智力 0 → 成功率 0，必失敗且 SP 不變。
	weak := &Unit{Intellect: 0}
	if ok, left := Leech(r, weak, 21); ok || left != 21 {
		t.Errorf("成功率 0 時應失敗且 SP 不變，得到 ok=%v left=%d", ok, left)
	}
}
