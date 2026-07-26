package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

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

// siteTiles 是五個「地點」tile。
//
// 這五個值是逐位元組解出來的 —— Ghidra 在此處失步，把 `CMP AX,0x64 / JZ`
// 整組漏掉，只讀反組譯輸出會少掉 0x64。
//
// ⚠ 舊註解寫「tile 值本身就是 DATA*.TXT 記錄索引」，**那是錯的**
// （`docs/re/79`）。原版收到回傳碼 `0x13` 之後跑的是 `222f:3233`，
// 那支函式把 tile 值當**設施選擇器**分派，不是拿去查文字表：
//
//	0x2e / 0x64 → 城鎮      0x25 → 神殿      0x26 → 學院      0x5b → 廢墟
//
// 拿 tile 值當文字索引的話，`0x25` 會去取第 37 筆敘述 ——
// 而 `DATA1.TXT` 只有 35 筆，所以錯得剛好「什麼都沒發生」，沒人注意到。
var siteTiles = map[byte]bool{
	0x25: true, 0x26: true, 0x2e: true, 0x5b: true, 0x64: true,
}

// SiteKind 是地點 tile 分派到的設施。
type SiteKind int

const (
	// SiteNone 這個 tile 不是地點 tile。
	SiteNone SiteKind = iota
	SiteTown
	SiteTemple
	SiteCollege
	// SiteRuins 是廢墟：印「You are walking through ruins」，不進任何設施。
	SiteRuins
)

// SiteFor 決定地點 tile 要進哪個設施，含兩個「世界壞掉了」的覆寫。
//
// 覆寫的門檻**刻意不一致**，照原版：神殿是 `> 0x7f`（`0x1739a` 的
// `cmp 0x7f / jbe`），城鎮是 `!= 0`（`0x19135` 的 `cmp 0 / jne`）。
// 統一成同一種比較會在旗標取中間值時行為分歧 ——
// 這與符印旗標「進門看 `!= 0`、傷害看 `< 0x80`」是同一類陷阱（`docs/re/63`）。
func SiteFor(tile, templeRuins, shardShattered byte) SiteKind {
	switch tile {
	case 0x2e:
		if shardShattered != 0 {
			return SiteRuins
		}
		return SiteTown
	case 0x64:
		// `0x64` **沒有**廢墟覆寫 —— 原版只對 `0x2e` 檢查那個旗標。
		return SiteTown
	case 0x25:
		if templeRuins > scenario.TempleRuinsThreshold {
			return SiteRuins
		}
		return SiteTemple
	case 0x26:
		return SiteCollege
	case 0x5b:
		return SiteRuins
	}
	return SiteNone
}

// TriggerKind 是落點 tile 決定的觸發方式。
type TriggerKind int

const (
	// TriggerNone 這個 tile 不觸發任何事件，連查都不查。
	TriggerNone TriggerKind = iota
	// TriggerLookup 要走特殊格清單查表。
	TriggerLookup
	// TriggerSite 是地點 tile（城鎮／神殿／學院／廢墟），用 SiteFor 分派。
	// 不查特殊格清單。
	TriggerSite
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
	case siteTiles[tile]:
		return TriggerSite
	default:
		return TriggerNone
	}
}
