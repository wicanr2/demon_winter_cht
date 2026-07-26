package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 八個用到 E 的公式。全部整數除法。
func TestEconomy_PricesDerivedFromE(t *testing.T) {
	e := Economy{E: 37, ShipBase: 12}

	cases := []struct {
		name string
		got  int
		want int
	}{
		{"市集買價 100×E/10", e.BuyPrice(100), 100 * 37 / 10},
		{"鑑定 E×5", e.IdentifyPrice(), 37 * 5},
		{"糧食單價 E/5", e.RationUnitPrice(), 37 / 5},
		{"修船 (75−50)×E/2", e.RepairPrice(50), (75 - 50) * 37 / 2},
		{"治療 E/5", e.HealRate(), 37 / 5},
		{"解毒 E×4", e.UnpoisonRate(), 37 * 4},
		{"解束縛 47E/5", e.UnbindRate(), 47 * 37 / 5},
		{"復活 E×10", e.ResurrectRate(), 37 * 10},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s：得到 %d，預期 %d", c.name, c.got, c.want)
		}
	}
}

// 兩個例外不看 E：買船與賣價。
func TestEconomy_ExceptionsIgnoreE(t *testing.T) {
	a := Economy{E: 10, ShipBase: 12}
	b := Economy{E: 90, ShipBase: 12}

	if a.ShipPrice() != b.ShipPrice() {
		t.Errorf("買船價不應隨 E 變動，得到 %d vs %d", a.ShipPrice(), b.ShipPrice())
	}
	if a.ShipPrice() != 120 {
		t.Errorf("買船價 = %d，預期 ShipBase×10 = 120", a.ShipPrice())
	}

	if a.SellPrice(100) != b.SellPrice(100) {
		t.Error("賣價不應隨 E 變動")
	}
	if a.SellPrice(101) != 50 {
		t.Errorf("賣價 = %d，預期 101/2 = 50（整數除法）", a.SellPrice(101))
	}

	// 買價套 E、賣價不套 —— 原版的不對稱設計。
	if a.BuyPrice(100) == a.SellPrice(100)*2 && b.BuyPrice(100) == b.SellPrice(100)*2 {
		t.Error("買價與賣價不應是固定倍數關係（買價套 E、賣價不套）")
	}
}

// 糧食單價下限 1：E 很小時不能變成 0。
func TestEconomy_RationPriceFloor(t *testing.T) {
	for e := 0; e < 5; e++ {
		if got := (Economy{E: e}).RationUnitPrice(); got != 1 {
			t.Errorf("E = %d 的糧食單價 = %d，下限應為 1", e, got)
		}
	}
	if got := (Economy{E: 20}).RationUnitPrice(); got != 4 {
		t.Errorf("E = 20 的糧食單價 = %d，預期 4", got)
	}
}

// 滿血的船修理費為 0，呼叫端該顯示「你的船看起來很好」。
func TestEconomy_RepairFullHull(t *testing.T) {
	e := Economy{E: 50}
	if got := e.RepairPrice(ShipMaxHull); got != 0 {
		t.Errorf("滿血修理費 = %d，預期 0", got)
	}
	if got := e.RepairPrice(ShipMaxHull + 10); got != 0 {
		t.Errorf("超過滿值時修理費 = %d，預期 0", got)
	}
}

// 治療所依狀態選服務：死亡 > 束縛 > 中毒 > 傷勢。
func TestEconomy_HealerQuotePriority(t *testing.T) {
	e := Economy{E: 30}

	cases := []struct {
		name    string
		status  UnitStatus
		level   int
		bind    int
		damage  int
		service HealerService
		cost    int
	}{
		{"死亡", StatusDead, 4, 3, 10, HealerResurrect, 4 * 30 * 10},
		// 解束縛乘的是**束縛等級**（3），不是角色等級（4）。
		{"束縛", StatusBindLow, 4, 3, 10, HealerUnbind, 3 * (47 * 30 / 5)},
		{"中毒", StatusPoison, 4, 3, 10, HealerUnpoison, 30 * 4},
		{"受傷", StatusNormal, 4, 3, 10, HealerHeal, 10 * (30 / 5)},
		{"健康", StatusNormal, 4, 3, 0, HealerNone, 0},
	}
	for _, c := range cases {
		svc, cost := e.HealerQuote(c.status, c.level, c.bind, c.damage)
		if svc != c.service {
			t.Errorf("%s：服務 = %d，預期 %d", c.name, svc, c.service)
		}
		if cost != c.cost {
			t.Errorf("%s：費用 = %d，預期 %d", c.name, cost, c.cost)
		}
	}
}

// 學院費用表：points 1→10 應為 30、70、120、180、250、330、420、520、630、750。
func TestCollegeGoldCost_MatchesTable(t *testing.T) {
	want := []int{30, 70, 120, 180, 250, 330, 420, 520, 630, 750}
	for i, w := range want {
		points := i + 1
		if got := CollegeGoldCost(points); got != w {
			t.Errorf("points %d 的費用 = %d，預期 %d", points, got, w)
		}
	}
}

