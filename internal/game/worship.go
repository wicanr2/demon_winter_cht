package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 紮營選單的 Worship（`1000:0ed0`，見 `docs/re/43`）。
//
// 向自己信奉的神祇祈求 —— 成功的話神祇會替你放一個法術，**力度固定 300**。
// 一般施法的力度是玩家投入的那幾點法力，300 是完全不同的量級：神祇出手
// 就是壓倒性的。代價是**每次成功都把祈禱成功率永久扣 5**。

// worshipStatusLimit 是能敬拜的狀態上限（原版 `< 2`）。
const worshipStatusLimit = 2

// WorshipPower 是神祇施法的力度。原版寫死 300（`1000:0f80`）。
const WorshipPower = 300

// worshipChanceDrop 是每次成功之後扣掉的祈禱成功率。
// 與神殿裡呼喚神祇共用同一個 `+0xeb`、同一個 −5（`docs/re/19` §3.3）。
const worshipChanceDrop = 5

// DeitySpellNone 是「這位神祇不賜法術」。
const DeitySpellNone = -1

// DeitySpellRandom 是「隨機挑一個」的哨兵（`rnd(11) + 0x1a` → 27–37）。
// **原版資料裡沒有任何一位神祇用這個值**，所以這條路實際上跑不到 ——
// 留著是因為程式碼真的有這個分支。
const DeitySpellRandom = -2

const (
	worshipRandomDie  = 11
	worshipRandomBase = 0x1a
)

// WorshipResult 是一次敬拜的結果。
type WorshipResult struct {
	// OK 為 true 代表祈求本身成立（不代表神祇回應了）。
	OK bool
	// Answered 為 true 代表神祇回應，法術放了出去。
	Answered bool
	// Reason 是沒祈求成的原因。
	Reason string
	// Chance 是這一次的成功率（祈求前的 `+0xeb`）。
	Chance int
	// SpellID 是神祇賜下的法術（Answered 才有意義）。
	SpellID int
	// Cast 是那個法術的施放結果。
	Cast CampCastResult
}

// CanWorship 回報這名角色現在能不能敬拜。
func CanWorship(c *Character) (bool, string) {
	switch {
	case c == nil:
		return false, "沒有這個人"
	case int(c.Status) >= worshipStatusLimit:
		return false, "現在沒辦法祈求"
	case !c.HasSkill(SkillShaman) && !c.HasSkill(SkillPriesthood):
		return false, "不懂得如何祈求"
	case c.WorshipedToday:
		return false, "今天已經祈求過了"
	case c.Deity < DeityMin || c.Deity > DeityMax:
		return false, "沒有信奉的神祇"
	}
	return true, ""
}

// Worship 向自己的神祇祈求。
//
// 順序照原版：**先記帳、再擲骰**（`+0xf1` 在擲骰之前就設起來，
// 所以失敗也用掉今天的機會），成功才扣成功率。
//
// target 是法術要落在誰身上。原版沒有選目標這一步（`FUN_1000_11e5`
// 自己處理），本專案讓呼叫端指定 —— 見 `docs/re/43` §3。
func Worship(r *rng.RNG, t *gamedata.Tables, c, target *Character) WorshipResult {
	if ok, why := CanWorship(c); !ok {
		return WorshipResult{Reason: why}
	}
	c.WorshipedToday = true

	spellID, err := t.DeitySpell(c.Deity)
	if err != nil {
		return WorshipResult{Reason: "查不到這位神祇"}
	}

	res := WorshipResult{OK: true, Chance: c.PrayChance, SpellID: spellID}
	if r.Roll(100) > c.PrayChance || spellID == DeitySpellNone {
		return res
	}
	res.Answered = true
	c.PrayChance -= worshipChanceDrop
	if c.PrayChance < 0 {
		c.PrayChance = 0
	}

	if spellID == DeitySpellRandom {
		spellID = r.Roll(worshipRandomDie) + worshipRandomBase
		res.SpellID = spellID
	}
	sp, err := t.Spell(spellID)
	if err != nil {
		res.Cast = CampCastResult{Reason: "查不到那個法術"}
		return res
	}
	// 神祇不吃施術者的法力 —— 力度是固定的 300，不是投入的 SP。
	// 借 CampCast 的效果套用，但法力由這裡先墊上再還回去。
	before := c.CurrentSP
	c.CurrentSP = WorshipPower
	res.Cast = CampCast(r, c, target, sp, WorshipPower)
	c.CurrentSP = before
	return res
}
