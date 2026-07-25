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
	}
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
		v := r.Roll(baseTraitDie) + mod
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

// baseTraitDie 是建角基礎骰的面數。**假設值**，見 RollTraits 說明。
const baseTraitDie = 15

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
