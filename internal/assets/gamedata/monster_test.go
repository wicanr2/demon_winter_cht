package gamedata

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadMonsterTable_RealFile 對真實的 MONSTER.DAT 跑，驗證已在
// docs/formats/game-data-tables.md 與事件表交叉驗證過的錨點資料。
// 找不到原始檔案時自動 skip（見 testdata_test.go 的 origDataDir）。
func TestLoadMonsterTable_RealFile(t *testing.T) {
	dir := origDataDir(t)
	table, err := LoadMonsterTable(filepath.Join(dir, "MONSTER.DAT"))
	if err != nil {
		t.Fatalf("LoadMonsterTable 失敗: %v", err)
	}

	if got, want := table.Len(), 99; got != want {
		t.Fatalf("怪物總數 = %d，want %d", got, want)
	}

	// 這組錨點已在事件表交叉驗證過（任務指派時給定，見 CLAUDE.md）。
	wantByIndex := map[int]Monster{
		26: {
			Name: "Kobold", Speed: 7, Strength: 7, Skill: 7, HP: 7,
			AttackType: 2, SpriteIndex: 3, NumAttacks: 0, Experience: 16,
			Level: 1, SP: 0, Special: 0,
		},
		85: {
			Name: "Uffuspgot", Speed: 11, Strength: 10, Skill: 10, HP: 22,
			AttackType: 4, SpriteIndex: 3, NumAttacks: 2, Experience: 500,
			Level: 6, SP: 13, Special: 0,
		},
		67: {
			Name: "Cave bear", Speed: 10, Strength: 26, Skill: 11, HP: 60,
			AttackType: 13, SpriteIndex: 7, NumAttacks: 4, Experience: 215,
			Level: 7, SP: 0, Special: 0,
		},
		2: {
			Name: "Orc", Speed: 6, Strength: 9, Skill: 7, HP: 15,
			AttackType: 2, SpriteIndex: 17, NumAttacks: 2, Experience: 26,
			Level: 2, SP: 0, Special: 0,
		},
		91: {
			Name: "Xeres", Speed: 15, Strength: 27, Skill: 18, HP: 66,
			AttackType: 0, SpriteIndex: 28, NumAttacks: 6, Experience: 3500,
			Level: 10, SP: 56, Special: 1,
		},
		97: {
			Name: "Eregore", Speed: 17, Strength: 20, Skill: 20, HP: 200,
			AttackType: 7, SpriteIndex: 2, NumAttacks: 5, Experience: 5000,
			Level: 8, SP: 200, Special: 1,
		},
		98: {
			Name: "Guardian", Speed: 14, Strength: 50, Skill: 15, HP: 75,
			AttackType: 0, SpriteIndex: 4, NumAttacks: 15, Experience: 10000,
			Level: 10, SP: 0, Special: 7,
		},
	}

	for idx, want := range wantByIndex {
		got, err := table.ByIndex(idx)
		if err != nil {
			t.Fatalf("ByIndex(%d) 失敗: %v", idx, err)
		}
		if got != want {
			t.Errorf("怪物索引 %d = %+v，want %+v", idx, got, want)
		}
	}

	// 蛇類怪物 attack_type 全部帶負號（毒咬），這是 docs 裡「無例外」的規律，
	// 挑一隻做 ByName 查詢的回歸測試。
	cobra, ok := table.ByName("Cobra")
	if !ok {
		t.Fatal(`ByName("Cobra") 找不到`)
	}
	if cobra.AttackType >= 0 {
		t.Errorf("Cobra.AttackType = %d，want 負值（毒咬）", cobra.AttackType)
	}

	if _, ok := table.ByName("這隻怪物不存在"); ok {
		t.Error("ByName 對不存在的名稱應回傳 ok=false")
	}
}

func TestMonsterTable_ByIndexOutOfRange(t *testing.T) {
	dir := origDataDir(t)
	table, err := LoadMonsterTable(filepath.Join(dir, "MONSTER.DAT"))
	if err != nil {
		t.Fatalf("LoadMonsterTable 失敗: %v", err)
	}
	if _, err := table.ByIndex(-1); err == nil {
		t.Error("ByIndex(-1) 應回傳 error")
	}
	if _, err := table.ByIndex(table.Len()); err == nil {
		t.Error("ByIndex(Len()) 應回傳 error（超出範圍）")
	}
}

// TestLoadMonsterTable_MalformedTokenCount 用合成資料測試「token 數不是 12
// 的倍數」的錯誤路徑，不依賴真實檔案。
func TestLoadMonsterTable_MalformedTokenCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MONSTER.DAT")
	// 只有 1 個名字 + 5 個數字欄位，湊不滿 12 個 token。
	content := "Orc\x006\x009\x007\x0015\x000\x00"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("寫入測試檔案失敗: %v", err)
	}
	if _, err := LoadMonsterTable(path); err == nil {
		t.Error("token 數不是 12 的倍數時 LoadMonsterTable 應回傳 error")
	}
}

// TestLoadMonsterTable_NonIntegerField 用合成資料測試「數字欄位不是合法整數」
// 的錯誤路徑。
func TestLoadMonsterTable_NonIntegerField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MONSTER.DAT")
	fields := make([]string, 12)
	fields[0] = "Orc"
	for i := 1; i < 12; i++ {
		fields[i] = "1"
	}
	fields[5] = "not-a-number"
	content := ""
	for _, f := range fields {
		content += f + "\x00"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("寫入測試檔案失敗: %v", err)
	}
	if _, err := LoadMonsterTable(path); err == nil {
		t.Error("數字欄位不是合法整數時 LoadMonsterTable 應回傳 error")
	}
}

func TestLoadMonsterTable_MissingFile(t *testing.T) {
	if _, err := LoadMonsterTable(filepath.Join(t.TempDir(), "不存在.DAT")); err == nil {
		t.Error("檔案不存在時 LoadMonsterTable 應回傳 error")
	}
}
