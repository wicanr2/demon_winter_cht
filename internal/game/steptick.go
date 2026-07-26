package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 每走一步的 HP 變動（`FUN_222f_0619`，`docs/re/63`）
//
// 這一支原本是追「符印還在時會傷害隊伍」找到的，讀完才發現它同時是
// **巨魔天生能力「再生」的實作** —— 同一個迴圈，靠 `ds:0x15ca` 這個
// 模式旗標分成兩條路：
//
//	模式 0（一般行走）    巨魔 HP +1，其他種族不動
//	模式 0x80（符印區）   全隊 HP −1，不分種族
//
// 兩條路寫在同一個 if-else 鏈裡，所以**符印區的巨魔不會一邊再生一邊流血**
// —— 0x80 走的是傷害那一支，種族判斷被 `||` 短路掉了。

// StepHPMode 是 `ds:0x15ca` 的模式值。
type StepHPMode int

const (
	// StepHPRegen 是一般行走（`0x164d5  mov ds:0x15ca,0`）。
	StepHPRegen StepHPMode = 0
	// StepHPDrain 是符印區（`0x16c0d`／`0x1acd1  mov ds:0x15ca,0x80`）。
	StepHPDrain StepHPMode = 0x80
)

// stepHPThreshold 是兩條路的分界（`0x16542  cmp ds:0x15ca,0x7f`）。
//
// 原版的條件是 `< 0x7f && != 1` 走再生、`> 0x7f || 不是巨魔` 走傷害。
// 值 `1` 被特別排除掉，但**全檔沒有任何地方寫 1** —— 那條路走不到，
// 大概是留給沒做完的第三種模式。這裡不模擬它。
const stepHPThreshold = 0x7f

// StepHPResult 是一次 tick 的結果。
type StepHPResult struct {
	// Changed 代表有人的 HP 動了（原版用它決定要不要重畫）。
	Changed bool
	// Died 是這一步倒下的人的索引。
	Died []int
	// AllDead 代表全隊都死了。
	AllDead bool
}

// StepHPTick 走一次每步 HP 變動。
//
// 死亡判定是 `HP 減到剛好 0`（`*pcVar1 == '\0'`）—— 原版先減再比，
// 所以 HP 1 的人這一步會死。已經死掉的人只計入死亡人數，不再扣血。
func StepHPTick(party []Character, mode StepHPMode) StepHPResult {
	var res StepHPResult
	dead := 0

	for i := range party {
		c := &party[i]
		switch {
		case c.Status == scenario.StatusDead:
			dead++

		case int(mode) < stepHPThreshold:
			// 再生：只有巨魔，而且要沒滿血
			if c.Race == gamedata.Troll && c.CurrentHP < c.MaxHP {
				c.CurrentHP++
				res.Changed = true
			}

		case int(mode) > stepHPThreshold || c.Race != gamedata.Troll:
			// 流血。0x80 > 0x7f 成立，所以**巨魔在符印區照樣受傷**。
			res.Changed = true
			c.CurrentHP--
			if c.CurrentHP == 0 {
				dead++
				c.Status = scenario.StatusDead
				res.Died = append(res.Died, i)
				res.Changed = false // 原版在死亡分支把重畫旗標清掉，改走另一段
			}
		}
	}

	res.AllDead = len(party) > 0 && dead >= len(party)
	return res
}

// GlyphDrainMode 回報隊伍現在該用哪一種 HP tick 模式。
//
// 符印所在的子地圖、而且那個符印還沒解除時流血；其餘一律再生模式。
// 原版的判定條件更細（`docs/re/58` §3 依 Y 座標把子地圖 56 分成三段），
// 這裡先用「整張子地圖」——**範圍比原版大**，差異記在 `docs/re/63`。
func GlyphDrainMode(flags [3]byte, subMap int) StepHPMode {
	idx := GlyphIndexFor(subMap)
	if idx >= 0 && GlyphActive(flags, idx) {
		return StepHPDrain
	}
	return StepHPRegen
}
