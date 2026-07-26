package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func newCreation(t *testing.T, race gamedata.Race) *CharacterCreation {
	t.Helper()
	c, err := NewCharacterCreation(rng.NewWithSeed(7), loadTables(t), race)
	if err != nil {
		t.Fatalf("NewCharacterCreation: %v", err)
	}
	return c
}

// 手冊「三次機會」＝ 初次擲點 + 兩次重擲。
func TestCreation_RerollBudget(t *testing.T) {
	c := newCreation(t, gamedata.Elf)
	if got := c.RerollsLeft(); got != MaxRerolls {
		t.Errorf("起手剩 %d 次重擲，預期 %d", got, MaxRerolls)
	}

	for i := 1; i <= MaxRerolls; i++ {
		if err := c.Reroll([]gamedata.Trait{gamedata.Speed}); err != nil {
			t.Fatalf("第 %d 次重擲失敗：%v", i, err)
		}
	}
	if got := c.RerollsLeft(); got != 0 {
		t.Errorf("用完後剩 %d 次，預期 0", got)
	}
	if err := c.Reroll([]gamedata.Trait{gamedata.Speed}); err == nil {
		t.Error("機會用完後還能重擲")
	}
}

// 一次操作可以重擲多項屬性 —— 英文手冊寫「該屬性（或多項屬性）」。
//
// 一次只能一項會讓三次機會變得比原版吝嗇很多，是很容易寫錯的地方。
func TestCreation_RerollMultipleAtOnce(t *testing.T) {
	c := newCreation(t, gamedata.Human)

	all := []gamedata.Trait{gamedata.Speed, gamedata.Strength,
		gamedata.Intellect, gamedata.Endurance, gamedata.Skill}
	if err := c.Reroll(all); err != nil {
		t.Fatal(err)
	}
	if got := c.RerollsLeft(); got != MaxRerolls-1 {
		t.Errorf("一次重擲五項應只算一次機會，剩 %d", got)
	}
}

