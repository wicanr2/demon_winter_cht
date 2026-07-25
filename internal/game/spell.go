package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 效果類型。決定一個法術走哪條判定路徑。
//
// 「數值增減」類走通式（Magnitude），「二元成敗」類直接把 K×SP/M 當百分比
// 做單次擲骰，不重擲、不取 magnitude —— 兩者不可共用同一條路徑。
const (
	// EffectInstantDeath 即死。走完整的死亡結算鏈，**會**正確觸發勝負判定。
	EffectInstantDeath = 2
	// EffectBindRelease 解除束縛。需要力度與**系別**同時相符。
	EffectBindRelease = 10
	// EffectBindApply 施加束縛。狀態欄存的是符文系 id，不是布林旗標。
	EffectBindApply = 11
	// EffectWither 枯萎。三屬性各自「以現值為上限重擲」，不是扣減。
	EffectWither = 14
)

// 非 HP 屬性的鉗制範圍。下限 3 恰好對上攻略記載的「屬性不會低於 3」。
const (
	traitEffectFloor = 3
	traitEffectCeil  = 255
)

// SpellMagnitude 是「數值增減」類效果的通式。
//
//	magnitude = Roll(K × SP / M)
//	若 magnitude < 上限的 1/3 → 重擲
//
// K 的正負決定增益或傷害方向；回傳值帶符號。
// **全程整數運算**（除法向零捨去），不使用浮點。
func SpellMagnitude(r *rng.RNG, sp int, s gamedata.Spell) int {
	if s.M == 0 {
		return 0
	}
	bound := s.K * sp / s.M
	sign := 1
	if bound < 0 {
		sign = -1
		bound = -bound
	}
	if bound <= 0 {
		return 0
	}

	// 重擲直到不低於上限的 1/3。上限很小時 1/3 會是 0，任何擲值都通過，
	// 不會空轉。
	floor := bound / 3
	for {
		v := r.Roll(bound)
		if v >= floor {
			return sign * v
		}
	}
}

// ApplyTraitEffect 把 magnitude 套到非 HP 屬性上，並鉗制在 [3, 255]。
func ApplyTraitEffect(current, magnitude int) int {
	v := current + magnitude
	if v < traitEffectFloor {
		v = traitEffectFloor
	}
	if v > traitEffectCeil {
		v = traitEffectCeil
	}
	return v
}

// SpellRate 是「二元成敗」類效果的成功率（百分比）。
//
//	rate = K × SP / M       整數除法
func SpellRate(sp int, s gamedata.Spell) int {
	if s.M == 0 {
		return 0
	}
	return s.K * sp / s.M
}

// CastInstantDeath 施放即死類法術（effect_type 2）。
//
// 成功時目標死亡。**呼叫端要走完整的死亡結算鏈**（含勝負判定）——
// 這點與 AOE 擊殺不同，兩者不可共用同一條路徑。
func CastInstantDeath(r *rng.RNG, sp int, s gamedata.Spell, target *Unit) bool {
	if r.Roll(100) > SpellRate(sp, s) {
		return false
	}
	target.HP = 0
	target.Status = StatusDead
	return true
}

// BindResult 記錄一次束縛施加的結果。
type BindResult struct {
	// Applied 為 false 時 Reason 說明原因。
	Applied bool
	// AlreadyBound 表示目標狀態 >= 2，束縛不可疊加。
	AlreadyBound bool
	// Resisted 表示抗性擲骰成功。
	Resisted bool
	// Rounds 是束縛剩餘回合數。
	Rounds int
}

// CastBind 施加束縛（effect_type 11）。
//
//	resist = 目標力量 × 4 − SP × 4 × K / M
//	若 Roll(100) < resist → 抵抗成功
//	否則 目標.束縛剩餘回合 = K × SP / M；目標.狀態 = 本次法術的符文系 id
//
// 抗性量綱「力量 × 4」與命中率的「技巧 × 4」是同一套縮放。
// **狀態欄存的是符文系 id**，這是解除判定能運作的前提。
func CastBind(r *rng.RNG, sp int, s gamedata.Spell, target *Unit) BindResult {
	if target.Status >= StatusBindLow {
		return BindResult{AlreadyBound: true}
	}

	resist := target.Strength*4 - sp*4*s.K/s.M
	if r.Roll(100) < resist {
		return BindResult{Resisted: true}
	}

	rounds := SpellRate(sp, s)
	target.BindRounds = rounds
	target.Status = UnitStatus(s.School)
	return BindResult{Applied: true, Rounds: rounds}
}

// CastBindRelease 解除束縛（effect_type 10）。
//
// 需要**力度足夠**且**系別相符** —— 狀態欄存的符文系 id 必須等於
// 解除法術的符文系。死亡狀態不可解。
func CastBindRelease(sp int, s gamedata.Spell, target *Unit) bool {
	if target.Status == StatusDead {
		return false
	}
	if target.Status < StatusBindLow {
		return false
	}
	if int(target.Status) != s.School {
		return false
	}
	if SpellRate(sp, s) < target.BindRounds {
		return false
	}
	target.Status = StatusNormal
	target.BindRounds = 0
	return true
}

// CastWither 施放枯萎（effect_type 14）。
//
// **不是扣減**：三項屬性各自「以現值為上限重擲」。
// 回傳三項的新值（速度、力量、技巧）。
func CastWither(r *rng.RNG, target *Unit) (speed, strength, skill int) {
	reroll := func(cur int) int {
		if cur <= 1 {
			return cur
		}
		return r.Roll(cur)
	}
	target.Speed = reroll(target.Speed)
	target.Strength = reroll(target.Strength)
	target.Skill = reroll(target.Skill)
	return target.Speed, target.Strength, target.Skill
}
