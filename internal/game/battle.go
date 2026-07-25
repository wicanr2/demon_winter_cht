package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 槽位配置：0–6 怪物、7–11 玩家、12–14 召喚／幻術生物。
//
// 12/13/14 三格正好對上手冊「同一場戰鬥中最多同時存在三隻召喚生物」。
const (
	MonsterSlotStart = 0
	MonsterSlotEnd   = 7 // 不含
	PlayerSlotStart  = 7
	PlayerSlotEnd    = 12 // 不含
	SummonSlotStart  = 12
	SummonSlotEnd    = 15 // 不含
)

// Outcome 是一場戰鬥的結局。
type Outcome int

const (
	// Ongoing 戰鬥還在進行。
	Ongoing Outcome = iota
	// Victory 怪物全滅。
	Victory
	// Defeat 隊伍全滅。
	Defeat
)

// Battle 是一場戰鬥的狀態機。
//
// 每回合開始時重建行動順序（玩家與怪物混在一起依速度降冪），然後依序行動 ——
// **不是**「玩家全體行動完再換敵人」的兩段式。
type Battle struct {
	rng   *rng.RNG
	units [CombatSlots]*Unit

	round int
	order []int
	// cursor 指向 order 裡下一個要行動的單位。
	cursor int

	// points 是目前行動單位的剩餘行動點數，pointsFor 記住那是誰的。
	// 見 action.go。
	points    int
	pointsFor *Unit
}

// NewBattle 建立一場戰鬥。units 依槽位放入，空槽傳 nil。
func NewBattle(r *rng.RNG, units []*Unit) *Battle {
	b := &Battle{rng: r}
	for _, u := range units {
		if u == nil || u.Slot < 0 || u.Slot >= CombatSlots {
			continue
		}
		// AITargetSlot 的零值 0 是合法槽位，會讓怪物一開始就「記得」
		// 打 0 號。統一設成「還沒有目標」，第一回合才會真的去挑。
		u.AITargetSlot = noAITarget
		if u.Side == 0 {
			u.Side = defaultSide(u.IsPlayer)
		}
		b.units[u.Slot] = u
	}
	return b
}

// Unit 取回某個槽位的單位，空槽回傳 nil。
func (b *Battle) Unit(slot int) *Unit {
	if slot < 0 || slot >= CombatSlots {
		return nil
	}
	return b.units[slot]
}

// Round 回傳目前回合數（第一回合是 1）。
func (b *Battle) Round() int { return b.round }

// Units 回傳全部非空槽的單位。
func (b *Battle) Units() []*Unit {
	var out []*Unit
	for _, u := range b.units {
		if u != nil {
			out = append(out, u)
		}
	}
	return out
}

// BeginRound 重建行動順序並把游標歸零。回合數 +1。
func (b *Battle) BeginRound() {
	b.round++
	b.order = TurnOrder(b.Units())
	b.cursor = 0
}

// Current 回傳目前該行動的單位。回合已跑完時回傳 nil。
func (b *Battle) Current() *Unit {
	for b.cursor < len(b.order) {
		u := b.units[b.order[b.cursor]]
		// 排序後才死掉的單位跳過 —— 順序是回合開始時算好的，
		// 中途死亡不會重排，但已死的不該再行動。
		if u != nil && u.Alive() {
			b.beginTurn(u)
			return u
		}
		b.cursor++
	}
	return nil
}

// EndTurn 讓目前單位結束行動，游標前進。
func (b *Battle) EndTurn() { b.cursor++ }

// RoundFinished 回報這一回合的所有單位是否都行動過了。
func (b *Battle) RoundFinished() bool { return b.Current() == nil }

// Outcome 判定勝負。
//
// 怪物全滅為勝、隊伍全滅為敗。**召喚與幻術生物屬玩家陣營但不算隊伍成員** ——
// 只剩召喚物存活時隊伍仍算全滅。
func (b *Battle) Outcome() Outcome {
	monstersAlive := false
	for s := MonsterSlotStart; s < MonsterSlotEnd; s++ {
		if u := b.units[s]; u != nil && u.Alive() {
			monstersAlive = true
			break
		}
	}
	partyAlive := false
	for s := PlayerSlotStart; s < PlayerSlotEnd; s++ {
		if u := b.units[s]; u != nil && u.Alive() {
			partyAlive = true
			break
		}
	}

	switch {
	case !partyAlive:
		return Defeat
	case !monstersAlive:
		return Victory
	default:
		return Ongoing
	}
}

