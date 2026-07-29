package main

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 跨地圖出口（`EXITS.DAT`，`docs/re/78` §6）。
//
// 這是 `CONTEXT.md` §7 A2 動作層裡最後一個擋住主線的東西：
// 出口表早就解好、也載進來了，**但從來沒有人查過它** ——
// `a.exits` 在啟動時載入之後就躺在那裡。
// 症狀是玩家走到樓梯上什麼都不會發生，而畫面上沒有任何異常。
//
// 觸發條件不是「每一步都查」，而且**不是看 tile 值，是看可通行表的值**：
//
//	可通行表[tile & 0x7f] == 0xfd  →  這一格是出口
//
// 來源是 `docs/re/58` 那張分派表的動作 `0x14`（`可通行性表 [0x5500+tile] == 0xfd`），
// 對應動作 `0x14` → `222f:32d4` → 重載出口表＝換地圖。
//
// ⚠ **`docs/re/05` §3 說閘門是「tile 遮罩後等於 `0x11` 或 `0x53`」，那是錯的。**
// 從起點走得到的五個出口，來源 tile 是 `0x24`／`0x13`／`0x00` ——
// 一個 `0x11` 都沒有。照那個條件寫，出口永遠不會觸發，
// 而症狀是「站在樓梯上什麼都不會發生」，看不出是條件錯了。
// 詳見 `docs/re/85`。
//
// **這個設計還解掉一個謎**：`EXITS.DAT` 沒有「來源地圖」欄位（`docs/re/78` §6），
// 55 筆座標怎麼分給 26 張地圖？答案是**由可通行表值 `0xfd` 分**：
// 同一組 (X,Y) 在別的地圖上那一格不是 `0xfd`，就不算出口。
// 實測 53/55 筆恰好被一張地圖認領，**沒有任何一筆被兩張圖同時認領**
//（`exitmaps_test.go`）。

// exitPassability 是「這一格是出口」的可通行表值。
const exitPassability = 0xfd

// checkExit 看腳下這一格是不是出口，是就換地圖。
//
// 回傳 true 表示換了地圖 —— 呼叫端要跳過這一步其餘的處理
// （踩到出口就離開這張圖了，原地的事件不該再觸發）。
func (a *app) checkExit(tile byte) bool {
	if a.exits == nil {
		return false
	}
	if a.tables.Passability(tile&0x7f).Raw() != exitPassability {
		return false
	}
	rec, ok := a.exits.Lookup(byte(a.party.X()), byte(a.party.Y()))
	if !ok {
		return false
	}
	if !a.changeMap(int(rec.ToMap), int(rec.ToX), int(rec.ToY)) {
		return false
	}
	// 原版換圖常式把 EXITS.DAT 第六欄寫進存檔 +0xaf；進圖初始化再
	// 以它設定遭遇／商隊共用的 ds:0x5c60（docs/re/107 §3）。
	a.save.MerchantBase = rec.MerchantBase
	return true
}

// changeMap 換到另一張地圖的指定座標。
//
// **`nSS.DAT` 不必在這裡存回。** `a.special` 是 per-map 的，
// 事件改寫的是那張圖自己那一份，物件留在記憶體裡 ——
// 換回來時狀態還在（原版是用 `ds:0x5c52` 那組旗標做同一件事，
// 見 `0x191c4`：換圖前把緩衝區刷進該圖的快取）。
func (a *app) changeMap(id, x, y int) bool {
	m, err := world.LoadByID(a.saveDir(), a.dataDir, id)
	if err != nil {
		// 換不過去就留在原地並說清楚 —— 靜默失敗會讓玩家
		// 以為那一格本來就沒事，然後在樓梯上反覆踩。
		a.message = fmt.Sprintf(a.tr.UI("mapchange.error"), id, err)
		return false
	}

	a.tiles = m
	a.mapID = id
	a.selectEventTable(id)
	a.world = game.NewWorld(m, a.tables)
	// Boardable 是閉包，重建 world 之後要重新掛上，
	// 不然換過一次地圖之後船就上不去了（而且完全沒有錯誤訊息）。
	a.world.Boardable = func(bx, by int) bool {
		return game.ReachableBoatAt(&a.save.Ships, bx, by, a.mapID) >= 0
	}
	a.drawTiles = ditheredTiles(m, uint16(a.ditherSeed), a.save.TempleRuins)
	a.party.TeleportTo(x, y)
	// 存檔要跟著走，不然存檔讀回來會在新座標配舊地圖。
	a.save.MapID = byte(id)
	a.message = fmt.Sprintf(a.tr.UI("mapchange.entered"), id, x, y)
	a.trace.note("換地圖 → %d (%d,%d)", id, x, y)
	return true
}

// selectEventTable 讓 DATA*.TXT 與地城編號同步。戶外地圖（>=9）沒有
// nSS 特殊格，所以保留目前指標即可；進 1..5 時一定切到同號資料表。
func (a *app) selectEventTable(id int) {
	if a.eventsOverride != "" || id < 1 || id > 5 {
		return
	}
	if table := a.eventTables[id]; table != nil {
		a.events = table
		a.eventsFile = fmt.Sprintf("DATA%d.TXT", id)
	}
}
