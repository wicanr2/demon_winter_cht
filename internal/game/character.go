package game

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 成長與分配的上限。HP 與 SP 封頂**不同**，要分開。
const (
	maxHPCap = 255
	maxSPCap = 200

	// 屬性的下限。建角時算出低於這個值就鉗制。
	traitFloor = 3

	// 人類每項屬性的額外加成。手冊記人類「優缺點：無」，但程式碼確實有給。
	humanTraitBonus = 2

	// 三次重擲的機會數與觸發門檻。來源是軟體世界中文說明書 p.10，
	// **尚未在程式碼中定位**，見 docs/spec/05-character.md 未解表。
	rerollChances  = 3
	rerollBelow    = 6
	numTraitsRolls = 5
)

// Character 是規則層看到的角色。
//
// 與 scenario.Character（存檔的位元組檢視）分開：那邊管「檔案長什麼樣」，
// 這邊管「規則怎麼算」。
type Character struct {
	Name  string
	Race  gamedata.Race
	Class gamedata.Class
	Level int

	// Traits 是五項屬性的天生值，索引用 gamedata.Trait。
	Traits [gamedata.NumTraits]int

	Experience int

	MaxHP, CurrentHP int
	MaxSP, CurrentSP int

	// Skills 是已學技能，索引即遊戲內部技能 id。
	Skills [gamedata.NumSkills]bool

	// Inventory 是 10 格裝備／道具。
	Inventory [InventorySlots]scenario.InventorySlot
	// Status 是戰鬥狀態（存檔 +0x102）。中毒的人睡覺會掉血，見 Rest。
	Status scenario.CombatStatus

	// EquippedWeapon／EquippedArmor 是目前裝備的那一格的索引。
	//
	// **待複核**：兩個欄位是反組譯推得（存檔 +0x100／+0x101），
	// 尚未做 DOSBox 動態複核。
	EquippedWeapon int
	EquippedArmor  int

	// PrayChance 是呼喚神祇的成功率（存檔 +0xeb，百分比）。
	// Deity 是信奉的神祇編號（+0xf0），0 代表沒有信仰。
	// BindLevel 是束縛效果的等級（+0xec），治療所解束縛依它計價。
	//
	// 三個都是**每個角色各自一份**的持久欄位 —— 祈禱成功率一度被實作成
	// 整隊共用一個值，那是錯的（見 docs/re/19 §3.3）。
	PrayChance int
	Deity      int
	BindLevel  int
}

// InventorySlots 是每個角色的道具欄格數。
const InventorySlots = 10

// Weapon 回傳目前裝備的武器那一格。沒裝備時回傳空槽。
func (c *Character) Weapon() scenario.InventorySlot {
	return c.slot(c.EquippedWeapon)
}

// Armor 回傳目前裝備的護甲那一格。
func (c *Character) Armor() scenario.InventorySlot {
	return c.slot(c.EquippedArmor)
}

func (c *Character) slot(i int) scenario.InventorySlot {
	if i < 0 || i >= InventorySlots {
		return scenario.InventorySlot{Type: 0xff}
	}
	return c.Inventory[i]
}

// FromSave 把存檔解出來的角色轉成規則層的表示。
func FromSave(c scenario.Character) Character {
	out := Character{
		Name:       c.Name,
		Race:       gamedata.Race(c.RaceByte),
		Class:      gamedata.Class(c.ClassByte),
		Level:      int(c.Level),
		Experience: c.Experience,
		MaxHP:      int(c.MaxHP),
		CurrentHP:  int(c.CurrentHP),
		MaxSP:      int(c.MaxSPBonus),
		CurrentSP:  int(c.CurrentSP),
		Status:     c.CombatStatus,
	}
	out.EquippedWeapon = int(c.WeaponSlotIndex)
	out.EquippedArmor = int(c.ArmorSlotIndex)
	out.PrayChance = int(c.PrayChance)
	out.Deity = int(c.Deity)
	out.BindLevel = int(c.BindLevel)
	out.Inventory = c.Inventory

	out.Traits[gamedata.Speed] = int(c.SpeedNatural)
	out.Traits[gamedata.Strength] = int(c.StrengthNatural)
	out.Traits[gamedata.Intellect] = int(c.Intellect)
	out.Traits[gamedata.Endurance] = int(c.Endurance)
	out.Traits[gamedata.Skill] = int(c.SkillNatural)

	for i, on := range c.SkillFlags {
		if i < gamedata.NumSkills {
			out.Skills[i] = on == 1
		}
	}
	return out
}

