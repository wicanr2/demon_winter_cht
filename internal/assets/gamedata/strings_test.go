package gamedata

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestLoadStringPool_RealFile 對真實的 FILES.DTT 跑，驗證
// docs/formats/resource-index.md 第 1.2 節列出的「已驗證分段」內容。
func TestLoadStringPool_RealFile(t *testing.T) {
	dir := origDataDir(t)
	pool, err := LoadStringPool(filepath.Join(dir, "FILES.DTT"))
	if err != nil {
		t.Fatalf("LoadStringPool 失敗: %v", err)
	}

	if got, want := pool.Len(), 501; got != want {
		t.Fatalf("字串總數 = %d，want %d", got, want)
	}

	wantRaces := []string{"Human", "Elf", "Dwarf", "Dark Elf", "Troll"}
	if got := pool.RaceNames(); !reflect.DeepEqual(got, wantRaces) {
		t.Errorf("RaceNames() = %v，want %v", got, wantRaces)
	}

	skills := pool.SkillNames()
	if got, want := len(skills), 32; got != want {
		t.Fatalf("SkillNames() 長度 = %d，want %d", got, want)
	}
	foundShamen := false
	for _, s := range skills {
		if s == "Shamen" {
			foundShamen = true
		}
	}
	if !foundShamen {
		t.Error(`SkillNames() 應含遊戲原始拼字 "Shamen"（glossary.md 備註記載的已知誤拼）`)
	}

	wantWeaponTypes := []string{
		"dagger", "small axe", "short sword", "mace",
		"morning star", "broad sword", "battle axe", "2-hand sword",
	}
	if got := pool.WeaponTypeNames(); !reflect.DeepEqual(got, wantWeaponTypes) {
		t.Errorf("WeaponTypeNames() = %v，want %v", got, wantWeaponTypes)
	}

	wantArmorTypes := []string{"cloth", "leather", "chain", "scale", "plate"}
	if got := pool.ArmorTypeNames(); !reflect.DeepEqual(got, wantArmorTypes) {
		t.Errorf("ArmorTypeNames() = %v，want %v", got, wantArmorTypes)
	}

	wantSummons := []string{
		"Coyote", "Zombie", "Brown bear", "Small dragon", "Ogre", "Evil spirit",
		"Fire demon", "Fire elemental", "Metal elemental", "Wind elemental",
		"Ice elemental", "Spirit elemental",
	}
	if got := pool.IllusionSummonNames(); !reflect.DeepEqual(got, wantSummons) {
		t.Errorf("IllusionSummonNames() = %v，want %v", got, wantSummons)
	}

	// 武器/護甲類型名稱要跟 ITEMS.DAT 的名稱、順序一致——兩個獨立檔案
	// 交叉印證同一份資料，是文件裡強調的驗證方式。
	itemTable, err := LoadItemTable(filepath.Join(dir, "ITEMS.DAT"))
	if err != nil {
		t.Fatalf("LoadItemTable 失敗: %v", err)
	}
	for i, wantName := range wantWeaponTypes {
		it, err := itemTable.ByIndex(i)
		if err != nil {
			t.Fatalf("ITEMS.DAT ByIndex(%d) 失敗: %v", i, err)
		}
		if it.Name != wantName {
			t.Errorf("ITEMS.DAT 索引 %d 名稱 = %q，與 FILES.DTT WeaponTypeNames() 的 %q 對不上", i, it.Name, wantName)
		}
	}
}

func TestStringPool_AtOutOfRange(t *testing.T) {
	dir := origDataDir(t)
	pool, err := LoadStringPool(filepath.Join(dir, "FILES.DTT"))
	if err != nil {
		t.Fatalf("LoadStringPool 失敗: %v", err)
	}
	if _, err := pool.At(-1); err == nil {
		t.Error("At(-1) 應回傳 error")
	}
	if _, err := pool.At(pool.Len()); err == nil {
		t.Error("At(Len()) 應回傳 error（超出範圍）")
	}
}

// TestLoadStringPool_Synthetic 用合成資料驗證「中段空字串保留、只丟棄檔尾
// 空 token」這個關鍵行為，不依賴真實檔案。
func TestLoadStringPool_Synthetic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "FILES.DTT")
	// "a" NUL "" NUL "b" NUL  -> 三個 token："a"、""、"b"，
	// split 出的第 4 個（檔尾空字串）要被丟棄。
	content := "a\x00\x00b\x00"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("寫入測試檔案失敗: %v", err)
	}
	pool, err := LoadStringPool(path)
	if err != nil {
		t.Fatalf("LoadStringPool 失敗: %v", err)
	}
	want := []string{"a", "", "b"}
	if got := pool.All(); !reflect.DeepEqual(got, want) {
		t.Errorf("All() = %#v，want %#v（中段空字串應保留）", got, want)
	}
}

func TestLoadStringPool_MissingFile(t *testing.T) {
	if _, err := LoadStringPool(filepath.Join(t.TempDir(), "不存在.DTT")); err == nil {
		t.Error("檔案不存在時 LoadStringPool 應回傳 error")
	}
}
