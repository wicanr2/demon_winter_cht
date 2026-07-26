package world

import (
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
		if _, err := LoadByID(dir, id); err != nil {
			t.Errorf("地圖 %d（獨立檔案）載不到：%v", id, err)
		}
	}

	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range sm.IDs() {
		if _, err := LoadByID(dir, id); err != nil {
			t.Errorf("地圖 %d（SUM.MAP 段）載不到：%v", id, err)
		}
	}
}

// 不存在的編號要回 error，而且訊息要說得出兩個來源都找過。
func TestLoadByIDUnknown(t *testing.T) {
	dir := origDataDir(t)
	_, err := LoadByID(dir, 99)
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
