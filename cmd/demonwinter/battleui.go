package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 玩家戰鬥指令。
//
// **熱鍵沿用原版**（A 攻擊、T 驅散不死、D 閃避、P 祈禱、L 汲取、? 檢視、ESC 結束回合），
// 只把顯示文字換成中文。玩過原版的人不必重學鍵位，這是中文化該有的樣子。
var playerCommands = []struct {
	key    ebiten.Key
	label  string
	action game.Action
}{
	{ebiten.KeyA, "A 攻擊", game.ActionAttack},
	{ebiten.KeyT, "T 驅散不死", game.ActionTurnUndead},
	{ebiten.KeyP, "P 祈禱", game.ActionPray},
	{ebiten.KeyL, "L 汲取法力", game.ActionLeech},
	{ebiten.KeyD, "D 閃避", game.ActionDodge},
	{ebiten.KeyEscape, "Esc 結束回合", game.ActionEndTurn},
}

// updateBattle 推進戰鬥。
//
// 玩家單位輪到時等玩家下指令；怪物與召喚物由簡單 AI 代打，按空白鍵逐步執行，
// 方便肉眼核對每一步。
func (a *app) updateBattle() error {
	if out := a.battle.Outcome(); out != game.Ongoing {
		if !inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			return nil
		}
		if out == game.Victory {
			a.logf("怪物全滅")
		} else {
			a.logf("隊伍全滅")
		}
		a.battle = nil
		return nil
	}

	cur := a.battle.Current()
	if cur == nil {
		a.battle.BeginRound()
		a.logf("── 第 %d 回合 ──", a.battle.Round())
		return nil
	}

	if cur.IsPlayer {
		return a.updatePlayerTurn(cur)
	}
	if !inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return nil
	}
	a.monsterTurn(cur)
	return nil
}

// updatePlayerTurn 處理玩家單位的一次按鍵。
func (a *app) updatePlayerTurn(u *game.Unit) error {
	// 轉向與前進：方向鍵轉到該面向（每次轉一格），Enter 前進。
	for _, kf := range keyFacing {
		if !inpututil.IsKeyJustPressed(kf.key) {
			continue
		}
		a.faceToward(u, kf.f)
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if !a.battle.Step(u) {
			a.logf("%s 前方過不去", u.Name)
		}
		return nil
	}

	for _, c := range playerCommands {
		if !inpututil.IsKeyJustPressed(c.key) {
			continue
		}
		a.runPlayerAction(u, c.action)
		return nil
	}
	return nil
}

// faceToward 把單位轉向指定方位，一次按鍵只轉一格。
//
// 原版的轉向鍵是「順時針／逆時針／迴轉」各 1 點，不是「直接面向某方」。
// 這裡讓方向鍵挑最短路徑轉一格，按鍵直覺但成本與原版一致。
func (a *app) faceToward(u *game.Unit, want game.Facing) {
	cur := game.Facing(u.Facing)
	if cur == want {
		if !a.battle.Step(u) {
			a.logf("%s 前方過不去", u.Name)
		}
		return
	}

	act := game.ActionTurnCW
	switch {
	case cur.Reverse() == want:
		act = game.ActionAboutFace
	case cur.CCW() == want:
		act = game.ActionTurnCCW
	}
	if !a.battle.TurnTo(u, act) {
		a.logf("%s 行動點不足", u.Name)
	}
}

