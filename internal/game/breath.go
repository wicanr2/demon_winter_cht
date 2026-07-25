package game

// 噴吐攻擊的波及範圍。
//
// 原版是 `FUN_138d_17b8`（Ghidra 138d:17b8 = DEMON.INT 檔位移 0x8c88，
// 錨點字串 `breathes!`）。怪物 AI 在種族／元素類型落在 8–12 時有 30%
// 機率走這一支（見 ai.go）。
//
// # 範圍的形狀
//
// 起點不是施放者自己，是**它面前那一格**（0x8ca2–0x8cc0）：
// 拿朝向去查方向增量表，加到施放者的 X／Y 上。
//
// 從那一格往正前方是一個**深度 4 的錐形**。以朝北（朝向 0）為例
// （0x8d95–0x8dab）：
//
//	目標.Y <= 起點.Y            ; 不能在起點後面（等於是可以的）
//	目標.Y >  起點.Y − 4        ; 深度 4，**含起點那一格**
//	|Δx|  <= |Δy|               ; 錐形：橫向偏移不超過縱向距離
//
// 朝東（朝向 1）那一段（0x8db4–0x8dcd）是同一個形狀轉 90 度，
// 四個方向各一段、寫死展開，不是共用一段程式。
//
// # 同族免疫
//
// 0x8cf9–0x8d22：目標的種族／元素類型是 4、7 或 10 時，若**施放者**的
// 類型是 10，這個目標整個跳過。看起來是「同系元素互不傷害」，
// 但只有施放者類型 10 這一種情況，另外兩種沒有對應分支 —— 照實作，不補對稱。
//
// # 還沒實作
//
// 傷害本身。原函式後半（1244 bytes 裡的大部分）在算傷害與抗性，
// 還沒逐指令讀完。這裡只解出「打到誰」，`BreathTargets` 回傳被波及的單位，
// 傷害由呼叫端決定 —— 寧可缺一半也不要猜一個公式進去。

// BreathDepth 是噴吐涵蓋幾格深，**含起點那一格**。
// 原版的兩道邊界是 `目標 <= 起點` 與 `目標 > 起點 − 4`（0x8d95／0x8d9a），
// 所以沿噴吐方向的距離落在 0–3。
const BreathDepth = 4

// breathImmuneRaces 是「施放者類型 10 時互不傷害」的那組類型。
var breathImmuneRaces = map[int]bool{4: true, 7: true, 10: true}

// breathImmuneCaster 是觸發上面那條免疫的施放者類型。
const breathImmuneCaster = 10

// BreathTargets 回傳這次噴吐會波及哪些單位。
//
// 起點是施放者面前那一格；範圍是從該格往正前方深度 BreathDepth 的錐形，
// 橫向偏移不超過縱向距離。不分敵我 —— 原版的迴圈掃全部 15 個槽位，
// 只跳過不在場上的（X == 0）與同族免疫的。
func (b *Battle) BreathTargets(caster *Unit) []*Unit {
	if caster == nil {
		return nil
	}
	dx, dy := Facing(caster.Facing).Delta()
	originX, originY := caster.X+dx, caster.Y+dy

	var out []*Unit
	for slot := 0; slot < CombatSlots; slot++ {
		u := b.Unit(slot)
		if u == nil || !u.Alive() {
			continue // 原版看的是 X == 0 的哨兵，等價於「不在場上」
		}
		if caster.RaceOrElement == breathImmuneCaster &&
			breathImmuneRaces[u.RaceOrElement] {
			continue
		}
		if inBreathCone(originX, originY, dx, dy, u.X, u.Y) {
			out = append(out, u)
		}
	}
	return out
}

// inBreathCone 判斷 (tx, ty) 是否落在從 (ox, oy) 朝 (dx, dy) 的錐形內。
//
// along 是沿噴吐方向的距離，across 是垂直方向的偏移。
// 命中條件：`0 <= along < BreathDepth` 且 `|across| <= along`。
//
// **起點那一格算命中**（along = 0，此時 across 只能是 0）——
// 原版排除的是「在起點後面」（0x8d95 的 `jg`、0x8db4 的 `jl`），
// 等於起點的情況不會被排掉。
func inBreathCone(ox, oy, dx, dy, tx, ty int) bool {
	var along, across int
	switch {
	case dx != 0: // 東西向
		along = (tx - ox) * dx
		across = ty - oy
	case dy != 0: // 南北向
		along = (ty - oy) * dy
		across = tx - ox
	default:
		return false
	}
	if along < 0 || along >= BreathDepth {
		return false
	}
	if across < 0 {
		across = -across
	}
	return across <= along
}
