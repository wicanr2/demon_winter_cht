package main

import (
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 自動戰鬥驅動（`-autofight`）。
//
// **這是驗收工具，不是作弊。** A4 的規則是「不用 debug 捷徑走完主線」
//（`docs/re/64` §3）—— 捷徑指的是跳過內容（`-battle-win`、`-give-item`
// 那一類）。這支不跳過任何東西：它**下的是玩家下得出來的同一組指令**，
// 走 `faceToward` 與 `runPlayerAction` 這兩個按鍵處理器自己在用的函式，
// 一樣扣行動點、一樣會被拒絕、一樣會打不中。
//
// 做它的理由記在 `docs/playtest/01` §4：按鍵重播腳本走不過戰鬥。
// 送 80 次 Enter 之後畫面還在第 1 回合，戰鬥紀錄整片「前方過不去」——
// 因為 Enter 是前進而正前方站著怪，而且怪每回合會移動，
// 盲送固定鍵序永遠對不準。**沒有這支就走不完任何一段主線。**
//
// 策略刻意笨：找最近的活敵人 → 面向它 → 貼著就打、沒貼著就走一步。
// 笨策略打不贏的仗，人類玩家也該覺得難 —— 它是用來「把仗打完」的，
// 不是用來證明引擎平衡的。

// autoFighter 是自動戰鬥的狀態。
type autoFighter struct {
	// tick 是幀計數。驅動器每 autoEveryFrames 幀才動一次 ——
	// 全速跑的話一場仗在幾十毫秒內結束，軌跡擠成一團看不出順序，
	// 截圖也永遠拍不到中間過程。
	tick int
	// stalls 是連續幾次「這一輪什麼也沒做成」。
	// 到達上限就結束回合 —— 不然行動點卡住時會無限迴圈，
	// 而畫面上看起來只是「停住」，分不出是卡死還是在思考。
	stalls int
}

// autoStallLimit 是放棄這一回合的門檻。
//
// 為什麼需要它：行動點不足以轉向、或目標被牆隔開時，
// 每一幀都會做出「合法但無效」的嘗試。沒有上限的話，
// 軌跡會停在同一個狀態，而那正是這支工具要消除的假象。
const autoStallLimit = 8

// autoEveryFrames 是兩個指令之間隔幾幀（60 fps，約 0.1 秒）。
const autoEveryFrames = 6

// ready 回報這一幀該不該動作。
func (f *autoFighter) ready() bool {
	f.tick++
	return f.tick%autoEveryFrames == 0
}

// updateAutoFight 代替玩家下一個指令。回傳 true 表示這一幀已經處理掉了。
//
// 只在玩家單位的回合介入。怪物回合、選單、結算畫面都不碰 ——
// 那些由 `updateBattle` 自己的路徑處理，攔截它們會讓驅動器
// 繞過遊戲真正的流程，那才會變成捷徑。
func (a *app) updateAutoFight() bool {
	if a.auto == nil || a.battle == nil {
		return false
	}
	// 選單開著時不介入：那表示上一個指令打開了子畫面，
	// 讓它照原本的路徑走完（目前的策略不會開選單，但別假設）。
	if a.spells != nil || a.useMenu != nil || a.spInput != nil || a.aoe != nil {
		return false
	}
	cur := a.battle.Current()
	if cur == nil || !cur.IsPlayer || !cur.Alive() {
		return false
	}
	if !a.auto.ready() {
		return true
	}

	target := a.nearestEnemy(cur)
	if target == nil {
		// 沒有活著的敵人 —— 交給 updateBattle 去結算。
		return false
	}
	// 生命降到一半時先保命。閃避會把這回合尚未花掉的行動點換成命中率
	// 修正；若仍照「貼近就連砍」的簡單策略，1 HP 的後排角色也會主動
	// 留在敵人面前攻擊，下一個怪物回合幾乎必死。一半生命不是隱藏資訊：
	// 玩家在畫面上看得到目前／最大生命，也能按同一個 D 指令。
	if cur.HP*2 <= cur.MaxHP && a.aliveEnemyCount(cur) > 1 {
		if dir, ok := a.safestRetreat(cur); ok {
			needed := game.ActionTurnCW
			if game.Facing(cur.Facing) == dir {
				needed = game.ActionForward
			}
			if a.battle.CanAct(needed) {
				a.faceToward(cur, dir)
				a.auto.stalls = 0
				return true
			}
		}
		a.runPlayerAction(cur, game.ActionDodge)
		a.auto.stalls = 0
		return true
	}

	// **沿用怪物 AI 的 `stepToward`**（`battleui.go`）。
	// 自己再寫一份「朝目標走」只會多一種行為 ——
	// 兩邊靠近目標的方式不一致的話，用這支驅動器驗出來的戰鬥
	// 就不代表玩家或怪物真正會遇到的情況。
	dir, ok := stepToward(cur, target)
	if !ok {
		return false
	}
	adjacent := absInt(target.X-cur.X)+absInt(target.Y-cur.Y) == 1
	if !adjacent {
		planned, found := a.battle.FirstStepToward(cur, target, dir)
		if !found {
			a.runPlayerAction(cur, game.ActionEndTurn)
			a.auto.stalls = 0
			return true
		}
		dir = planned
	}
	before := *cur

	if adjacent {
		if game.Facing(cur.Facing) != dir {
			if !a.battle.CanAct(game.ActionTurnCW) {
				a.runPlayerAction(cur, game.ActionEndTurn)
				return true
			}
			a.faceToward(cur, dir)
		} else {
			if !a.battle.CanAct(game.ActionAttack) {
				a.runPlayerAction(cur, game.ActionEndTurn)
				return true
			}
			a.runPlayerAction(cur, game.ActionAttack)
		}
	} else {
		// faceToward 在「已經面向該方位」時本身就會前進一步，
		// 所以移動與轉向共用同一個呼叫。
		needed := game.ActionTurnCW
		if game.Facing(cur.Facing) == dir {
			needed = game.ActionForward
		}
		if !a.battle.CanAct(needed) {
			a.runPlayerAction(cur, game.ActionEndTurn)
			return true
		}
		a.faceToward(cur, dir)
	}

	// 什麼都沒變（行動點不夠、被牆擋住）就記一次空轉。
	if before.X == cur.X && before.Y == cur.Y && before.Facing == cur.Facing {
		a.stalled()
	} else {
		a.auto.stalls = 0
	}
	return true
}

func (a *app) aliveEnemyCount(u *game.Unit) int {
	n := 0
	for _, o := range a.battle.Units() {
		if o.Alive() && o.OnPlayerSide() != u.OnPlayerSide() {
			n++
		}
	}
	return n
}

// safestRetreat 找一個可走、而且能拉開最近敵人距離的相鄰格。
// 只看戰場上玩家本來就看得到的位置與佔位；找不到更安全的格就留在原地閃避。
func (a *app) safestRetreat(u *game.Unit) (game.Facing, bool) {
	minEnemyDistance := func(x, y int) int {
		best := 1 << 30
		for _, o := range a.battle.Units() {
			if !o.Alive() || o.OnPlayerSide() == u.OnPlayerSide() {
				continue
			}
			d := absInt(o.X-x) + absInt(o.Y-y)
			if d < best {
				best = d
			}
		}
		return best
	}

	bestDist := minEnemyDistance(u.X, u.Y)
	var best game.Facing
	found := false
	for _, dir := range []game.Facing{game.North, game.East, game.South, game.West} {
		probe := *u
		probe.Facing = int(dir)
		if !a.battle.CanStep(&probe) {
			continue
		}
		dx, dy := dir.Delta()
		d := minEnemyDistance(u.X+dx, u.Y+dy)
		if d > bestDist {
			best, bestDist, found = dir, d, true
		}
	}
	return best, found
}

// stalled 記一次空轉，到達上限就結束這名單位的回合。
func (a *app) stalled() {
	a.auto.stalls++
	if a.auto.stalls < autoStallLimit {
		return
	}
	a.auto.stalls = 0
	if cur := a.battle.Current(); cur != nil {
		a.runPlayerAction(cur, game.ActionEndTurn)
	}
}

// nearestEnemy 找曼哈頓距離最近的活敵人。
func (a *app) nearestEnemy(u *game.Unit) *game.Unit {
	var best *game.Unit
	bestDist := 1 << 30
	for _, o := range a.battle.Units() {
		if o == u || !o.Alive() || o.OnPlayerSide() == u.OnPlayerSide() {
			continue
		}
		d := absInt(o.X-u.X) + absInt(o.Y-u.Y)
		if d < bestDist {
			best, bestDist = o, d
		}
	}
	return best
}

// autoAdvance 回報「自動戰鬥要不要替玩家按下空白鍵」。
//
// 空白鍵在戰鬥裡只有一個用途：看完怪物的一步、看完結算。
// 玩家按它不會改變任何規則，所以自動按也不會 —— 這仍然不是捷徑。
func (a *app) autoAdvance() bool {
	return a.auto != nil && a.auto.ready()
}

// newAutoFighter 在旗標開著時建驅動器，否則回 nil（＝不介入）。
func newAutoFighter(on bool) *autoFighter {
	if !on {
		return nil
	}
	return &autoFighter{}
}