// Kill 執行死亡結算：X／Y 清零、狀態設為死亡。
//
// 兩個欄位都要清 —— 行動順序的排除條件同時看「狀態 > 1」與「X == 0」，
// 只設其中一個仍會被另一條擋掉，但語意不完整。
func (b *Battle) Kill(u *Unit) {
	u.HP = 0
	u.X, u.Y = 0, 0
	u.Status = StatusDead
}

// Enemies 回傳目標單位的敵對陣營中還活著的槽位。
//
// 看的是 Side 不是 IsPlayer —— 被附身的隊員站在怪物那一邊，
// 對其餘隊員來說就是敵人（見 side.go）。
func (b *Battle) Enemies(of *Unit) []int {
	var out []int
	for s, u := range b.units {
		if u == nil || !u.Alive() {
			continue
		}
		if u.OnPlayerSide() != of.OnPlayerSide() {
			out = append(out, s)
		}
	}
	return out
}

// ResolveAttack 執行一次攻擊並處理死亡結算。
func (b *Battle) ResolveAttack(attacker, target *Unit, hitModifier int) AttackResult {
	res := Attack(b.rng, attacker, target, hitModifier)
	if res.Killed {
		b.Kill(target)
	}
	return res
}

// FreeSummonSlot 回傳第一個空的召喚槽位，全滿時回傳 −1。
func (b *Battle) FreeSummonSlot() int {
	for s := SummonSlotStart; s < SummonSlotEnd; s++ {
		if b.units[s] == nil || !b.units[s].Alive() {
			return s
		}
	}
	return -1
}

// SummonKind 區分召喚與幻術。兩者共用同一套機制，只差成本倍率與是否保留法力。
type SummonKind int

const (
	// KindSummon 召喚（法術 id 0x18）：成本 ×4，保留表值法力。
	KindSummon SummonKind = iota
	// KindIllusion 幻術（法術 id 0x19）：成本 ×2，**法力歸零**。
	// 幻化的生物因此不能施法 —— 手冊明說這是它的劣勢。
	KindIllusion
)

// SummonCost 回傳召喚或幻化某個生物要花的法力。
func SummonCost(e gamedata.SummonEntry, kind SummonKind) int {
	if kind == KindIllusion {
		return e.IllusionCost()
	}
	return e.SummonCost()
}

// PlaceSummon 把一隻召喚／幻術生物放進戰場。
//
// 呼叫端要先確認 slot 有效、施法者法力足夠。回傳新單位。
//
// 陣營：召喚 14／幻術 13，**皆屬玩家陣營**（會被友方 AOE 波及）。
// 擊殺經驗值恆為 0 —— 原版從未寫入那個欄位，對應手冊
// 「殺死召喚生物也不會獲得經驗值」。
func (b *Battle) PlaceSummon(slot int, e gamedata.SummonEntry, kind SummonKind, x, y int) *Unit {
	u := &Unit{
		Slot:          slot,
		X:             x,
		Y:             y,
		Speed:         int(e.Word(0)),
		Strength:      int(e.Word(1)),
		Skill:         int(e.Word(2)),
		HP:            int(e.Word(3)),
		MaxHP:         int(e.Word(3)),
		Armor:         int(e.Word(6)),
		WeaponIndex:   int(e.Word(5)),
		RaceOrElement: int(e.Word(4)),
		IsPlayer:      true,
		Side:          SideSummon,
	}
	if kind == KindIllusion {
		u.Side = SideIllusion
	}
	if kind == KindIllusion {
		u.MaxSP, u.CurrentSP = 0, 0
	} else {
		u.MaxSP = int(e.Word(9))
		u.CurrentSP = u.MaxSP
	}
	u.Facing = b.rng.Roll(4) - 1
	b.units[slot] = u
	return u
}

// RollMonsterStats 對怪物的速度與生命做進場擾動。
//
//	速度 = 基礎速度 × (6·state + 7·M) / (10 × M)      下限 3
//	生命 = 基礎生命 × (8·state + 6·M) / (10 × M)      下限 1
//
// 原版走軟體浮點（`基礎值 × (rand01 × 0.6 + 0.7)`），但這個整數式可精確重現：
// `gcd(6,M) = gcd(8,M) = 1` 且 M 為質數 → 精確值永不為整數 → 捨入不跨邊界。
// 210 萬組比對零不一致，推導見 docs/re/20 §8。
func RollMonsterStats(r *rng.RNG, baseSpeed, baseHP int) (speed, hp int) {
	const m = int64(rng.Modulus)

	s := int64(r.Next())
	speed = int(int64(baseSpeed) * (6*s + 7*m) / (10 * m))
	if speed < 3 {
		speed = 3
	}

	s = int64(r.Next())
	hp = int(int64(baseHP) * (8*s + 6*m) / (10 * m))
	if hp < 1 {
		hp = 1
	}
	return speed, hp
}