// runPlayerAction 執行一個玩家指令。
//
// **先檢查前置條件再扣點。** 沒有目標的攻擊、非不死系的驅散，
// 都應該是「什麼都沒發生」而不是「白花三點」。
func (a *app) runPlayerAction(u *game.Unit, act game.Action) {
	switch act {
	case game.ActionAttack:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf("%s 正前方沒有敵人", u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf("%s 行動點不足", u.Name)
			return
		}
		a.reportAttack(u, target, a.battle.ResolveAttack(u, target, 0))

	case game.ActionTurnUndead:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf("%s 正前方沒有敵人", u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf("%s 行動點不足", u.Name)
			return
		}
		if game.TurnUndead(a.rng, u, target) {
			a.battle.Kill(target)
			a.logf("%s 驅散了 %s", u.Name, target.Name)
		} else {
			a.logf("%s 的驅散無效", u.Name)
		}

	case game.ActionPray:
		if _, ok := a.battle.Spend(act); !ok {
			a.logf("%s 行動點不足", u.Name)
			return
		}
		granted, next := game.Pray(a.rng, a.prayChance)
		a.prayChance = next
		if granted {
			a.logf("%s 的祈禱得到回應", u.Name)
		} else {
			a.logf("%s 的祈禱沒有回應", u.Name)
		}

	case game.ActionLeech:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf("%s 正前方沒有敵人", u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf("%s 行動點不足", u.Name)
			return
		}
		if ok, amount := game.Leech(a.rng, u, target.CurrentSP); ok {
			target.CurrentSP -= amount
			u.CurrentSP += amount
			a.logf("%s 汲走 %s 的 %d 點法力", u.Name, target.Name, amount)
		} else {
			a.logf("%s 的汲取失敗", u.Name)
		}

	case game.ActionDodge:
		bonus := a.battle.DoDodge()
		a.logf("%s 進入閃避（命中率 %d）", u.Name, game.DodgeHitModifier(bonus))

	case game.ActionEndTurn:
		a.battle.Spend(act)
		a.logf("%s 結束回合", u.Name)
	}
}

// monsterTurn 讓一隻怪物行動。
//
// AI 很簡單：前方有人就打，沒有就轉向最近的敵人，都不行就結束回合。
// 這不是原版行為 —— 原版的怪物 AI 尚未反組譯，這裡先讓戰鬥能跑完。
func (a *app) monsterTurn(u *game.Unit) {
	if target := a.battle.TargetInFront(u); target != nil {
		if _, ok := a.battle.Spend(game.ActionAttack); ok {
			a.reportAttack(u, target, a.battle.ResolveAttack(u, target, 0))
			return
		}
	}

	enemies := a.battle.Enemies(u)
	if len(enemies) == 0 {
		a.battle.Spend(game.ActionEndTurn)
		return
	}
	target := a.battle.Unit(enemies[0])
	if want, ok := stepToward(u, target); ok && game.Facing(u.Facing) != want {
		if a.battle.TurnTo(u, turnActionToward(game.Facing(u.Facing), want)) {
			return
		}
	}
	if a.battle.Step(u) {
		return
	}
	a.battle.Spend(game.ActionEndTurn)
}