// 只重擲指定的那幾項，其餘不能被動到。
func TestCreation_RerollLeavesOthersAlone(t *testing.T) {
	c := newCreation(t, gamedata.Human)
	before := c.Traits

	if err := c.Reroll([]gamedata.Trait{gamedata.Speed}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < gamedata.NumTraits; i++ {
		if gamedata.Trait(i) == gamedata.Speed {
			continue
		}
		if c.Traits[i] != before[i] {
			t.Errorf("屬性 %d 沒被指定卻變了：%d → %d", i, before[i], c.Traits[i])
		}
	}
}

// 沒指定任何屬性時不消耗機會 —— 那是玩家按錯，不該罰他。
func TestCreation_EmptyRerollCostsNothing(t *testing.T) {
	c := newCreation(t, gamedata.Human)
	before := c.Traits

	if err := c.Reroll(nil); err != nil {
		t.Fatal(err)
	}
	if got := c.RerollsLeft(); got != MaxRerolls {
		t.Errorf("空的重擲不該消耗機會，剩 %d", got)
	}
	if c.Traits != before {
		t.Error("空的重擲不該改動屬性")
	}
}

func TestCreation_RerollRejectsBadTrait(t *testing.T) {
	c := newCreation(t, gamedata.Human)
	if err := c.Reroll([]gamedata.Trait{99}); err == nil {
		t.Error("超出範圍的屬性索引應回傳錯誤")
	}
}

// 「低於 6」是手冊給玩家的建議，不是能不能重擲的門檻。
//
// 英文原版寫「建議凡是低於 6 的數值都應該重骰」——「建議」兩個字是關鍵，
// 加成 gate 會讓玩家沒辦法重擲一個他嫌不夠好的 7。
func TestCreation_AdviceIsNotAGate(t *testing.T) {
	c := newCreation(t, gamedata.Human)
	// 人為把速度拉高，確認高於門檻仍然重擲得動。
	c.Traits[gamedata.Speed] = 15

	if c.BelowAdvice(gamedata.Speed) {
		t.Fatal("15 不該低於建議門檻")
	}
	if err := c.Reroll([]gamedata.Trait{gamedata.Speed}); err != nil {
		t.Errorf("高於建議門檻的屬性也應該重擲得動：%v", err)
	}
}

func TestCreation_BelowAdvice(t *testing.T) {
	c := newCreation(t, gamedata.Human)
	c.Traits[gamedata.Speed] = RerollAdviceThreshold - 1
	c.Traits[gamedata.Strength] = RerollAdviceThreshold

	if !c.BelowAdvice(gamedata.Speed) {
		t.Error("低於門檻應回 true")
	}
	if c.BelowAdvice(gamedata.Strength) {
		t.Error("等於門檻不算低於")
	}
}

// 平均值方框：人類每項 +2，所以平均一定比其他種族高。
//
// 手冊記人類「優缺點：無」，但程式碼給每項 +2 —— 平均值顯示要照程式碼走，
// 不然玩家拿手冊的印象去比會覺得數字怪怪的。
func TestCreation_RaceAverage(t *testing.T) {
	human := newCreation(t, gamedata.Human)
	elf := newCreation(t, gamedata.Elf)
	tb := loadTables(t)

	for i := 0; i < gamedata.NumTraits; i++ {
		tr := gamedata.Trait(i)

		gotHuman, err := human.RaceAverage(tr)
		if err != nil {
			t.Fatal(err)
		}
		if gotHuman != baseTraitExpectation+humanTraitBonus {
			t.Errorf("人類屬性 %d 平均 = %d，預期 %d",
				i, gotHuman, baseTraitExpectation+humanTraitBonus)
		}

		mod, err := tb.RaceModifier(gamedata.Elf, tr)
		if err != nil {
			t.Fatal(err)
		}
		gotElf, err := elf.RaceAverage(tr)
		if err != nil {
			t.Fatal(err)
		}
		want := baseTraitExpectation + mod
		if want < traitFloor {
			want = traitFloor
		}
		if gotElf != want {
			t.Errorf("精靈屬性 %d 平均 = %d，預期 %d", i, gotElf, want)
		}
	}
}

// 建角完成後的初始值。
func TestCreation_Finish(t *testing.T) {
	c := newCreation(t, gamedata.Human)
	c.Traits[gamedata.Endurance] = 12
	c.Traits[gamedata.Intellect] = 14

	ch := c.Finish("Kern", gamedata.Wizard)

	if ch.Name != "Kern" || ch.Race != gamedata.Human || ch.Class != gamedata.Wizard {
		t.Errorf("基本欄位不對：%+v", ch)
	}
	if ch.Level != 1 {
		t.Errorf("初始等級 = %d，預期 1", ch.Level)
	}
	if ch.MaxHP != 12 || ch.CurrentHP != 12 {
		t.Errorf("初始 HP = %d/%d，預期等於耐力 12", ch.CurrentHP, ch.MaxHP)
	}
	// 巫師的初始 SP = 智力，這是實測有依據的兩個職業之一。
	if ch.MaxSP != 14 {
		t.Errorf("巫師初始 SP = %d，預期等於智力 14", ch.MaxSP)
	}
}

// 沒有實測依據的職業給 0，不猜公式。
func TestCreation_UntestedClassesGetZeroSP(t *testing.T) {
	c := newCreation(t, gamedata.Human)
	c.Traits[gamedata.Intellect] = 14

	for _, class := range []gamedata.Class{
		gamedata.Ranger, gamedata.Paladin, gamedata.Barbarian, gamedata.Monk,
		gamedata.Cleric, gamedata.Thief, gamedata.Sorcerer,
		gamedata.Visionary, gamedata.Scholar,
	} {
		if sp := c.Finish("X", class).MaxSP; sp != 0 {
			t.Errorf("職業 %d 的初始 SP = %d，未實測的職業應給 0 而不是猜一個值",
				class, sp)
		}
	}
}

// 建出來的角色要能存進存檔並讀回來 —— 建角不能生出存不了的角色。
func TestCreation_ResultSurvivesSaveRoundTrip(t *testing.T) {
	c := newCreation(t, gamedata.Dwarf)
	ch := c.Finish("Brolor", gamedata.Cleric)

	var rec scenario.Character
	ch.ApplyTo(&rec)
	back := FromSave(rec)

	if back.Name != ch.Name || back.Race != ch.Race || back.Class != ch.Class ||
		back.Level != ch.Level || back.MaxHP != ch.MaxHP {
		t.Errorf("存檔一趟後走樣：\n得到 %+v\n預期 %+v", back, ch)
	}
	if back.Traits != ch.Traits {
		t.Errorf("屬性走樣：%v vs %v", back.Traits, ch.Traits)
	}
}

// 擲點分布要與原版一致：2d6 + 1 + 種族修正，下限 3。
//
// 這條擋的是「期望值對、分布錯」—— 這個 bug 真的發生過：實作原本是
// Roll(15)（均勻 1–15），期望值剛好也是 8，所以畫面上的「該種族平均」
// 一直是對的，沒人看得出來。分布錯的後果是玩家會擲出原版不可能出現的
// 極端值，而且少了 2d6 往中間集中的手感。
func TestRollTraits_MatchesOriginalDistribution(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(20260726)

	// 用種族修正為 0 的那些項來看純粹的骰子範圍。
	const rolls = 20000
	seen := map[int]int{}
	for i := 0; i < rolls; i++ {
		traits, err := RollTraits(r, tb, gamedata.Elf)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < gamedata.NumTraits; j++ {
			mod, err := tb.RaceModifier(gamedata.Elf, gamedata.Trait(j))
			if err != nil {
				t.Fatal(err)
			}
			if mod != 0 {
				continue
			}
			seen[traits[j]]++
		}
	}
	if len(seen) == 0 {
		t.Fatal("精靈沒有任何修正為 0 的屬性，測試前提不成立")
	}

	// 2d6 + 1 → 3..13。原本的 Roll(15) 會擲出 1、2、14、15。
	const lo, hi = 2*1 + traitRollBonus, 2*traitDie + traitRollBonus
	for v, n := range seen {
		if v < lo || v > hi {
			t.Errorf("擲出 %d（%d 次），超出 2d6+%d 的範圍 %d–%d",
				v, n, traitRollBonus, lo, hi)
		}
	}

	// 中間值要比極端值常見很多 —— 均勻分布不會有這個性質。
	mid, edge := seen[8], seen[lo]+seen[hi]
	if mid <= edge {
		t.Errorf("中央值 8 出現 %d 次，兩端合計 %d 次 —— 看起來不像 2d6 的三角分布",
			mid, edge)
	}
}

// 三次機會 = 初次擲點 + 兩次重擲，與原版的迴圈次數一致。
//
// 原版把計數器設 0，每輪玩家選完後 +1、`cmp 3` 沒到就再擲一輪
// （0x13b9d／0x13d60–0x13d69）；而且起手先把五項旗標全設成 1，
// 所以第一輪就是「全部擲一次」。三輪 = 初擲 + 兩次重擲。
func TestCreation_RerollBudgetMatchesOriginalLoop(t *testing.T) {
	const originalRollRounds = 3
	if MaxRerolls != originalRollRounds-1 {
		t.Errorf("MaxRerolls = %d，原版總共擲 %d 輪（含初擲），所以應該是 %d",
			MaxRerolls, originalRollRounds, originalRollRounds-1)
	}
}

// 新建的角色是**徒手空背包**，不是十把匕首。
//
// `scenario.InventorySlot` 的零值是 `Type == 0`，而 `0` 是匕首那個道具型別；
// 空槽的值是 `0xff`。不明確清空的話，剛建好的角色名冊上會顯示「匕首」。
//
// **這個 bug 只有真的開新遊戲時才看得到** —— 從存檔載入的角色每一格都是
// 真資料，零值永遠不會出現。它躲過了 `-newgame` 之前所有的驗收。
func TestFinishClearsInventory(t *testing.T) {
	out := newCreation(t, gamedata.Human).Finish("Test", gamedata.Ranger)

	for i, slot := range out.Inventory {
		if !slot.Empty() {
			t.Errorf("背包第 %d 格 type=0x%02x，預期空槽 0x%02x（0 是匕首）",
				i, slot.Type, scenario.SlotEmpty)
		}
	}
	if !out.Weapon().Empty() {
		t.Error("新角色應該徒手")
	}
	if !out.Armor().Empty() {
		t.Error("新角色應該沒有護甲")
	}
	if out.ArmorRating() != 0 {
		t.Errorf("護甲值 = %d，預期 0", out.ArmorRating())
	}
}

// 新角色的「含加成」屬性要等於天生值。
//
// 新角色身上沒有裝備，兩組本來就該相等 —— 原版建角也是做這件事
// （`0x14d95`，`docs/re/89`）。
//
// **不同步的後果只在存檔裡看得見**：名冊顯示的是規則層的 `Traits`
// （自己擲的點），而存檔的「含加成」欄位會留著載入時的舊值 ——
// 對新角色來說那是出貨存檔那五個人的數字。重讀存檔就換人。
func TestFinishSyncsTraitsWithBonus(t *testing.T) {
	out := newCreation(t, gamedata.Dwarf).Finish("Test", gamedata.Wizard)

	if got, want := int(out.TraitsWithBonus.Strength), out.Traits[gamedata.Strength]; got != want {
		t.Errorf("力量：含加成 %d、天生 %d", got, want)
	}
	if got, want := int(out.TraitsWithBonus.Skill), out.Traits[gamedata.Skill]; got != want {
		t.Errorf("技巧：含加成 %d、天生 %d", got, want)
	}
	if got, want := int(out.TraitsWithBonus.Speed), out.Traits[gamedata.Speed]; got != want {
		t.Errorf("速度：含加成 %d、天生 %d", got, want)
	}
	if got, want := int(out.TraitsWithBonus.MaxSP), out.MaxSP; got != want {
		t.Errorf("法力上限：天生 %d、實際 %d", got, want)
	}
}

// 存檔往返不能掉「含加成」那一組。
//
// `ApplyTo` 的註解自己寫著「FromSave 多讀一個欄位，這裡就要多寫一個，
// 否則那個欄位會在存檔時悄悄退回舊值」—— 這四個欄位原本兩邊都沒接，
// 所以連「退回舊值」都沒察覺。
func TestTraitsWithBonusRoundTrip(t *testing.T) {
	out := newCreation(t, gamedata.Elf).Finish("RT", gamedata.Ranger)

	var rec scenario.Character
	// 先塞進別人的值，模擬「新角色寫進舊記錄」。
	rec.StrengthBonus, rec.SkillBonus, rec.SpeedBonus, rec.MaxSPNatural = 99, 98, 97, 96
	out.ApplyTo(&rec)

	if rec.StrengthBonus != out.TraitsWithBonus.Strength {
		t.Errorf("力量(含加成) = %d，預期 %d（舊值 99 應該被蓋掉）",
			rec.StrengthBonus, out.TraitsWithBonus.Strength)
	}
	if rec.SkillBonus != out.TraitsWithBonus.Skill {
		t.Errorf("技巧(含加成) = %d，預期 %d", rec.SkillBonus, out.TraitsWithBonus.Skill)
	}
	if rec.SpeedBonus != out.TraitsWithBonus.Speed {
		t.Errorf("速度(含加成) = %d，預期 %d", rec.SpeedBonus, out.TraitsWithBonus.Speed)
	}
	if rec.MaxSPNatural != out.TraitsWithBonus.MaxSP {
		t.Errorf("天生法力上限 = %d，預期 %d", rec.MaxSPNatural, out.TraitsWithBonus.MaxSP)
	}

	// 反向：讀回來要一致。
	back := FromSave(rec)
	if back.TraitsWithBonus != out.TraitsWithBonus {
		t.Errorf("讀回來 = %+v，預期 %+v", back.TraitsWithBonus, out.TraitsWithBonus)
	}
}
