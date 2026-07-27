package main

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 壓牆走廊的操作層（地點劇情 case 1／2，地圖 1）。
//
// 規則在 `internal/game/crushingwalls.go`。這裡做的是重載地圖、重畫、
// 以及被夾到時走全隊死亡那條路。
//
// **這是 A7（地點劇情 16 格）第一組接上的非對話 case。** 之前接的
// 10／11（密語）與 14（艾瑞戈爾）都是文字互動，這一組會改地圖也會殺人。

const (
	// plotCaseMachinery 是走廊兩端（地圖 1 的 (15,38)／(23,38)）：
	// **重新載入地圖，牆全部復位**（原版 `0x19f5b` 的 `122f:28d0(1, 0)`）。
	plotCaseMachinery = 1
	// plotCaseCrush 是走廊中段：牆各推進一列，夾到就死。
	plotCaseCrush = 2
	// plotCaseArmory 是兵器庫的四座台座（地圖 1，見 plotgiftui.go）。
	plotCaseArmory = 3
	// plotCaseBlacksmith 是鐵匠鋪（見 plotgiftui.go）。
	plotCaseBlacksmith = 4
)

// resetCrushingWalls 是 case 1：從檔案重讀地圖。
//
// **走的是 `world.LoadByID` 的三段優先序**（存檔目錄 → 原版資料目錄），
// 與換地圖同一條路 —— 這樣玩家推開過的家具（寫在存檔目錄的 `MAP1.MAP`）
// 不會被這一步還原掉，只有壓牆那些「沒存過的」記憶體改動會消失。
// 那正是原版的行為：case 2 不存檔，所以重讀就等於復位。
func (a *app) resetCrushingWalls() {
	m, err := world.LoadByID(a.saveDir(), a.dataDir, a.mapID)
	if err != nil {
		a.message = fmt.Sprintf("重載地圖失敗：%v", err)
		return
	}
	a.tiles = m
	a.drawTiles = ditheredTiles(a.tiles, uint16(a.ditherSeed), a.save.TempleRuins)
	a.message = a.tr.UI("crush.machinery", "聽見沉重機械運轉的聲音")
	a.trace.note("壓牆走廊：機械復位")
}

// advanceCrushingWalls 是 case 2：牆推進一列，夾到就全隊死亡。
func (a *app) advanceCrushingWalls() {
	res := game.AdvanceCrushingWalls(a.tiles, a.party.X(), a.party.Y())
	a.drawTiles = ditheredTiles(a.tiles, uint16(a.ditherSeed), a.save.TempleRuins)

	if !res.Crushed {
		a.message = a.tr.UI("crush.closing", "兩側的牆又逼近了一些")
		a.trace.note("壓牆走廊：推進到第 %d 列", res.Row)
		return
	}
	// 原版印兩行然後走全隊死亡那一支。這裡把全員 HP 歸零再交給既有的
	// 死亡畫面 —— **不要另外做一套死法**，那會漏掉存檔與名單那些收尾。
	for i := range a.members {
		a.members[i].CurrentHP = 0
	}
	a.message = a.tr.UI("crush.crushed", "隊伍被牆壓碎了")
	a.trace.note("壓牆走廊：全隊被壓死")
	a.checkPartyDeath()
}
