package main

import (
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 鐘（case 12）與旅人的床（case 13）—— 兩個看時辰的地點劇情
// （`docs/re/100` §2／§3）。
//
// 兩支都先問是非題，所以走同一個 `confirmScreen`。

// ringBell 是地點劇情 case 12（地圖 1 的 (26,8)）。
func (a *app) ringBell() {
	a.askConfirm(a.tr.UI("bell.ask"), func() {
		if game.RingBell(a.clock.Hour()) == game.BellNothing {
			a.message = a.tr.UI("bell.nothing")
			a.trace.note("鐘：%d 時，還沒入夜", a.clock.Hour())
			return
		}
		opened := game.OpenBellDoor(a.tiles)
		a.drawTiles = ditheredTiles(a.tiles, uint16(a.ditherSeed), a.save.TempleRuins)
		a.message = a.tr.UI("bell.angels")
		a.trace.note("鐘：%d 時，(%d,%d) 的門%s",
			a.clock.Hour(), game.BellDoor.X, game.BellDoor.Y,
			map[bool]string{true: "開了", false: "本來就開著"}[opened])
	}, nil)
}

// sleepAtNpc 是地點劇情 case 13（地圖 4 的 (21,57)）。
//
// **時辰已經過 24 就連問都不問**（原版 `0x1a4bc` 的 JBE 反面直接 return 2）。
func (a *app) sleepAtNpc() {
	if a.clock.Hour() > game.NightHour {
		a.trace.note("旅人的床：%d 時，已經太晚了", a.clock.Hour())
		return
	}
	a.askConfirm(a.tr.UI("npcbed.ask"), func() {
		// **只撥時鐘，不回血、不回法力、不換日。**
		// 原版就只有 `party[+0x9f] = 25` 這一行 —— 與紮營睡覺是兩套機制。
		a.clock.SleepUntil(game.BellSleepHour, false)
		a.message = a.tr.UI("npcbed.slept")
		a.trace.note("旅人的床：睡到 %d 時", a.clock.Hour())
	}, func() {
		a.message = a.tr.UI("npcbed.farewell")
		a.trace.note("旅人的床：婉拒")
	})
}
