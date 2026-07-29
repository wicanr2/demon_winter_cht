package main

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 世界子地圖的邊界換圖（步進分派碼 `0x15`／`0x19`）。
//
// **這一段是「解完了但沒有呼叫端」的典型。** `world.CrossEdge` 早就照
// `DEMON.INT` 0x16fec–0x17114 實作好、也有單元測試，`docs/re/58` §2 連
// 兩個回傳碼都列進表裡了 —— 但全專案沒有任何地方呼叫它。
//
// 症狀不是報錯，是**世界只有一格**：新遊戲在子地圖 34，往任何方向走到
// 第 3／60 格就再也過不去，另外 20 張子地圖一張都到不了。而畫面上看起來
// 完全正常（撞牆就是撞牆），所以四輪試玩都沒撞到 —— 那幾輪都用 `-map`
// 直接跳到目標子地圖，跳過了唯一會用到這段程式碼的路徑。
//
// 主線的三個符印在東南角的 55／56／66，`-glyphs` 之所以「非有不可」
// （`docs/re/64` §3），真正的原因就是這裡沒接。
//
// # 原版的兩個分支
//
//	0x16fe1  cmpb $0xa, es:0xa3(%bx)   ; 地城（< 10）整段跳過
//	換圖成功 → +0xa3 ±1／±10、wrap 的那一軸寫 4 或 0x3b、+0xa9 清 0、回 0x15
//	走到邊緣 → 座標**從全域退回移動前的值**、回 0x19
//
// 退回那一支是 `mov 0x50f0,%ax; mov %ax,-0xa(%bp)`（X）與
// `0x50ee`（Y）—— 兩軸都還原，所以隊伍**不會停在邊界格上**。
// 這件事要照做：停在邊界格上的話下一步又觸發一次判定，
// 玩家會看到訊息連發。
//
// 動作 `0x19` 印的是 *"Your crew refuses to sail any further…"*
//（`docs/re/50` 動作表）。**措辭假設玩家在船上**是合理的 ——
// 世界邊緣那一圈是海，走著到不了。原版沒有分「在船上／沒在船上」，
// 這裡也不分。

// crossSubMapEdge 處理剛走到的那一格是不是子地圖邊界。
//
// prevX／prevY 是**移動前**的座標，退回那一支要用。
// 回傳 true 表示這一步到此為止（換了圖或被擋住），呼叫端不要再跑事件。
func (a *app) crossSubMapEdge(prevX, prevY int) bool {
	res := world.CrossEdge(a.mapID, a.party.X(), a.party.Y())

	if res.Blocked {
		a.party.TeleportTo(prevX, prevY)
		a.message = a.tr.UI("world.edge")
		a.trace.note("世界邊緣：子地圖 %d 擋住，退回 (%d,%d)", a.mapID, prevX, prevY)
		return true
	}
	if !res.Crossed {
		return false
	}

	// 原版在子地圖換算之前就能命中相鄰地圖的船。Walk 的 Boardable 已
	// 放行邊界格；換算後用新地圖的精確座標取得同一艘船，完成登船狀態。
	boarded := -1
	if !game.Sailing(a.save.Boat) {
		boarded = game.BoatAt(&a.save.Ships, res.X, res.Y, res.MapID)
	}
	if !a.changeMap(res.MapID, res.X, res.Y) {
		return false
	}
	if boarded >= 0 && a.mapID == res.MapID &&
		a.party.X() == res.X && a.party.Y() == res.Y {
		a.save.Boat = game.BoatValue(boarded)
		a.party.SetSailing(true)
		a.trace.note("跨子地圖登船：第 %d 艘", boarded+1)
	}
	// **`0x15` 的動作是「什麼都不做」**（`docs/re/58` §2）——
	// 換子地圖在移動常式裡就做完了，畫面只是重畫。
	// `changeMap` 給的「進入地圖 N」是給樓梯用的，這裡要清掉：
	// 走世界地圖時每隔幾步就跳一次，那行字會蓋掉真正該看的訊息。
	a.message = ""
	return true
}
