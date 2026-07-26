package world

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

// `EXITS.DAT` 的每一個目的地都要載得到。
//
// 引擎的換地圖（`cmd/demonwinter/mapchange.go`）依賴這件事。
// 不成立的話症狀是「走到某個樓梯就換不過去」—— 而那個樓梯可能在主線上，
// 玩家會卡在那裡，而畫面上只有一行錯誤訊息。
//
// 兩個來源：1／3／5 是獨立的 `MAPn.MAP`，其餘全在 `SUM.MAP` 的段裡。
// **這條測試就是在釘「兩個來源加起來蓋得住全部 13 個目的地編號」。**
func TestAllExitDestinationsLoadable(t *testing.T) {
	dir := origDataDir(t)
	table, err := LoadExits(filepath.Join(dir, "EXITS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatal(err)
	}

	// 獨立檔案的那三張。
	standalone := map[int]bool{}
	for _, id := range []int{1, 3, 5} {
		if _, err := LoadMap(filepath.Join(dir, mapFileName(id))); err == nil {
			standalone[id] = true
		}
	}

	dests := map[int]int{} // 目的地編號 → 幾筆用到它
	for _, r := range table.All() {
		dests[int(r.ToMap)]++
	}
	if len(dests) == 0 {
		t.Fatal("出口表沒有任何目的地 —— 解析大概壞了")
	}

	for id, n := range dests {
		if standalone[id] {
			continue
		}
		if _, ok := sm.Segment(id); ok {
			continue
		}
		t.Errorf("目的地地圖 %d（%d 筆出口用到）既沒有 MAP%d.MAP 也不在 SUM.MAP 裡",
			id, n, id)
	}
}

// mapFileName 是獨立地城檔的檔名。
func mapFileName(id int) string {
	return "MAP" + string(rune('0'+id)) + ".MAP"
}

// 出口的目的地一律不落在牆上。
//
// 這條是「踩過去會不會立刻被卡住」的最低要求。`docs/re/78` §6 記了
// 「目的地一律差一格」（不然會被彈回來），但沒有人驗過落點本身可通行 ——
// 落在牆上的話玩家換完地圖就動不了，而那看起來像引擎壞掉。
func TestExitDestinationsAreWalkable(t *testing.T) {
	dir := origDataDir(t)
	table, err := LoadExits(filepath.Join(dir, "EXITS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatal(err)
	}
	// 可通行性用引擎自己那一份（`gamedata.Tables`），不自己再解一次 ——
	// 測試用的判定與遊戲用的判定不一致，測試就沒有意義了。
	tables, err := gamedata.LoadTables(filepath.Join(dir, "FILES.DAT"))
	if err != nil {
		t.Skipf("讀不到 FILES.DAT，略過：%v", err)
	}

	maps := map[int]*Map{}
	getMap := func(id int) *Map {
		if m, ok := maps[id]; ok {
			return m
		}
		var m *Map
		switch id {
		case 1, 3, 5:
			m, _ = LoadMap(filepath.Join(dir, mapFileName(id)))
		default:
			if seg, ok := sm.Segment(id); ok {
				m = seg
			}
		}
		maps[id] = m
		return m
	}

	var blocked int
	for _, r := range table.All() {
		m := getMap(int(r.ToMap))
		if m == nil {
			continue
		}
		tile, err := m.TileAt(int(r.ToX), int(r.ToY))
		if err != nil {
			t.Errorf("圖%d 的落點 (%d,%d) 超出範圍", r.ToMap, r.ToX, r.ToY)
			continue
		}
		if tables.Passability(tile & 0x7f).Blocked() {
			blocked++
			t.Logf("圖%d 的落點 (%d,%d) 是 tile 0x%02x（可通行表 0xff ＝ 牆）",
				r.ToMap, r.ToX, r.ToY, tile)
		}
	}
	// 用實測值而不是「應該是 0」—— 先量出來，有了基準才知道之後的改動
	// 是修好了還是弄壞了。目前量到 0。
	if blocked != 0 {
		t.Errorf("有 %d 個出口落點在牆上（原本量到 0）", blocked)
	}
}

// 每一筆出口恰好被一張地圖認領。
//
// `EXITS.DAT` 沒有「來源地圖」欄位（`docs/re/78` §6），55 筆座標要分給
// 26 張地圖。分法是**可通行表值 `0xfd`**：同一組 (X,Y) 在別的地圖上
// 那一格不是 `0xfd`，就不算那張圖的出口（`docs/re/85`）。
//
// **這條測試就是那個判讀的證明。** 如果 `0xfd` 不是出口標記，
// 一筆記錄會被很多張圖同時認領（座標碰撞在 26 張 64×64 的圖上很常見），
// 或者一筆都認領不到。實測是 53 筆恰好一張、**0 筆多張** ——
// 撞號零次才是證據，「大多數只有一張」不是。
func TestEachExitClaimedByExactlyOneMap(t *testing.T) {
	dir := origDataDir(t)
	tbl, err := LoadExits(filepath.Join(dir, "EXITS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatal(err)
	}
	tables, err := gamedata.LoadTables(filepath.Join(dir, "FILES.DAT"))
	if err != nil {
		t.Skipf("讀不到 FILES.DAT，略過：%v", err)
	}

	ids := append([]int{1, 3, 5}, sm.IDs()...)
	claims := make([][]int, tbl.Len())
	for _, id := range ids {
		var m *Map
		switch id {
		case 1, 3, 5:
			m, _ = LoadMap(filepath.Join(dir, mapFileName(id)))
		default:
			if seg, ok := sm.Segment(id); ok {
				m = seg
			}
		}
		if m == nil {
			continue
		}
		for i, r := range tbl.All() {
			tile, err := m.TileAt(int(r.FromX), int(r.FromY))
			if err != nil {
				continue
			}
			if tables.Passability(tile&0x7f).Raw() == exitMarker {
				claims[i] = append(claims[i], id)
			}
		}
	}

	one, none, many := 0, 0, 0
	for i, c := range claims {
		switch len(c) {
		case 0:
			none++
		case 1:
			one++
		default:
			many++
			t.Errorf("出口 %d（%d,%d）被 %v 同時認領 —— 那就分不出它屬於哪張圖",
				i, tbl.All()[i].FromX, tbl.All()[i].FromY, c)
		}
	}
	if many != 0 {
		t.Fatalf("有 %d 筆被多張圖認領，`0xfd 是出口標記` 這個判讀不成立", many)
	}
	// 實測值，不是門檻。改動之後對不上就要重新解釋，不是把數字調鬆。
	if one != 53 || none != 2 {
		t.Errorf("認領分佈 = 恰好一張 %d、沒人 %d，原本量到 53／2", one, none)
	}
}

// exitMarker 是「這一格是出口」的可通行表值（與 cmd/demonwinter 的
// exitPassability 同一個值；分開寫是因為套件不同，改動時兩邊要一起改）。
const exitMarker = 0xfd