// HasSkill 回報是否已學某項技能。
func (c *Character) HasSkill(s gamedata.SkillID) bool {
	if s < 0 || int(s) >= gamedata.NumSkills {
		return false
	}
	return c.Skills[s]
}

// RemainingSkillPoints 回傳還能用來買技能的智慧點數。
//
//	剩餘 = 智慧 − Σ(已學技能各自的學費)
//
// 也就是智慧本身就是技能點總量。學院教技能與神殿改宗都用這個值當門檻。
func (c *Character) RemainingSkillPoints(t *gamedata.Tables) (int, error) {
	remaining := c.Traits[gamedata.Intellect]
	for s := 0; s < gamedata.NumSkills; s++ {
		if !c.Skills[s] {
			continue
		}
		cost, err := t.SkillCost(gamedata.SkillID(s), c.Class)
		if err != nil {
			return 0, fmt.Errorf("角色 %s 技能 %d: %w", c.Name, s, err)
		}
		remaining -= cost
	}
	return remaining, nil
}

// TraitSum 回傳五項屬性的總和。
func (c *Character) TraitSum() int {
	sum := 0
	for _, v := range c.Traits {
		sum += v
	}
	return sum
}

// RollTraits 依種族擲出五項屬性的初始值。
//
//	屬性 = 基礎骰 + 種族修正
//	若 屬性 < 3 → 3            ← 鉗制在人類 +2 之前
//	若 種族 == 人類 → 屬性 += 2
//
// **鉗制順序要注意**：下限是 3，但人類因為後套 +2，實際下限是 5。
//
// 基礎骰與種族無關，期望值精確為 8。**推定是 Roll(15)** —— 依據是
// `E[Roll(n)] = (n+1)/2 = 8` 只有 n=15 成立，且觀察到的最大值也是 15。
// 沒有找到實際呼叫該骰子的指令，標假設，見 docs/spec/05-character.md。
func RollTraits(r *rng.RNG, t *gamedata.Tables, race gamedata.Race) ([gamedata.NumTraits]int, error) {
	var out [gamedata.NumTraits]int
	for i := 0; i < gamedata.NumTraits; i++ {
		mod, err := t.RaceModifier(race, gamedata.Trait(i))
		if err != nil {
			return out, err
		}
		v := r.Roll(traitDie) + r.Roll(traitDie) + traitRollBonus + mod
		if v < traitFloor {
			v = traitFloor
		}
		if race == gamedata.Human {
			v += humanTraitBonus
		}
		out[i] = v
	}
	return out, nil
}

// 建角擲點的骰子。**已從原版反組譯確認**（DEMON.INT 0x13bba–0x13c0d）：
//
//	13bba  mov ax,6 / call rnd     ; 第一顆 d6
//	13bc5  mov ax,6 / call rnd     ; 第二顆 d6
//	13bd2  add ax,cx               ; 兩顆相加
//	13bdb  call 106a:026c          ; 取種族修正
//	13be6  add ax,cx               ; + 種族修正
//	13be8  inc ax                  ; **+1**
//	13bec  cmp ax,3 / jge          ; 下限 3
//	13c02  cmp es:[bx+si+0xf5],0   ; 種族 == 人類？
//	13c0a  inc / inc               ; 人類 +2
//
// 所以公式是 `max(3, 2d6 + 種族修正 + 1)`，人類再 +2。
//
// **這裡原本寫的是 `Roll(15)`，標著「假設值」——期望值剛好也是 8，
// 所以「該種族平均」那一欄一直是對的，錯的是分布**：均勻 1–15 會擲出
// 原版不可能出現的 1、2、14、15，而且沒有 2d6 往中間集中的特性。
// 平均值對不代表分布對，這種錯在畫面上完全看不出來。
const (
	traitDie       = 6
	traitRollBonus = 1
)

