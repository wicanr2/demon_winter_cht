package world

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExits_RealFile(t *testing.T) {
	dir := origDataDir(t)
	et, err := LoadExits(filepath.Join(dir, "EXITS.DAT"))
	if err != nil {
		t.Fatalf("LoadExits 失敗: %v", err)
	}

	if got := et.Len(); got != exitRecordCount {
		t.Fatalf("Len() = %d, want %d", got, exitRecordCount)
	}

	all := et.All()
	if len(all) != exitRecordCount {
		t.Fatalf("All() 長度 = %d, want %d", len(all), exitRecordCount)
	}

	// 驗收要求：所有 X/Y 都在 1–62（見 docs/formats/town-and-map.md §3、
	// docs/re/05-event-triggering.md §1.4）。
	for i, r := range all {
		if r.X < 1 || r.X > 62 || r.Y < 1 || r.Y > 62 {
			t.Errorf("記錄 #%d 座標 (%d,%d) 超出 1–62 範圍", i, r.X, r.Y)
		}
	}

	// Lookup 應該能查到全部記錄裡至少某一筆真實座標（用 All() 第一筆自我驗證）。
	first := all[0]
	got, ok := et.Lookup(first.X, first.Y)
	if !ok {
		t.Fatalf("Lookup(%d,%d) 應該命中第一筆記錄，卻沒有", first.X, first.Y)
	}
	if got.Type != first.Type {
		t.Errorf("Lookup(%d,%d).Type = %d, want %d（重複座標應回傳第一筆，比照原版線性掃描行為）", first.X, first.Y, got.Type, first.Type)
	}

	// Lookup 對不存在的座標應回傳 ok=false，而不是 panic 或回傳零值誤導呼叫端。
	if _, ok := et.Lookup(0, 0); ok {
		t.Error("Lookup(0,0) 預期 ok=false（EXITS.DAT 已驗證沒有任何 0 byte），卻回傳 true")
	}
}

func TestExitRecord_CategorySubValue(t *testing.T) {
	// type_byte = 4*32 + 5 = 133 → 類別4(傳送)、子值5。
	r := ExitRecord{X: 1, Y: 1, Type: 133}
	if got := r.Category(); got != 4 {
		t.Errorf("Category() = %d, want 4", got)
	}
	if got := r.SubValue(); got != 5 {
		t.Errorf("SubValue() = %d, want 5", got)
	}
}

func TestLoadExits_WrongSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "BAD.DAT")
	if err := os.WriteFile(path, make([]byte, 10), 0o644); err != nil {
		t.Fatalf("寫測試檔失敗: %v", err)
	}
	if _, err := LoadExits(path); err == nil {
		t.Error("LoadExits 對長度不符的檔案預期回傳 error，卻沒有")
	}
}

func TestLoadExits_MissingFile(t *testing.T) {
	if _, err := LoadExits("/nonexistent/path/EXITS.DAT"); err == nil {
		t.Error("LoadExits 對不存在的檔案預期回傳 error，卻沒有")
	}
}
