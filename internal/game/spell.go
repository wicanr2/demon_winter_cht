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

// 走通式（Magnitude）的數值增減效果。K 的正負決定增益或傷害方向。
//
// 對照表見 docs/re/15 §2 的 type 欄。
const (
	// EffectAOE 範圍傷害。判定另有一套（docs/re/16 的 AOE handler），不走這裡。
	EffectAOE = 1
	// EffectSkillMod 技巧增減。
	EffectSkillMod = 3
	// EffectStrengthMod 力量增減。
	EffectStrengthMod = 4
	// EffectHP 生命值增減。K 為負是傷害、為正是治療。
	EffectHP = 5
	// EffectSpeedMod 速度增減。
	EffectSpeedMod = 6
	// EffectArmorMod 護甲增減。
	EffectArmorMod = 7
	// EffectSPMod 法力增減。
	EffectSPMod = 13
)

// CastMagnitudeEffect 套用「數值增減」類法術，回傳實際變動量。
//
// 走通式 SpellMagnitude，再依 effect_type 落到對應欄位。
// 回傳 ok=false 代表這個 effect_type 不走通式（呼叫端該改走特殊判定）。
//
// **屬性類鉗制在 [3,255]，生命與法力不套那個下限** ——
// 下限 3 是屬性的規則（攻略記載「屬性不會低於 3」），
// 拿去鉗生命值會讓角色永遠死不了。
func CastMagnitudeEffect(r *rng.RNG, sp int, s gamedata.Spell, target *Unit) (int, bool) {
	if target == nil {
		return 0, false
	}
	mag := SpellMagnitude(r, sp, s)

	switch s.Effect {
	case EffectHP:
		before := target.HP
		target.HP += mag
		if target.HP > target.MaxHP {
			target.HP = target.MaxHP
		}
		if target.HP < 0 {
			target.HP = 0
		}
		return target.HP - before, true

	case EffectSPMod:
		before := target.CurrentSP
		target.CurrentSP += mag
		if target.CurrentSP > target.MaxSP {
			target.CurrentSP = target.MaxSP
		}
		if target.CurrentSP < 0 {
			target.CurrentSP = 0
		}
		return target.CurrentSP - before, true

	case EffectSkillMod:
		before := target.Skill
		target.Skill = ApplyTraitEffect(target.Skill, mag)
		return target.Skill - before, true

	case EffectStrengthMod:
		before := target.Strength
		target.Strength = ApplyTraitEffect(target.Strength, mag)
		return target.Strength - before, true

	case EffectSpeedMod:
		before := target.Speed
		target.Speed = ApplyTraitEffect(target.Speed, mag)
		return target.Speed - before, true

	case EffectArmorMod:
		before := target.Armor
		target.Armor = ApplyTraitEffect(target.Armor, mag)
		return target.Armor - before, true
	}
	return 0, false
}

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
	// 符文系 1 是 4 的別名，比對前先重映射。
	school := s.School
	if school == 1 {
		school = 4
	}
	if int(target.Status) != school {
		return false
	}
	// 兩層門檻：力度要夠（K×SP/M >= 殘餘回合）、系別要對。
	if SpellRate(sp, s) < target.BindRounds {
		return false
	}
	target.Status = StatusNormal
	target.BindRounds = 0
	return true
}

// CastWither 施放枯萎（effect_type 14）。
//
//	若 Roll(100) > K × SP / M → 失敗
//	否則三項各自獨立重擲：屬性 = max(3, Roll(該屬性當前值))
//
// **不是「扣減」而是「以現值為上限重擲」** —— 與一般屬性削弱
// （effect_type 3/4/6）的數學行為不同，不可共用實作。
// K/M 只決定是否觸發，不參與屬性計算。下限 3 與全系統的屬性下限一致。
func CastWither(r *rng.RNG, sp int, s gamedata.Spell, target *Unit) bool {
	if r.Roll(100) > SpellRate(sp, s) {
		return false
	}
	reroll := func(cur int) int {
		if cur < 1 {
			cur = 1
		}
		v := r.Roll(cur)
		if v < traitEffectFloor {
			v = traitEffectFloor
		}
		return v
	}
	target.Speed = reroll(target.Speed)
	target.Strength = reroll(target.Strength)
	target.Skill = reroll(target.Skill)
	return true
}

// AOERadius 是範圍法術的半徑（Chebyshev 距離）。
//
// **框的大小是寫死的常數，不隨法術威力或投入的法力改變** ——
// 所有範圍法術共用同一個 5×5，威力只影響傷害量。
const AOERadius = 2

// aoeResistSchool 是「特定種族免疫」那條規則的符文系。
//
// 原版：`spell_school_id == 4 且 目標種族 ∈ {7,10}` → 效果歸零。
// 這條規則在單體與 AOE 兩處各自實作了一次，兩邊都要有。
const aoeResistSchool = 4

// aoeResistRaces 是對該系免疫的目標種族。
var aoeResistRaces = map[int]bool{7: true, 10: true}

// AOEHit 是一個被範圍法術波及的單位。
type AOEHit struct {
	Unit *Unit
	// Delta 是生命變化量，負數是傷害。
	Delta int
	// Killed 表示這一擊讓它倒下。
	Killed bool
	// Resisted 表示因種族免疫而完全無效。
	Resisted bool
}

// CastAOE 以 (centerX, centerY) 為中心施放範圍法術，回傳被波及的單位。
//
// **不分敵我。** 原版掃描全部 15 個槽位，只要座標落在框內就套用效果，
// 沒有排除隊友 —— 範圍法術會誤傷己方，那是原版行為不是 bug。
//
// 實作方式也照原版：**掃單位、判斷是否落在框內**，不是逐格掃地圖。
// 兩者結果相同，但照著寫比較不會在「空格也算一次」這種地方分岔。
func CastAOE(r *rng.RNG, b *Battle, s gamedata.Spell, sp, centerX, centerY int) []AOEHit {
	var hits []AOEHit

	for slot := 0; slot < CombatSlots; slot++ {
		u := b.Unit(slot)
		if u == nil || !u.Alive() {
			continue
		}
		if abs(u.X-centerX) > AOERadius || abs(u.Y-centerY) > AOERadius {
			continue
		}

		mag := SpellMagnitude(r, sp, s)
		resisted := s.School == aoeResistSchool && aoeResistRaces[u.RaceOrElement]
		if resisted {
			mag = 0
		}

		// **AOE 恆為傷害，magnitude 一律相減。**
		//
		// 這裡與單體版本不同：單體走 CastMagnitudeEffect，K 的正負決定
		// 增益或傷害；AOE 的原版程式碼是
		//
		//	if (hp > magnitude) { hp -= magnitude } else { dies }
		//
		// 不看 K 的正負。表裡三個 AOE 法術（火焰風暴 K=15、冰雹風暴 K=8、
		// 暴風 K=7）K 全是正的，照單體那套寫會變成「範圍治療」——
		// 實際踩過這個坑：施放火焰風暴後全隊回滿血。
		before := u.HP
		if u.HP > mag {
			u.HP -= mag
		} else {
			u.HP = 0
		}

		hit := AOEHit{Unit: u, Delta: u.HP - before, Resisted: resisted}
		if !u.Alive() {
			hit.Killed = true
			b.Kill(u)
		}
		hits = append(hits, hit)
	}
	return hits
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
