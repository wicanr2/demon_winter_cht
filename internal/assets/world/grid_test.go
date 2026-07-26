package world

import (
	"path/filepath"
	"testing"
)

// 四個方向各走出去一次，編號與座標都要對。
//
// 這四條直接對應反組譯的四段（0x16fec／0x17037／0x17082／0x170cd）。
// X 方向動十位數、Y 方向動個位數 —— 兩個軸弄反的話，玩家往東走會跑到
// 南邊那張圖，而且不會有任何錯誤訊息。
func TestCrossEdge_FourDirections(t *testing.T) {
	const start = 44 // 欄 4、列 4，四面都有鄰居

	for _, c := range []struct {
		name   string
		x, y   int
		wantID int
		wantX  int
		wantY  int
	}{
		{"往西", edgeLow, 30, 34, WalkMax, 30},
		{"往東", edgeHigh, 30, 54, WalkMin, 30},
		{"往北", 30, edgeLow, 43, 30, WalkMax},
		{"往南", 30, edgeHigh, 45, 30, WalkMin},
	} {
		got := CrossEdge(start, c.x, c.y)
		if !got.Crossed || got.Blocked {
			t.Errorf("%s：Crossed=%v Blocked=%v，預期換圖成功", c.name, got.Crossed, got.Blocked)
			continue
		}
		if got.MapID != c.wantID || got.X != c.wantX || got.Y != c.wantY {
			t.Errorf("%s：得到 圖 %d (%d,%d)，預期 圖 %d (%d,%d)",
				c.name, got.MapID, got.X, got.Y, c.wantID, c.wantX, c.wantY)
		}
	}
}

// 世界邊緣過不去，而且座標不能被改動。
//
// 擋住時原版是「座標還原、回一個不同的返回碼」；這裡若誤把座標送到
// 另一側，玩家會在邊界瞬移到同一張圖的對面。
func TestCrossEdge_BlockedAtWorldEdge(t *testing.T) {
	for _, c := range []struct {
		name  string
		mapID int
		x, y  int
	}{
		{"最西欄往西", 17, edgeLow, 30},
		{"最東欄往東", 77, edgeHigh, 30},
		{"最北列往北", 41, 30, edgeLow},
		{"最南列往南", 47, 30, edgeHigh},
	} {
		got := CrossEdge(c.mapID, c.x, c.y)
		if !got.Blocked {
			t.Errorf("%s：預期被擋住，卻得到 圖 %d (%d,%d)", c.name, got.MapID, got.X, got.Y)
		}
		if got.MapID != c.mapID || got.X != c.x || got.Y != c.y {
			t.Errorf("%s：被擋住時不該改動位置，卻變成 圖 %d (%d,%d)",
				c.name, got.MapID, got.X, got.Y)
		}
	}
}

// 不在邊界上就什麼都不做。
func TestCrossEdge_NoOpInside(t *testing.T) {
	for _, xy := range [][2]int{{WalkMin, WalkMin}, {30, 30}, {WalkMax, WalkMax}} {
		got := CrossEdge(44, xy[0], xy[1])
		if got.Crossed || got.Blocked {
			t.Errorf("(%d,%d) 不在邊界上，卻回報 Crossed=%v Blocked=%v",
				xy[0], xy[1], got.Crossed, got.Blocked)
		}
	}
}

// 地城沒有這套換圖規則。
func TestCrossEdge_IgnoresDungeons(t *testing.T) {
	for _, id := range []int{1, 2, 3, 4, 5} {
		got := CrossEdge(id, edgeLow, 30)
		if got.Crossed || got.Blocked {
			t.Errorf("地城 %d 不該套用世界換圖規則", id)
		}
	}
}

func TestSubMapID_RoundTrip(t *testing.T) {
	for col := SubMapMinAxis; col <= SubMapMaxAxis; col++ {
		for row := SubMapMinAxis; row <= SubMapMaxAxis; row++ {
			id, err := SubMapID(col, row)
			if err != nil {
				t.Fatal(err)
			}
			if SubMapColumn(id) != col || SubMapRow(id) != row {
				t.Errorf("編號 %d 拆回 (欄 %d, 列 %d)，預期 (%d,%d)",
					id, SubMapColumn(id), SubMapRow(id), col, row)
			}
			if !IsWorldSubMap(id) {
				t.Errorf("編號 %d 應被視為世界子地圖", id)
			}
		}
	}
	if _, err := SubMapID(0, 3); err == nil {
		t.Error("欄 0 應回傳錯誤")
	}
	if _, err := SubMapID(3, 8); err == nil {
		t.Error("列 8 應回傳錯誤")
	}
}

// SUM.MAP 裡真實存在的子地圖編號，全部都要落在 7×7 網格內。
//
// 這條把「網格範圍是欄列各 1–7」拿真實資料再驗一次：換圖程式碼說範圍是
// 1–7，資料裡要是冒出 0 或 8 開頭的編號，就代表其中一邊讀錯了。
func TestSumMapIDs_FitTheGrid(t *testing.T) {
	dir := origDataDir(t)
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatal(err)
	}
	world, dungeon := 0, 0
	for _, id := range sm.IDs() {
		if id < SubMapMinID {
			dungeon++ // 編號 < 11 是地城，不在世界網格上
			continue
		}
		if !IsWorldSubMap(id) {
			t.Errorf("子地圖 %d 既不是地城、也不落在 7×7 網格內（欄 %d 列 %d）",
				id, SubMapColumn(id), SubMapRow(id))
			continue
		}
		world++
	}
	t.Logf("SUM.MAP：世界子地圖 %d 張、地城 %d 張", world, dungeon)
	if world == 0 {
		t.Error("一張世界子地圖都沒有，測試前提不成立")
	}
}
