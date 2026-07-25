package world

import (
	"os"
	"path/filepath"
	"testing"
)

// wantMapHeaders 是三個獨立 .MAP 檔已驗證的 header byte
// （見 docs/formats/town-and-map.md §2.1）。
var wantMapHeaders = map[string]byte{
	"MAP1.MAP": 0x00,
	"MAP3.MAP": 0x97,
	"MAP5.MAP": 0x09,
}

func TestLoadMap_RealFiles(t *testing.T) {
	dir := origDataDir(t)

	for name, wantHeader := range wantMapHeaders {
		name, wantHeader := name, wantHeader
		t.Run(name, func(t *testing.T) {
			m, err := LoadMap(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("LoadMap(%s) 失敗: %v", name, err)
			}

			if m.Header() != wantHeader {
				t.Errorf("%s header = 0x%02x, want 0x%02x", name, m.Header(), wantHeader)
			}

			if got := m.FilledCount(); got != mapTileCount {
				t.Errorf("%s FilledCount() = %d, want %d（獨立 .MAP 檔應該整張填滿）", name, got, mapTileCount)
			}

			tiles := m.Tiles()
			if len(tiles) != mapTileCount {
				t.Fatalf("%s Tiles() 長度 = %d, want %d", name, len(tiles), mapTileCount)
			}

			// 錨點：MAP1.MAP 座標 (0,0) 對應原始檔案 offset 1（header 之後第一個 byte）。
			tile00, err := m.TileAt(0, 0)
			if err != nil {
				t.Fatalf("%s TileAt(0,0) 失敗: %v", name, err)
			}
			if tile00 != tiles[0] {
				t.Errorf("%s TileAt(0,0) = %d, want %d (= tiles[0])", name, tile00, tiles[0])
			}

			// 錨點：座標 (63,63) 是陣列最後一格。
			tileLast, err := m.TileAt(MapWidth-1, MapHeight-1)
			if err != nil {
				t.Fatalf("%s TileAt(63,63) 失敗: %v", name, err)
			}
			if tileLast != tiles[mapTileCount-1] {
				t.Errorf("%s TileAt(63,63) = %d, want %d (= tiles[%d])", name, tileLast, tiles[mapTileCount-1], mapTileCount-1)
			}
		})
	}
}

func TestLoadMap_OutOfRange(t *testing.T) {
	dir := origDataDir(t)
	m, err := LoadMap(filepath.Join(dir, "MAP1.MAP"))
	if err != nil {
		t.Fatalf("LoadMap 失敗: %v", err)
	}

	cases := [][2]int{{-1, 0}, {0, -1}, {MapWidth, 0}, {0, MapHeight}}
	for _, c := range cases {
		if _, err := m.TileAt(c[0], c[1]); err == nil {
			t.Errorf("TileAt(%d,%d) 預期回傳 error，卻沒有", c[0], c[1])
		}
	}
}

func TestLoadMap_MissingFile(t *testing.T) {
	if _, err := LoadMap("/nonexistent/path/MAP1.MAP"); err == nil {
		t.Error("LoadMap 對不存在的檔案預期回傳 error，卻沒有")
	}
}

func TestLoadMap_WrongSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "BAD.MAP")
	if err := os.WriteFile(path, make([]byte, 100), 0o644); err != nil {
		t.Fatalf("寫測試檔失敗: %v", err)
	}
	if _, err := LoadMap(path); err == nil {
		t.Error("LoadMap 對長度不符的檔案預期回傳 error，卻沒有")
	}
}
