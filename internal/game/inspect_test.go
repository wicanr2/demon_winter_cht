package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func inspectTable() *scenario.ItemLocTable {
	return &scenario.ItemLocTable{Records: []scenario.ItemLoc{
		{X: 0x0e, Y: 0x16, MapID: 1},
		{X: 0x28, Y: 0x17, MapID: 3},
		{X: 0x0d, Y: 0x1a, MapID: 1},
		{X: 0x0d, Y: 0x1a, MapID: 1}, // 出貨資料真的有兩筆同座標
		{X: 0x11, Y: 0x22, MapID: scenario.ItemLocTaken},
	}}
}

// 只標目前這張子地圖的格子。
func TestInspectFiltersBySubmap(t *testing.T) {
	got := InspectSurroundings(inspectTable(), 1)
	if len(got) != 3 {
		t.Fatalf("子地圖 1 應有 3 格（含一對重複），得到 %d：%v", len(got), got)
	}
	for _, s := range got {
		if s.X == 0x28 {
			t.Errorf("標到了別張圖的格子：%v", s)
		}
	}
}

// 拿走的記錄（子地圖 0）不該出現在任何一張圖上。
func TestInspectSkipsTakenItems(t *testing.T) {
	for _, mapID := range []byte{1, 2, 3, 4, 5} {
		for _, s := range InspectSurroundings(inspectTable(), mapID) {
			if s.X == 0x11 && s.Y == 0x22 {
				t.Errorf("子地圖 %d 標到了已被拿走的道具", mapID)
			}
		}
	}
}

// 沒有位置表、或參數本身就是「已拿走」時回 nil，不 panic。
func TestInspectDegradesQuietly(t *testing.T) {
	if got := InspectSurroundings(nil, 1); got != nil {
		t.Errorf("沒有位置表卻回了 %v", got)
	}
	if got := InspectSurroundings(inspectTable(), scenario.ItemLocTaken); got != nil {
		t.Errorf("子地圖 0 不是一張真的圖，卻回了 %v", got)
	}
}
