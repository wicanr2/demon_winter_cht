package scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func partyDatPath() string {
	return filepath.Join(dataDir, "PARTY.DAT")
}

func loadTestSave(t *testing.T) *SaveGame {
	t.Helper()
	path := partyDatPath()
	skipIfMissing(t, path)
	sg, err := LoadSaveGame(path)
	if err != nil {
		t.Fatalf("LoadSaveGame(%s) 失敗: %v", path, err)
	}
	return sg
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
	if save.Gold < 0 {
		t.Errorf("Gold = %d，不應為負", save.Gold)
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

// 把 docs/formats/game-data-tables.md §1.6 那張「五名角色完整解碼」表釘進測試。
//
// 這張表是整份欄位表最強的一次交叉驗證：技能與職業相稱、武器與技能相稱、
// 護甲與職業限制相稱。任何一個欄位偏移判錯，這裡就會紅。
//
// 先前的測試只驗 Raw[raceOffset] == RaceByte 這類自我一致性，
// 換成錯的偏移一樣會過 —— 那種測試給不了信心。
func TestLoadSaveGame_MatchesVerifiedDecodeTable(t *testing.T) {
	sg := loadTestSave(t)

	want := []struct {
		name   string
		race   byte
		class  byte
		level  byte
		skills []int
	}{
		{"Wopple", 1, 6, 3, []int{17, 21}}, // 精靈 巫師：火焰符文 + 靈魂符文
		{"Stumpy", 0, 7, 3, []int{12, 14}}, // 人類 術士：幻術 + 召喚
		{"Podgom", 0, 5, 3, []int{1, 10}},  // 人類 盜賊：劍術 + 察覺陷阱
		{"Norman", 0, 1, 3, []int{1, 29}},  // 人類 聖騎士：劍術 + 讀心
		{"Menhir", 0, 0, 3, []int{1, 8}},   // 人類 遊俠：劍術 + 狩獵
	}

	for i, w := range want {
		ch := sg.Characters[i]

		if ch.Name != w.name {
			t.Errorf("角色 %d 姓名 = %q，預期 %q", i, ch.Name, w.name)
		}
		if ch.RaceByte != w.race {
			t.Errorf("%s 種族 = %d，預期 %d", w.name, ch.RaceByte, w.race)
		}
		if ch.ClassByte != w.class {
			t.Errorf("%s 職業 = %d，預期 %d", w.name, ch.ClassByte, w.class)
		}
		if ch.Level != w.level {
			t.Errorf("%s 等級 = %d，預期 %d", w.name, ch.Level, w.level)
		}

		var got []int
		for id, on := range ch.SkillFlags {
			if on == 1 {
				got = append(got, id)
			}
		}
		if len(got) != len(w.skills) {
			t.Errorf("%s 已學技能 = %v，預期 %v", w.name, got, w.skills)
			continue
		}
		for k := range w.skills {
			if got[k] != w.skills[k] {
				t.Errorf("%s 已學技能 = %v，預期 %v", w.name, got, w.skills)
				break
			}
		}
	}
}

// 手冊「角色剛創建時只能選兩項技能」—— 五個人都必須恰好兩項。
// 技能旗標偏移若判錯，這裡會讀到一堆隨機的 1。
func TestLoadSaveGame_EachCharacterHasExactlyTwoSkills(t *testing.T) {
	sg := loadTestSave(t)

	for i, ch := range sg.Characters {
		n := 0
		for _, on := range ch.SkillFlags {
			if on != 0 && on != 1 {
				t.Errorf("角色 %d(%s) 技能旗標出現非 0/1 的值 %d，偏移可能判錯", i, ch.Name, on)
			}
			if on == 1 {
				n++
			}
		}
		if n != 2 {
			t.Errorf("角色 %d(%s) 已學 %d 項技能，預期恰好 2 項", i, ch.Name, n)
		}
	}
}

// 職業值必須落在 0–9（技能學費表的欄索引）。
func TestLoadSaveGame_ClassWithinRange(t *testing.T) {
	sg := loadTestSave(t)
	for i, ch := range sg.Characters {
		if ch.ClassByte > 9 {
			t.Errorf("角色 %d(%s) 職業 = %d，超出 0–9", i, ch.Name, ch.ClassByte)
		}
	}
}

// 真實存檔的裝備欄要解得出合理的內容。
//
// 五名起始角色都有裝備（攻略記載開局就配好），
// 解出來全空就代表相位或欄位位移錯了。
func TestInventory_RealSaveHasEquipment(t *testing.T) {
	save, err := LoadSaveGame(filepath.Join(dataDir, "PARTY.DAT"))
	if err != nil {
		t.Fatal(err)
	}

	occupied := 0
	for i := range save.Characters {
		c := save.Characters[i]
		for _, slot := range c.Inventory {
			if !slot.Empty() {
				occupied++
			}
		}
	}
	if occupied == 0 {
		t.Fatal("五名角色的道具欄全空，欄位位移大概錯了")
	}

	// 起始裝備都沒附魔，附魔欄的儲存值是 10（＝加成 0）。
	for i := range save.Characters {
		for j, slot := range save.Characters[i].Inventory {
			if slot.Empty() {
				continue
			}
			if slot.Enchant != 0 {
				t.Errorf("角色 %d 第 %d 格的附魔 = %d，起始裝備應為 0",
					i+1, j, slot.Enchant)
			}
			if int(slot.Type) > 30 {
				t.Errorf("角色 %d 第 %d 格的型別 %d 超出 ITEMS.DAT 的 30 件商品",
					i+1, j, slot.Type)
			}
		}
	}
}

// 空槽的其餘欄位沒有意義，不能解讀成「附魔 −10」。
func TestInventory_EmptySlotHasNoEnchant(t *testing.T) {
	raw := make([]byte, inventorySlotLen)
	for i := range raw {
		raw[i] = 0
	}
	raw[slotType] = slotEmpty

	got := parseInventorySlot(raw)
	if !got.Empty() {
		t.Fatal("型別 0xFF 應判定為空槽")
	}
	if got.Enchant != 0 {
		t.Errorf("空槽的附魔 = %d，不該把 0 解讀成 −10", got.Enchant)
	}
}

// 兩組特效各有一個條件旗標，都啟用時相加。
func TestInventory_WeaponEffectConditions(t *testing.T) {
	mk := func(condA, effA, condB, effB byte) InventorySlot {
		raw := make([]byte, inventorySlotLen)
		raw[slotType] = 5
		raw[slotEnchant] = storedOffset
		raw[slotCondA], raw[slotEffectA] = condA, effA
		raw[slotCondB], raw[slotEffectB] = condB, effB
		return parseInventorySlot(raw)
	}

	if got := mk(0, 13, 0, 14).WeaponEffect; got != 0 {
		t.Errorf("條件旗標都沒啟用時特效應為 0，得到 %d", got)
	}
	if got := mk(effectCondEnabled, 13, 0, 14).WeaponEffect; got != 3 {
		t.Errorf("只啟用 A 組時特效 = %d，預期 13−10 = 3", got)
	}
	if got := mk(effectCondEnabled, 13, effectCondEnabled, 14).WeaponEffect; got != 7 {
		t.Errorf("兩組都啟用時特效 = %d，預期 3+4 = 7", got)
	}
}
