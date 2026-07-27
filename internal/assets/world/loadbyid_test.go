package world

import (
	"os"
	"path/filepath"
	"testing"
)

// 每一個「存檔裡可能出現的地圖編號」都要載得到。
//
// 這條測試存在的理由是一個真的發生過的 regression：`-map` 預設改成
// 「用存檔的 MapID」之後，出貨存檔的 `1` 走進了只查 `SUM.MAP` 的那一份
// 實作，**啟動直接失敗**。當時只用 `-newgame`（MapID 34）驗過。
//
// 所以這裡明確蓋住**兩個來源都要覆蓋**：獨立檔案的 1／3／5，
// 以及 `SUM.MAP` 的每一段。
func TestLoadByIDCoversBothSources(t *testing.T) {
	dir := origDataDir(t)

	// 獨立檔案那三張。**1 是關鍵** —— 出貨存檔就是它。
	for _, id := range []int{1, 3, 5} {
		if _, err := LoadByID("", dir, id); err != nil {
			t.Errorf("地圖 %d（獨立檔案）載不到：%v", id, err)
		}
	}

	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range sm.IDs() {
		if _, err := LoadByID("", dir, id); err != nil {
			t.Errorf("地圖 %d（SUM.MAP 段）載不到：%v", id, err)
		}
	}
}

// 不存在的編號要回 error，而且訊息要說得出兩個來源都找過。
func TestLoadByIDUnknown(t *testing.T) {
	dir := origDataDir(t)
	_, err := LoadByID("", dir, 99)
	if err == nil {
		t.Fatal("地圖 99 不存在，應該回 error")
	}
	if !contains(err.Error(), "MAP99.MAP") || !contains(err.Error(), "SUM.MAP") {
		t.Errorf("錯誤訊息沒把兩個來源都講清楚：%v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// 存檔目錄裡的 `MAPn.MAP` 蓋過原版那一份。
//
// **這一條沒有的話，推開的家具與解開的密語門會在換張地圖之後長回去**
// （`docs/re/95` §3.9：原版改完 tile 就把整張圖寫回檔案）。
// 症狀是「謎題白解」，而畫面上完全看不出原因。
func TestLoadByIDPrefersSaveDir(t *testing.T) {
	dir := origDataDir(t)
	orig, err := LoadByID("", dir, 5)
	if err != nil {
		t.Skip(err)
	}
	// 造一份改過的：把 (11,18) 那面牆打開（密語門，`docs/re/84`）。
	if err := orig.SetTileAt(11, 18, 0); err != nil {
		t.Fatal(err)
	}
	save := t.TempDir()
	if err := SaveMap(filepath.Join(save, MapFileName(5)), orig); err != nil {
		t.Fatal(err)
	}

	got, err := LoadByID(save, dir, 5)
	if err != nil {
		t.Fatal(err)
	}
	if tile, _ := got.TileAt(11, 18); tile != 0 {
		t.Errorf("讀回來的 (11,18) = %#x，預期 0 —— 存檔目錄那一份沒被優先讀", tile)
	}
	// 存檔目錄沒有那個檔就退回原版。
	if _, err := LoadByID(t.TempDir(), dir, 5); err != nil {
		t.Errorf("存檔目錄沒有 MAP5.MAP 時退不回原版：%v", err)
	}
}

// 逐位元組往返：讀進來再寫出去要一模一樣，**含那個語意未解的 header**。
func TestMapRoundTripsByteForByte(t *testing.T) {
	dir := origDataDir(t)
	for _, id := range []int{1, 3, 5} {
		src := filepath.Join(dir, MapFileName(id))
		raw, err := os.ReadFile(src)
		if err != nil {
			t.Skip(err)
		}
		m, err := LoadMap(src)
		if err != nil {
			t.Fatal(err)
		}
		got := m.Encode()
		if len(got) != len(raw) {
			t.Fatalf("MAP%d 長度 %d，預期 %d", id, len(got), len(raw))
		}
		for i := range raw {
			if raw[i] != got[i] {
				t.Fatalf("MAP%d 第 %d 個 byte %#x → %#x", id, i, raw[i], got[i])
			}
		}
	}
}