// stepToward 回傳朝目標靠近該面向哪一邊。差距大的那個軸優先。
func stepToward(u, target *game.Unit) (game.Facing, bool) {
	dx, dy := target.X-u.X, target.Y-u.Y
	if dx == 0 && dy == 0 {
		return 0, false
	}
	if abs(dx) >= abs(dy) {
		if dx > 0 {
			return game.East, true
		}
		return game.West, true
	}
	if dy > 0 {
		return game.South, true
	}
	return game.North, true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// turnActionToward 挑一個轉向動作，往目標面向靠近一格。
func turnActionToward(cur, want game.Facing) game.Action {
	switch {
	case cur.CW() == want:
		return game.ActionTurnCW
	case cur.CCW() == want:
		return game.ActionTurnCCW
	default:
		return game.ActionAboutFace
	}
}

func (a *app) reportAttack(attacker, target *game.Unit, res game.AttackResult) {
	switch {
	case !res.Hit:
		a.logf("%s 落空", attacker.Name)
	case res.NoEffect:
		a.logf("%s 對 %s 無效", attacker.Name, target.Name)
	case res.Killed:
		a.logf("%s 擊殺 %s（%d 點）", attacker.Name, target.Name, res.Damage)
	default:
		verb := "命中"
		if res.Critical {
			verb = "重擊"
		}
		a.logf("%s %s %s %d 點", attacker.Name, verb, target.Name, res.Damage)
	}
}

var facingArrow = []string{"↑", "→", "↓", "←"}

// gridColor 是戰場格線的顏色，刻意比行動游標暗。
var gridColor = color.RGBA{0x40, 0x40, 0x40, 0xff}

// drawBattlefield 畫戰鬥網格：雙方單位的位置與面向。
//
// **底圖是空的。** 手冊說戰場是「該區域的放大地圖」，但那張放大圖怎麼生成
// 還沒反組譯出來。畫格線比畫一張猜的地形圖誠實 ——
// 玩家至少看得出誰站哪一格。
func (a *app) drawBattlefield(dst *ebiten.Image) {
	cell := gfx.TileWidth * layout.TileScale
	cur := a.battle.Current()

	for gy := 0; gy < layout.MapHeight/cell; gy++ {
		for gx := game.BattleGridMinX; gx < layout.MapWidth/cell; gx++ {
			ui.StrokeRect(dst, gx*cell, gy*cell, cell, cell, gridColor)
		}
	}

	for _, u := range a.battle.Units() {
		if !u.Alive() {
			continue
		}
		x, y := u.X*cell, u.Y*cell
		if x < 0 || x >= layout.MapWidth || y < 0 || y >= layout.MapHeight {
			continue
		}
		mark := "怪"
		if u.IsPlayer {
			mark = "我"
		}
		a.font.Draw(dst, mark, x, y)
		a.font.Draw(dst, facingArrow[u.Facing&3], x+16, y)
		if u == cur {
			ui.StrokeRect(dst, x, y, cell, cell, markerColor)
		}
	}
}

// drawBattle 畫戰鬥狀態：行動點、單位血量、最近幾行紀錄、可用指令。
func (a *app) drawBattle(dst *ebiten.Image) {
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	cur := a.battle.Current()
	line(fmt.Sprintf("戰鬥　第 %d 回合", a.battle.Round()))
	if cur != nil {
		line(fmt.Sprintf("%s 行動　%d 點", cur.Name, a.battle.Points()))
	}

	// 單位最多 15 個，加上抬頭與紀錄會超出地圖高度。先算出能放幾行，
	// 剩下的留給戰鬥紀錄 —— 紀錄比完整名單重要，看不到最後一擊很難受。
	const headerLines = 2
	rows := layout.MapHeight/ui.LineHeight - headerLines - battleLogLines - 1
	units := a.battle.Units()
	hidden := 0
	if len(units) > rows {
		hidden = len(units) - rows
		units = units[:rows]
	}

	for _, u := range units {
		tag := " "
		if u == cur {
			tag = ">"
		}
		state := ""
		if !u.Alive() {
			state = " 陣亡"
		}
		name := u.Name
		if len(name) > 8 {
			name = name[:8]
		}
		line(fmt.Sprintf("%s%-8s%3d/%-3d%s", tag, name, u.HP, u.MaxHP, state))
	}
	if hidden > 0 {
		line(fmt.Sprintf(" …另有 %d 個單位", hidden))
	}

	line("")
	// 紀錄依面板寬度斷行。不斷行的話長名字會把「命中 12 點」的尾巴
	// 畫到畫布外，看起來像少了一個字。
	width := layout.CanvasWidth - layout.StatusX
	for _, s := range a.log {
		for _, ln := range textlayout.WrapMixed(s, width) {
			line(ln)
		}
	}
}

// drawBattleCommands 把可用指令畫在文字視窗的位置。
func (a *app) drawBattleCommands(dst *ebiten.Image) {
	cur := a.battle.Current()
	y := layout.TextBoxTop + layout.BoxPadY

	if a.battle.Outcome() != game.Ongoing {
		a.font.Draw(dst, "戰鬥結束　空白鍵：繼續", layout.BoxPadX, y)
		return
	}
	if cur == nil || !cur.IsPlayer {
		a.font.Draw(dst, "空白鍵：讓對方行動", layout.BoxPadX, y)
		return
	}

	// 付不起的指令仍然列出來但標成不可用 —— 藏起來會讓玩家以為功能不存在。
	x := layout.BoxPadX
	for i, c := range playerCommands {
		label := c.label
		if !a.battle.CanAct(c.action) {
			label = "(" + label + ")"
		}
		if i == 3 {
			x = layout.BoxPadX
			y += ui.LineHeight
		}
		x = a.font.Draw(dst, label+"  ", x, y)
	}
	a.font.Draw(dst, "方向鍵：轉向／前進　Enter：前進",
		layout.BoxPadX, y+ui.LineHeight)
}
