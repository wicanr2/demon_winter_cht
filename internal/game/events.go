package game

// 事件觸發的閘門：移動後看落點 tile 決定要不要查表。
//
// **座標 → 事件的查表本身不在這裡** —— 它在
// `internal/assets/scenario.SpecialTiles.Lookup`（資料是 `nSS.DAT`）。
// 這個檔案原本有一份 `LookupEvent`，餵的是 `EXITS.DAT`，那是
// `docs/re/05` §1.3 把兩個緩衝區認成同一塊造成的（`docs/re/77` §3）。
// 演算法本身是對的，但輸入的檔案錯了，所以已經整段移除 ——
// 留著兩份實作只會讓下一個人挑錯的那一份用。

// 觸發閘門用到的 tile 值。移動後看落點 tile 決定要不要查特殊格清單。
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
	// TriggerLookup 要走特殊格清單查表。
	TriggerLookup
	// TriggerDirectIndex tile 值本身就是事件索引，不查特殊格清單。
	TriggerDirectIndex
	// TriggerHardBlock 寫死的阻擋，完全不查特殊格清單。
	TriggerHardBlock
)

// TriggerFor 依落點 tile 值決定觸發方式。tile 應已遮罩過 &0x7f。
//
// **不是每一步都查特殊格清單** —— 只有少數 tile 值會開啟查表，
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
