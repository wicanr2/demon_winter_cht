package gamedata

import (
	"encoding/binary"
	"fmt"
	"os"
)

// FILES.DAT 是一個沒有檔頭的固定佈局表集合：四張表硬編碼在固定偏移上，
// 原版程式直接以常數偏移存取。佈局考證見 docs/re/21-skills-races-and-files-dat.md
// 與 docs/re/22-resource-arena-and-passability.md。
const (
	filesDatSize = 0x8ce

	offPassability = 0x040 // 101 bytes，索引 = tile 值 0..100
	offSkillCost   = 0x158 // 31 技能 × 10 職業，1 byte／格
	offRaceMax     = 0x422 // 5 種族 × 6 words
	offSummon      = 0x7c6 // 12 筆 × 22 bytes

	numTiles    = 101
	NumSkills   = 31
	NumClasses  = 10
	NumRaces    = 5
	NumTraits   = 5
	numSummons  = 12
	summonWords = 11
)

// Passability 是可通行性表的一格值。原版把 tile 值（先 & 0x7f）當索引查這張表。
type Passability byte

// Blocked 回報這個 tile 在一般地圖上是否不可通行。
//
// 0xff 是「牆」。0xfd／0xfe 也是高位值，但語意尚未逐一確認
// （見 docs/spec/04-movement.md），因此不併入 Blocked，由呼叫端自行判斷。
func (p Passability) Blocked() bool { return p == 0xff }

// Raw 取回原始位元組，供尚未歸類的值（0xfd／0xfe）判斷用。
func (p Passability) Raw() byte { return byte(p) }

// Trait 是五項屬性的索引。順序與種族上限表、建角擲點的欄位順序一致。
type Trait int

const (
	Speed Trait = iota
	Strength
	Intellect
	Endurance
	Skill
)

// Race 種族 id，與角色記錄 +0xf5 的值相同。
type Race int

const (
	Human Race = iota
	Elf
	Dwarf
	DarkElf
	Troll
)

// Class 職業 id，與角色記錄 +0xf6 的值相同。
type Class int

// SkillID 是遊戲內部的技能編號，也是角色記錄 +0xc8 起技能旗標陣列的索引。
//
// 這個順序與攻略、手冊附錄 A 的排列都不同（那兩者依英文名字母序）。
// 由攻略與手冊兩份獨立成本表分別對 FILES.DAT 指紋比對得出，兩者結果一致。
type SkillID int

const (
	SkillFencing SkillID = iota
	SkillSword
	SkillAxe
	SkillMace
	SkillKarate
	SkillKungFu
	SkillBerserking
	SkillTactics
	SkillHunting
	SkillPersuasion
	SkillDetectTraps
	SkillDisarmTraps
	SkillIllusion
	SkillPossession
	SkillSummoning
	SkillShaman
	SkillPriesthood
	SkillFireRunes
	SkillMetalRunes
	SkillWindRunes
	SkillIceRunes
	SkillSpiritRunes
	SkillWeaponLore
	SkillPotionLore
	SkillItemLore
	SkillMonsterLore
	SkillViewLand
	SkillViewRoom
	SkillViewItem
	SkillViewMind
	SkillBarkskin
)

// 種族天生能力用的偽技能 id。它們接在 0..30 的真技能之後，
// 只出現在種族上限表的第 6 個 word，不佔技能旗標陣列的格子。
const (
	SkillRegeneration SkillID = 31 // 巨魔
	SkillDetectAura   SkillID = 32 // 精靈
	SkillDarkVision   SkillID = 33 // 矮人
	SkillPowerLeech   SkillID = 34 // 黑暗精靈
)

// SummonEntry 是召喚／幻術生物表的一筆記錄。
//
// 22 bytes 中只有部分欄位語意已確認；未確認的以 Word 索引保留原值，
// 不強行命名。考證見 docs/re/20-summon-and-combat-units.md。
type SummonEntry struct {
	words [summonWords]uint16
}

// Word 取回第 i 個 word 的原始值（0..10）。
func (e SummonEntry) Word(i int) uint16 { return e.words[i] }

// PowerBase 是成本公式的基數（word 8）。
// 召喚成本 = PowerBase × 4，幻術成本 = PowerBase × 2。
func (e SummonEntry) PowerBase() int { return int(e.words[8]) }

// SummonCost 回傳召喚（法術 id 0x18）所需的法力點數。
func (e SummonEntry) SummonCost() int { return e.PowerBase() * 4 }

// IllusionCost 回傳幻術（法術 id 0x19）所需的法力點數。
func (e SummonEntry) IllusionCost() int { return e.PowerBase() * 2 }

