package game

// 戰鬥網格上的定位與目標選擇。
//
// # 戰場的形狀（`docs/re/36`）
//
// 原版開戰時做的第一件事是把一塊 64×64 的緩衝清成 0，然後：
//
//  1. 在第 5 與第 21 列／欄畫一圈 17×17 的牆（tile 3）
//  2. 把大地圖上隊伍周圍的 **3×3 個世界 tile**，每一個攤成 **5×5** 的
//     戰場格，拼成 15×15 貼進 (6, 6)
//
// 所以**可站的範圍是 (6,6)–(20,20)，15×15，中心正是 (13,13)** ——
// 佈陣把隊伍放在 (13,13) 的道理就在這裡（`docs/re/35`）。
//
// 本專案沿用原版的絕對座標，只把緩衝縮到 22×22（原版 64×64 的其餘部分
// 是空的）。座標一致，之後任何反組譯發現都能直接對號入座。
const (
	// BattleGridWidth／BattleGridHeight 是緩衝尺寸，含牆。
	BattleGridWidth  = 22
	BattleGridHeight = 22

	// BattleWallLow／BattleWallHigh 是牆框所在的列／欄。
	BattleWallLow  = 5
	BattleWallHigh = 21

	// BattleFieldMin／BattleFieldMax 是可站範圍（含端點），
	// BattleFieldSize 是它的邊長。
	BattleFieldMin  = BattleWallLow + 1
	BattleFieldMax  = BattleWallHigh - 1
	BattleFieldSize = BattleFieldMax - BattleFieldMin + 1

	// BattleCentreX／BattleCentreY 是佈陣中心，原版寫死 (13, 13)。
	BattleCentreX = (BattleFieldMin + BattleFieldMax) / 2
	BattleCentreY = BattleCentreX

	// BattleBlockSize 是一個世界 tile 攤開成幾格。
	BattleBlockSize = 5
	// BattleBlocks 是一邊有幾個區塊（3×3 個世界 tile）。
	BattleBlocks = BattleFieldSize / BattleBlockSize

	// BattleWallTile 是牆框的 tile 值。
	BattleWallTile = 3

	// BattleGridMinX 是可站的最小 X。**不是 0。**
	//
	// 原版的行動順序把 `X == 0` 當成「空槽或已死」的哨兵值排除掉
	// （見 TurnOrder 的三個排除條件）。可站範圍從 6 起，本來就不會撞到 0，
	// 但這個哨兵語意仍然成立 —— 死亡結算會把 X 清成 0。
	BattleGridMinX = 1
)

// InField 回報座標在不在可站範圍內（只看邊界，不看地形與佔位）。
func InField(x, y int) bool {
	return x >= BattleFieldMin && x <= BattleFieldMax &&
		y >= BattleFieldMin && y <= BattleFieldMax
}

// UnitAt 回傳站在 (x, y) 的存活單位，沒有則回傳 nil。
func (b *Battle) UnitAt(x, y int) *Unit {
	for _, u := range b.units {
		if u != nil && u.Alive() && u.X == x && u.Y == y {
			return u
		}
	}
	return nil
}

// FrontOf 回傳單位正前方那一格的座標。
func FrontOf(u *Unit) (x, y int) {
	dx, dy := Facing(u.Facing).Delta()
	return u.X + dx, u.Y + dy
}

// TargetInFront 回傳單位正前方的敵方單位。
//
// 手冊：「按下 A 會用目前裝備的武器攻擊正前方的怪物」——
// 攻擊沒有選目標的步驟，面向決定打誰。前方沒有敵人就回 nil，
// 呼叫端應拒絕這次攻擊而不是扣點數。
func (b *Battle) TargetInFront(u *Unit) *Unit {
	x, y := FrontOf(u)
	other := b.UnitAt(x, y)
	if other == nil || other.OnPlayerSide() == u.OnPlayerSide() {
		return nil
	}
	return other
}

// CanStep 回報單位能不能往正前方走一步（只看地形與佔位，不看點數）。
//
// 原版的判定只有一條（`FUN_1990_07ed`）：
//
//	if (地圖[目標] == *(char*)0x51d8) 就走得過去
//
// `[0x51d8]` 是「空地值」，開戰時從隊伍腳下那一格取樣。這一條同時擋掉
// 「地形不對」與「有人站著」—— 因為擺好的單位會把自己的圖塊蓋進地圖，
// 那一格從此不再等於空地值。**一張地圖同時當地形與佔位表**（`docs/re/35` §3）。
//
// 本專案的地形緩衝是唯讀的，所以兩件事分開問：地形問 Terrain、佔位問 UnitAt。
func (b *Battle) CanStep(u *Unit) bool {
	x, y := FrontOf(u)
	if !InField(x, y) {
		return false
	}
	if b.Terrain != nil && !b.Terrain.Walkable(x, y) {
		return false
	}
	return b.UnitAt(x, y) == nil
}

// Step 讓單位往正前方走一步並扣點。走不動時回傳 false 且不扣點。
func (b *Battle) Step(u *Unit) bool {
	if !b.CanStep(u) {
		return false
	}
	if _, ok := b.Spend(ActionForward); !ok {
		return false
	}
	dx, dy := Facing(u.Facing).Delta()
	u.X += dx
	u.Y += dy
	return true
}

// TurnTo 讓單位轉向並扣點。
//
// 三個轉向鍵都是 1 點：順時針、逆時針、迴轉。
func (b *Battle) TurnTo(u *Unit, a Action) bool {
	var next Facing
	switch a {
	case ActionTurnCW:
		next = Facing(u.Facing).CW()
	case ActionTurnCCW:
		next = Facing(u.Facing).CCW()
	case ActionAboutFace:
		next = Facing(u.Facing).Reverse()
	default:
		return false
	}
	if _, ok := b.Spend(a); !ok {
		return false
	}
	u.Facing = int(next)
	return true
}
