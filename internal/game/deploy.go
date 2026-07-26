package game

import "github.com/wicanr2/demon_winter_cht/internal/rng"

// 開戰時的擺位（出處 `docs/re/35`）。
//
// 原版不是「兩軍列隊對峙」，是**一團混戰**：隊伍照 3×3 陣型站在中央，
// 怪物散落在中心 ±2 格的範圍內，開場就貼在你臉上。
//
// # 座標系的落差
//
// 原版的戰場座標活在一塊 64 寬的緩衝裡，隊伍中心在 (13, 13)，
// 畫面是一個跟著隊伍捲動的 9×9 視窗（`[0x50f0] += sign(單位X − [0x50f0])`）。
// 本專案的戰場就是那個 9×9，沒有捲動 —— 所以這裡把中心定在 (4, 4)。
//
// **相對幾何完全一致**（陣型 3×3、怪物 ±2），差的只是原點，而原點是任意的：
// 13 只是隊伍當時站在緩衝裡的位置。絕對座標對不上的部分見 `docs/re/35` §4。

const (
	// BattleCentreX／BattleCentreY 是佈陣中心，對應原版的 (13, 13)。
	BattleCentreX = BattleGridWidth / 2
	BattleCentreY = BattleGridHeight / 2

	// 怪物的散佈：`中心 + rnd(5) − 3`，值域 −2..+2。
	monsterScatterDie    = 5
	monsterScatterOffset = 3

	// monsterScatterLimit 是重擲上限。**原版沒有上限** ——
	// 中心 ±2 全是牆的話 `do { } while` 會轉不出來。這是保險，不是規則。
	monsterScatterLimit = 200
)

// DeployPartyAt 依陣型算出成員 member 該站的格子。
//
// 不在陣型裡的成員（`Formation` 九格都沒有他）回 ok=false ——
// 原版佈陣迴圈是掃九格、看格子裡是誰，沒被放進去的人根本不會上場。
func DeployPartyAt(f Formation, member int) (x, y int, ok bool) {
	cell := f.CellOf(member)
	if cell < 0 {
		return 0, 0, false
	}
	dx, dy := FormationOffset(cell)
	return BattleCentreX + dx, BattleCentreY + dy, true
}

// OpenGround 回報一格能不能站人。
//
// 原版的判定只有一條：**那一格的地形值等於隊伍腳下那一格的值**
// （`[0x51d8]`，開戰時從 `map[13][13]` 取樣）。已經有人站的格子會被排除，
// 不是因為另外檢查了佔位 —— 而是**每擺一個單位就把牠的圖塊蓋進地形緩衝**，
// 那一格的值從此不再等於空地值。地圖本身就是佔位表。
//
// 本專案的地形緩衝是唯讀的（`BattleTerrain` 直接來自世界地圖），
// 所以佔位要另外問，由 occupied 回呼提供。
func OpenGround(t *BattleTerrain, occupied func(x, y int) bool, ground byte, x, y int) bool {
	if x < BattleGridMinX || x >= BattleGridWidth || y < 0 || y >= BattleGridHeight {
		return false
	}
	if t != nil && t.TileAt(x, y) != ground {
		return false
	}
	return occupied == nil || !occupied(x, y)
}

// GroundTile 回傳「空地」的地形值 —— 就是佈陣中心那一格。
//
// 原版 `*(byte*)0x51d8 = map[0x34d]`，`0x34d` = 13×64 + 13 = 隊伍腳下。
// 拿隊伍站的地方當「可站」的定義，不需要另一張可通行性表。
func GroundTile(t *BattleTerrain) byte {
	if t == nil {
		return 0
	}
	return t.TileAt(BattleCentreX, BattleCentreY)
}

// ScatterMonster 擲一個怪物的落點：中心 ±2，直到擲到空地為止。
//
// ok=false 代表擲了 monsterScatterLimit 次都沒有空位 —— 原版會卡死在
// 那個 `do { } while`，這裡回報失敗讓呼叫端跳過這隻怪。
func ScatterMonster(r *rng.RNG, t *BattleTerrain, occupied func(x, y int) bool) (x, y int, ok bool) {
	ground := GroundTile(t)
	for i := 0; i < monsterScatterLimit; i++ {
		x = BattleCentreX + r.Roll(monsterScatterDie) - monsterScatterOffset
		y = BattleCentreY + r.Roll(monsterScatterDie) - monsterScatterOffset
		if OpenGround(t, occupied, ground, x, y) {
			return x, y, true
		}
	}
	return 0, 0, false
}