// 學院費用要能用真實的技能學費表算出來，值域合理。
func TestCollegeGoldCost_WithRealSkillCosts(t *testing.T) {
	tb := loadTables(t)

	for s := 0; s < gamedata.NumSkills; s++ {
		for c := 0; c < gamedata.NumClasses; c++ {
			points, err := tb.SkillCost(gamedata.SkillID(s), gamedata.Class(c))
			if err != nil {
				t.Fatalf("SkillCost: %v", err)
			}
			cost := CollegeGoldCost(points)
			if cost < 30 || cost > 750 {
				t.Errorf("技能 %d 職業 %d：points %d → 費用 %d，超出 30–750",
					s, c, points, cost)
			}
		}
	}
}

func TestTempleDonationAndCap(t *testing.T) {
	if got := TempleDonation(100, 50); got != 150 {
		t.Errorf("捐獻應 1:1 轉經驗，得到 %d，預期 150", got)
	}
	if got := TempleDonation(ValueCap-10, 100); got != ValueCap {
		t.Errorf("經驗應封頂 0x00FFFFFF，得到 %d", got)
	}
	if got := CapValue(-5); got != 0 {
		t.Errorf("負值應鉗為 0，得到 %d", got)
	}
}

func TestPrayCost(t *testing.T) {
	if got := PrayCost(7); got != 350 {
		t.Errorf("等級 7 的祈禱費用 = %d，預期 350", got)
	}
}

// 議價初值：隊伍有人會說服 → 0，否則 1。
func TestNewHaggleStates(t *testing.T) {
	withSkill := NewHaggleStates(true)
	if len(withSkill) != 30 {
		t.Fatalf("應有 30 件商品的狀態，得到 %d", len(withSkill))
	}
	for i, s := range withSkill {
		if s != 0 {
			t.Fatalf("有說服技能時初值應為 0，商品 %d 得到 %d", i, s)
		}
	}
	for i, s := range NewHaggleStates(false) {
		if s != 1 {
			t.Fatalf("無說服技能時初值應為 1，商品 %d 得到 %d", i, s)
		}
	}
}

func TestPartyHasPersuasion(t *testing.T) {
	var a, b Character
	if PartyHasPersuasion([]Character{a, b}) {
		t.Error("沒人學說服時應回傳 false")
	}
	b.Skills[gamedata.SkillPersuasion] = true
	if !PartyHasPersuasion([]Character{a, b}) {
		t.Error("有人學說服時應回傳 true")
	}
}

// s = 0 時必定議價成功（兩個門檻都是 0）。
func TestHaggle_ZeroStateAlwaysSucceeds(t *testing.T) {
	r := rng.NewWithSeed(1)
	for i := 0; i < 1000; i++ {
		out, next := Haggle(r, 0)
		if out != HaggleSuccess {
			t.Fatalf("s = 0 應必定成功，得到 %d", out)
		}
		if next != 1 {
			t.Fatalf("成功後 s 應 +1，得到 %d", next)
		}
	}
}

// 「商人不為所動」之後是死路：s >= 100 讓下一次議價必定觸怒對方。
//
// 這是原版的關鍵設計，不是 bug —— s×10 >= 1000 恆大於 Roll(100) 的上限。
func TestHaggle_UnmovedIsADeadEnd(t *testing.T) {
	r := rng.NewWithSeed(2)

	// 直接構造「不為所動」之後的狀態。
	s := HaggleState(100)
	for i := 0; i < 500; i++ {
		out, next := Haggle(r, s)
		if out != HaggleOffended {
			t.Fatalf("s = %d 時應必定觸怒對方，得到 %d", s, out)
		}
		if !next.Refused() {
			t.Fatalf("觸怒後應永久拒賣，得到 s = %d", next)
		}
	}
}

// 議價的機率曲線。
//
// 規格的風險曲線表寫 s=2 時「觸怒 20%、談不攏 30%」，那是**名目值**：
// Roll(100) 回傳 1..100，所以 `Roll(100) < 20` 實際命中 1..19 = 19%。
// 兩個門檻串起來的實際分佈是 19% / 81%×29% / 81%×71%。
func TestHaggle_ProbabilityCurve(t *testing.T) {
	r := rng.NewWithSeed(31337)

	const iters = 200000
	var offended, unmoved, success int
	for i := 0; i < iters; i++ {
		switch out, _ := Haggle(r, 2); out {
		case HaggleOffended:
			offended++
		case HaggleUnmoved:
			unmoved++
		default:
			success++
		}
	}

	check := func(name string, n int, want float64) {
		got := float64(n) / float64(iters)
		if got < want-0.01 || got > want+0.01 {
			t.Errorf("%s = %.4f，預期約 %.2f", name, got, want)
		}
	}
	check("觸怒", offended, 0.19)
	check("不為所動", unmoved, 0.81*0.29)
	check("成功", success, 0.81*0.71)
}

// 每次成功議價打掉標價的 6%，下限 2 金幣。
func TestHagglePrice(t *testing.T) {
	cases := []struct {
		list int
		s    HaggleState
		want int
	}{
		{100, 0, 100},
		{100, 1, 94},
		{100, 2, 88},
		{100, 5, 70},
		{100, -1, 100}, // −1 視為 0
		{100, 101, 94}, // > 99 先減 100
		{10, 20, 2},    // 折到見底，下限 2
		{1, 0, 2},      // 下限 2 是無條件的，標價比它低也會被抬到 2
	}
	for _, c := range cases {
		if got := HagglePrice(c.list, c.s); got != c.want {
			t.Errorf("HagglePrice(%d, %d) = %d，預期 %d", c.list, c.s, got, c.want)
		}
	}
}
