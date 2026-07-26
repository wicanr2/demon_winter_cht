package game

import (
	"sort"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 戰場最多同場 15 個單位：槽位 0–6 怪物、7–14 玩家。
const CombatSlots = 15

// weaponDamageDice 是武器傷害骰表，內嵌於 DEMON.INT（31f0:1785），不是資料檔。
//
// **完整表有 14 項**：索引 0 是徒手、1–8 是玩家武器、9–13 是怪物與召喚生物的
// 天生攻擊。索引 1–8 的值與手冊武器傷害範圍（1-3、1-4、1-6、1-6、1-7、1-8、
// 1-10、1-12）逐項吻合。
var weaponDamageDice = [14]int{2, 3, 4, 6, 6, 7, 8, 10, 12, 7, 13, 11, 14, 3}

// WeaponDamageDie 回傳武器索引對應的傷害骰面數。超出表範圍回傳 0。
func WeaponDamageDie(idx int) int {
	if idx < 0 {
		idx = -idx
	}
	if idx >= len(weaponDamageDice) {
		return 0
	}
	return weaponDamageDice[idx]
}

// UnitStatus 是戰鬥單位的狀態，存在 +0x102。
//
// **這是單一列舉值，不是位元旗標。**
type UnitStatus int

const (
	StatusNormal UnitStatus = 0
	StatusPoison UnitStatus = 1
	// 2、3、4 是束縛三級；實際存的是施加束縛那個法術的符文系 id，
	// 解除判定要靠它比對系別。
	StatusBindLow  UnitStatus = 2
	StatusBindHigh UnitStatus = 4
	StatusDead     UnitStatus = 5
)

// CombatStyle 是玩家角色的戰鬥風格碼，存在 unit+0x22。
//
// **這個欄位是雙用途的**：怪物與召喚生物在同一格存種族／元素類型（7、10、13…），
// 玩家角色存風格（0x15–0x18）。抗性判定當種族用、命中與傷害當風格用，
// 實作時不能只留一種語意。
type CombatStyle int

const (
	StyleNone    CombatStyle = 0
	StyleKarate  CombatStyle = 0x15
	StyleKungFu  CombatStyle = 0x16
	StyleBoth    CombatStyle = 0x17
	StyleFencing CombatStyle = 0x18
)

// 鬥劍加成適用的武器索引：短劍、闊劍、雙手劍。
//
// 這三個數字寫死在機器碼裡。對上手冊附錄 E 的武器表就清楚了 ——
// 索引 0 是徒手、武器從 1 起算，3／6／8 正好是全部三把 Sword 系武器。
var fencingWeapons = map[int]bool{3: true, 6: true, 8: true}

// StyleFor 依武器索引與已學技能算出戰鬥風格。建立戰鬥單位時算一次。
func StyleFor(weaponIdx int, hasSkill func(gamedata.SkillID) bool) CombatStyle {
	if weaponIdx == 0 {
		karate := hasSkill(gamedata.SkillKarate)
		kungfu := hasSkill(gamedata.SkillKungFu)
		switch {
		case karate && kungfu:
			return StyleBoth
		case kungfu:
			return StyleKungFu
		case karate:
			return StyleKarate
		}
		return StyleNone
	}
	if fencingWeapons[weaponIdx] && hasSkill(gamedata.SkillFencing) {
		return StyleFencing
	}
	return StyleNone
}

// Unit 是戰場上的一個單位。玩家與怪物共用同一個結構 ——
// 原版的普通攻擊也是同一套核心邏輯，不必分兩份實作。
type Unit struct {
	Slot int
	Name string

	// X 為 0 代表空槽或已死，行動順序會排除。
	X, Y int

	Speed, Strength, Skill, Intellect int
	Level                             int

	HP, MaxHP int
	Armor     int

	// WeaponIndex 帶符號：**負數代表附毒**，取絕對值才是骰表索引。
	WeaponIndex int

	// Style 見 CombatStyle 的雙用途說明。怪物在這裡存種族／元素類型。
	Style CombatStyle

	Status UnitStatus

	// StatusCount 為負代表暫時失去行動：該回合不能動，且每回合 +1。
	// 功夫暈眩會把它設成 −1，連續命中可以疊加。
	StatusCount int

	// BindRounds 是束縛剩餘回合數。解除判定要拿它與解除法術的力度比。
	BindRounds int

	// Facing 是單位面向（0–3），召喚生物進場時擲定。
	Facing int

	// MaxSP／CurrentSP 是法力。幻化出來的生物法力被歸零，因此不能施法。
	MaxSP, CurrentSP int

	// IsPlayer 代表這個單位有沒有玩家角色記錄（＝在不在槽位 7–14）。
	// **魅惑不會改變它** —— 會變的是 Side。
	IsPlayer bool

	// Side 是原版單位記錄的 `+0x20`（陣營），值域與語意見 side.go。
	// 零值交給 NewBattle 依 IsPlayer 補成 SidePlayer／SideMonster。
	Side int

	// AITargetSlot 是怪物記住的攻擊目標槽位，對應原版戰鬥單位記錄的
	// `unit+0x1e`（見 ai.go）。零值 0 是合法槽位，所以新單位要由建立端
	// 設成 noAITarget —— NewBattle 會統一處理。
	AITargetSlot int

	// Berserking 是否已學狂暴（技能 id 6），影響爆擊門檻。
	Berserking bool

	// WeaponEffect／EnchantBonus 是玩家武器的特效與附魔加成。
	WeaponEffect int
	EnchantBonus int

	// RaceOrElement 是目標種族／元素類型，用於「對特定種族再加一次特效」。
	// 怪物的這個值與 Style 是同一格，這裡拆成兩個欄位表達兩種語意。
	RaceOrElement int
}

// Alive 回報單位是否還在場上。
func (u *Unit) Alive() bool { return u.HP > 0 && u.Status != StatusDead }

// TurnOrder 依速度降冪排出這一回合的行動順序，回傳槽位索引。
//
// 三個排除條件（照原版順序）：
//   - 狀態 > 1（束縛以上，含死亡）
//   - X 座標為 0（空槽或已死）
//   - 狀態計數 < 0（暫時失去行動），**且該計數本回合 +1**
//
// **必須是穩定排序。** 原版是氣泡排序且只在嚴格 `<` 時交換，速度相同的單位
// 維持建表順序（＝槽位遞增）。用不穩定排序會讓同速單位的先後與原版不同。
func TurnOrder(units []*Unit) []int {
	var order []int
	for _, u := range units {
		if u == nil {
			continue
		}
		if u.Status > StatusPoison {
			continue
		}
		if u.X == 0 {
			continue
		}
		if u.StatusCount < 0 {
			u.StatusCount++
			continue
		}
		order = append(order, u.Slot)
	}

	bySlot := make(map[int]*Unit, len(units))
	for _, u := range units {
		if u != nil {
			bySlot[u.Slot] = u
		}
	}
	sort.SliceStable(order, func(i, j int) bool {
		return bySlot[order[i]].Speed > bySlot[order[j]].Speed
	})
	return order
}

// AttackResult 記錄一次普通攻擊的結果，方便顯示訊息與測試。
type AttackResult struct {
	Hit      bool
	Critical bool
	// Damage 是扣除護甲後實際套用的傷害。NoEffect 為 true 時這個值無意義。
	Damage int
	// NoEffect 表示傷害算出來 < 1，**整條扣血路徑被跳過**（不是鉗制為 0）。
	NoEffect bool
	Poisoned bool
	Stunned  bool
	Killed   bool
}

// critThresholdBase 是爆擊門檻的基準值。Roll(100) 嚴格大於門檻才爆擊，
// 所以 90 對應 10%。
const critThresholdBase = 90

// Attack 執行一次普通攻擊（動作 case 5）。
//
// hitModifier 是命中率的各項外部修正（例如目標閃避時傳 −4 × 閃避計數）。
func Attack(r *rng.RNG, attacker, target *Unit, hitModifier int) AttackResult {
	var res AttackResult

	if r.Roll(100) > attacker.Skill*4+hitModifier {
		return res // 落空
	}
	res.Hit = true

	// 爆擊是命中後的第二次獨立擲骰。
	threshold := critThresholdBase
	if attacker.IsPlayer {
		if attacker.Berserking {
			threshold = 75
		}
		if attacker.Style == StyleFencing {
			threshold -= 8
		}
	}
	res.Critical = r.Roll(100) > threshold

	damage := rawDamage(r, attacker, target)
	if res.Critical {
		damage <<= 1
	}
	damage -= target.Armor

	if damage < 1 {
		res.NoEffect = true
		return res
	}

	res.Damage = damage
	target.HP -= damage
	if target.HP <= 0 {
		res.Killed = true
		return res
	}

	// 毒武器：武器索引為負數代表附毒。
	if attacker.WeaponIndex < 0 && r.Roll(100) < 15 && target.Status == StatusNormal {
		target.Status = StatusPoison
		res.Poisoned = true
	}

	// 功夫暈眩。狀態計數為負的單位該回合不能行動，連續命中可以疊加。
	unarmed := attacker.WeaponIndex == 0
	kungfu := attacker.Style == StyleKungFu || attacker.Style == StyleBoth
	if unarmed && attacker.IsPlayer && kungfu &&
		r.Roll(100) <= attacker.Skill*2 && target.Status < StatusBindLow {
		if target.StatusCount < 0 {
			target.StatusCount--
		} else {
			target.StatusCount = -1
		}
		res.Stunned = true
	}

	return res
}

// rawDamage 算出扣護甲、乘爆擊之前的傷害。
func rawDamage(r *rng.RNG, attacker, target *Unit) int {
	idx := attacker.WeaponIndex
	if idx < 0 {
		idx = -idx
	}

	strengthMod := (attacker.Strength - 7) / 2

	// 空手道：傷害改用技巧算，技巧高時可以超過雙手劍。
	if idx == 0 && attacker.IsPlayer &&
		(attacker.Style == StyleKarate || attacker.Style == StyleBoth) {
		skillDie := attacker.Skill - 5
		if skillDie < 0 {
			skillDie = -skillDie
		}
		return r.Roll(skillDie) + strengthMod
	}

	// 索引 13 是天生攻擊特例：用力量的一半當骰面，且**不加力量修正**。
	if idx == 13 {
		return r.Roll(attacker.Strength / 2)
	}

	damage := r.Roll(WeaponDamageDie(idx)) + strengthMod
	if attacker.IsPlayer {
		damage += attacker.WeaponEffect + attacker.EnchantBonus
		if target.RaceOrElement == 7 || target.RaceOrElement == 10 {
			damage += attacker.WeaponEffect
		}
	}
	return damage
}

// TurnUndead 執行驅散不死（動作 case 8）。成功則目標即死。
func TurnUndead(r *rng.RNG, caster, target *Unit) bool {
	rate := (18*(caster.Intellect-caster.Level) + 18) / 5
	if r.Roll(100) > rate {
		return false
	}
	target.HP = 0
	return true
}

// Dodge 執行閃避（動作 case 9）。actionPoints 是當前行動點。
//
// 回傳的計數存進單位；命中率修正是它的 −4 倍。
func Dodge(u *Unit, actionPoints int) int {
	u.StatusCount = actionPoints / 3
	return u.StatusCount
}

// DodgeHitModifier 把閃避計數換算成命中率修正。
func DodgeHitModifier(dodgeCount int) int { return dodgeCount * -4 }

// PrayInitialChance 是祈禱的初始成功率。
//
// 手冊記載 20%，**初始化點也已找到**：神殿那一支（`DEMON.INT 0x1c54f`）
// 在寫入信奉的神祇（角色記錄 `+0xf0`）的同一段裡把 `+0xeb` 設成 0x14 = 20。
// 所以 20 不只是手冊數字，是原版寫進存檔的初值。
// 配合「成功後 −5」，成功率序列是 20 → 15 → 10 → 5 → 0。
// 見 docs/re/27 §4。
const PrayInitialChance = 20

// Pray 執行祈禱（動作 case 12）。
//
// 成功與否都不改 chance 以外的狀態；chance 在**成功時**永久 −5。
// 回傳成功與否，以及更新後的 chance。
func Pray(r *rng.RNG, chance int) (bool, int) {
	if r.Roll(100) > chance {
		return false, chance
	}
	return true, chance - 5
}

// Leech 執行汲取法力（動作 case 13）。成功則目標損失當前 SP 的一半。
func Leech(r *rng.RNG, caster *Unit, targetSP int) (bool, int) {
	if r.Roll(100) > 2*caster.Intellect {
		return false, targetSP
	}
	return true, targetSP - targetSP/2
}
