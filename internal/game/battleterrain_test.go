package game

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

func loadWorldMap(t *testing.T) *world.Map {
	t.Helper()
	m, err := world.LoadMap(filepath.Join(repoRoot(t), "workplace", "orig", "demwin", "DEM_DATA", "MAP1.MAP"))
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}
	return m
}

// 幾何：15×15 可站範圍位在 (6,6)–(20,20)，中心 (13,13)，3×3 個 5×5 區塊。
func TestBattleField_Geometry(t *testing.T) {
	if BattleFieldMin != 6 || BattleFieldMax != 20 {
		t.Errorf("可站範圍 %d–%d，預期 6–20", BattleFieldMin, BattleFieldMax)
	}
	if BattleFieldSize != 15 {
		t.Errorf("邊長 %d，預期 15", BattleFieldSize)
	}
	if BattleCentreX != 13 || BattleCentreY != 13 {
		t.Errorf("中心 (%d,%d)，預期 (13,13)", BattleCentreX, BattleCentreY)
	}
	if BattleBlocks != 3 || BattleBlockSize != 5 {
		t.Errorf("區塊 %d×%d，預期 3 個 5 格", BattleBlocks, BattleBlockSize)
	}
}

// 每一個世界 tile 都攤成一整塊 5×5 —— 這是「放大地圖」的具體意思。
func TestBattleTerrain_ExpandsEachWorldTileToFiveByFive(t *testing.T) {
	m := loadWorldMap(t)
	const cx, cy = 32, 32

	bt, err := NewBattleTerrain(m, cx, cy)
	if err != nil {
		t.Fatal(err)
	}
	for by := 0; by < BattleBlocks; by++ {
		for bx := 0; bx < BattleBlocks; bx++ {
			want, err := m.TileAt(cx-1+bx, cy-1+by)
			if err != nil {
				t.Fatal(err)
			}
			ox := BattleFieldMin + bx*BattleBlockSize
			oy := BattleFieldMin + by*BattleBlockSize
			for dy := 0; dy < BattleBlockSize; dy++ {
				for dx := 0; dx < BattleBlockSize; dx++ {
					if got := bt.TileAt(ox+dx, oy+dy); got != want {
						t.Fatalf("區塊 (%d,%d) 的 (%d,%d) 是 %d，預期 %d",
							bx, by, dx, dy, got, want)
					}
				}
			}
		}
	}
}

// 中心那一格就是隊伍腳下的世界 tile —— 空地值的來源。
func TestBattleTerrain_GroundIsThePartyTile(t *testing.T) {
	m := loadWorldMap(t)
	const cx, cy = 32, 32

	bt, err := NewBattleTerrain(m, cx, cy)
	if err != nil {
		t.Fatal(err)
	}
	want, err := m.TileAt(cx, cy)
	if err != nil {
		t.Fatal(err)
	}
	if got := bt.Ground(); got != want {
		t.Errorf("空地值 %d，預期隊伍腳下的 %d", got, want)
	}
	if !bt.Walkable(BattleCentreX, BattleCentreY) {
		t.Error("隊伍腳下那一格一定站得上去")
	}
}

// 牆框在第 5 與第 21 列／欄，而且擋得住。
func TestBattleTerrain_WallRing(t *testing.T) {
	m := loadWorldMap(t)
	bt, err := NewBattleTerrain(m, 32, 32)
	if err != nil {
		t.Fatal(err)
	}
	for i := BattleWallLow; i <= BattleWallHigh; i++ {
		for _, p := range [][2]int{
			{i, BattleWallLow}, {i, BattleWallHigh},
			{BattleWallLow, i}, {BattleWallHigh, i},
		} {
			if got := bt.TileAt(p[0], p[1]); got != BattleWallTile {
				t.Fatalf("(%d,%d) 是 %d，預期牆 %d", p[0], p[1], got, BattleWallTile)
			}
			if bt.Walkable(p[0], p[1]) {
				t.Fatalf("(%d,%d) 是牆，不該走得過去", p[0], p[1])
			}
		}
	}
}

