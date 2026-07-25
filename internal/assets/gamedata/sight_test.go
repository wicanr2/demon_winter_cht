package gamedata

import (
	"path/filepath"
	"testing"
)

func loadSight(t *testing.T) *SightShadow {
	t.Helper()
	tb, err := LoadTables(filepath.Join(origDataDir(t), "FILES.DAT"))
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	return tb.Sight()
}

// 組數必須剛好 49，而且剛好用掉 161 bytes。
//
// 這條就是「表切對了沒」的關鍵：組數是從讀取端的迴圈邊界獨立算出來的
// （游標 10 起、內層 7 次、每輪 +2、到 73 為止 → 9×9 的內部 7×7 = 49 格），
// 不是從資料湊的。切法只要差一個 byte，組數就不會停在 49。
func TestSightShadow_HasFortyNineGroups(t *testing.T) {
	s := loadSight(t)

	// 有幾組是空的：資料裡用 0xff 表示「這一格什麼都遮不到」。
	total, empty := 0, 0
	for _, g := range s.shadow {
		if len(g) == 0 {
			empty++
		}
		total += len(g)
	}
	t.Logf("49 組裡有 %d 組是空的（0xff 哨兵）", empty)
	// 161 bytes 被消耗，其中空組各佔 1 byte（0xff）但貢獻 0 個元素。
	if total+empty != 161 {
		t.Errorf("49 組共 %d 個元素 + %d 個哨兵 = %d bytes，預期 161", total, empty, total+empty)
	}
}

// 每一個被遮蔽的格子，都必須比投影它的那一格離中心更遠、而且方向一致。
//
// 這是「這張表是以中央為視點的陰影錐」這個解讀的硬條件。表要是被切錯，
// 元素會亂跳，這條會立刻出現大量違反 —— 實際上 161 個元素零違反。
func TestSightShadow_PointsAwayFromCentre(t *testing.T) {
	s := loadSight(t)
	const centre = SightGridSize / 2

	sign := func(v int) int {
		switch {
		case v > 0:
			return 1
		case v < 0:
			return -1
		}
		return 0
	}
	maxAbs := func(a, b int) int {
		if a < 0 {
			a = -a
		}
		if b < 0 {
			b = -b
		}
		if a > b {
			return a
		}
		return b
	}

	for y := sightInteriorMin; y <= sightInteriorMax; y++ {
		for x := sightInteriorMin; x <= sightInteriorMax; x++ {
			dx, dy := x-centre, y-centre
			for _, c := range s.ShadowAt(x, y) {
				sx, sy := c%SightGridSize, c/SightGridSize
				ex, ey := sx-centre, sy-centre

				if maxAbs(ex, ey) <= maxAbs(dx, dy) {
					t.Errorf("格 (%d,%d) 遮到 (%d,%d)，但後者離中心沒有比較遠", x, y, sx, sy)
				}
				if (dx != 0 && sign(ex) != sign(dx)) || (dy != 0 && sign(ey) != sign(dy)) {
					t.Errorf("格 (%d,%d) 遮到 (%d,%d)，方向不一致", x, y, sx, sy)
				}
			}
		}
	}
}

// 最外圈不投影陰影 —— 它後面已經沒有格子了。界外也一樣。
func TestSightShadow_BorderCastsNothing(t *testing.T) {
	s := loadSight(t)
	for _, c := range [][2]int{
		{0, 0}, {4, 0}, {8, 8}, {0, 4}, {8, 4}, {4, 8},
		{-1, 4}, {9, 4}, {4, -1}, {4, 9},
	} {
		if got := s.ShadowAt(c[0], c[1]); got != nil {
			t.Errorf("(%d,%d) 不該投影陰影，卻回傳 %v", c[0], c[1], got)
		}
	}
}

// 擋視線的 tile 清單直接對照 0x17472 的判斷式。
func TestBlocksSight(t *testing.T) {
	for _, v := range []byte{0x0d, 0x12, 0x13, 0x2a, 0x31, 0x5e, 0x5f, 0x60, 0x61} {
		if !BlocksSight(v) {
			t.Errorf("tile 0x%02x 應該擋視線", v)
		}
	}
	// 邊界：0x5d 與 0x62 都在區間外。0x62 尤其要確認 —— 它是海面，
	// 戰場上不該因為一格海就遮住後面。
	for _, v := range []byte{0x00, 0x0c, 0x0e, 0x5d, 0x62, 0x14} {
		if BlocksSight(v) {
			t.Errorf("tile 0x%02x 不該擋視線", v)
		}
	}
}

// 整張戰場算一次：沒有任何擋視線地形時，不能有格子被遮住。
func TestHiddenCells_NoBlockersNoShadow(t *testing.T) {
	s := loadSight(t)
	tiles := make([]byte, sightCellCount) // 全 0，0 不在擋視線清單裡

	hidden, err := s.HiddenCells(tiles)
	if err != nil {
		t.Fatal(err)
	}
	for i, h := range hidden {
		if h {
			t.Errorf("沒有任何遮蔽物，第 %d 格卻被遮住", i)
		}
	}
}

// 放一個擋視線地形，被遮住的格子要與該格的陰影錐完全一致。
func TestHiddenCells_MatchesShadowOfBlocker(t *testing.T) {
	s := loadSight(t)
	const bx, by = 2, 2

	tiles := make([]byte, sightCellCount)
	tiles[by*SightGridSize+bx] = SightBlockerTiles[0]

	hidden, err := s.HiddenCells(tiles)
	if err != nil {
		t.Fatal(err)
	}

	want := map[int]bool{}
	for _, c := range s.ShadowAt(bx, by) {
		want[c] = true
	}
	if len(want) == 0 {
		t.Fatal("(2,2) 的陰影是空的，測試前提不成立")
	}
	for i, h := range hidden {
		if h != want[i] {
			t.Errorf("第 %d 格遮蔽 = %v，預期 %v", i, h, want[i])
		}
	}
}

func TestHiddenCells_RejectsWrongLength(t *testing.T) {
	s := loadSight(t)
	if _, err := s.HiddenCells(make([]byte, 80)); err == nil {
		t.Error("長度不是 81 應回傳 error")
	}
}
