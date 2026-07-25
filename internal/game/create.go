package game

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 建角流程：選種族 → 擲點（可重擲）→ 選職業 → 命名。
//
// **兩本手冊對重擲的描述不一致，這裡以英文原版為準。**
//
//	英文原版（docs/manual/part-1.md）：
//	  「你有三次機會可以調整數值。按下該屬性（或多項屬性）對應的數字，
//	    再按 ESC 重新生成新數值。這個動作最多只能做兩次；
//	    最後一次生成的數值就是最終結果。建議凡是低於 6 的數值都應該重骰。」
//
//	軟體世界 1990 中譯本（docs/manual-cht/01-transcript.md）：
//	  「三次機會……按該項屬性前的號碼再按 ENTER，電腦會重給一個點數。」
//
// 三處差異與裁決：
//
//  1. **一次能重擲幾項**：英文說可選「多項屬性」，中譯說一次一項。
//     取英文 —— 中譯本是譯本，不是獨立來源。
//  2. **「三次機會」對「最多兩次」**：英文自己這句就前後不一。合理解讀是
//     初次擲點算第一次「生成」，之後還能重擲兩次，合計三次生成。
//  3. **「低於 6」是建議不是限制**：英文寫「建議凡是低於 6 的數值都應該重骰」，
//     是給玩家的判斷提示，不是能不能重擲的門檻。不加這道 gate。
//
// 反組譯**尚未定位**這段邏輯。依本專案的 oracle 優先序（人工實測 > DOSBox >
// 手冊／攻略 > 反編譯），手冊本來就排在反編譯之前，所以這不算「無憑實作」；
// 之後若在程式碼裡找到，以程式碼複核這三個裁決。

// MaxRerolls 是建角時能重擲幾次。
//
// 初次擲點算第一次「生成」，所以這裡是 2，合起來是手冊說的三次機會。
const MaxRerolls = 2

// RerollAdviceThreshold 是手冊建議重擲的門檻。
//
// **只是顯示用的提示，不是限制。** 玩家想重擲 12 也可以。
const RerollAdviceThreshold = 6

// CharacterCreation 是一次建角的進行中狀態。
type CharacterCreation struct {
	rng    *rng.RNG
	tables *gamedata.Tables

	Race  gamedata.Race
	Class gamedata.Class
	Name  string

	// Traits 是目前的五項屬性。
	Traits [gamedata.NumTraits]int

	// rerollsUsed 是已用掉的重擲次數。
	rerollsUsed int
}

// NewCharacterCreation 選定種族並擲出第一組屬性。
func NewCharacterCreation(r *rng.RNG, t *gamedata.Tables, race gamedata.Race) (*CharacterCreation, error) {
	c := &CharacterCreation{rng: r, tables: t, Race: race}
	traits, err := RollTraits(r, t, race)
	if err != nil {
		return nil, err
	}
	c.Traits = traits
	return c, nil
}

// RerollsLeft 回傳還能重擲幾次。
func (c *CharacterCreation) RerollsLeft() int { return MaxRerolls - c.rerollsUsed }

// Reroll 重擲指定的幾項屬性，算一次機會。
//
// **一次操作可以指定多項** —— 英文手冊寫「該屬性（或多項屬性）」。
// 一次只重擲一項會讓三次機會變得比原版吝嗇很多。
//
// 沒有指定任何屬性時不消耗機會（那是玩家按錯，不該罰他）。
func (c *CharacterCreation) Reroll(which []gamedata.Trait) error {
	if len(which) == 0 {
		return nil
	}
	if c.RerollsLeft() <= 0 {
		return fmt.Errorf("game: 重擲機會已用完")
	}

	fresh, err := RollTraits(c.rng, c.tables, c.Race)
	if err != nil {
		return err
	}
	for _, tr := range which {
		if tr < 0 || int(tr) >= gamedata.NumTraits {
			return fmt.Errorf("game: 屬性索引 %d 超出範圍", tr)
		}
		c.Traits[tr] = fresh[tr]
	}
	c.rerollsUsed++
	return nil
}

// RaceAverage 回傳某項屬性在這個種族的平均值，供畫面右側的方框顯示。
//
// 手冊：「畫面最右側的方框中，會顯示該種族這些屬性的平均值。」
// 玩家拿它跟擲出來的點數比，決定要不要重擲。
//
//	平均 = 基礎骰期望值 + 種族修正（人類再 +2）
//
// 基礎骰期望值精確為 8（見 docs/spec/05-character.md）。
// 下限鉗制（低於 3 拉到 3）會讓實際期望略高於這個值，
// 但那只在修正很負的少數格子才會發生，顯示用的近似值不需要那麼精。
func (c *CharacterCreation) RaceAverage(tr gamedata.Trait) (int, error) {
	mod, err := c.tables.RaceModifier(c.Race, tr)
	if err != nil {
		return 0, err
	}
	avg := baseTraitExpectation + mod
	if avg < traitFloor {
		avg = traitFloor
	}
	if c.Race == gamedata.Human {
		avg += humanTraitBonus
	}
	return avg, nil
}

// baseTraitExpectation 是基礎骰的期望值。Roll(15) 回傳 1..15，期望 8。
const baseTraitExpectation = 8

// BelowAdvice 回報某項屬性是否低於手冊建議的重擲門檻。
func (c *CharacterCreation) BelowAdvice(tr gamedata.Trait) bool {
	if tr < 0 || int(tr) >= gamedata.NumTraits {
		return false
	}
	return c.Traits[tr] < RerollAdviceThreshold
}

// Finish 把建角結果變成一名角色。
//
// 初始值照 docs/spec/05-character.md：
//   - 最大 HP = 耐力（1:1）
//   - 等級 1
//   - 初始 SP 依職業而異，目前只確定巫師 = 智力、遊俠 = 0，
//     其餘職業未逐一測試 —— 未確定的一律給 0 而不是猜一個公式。
func (c *CharacterCreation) Finish(name string, class gamedata.Class) Character {
	c.Name = name
	c.Class = class

	out := Character{
		Name:   name,
		Race:   c.Race,
		Class:  class,
		Level:  1,
		Traits: c.Traits,
	}
	out.MaxHP = c.Traits[gamedata.Endurance]
	out.CurrentHP = out.MaxHP
	out.MaxSP = initialSP(class, c.Traits[gamedata.Intellect])
	out.CurrentSP = out.MaxSP
	return out
}

// initialSP 回傳初始法力值。
//
// **只有巫師與遊俠有實測依據**（docs/spec/05-character.md）。其餘職業未測，
// 給 0 —— 猜一個公式會讓「這個值哪來的」變成日後查不出來的謎。
func initialSP(class gamedata.Class, intellect int) int {
	switch class {
	case gamedata.Wizard:
		return intellect
	default:
		return 0
	}
}
