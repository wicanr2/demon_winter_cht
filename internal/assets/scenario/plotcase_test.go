package scenario

import (
	"os"
	"path/filepath"
	"testing"
)

// 這一組測試釘住 `docs/re/83` §2 那條**可以被資料打死**的預測：
//
//	nSS.DAT 類別 5 的「值」是全域唯一的事件編號。
//
// 判讀鏈是「`cmp ds:0x5c62,5` → 16 格跳表 → 選擇子是 `ds:0x52f6` ＝ attr 低 5 bit」，
// 三段都在反組譯裡看得到。但反組譯讀起來合理不等於對 ——
// 如果那個值只是某種區域旗標，五張獨立的地圖各自編號，撞號幾乎必然。
// **零碰撞才是證據**，所以把它釘成測試而不是寫在文件裡就算了。

// allSSPath 是母本的位置。母本沒被遊玩污染，是唯一可信的普查來源
//（磁碟上的 `nSS.DAT` 出廠就是髒的，見 `docs/re/78`）。
func allSSPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(dataDir, "ALL_SS.DAT")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("找不到原版資料 %s，略過：%v", p, err)
	}
	return p
}

func loadAllSS(t *testing.T) []*SpecialTiles {
	t.Helper()
	raw, err := os.ReadFile(allSSPath(t))
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := SplitAllSS(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]*SpecialTiles, 0, len(blocks))
	for i, b := range blocks {
		st, err := ParseSpecialTiles(b)
		if err != nil {
			t.Fatalf("地圖 %d 解析失敗：%v", i+1, err)
		}
		out = append(out, st)
	}
	return out
}

// 類別 5 的值跨五張地圖不重複，而且恰好用滿 1–15。
func TestPlotCasesAreGloballyUnique(t *testing.T) {
	owner := map[int]int{} // case → 哪張地圖
	for i, st := range loadAllSS(t) {
		mapID := i + 1
		for _, tile := range st.Tiles {
			c := tile.PlotCase()
			if c < 0 {
				continue
			}
			if prev, ok := owner[c]; ok && prev != mapID {
				t.Errorf("case %d 同時出現在地圖 %d 與 %d —— 不是全域唯一的編號",
					c, prev, mapID)
			}
			owner[c] = mapID
		}
	}

	// 16 格表的 case 0 是預設分支，資料不該用它。
	for c := 1; c <= 15; c++ {
		if _, ok := owner[c]; !ok {
			t.Errorf("case %d 在資料裡找不到 —— 16 格表應該被 1–15 用滿", c)
		}
	}
	if _, ok := owner[0]; ok {
		t.Error("case 0 是預設分支，資料不該用到")
	}
	if len(owner) != 15 {
		t.Errorf("類別 5 的相異值 = %d，預期 15", len(owner))
	}
}

// 對照組：同樣的「全域唯一」檢查套在類別 1／2 上必須失敗。
//
// 沒有對照組的話，上面那條測試只是在描述資料，不是在區分假說。
// 類別 1／2 的值恆為 0（索引由掃描計數器給），所以它們一定會撞號 ——
// 這證明「零碰撞」不是 `nSS.DAT` 的普遍性質，是類別 5 獨有的。
func TestEventClassValuesAreNotUnique(t *testing.T) {
	seen := map[byte]int{}
	collisions := 0
	for i, st := range loadAllSS(t) {
		mapID := i + 1
		for _, tile := range st.Tiles {
			cls := tile.Class()
			if cls != SpecialClassEventA && cls != SpecialClassEventB {
				continue
			}
			if prev, ok := seen[tile.Value()]; ok && prev != mapID {
				collisions++
			}
			seen[tile.Value()] = mapID
		}
	}
	if collisions == 0 {
		t.Error("類別 1／2 的值居然也全域唯一 —— 那 case 5 的零碰撞就不算證據了")
	}
}

// 艾瑞戈爾那一格全遊戲只有一個，在地圖 1 的 (60,1)。
//
// 引擎把 case 14 直接接到艾瑞戈爾（`locationPlot`），沒有再檢查地圖編號 ——
// 這條測試就是那個簡化的前提。多出第二格的話這裡會先炸。
func TestEregoreTileIsUnique(t *testing.T) {
	var found []struct {
		mapID int
		x, y  byte
	}
	for i, st := range loadAllSS(t) {
		for _, tile := range st.Tiles {
			if tile.PlotCase() == PlotCaseEregore {
				found = append(found, struct {
					mapID int
					x, y  byte
				}{i + 1, tile.X, tile.Y})
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("case %d 有 %d 格，預期 1：%+v", PlotCaseEregore, len(found), found)
	}
	if found[0].mapID != 1 || found[0].x != 60 || found[0].y != 1 {
		t.Errorf("艾瑞戈爾在地圖 %d 的 (%d,%d)，預期地圖 1 的 (60,1)",
			found[0].mapID, found[0].x, found[0].y)
	}
}