// 世界地圖界外的區塊填成牆 —— **不能留 0**，因為 0 是合法的世界 tile，
// 拿它當哨兵會讓地形 0 的戰場整片畫不出來。
func TestBattleTerrain_OutsideWorldBecomesWall(t *testing.T) {
	m := loadWorldMap(t)
	bt, err := NewBattleTerrain(m, 0, 0) // 左上角，西側與北側都在界外
	if err != nil {
		t.Fatal(err)
	}
	if got := bt.TileAt(BattleFieldMin, BattleFieldMin); got != BattleWallTile {
		t.Errorf("界外區塊是 %d，預期牆 %d", got, BattleWallTile)
	}
	if bt.Walkable(BattleFieldMin, BattleFieldMin) {
		t.Error("界外那一塊不該站得上去")
	}
}

// 0 是合法的世界 tile —— 這條測試就是在釘「別拿 0 當哨兵」。
//
// MAP1 上 tile 0 到處都是，所以 `v != 0` 那種畫法會把一整片戰場畫成黑的。
// 該畫不該畫要看座標（InArena），不是看值。
func TestBattleTerrain_ZeroIsARealTile(t *testing.T) {
	m := loadWorldMap(t)
	bt, err := NewBattleTerrain(m, 32, 32)
	if err != nil {
		t.Fatal(err)
	}
	zero := false
	for y := BattleFieldMin; y <= BattleFieldMax && !zero; y++ {
		for x := BattleFieldMin; x <= BattleFieldMax; x++ {
			if bt.TileAt(x, y) == 0 {
				zero = true
				break
			}
		}
	}
	if !zero {
		t.Skip("這個位置剛好沒有 tile 0，換個地方再測")
	}
	if !InArena(BattleFieldMin, BattleFieldMin) {
		t.Error("可站範圍內的格子一定在戰場裡")
	}
	if InArena(BattleWallLow-1, BattleWallLow-1) {
		t.Error("牆框外的格子不該算在戰場裡")
	}
}

func TestBattleTerrain_NeedsAMap(t *testing.T) {
	if _, err := NewBattleTerrain(nil, 10, 10); err == nil {
		t.Error("沒有地圖應該回 error")
	}
}

func tile4TestMap(t *testing.T, centre, east, south byte) *world.Map {
	t.Helper()
	m := &world.Map{}
	const cx, cy = 32, 32
	for y := cy - 1; y <= cy+1; y++ {
		for x := cx - 1; x <= cx+1; x++ {
			if err := m.SetTileAt(x, y, 7); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, v := range []struct {
		x, y int
		tile byte
	}{
		{cx, cy, centre},
		{cx + 1, cy, east},
		{cx, cy + 1, south},
	} {
		if err := m.SetTileAt(v.x, v.y, v.tile); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

func TestBattleTerrain_Tile4HorizontalNarrowWay(t *testing.T) {
	const cx, cy = 32, 32
	bt, err := NewBattleTerrain(tile4TestMap(t, 4, 0, 9), cx, cy)
	if err != nil {
		t.Fatal(err)
	}
	ox, oy := BattleFieldMin+BattleBlockSize, BattleFieldMin+BattleBlockSize
	for dy := 0; dy < BattleBlockSize; dy++ {
		for dx := 0; dx < BattleBlockSize; dx++ {
			want := byte(9)
			if dy == BattleBlockSize/2 {
				want = 0
			}
			if dx == BattleBlockSize/2 && dy == BattleBlockSize/2 {
				want = 4
			}
			if got := bt.TileAt(ox+dx, oy+dy); got != want {
				t.Fatalf("tile 4 橫向窄道 (%d,%d) = %d，預期 %d", dx, dy, got, want)
			}
		}
	}
}

func TestBattleTerrain_Tile4VerticalNarrowWay(t *testing.T) {
	const cx, cy = 32, 32
	bt, err := NewBattleTerrain(tile4TestMap(t, 4, 8, 9), cx, cy)
	if err != nil {
		t.Fatal(err)
	}
	ox, oy := BattleFieldMin+BattleBlockSize, BattleFieldMin+BattleBlockSize
	for dy := 0; dy < BattleBlockSize; dy++ {
		for dx := 0; dx < BattleBlockSize; dx++ {
			want := byte(4)
			if dx == BattleBlockSize/2 && dy != BattleBlockSize/2 {
				want = 0
			}
			if got := bt.TileAt(ox+dx, oy+dy); got != want {
				t.Fatalf("tile 4 縱向窄道 (%d,%d) = %d，預期 %d", dx, dy, got, want)
			}
		}
	}
}
