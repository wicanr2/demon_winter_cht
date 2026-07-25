package game

import (
	"sort"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 手冊附錄 C「符咒點數的最低值」全部 35 個法術。
//
// docs/spec/02-combat.md 說這 35 個值與法術表的 M 欄「只差一個 3」
// （魔法火炬，手冊記 3、表記 2）。這裡用多重集合比對把那句話釘住。
var manualMinimumSP = []int{
	// 火魔法：施火柱 1、火風暴 10、火罩 4、火焰棒 16、魔炬 3、鎔熔術 11
	1, 10, 4, 16, 3, 11,
	// 地魔法：Armor 2、解除綑綁 11、鍊術 10、Death Blade 15、銹甲 3、力量 1、劍術 2
	2, 11, 10, 15, 3, 1, 2,
	// 冰魔法：致寒 1、晶光 2、冰凍 9、電風暴 7、冰罩 3、致慢 3
	1, 2, 9, 7, 3, 3,
	// 魂魔法：笨拙 2、解毒 9、治病 1、起死回生 25、Sanctuary 3、
	//         Spirit Wrack 20、轉移 3、致弱 1、Wither Strike 15
	2, 9, 1, 25, 3, 20, 3, 1, 15,
	// 風魔法：生命氣息 5、解困 13、矗立 11、Tempest 6、行風 10、
	//         Wings 4、Wings of Victory 1
	5, 13, 11, 6, 10, 4, 1,
}

func TestSpellTable_MinimumSPMatchesManual(t *testing.T) {
	tb := loadTables(t)

	if len(manualMinimumSP) != 35 {
		t.Fatalf("手冊清單應有 35 個法術，實際 %d", len(manualMinimumSP))
	}

	var tableM []int
	for i := 0; i < tb.NumSpells(); i++ {
		s, err := tb.Spell(i)
		if err != nil {
			t.Fatalf("Spell(%d): %v", i, err)
		}
		if s.Empty() {
			continue
		}
		tableM = append(tableM, s.M)
	}

	// 多重集合相減：手冊有、表裡沒有的值。
	remain := append([]int(nil), tableM...)
	var missing []int
	for _, want := range manualMinimumSP {
		found := -1
		for i, v := range remain {
			if v == want {
				found = i
				break
			}
		}
		if found < 0 {
			missing = append(missing, want)
			continue
		}
		remain = append(remain[:found], remain[found+1:]...)
	}

	sort.Ints(missing)
	if len(missing) != 1 || missing[0] != 3 {
		t.Errorf("手冊的最低 SP 在表中缺少 %v，預期只缺一個 3（魔法火炬，手冊記 3、表記 2）", missing)
	}
}

// 復甦術：K = M = 25，正好重現手冊「25 點、25% 機率」。
func TestSpellTable_ResurrectMatchesManual(t *testing.T) {
	tb := loadTables(t)

	found := false
	for i := 0; i < tb.NumSpells(); i++ {
		s, _ := tb.Spell(i)
		if s.K == 25 && s.M == 25 {
			found = true
			// 以 25 SP 施放時成功率剛好 25%。
			if got := SpellRate(25, s); got != 25 {
				t.Errorf("以 25 SP 施放的成功率 = %d%%，預期 25%%", got)
			}
		}
	}
	if !found {
		t.Error("找不到 K = M = 25 的法術（復甦術）")
	}
}

func TestSpellTable_Bounds(t *testing.T) {
	tb := loadTables(t)
	if tb.NumSpells() != 43 {
		t.Errorf("法術表筆數 = %d，預期 43", tb.NumSpells())
	}
	if _, err := tb.Spell(-1); err == nil {
		t.Error("負索引應回傳錯誤")
	}
	if _, err := tb.Spell(43); err == nil {
		t.Error("超出範圍的索引應回傳錯誤")
	}
}

// 通式：magnitude 落在 [上限/3, 上限]，且 K 的正負決定方向。
func TestSpellMagnitude_RangeAndSign(t *testing.T) {
	r := rng.NewWithSeed(1234)

	// K=15, M=10, SP=20 → 上限 30，下限 10。
	heal := gamedata.Spell{K: 15, M: 10}
	lo, hi := 1<<30, -(1 << 30)
	for i := 0; i < 20000; i++ {
		v := SpellMagnitude(r, 20, heal)
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	if hi != 30 {
		t.Errorf("magnitude 上界 = %d，預期 30", hi)
	}
	if lo < 10 {
		t.Errorf("magnitude 下界 = %d，不應低於上限的 1/3（10）", lo)
	}

	// K 為負 → 方向相反。
	harm := gamedata.Spell{K: -15, M: 10}
	for i := 0; i < 1000; i++ {
		if v := SpellMagnitude(r, 20, harm); v > 0 {
			t.Fatalf("K 為負時 magnitude 應為負，得到 %d", v)
		}
	}
}

// 上限很小時 1/3 會是 0，任何擲值都通過，不能空轉。
func TestSpellMagnitude_SmallBoundTerminates(t *testing.T) {
	r := rng.NewWithSeed(9)
	s := gamedata.Spell{K: 1, M: 10}
	// SP 5 → 上限 = 1*5/10 = 0 → 回傳 0，不進迴圈。
	if got := SpellMagnitude(r, 5, s); got != 0 {
		t.Errorf("上限為 0 時應回傳 0，得到 %d", got)
	}
	// SP 20 → 上限 2，下限 0，任何擲值都通過。
	for i := 0; i < 1000; i++ {
		if v := SpellMagnitude(r, 20, s); v < 1 || v > 2 {
			t.Fatalf("magnitude = %d，應落在 1..2", v)
		}
	}
	// M 為 0 的空記錄不得除以零。
	if got := SpellMagnitude(r, 10, gamedata.Spell{K: 5, M: 0}); got != 0 {
		t.Errorf("M 為 0 時應回傳 0，得到 %d", got)
	}
}

// 非 HP 屬性鉗制在 [3, 255]。下限 3 對上攻略「屬性不會低於 3」。
func TestApplyTraitEffect_Clamp(t *testing.T) {
	cases := []struct{ cur, mag, want int }{
		{10, 5, 15},
		{10, -20, 3},
		{3, -1, 3},
		{250, 20, 255},
		{255, 1, 255},
	}
	for _, c := range cases {
		if got := ApplyTraitEffect(c.cur, c.mag); got != c.want {
			t.Errorf("ApplyTraitEffect(%d, %d) = %d，預期 %d", c.cur, c.mag, got, c.want)
		}
	}
}

func TestSpellRate(t *testing.T) {
	// 整數除法向零捨去。
	s := gamedata.Spell{K: 25, M: 16}
	if got := SpellRate(20, s); got != 31 { // 25*20/16 = 31.25 → 31
		t.Errorf("SpellRate = %d，預期 31", got)
	}
	if got := SpellRate(10, gamedata.Spell{K: 5, M: 0}); got != 0 {
		t.Errorf("M 為 0 時應回傳 0，得到 %d", got)
	}
}

func TestCastInstantDeath(t *testing.T) {
	r := rng.NewWithSeed(55)

	// K=100, M=1, SP=1 → rate 100，必定成功。
	sure := gamedata.Spell{K: 100, M: 1}
	tgt := &Unit{HP: 99}
	if !CastInstantDeath(r, 1, sure, tgt) {
		t.Fatal("rate 100 時應必定成功")
	}
	if tgt.HP != 0 || tgt.Status != StatusDead {
		t.Errorf("成功後應 HP 歸零且狀態為死亡，得到 HP %d 狀態 %d", tgt.HP, tgt.Status)
	}

	// rate 0 → 必定失敗，目標不受影響。
	never := gamedata.Spell{K: 0, M: 1}
	alive := &Unit{HP: 99}
	if CastInstantDeath(r, 1, never, alive) {
		t.Error("rate 0 時應必定失敗")
	}
	if alive.HP != 99 {
		t.Errorf("失敗時目標 HP 不應改變，得到 %d", alive.HP)
	}
}

// 束縛不可疊加：狀態已 >= 2 直接失敗。
func TestCastBind_NoStacking(t *testing.T) {
	r := rng.NewWithSeed(66)
	s := gamedata.Spell{School: 3, K: 1, M: 10}

	tgt := &Unit{Strength: 1, Status: StatusBindLow}
	res := CastBind(r, 100, s, tgt)
	if res.Applied || !res.AlreadyBound {
		t.Errorf("已被束縛時應直接失敗，得到 %+v", res)
	}
}

// 成功施加時，狀態欄存的是**符文系 id**，不是布林旗標 ——
// 這是解除判定能運作的前提。
func TestCastBind_StoresSchoolID(t *testing.T) {
	r := rng.NewWithSeed(77)
	// 力量 1 → resist = 4 − 大數，必為負，Roll(100) 不可能小於它 → 必定施加。
	s := gamedata.Spell{School: 4, K: 10, M: 1}

	tgt := &Unit{Strength: 1}
	res := CastBind(r, 20, s, tgt)
	if !res.Applied {
		t.Fatalf("抗性為負時應必定施加，得到 %+v", res)
	}
	if int(tgt.Status) != s.School {
		t.Errorf("狀態 = %d，應等於符文系 id %d", tgt.Status, s.School)
	}
	if tgt.BindRounds != SpellRate(20, s) {
		t.Errorf("束縛回合 = %d，預期 %d", tgt.BindRounds, SpellRate(20, s))
	}
}

// 力量高的目標抵抗成功。
func TestCastBind_Resist(t *testing.T) {
	r := rng.NewWithSeed(88)
	// 力量 30 → resist = 120 − 小數，Roll(100) 必定小於它 → 必定抵抗。
	s := gamedata.Spell{School: 2, K: 1, M: 100}

	tgt := &Unit{Strength: 30}
	res := CastBind(r, 1, s, tgt)
	if !res.Resisted {
		t.Errorf("力量 30 對弱法術應必定抵抗，得到 %+v", res)
	}
	if tgt.Status != StatusNormal {
		t.Errorf("抵抗成功時狀態不應改變，得到 %d", tgt.Status)
	}
}

// 解除束縛：力度足夠**且**系別相符才成功。
func TestCastBindRelease_NeedsMatchingSchool(t *testing.T) {
	strong := gamedata.Spell{School: 3, K: 100, M: 1} // rate 很大，力度一定夠

	// 系別相符 → 成功。
	ok := &Unit{Status: UnitStatus(3), BindRounds: 5}
	if !CastBindRelease(10, strong, ok) {
		t.Error("系別相符且力度足夠時應解除成功")
	}
	if ok.Status != StatusNormal || ok.BindRounds != 0 {
		t.Errorf("解除後應回到正常狀態，得到狀態 %d 回合 %d", ok.Status, ok.BindRounds)
	}

	// 系別不符 → 失敗。
	wrong := &Unit{Status: UnitStatus(4), BindRounds: 5}
	if CastBindRelease(10, strong, wrong) {
		t.Error("系別不符時不應解除")
	}
	if wrong.Status != UnitStatus(4) {
		t.Errorf("失敗時狀態不應改變，得到 %d", wrong.Status)
	}

	// 力度不足 → 失敗。
	weak := gamedata.Spell{School: 3, K: 1, M: 100}
	tough := &Unit{Status: UnitStatus(3), BindRounds: 99}
	if CastBindRelease(1, weak, tough) {
		t.Error("力度不足時不應解除")
	}

	// 死亡不可解。
	dead := &Unit{Status: StatusDead, BindRounds: 1}
	if CastBindRelease(10, strong, dead) {
		t.Error("死亡狀態不應被解除束縛")
	}

	// 沒被束縛也不該「解除成功」。
	normal := &Unit{Status: StatusNormal}
	if CastBindRelease(10, strong, normal) {
		t.Error("未被束縛時不應回報解除成功")
	}
}

// 枯萎是「以現值為上限重擲」，不是扣減 —— 結果不會超過原值，
// 下限鉗制在 3。
func TestCastWither_RerollsWithinCurrent(t *testing.T) {
	r := rng.NewWithSeed(101)
	sure := gamedata.Spell{K: 100, M: 1} // rate 100，必定觸發

	sawLower := false
	for i := 0; i < 2000; i++ {
		u := &Unit{Speed: 20, Strength: 18, Skill: 16}
		if !CastWither(r, 1, sure, u) {
			t.Fatal("rate 100 時應必定觸發")
		}
		if u.Speed < traitEffectFloor || u.Speed > 20 ||
			u.Strength < traitEffectFloor || u.Strength > 18 ||
			u.Skill < traitEffectFloor || u.Skill > 16 {
			t.Fatalf("重擲結果超出 [3, 原值]：速度 %d 力量 %d 技巧 %d",
				u.Speed, u.Strength, u.Skill)
		}
		if u.Speed < 20 || u.Strength < 18 || u.Skill < 16 {
			sawLower = true
		}
	}
	if !sawLower {
		t.Error("重擲應該常常低於原值")
	}
}

// K/M 只決定是否觸發，不參與屬性計算；失敗時屬性完全不動。
func TestCastWither_FailureLeavesTraitsAlone(t *testing.T) {
	r := rng.NewWithSeed(202)
	never := gamedata.Spell{K: 0, M: 1} // rate 0

	u := &Unit{Speed: 20, Strength: 18, Skill: 16}
	if CastWither(r, 1, never, u) {
		t.Fatal("rate 0 時應必定失敗")
	}
	if u.Speed != 20 || u.Strength != 18 || u.Skill != 16 {
		t.Errorf("失敗時屬性不應改變，得到 %d %d %d", u.Speed, u.Strength, u.Skill)
	}
}

// 解除術的符文系 1 是 4 的別名。
func TestCastBindRelease_SchoolOneAliasesToFour(t *testing.T) {
	s := gamedata.Spell{School: 1, K: 100, M: 1}

	// 目標被系別 4 束縛，用 school 1 的解除術應該成功（別名重映射）。
	u := &Unit{Status: UnitStatus(4), BindRounds: 3}
	if !CastBindRelease(10, s, u) {
		t.Error("符文系 1 應重映射為 4，對系別 4 的束縛可解")
	}

	// 目標被系別 1 束縛，用 school 1 的解除術反而不該成功
	// （重映射之後變成 4，與狀態 1 不符）。
	v := &Unit{Status: UnitStatus(1), BindRounds: 3}
	if CastBindRelease(10, s, v) {
		t.Error("重映射後系別為 4，不應解除狀態 1")
	}
}

// 通式效果落到正確的欄位，而且方向由 K 的正負決定。
func TestCastMagnitudeEffect_Routing(t *testing.T) {
	cases := []struct {
		name   string
		effect int
		get    func(*Unit) int
	}{
		{"生命", EffectHP, func(u *Unit) int { return u.HP }},
		{"法力", EffectSPMod, func(u *Unit) int { return u.CurrentSP }},
		{"技巧", EffectSkillMod, func(u *Unit) int { return u.Skill }},
		{"力量", EffectStrengthMod, func(u *Unit) int { return u.Strength }},
		{"速度", EffectSpeedMod, func(u *Unit) int { return u.Speed }},
		{"護甲", EffectArmorMod, func(u *Unit) int { return u.Armor }},
	}
	for _, c := range cases {
		// K 為正 = 增益。
		u := &Unit{HP: 20, MaxHP: 60, CurrentSP: 10, MaxSP: 60,
			Skill: 10, Strength: 10, Speed: 10, Armor: 10}
		before := c.get(u)
		delta, ok := CastMagnitudeEffect(rng.NewWithSeed(3), 10,
			gamedata.Spell{Effect: c.effect, K: 3, M: 1}, u)
		if !ok {
			t.Fatalf("%s 應走通式", c.name)
		}
		if delta <= 0 || c.get(u) != before+delta {
			t.Errorf("%s：K 為正應增加，delta=%d，%d → %d",
				c.name, delta, before, c.get(u))
		}
	}
}

// 屬性下限是 3，生命與法力不套那個下限 —— 套了角色會永遠死不了。
func TestCastMagnitudeEffect_FloorsDiffer(t *testing.T) {
	spell := func(effect, k int) gamedata.Spell {
		return gamedata.Spell{Effect: effect, K: k, M: 1}
	}

	u := &Unit{HP: 5, MaxHP: 60, Strength: 5, Skill: 5, Speed: 5, Armor: 5}
	CastMagnitudeEffect(rng.NewWithSeed(1), 40, spell(EffectHP, -30), u)
	if u.HP != 0 {
		t.Errorf("大量傷害後生命 = %d，預期歸零（不套屬性的下限 3）", u.HP)
	}

	u2 := &Unit{Strength: 5, HP: 10, MaxHP: 10}
	CastMagnitudeEffect(rng.NewWithSeed(1), 40, spell(EffectStrengthMod, -30), u2)
	if u2.Strength != traitEffectFloor {
		t.Errorf("大量削弱後力量 = %d，預期鉗在下限 %d", u2.Strength, traitEffectFloor)
	}
}

// 治療不能超過上限。
func TestCastMagnitudeEffect_Caps(t *testing.T) {
	u := &Unit{HP: 55, MaxHP: 60, CurrentSP: 55, MaxSP: 60}
	CastMagnitudeEffect(rng.NewWithSeed(2), 40,
		gamedata.Spell{Effect: EffectHP, K: 30, M: 1}, u)
	if u.HP != u.MaxHP {
		t.Errorf("治療後生命 = %d，預期封在上限 %d", u.HP, u.MaxHP)
	}

	CastMagnitudeEffect(rng.NewWithSeed(2), 40,
		gamedata.Spell{Effect: EffectSPMod, K: 30, M: 1}, u)
	if u.CurrentSP != u.MaxSP {
		t.Errorf("回復後法力 = %d，預期封在上限 %d", u.CurrentSP, u.MaxSP)
	}
}

// 不走通式的 effect_type 要明確回報，不能靜默當成有效果。
func TestCastMagnitudeEffect_RejectsSpecialTypes(t *testing.T) {
	u := &Unit{HP: 20, MaxHP: 20}
	for _, effect := range []int{
		EffectAOE, EffectInstantDeath, EffectBindApply,
		EffectBindRelease, EffectWither,
	} {
		if _, ok := CastMagnitudeEffect(rng.NewWithSeed(1), 10,
			gamedata.Spell{Effect: effect, K: 5, M: 1}, u); ok {
			t.Errorf("effect_type %d 有自己的判定，不該走通式", effect)
		}
	}
}

func TestCastMagnitudeEffect_NilTarget(t *testing.T) {
	if _, ok := CastMagnitudeEffect(rng.NewWithSeed(1), 10,
		gamedata.Spell{Effect: EffectHP, K: 3, M: 1}, nil); ok {
		t.Error("沒有目標時應回傳 ok=false")
	}
}
