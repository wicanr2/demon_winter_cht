package world

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSumMapSizeTable_SumsToFileSize 是最硬的驗證錨點：23 筆 size 表加總
// 必須精確等於 SUM.MAP 的實際檔案大小 15,743（見
// docs/re/03-audio-and-resources.md §3.2）。這條測試不需要真實檔案也能跑
// （純檢查常數表本身），真實檔案大小比對放在 TestLoadSumMap_RealFile。
func TestSumMapSizeTable_SumsToFileSize(t *testing.T) {
	const want = 15743
	if got := sumMapTotalSize(); got != want {
		t.Fatalf("23 筆 size 表加總 = %d, want %d", got, want)
	}
	if len(sumMapIDs) != sumMapSegmentCount || len(sumMapSizes) != sumMapSegmentCount {
		t.Fatalf("sumMapIDs/sumMapSizes 長度應為 %d，實際 %d/%d", sumMapSegmentCount, len(sumMapIDs), len(sumMapSizes))
	}
}

func TestLoadSumMap_RealFile(t *testing.T) {
	dir := origDataDir(t)
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatalf("LoadSumMap 失敗: %v", err)
	}

	ids := sm.IDs()
	if len(ids) != sumMapSegmentCount {
		t.Fatalf("IDs() 長度 = %d, want %d", len(ids), sumMapSegmentCount)
	}

	// 驗收要求：map ID 集合必須包含 2 與 4（這兩個地城沒有獨立 .MAP 檔，
	// ITEMLOCB.DAT 引用了它們，見 docs/formats/town-and-map.md §5-3、
	// docs/re/03-audio-and-resources.md §3.2 尾段）。
	if _, ok := sm.Segment(2); !ok {
		t.Error("SumMap 缺少 ID=2 的子地圖")
	}
	if _, ok := sm.Segment(4); !ok {
		t.Error("SumMap 缺少 ID=4 的子地圖")
	}

	// 每個 sub-map 解壓後應該是 64x64 的 Map（Tiles() 長度固定 4096），
	// 且 tile 值域應落在 [0,127]（RLE 的高位元被拆掉，見 decodeRLE）。
	for _, id := range ids {
		m, ok := sm.Segment(id)
		if !ok {
			t.Errorf("Segment(%d) 不存在", id)
			continue
		}
		tiles := m.Tiles()
		if len(tiles) != mapTileCount {
			t.Errorf("id=%d Tiles() 長度 = %d, want %d", id, len(tiles), mapTileCount)
		}
		for _, v := range tiles {
			if v > 0x7f {
				t.Errorf("id=%d 出現值域外的 tile 值 %d（RLE 應已拆掉高位元）", id, v)
				break
			}
		}
	}
}

// 23 個子地圖每一個都要精確填滿 4096 格。
//
// 這條原本寫成「誠實記錄格數、不斷言等於 4096」，理由是當時實測只有
// 4/23 填滿，就照著寫進文件說「來源資料會提前耗盡」。那不是格式特性，
// 是 decodeRLE 的兩個 bug（次數 0 沒當 256、沒跳過區塊首 byte）；把
// 觀察到的錯誤結果寫成規格，等於用測試把 bug 釘死。
//
// 缺格數一定是 256 的倍數（少解一個 256 連），所以這條抓得很準。
func TestLoadSumMap_AllSegmentsFill(t *testing.T) {
	dir := origDataDir(t)
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatalf("LoadSumMap 失敗: %v", err)
	}

	for _, id := range sm.IDs() {
		m, _ := sm.Segment(id)
		if filled := m.FilledCount(); filled != mapTileCount {
			t.Errorf("子地圖 %d 只填了 %d/%d 格，缺 %d 格",
				id, filled, mapTileCount, mapTileCount-filled)
		}
	}
}

func TestLoadSumMap_WrongSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "BAD.MAP")
	if err := os.WriteFile(path, make([]byte, 100), 0o644); err != nil {
		t.Fatalf("寫測試檔失敗: %v", err)
	}
	if _, err := LoadSumMap(path); err == nil {
		t.Error("LoadSumMap 對長度不符的檔案預期回傳 error，卻沒有")
	}
}
