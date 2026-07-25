package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/world"

// EXITS.DAT 在記憶體裡的緩衝區比檔案大：檔案 330 bytes（110 筆 3-byte 記錄），
// 緩衝區 511 bytes，尾端留給傳送目標座標。掃描以 `record[0] == 0` 為終止標記，
// 而真實檔案的 110 筆裡沒有這個標記 —— 它落在檔案之後的零區。
// 見 docs/spec/03-events.md。
const (
	exitBufferSize = 0x1ff // 511
	exitRecordSize = 3
)

// EventCategory 是事件類別（type >> 5，值域 0–7）。
type EventCategory int

const (
	// CatText 是純敘述房間。真實 EXITS.DAT 裡佔 94/110。
	CatText EventCategory = 0
	// CatOneShot 事件索引額外 +1；帶 struct+0xa5 旗標分支。
	CatOneShot EventCategory = 1
	// CatRepeatGuard 事件索引額外 +1，另外武裝防重複觸發的倒數計時器。
	CatRepeatGuard EventCategory = 2
	// CatTeleport 傳送。真實 EXITS.DAT 沒有這一類，實作了但無資料可驗。
	CatTeleport EventCategory = 4
)

// noEvent 是查表沒命中時的回傳碼，對應原版的 0xff。
const noEvent = 0xff

// EventQuery 是一次座標查表的結果。
//
// 原版把輸入座標與輸出結果塞在同一對全域（0x52f4/0x52f6）裡先讀後寫，
// 這裡拆成明確的輸入參數與回傳值，語意等價但不共用儲存。
type EventQuery struct {
	// Found 為 false 表示這個座標沒有事件記錄（原版回傳 0xff）。
	Found bool

	// Category 是事件類別。
	Category EventCategory

	// Index 是 DATA*.TXT 的記錄索引。
	//
	// DATA*.TXT 是變長記錄、無法隨機定位，所以這個值的語意是
	// 「從頭 parse 到第幾筆」的計數，不是位元組位移。
	Index int

	// SubValue 是 type % 32。
	SubValue int

	// RecordIndex 是命中的那筆記錄在 EXITS 表中的位置，
	// 供「打完一次性遭遇後把 type byte 清零」用。
	RecordIndex int

	// TeleportX/TeleportY 只在 Category == CatTeleport 時有意義。
	TeleportX, TeleportY byte
}

// ExitSource 提供 EXITS 表的原始位元組檢視。由 assets 層的 ExitTable 實作。
type ExitSource interface {
	All() []world.ExitRecord
}

// LookupEvent 依座標查事件，對應原版 FUN_222f_1321。
//
// 掃描時同時累計兩個計數器：類別 1／2 累加事件計數、類別 4 累加傳送計數。
// **命中之前**掃過的記錄才計入 —— 這是事件索引的來源，也是為什麼
// 「打完把 type byte 清零」會讓後面所有事件的索引往前移一格。
//
// teleportTail 是 511-byte 緩衝區尾端的傳送目標座標區；傳 nil 表示沒有
// （真實 EXITS.DAT 沒有類別 4 的記錄，這條路徑無資料可驗）。
func LookupEvent(src ExitSource, x, y byte, teleportTail []byte) EventQuery {
	records := src.All()

	eventCount := 0
	teleportCount := 0

	for i, rec := range records {
		// 終止標記。真實檔案的 110 筆都不為 0，標記落在檔案之後的零區。
		if rec.X == 0 {
			return EventQuery{Found: false}
		}

		if rec.X == x && rec.Y == y {
			return finishLookup(rec, i, eventCount, teleportCount, teleportTail)
		}

		switch rec.Type & 0xe0 {
		case 0x20, 0x40:
			eventCount++
		case 0x80:
			teleportCount++
		}
	}

	// 掃完 110 筆都沒命中，等同踩到緩衝區零區的終止標記。
	return EventQuery{Found: false}
}

func finishLookup(rec world.ExitRecord, recIdx, eventCount, teleportCount int, tail []byte) EventQuery {
	q := EventQuery{
		Found:       true,
		Category:    EventCategory(rec.Type / 32),
		Index:       eventCount,
		SubValue:    int(rec.Type % 32),
		RecordIndex: recIdx,
	}

	if q.Category == CatOneShot || q.Category == CatRepeatGuard {
		q.Index++
	}

	if q.Category != CatTeleport {
		return q
	}

	// 傳送目標存在同一張表的尾端、反向成長：第 n 個傳送點的目標座標
	// 在 0x1ff − (2n + 2)。
	idx := exitBufferSize - (teleportCount*2 + 2)
	if tail != nil && idx >= 0 && idx+1 < len(tail) {
		q.TeleportX = tail[idx]
		q.TeleportY = tail[idx+1]
	}
	return q
}

// 觸發閘門用到的 tile 值。移動後看落點 tile 決定要不要查 EXITS。
// 見 docs/spec/03-events.md「觸發閘門」與「第二路徑」。
const (
	tileEventGateA = 0x11
	tileEventGateB = 0x53
	tileHardBlock  = 0x35
)

// directIndexTiles 是「tile 值本身就是 DATA*.TXT 記錄索引」的五個 tile。
// 這條路徑完全不經過 EXITS.DAT。
//
// 這五個值是逐位元組解出來的 —— Ghidra 在此處失步，把 `CMP AX,0x64 / JZ`
// 整組漏掉，只讀反組譯輸出會少掉 0x64。
var directIndexTiles = map[byte]bool{
	0x25: true, 0x26: true, 0x2e: true, 0x5b: true, 0x64: true,
}

// TriggerKind 是落點 tile 決定的觸發方式。
type TriggerKind int

const (
	// TriggerNone 這個 tile 不觸發任何事件，連查都不查。
	TriggerNone TriggerKind = iota
	// TriggerLookup 要走 EXITS.DAT 查表。
	TriggerLookup
	// TriggerDirectIndex tile 值本身就是事件索引，不查 EXITS.DAT。
	TriggerDirectIndex
	// TriggerHardBlock 寫死的阻擋，完全不查 EXITS.DAT。
	TriggerHardBlock
)

// TriggerFor 依落點 tile 值決定觸發方式。tile 應已遮罩過 &0x7f。
//
// **不是每一步都查 EXITS.DAT** —— 只有少數 tile 值會開啟查表，
// 這個閘門要照做，否則每步掃 110 筆不只慢，語意也不對。
func TriggerFor(tile byte) TriggerKind {
	switch {
	case tile == tileHardBlock:
		return TriggerHardBlock
	case tile == tileEventGateA || tile == tileEventGateB:
		return TriggerLookup
	case directIndexTiles[tile]:
		return TriggerDirectIndex
	default:
		return TriggerNone
	}
}
