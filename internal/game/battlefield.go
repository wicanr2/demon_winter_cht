package game

// 戰鬥網格上的定位與目標選擇。
//
// **戰場尺寸尚未從原版釘出來。** 手冊只說戰場是「該區域的放大地圖」，
// 邊緣有離場點；反組譯也還沒定位到邊界檢查。這裡用一個保守的界線常數
// 擋住越界，數值本身標為待確認 —— 它只影響「能不能再往前一步」，
// 不影響命中、傷害、行動點那些已驗證的規則。
const (
	// BattleGridWidth／BattleGridHeight 是暫定的戰場邊界。**未經原版確認。**
	//
	// 取 9×9 是為了與呈現層的視野格數一致 —— 規則允許走到畫不出來的地方，
	// 單位就會憑空消失。兩邊的一致性由 internal/ui/layout 的測試釘住。
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
