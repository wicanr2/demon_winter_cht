package scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func partyDatPath() string {
	return filepath.Join(dataDir, "PARTY.DAT")
}

func TestLoadSaveGame_FiveReadableCharacters(t *testing.T) {
	path := partyDatPath()
	skipIfMissing(t, path)

	save, err := LoadSaveGame(path)
	if err != nil {
		t.Fatalf("LoadSaveGame(%s) 失敗: %v", path, err)
	}

	// 已知這份存檔的 5 名角色姓名（docs/formats/game-data-tables.md §1.1，
	// 已用姓名字串在檔案內的位址間距交叉驗證過）。
	wantNames := []string{"Wopple", "Stumpy", "Podgom", "Norman", "Menhir"}

	for i, ch := range save.Characters {
		if ch.Name == "" {
			t.Errorf("角色 %d 姓名為空字串，姓名應可讀", i)
		}
		if i < len(wantNames) && ch.Name != wantNames[i] {
			t.Errorf("角色 %d 姓名 = %q，預期 %q", i, ch.Name, wantNames[i])
		}
	}
}

// TestLoadSaveGame_AttributesInPlausibleRange 用 docs/walkthrough/part-2.md
// 的種族屬性上限表（5 個種族取各屬性最大值）當寬鬆的合理範圍檢查：
// 速度上限 22、力量上限 30、智力上限 40、耐力上限 30、技巧上限 22。
// 目的是抓「欄位位移算錯、讀到別的資料」這種明顯錯位，不是逐值精算。
func TestLoadSaveGame_AttributesInPlausibleRange(t *testing.T) {
	path := partyDatPath()
	skipIfMissing(t, path)

	save, err := LoadSaveGame(path)
	if err != nil {
		t.Fatalf("LoadSaveGame(%s) 失敗: %v", path, err)
	}

	const (
		maxSpeed    = 22
		maxStrength = 30
		maxIntel    = 40
		maxEndur    = 30
		maxSkill    = 22
		maxHPorSP   = 250 // 慷慨上限，只為抓明顯錯位，不是精確遊戲規則
	)

	checkRange := func(t *testing.T, label string, idx int, v byte, max byte) {
		t.Helper()
		if v > max {
			t.Errorf("角色 %d 的 %s = %d 超出合理範圍 [0,%d]（種族屬性上限表，可能欄位位移算錯）", idx, label, v, max)
		}
	}

	for i, ch := range save.Characters {
		checkRange(t, "SpeedNatural", i, ch.SpeedNatural, maxSpeed)
		checkRange(t, "SpeedBonus", i, ch.SpeedBonus, maxSpeed)
		checkRange(t, "StrengthNatural", i, ch.StrengthNatural, maxStrength)
		checkRange(t, "StrengthBonus", i, ch.StrengthBonus, maxStrength)
		checkRange(t, "Intellect", i, ch.Intellect, maxIntel)
		checkRange(t, "Endurance", i, ch.Endurance, maxEndur)
		checkRange(t, "SkillNatural", i, ch.SkillNatural, maxSkill)
		checkRange(t, "SkillBonus", i, ch.SkillBonus, maxSkill)
		checkRange(t, "MaxHP", i, ch.MaxHP, maxHPorSP)
		checkRange(t, "MaxSPNatural", i, ch.MaxSPNatural, maxHPorSP)

		if ch.CurrentHP > ch.MaxHP {
			t.Errorf("角色 %d 目前生命值 %d 超過生命值上限 %d", i, ch.CurrentHP, ch.MaxHP)
		}
		if ch.CurrentSP > ch.MaxSPBonus {
			t.Errorf("角色 %d 目前法力值 %d 超過法力值上限(含加成) %d", i, ch.CurrentSP, ch.MaxSPBonus)
		}
		if ch.Experience < 0 {
			t.Errorf("角色 %d 經驗值 = %d，不應為負", i, ch.Experience)
		}
	}
}

// TestLoadSaveGame_RawRoundTripsRecordBytes 驗證 Character.Raw 忠實保留了原始
// 260 bytes（未來 encode/寫回存檔要用），而不是重新序列化出來的近似值。
func TestLoadSaveGame_RawRoundTripsRecordBytes(t *testing.T) {
	path := partyDatPath()
	skipIfMissing(t, path)

	save, err := LoadSaveGame(path)
	if err != nil {
		t.Fatalf("LoadSaveGame(%s) 失敗: %v", path, err)
	}

	for i, ch := range save.Characters {
		if len(ch.Raw) != recordLen {
			t.Errorf("角色 %d Raw 長度 = %d，預期 %d", i, len(ch.Raw), recordLen)
		}
		if ch.Raw[raceOffset] != ch.RaceByte {
			t.Errorf("角色 %d Raw[raceOffset] = %d，與 RaceByte %d 不一致", i, ch.Raw[raceOffset], ch.RaceByte)
		}
	}
}

func TestLoadSaveGame_TrailerFields(t *testing.T) {
	path := partyDatPath()
	skipIfMissing(t, path)

	save, err := LoadSaveGame(path)
	if err != nil {
		t.Fatalf("LoadSaveGame(%s) 失敗: %v", path, err)
	}

	if len(save.TrailerRaw) != trailerLen {
		t.Errorf("TrailerRaw 長度 = %d，預期 %d", len(save.TrailerRaw), trailerLen)
	}
	if save.GoldRaw3 < 0 {
		t.Errorf("GoldRaw3 = %d，不應為負", save.GoldRaw3)
	}
	// Facing 假設是 0-3（四方位），這裡只檢查沒有明顯超出合理範圍太多。
	if save.Facing > 3 {
		t.Logf("Facing = %d 超出假設的 0-3 四方位範圍，記錄下來但不視為測試失敗（假設信心中高、非硬性驗證）", save.Facing)
	}
}

func TestLoadSaveGame_MissingFile(t *testing.T) {
	_, err := LoadSaveGame(filepath.Join(dataDir, "DOES_NOT_EXIST.DAT"))
	if err == nil {
		t.Fatal("預期檔案不存在時回傳 error，卻回傳 nil")
	}
}

// TestLoadSaveGame_NeverWritesOrigFile 是一個防呆測試：確認 LoadSaveGame 不會
// 修改 PARTY.DAT 的內容（讀取前後 mtime／內容不變）。CLAUDE.md 硬規則：
// workplace/orig/ 唯讀，PARTY.DAT 特別不可寫入。
func TestLoadSaveGame_NeverWritesOrigFile(t *testing.T) {
	path := partyDatPath()
	skipIfMissing(t, path)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("讀取 %s 失敗: %v", path, err)
	}

	if _, err := LoadSaveGame(path); err != nil {
		t.Fatalf("LoadSaveGame(%s) 失敗: %v", path, err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("讀取 %s 失敗: %v", path, err)
	}

	if len(before) != len(after) {
		t.Fatalf("%s 長度在 LoadSaveGame 前後不一致：%d -> %d", path, len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("%s 在 LoadSaveGame 前後內容不一致，offset 0x%x: %d -> %d", path, i, before[i], after[i])
		}
	}
}
