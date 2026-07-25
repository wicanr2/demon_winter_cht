package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"

// Facing 是隊伍面向，順時針編碼。與原版 struct+0xa4 的值相同。
type Facing int

const (
	North Facing = iota
	East
	South
	West
)

// Delta 回傳這個面向的座標位移，戰鬥網格也用同一組。
func (f Facing) Delta() (dx, dy int) { return f.delta() }

// CW 回傳順時針轉一格後的面向。
func (f Facing) CW() Facing { return (f + 1) % 4 }

// CCW 回傳逆時針轉一格後的面向。
func (f Facing) CCW() Facing { return (f + 3) % 4 }

// Reverse 回傳迴轉後的面向。
func (f Facing) Reverse() Facing { return (f + 2) % 4 }

// delta 回傳這個面向的座標位移。
// 對應原版 0x15da（ΔX）與 0x15d2（ΔY）兩張表。
func (f Facing) delta() (dx, dy int) {
	switch f {
	case North:
		return 0, -1
	case East:
		return 1, 0
	case South:
		return 0, 1
	case West:
		return -1, 0
	}
	return 0, 0
}

// LinearDelta 回傳這個面向在 64 寬地圖上的線性位移（ΔY×64 + ΔX）。
// 對應原版 0x2107 表的 −64, 1, 64, −1。導出而非寫死，避免兩張表各自漂移。
func (f Facing) LinearDelta() int {
	dx, dy := f.delta()
	return dy*MapWidth + dx
}

// 地圖尺寸。世界地圖與地城都是 64×64。
const (
	MapWidth  = 64
	MapHeight = 64
)

// 子地圖內視為可通行的例外 tile。子地圖自己的圖塊都屬 0xff 類，
// 其中這兩個是可走的街道；踏到非 0xff 的一般地形就代表走出子地圖。
const (
	tileSubmapWalkableA = 0x14
	tileSubmapWalkableB = 0x62
)

// 移動成本較高、要多算一步的 tile。證據見 docs/re/08 §6.4。
// 語意推測是山丘（手冊：穿過時間為一般地形兩倍），但未直接驗證。
var costlyTiles = map[byte]bool{0x0e: true, 0x2b: true}

// MoveResult 描述一次移動嘗試的結果。
type MoveResult int

const (
	// MoveOK 座標已更新。
	MoveOK MoveResult = iota
	// MoveBlocked 撞牆，座標不變。
	MoveBlocked
	// MoveExitedSubmap 踏出子地圖，深度已減一，座標與面向由歷史堆疊還原。
	MoveExitedSubmap
)

// TileSource 提供地圖圖塊。由 assets 層的 Map 實作。
type TileSource interface {
	TileAt(x, y int) (byte, error)
}

// submapFrame 是子地圖歷史堆疊的一層，記錄進入前的位置。
// 對應原版 struct+0x22..+0x26 的 6-byte-per-entry 堆疊。
type submapFrame struct {
	x, y    int
	facing  Facing
	dataset int
}

// Party 是隊伍在世界中的位置狀態。
//
// 只管「在哪裡、面向哪、在第幾層」，不含成員資料。
type Party struct {
	x, y    int
	facing  Facing
	dataset int
	stack   []submapFrame
}

// NewParty 建立位於指定座標的隊伍。
func NewParty(x, y int, facing Facing, dataset int) *Party {
	return &Party{x: x, y: y, facing: facing, dataset: dataset}
}

func (p *Party) X() int         { return p.x }
func (p *Party) Y() int         { return p.y }
func (p *Party) Facing() Facing { return p.facing }
func (p *Party) Dataset() int   { return p.dataset }

// Depth 回傳目前的子地圖深度。0 代表在大地圖上。
func (p *Party) Depth() int { return len(p.stack) }

// Turn 改變面向而不移動。
func (p *Party) Turn(f Facing) { p.facing = f }

// EnterSubmap 壓入目前位置並下降一層，落點為指定座標與面向。
func (p *Party) EnterSubmap(x, y int, facing Facing, dataset int) {
	p.stack = append(p.stack, submapFrame{x: p.x, y: p.y, facing: p.facing, dataset: p.dataset})
	p.x, p.y, p.facing, p.dataset = x, y, facing, dataset
}

// World 把地圖與可通行性表綁在一起，供移動判定使用。
type World struct {
	tiles  TileSource
	tables *gamedata.Tables
}

// NewWorld 建立移動判定所需的世界檢視。
func NewWorld(tiles TileSource, tables *gamedata.Tables) *World {
	return &World{tiles: tiles, tables: tables}
}

// tileAt 取得遮罩後的 tile 值。超出地圖範圍回傳 ok=false。
//
// 原版是否有 64×64 邊界檢查未確認（見 04-movement.md 未解表）。
// 這裡明確擋住，避免讀到相鄰列的資料 —— 寧可保守也不要靜默讀錯格子。
func (w *World) tileAt(x, y int) (byte, bool) {
	if x < 0 || x >= MapWidth || y < 0 || y >= MapHeight {
		return 0, false
	}
	t, err := w.tiles.TileAt(x, y)
	if err != nil {
		return 0, false
	}
	return t & 0x7f, true
}

// Walk 讓隊伍沿目前面向前進一格，並回報結果與這一步是否推進了小時。
//
// 呼叫順序對應 docs/spec/04-movement.md「一步移動的完整順序」的第 6–9 步。
// 事件觸發（第 10 步）不在這裡，由呼叫端依回傳的 tile 值處理。
func (w *World) Walk(p *Party, c *Clock) (result MoveResult, tile byte, hourAdvanced bool) {
	dx, dy := p.facing.delta()
	tx, ty := p.x+dx, p.y+dy

	t, ok := w.tileAt(tx, ty)
	if !ok {
		return MoveBlocked, 0, false
	}

	blocked := w.tables.Passability(t).Blocked()

	if p.Depth() > 0 {
		// 子地圖內的判定是反過來的：0xff 才是路，非 0xff 代表踏出去了。
		if !blocked {
			p.exitSubmap()
			return MoveExitedSubmap, t, false
		}
		if t != tileSubmapWalkableA && t != tileSubmapWalkableB {
			return MoveBlocked, t, false
		}
	} else if blocked {
		return MoveBlocked, t, false
	}

	p.x, p.y = tx, ty

	// 困難地形先多記一步。這是「呼叫兩次」而不是「一次加 2」——
	// 兩次各自檢查 11 的門檻，行為與 += 2 不同。
	if costlyTiles[t] {
		hourAdvanced = c.Step()
	}
	if c.Step() {
		hourAdvanced = true
	}

	return MoveOK, t, hourAdvanced
}

// exitSubmap 彈出一層，還原進入前的座標、面向與 dataset。
func (p *Party) exitSubmap() {
	if len(p.stack) == 0 {
		return
	}
	f := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	p.x, p.y, p.facing, p.dataset = f.x, f.y, f.facing, f.dataset
}
