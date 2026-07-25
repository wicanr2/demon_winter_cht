package game

// 戰鬥網格上的定位與目標選擇。
const (
	// BattleGridWidth／BattleGridHeight 是戰場邊界，**已從原版確認是 9×9**。
	//
	// 證據是 FILES.DAT 0x0a8 的視線遮蔽表（見 gamedata/sight.go）：它的
	// 元素是網格索引、上限 81，讀取端的迴圈把游標從 10 起、每列走 7 格、
	// 每輪跳 2、到 73 為止 —— 展開來正好是 9 寬網格的內部 7×7，而表也
	// 剛好切出 49 組。索引展開、上限、掃描範圍三者只有在 9×9 時才自洽。
	//
	// 這兩個常數原本標「未經原版確認，取 9×9 是為了與呈現層一致」，
	// 現在不必再靠那個理由 —— 不過一致性本身仍然要維持，由
	// internal/ui/layout 的測試釘住。
	BattleGridWidth  = 9
	BattleGridHeight = 9

	// BattleGridMinX 是可站的最小 X。**不是 0。**
	//
	// 原版的行動順序把 `X == 0` 當成「空槽或已死」的哨兵值排除掉
	// （見 TurnOrder 的三個排除條件）。所以第 0 欄不是戰場的一部分 ——
	// 單位走進去就會從行動順序裡消失，看起來像原地蒸發。
	BattleGridMinX = 1
)

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
	if other == nil || other.IsPlayer == u.IsPlayer {
		return nil
	}
	return other
}

// CanStep 回報單位能不能往正前方走一步（只看地形與佔位，不看點數）。
func (b *Battle) CanStep(u *Unit) bool {
	x, y := FrontOf(u)
	if x < BattleGridMinX || x >= BattleGridWidth || y < 0 || y >= BattleGridHeight {
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
