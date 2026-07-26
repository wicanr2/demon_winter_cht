package scenario

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// 最低驗收：讀出來再寫回去，byte-for-byte 相同。
//
// 這條過不了就代表某個欄位的 encode 與 decode 不對稱。格式還有一大片未解區域，
// 「重建」會把那些 bytes 填成 0 —— 遊戲不會報錯，只會有某些狀態悄悄消失。
//
// **但這條比看起來弱得多，別把它當唯一防線。** 實測種了三個錯進去都沒被抓到：
//   - 經驗值寫 3 bytes 而不是 4：數值封頂 0x00FFFFFF，第 4 個 byte 本來就是 0
//   - 姓名欄後段清成 0：這份存檔的姓名欄本來就是 0 填充
//   - 等級寫到 +1 的位移：下一行寫種族時又把它蓋回去
//
// 真正釘住「哪個欄位在哪個位移、佔幾個 byte」的是下面的
// TestEncode_FieldsWriteExactBytes。
func TestSaveGame_RoundTripIsByteIdentical(t *testing.T) {
	for _, name := range []string{"PARTY.DAT", "PARTY.BAK"} {
		path := filepath.Join(dataDir, name)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("讀 %s: %v", name, err)
		}

		save, err := LoadSaveGame(path)
		if err != nil {
			t.Fatalf("解析 %s: %v", name, err)
		}
		got, err := save.Encode()
		if err != nil {
			t.Fatalf("編碼 %s: %v", name, err)
		}

		if len(got) != len(original) {
			t.Fatalf("%s 編碼後 %d bytes，原檔 %d bytes", name, len(got), len(original))
		}
		if !bytes.Equal(got, original) {
			t.Errorf("%s 來回一趟後內容不同：%s", name, firstDiff(original, got))
		}
	}
}

// firstDiff 回報第一個不同的位移與前後幾個 byte，方便定位是哪個欄位。
func firstDiff(want, got []byte) string {
	for i := range want {
		if want[i] == got[i] {
			continue
		}
		lo := i - 4
		if lo < 0 {
			lo = 0
		}
		hi := i + 5
		if hi > len(want) {
			hi = len(want)
		}
		return fmt.Sprintf("位移 0x%03x：原檔 %02x → 編碼 %02x（附近 原 % x ／ 新 % x）",
			i, want[i], got[i], want[lo:hi], got[lo:hi])
	}
	return "長度不同但內容前綴相同"
}

// 改過的欄位要真的寫進去，而且只動到該動的地方。
func TestSaveGame_EditedFieldsPersist(t *testing.T) {
	path := filepath.Join(dataDir, "PARTY.DAT")
	save, err := LoadSaveGame(path)
	if err != nil {
		t.Fatal(err)
	}

	save.Characters[0].CurrentHP = 7
	save.Characters[0].Experience = 123456
	save.Characters[0].Level = 9
	save.Characters[2].Name = "Test"
	save.PositionX = 40
	save.PositionY = 41
	save.Facing = 3
	save.GoldRaw3 = 65432
	save.Hour = 22
	save.Day = 19
	save.Month = 6
	save.Rations = 31
	save.MapID = 47
	save.LightSource = 3

	data, err := save.Encode()
	if err != nil {
		t.Fatal(err)
	}

	tmp := filepath.Join(t.TempDir(), "PARTY.DAT")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatal(err)
	}
	back, err := LoadSaveGame(tmp)
	if err != nil {
		t.Fatal(err)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{"角色 1 目前生命", back.Characters[0].CurrentHP, byte(7)},
		{"角色 1 經驗值", back.Characters[0].Experience, 123456},
		{"角色 1 等級", back.Characters[0].Level, byte(9)},
		{"角色 3 姓名", back.Characters[2].Name, "Test"},
		{"隊伍 X", back.PositionX, byte(40)},
		{"隊伍 Y", back.PositionY, byte(41)},
		{"面向", back.Facing, byte(3)},
		{"金幣", back.GoldRaw3, 65432},
		{"時辰", back.Hour, byte(22)},
		{"日", back.Day, byte(19)},
		{"月", back.Month, byte(6)},
		{"糧食", back.Rations, byte(31)},
		{"子地圖", back.MapID, byte(47)},
		{"光源", back.LightSource, byte(3)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v，預期 %v", c.name, c.got, c.want)
		}
	}

	// 沒改到的角色必須一個 byte 都沒變。
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const rec2 = 1 * recordLen
	if !bytes.Equal(data[rec2:rec2+recordLen], original[rec2:rec2+recordLen]) {
		t.Error("沒動到的角色 2 記錄被改寫了")
	}
}

