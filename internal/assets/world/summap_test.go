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

// TestLoadSumMap_FilledCounts 誠實記錄每個 sub-map 實際解壓出的格數，
// 不強行斷言全部等於 4096——依 decodeRLE 的文件說明，19/23 個區塊會
// 因來源位元組提前耗盡而少於 4096。這裡把實際數字印出來供人工核對，
// 並只對「不可能超過 4096、不可能是負數」這種結構性條件斷言。
func TestLoadSumMap_FilledCounts(t *testing.T) {
	dir := origDataDir(t)
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatalf("LoadSumMap 失敗: %v", err)
	}

	fullCount := 0
	for _, id := range sm.IDs() {
		m, _ := sm.Segment(id)
		filled := m.FilledCount()
		t.Logf("id=%-3d filled=%d/%d", id, filled, mapTileCount)
		if filled < 0 || filled > mapTileCount {
			t.Errorf("id=%d FilledCount()=%d 超出合理範圍 [0,%d]", id, filled, mapTileCount)
		}
		if filled == mapTileCount {
			fullCount++
		}
	}
	t.Logf("23 個子地圖中，恰好填滿 4096 格的有 %d 個（其餘因來源資料提前耗盡而未填滿，見 decodeRLE 文件說明）", fullCount)
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
