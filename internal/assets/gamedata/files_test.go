package gamedata

import (
	"path/filepath"
	"testing"
)

func loadTestTables(t *testing.T) *Tables {
	t.Helper()
	tb, err := LoadTables(filepath.Join(origDataDir(t), "FILES.DAT"))
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	return tb
}

// manualSkillCosts 是手冊附錄 A「SKILL POINT COSTS BY CLASS」的完整內容，
// 依英文技能名字母序抄錄，欄位順序為職業 1..10
// （流浪漢/武士/野蠻人/和尚/牧師/賊/巫師/魔法師/幻想者/學究）。
//
// 兩份獨立紙本來源（SSI 英文原版手冊、軟體世界 1990 中文說明書）在訂正後一致，
// 見 docs/manual-cht/02-appendix.md。這張表是 SkillID 常數順序的驗收依據。
var manualSkillCosts = map[SkillID][NumClasses]byte{
	SkillAxe:         {3, 5, 1, 6, 10, 3, 9, 8, 8, 8},
	SkillBarkskin:    {5, 6, 4, 6, 10, 10, 10, 10, 10, 10},
	SkillBerserking:  {4, 6, 2, 4, 5, 4, 9, 9, 8, 8},
	SkillDetectTraps: {3, 4, 3, 3, 4, 1, 6, 6, 5, 6},
	SkillDisarmTraps: {6, 7, 7, 7, 8, 3, 9, 9, 9, 8},
	SkillFencing:     {4, 3, 5, 10, 8, 6, 10, 10, 7, 7},
	SkillFireRunes:   {10, 10, 10, 9, 8, 10, 5, 10, 10, 9},
	SkillHunting:     {1, 4, 2, 4, 5, 4, 7, 9, 7, 6},
	SkillIceRunes:    {10, 10, 6, 9, 8, 10, 4, 10, 9, 9},
	SkillIllusion:    {9, 9, 10, 8, 7, 8, 10, 3, 5, 9},
	SkillItemLore:    {9, 10, 10, 6, 10, 10, 6, 7, 7, 4},
	SkillKarate:      {3, 6, 5, 1, 3, 2, 6, 6, 4, 4},
	SkillKungFu:      {6, 8, 8, 3, 5, 5, 8, 8, 7, 7},
	SkillMace:        {2, 4, 1, 4, 2, 1, 6, 6, 5, 5},
	SkillMetalRunes:  {10, 10, 10, 9, 8, 10, 4, 10, 9, 9},
	SkillMonsterLore: {2, 4, 5, 4, 5, 5, 5, 5, 4, 2},
	SkillPersuasion:  {4, 2, 7, 5, 2, 4, 5, 7, 6, 7},
	SkillPossession:  {10, 10, 10, 9, 8, 10, 10, 5, 7, 10},
	SkillPotionLore:  {8, 10, 10, 4, 8, 10, 3, 4, 8, 2},
	SkillPriesthood:  {5, 2, 9, 4, 3, 9, 5, 7, 4, 8},
	SkillShaman:      {5, 8, 5, 4, 3, 9, 7, 5, 4, 8},
	SkillSpiritRunes: {10, 7, 10, 9, 6, 10, 4, 10, 7, 9},
	SkillSummoning:   {10, 10, 10, 10, 9, 10, 10, 5, 8, 10},
	SkillSword:       {3, 2, 4, 6, 10, 3, 8, 7, 6, 6},
	SkillTactics:     {3, 2, 5, 4, 4, 4, 6, 6, 4, 1},
	SkillViewItem:    {10, 10, 10, 10, 8, 10, 10, 10, 2, 8},
	SkillViewLand:    {10, 10, 10, 10, 10, 10, 10, 10, 3, 7},
	SkillViewMind:    {9, 2, 10, 8, 4, 5, 8, 8, 2, 10},
	SkillViewRoom:    {10, 10, 10, 9, 10, 9, 10, 10, 4, 10},
	SkillWeaponLore:  {7, 7, 7, 8, 10, 7, 9, 9, 8, 3},
	SkillWindRunes:   {6, 10, 10, 9, 8, 8, 5, 10, 10, 9},
}

// 手冊附錄 A 全部 310 格必須與 FILES.DAT 相符。
// 這同時驗收 SkillID 常數的排列順序 —— 排錯任何一項這裡就會紅。
func TestSkillCost_MatchesManualAppendixA(t *testing.T) {
	tb := loadTestTables(t)

	if len(manualSkillCosts) != NumSkills {
		t.Fatalf("手冊表應有 %d 個技能，實際 %d", NumSkills, len(manualSkillCosts))
	}

	for skill, want := range manualSkillCosts {
		for c := 0; c < NumClasses; c++ {
			got, err := tb.SkillCost(skill, Class(c))
			if err != nil {
				t.Fatalf("SkillCost(%d, %d): %v", skill, c, err)
			}
			if got != int(want[c]) {
				t.Errorf("技能 id %d × 職業 %d：FILES.DAT = %d，手冊 = %d",
					skill, c+1, got, want[c])
			}
		}
	}
}

