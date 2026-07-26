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

	// 0x0a8 起 176 bytes 是戰場視線遮蔽表，見 sight.go。
	// （起點是 0x0a8 不是 0x0a5：可通行性那一段在 arena 裡佔 104 bytes，
	// 只有前 101 個是有效 tile。）

	offSkillCost = 0x158 // 31 技能 × 10 職業，1 byte／格
	offRaceMax   = 0x422 // 5 種族 × 6 words
	offSummon    = 0x7c6 // 12 筆 × 22 bytes

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

// Class 職業 id，與角色記錄 +0xf6 的值相同，也是技能學費表（0x158）的欄索引。
type Class int

const (
	Ranger Class = iota
	Paladin
	Barbarian
	Monk
	Cleric
	Thief
	Wizard
	Sorcerer
	Visionary
	Scholar
)

// NumClasses 已在下方以表格維度定義，這裡的常數順序必須與它一致。

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
	spells      [numSpells]Spell
	deitySpell  [NumDeities]int
	sight       *SightShadow

	// terrainGroups／encounters 是隨機遭遇的兩張表，見 encounter.go。
	terrainGroups []byte
	encounters    []EncounterGroup
}

// Terrain 回傳某個 tile 屬於哪一種地形。
//
// **地形就是可通行性表的值** —— 原版沒有另一張地形表（見 encounter.go）。
// 不可通行的 tile（0xfd／0xfe／0xff）沒有地形，ok 回 false。
func (t *Tables) Terrain(tile byte) (Terrain, bool) {
	p := t.Passability(tile)
	if byte(p) >= NumTerrains {
		return 0, false
	}
	return Terrain(p), true
}

// Sight 回傳戰場視線遮蔽表（見 sight.go）。
func (t *Tables) Sight() *SightShadow { return t.sight }

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

	sight, err := parseSightShadow(data)
	if err != nil {
		return nil, err
	}
	t.sight = sight
	t.terrainGroups, t.encounters = parseEncounters(data)

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

	for i := 0; i < NumDeities; i++ {
		t.deitySpell[i] = int(int16(binary.LittleEndian.Uint16(data[offDeitySpell+i*2:])))
	}

	for i := 0; i < numSpells; i++ {
		base := offSpells + i*10
		w := func(k int) int {
			return int(int16(binary.LittleEndian.Uint16(data[base+k*2:])))
		}
		t.spells[i] = Spell{School: w(0), Effect: w(1), K: w(2), M: w(3), W4: w(4)}
	}

	return t, nil
}

// DeitySpell 回傳這位神祇賜予的法術 id。**deity 是 1-based**（存檔
// `char+0xf0` 的值），回傳 −1 代表這位神祇不賜法術。
func (t *Tables) DeitySpell(deity int) (int, error) {
	if deity < 1 || deity > NumDeities {
		return 0, fmt.Errorf("神祇編號 %d 超出 1..%d", deity, NumDeities)
	}
	return t.deitySpell[deity-1], nil
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

// 法術參數表：FILES.DAT offset 0x45e–0x60c，43 筆 × 10 bytes。
//
// **不在 DEMON.INT 裡** —— 這是先前一直找不到的原因。
// 索引 0..42 與 FILES.DTT 的「名稱 + 訊息」字串順序 1:1。
const (
	offSpells = 0x45e

	// offDeitySpell/NumDeities：**每位神祇賜予的法術 id**（11 個 word，
	// −1 = 這位不賜法術）。紮營選單的 Worship 拿它查要放哪一個法術。
	//
	// 位置是從資源競技場的指標鏈推出來的（`docs/re/43`）：
	//
	//	[0x4e28] = [0x5510] + 0x3c      ; 法術表
	//	[0x5300] = [0x4e28] + 0x1ae     ; 43 法術 × 10 bytes = 0x1ae
	//	[0x4cc4] = [0x5300] + 0x1a4     ; 30 道具 × 14 bytes = 0x1a4
	//	[0x51d4] = [0x4cc4] + 0x16      ; 11 word = 0x16
	//
	// 反推 `[0x5510]` = 0x422（種族上限表），一路加下來就是 0x7ae。
	// 每一段的長度都與已知的表大小分毫不差 —— 這條鏈自己就把位置釘死了。
	offDeitySpell = 0x7ae
	NumDeities    = 11
	numSpells     = 43
)

// Spell 是法術參數表的一筆記錄。五個 signed 16-bit word。
type Spell struct {
	// School 是所屬符文系 id。束縛狀態欄存的就是這個值，
	// 解除判定要靠它比對系別。
	School int
	// Effect 決定效果類型（套用到哪個屬性欄位，或走哪條特殊判定）。
	Effect int
	// K 是效果強度係數，**正負決定增益或傷害方向**。
	K int
	// M 是最低施法 SP，也是通式裡的分母。
	// 有程式碼直證：FUN_1000_11e5 裡 `if (SP < M) → "not enough points"`。
	M int
	// W4 語意未解，保留原值。
	W4 int
}

// Empty 回報這是不是一筆空記錄（表中有幾筆全零的佔位）。
func (s Spell) Empty() bool { return s.School == 0 && s.Effect == 0 && s.M == 0 }

// Spell 取回第 i 筆法術參數。
func (t *Tables) Spell(i int) (Spell, error) {
	if i < 0 || i >= numSpells {
		return Spell{}, fmt.Errorf("法術索引 %d 超出範圍 0..%d", i, numSpells-1)
	}
	return t.spells[i], nil
}

// NumSpells 回傳法術表的筆數。
func (t *Tables) NumSpells() int { return numSpells }
