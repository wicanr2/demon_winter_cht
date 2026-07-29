package main

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// shiftTombstones 是地點劇情 case 5：墓園重排（`docs/re/100`）。
//
// 原版三段：從 `MAP3.MAP` 重讀 → 隨機長 30 塊擋路石 → 三格強制留通。
// **重讀走的是換地圖那條三段優先序**（存檔目錄 → 原版資料目錄），
// 與壓牆走廊的 case 1 同一個理由：玩家推開過的家具不該被這一步還原。
func (a *app) shiftTombstones() {
	m, err := world.LoadByID(a.saveDir(), a.dataDir, a.mapID)
	if err != nil {
		a.message = fmt.Sprintf(a.tr.UI("tombstone.reload_failed"), err)
		return
	}
	a.tiles = m
	stones := game.TombstoneShift(a.rng, a.tiles)
	a.drawTiles = ditheredTiles(a.tiles, uint16(a.ditherSeed), a.save.TempleRuins)

	// 原版印兩行（`The tombstones` / `shift before you`），這裡併成一句。
	a.message = a.tr.UI("tombstone.shift")
	a.trace.note("墓園：重排，長出 %d 塊擋路石", len(stones))
}
