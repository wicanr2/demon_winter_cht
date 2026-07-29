package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// E = 30 讓四個費率都是整數：治療 6／解毒 120／解束縛 282／復活 300。
func testEconomy() Economy { return Economy{E: 30} }

func hurtCharacter() *Character {
	return &Character{Name: "甲", Level: 4, MaxHP: 20, CurrentHP: 12, MaxSP: 10}
}

// 治療：補滿血、扣對錢。
func TestHeal_Wounds(t *testing.T) {
	c := hurtCharacter()
	svc, res := Heal(testEconomy(), c, 100)
	if svc != HealerHeal {
		t.Fatalf("服務項目 %d，預期治療", svc)
	}
	// 傷 8 點 × 費率 6 = 48
	if !res.OK || res.Cost != 48 || res.Gold != 52 {
		t.Fatalf("結果 %+v", res)
	}
	if c.CurrentHP != c.MaxHP {
		t.Errorf("治療後 HP %d，應該補滿到 %d", c.CurrentHP, c.MaxHP)
	}
}

// 復活只回 1 點生命，不是滿血 —— 這是原版明講的，也是手冊寫的。
func TestHeal_ResurrectGivesOneHP(t *testing.T) {
	c := hurtCharacter()
	c.Status = scenario.StatusDead
	c.CurrentHP = 0
	c.BindLevel = 3

	svc, res := Heal(testEconomy(), c, 5000)
	if svc != HealerResurrect || !res.OK {
		t.Fatalf("服務 %d 結果 %+v", svc, res)
	}
	// 等級 4 × 復活費率 300
	if res.Cost != 1200 {
		t.Errorf("復活費用 %d，預期 1200", res.Cost)
	}
	if c.CurrentHP != 1 {
		t.Errorf("復活後 HP %d，預期 1（不是滿血）", c.CurrentHP)
	}
	if c.Status != scenario.StatusNormal {
		t.Errorf("復活後狀態 %d，預期正常", c.Status)
	}
	if c.BindLevel != 0 {
		t.Errorf("復活應該一併清掉束縛等級，得到 %d", c.BindLevel)
	}
}

// 解束縛的費用乘的是**束縛等級**，不是角色等級。
func TestHeal_UnbindUsesBindLevel(t *testing.T) {
	c := hurtCharacter()
	c.Status = scenario.StatusBound1
	c.BindLevel = 2

	svc, res := Heal(testEconomy(), c, 5000)
	if svc != HealerUnbind {
		t.Fatalf("服務項目 %d，預期解束縛", svc)
	}
	if want := 2 * (47 * 30 / 5); res.Cost != want {
		t.Errorf("費用 %d，預期 %d（束縛等級 2 × 費率）", res.Cost, want)
	}
	if c.Status != scenario.StatusNormal {
		t.Error("解束縛後狀態應該回正常")
	}
	// 解束縛不補血。
	if c.CurrentHP != 12 {
		t.Errorf("解束縛不該補血，HP 變成 %d", c.CurrentHP)
	}
}

func TestHeal_NotEnoughGold(t *testing.T) {
	c := hurtCharacter()
	_, res := Heal(testEconomy(), c, 10)
	if res.OK || res.Gold != 10 {
		t.Errorf("錢不夠卻治好了：%+v", res)
	}
	if c.CurrentHP != 12 {
		t.Error("沒付錢卻補了血")
	}
}

func TestHeal_HealthyNeedsNothing(t *testing.T) {
	c := hurtCharacter()
	c.CurrentHP = c.MaxHP
	svc, res := Heal(testEconomy(), c, 100)
	if svc != HealerNone || res.OK || res.Gold != 100 {
		t.Errorf("健康的人不該被收錢：服務 %d 結果 %+v", svc, res)
	}
}

func TestHeal_ReasonCarriesNameAsData(t *testing.T) {
	c := hurtCharacter()
	c.Name, c.CurrentHP = "冒險者", c.MaxHP
	_, res := Heal(Economy{}, c, 0)
	if res.Reason != "reason.heal.not_needed" ||
		len(res.ReasonArgs) != 1 || res.ReasonArgs[0] != "冒險者" {
		t.Fatalf("治療原因應拆成 key＋名稱參數，得到 %#v／%#v",
			res.Reason, res.ReasonArgs)
	}
}

