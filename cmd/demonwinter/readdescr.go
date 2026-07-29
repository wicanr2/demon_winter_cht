package main

import (
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// `R) Read descr.` —— 重讀腳下這一格的敘述
//
// 原版主選單的第 13 項（`docs/re/96` 那張 15 格表），走移動迴圈的
// switch case 6（`222f:1282`）：
//
//	if (0x4e2e != 1) return 0xfffd      ; 沒東西可讀
//	0x4e2e = 2
//	FUN_222f_0a90()                     ; 再查一次，這次閘門放行
//
// **這是類別 1 與類別 2 唯一看得見的差別。** 類別 2 的事件看過之後
// 站上去會武裝倒數，按 `R` 就能再讀一次；類別 1 看過就永遠沒了，
// 倒數從沒被武裝過，按 `R` 什麼都不會發生。
//
// 規則層在 `internal/game/eventgate.go`。

// readDescription 處理 `R`。
func (a *app) readDescription() {
	// 世界地圖以外的畫面各自吃自己的按鍵，這裡只在移動模式下才有意義；
	// 呼叫端已經在移動迴圈那一層，不必再判。
	counter, ok := game.RequestReread(a.reread)
	a.reread = counter
	if !ok {
		// 原版就是靜默回 `0xfffd` —— 沒有「這裡沒有東西可讀」這種訊息。
		// 本專案補一則，因為缺了它玩家分不出「沒事」與「按錯鍵」。
		a.message = a.tr.UI("read.nothing")
		return
	}
	under, ok := a.world.TileUnder(a.party)
	if !ok {
		return
	}
	a.checkEvent(under)
}

// armReread 只跑閘門的「武裝」那一支，**永遠不會觸發事件**。
//
// 原版每一次指令都會重查腳下這一格（`222f:0c17` 排在移動處理之前），
// 所以站著撞牆或轉向都能把「重讀」備好。本專案的事件檢查只掛在
// 「走成功」那一條，補這一支把觀察得到的行為對回去 ——
// 只取閘門回傳 `EventNone` 時的倒數，其餘一律不動。
func (a *app) armReread() {
	st := a.special[a.mapID]
	if st == nil {
		return
	}
	hit := st.Lookup(byte(a.party.X()), byte(a.party.Y()))
	if hit == nil || hit.Tile.Dead() {
		return
	}
	act, counter := game.EventGate(int(hit.Tile.Class()),
		int(hit.Tile.Value()), a.reread)
	if act == game.EventNone {
		a.reread = counter
	}
}
