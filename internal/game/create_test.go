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
