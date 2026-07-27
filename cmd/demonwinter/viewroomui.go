package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 觀室 View Room（`V`，原版動作 `0x0f`，規則在 `internal/game/viewroom.go`）。
//
// 手冊：「透過房門看到隔壁房間的情況」「用觀室技巧也能看到陷阱，
// **但不會標記為『已注意』**」。
//
// 第二句就是這裡最重要的一行程式：命中陷阱時走 **peek 模式** ——
// 原版把 `param != 0` 傳給事件消費者（`25be:0263`），那一支在 peek 模式
// 只印 `A trap!` 就返回：不擲點、不扣血、不改寫 `nSS.DAT`。
// 所以觀室看到的陷阱**踩上去照樣是滿的**，`L` 標記的才會半數失效。

// viewRoom 是 `V` 指令。
func (a *app) viewRoom() {
	res := game.ViewRoom(a.members, a.special[a.mapID], a.world.Tiles(),
		a.party.X(), a.party.Y(), a.party.Facing(), &a.save.ViewRoomUses)

	switch {
	case res.NoSkill:
		// **原版什麼都不印。** 這裡補一句 —— 玩家按了鍵卻毫無反應
		// 會以為是壞掉了，而這個差異不影響任何規則。
		a.message = a.tr.UI("viewroom.noskill", "隊伍裡沒有人會觀室")
		a.trace.note("觀室：沒人會")
		return

	case res.Exhausted:
		a.message = a.tr.UI("viewroom.weak", "你們的靈視之力已經耗盡")
		a.trace.note("觀室：次數用完")
		return

	case res.Hit == nil:
		a.message = a.tr.UI("viewroom.nothing", "什麼也沒看到")
		a.trace.note("觀室：什麼都沒有（剩 %d 次）",
			game.PsychicUsesPerDay-int(a.save.ViewRoomUses))
		return
	}

	a.trace.note("觀室：(%d,%d) 類別 %d", res.X, res.Y, res.Hit.Tile.Class())
	a.peekSpecial(res.Hit)
}

// peekSpecial 用 peek 模式顯示一格特殊格（原版 `25be:0263` 帶 `param != 0`）。
//
// 與踩上去的差別只有一處：**陷阱只announce、不觸發**。
// 其餘類別（劇情、事件文字）照樣完整顯示 —— 原版的 peek 旗標只擋陷阱那一支。
func (a *app) peekSpecial(hit *scenario.SpecialHit) {
	switch cls := hit.Tile.Class(); {
	case cls == scenario.SpecialClassTrap || cls == scenario.SpecialClassTrapAlt:
		// 原版 `0x19a68`：印 `A trap!`、等按鍵、回傳 2。**到此為止。**
		a.message = a.tr.UI("viewroom.trap", "前方有陷阱")
		return

	case hit.Tile.PlotCase() >= 0:
		a.locationPlot(hit.Tile.PlotCase())
		return
	}
	// 事件文字。**不 MarkVisited** —— 那是「走到過」的標記，
	// 隔著牆看一眼不算走到過。
	a.showEvent(hit.EventIndex)
}

// viewRoomKey 是 `V`。手冊把它列在行走模式，與紮營選單的 `V 觀地` 分開
// （觀地是一天一次的鳥瞰，觀室是往前看三格）。
func (a *app) checkViewRoomKey() bool {
	if !inpututil.IsKeyJustPressed(ebiten.KeyV) {
		return false
	}
	a.viewRoom()
	return true
}
