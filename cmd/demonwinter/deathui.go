package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 全隊死亡畫面（原版動作 `0x18` ＝ `25be:000c`，見 `docs/re/58` §4）。
//
// **在這之前，一支全員陣亡的隊伍照樣可以走路、紮營、進城。**
// 引擎只在戰鬥結算裡印一行「隊伍全滅」的紀錄，然後就回世界地圖 ——
// 而畫面上沒有任何異常。原版有這一格動作，11 個呼叫端散在戰鬥結算、
// 飢餓／中毒推進、事件處理那幾條路上。
//
// 兩處刻意與原版不同：
//
//   - 原版印完回傳 −1，上游把玩家丟回 DOS／建角程式。這裡按鍵結束程式 ——
//     與結局畫面同一個處理（`ending.go` 也是 `ebiten.Termination`）。
//     **本來想回標題**，但 `a.title` 只是開場那張圖不是一個畫面狀態，
//     為了「回標題」去補一套流程不划算，而且原版也沒有那個概念。
//   - **不自動存檔。** 死亡不該覆蓋玩家上一次的進度；存檔留在原地，
//     玩家可以從那裡重來。這是與原版的實質差異裡最重要的一條。

// deathScreen 是全隊死亡的顯示狀態。
type deathScreen struct{}

// checkPartyDeath 在任何可能扣血的動作之後叫一次。
//
// 回傳 true 表示隊伍全滅、畫面已切換 —— 呼叫端要停止這一步剩下的處理。
func (a *app) checkPartyDeath() bool {
	if a.death != nil {
		return true
	}
	if !game.PartyWiped(a.members) {
		return false
	}
	a.death = &deathScreen{}
	a.trace.note("全隊死亡")
	// 戰鬥畫面要收掉 —— 不收的話死亡畫面底下還壓著一場打不完的仗。
	a.battle = nil
	a.settled = false
	return true
}

func (a *app) updateDeath() error {
	if len(inpututil.AppendJustPressedKeys(nil)) == 0 {
		return nil
	}
	// 結束程式。**存檔不動** —— 玩家從上一次存檔重來。
	return ebiten.Termination
}

func (a *app) drawDeath(dst *ebiten.Image) {
	y := ui.LineHeight * 4
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX*2, y)
		y += ui.LineHeight
	}
	for i := range game.PartyDeathLines {
		line(a.tr.UI(deathLineKey(i)))
	}
	line("　")
	line(a.tr.UI("death.dismiss"))
}

// deathLineKey 是第 i 行的翻譯 key。
func deathLineKey(i int) string {
	return "death.line" + string(rune('0'+i))
}