// Tables 是 FILES.DAT 全部四張表的解碼結果。
//
// 對外只露出查詢方法，檔案偏移與位元組佈局不外洩。
type Tables struct {
	passability [numTiles]Passability
	skillCost   [NumSkills][NumClasses]byte
	raceMax     [NumRaces][NumTraits]int
	raceBonus   [NumRaces]SkillID
	summons     [numSummons]SummonEntry
}

// LoadTables 從 FILES.DAT 解出全部四張表。
func LoadTables(path string) (*Tables, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取 FILES.DAT: %w", err)
	}
	return ParseTables(data)
}

// ParseTables 從 FILES.DAT 的完整內容解出全部四張表。
func ParseTables(data []byte) (*Tables, error) {
	if len(data) != filesDatSize {
		return nil, fmt.Errorf("FILES.DAT 長度應為 %d bytes，實際 %d", filesDatSize, len(data))
	}

	t := &Tables{}

	for i := 0; i < numTiles; i++ {
		t.passability[i] = Passability(data[offPassability+i])
	}

	for s := 0; s < NumSkills; s++ {
		for c := 0; c < NumClasses; c++ {
			t.skillCost[s][c] = data[offSkillCost+s*NumClasses+c]
		}
	}

	for r := 0; r < NumRaces; r++ {
		base := offRaceMax + r*12
		for tr := 0; tr < NumTraits; tr++ {
			t.raceMax[r][tr] = int(binary.LittleEndian.Uint16(data[base+tr*2:]))
		}
		t.raceBonus[r] = SkillID(binary.LittleEndian.Uint16(data[base+NumTraits*2:]))
	}

	for i := 0; i < numSummons; i++ {
		base := offSummon + i*22
		for w := 0; w < summonWords; w++ {
			t.summons[i].words[w] = binary.LittleEndian.Uint16(data[base+w*2:])
		}
	}

	return t, nil
}

// Passability 查 tile 的可通行性。tile 應已遮罩過 &0x7f。
// 超出表範圍（>100）視為未定義，回傳 blocked。
func (t *Tables) Passability(tile byte) Passability {
	if int(tile) >= numTiles {
		return 0xff
	}
	return t.passability[tile]
}

// SkillCost 回傳某職業學某技能要花的智慧點數（值域 1–10）。
func (t *Tables) SkillCost(s SkillID, c Class) (int, error) {
	if s < 0 || int(s) >= NumSkills {
		return 0, fmt.Errorf("技能 id %d 超出範圍 0..%d", s, NumSkills-1)
	}
	if c < 0 || int(c) >= NumClasses {
		return 0, fmt.Errorf("職業 id %d 超出範圍 0..%d", c, NumClasses-1)
	}
	return int(t.skillCost[s][c]), nil
}

// RaceMax 回傳某種族某項屬性的上限。
func (t *Tables) RaceMax(r Race, tr Trait) (int, error) {
	if r < 0 || int(r) >= NumRaces {
		return 0, fmt.Errorf("種族 id %d 超出範圍 0..%d", r, NumRaces-1)
	}
	if tr < 0 || int(tr) >= NumTraits {
		return 0, fmt.Errorf("屬性 id %d 超出範圍 0..%d", tr, NumTraits-1)
	}
	return t.raceMax[r][tr], nil
}

// RaceModifier 回傳某種族某項屬性相對人類的修正值。
//
//	修正 = (種族上限 − 人類上限) / 4      整數除法、向零捨去
//
// 注意人類另有每項 +2 的加成，那個不在本函式內，由建角流程另外套用
// （且要在下限鉗制之後）。見 docs/spec/05-character.md。
func (t *Tables) RaceModifier(r Race, tr Trait) (int, error) {
	rm, err := t.RaceMax(r, tr)
	if err != nil {
		return 0, err
	}
	hm := t.raceMax[Human][tr]
	return (rm - hm) / 4, nil
}

// RaceBonusSkill 回傳種族天生能力的偽技能 id。人類為 0（無）。
func (t *Tables) RaceBonusSkill(r Race) (SkillID, error) {
	if r < 0 || int(r) >= NumRaces {
		return 0, fmt.Errorf("種族 id %d 超出範圍 0..%d", r, NumRaces-1)
	}
	return t.raceBonus[r], nil
}

// Summon 回傳第 i 筆召喚生物記錄。
func (t *Tables) Summon(i int) (SummonEntry, error) {
	if i < 0 || i >= numSummons {
		return SummonEntry{}, fmt.Errorf("召喚表索引 %d 超出範圍 0..%d", i, numSummons-1)
	}
	return t.summons[i], nil
}

// NumSummons 回傳召喚表的筆數。
func (t *Tables) NumSummons() int { return numSummons }
