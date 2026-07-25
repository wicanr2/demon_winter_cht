package game

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("game: runtime.Caller 失敗")
	}
	// file 是 .../internal/game/move_test.go
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func loadTables(t *testing.T) *gamedata.Tables {
	t.Helper()
	p := filepath.Join(repoRoot(t), "workplace", "orig", "demwin", "DEM_DATA", "FILES.DAT")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("game: 找不到 %s，略過需要真實資料的測試", p)
	}
	tb, err := gamedata.LoadTables(p)
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	return tb
}

// fakeTiles 是可控的 64×64 圖塊來源，用來構造特定的通行情境。
type fakeTiles struct {
	def   byte
	cells map[[2]int]byte
}

func newFakeTiles(def byte) *fakeTiles {
	return &fakeTiles{def: def, cells: map[[2]int]byte{}}
}

func (f *fakeTiles) set(x, y int, tile byte) { f.cells[[2]int{x, y}] = tile }

func (f *fakeTiles) TileAt(x, y int) (byte, error) {
	if x < 0 || x >= MapWidth || y < 0 || y >= MapHeight {
		return 0, fmt.Errorf("座標 (%d,%d) 超出地圖", x, y)
	}
	if t, ok := f.cells[[2]int{x, y}]; ok {
		return t, nil
	}
	return f.def, nil
}

// 驗收 1：面向位移表與線性位移必須一致。
func TestFacing_DeltaTable(t *testing.T) {
	cases := []struct {
		f          Facing
		dx, dy, ld int
	}{
		{North, 0, -1, -64},
		{East, 1, 0, 1},
		{South, 0, 1, 64},
		{West, -1, 0, -1},
	}
	for _, tc := range cases {
		dx, dy := tc.f.delta()
		if dx != tc.dx || dy != tc.dy {
			t.Errorf("面向 %d：位移得到 (%d,%d)，預期 (%d,%d)", tc.f, dx, dy, tc.dx, tc.dy)
		}
		if got := tc.f.LinearDelta(); got != tc.ld {
			t.Errorf("面向 %d：線性位移得到 %d，預期 %d", tc.f, got, tc.ld)
		}
	}
}

// 找出可通行性表中一個可走、一個不可走的 tile，讓測試不寫死 tile 值。
func pickTiles(t *testing.T, tb *gamedata.Tables) (walkable, wall byte) {
	t.Helper()
	found := [2]bool{}
	for i := 0; i <= 100; i++ {
		b := tb.Passability(byte(i)).Blocked()
		if !b && !found[0] {
			walkable, found[0] = byte(i), true
		}
		if b && !found[1] {
			wall, found[1] = byte(i), true
		}
	}
	if !found[0] || !found[1] {
		t.Fatal("可通行性表中找不到可走與不可走的 tile 各一")
	}
	return walkable, wall
}

func TestWalk_MovesOnWalkableTile(t *testing.T) {
	tb := loadTables(t)
	walkable, _ := pickTiles(t, tb)

	tiles := newFakeTiles(walkable)
	w := NewWorld(tiles, tb)
	p := NewParty(10, 10, East, 0)
	c := NewClock()

	res, _, _ := w.Walk(p, c)

	if res != MoveOK {
		t.Fatalf("應可通行，得到結果 %d", res)
	}
	if p.X() != 11 || p.Y() != 10 {
		t.Errorf("座標應為 (11,10)，得到 (%d,%d)", p.X(), p.Y())
	}
}

func TestWalk_BlockedByWall(t *testing.T) {
	tb := loadTables(t)
	walkable, wall := pickTiles(t, tb)

	tiles := newFakeTiles(walkable)
	tiles.set(11, 10, wall)
	w := NewWorld(tiles, tb)
	p := NewParty(10, 10, East, 0)
	c := NewClock()

	res, _, _ := w.Walk(p, c)

	if res != MoveBlocked {
		t.Fatalf("應撞牆，得到結果 %d", res)
	}
	if p.X() != 10 || p.Y() != 10 {
		t.Errorf("撞牆後座標不應改變，得到 (%d,%d)", p.X(), p.Y())
	}
	if c.Steps() != 0 {
		t.Errorf("撞牆不應計步，計數器為 %d", c.Steps())
	}
}

func TestWalk_MapBoundary(t *testing.T) {
	tb := loadTables(t)
	walkable, _ := pickTiles(t, tb)

	tiles := newFakeTiles(walkable)
	w := NewWorld(tiles, tb)
	c := NewClock()

	// 站在左上角往北、往西都應該被擋住。
	for _, f := range []Facing{North, West} {
		p := NewParty(0, 0, f, 0)
		if res, _, _ := w.Walk(p, c); res != MoveBlocked {
			t.Errorf("面向 %d 走出地圖邊界應被擋住，得到 %d", f, res)
		}
		if p.X() != 0 || p.Y() != 0 {
			t.Errorf("面向 %d：座標不應改變，得到 (%d,%d)", f, p.X(), p.Y())
		}
	}
}

