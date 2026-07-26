package game

import "github.com/wicanr2/demon_winter_cht/internal/rng"

// 開戰時的擺位（出處 `docs/re/35`）。
//
// 原版不是「兩軍列隊對峙」，是**一團混戰**：隊伍照 3×3 陣型站在中央，
// 怪物散落在中心 ±2 格的範圍內，開場就貼在你臉上。
//
// 座標與原版一致：戰場 15×15 位於 (6,6)–(20,20)，中心 (13,13)
// （見 `battlefield.go` 的常數與 `docs/re/36`）。

const (
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
func OpenGround(t *BattleTerrain, occupied func(x, y int) bool, x, y int) bool {
	if !t.Walkable(x, y) {
		return false
	}
	return occupied == nil || !occupied(x, y)
}

// ScatterMonster 擲一個怪物的落點：中心 ±2，直到擲到空地為止。
//
// ok=false 代表擲了 monsterScatterLimit 次都沒有空位 —— 原版會卡死在
// 那個 `do { } while`，這裡回報失敗讓呼叫端跳過這隻怪。
func ScatterMonster(r *rng.RNG, t *BattleTerrain, occupied func(x, y int) bool) (x, y int, ok bool) {
	for i := 0; i < monsterScatterLimit; i++ {
		x = BattleCentreX + r.Roll(monsterScatterDie) - monsterScatterOffset
		y = BattleCentreY + r.Roll(monsterScatterDie) - monsterScatterOffset
		if OpenGround(t, occupied, x, y) {
			return x, y, true
		}
	}
	return 0, 0, false
}