// 買糧：份數範圍、扣款、上限。
func TestBuyRations(t *testing.T) {
	e := testEconomy() // 單價 = 30/5 = 6
	res := BuyRations(e, 100, 8, 10)
	if !res.OK || res.Cost != 60 || res.Gold != 40 {
		t.Fatalf("買 10 份：%+v", res)
	}

	if res := BuyRations(e, 100, 8, 0); res.OK {
		t.Error("買 0 份不該成立")
	}
	if res := BuyRations(e, 100, 8, MaxRations+1); res.OK {
		t.Errorf("超過一次上限 %d 不該成立", MaxRations)
	}
	if res := BuyRations(e, 10, 8, 10); res.OK {
		t.Error("錢不夠卻買成了")
	}
	// 糧食是一個 byte，帶不了超過 255 份。
	if res := BuyRations(e, 10000, 250, 20); res.OK {
		t.Error("超過 255 份的上限卻買成了")
	}
}

// 捐獻：1 金換 1 點經驗，沒有倍率。
func TestDonate(t *testing.T) {
	c := hurtCharacter()
	c.Experience = 500

	res := Donate(c, 1000, 250)
	if !res.OK || res.Gold != 750 {
		t.Fatalf("捐 250：%+v", res)
	}
	if c.Experience != 750 {
		t.Errorf("經驗值 %d，預期 750", c.Experience)
	}

	if res := Donate(c, 100, 200); res.OK {
		t.Error("捐超過身上的錢卻成功了")
	}
	if res := Donate(c, 100, 0); res.OK {
		t.Error("捐 0 不該成立")
	}
}

// 捐獻封頂在 0x00FFFFFF —— 與經驗值本身同一條規則。
func TestDonate_CapsExperience(t *testing.T) {
	c := hurtCharacter()
	c.Experience = ValueCap - 10
	Donate(c, ValueCap, 1000)
	if c.Experience != ValueCap {
		t.Errorf("經驗值 %d，應鉗在 %d", c.Experience, ValueCap)
	}
}

// devotee 造一名已改宗的信徒：有那個教派的技能，信的也是同一位神。
func devotee(deity int) *Character {
	c := hurtCharacter()
	c.Deity = deity
	c.Skills[DeityOrder(deity)] = true
	return c
}

// 祈禱：把成功率補回 20，費用是等級 × 50。
func TestPrayAtTemple(t *testing.T) {
	c := devotee(3)
	c.PrayChance = 5

	res := PrayAtTemple(c, 1000, 3)
	if !res.OK || res.Cost != 200 || res.Gold != 800 {
		t.Fatalf("4 級祈禱：%+v", res)
	}
	if c.PrayChance != FavorMax {
		t.Errorf("成功率 %d，預期 %d", c.PrayChance, FavorMax)
	}
}

func TestPrayAtTemple_WrongDeity(t *testing.T) {
	c := devotee(3)
	c.PrayChance = 5

	if res := PrayAtTemple(c, 1000, 7); res.OK {
		t.Error("信別的神卻在這裡祈禱成功了")
	}
	if c.PrayChance != 5 {
		t.Error("被拒絕卻改了成功率")
	}
}

// **沒有信仰的人祈禱不了。** 這一條一度寫反（以為原版不擋）——
// `278d:0d9a` 的 `cmpb $0, [+0xc8+教派] / je` 是「沒有那個教派的技能就跳去拒絕」。
func TestPrayAtTemple_NeedsConversionFirst(t *testing.T) {
	c := hurtCharacter()
	c.Deity = 0

	if res := PrayAtTemple(c, 1000, 7); res.OK {
		t.Errorf("還沒改宗的人不該祈禱得了：%+v", res)
	}
	// 有神祇編號但沒有教派技能 —— 一樣不行。
	c.Deity = 7
	if res := PrayAtTemple(c, 1000, 7); res.OK {
		t.Error("沒有教派技能不該祈禱得了")
	}
}

func TestPrayAtTemple_AlreadyFull(t *testing.T) {
	c := devotee(3)
	c.PrayChance = FavorMax
	if res := PrayAtTemple(c, 1000, 3); res.OK || res.Gold != 1000 {
		t.Errorf("成功率已滿卻收了錢：%+v", res)
	}
}

func TestPrayAtTemple_NotEnoughGold(t *testing.T) {
	c := devotee(3)
	c.PrayChance = 0
	if res := PrayAtTemple(c, 10, 3); res.OK || res.Cost != 200 {
		t.Errorf("錢不夠卻祈禱成功：%+v", res)
	}
}