// 每一列的十元組都不重複，「手冊某技能 ↔ 程式某列」的對應才是唯一的。
// 這是上面那個測試能當成順序驗收的前提，明確測出來。
func TestSkillCost_RowsAreUnique(t *testing.T) {
	tb := loadTestTables(t)

	seen := make(map[[NumClasses]byte]SkillID, NumSkills)
	for s := 0; s < NumSkills; s++ {
		var row [NumClasses]byte
		for c := 0; c < NumClasses; c++ {
			v, err := tb.SkillCost(SkillID(s), Class(c))
			if err != nil {
				t.Fatalf("SkillCost: %v", err)
			}
			row[c] = byte(v)
		}
		if prev, dup := seen[row]; dup {
			t.Errorf("技能 id %d 與 %d 的成本列完全相同 %v，指紋比對會有歧義", s, prev, row)
		}
		seen[row] = SkillID(s)
	}
}

// 手冊附錄 B「各人種屬性的最大值」25 格。
func TestRaceMax_MatchesManualAppendixB(t *testing.T) {
	tb := loadTestTables(t)

	want := [NumRaces][NumTraits]int{
		Human:   {20, 24, 32, 22, 21},
		Elf:     {20, 15, 40, 15, 20},
		Dwarf:   {15, 30, 24, 25, 22},
		DarkElf: {22, 14, 40, 15, 17},
		Troll:   {14, 24, 20, 30, 18},
	}

	for r := 0; r < NumRaces; r++ {
		for tr := 0; tr < NumTraits; tr++ {
			got, err := tb.RaceMax(Race(r), Trait(tr))
			if err != nil {
				t.Fatalf("RaceMax(%d,%d): %v", r, tr, err)
			}
			if got != want[r][tr] {
				t.Errorf("種族 %d 屬性 %d：得到 %d，手冊 %d", r, tr, got, want[r][tr])
			}
		}
	}
}

// 種族修正表。手冊附錄的巨魔那一列有誤（記「力量 −1、無速度修正」），
// 這裡以公式推導值為準 —— 依 docs/spec/05-character.md，
// DOSBox 實測畫面與公式吻合，手冊錯。
func TestRaceModifier_DerivedTable(t *testing.T) {
	tb := loadTestTables(t)

	want := [NumRaces][NumTraits]int{
		Human:   {0, 0, 0, 0, 0},
		Elf:     {0, -2, 2, -1, 0},
		Dwarf:   {-1, 1, -2, 0, 0},
		DarkElf: {0, -2, 2, -1, -1},
		Troll:   {-1, 0, -3, 2, 0},
	}

	for r := 0; r < NumRaces; r++ {
		for tr := 0; tr < NumTraits; tr++ {
			got, err := tb.RaceModifier(Race(r), Trait(tr))
			if err != nil {
				t.Fatalf("RaceModifier(%d,%d): %v", r, tr, err)
			}
			if got != want[r][tr] {
				t.Errorf("種族 %d 屬性 %d 修正：得到 %d，預期 %d", r, tr, got, want[r][tr])
			}
		}
	}
}

func TestRaceBonusSkill(t *testing.T) {
	tb := loadTestTables(t)

	want := map[Race]SkillID{
		Human:   0,
		Elf:     SkillDetectAura,
		Dwarf:   SkillDarkVision,
		DarkElf: SkillPowerLeech,
		Troll:   SkillRegeneration,
	}

	for r, exp := range want {
		got, err := tb.RaceBonusSkill(r)
		if err != nil {
			t.Fatalf("RaceBonusSkill(%d): %v", r, err)
		}
		if got != exp {
			t.Errorf("種族 %d 附贈技能：得到 %d，預期 %d", r, got, exp)
		}
	}
}