// LevelUpResult 記錄一次升級的成長量，方便呼叫端顯示與測試。
type LevelUpResult struct {
	HPGain int
	SPGain int
	// TraitGains 是三點分配後每項屬性各加了多少。
	TraitGains [gamedata.NumTraits]int
	// Skipped 為 true 表示屬性總和已達種族上限總和，整個分配被跳過。
	Skipped bool
}

// LevelUp 讓角色升一級，並套用 HP／SP 成長與三點屬性分配。
//
// 呼叫端要先自行檢查經驗值是否達門檻 —— 這個函式只管成長，不管門檻。
//
// **升級不回血也不回魔**：目前 HP／SP 完全不被觸碰。這與一般 CRPG 直覺不同，
// 但要照做。
func LevelUp(r *rng.RNG, t *gamedata.Tables, c *Character) (LevelUpResult, error) {
	var res LevelUpResult
	c.Level++

	// HP：N = 耐力 × 10 / 17 + 1，擲兩次取較大。
	n := c.Traits[gamedata.Endurance]*10/17 + 1
	res.HPGain = maxRoll(r, n)
	c.MaxHP = min(maxHPCap, c.MaxHP+res.HPGain)

	// SP：N = 智慧 / 2 + 1，同樣擲兩次取較大。
	n = c.Traits[gamedata.Intellect]/2 + 1
	res.SPGain = maxRoll(r, n)
	c.MaxSP = min(maxSPCap, c.MaxSP+res.SPGain)

	gains, skipped, err := allocateTraitPoints(r, t, c)
	if err != nil {
		return res, err
	}
	res.TraitGains = gains
	res.Skipped = skipped
	return res, nil
}

// maxRoll 擲兩次 Roll(n) 取較大值。
func maxRoll(r *rng.RNG, n int) int {
	a, b := r.Roll(n), r.Roll(n)
	if a > b {
		return a
	}
	return b
}

