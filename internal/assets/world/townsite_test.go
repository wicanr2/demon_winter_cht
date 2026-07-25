package world

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

// 城鎮座標表（從 DEMON.INT 機器碼讀出）與地圖資料（從 SUM.MAP／MAP1.MAP
// 解壓）必須互相對得上：25 個座標每一個都要落在一格城鎮 tile 上。
//
// 這兩份資料完全獨立 —— 一份在執行檔的程式碼段，一份在資料檔裡壓縮著 ——
// 所以它是本專案少數「錯了就一定看得出來」的檢查之一，而不是自己驗自己。
// SUM.MAP 的 RLE 解壓錯 256 格或差一個 byte，這條就會從 25/25 掉到 1/25。
//
// 它同時也守住反方向：座標表要是抄錯一個數字，同樣會掉分。
func TestTownSites_LandOnTownTiles(t *testing.T) {
	dir := origDataDir(t)

	// 蒐集全世界的地圖：獨立 .MAP 檔 + SUM.MAP 的 23 個子地圖。
	type surface struct {
		name  string
		tiles []byte
	}
	var world []surface
	for _, n := range []string{"MAP1.MAP", "MAP3.MAP", "MAP5.MAP"} {
		m, err := LoadMap(filepath.Join(dir, n))
		if err != nil {
			t.Fatalf("LoadMap(%s): %v", n, err)
		}
		tiles := m.Tiles()
		world = append(world, surface{n, append([]byte(nil), tiles[:]...)})
	}
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatalf("LoadSumMap: %v", err)
	}
	for _, id := range sm.IDs() {
		m, _ := sm.Segment(id)
		tiles := m.Tiles()
		world = append(world, surface{
			name:  "SUM 子地圖 " + itoa(id),
			tiles: append([]byte(nil), tiles[:]...),
		})
	}

	tt, err := gamedata.LoadTownTable(dir)
	if err != nil {
		t.Fatalf("LoadTownTable: %v", err)
	}

	hit := 0
	for _, town := range tt.All() {
		var found string
		for _, s := range world {
			if v := s.tiles[town.Y*MapWidth+town.X]; gamedata.IsTownTile(v) {
				found = s.name
				break
			}
		}
		if found == "" {
			t.Errorf("城鎮 %d %s (%d,%d)：整個世界都沒有城鎮 tile",
				town.Number, town.Name, town.X, town.Y)
			continue
		}
		hit++
		t.Logf("城鎮 %2d %-16s (%2d,%2d) → %s", town.Number, town.Name, town.X, town.Y, found)
	}
	if hit != gamedata.NumTowns {
		t.Errorf("只有 %d/%d 座城鎮對上地圖", hit, gamedata.NumTowns)
	}
}

// 反方向：地圖上的城鎮 tile 幾乎都要在座標表裡。
//
// 只查單向（表 → 地圖）會漏掉「地圖上有第 26 個城鎮但表只有 25 筆」這種錯。
// 目前全世界 27 格城鎮 tile，25 格對上表，多出來的 2 格都在子地圖 54，
// 語意未解 —— 所以門檻設在「未對上的不超過 2 格」，多一格就要回頭查。
func TestTownTiles_MostlyAccountedFor(t *testing.T) {
	dir := origDataDir(t)
	tt, err := gamedata.LoadTownTable(dir)
	if err != nil {
		t.Fatalf("LoadTownTable: %v", err)
	}

	var all [][3]int // x, y, tile
	scan := func(tiles []byte) {
		for i, v := range tiles {
			if gamedata.IsTownTile(v) {
				all = append(all, [3]int{i % MapWidth, i / MapWidth, int(v)})
			}
		}
	}
	for _, n := range []string{"MAP1.MAP", "MAP3.MAP", "MAP5.MAP"} {
		m, err := LoadMap(filepath.Join(dir, n))
		if err != nil {
			t.Fatalf("LoadMap(%s): %v", n, err)
		}
		tiles := m.Tiles()
		scan(tiles[:])
	}
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatalf("LoadSumMap: %v", err)
	}
	for _, id := range sm.IDs() {
		m, _ := sm.Segment(id)
		tiles := m.Tiles()
		scan(tiles[:])
	}

	const maxOrphans = 2
	orphans := 0
	for _, p := range all {
		if _, ok := tt.TownAt(p[0], p[1]); !ok {
			orphans++
			t.Logf("城鎮 tile 0x%02x 在 (%d,%d) 不在座標表裡", p[2], p[0], p[1])
		}
	}
	if orphans > maxOrphans {
		t.Errorf("有 %d 格城鎮 tile 對不到城鎮表，超過已知的 %d 格", orphans, maxOrphans)
	}
	t.Logf("全世界 %d 格城鎮 tile，%d 格對上城鎮表", len(all), len(all)-orphans)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
