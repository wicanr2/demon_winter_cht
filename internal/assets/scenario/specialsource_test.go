package scenario

import (
	"os"
	"path/filepath"
	"testing"

)

// 這一組測試釘住「特殊格清單從哪裡來」。
//
// 這件事會安靜地出錯：來源挑錯了，遊戲照樣跑、照樣顯示事件，
// 只是玩家繼承了別人走過的痕跡，或是關掉遊戲就把進度丟了。
// 兩種都在畫面上看不出來 —— 所以要靠測試。

// makeList 造一份 511 bytes 的清單：一筆記錄 (x, y, attr)，其餘留零。
func makeList(x, y, attr byte) []byte {
	b := make([]byte, SpecialTileFileSize)
	b[0], b[1], b[2] = x, y, attr
	return b
}

// writeDataDir 造一個假的原版資料目錄：母本 ALL_SS.DAT 的屬性是 0x20，
// 而磁碟上的 nSS.DAT 是「玩過的」0x21 —— 就是原版出廠的情況。
func writeDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	all := make([]byte, ALLSSBlockSize*SpecialTileMapCount)
	for i := 0; i < SpecialTileMapCount; i++ {
		copy(all[i*ALLSSBlockSize:], makeList(9, 27, 0x20))
	}
	if err := os.WriteFile(filepath.Join(dir, "ALL_SS.DAT"), all, 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= SpecialTileMapCount; i++ {
		p := filepath.Join(dir, SpecialTileFileName(i))
		if err := os.WriteFile(p, makeList(9, 27, 0x21), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func attrOf(t *testing.T, m map[int]*SpecialTiles, mapID int) byte {
	t.Helper()
	st := m[mapID]
	if st == nil {
		t.Fatalf("子地圖 %d 沒有清單", mapID)
	}
	if len(st.Tiles) == 0 {
		t.Fatalf("子地圖 %d 的清單是空的", mapID)
	}
	return st.Tiles[0].Attr
}

// 全新開始必須從母本重建，**不能**沿用磁碟上那份 nSS.DAT ——
// 原版出廠的 1SS／2SS 是壓片前玩到一半的狀態（docs/re/78 §2）。
func TestLoadSpecialTiles_FreshRebuildsFromMaster(t *testing.T) {
	dataDir := writeDataDir(t)
	got, err := LoadSpecialTileSet(t.TempDir(), dataDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if a := attrOf(t, got, 1); a != 0x20 {
		t.Errorf("全新開始的屬性 = 0x%02x，預期 0x20（母本）。"+
			"0x21 表示沿用了磁碟上那份玩過的清單", a)
	}
}

// 有進度存檔時要讀存檔目錄那份，不是母本 —— 否則每次啟動都把進度重置。
func TestLoadSpecialTiles_PrefersSaveDir(t *testing.T) {
	dataDir := writeDataDir(t)
	saveDir := t.TempDir()
	p := filepath.Join(saveDir, SpecialTileFileName(1))
	if err := os.WriteFile(p, makeList(9, 27, 0x41), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSpecialTileSet(saveDir, dataDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if a := attrOf(t, got, 1); a != 0x41 {
		t.Errorf("屬性 = 0x%02x，預期 0x41（存檔目錄那份）", a)
	}
}

// 存檔目錄還沒有清單（舊存檔升級上來）→ 退回母本，不是報錯。
func TestLoadSpecialTiles_EmptySaveDirFallsBackToMaster(t *testing.T) {
	dataDir := writeDataDir(t)
	got, err := LoadSpecialTileSet(t.TempDir(), dataDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if a := attrOf(t, got, 1); a != 0x20 {
		t.Errorf("屬性 = 0x%02x，預期 0x20（母本）", a)
	}
}

// 連母本都沒有 → 退回讀原版資料目錄的 nSS.DAT（髒的，但有總比沒有好）。
func TestLoadSpecialTiles_NoMasterFallsBackToDataDir(t *testing.T) {
	dataDir := writeDataDir(t)
	if err := os.Remove(filepath.Join(dataDir, "ALL_SS.DAT")); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSpecialTileSet(t.TempDir(), dataDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if a := attrOf(t, got, 1); a != 0x21 {
		t.Errorf("屬性 = 0x%02x，預期 0x21（資料目錄那份）", a)
	}
}

// 什麼都沒有不算錯：大地圖與城鎮本來就沒有特殊格清單。
func TestLoadSpecialTiles_NothingIsNotAnError(t *testing.T) {
	got, err := LoadSpecialTileSet(t.TempDir(), t.TempDir(), true)
	if err != nil {
		t.Fatalf("缺檔不該是錯誤：%v", err)
	}
	if len(got) != 0 {
		t.Errorf("預期空的 map，得到 %d 筆", len(got))
	}
}

// WriteSpecialTileSet 必須寫進指定目錄，而且寫出來的能再讀回來。
// 原版資料目錄是唯讀的（CLAUDE.md 硬規則），呼叫端要負責傳存檔目錄進來。
func TestWriteSpecialTileSet_RoundTrips(t *testing.T) {
	saveDir := filepath.Join(t.TempDir(), "save")
	set := map[int]*SpecialTiles{
		1: {Tiles: []SpecialTile{{X: 9, Y: 27, Attr: 0x20}}},
		2: nil, // nil 要被跳過，不是 panic
	}
	set[1].MarkVisited(0)
	if err := WriteSpecialTileSet(saveDir, set); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(saveDir, SpecialTileFileName(1)))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != SpecialTileFileSize {
		t.Errorf("寫出 %d bytes，預期 %d", len(raw), SpecialTileFileSize)
	}
	back, err := ParseSpecialTiles(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Tiles) != 1 || back.Tiles[0].Attr != 0x21 {
		t.Errorf("讀回來的清單 = %+v，預期一筆 attr 0x21", back.Tiles)
	}
	if _, err := os.Stat(filepath.Join(saveDir, SpecialTileFileName(2))); !os.IsNotExist(err) {
		t.Error("nil 的項目不該產生檔案")
	}
}
