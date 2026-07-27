package world

import (
	"path/filepath"
	"testing"
)

// 幽靈司祭那道密語答對之後打開的那一格（`docs/re/84`）。
//
// 原版把它寫成**線性索引常數** `0x48b`（`0x1a375`）。換算 `y*64 + x`
// ＝ (11,18)。這條測試釘住三件事同時成立：
//
//  1. 座標換算是 `y*64 + x`（不是 `x*64 + y`）
//  2. `MAP5.MAP` 的檔頭處理正確（tile 陣列從 offset 0 起算）
//  3. 那一格原本真的是牆
//
// 三者任一錯掉，`0x48b` 就不會落在牆上 —— 而遊戲裡的症狀會是
// 「答對了但牆沒開」或「答對了卻在地圖別處挖了個洞」，
// 兩種都不會有錯誤訊息。
func TestSpectralPriestDoorIsAWall(t *testing.T) {
	dir := origDataDir(t)
	m, err := LoadMap(filepath.Join(dir, "MAP5.MAP"))
	if err != nil {
		t.Fatal(err)
	}

	const (
		linear   = 0x48b // 原版寫死的常數
		wallTile = 0x0d  // 可通行表的值是 0xff（牆）
	)
	x, y := linear%MapWidth, linear/MapWidth
	if x != 11 || y != 18 {
		t.Fatalf("0x%x 換算成 (%d,%d)，預期 (11,18) —— 座標公式不對", linear, x, y)
	}

	got, err := m.TileAt(x, y)
	if err != nil {
		t.Fatal(err)
	}
	if got != wallTile {
		t.Errorf("(%d,%d) 的 tile = 0x%02x，預期 0x%02x（牆）", x, y, got, wallTile)
	}

	// 觸發格在正南兩格，本身要走得到（不能也是牆）。
	trigger, err := m.TileAt(11, 20)
	if err != nil {
		t.Fatal(err)
	}
	if trigger == wallTile {
		t.Errorf("觸發格 (11,20) 也是牆 —— 那玩家根本站不上去")
	}
}

// SetTileAt 只動這個 Map 物件，而且範圍檢查要擋住越界。
//
// （寫回檔案是 `SaveMap` 的事 —— 原版改完 tile 確實會存檔，
// 見 `docs/re/95` §3.9，但那一步在呼叫端不在這裡。）
func TestSetTileAt(t *testing.T) {
	dir := origDataDir(t)
	m, err := LoadMap(filepath.Join(dir, "MAP5.MAP"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetTileAt(11, 18, 0); err != nil {
		t.Fatal(err)
	}
	if got, _ := m.TileAt(11, 18); got != 0 {
		t.Errorf("改寫後 tile = 0x%02x，預期 0", got)
	}
	if err := m.SetTileAt(-1, 0, 0); err == nil {
		t.Error("越界座標應該回 error，不是 panic 也不是靜默寫入")
	}
	if err := m.SetTileAt(0, MapHeight, 0); err == nil {
		t.Error("越界座標應該回 error")
	}
}