// 驗收 3：計數器為 10 時踏上困難地形，小時 +1 且計數器停在 2。
func TestWalk_CostlyTileCallsStepTwice(t *testing.T) {
	tb := loadTables(t)
	walkable, _ := pickTiles(t, tb)

	// 0x0e 必須是可通行的，否則這個情境不成立。
	const costly = byte(0x0e)
	if tb.Passability(costly).Blocked() {
		t.Skip("tile 0x0e 在可通行性表中不可通行，略過此情境")
	}

	tiles := newFakeTiles(walkable)
	tiles.set(11, 10, costly)
	w := NewWorld(tiles, tb)
	p := NewParty(10, 10, East, 0)

	c := NewClock()
	c.steps = 10
	startHour := c.Hour()

	_, tile, advanced := w.Walk(p, c)

	if tile != costly {
		t.Fatalf("目標 tile 應為 0x%02x，得到 0x%02x", costly, tile)
	}
	if !advanced {
		t.Error("計數器 10 踏上困難地形應推進小時")
	}
	if c.Hour() != startHour+1 {
		t.Errorf("小時應為 %d，得到 %d", startHour+1, c.Hour())
	}
	if c.Steps() != 2 {
		t.Errorf("計數器應停在 2（第一次呼叫達門檻歸 1，第二次 +1），得到 %d", c.Steps())
	}
}

// 一般地形踏一步只算一步。與上面對照，確認困難地形的差別真的來自 tile。
func TestWalk_NormalTileCountsOneStep(t *testing.T) {
	tb := loadTables(t)
	walkable, _ := pickTiles(t, tb)

	tiles := newFakeTiles(walkable)
	w := NewWorld(tiles, tb)
	p := NewParty(10, 10, East, 0)

	c := NewClock()
	c.steps = 10
	startHour := c.Hour()

	if _, _, advanced := w.Walk(p, c); !advanced {
		t.Error("計數器 10 再走一步應推進小時")
	}
	if c.Hour() != startHour+1 {
		t.Errorf("小時應為 %d，得到 %d", startHour+1, c.Hour())
	}
	if c.Steps() != 1 {
		t.Errorf("一般地形應停在 1，得到 %d", c.Steps())
	}
}

// 子地圖：0xff 類才是路，踏到非 0xff 的一般地形就彈回上一層。
func TestWalk_SubmapExitOnNonWallTile(t *testing.T) {
	tb := loadTables(t)
	walkable, _ := pickTiles(t, tb)

	tiles := newFakeTiles(walkable)
	w := NewWorld(tiles, tb)

	p := NewParty(30, 30, South, 5)
	p.EnterSubmap(10, 10, East, 7)
	if p.Depth() != 1 {
		t.Fatalf("進入子地圖後深度應為 1，得到 %d", p.Depth())
	}

	c := NewClock()
	res, _, _ := w.Walk(p, c)

	if res != MoveExitedSubmap {
		t.Fatalf("在子地圖內踏到一般地形應退出，得到結果 %d", res)
	}
	if p.Depth() != 0 {
		t.Errorf("退出後深度應為 0，得到 %d", p.Depth())
	}
	if p.X() != 30 || p.Y() != 30 || p.Facing() != South || p.Dataset() != 5 {
		t.Errorf("應還原到進入前的 (30,30) 面向 South dataset 5，得到 (%d,%d) 面向 %d dataset %d",
			p.X(), p.Y(), p.Facing(), p.Dataset())
	}
}

func TestWalk_SubmapWalkableStreetTiles(t *testing.T) {
	tb := loadTables(t)
	walkable, wall := pickTiles(t, tb)

	for _, street := range []byte{tileSubmapWalkableA, tileSubmapWalkableB} {
		// 街道 tile 本身必須是 0xff 類，子地圖的例外規則才適用。
		if !tb.Passability(street).Blocked() {
			t.Skipf("tile 0x%02x 不是 0xff 類，子地圖例外規則不適用", street)
		}

		tiles := newFakeTiles(wall)
		tiles.set(11, 10, street)
		w := NewWorld(tiles, tb)

		p := NewParty(30, 30, South, 5)
		p.EnterSubmap(10, 10, East, 7)
		c := NewClock()

		res, _, _ := w.Walk(p, c)

		if res != MoveOK {
			t.Errorf("子地圖內街道 tile 0x%02x 應可通行，得到結果 %d", street, res)
		}
		if p.Depth() != 1 {
			t.Errorf("走在街道上不應退出子地圖，深度為 %d", p.Depth())
		}
		_ = walkable
	}
}

func TestWalk_SubmapBlockedByNonStreetWall(t *testing.T) {
	tb := loadTables(t)
	_, wall := pickTiles(t, tb)
	if wall == tileSubmapWalkableA || wall == tileSubmapWalkableB {
		t.Skip("挑到的牆 tile 剛好是街道例外，略過")
	}

	tiles := newFakeTiles(wall)
	w := NewWorld(tiles, tb)

	p := NewParty(30, 30, South, 5)
	p.EnterSubmap(10, 10, East, 7)
	c := NewClock()

	res, _, _ := w.Walk(p, c)

	if res != MoveBlocked {
		t.Errorf("子地圖內非街道的 0xff tile 應撞牆，得到結果 %d", res)
	}
	if p.Depth() != 1 {
		t.Errorf("撞牆不應改變深度，得到 %d", p.Depth())
	}
}

func TestParty_TurnDoesNotMove(t *testing.T) {
	p := NewParty(10, 10, North, 0)
	p.Turn(West)

	if p.Facing() != West {
		t.Errorf("面向應為 West，得到 %d", p.Facing())
	}
	if p.X() != 10 || p.Y() != 10 {
		t.Errorf("轉向不應移動，得到 (%d,%d)", p.X(), p.Y())
	}
}