// allocateTraitPoints 分配升級的 3 點屬性。
//
// 已滿的屬性會重骰換一項；但**五項總和已達種族上限總和時整個分配跳過**，
// 否則那個「已滿則重骰」的迴圈會永遠轉不出來。
func allocateTraitPoints(r *rng.RNG, t *gamedata.Tables, c *Character) ([gamedata.NumTraits]int, bool, error) {
	var gains [gamedata.NumTraits]int

	capSum := 0
	var caps [gamedata.NumTraits]int
	for i := 0; i < gamedata.NumTraits; i++ {
		m, err := t.RaceMax(c.Race, gamedata.Trait(i))
		if err != nil {
			return gains, false, err
		}
		caps[i] = m
		capSum += m
	}
	if c.TraitSum() >= capSum {
		return gains, true, nil
	}

	for k := 0; k < 3; k++ {
		var idx int
		for {
			idx = r.Roll(numTraitsRolls) - 1
			if c.Traits[idx] != caps[idx] {
				break
			}
		}
		c.Traits[idx]++
		gains[idx]++
	}
	return gains, false, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ApplyTo 把規則層的角色狀態寫回存檔記錄。
//
// **只覆蓋 FromSave 讀進來的那些欄位。** 存檔還有一大片未解區域
// （道具槽的部分 byte、Unknown103 等），那些一律留原值 ——
// 規則層根本不知道它們代表什麼，寫進去等於亂猜。
//
// 這是 FromSave 的反向操作，兩者必須成對維護：FromSave 多讀一個欄位，
// 這裡就要多寫一個，否則那個欄位會在存檔時悄悄退回舊值。
func (c Character) ApplyTo(rec *scenario.Character) {
	rec.Inventory = c.Inventory
	rec.WeaponSlotIndex = slotIndexByte(c.EquippedWeapon)
	rec.ArmorSlotIndex = slotIndexByte(c.EquippedArmor)
	rec.CombatStatus = c.Status
	rec.PrayChance = byte(c.PrayChance)
	rec.Deity = byte(c.Deity)
	rec.BindLevel = byte(c.BindLevel)

	rec.Name = c.Name
	rec.RaceByte = byte(c.Race)
	rec.ClassByte = byte(c.Class)
	rec.Level = byte(c.Level)
	rec.Experience = CapValue(c.Experience)
	rec.MaxHP = byte(c.MaxHP)
	rec.CurrentHP = byte(c.CurrentHP)
	rec.MaxSPBonus = byte(c.MaxSP)
	rec.CurrentSP = byte(c.CurrentSP)

	rec.SpeedNatural = byte(c.Traits[gamedata.Speed])
	rec.StrengthNatural = byte(c.Traits[gamedata.Strength])
	rec.Intellect = byte(c.Traits[gamedata.Intellect])
	rec.Endurance = byte(c.Traits[gamedata.Endurance])
	rec.SkillNatural = byte(c.Traits[gamedata.Skill])

	for i := range rec.SkillFlags {
		if i >= gamedata.NumSkills {
			break
		}
		if c.Skills[i] {
			rec.SkillFlags[i] = 1
		} else {
			rec.SkillFlags[i] = 0
		}
	}
}

// slotIndexByte 把裝備槽索引寫回存檔用的 byte。
//
// 存檔用 0xFF 表示「沒有裝備」（原版 Stumpy 的護甲欄就是 255）。
// 規則層另有 −1 這種寫法（測試夾具、還沒配裝的新角色），一併收斂成 0xFF。
func slotIndexByte(i int) byte {
	if i < 0 || i >= InventorySlots {
		return 0xff
	}
	return byte(i)
}

// 裝備換算成戰鬥數值。
//
// 兩張表都不在 `ITEMS.DAT` 裡：
//   - 武器骰表內嵌在 `DEMON.INT`（`31f0:1785`），索引是 `ITEMS.DAT 索引 + 1`
//     （戰鬥碼直接這樣用，見 docs/formats/game-data-tables.md §1.3）
//   - 護甲防護值是手冊記載的 1–5，對應 `ITEMS.DAT` 索引 8–12 的五件護甲
//     （布甲 1 … 板甲 5），所以防護值 = 索引 − 7
const (
	// armorFirstIndex／armorLastIndex 是五件護甲在 ITEMS.DAT 的索引範圍。
	armorFirstIndex = 8
	armorLastIndex  = 12
	// armorRatingBase 讓「索引 − base」等於防護值。
	armorRatingBase = armorFirstIndex - 1
)

// WeaponDieIndex 回傳角色目前武器在傷害骰表裡的索引。
//
// 沒裝備武器時回 0（徒手），骰面 2 —— 這是表裡刻意留的那一格，
// 不是 padding。
func (c *Character) WeaponDieIndex() int {
	w := c.Weapon()
	if w.Empty() || int(w.Type) >= armorFirstIndex {
		return 0
	}
	return int(w.Type) + 1
}

// ArmorRating 回傳角色目前的護甲防護值。沒穿護甲回 0。
func (c *Character) ArmorRating() int {
	a := c.Armor()
	if a.Empty() || int(a.Type) < armorFirstIndex || int(a.Type) > armorLastIndex {
		return 0
	}
	return int(a.Type) - armorRatingBase
}

// CombatUnit 依角色目前的狀態與裝備建一個戰鬥單位。
//
// **裝備一定要帶進來。** 少帶的話角色會空手、零護甲上場 ——
// 戰鬥數字全部偏掉，但畫面上看不出哪裡不對。
func (c *Character) CombatUnit(slot, x, y int, facing Facing) *Unit {
	w := c.Weapon()
	return &Unit{
		Slot: slot, Name: c.Name,
		X: x, Y: y, Facing: int(facing),
		Speed:        c.Traits[gamedata.Speed],
		Strength:     c.Traits[gamedata.Strength],
		Skill:        c.Traits[gamedata.Skill],
		Intellect:    c.Traits[gamedata.Intellect],
		Level:        c.Level,
		HP:           c.CurrentHP,
		MaxHP:        c.MaxHP,
		MaxSP:        c.MaxSP,
		CurrentSP:    c.CurrentSP,
		Armor:        c.ArmorRating(),
		WeaponIndex:  c.WeaponDieIndex(),
		WeaponEffect: w.WeaponEffect,
		EnchantBonus: w.Enchant,
		IsPlayer:     true,
		Berserking:   c.HasSkill(gamedata.SkillBerserking),
		Style: StyleFor(c.WeaponDieIndex(), func(s gamedata.SkillID) bool {
			return c.HasSkill(s)
		}),
	}
}