// 手冊附錄 D「CHANTS」12 筆，幻術／召喚成本。
// 順序為 Coyote、僵屍、棕熊、小龍、食人巨妖、邪靈、火魔、火/土/風/冰/靈魂使者。
func TestSummonCosts_MatchesManualAppendixD(t *testing.T) {
	tb := loadTestTables(t)

	want := []struct {
		name      string
		illusion  int
		summoning int
	}{
		{"Coyote", 2, 4},
		{"僵屍", 4, 8},
		{"棕熊", 6, 12},
		{"小龍", 8, 16},
		{"食人巨妖", 10, 20},
		{"邪靈", 14, 28},
		{"火魔", 18, 36},
		{"火使者", 20, 40},
		{"土使者", 20, 40},
		{"風使者", 20, 40},
		{"冰使者", 20, 40},
		{"靈魂使者", 20, 40},
	}

	if tb.NumSummons() != len(want) {
		t.Fatalf("召喚表筆數：得到 %d，手冊 %d", tb.NumSummons(), len(want))
	}

	for i, w := range want {
		e, err := tb.Summon(i)
		if err != nil {
			t.Fatalf("Summon(%d): %v", i, err)
		}
		if got := e.IllusionCost(); got != w.illusion {
			t.Errorf("%s 幻術成本：得到 %d，手冊 %d", w.name, got, w.illusion)
		}
		if got := e.SummonCost(); got != w.summoning {
			t.Errorf("%s 召喚成本：得到 %d，手冊 %d", w.name, got, w.summoning)
		}
	}
}

// 可通行性表：值域與邊界。
func TestPassability_RangeAndBounds(t *testing.T) {
	tb := loadTestTables(t)

	// 表只涵蓋 tile 0..100，超出一律視為不可通行。
	if !tb.Passability(numTiles).Blocked() {
		t.Errorf("tile %d 超出表範圍，應視為 blocked", numTiles)
	}
	if !tb.Passability(127).Blocked() {
		t.Error("tile 127（遮罩後理論上限）超出表範圍，應視為 blocked")
	}

	// 觀察到的值域：0–7 與 0xfd/0xfe/0xff。出現其他值代表偏移或長度判斷有誤。
	for tile := 0; tile < numTiles; tile++ {
		v := tb.Passability(byte(tile)).Raw()
		lowRange := v <= 7
		highRange := v == 0xfd || v == 0xfe || v == 0xff
		if !lowRange && !highRange {
			t.Errorf("tile %d 的可通行性值 0x%02x 不在已知值域（0–7 或 0xfd/0xfe/0xff）", tile, v)
		}
	}
}

func TestParseTables_WrongSize(t *testing.T) {
	if _, err := ParseTables(make([]byte, 100)); err == nil {
		t.Error("長度不符時應回傳錯誤")
	}
}

func TestTables_OutOfRangeArgs(t *testing.T) {
	tb := loadTestTables(t)

	if _, err := tb.SkillCost(SkillID(NumSkills), 0); err == nil {
		t.Error("技能 id 超出範圍應回傳錯誤")
	}
	if _, err := tb.SkillCost(0, Class(NumClasses)); err == nil {
		t.Error("職業 id 超出範圍應回傳錯誤")
	}
	if _, err := tb.RaceMax(Race(NumRaces), 0); err == nil {
		t.Error("種族 id 超出範圍應回傳錯誤")
	}
	if _, err := tb.RaceMax(0, Trait(NumTraits)); err == nil {
		t.Error("屬性 id 超出範圍應回傳錯誤")
	}
	if _, err := tb.Summon(-1); err == nil {
		t.Error("召喚表索引為負應回傳錯誤")
	}
}

// 神祇賜予的法術表：11 個 word，最後一位不賜法術。
//
// 位置是從資源競技場的指標鏈推的（docs/re/43），這條測試同時釘住
// 「長度 11」與「值域落在法術表之內」——鏈算錯一段，兩者都會爆。
func TestTables_DeitySpell(t *testing.T) {
	tb := loadTestTables(t)

	if _, err := tb.DeitySpell(0); err == nil {
		t.Error("神祇 0 不存在，應該回 error")
	}
	if _, err := tb.DeitySpell(NumDeities + 1); err == nil {
		t.Error("超出範圍應該回 error")
	}

	granted := 0
	for d := 1; d <= NumDeities; d++ {
		id, err := tb.DeitySpell(d)
		if err != nil {
			t.Fatalf("神祇 %d: %v", d, err)
		}
		if id == -1 {
			continue
		}
		granted++
		if id < 0 || id >= tb.NumSpells() {
			t.Errorf("神祇 %d 的法術 id %d 不在 0..%d", d, id, tb.NumSpells()-1)
		}
		if sp, err := tb.Spell(id); err != nil || sp.Empty() {
			t.Errorf("神祇 %d 指向空法術 %d", d, id)
		}
	}
	if granted != NumDeities-1 {
		t.Errorf("有 %d 位神祇賜法術，預期 %d 位（只有最後一位不賜）",
			granted, NumDeities-1)
	}
}