// 升級門檻：手冊的 20 級表。
func TestExpForNextLevel(t *testing.T) {
	for _, c := range []struct{ level, want int }{
		{1, 300}, {5, 2800}, {10, 37700}, {20, 12752200},
		{21, 0}, // 已達頂級
		{0, 0}, {-1, 0},
	} {
		if got := ExpForNextLevel(c.level); got != c.want {
			t.Errorf("%d 級的門檻 = %d，預期 %d", c.level, got, c.want)
		}
	}
}

func TestCanLevelUp(t *testing.T) {
	c := hurtCharacter()
	c.Level, c.Experience = 1, 299
	if ok, short := c.CanLevelUp(); ok || short != 1 {
		t.Errorf("差 1 點卻回報可升級：ok=%v short=%d", ok, short)
	}
	c.Experience = 300
	if ok, _ := c.CanLevelUp(); !ok {
		t.Error("剛好達門檻應該可以升級")
	}
	c.Level = MaxLevel
	if ok, short := c.CanLevelUp(); ok || short != 0 {
		t.Errorf("最高等級應該回 (false, 0)，得到 (%v, %d)", ok, short)
	}
}

// 改宗：奇數神歸薩滿、偶數神歸司祭。
//
// 這是 `0x10 − (神祇編號 mod 2)` 的直接後果 —— 索引永遠是 15 或 16，
// 不會像 `docs/re/19` 擔心的那樣掉到「召喚」以下的格子。
func TestDeityOrder(t *testing.T) {
	for deity := DeityMin; deity <= DeityMax; deity++ {
		got := DeityOrder(deity)
		want := SkillPriesthood
		if deity%2 == 1 {
			want = SkillShaman
		}
		if got != want {
			t.Errorf("神祇 %d 的教派 = %d，預期 %d", deity, got, want)
		}
		if got != SkillShaman && got != SkillPriesthood {
			t.Errorf("神祇 %d 算出教派技能 %d，只能是 15 或 16", deity, got)
		}
	}
}

func TestConvertAtTemple(t *testing.T) {
	tb := loadTables(t)
	c := hurtCharacter()
	c.Class = 4 // 牧師
	c.Traits[2] = 30

	res, err := ConvertAtTemple(tb, c, 500, 4) // 偶數 → 司祭
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("改宗失敗：%+v", res)
	}
	// **改宗不收金幣** —— Cost 填的是智力點數。
	if res.Gold != 500 {
		t.Errorf("改宗扣了錢，剩 %d，預期 500", res.Gold)
	}
	if !c.HasSkill(SkillPriesthood) {
		t.Error("改宗到偶數神應該學會司祭")
	}
	if c.Deity != 4 {
		t.Errorf("信奉的神 %d，預期 4", c.Deity)
	}
	if c.PrayChance != FavorMax {
		t.Errorf("改宗後祈禱成功率 %d，預期 %d", c.PrayChance, FavorMax)
	}
}

// 已經有薩滿或司祭任一項技能就不能再改宗 —— 一輩子只能一次。
func TestConvertAtTemple_OnlyOnce(t *testing.T) {
	tb := loadTables(t)
	c := hurtCharacter()
	c.Traits[2] = 30
	c.Skills[SkillShaman] = true

	res, err := ConvertAtTemple(tb, c, 500, 4)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Error("已經是薩滿卻還改宗成司祭")
	}
	if c.HasSkill(SkillPriesthood) {
		t.Error("被拒絕卻學會了司祭")
	}
}

func TestConvertAtTemple_NotEnoughPoints(t *testing.T) {
	tb := loadTables(t)
	c := hurtCharacter()
	c.Traits[2] = 1 // 智力 1，任何教派都學不起

	res, err := ConvertAtTemple(tb, c, 500, 3)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Error("點數不夠卻改宗成功")
	}
	if c.Deity != 0 {
		t.Error("被拒絕卻改了信仰")
	}
}

// 改宗完就祈禱得了 —— 兩支合起來才是完整的流程。
func TestConvertThenPray(t *testing.T) {
	tb := loadTables(t)
	c := hurtCharacter()
	c.Traits[2] = 30

	if res, err := ConvertAtTemple(tb, c, 500, 3); err != nil || !res.OK {
		t.Fatalf("改宗失敗：%+v %v", res, err)
	}
	c.PrayChance = 0
	if res := PrayAtTemple(c, 1000, 3); !res.OK {
		t.Errorf("改宗完卻祈禱不了：%+v", res)
	}
}
