package game

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

// 戰場地形：進戰鬥時從世界地圖切一塊下來放大（`docs/re/36`）。
//
// 原版的作法（一般遭遇那條路徑）：
//
//  1. 把 64×64 的地圖緩衝清成 0。
//  2. 在第 5 與第 21 列／欄畫一圈 17×17 的牆（tile 3）。
//  3. 取大地圖上隊伍周圍的 **3×3 個世界 tile**，每一個攤成 **5×5** 的
//     戰場格，九個區塊拼成 15×15 貼進 (6, 6)。
//
// 九個區塊的貼圖起點來自 `ds:0x0a0e` 那張表 —— 值是
// `0186 018b 0190 02c6 02cb 02d0 0406 040b 0410`，換算成 64 寬緩衝的座標
// 正是 (6,6)(11,6)(16,6)(6,11)… 一格不差。
//
// 所以**戰場不是「大地圖的 9×9 局部」，是「3×3 個世界 tile 各放大五倍」**。
// 與手冊講的「戰場是該區域的放大地圖」一致 —— 是真的放大，不是取一小塊。
//
// # 一張地圖同時當地形與佔位表
//
// 能不能站、能不能走，原版只問一件事：**那一格的值等不等於「空地值」**
// （`[0x51d8]`，開戰時從隊伍腳下取樣）。擺好的單位會把圖塊蓋進地圖，
// 那一格從此不再等於空地值，所以佔位不必另外記。
//
// 本專案的地形緩衝唯讀，佔位由 `Battle.UnitAt` 負責，判定拆成兩半。
//
// # 尚未實作
//
// tile 值為 4 的區塊會在區塊之間開／封一道口子（`ds:0x0a20`／`0x0a32`
// 兩張鄰接位移表），條件與語意未解。這裡照抄區塊的 tile，不猜那個分支。

// BattleTerrainSize 是地形緩衝的邊長，與 BattleGridWidth 相同（含牆）。
const BattleTerrainSize = BattleGridWidth

// BattleTerrain 是一場戰鬥的地形，索引 = y*BattleTerrainSize + x。
//
// ⚠ **`0` 是合法的世界 tile 值**（大地圖上到處都是），所以不能拿它當
// 「這一格沒東西」的哨兵。哪些格子有內容由**座標**決定：牆框
// `BattleWallLow`–`BattleWallHigh` 以內都有，以外沒有。
// 世界地圖界外的區塊填成牆，不是留 0。
type BattleTerrain [BattleTerrainSize * BattleTerrainSize]byte

// InArena 回報這一格在不在戰場（含牆框）之內 —— 也就是**畫不畫得出來**。
// 別用「地形值是不是 0」來判斷，0 是合法的地形。
func InArena(x, y int) bool {
	return x >= BattleWallLow && x <= BattleWallHigh &&
		y >= BattleWallLow && y <= BattleWallHigh
}

// TileAt 回傳 (x, y) 的地形值。緩衝外回 0。
func (t *BattleTerrain) TileAt(x, y int) byte {
	if x < 0 || x >= BattleTerrainSize || y < 0 || y >= BattleTerrainSize {
		return 0
	}
	return t[y*BattleTerrainSize+x]
}

// Ground 回傳「空地值」—— 就是佈陣中心那一格。
//
// 原版 `*(byte*)0x51d8 = 地圖[0x34d]`，`0x34d` = 13×64 + 13 = 隊伍腳下。
// 拿隊伍站的地方當「可站」的定義，不需要另一張可通行性表。
func (t *BattleTerrain) Ground() byte {
	if t == nil {
		return 0
	}
	return t.TileAt(BattleCentreX, BattleCentreY)
}

// Walkable 回報這一格的地形能不能站人（不看有沒有人站著）。
// t 為 nil 時只看邊界 —— 單元測試常這樣用。
//
// 邊界情況：隊伍站在世界 tile 3 上時，空地值與牆同值，牆就變成可站。
// 原版有同樣的問題（判定就只有 `== [0x51d8]` 一條），這裡照舊不特別處理。
func (t *BattleTerrain) Walkable(x, y int) bool {
	if !InField(x, y) {
		return false
	}
	if t == nil {
		return true
	}
	return t.TileAt(x, y) == t.Ground()
}

// NewBattleTerrain 以世界座標 (cx, cy) 為中心切出戰場地形。
//
// 取 (cx−1 … cx+1, cy−1 … cy+1) 九個世界 tile，各攤成 5×5。
// 世界地圖界外的區塊填成牆 —— 站不上去，而且畫得出來。
func NewBattleTerrain(m *world.Map, cx, cy int) (*BattleTerrain, error) {
	if m == nil {
		return nil, fmt.Errorf("game: 戰場地形需要世界地圖")
	}

	var t BattleTerrain

	// 牆框：第 5 與第 21 列／欄，長 17 格。
	for i := BattleWallLow; i <= BattleWallHigh; i++ {
		t[BattleWallLow*BattleTerrainSize+i] = BattleWallTile
		t[BattleWallHigh*BattleTerrainSize+i] = BattleWallTile
		t[i*BattleTerrainSize+BattleWallLow] = BattleWallTile
		t[i*BattleTerrainSize+BattleWallHigh] = BattleWallTile
	}

	half := BattleBlocks / 2
	for by := 0; by < BattleBlocks; by++ {
		for bx := 0; bx < BattleBlocks; bx++ {
			v := byte(BattleWallTile) // 界外當牆：站不上去，而且畫得出來
			wx, wy := cx-half+bx, cy-half+by
			if wx >= 0 && wx < world.MapWidth && wy >= 0 && wy < world.MapHeight {
				if tv, err := m.TileAt(wx, wy); err == nil {
					v = tv
				}
			}
			ox := BattleFieldMin + bx*BattleBlockSize
			oy := BattleFieldMin + by*BattleBlockSize
			for dy := 0; dy < BattleBlockSize; dy++ {
				for dx := 0; dx < BattleBlockSize; dx++ {
					t[(oy+dy)*BattleTerrainSize+ox+dx] = v
				}
			}
		}
	}
	return &t, nil
}
