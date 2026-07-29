package main

import (
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// circleOfLightDoor 是地點劇情表的 case 15：光之環入口的緋紅力場
// （`docs/re/65` §3、`docs/re/98` §1）。
//
// 原版的順序是「先讓隊伍走上觸發格 → 印訊息 → 把座標寫回去」，
// 不是「不准走」。之前引擎在移動前擋，而且**只看子地圖編號不看座標** ——
// 副作用是符印沒解完就在整張地圖 5 上一步都走不了。
//
// 三個符印都解完時原版直接回 3，**什麼訊息都不印**（`0x1a5c5`）：
// 力場散了，那四格就只是普通的地板。
func (a *app) circleOfLightDoor() {
	if game.CircleOfLightOpen(a.save.GlyphFlags) {
		return
	}
	a.message = a.tr.UI("plot.forcefield")
	x, y := game.CircleOfLightPushBack(a.party.X(), a.party.Y(), a.party.Facing())
	a.party.TeleportTo(x, y)
}