// 姓名變短時不能把後面的 bytes 清成 0 —— 那是「看起來比較乾淨」的破壞。
func TestSaveGame_ShorterNameKeepsTrailingBytes(t *testing.T) {
	save, err := LoadSaveGame(filepath.Join(dataDir, "PARTY.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), save.Characters[0].Raw[:nameFieldLen]...)

	save.Characters[0].Name = "Al"
	data, err := save.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if data[0] != 'A' || data[1] != 'l' || data[2] != 0 {
		t.Errorf("姓名沒寫對：% x", data[:4])
	}
	// 第 4 個 byte 之後應與原檔相同。
	for i := 3; i < nameFieldLen; i++ {
		if data[i] != before[i] {
			t.Errorf("姓名欄位第 %d byte 被動到：%02x → %02x", i, before[i], data[i])
			break
		}
	}
}

// 過長的姓名要截斷，而且結尾一定有 NUL —— 沒有的話讀回來會吃到下一個欄位。
func TestSaveGame_LongNameTruncated(t *testing.T) {
	save, err := LoadSaveGame(filepath.Join(dataDir, "PARTY.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	save.Characters[0].Name = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	data, err := save.Encode()
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "PARTY.DAT")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatal(err)
	}
	back, err := LoadSaveGame(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Characters[0].Name) > nameFieldLen-1 {
		t.Errorf("姓名 %q 長度 %d，超過欄位容量", back.Characters[0].Name,
			len(back.Characters[0].Name))
	}
	if data[nameFieldLen-1] != 0 && data[len(back.Characters[0].Name)] != 0 {
		t.Error("截斷後沒有 NUL 結尾")
	}
}

// SaveTo 寫檔：寫成功要能讀回來，而且不留暫存檔。
func TestSaveTo(t *testing.T) {
	save, err := LoadSaveGame(filepath.Join(dataDir, "PARTY.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "PARTY.DAT")

	save.Characters[0].CurrentHP = 11
	if err := save.SaveTo(path); err != nil {
		t.Fatal(err)
	}
	back, err := LoadSaveGame(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Characters[0].CurrentHP != 11 {
		t.Errorf("寫回後目前生命 = %d，預期 11", back.Characters[0].CurrentHP)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "PARTY.DAT" {
			t.Errorf("目錄裡多了 %q，暫存檔沒清乾淨", e.Name())
		}
	}
}

// 覆蓋既有存檔也要能讀回來 —— 這是實際玩的時候唯一會走的路徑。
func TestSaveTo_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PARTY.DAT")

	original, err := os.ReadFile(filepath.Join(dataDir, "PARTY.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	save, err := LoadSaveGame(path)
	if err != nil {
		t.Fatal(err)
	}
	save.PositionX = 55
	if err := save.SaveTo(path); err != nil {
		t.Fatal(err)
	}
	back, err := LoadSaveGame(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.PositionX != 55 {
		t.Errorf("覆蓋後 X = %d，預期 55", back.PositionX)
	}
}

// 逐欄位釘住「寫到哪個位移、動幾個 byte」。
//
// 作法是把欄位改成一個與原值不同的值，編碼後比對整份檔案，
// **要求恰好只有預期的那幾個 byte 變動**。位移寫錯會踩到別的欄位、
// 寬度寫錯會少動或多動 byte，兩種都會被抓到 ——
// 這是 round-trip 測試漏掉的那一半。
func TestEncode_FieldsWriteExactBytes(t *testing.T) {
	base, err := os.ReadFile(filepath.Join(dataDir, "PARTY.DAT"))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		mutate func(*SaveGame)
		off    int // 檔案內的絕對位移
		length int
	}{
		{"角色1 等級", func(s *SaveGame) { s.Characters[0].Level ^= 0xff },
			levelOffset, 1},
		{"角色1 種族", func(s *SaveGame) { s.Characters[0].RaceByte ^= 0xff },
			raceOffset, 1},
		{"角色1 職業", func(s *SaveGame) { s.Characters[0].ClassByte ^= 0xff },
			classOffset, 1},
		// 刻意用超過封頂（0x00FFFFFF）的值：合法值的第 4 個 byte 恆為 0，
		// 拿合法值測只會動到 3 個 byte，測不出欄位到底是 3 還是 4 bytes 寬。
		{"角色1 經驗值（釘住 4 bytes 寬）",
			func(s *SaveGame) { s.Characters[0].Experience = 0x7FFFFFFF },
			expOffset, expLen},
		{"角色1 技能旗標", func(s *SaveGame) { s.Characters[0].SkillFlags[0] ^= 1 },
			skillFlagsOffset, 1},
		{"角色1 目前生命", func(s *SaveGame) { s.Characters[0].CurrentHP ^= 0xff },
			expOffset + attrCurrentHPOffset, 1},
		{"角色1 智力", func(s *SaveGame) { s.Characters[0].Intellect ^= 0xff },
			expOffset + attrIntellectOffset, 1},
		{"角色1 武器槽", func(s *SaveGame) { s.Characters[0].WeaponSlotIndex ^= 0xff },
			weaponSlotOffset, 1},
		{"角色2 目前法力", func(s *SaveGame) { s.Characters[1].CurrentSP ^= 0xff },
			recordLen + expOffset + attrCurrentSPOffset, 1},
		{"角色5 等級", func(s *SaveGame) { s.Characters[4].Level ^= 0xff },
			4*recordLen + levelOffset, 1},
		{"隊伍 X", func(s *SaveGame) { s.PositionX ^= 0xff },
			trailerStart + positionXOffset, 1},
		{"隊伍 Y", func(s *SaveGame) { s.PositionY ^= 0xff },
			trailerStart + positionYOffset, 1},
		{"面向", func(s *SaveGame) { s.Facing ^= 0xff },
			trailerStart + facingOffset, 1},
		{"金幣", func(s *SaveGame) { s.GoldRaw3 = 0xABCDEF },
			trailerStart + goldOffset, 3},
	}

	for _, c := range cases {
		save, err := LoadSaveGame(filepath.Join(dataDir, "PARTY.DAT"))
		if err != nil {
			t.Fatal(err)
		}
		c.mutate(save)
		got, err := save.Encode()
		if err != nil {
			t.Fatalf("%s：編碼失敗 %v", c.name, err)
		}

		changed := changedRanges(base, got)
		want := []byteRange{{c.off, c.length}}
		if !sameRanges(changed, want) {
			t.Errorf("%s：實際變動 %v，預期只有位移 0x%03x 起 %d bytes",
				c.name, changed, c.off, c.length)
		}
	}
}

type byteRange struct{ off, length int }

// changedRanges 回傳兩份資料所有連續不同的區段。
func changedRanges(a, b []byte) []byteRange {
	var out []byteRange
	i := 0
	for i < len(a) && i < len(b) {
		if a[i] == b[i] {
			i++
			continue
		}
		start := i
		for i < len(a) && i < len(b) && a[i] != b[i] {
			i++
		}
		out = append(out, byteRange{start, i - start})
	}
	return out
}

func sameRanges(got, want []byteRange) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// 道具槽要真的寫得回去 —— 買到的東西不能在存檔時人間蒸發。
//
// 這是很容易漏的一條：encode 原本只把 InventorySlotsRaw 抄回去，
// 規則層改的是解析後的 Inventory，兩者不同步的話買賣與換裝全都是假的，
// 而且遊戲不會報錯，玩家只會發現「東西買了又不見」。
func TestSaveGame_InventoryChangesPersist(t *testing.T) {
	save, err := LoadSaveGame(filepath.Join(dataDir, "PARTY.DAT"))
	if err != nil {
		t.Fatal(err)
	}

	c := &save.Characters[1] // Stumpy：只有第 0 格有東西
	if !c.Inventory[5].Empty() {
		t.Fatalf("前提不成立：第 5 格不是空的（%+v）", c.Inventory[5])
	}
	c.Inventory[5] = InventorySlot{Type: 4, Identified: true}
	c.Inventory[6] = InventorySlot{Type: 16, Effect: 3, Power: 8, Total: 5, Used: 2, Enchant: 2}
	c.Inventory[0] = InventorySlot{Type: slotEmpty} // 賣掉原本那件
	c.WeaponSlotIndex = 5

	data, err := save.Encode()
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(t.TempDir(), "PARTY.DAT")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatal(err)
	}
	back, err := LoadSaveGame(tmp)
	if err != nil {
		t.Fatal(err)
	}

	got := back.Characters[1]
	if got.Inventory[5].Type != 4 || !got.Inventory[5].Identified {
		t.Errorf("買到的道具沒寫回去：%+v", got.Inventory[5])
	}
	// 新造的一格不能帶著舊道具的附魔或次數。
	if got.Inventory[5].Enchant != 0 || got.Inventory[5].Total != 0 {
		t.Errorf("新道具帶了殘值：%+v", got.Inventory[5])
	}
	want := InventorySlot{Type: 16, Effect: 3, Power: 8, Total: 5, Used: 2, Enchant: 2}
	if got.Inventory[6] != want {
		t.Errorf("有效果的道具走樣：\n得到 %+v\n預期 %+v", got.Inventory[6], want)
	}
	if !got.Inventory[0].Empty() {
		t.Errorf("賣掉的那一格還在：%+v", got.Inventory[0])
	}
	if got.WeaponSlotIndex != 5 {
		t.Errorf("裝備槽索引 %d，預期 5", got.WeaponSlotIndex)
	}
}

// 清空一格只寫型別，其餘 bytes 留原樣 —— 照原版做的。
//
// 兩份原版存檔都看得到這種殘值：Wopple 交出去的那幾格型別是 0xFF，
// 附魔與 +0x0f 還留著前一件的值。清成 0 會讓存檔與原版不再等價。
func TestSaveGame_ClearedSlotKeepsTrailingBytes(t *testing.T) {
	path := filepath.Join(dataDir, "PARTY.DAT")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	save, err := LoadSaveGame(path)
	if err != nil {
		t.Fatal(err)
	}
	save.Characters[0].Inventory[0] = InventorySlot{Type: slotEmpty}

	data, err := save.Encode()
	if err != nil {
		t.Fatal(err)
	}
	base := inventoryStart
	if data[base] != slotEmpty {
		t.Errorf("型別 = %#02x，預期 %#02x", data[base], slotEmpty)
	}
	if !bytes.Equal(data[base+1:base+inventorySlotLen], original[base+1:base+inventorySlotLen]) {
		t.Errorf("清空那一格的其餘 bytes 被動了：\n原 % x\n新 % x",
			original[base+1:base+inventorySlotLen], data[base+1:base+inventorySlotLen])
	}
}
