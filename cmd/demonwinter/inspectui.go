package main

import "github.com/wicanr2/demon_winter_cht/internal/game"

// 探查周圍（`I`）的呈現層。
//
// 規則層在 `internal/game/inspect.go`，那裡也記著它為什麼**不在動作分派表裡**。
// 這一支只做兩件事：查出要標哪幾格、存起來給 drawWorld 用。
//
// **原版不印任何訊息。** 標記出現在地圖上就是全部的回饋 ——
// 這張子地圖沒有道具時按 `I` 什麼都不會發生，那是原版的行為，不是漏接。
func (a *app) inspectSurroundings() {
	a.inspectSpots = game.InspectSurroundings(a.itemloc, byte(a.mapID))
	a.trace.note("探查周圍：子地圖 %d 標出 %d 格", a.mapID, len(a.inspectSpots))
}
