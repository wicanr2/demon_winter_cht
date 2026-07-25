package gamedata

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadItemTable_RealFile 對真實的 ITEMS.DAT 跑，驗證武器／護甲的順序
// 與 translations/glossary.md 第 8 節「武器與護甲」一致——這是文件裡標註
// 的最強驗證錨點。
func TestLoadItemTable_RealFile(t *testing.T) {
	dir := origDataDir(t)
	table, err := LoadItemTable(filepath.Join(dir, "ITEMS.DAT"))
	if err != nil {
		t.Fatalf("LoadItemTable 失敗: %v", err)
	}

	if got, want := table.Len(), 30; got != want {
		t.Fatalf("道具總數 = %d，want %d", got, want)
	}

	// translations/glossary.md 第 8 節：武器 8 種（索引 0-7），
	// 售價隨強度遞增；護甲 5 種（索引 8-12），售價隨防護力遞增。
	wantWeapons := []struct {
		name  string
		price int
	}{
		{"dagger", 2}, {"small axe", 6}, {"short sword", 15}, {"mace", 13},
		{"morning star", 20}, {"broad sword", 30}, {"battle axe", 65}, {"2-hand sword", 100},
	}
	for i, want := range wantWeapons {
		got, err := table.ByIndex(i)
		if err != nil {
			t.Fatalf("ByIndex(%d) 失敗: %v", i, err)
		}
		if got.Name != want.name {
			t.Errorf("索引 %d 名稱 = %q，want %q（武器順序需與 glossary.md 第 8 節一致）", i, got.Name, want.name)
		}
		if got.Kind != ItemKindWeapon {
			t.Errorf("%s 的 Kind = %v，want ItemKindWeapon", got.Name, got.Kind)
		}
		if got.Price != want.price {
			t.Errorf("%s 的 Price = %d，want %d", got.Name, got.Price, want.price)
		}
		if !got.WeaponSlot {
			t.Errorf("%s 的 WeaponSlot = false，want true（武器應佔用武器手欄位）", got.Name)
		}
	}

	wantArmor := []struct {
		name  string
		price int
	}{
		{"cloth", 5}, {"leather", 15}, {"chain", 40}, {"scale", 200}, {"plate", 400},
	}
	for i, want := range wantArmor {
		idx := 8 + i
		got, err := table.ByIndex(idx)
		if err != nil {
			t.Fatalf("ByIndex(%d) 失敗: %v", idx, err)
		}
		if got.Name != want.name {
			t.Errorf("索引 %d 名稱 = %q，want %q（護甲順序需與 glossary.md 第 8 節一致）", idx, got.Name, want.name)
		}
		if got.Kind != ItemKindArmor {
			t.Errorf("%s 的 Kind = %v，want ItemKindArmor", got.Name, got.Kind)
		}
		if got.Price != want.price {
			t.Errorf("%s 的 Price = %d，want %d", got.Name, got.Price, want.price)
		}
		if got.WeaponSlot {
			t.Errorf("%s 的 WeaponSlot = true，want false（護甲不佔用武器手欄位）", got.Name)
		}
	}

	// 同大類（8 把武器）f3-f6 未定欄位數值完全相同的規律，是文件裡用來
	// 排除「逐項武器獨立傷害骰」假設的關鍵觀察，回歸測試鎖住這個事實。
	dagger, ok := table.ByName("dagger")
	if !ok {
		t.Fatal(`ByName("dagger") 找不到`)
	}
	twoHand, err := table.ByIndex(7)
	if err != nil {
		t.Fatalf("ByIndex(7) 失敗: %v", err)
	}
	if dagger.EffectClasses != twoHand.EffectClasses {
		t.Errorf("同為武器類的 dagger 與 2-hand sword 效果類別候選應完全相同，得到 %v vs %v",
			dagger.EffectClasses, twoHand.EffectClasses)
	}

	// vial 出現兩次，ByName 只保證回傳其中一筆，All() 才能看到兩筆都在。
	all := table.All()
	vialCount := 0
	for _, it := range all {
		if it.Name == "vial" {
			vialCount++
		}
	}
	if vialCount != 2 {
		t.Errorf("ITEMS.DAT 裡 vial 應出現 2 次，實際 %d 次", vialCount)
	}
}

func TestItemTable_ByIndexOutOfRange(t *testing.T) {
	dir := origDataDir(t)
	table, err := LoadItemTable(filepath.Join(dir, "ITEMS.DAT"))
	if err != nil {
		t.Fatalf("LoadItemTable 失敗: %v", err)
	}
	if _, err := table.ByIndex(-1); err == nil {
		t.Error("ByIndex(-1) 應回傳 error")
	}
	if _, err := table.ByIndex(table.Len()); err == nil {
		t.Error("ByIndex(Len()) 應回傳 error（超出範圍）")
	}
}

func TestLoadItemTable_MalformedTokenCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ITEMS.DAT")
	// 只有 1 個名字 + 3 個數字欄位，湊不滿 8 個 token。
	content := "dagger\x002\x001\x003\x00"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("寫入測試檔案失敗: %v", err)
	}
	if _, err := LoadItemTable(path); err == nil {
		t.Error("token 數不是 8 的倍數時 LoadItemTable 應回傳 error")
	}
}

func TestLoadItemTable_NonIntegerField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ITEMS.DAT")
	fields := []string{"dagger", "2", "1", "3", "not-a-number", "10", "8", "9"}
	content := ""
	for _, f := range fields {
		content += f + "\x00"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("寫入測試檔案失敗: %v", err)
	}
	if _, err := LoadItemTable(path); err == nil {
		t.Error("數字欄位不是合法整數時 LoadItemTable 應回傳 error")
	}
}

func TestLoadItemTable_MissingFile(t *testing.T) {
	if _, err := LoadItemTable(filepath.Join(t.TempDir(), "不存在.DAT")); err == nil {
		t.Error("檔案不存在時 LoadItemTable 應回傳 error")
	}
}
