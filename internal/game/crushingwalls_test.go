package game

import "testing"

// fakeMap 是一張 64×64 的空地圖，給壓牆測試用。
type fakeMap struct{ t [MapWidth * MapHeight]byte }

func (m *fakeMap) TileAt(x, y int) (byte, error) {
	if x < 0 || x >= MapWidth || y < 0 || y >= MapHeight {
		return 0, errOutOfMap
	}
	return m.t[y*MapWidth+x], nil
}

func (m *fakeMap) SetTileAt(x, y int, t byte) error {
	if x < 0 || x >= MapWidth || y < 0 || y >= MapHeight {
		return errOutOfMap
	}
	m.t[y*MapWidth+x] = t
	return nil
}

var errOutOfMap = errString("超出地圖")

type errString string

func (e errString) Error() string { return string(e) }

// 第一次推進填第 35 列與第 41 列（`0x4c − 35`），六格寬。
func TestCrushFirstStepFillsBothSides(t *testing.T) {
	m := &fakeMap{}
	res := AdvanceCrushingWalls(m, 1, 1)
	if res.Row != crushScanTop {
		t.Fatalf("填了第 %d 列，預期第 %d 列", res.Row, crushScanTop)
	}
	for x := crushColLeft; x <= crushColRight; x++ {
		for _, y := range []int{crushScanTop, crushMirror - crushScanTop} {
			if got, _ := m.TileAt(x, y); got != CrushWallTile {
				t.Errorf("(%d,%d) = %#x，預期牆 %#x", x, y, got, CrushWallTile)
			}
		}
	}
	// 走廊外面不能被動到。
	if got, _ := m.TileAt(crushColLeft-1, crushScanTop); got == CrushWallTile {
		t.Error("走廊左邊那一格也被填成牆了")
	}
}

// 連續推進會一列一列往中間夾，第 38 列是兩側的匯合點。
func TestCrushAdvancesRowByRow(t *testing.T) {
	m := &fakeMap{}
	for i, want := range []int{35, 36, 37, 38} {
		if got := AdvanceCrushingWalls(m, 1, 1).Row; got != want {
			t.Fatalf("第 %d 次推進到第 %d 列，預期第 %d 列", i+1, got, want)
		}
	}
}

// **站在被填的格子上就是死。** 這一條是整個機關的重點。
func TestCrushKillsPartyStandingInTheGap(t *testing.T) {
	m := &fakeMap{}
	if res := AdvanceCrushingWalls(m, crushColLeft, crushScanTop); !res.Crushed {
		t.Error("站在剛被填成牆的那一格卻沒被壓死")
	}
	// 走廊外面安全。
	m2 := &fakeMap{}
	if res := AdvanceCrushingWalls(m2, 1, 1); res.Crushed {
		t.Error("站在走廊外面卻被壓死了")
	}
}

// 鏡射那一側同樣會壓死人（下半列 = 76 − 上半列）。
func TestCrushKillsOnTheMirroredSide(t *testing.T) {
	m := &fakeMap{}
	y := crushMirror - crushScanTop // 41
	if res := AdvanceCrushingWalls(m, crushColRight, y); !res.Crushed {
		t.Errorf("站在鏡射側第 %d 列卻沒被壓死", y)
	}
}

// nil 地圖不 panic。
func TestCrushHandlesNilMap(t *testing.T) {
	if AdvanceCrushingWalls(nil, 1, 1).Crushed {
		t.Error("nil 地圖卻回報被壓死")
	}
}
