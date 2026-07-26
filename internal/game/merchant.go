package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 商隊遭遇（`DEMON.INT 0x1d560`–`0x1d6b5`，見 `docs/re/32`）。
//
// 掉寶生成器有兩個呼叫端，戰鬥勝利是一個，這是另一個：
// 路上遇到一群商人，打招呼就能看他們的貨。
//
// **他們的貨可能被詛咒。** 這是商隊與市集最大的差別 —— 市集賣的是
// 沒有效果的平凡裝備（安全但無趣），商隊賣的是掉寶生成器生出來的東西
// （可能有效果、也可能是負附魔的詛咒品），而且**未鑑定**。

// 商隊規模對應的形容詞索引表（`ds:0x3612`）。索引是規模、值是形容詞編號。
//
// 規模 2 與 3 共用「travelling」、6 與 7 共用「upper class」、
// 8 與 9 共用「wealthy」—— 七個形容詞蓋十種規模。
var merchantAdjectiveIndex = [MerchantMaxSize + 1]int{0, 1, 2, 2, 3, 4, 5, 5, 6, 6}

// MerchantMaxSize 是商隊規模的上限。原版算出來之後鉗在 9（`0x1d567`）。
const MerchantMaxSize = 9

// merchantAdjectives 是七個形容詞（`ds:0x3632` 遠指標表的前七項）。
// 譯名走 assets/lang 的 MERCHANT 目錄。
var merchantAdjectives = []string{
	"ragged looking",
	"poor looking",
	"travelling",
	"adventuring",
	"well dressed",
	"upper class",
	"wealthy",
}

// MerchantAdjective 依商隊規模回傳形容詞（原文）。
//
// 訊息是 `"You see a group of" + 形容詞 + " merchants"`。
func MerchantAdjective(size int) string {
	if size < 0 {
		size = 0
	}
	if size > MerchantMaxSize {
		size = MerchantMaxSize
	}
	return merchantAdjectives[merchantAdjectiveIndex[size]]
}

// 商隊貨物的兩個擲點（`0x1d63c`／`0x1d656`）。
const (
	// 詛咒機率 = rnd(120) − 80，負的鉗成 0 —— 所以**最多四成的貨是詛咒品**，
	// 而且有一半以上的商隊完全乾淨。
	merchantCurseDie    = 120
	merchantCurseOffset = 80

	// 貨物件數 = rnd(4) + 6 → 7–10 件。
	merchantWaresDie  = 4
	merchantWaresBase = 6
)

// MerchantCurseChance 擲出這一支商隊的詛咒機率（百分比，已鉗在 0 以上）。
func MerchantCurseChance(r *rng.RNG) int {
	c := r.Roll(merchantCurseDie) - merchantCurseOffset
	if c < 0 {
		return 0
	}
	return c
}

// MerchantWareCount 擲出這一支商隊帶了幾件貨（7–10）。
func MerchantWareCount(r *rng.RNG) int {
	return r.Roll(merchantWaresDie) + merchantWaresBase
}

// MerchantWare 是商隊的一件貨：東西本身，加上這一家的開價。
type MerchantWare struct {
	Item scenario.InventorySlot
	// Price 是這一家開的價（`docs/re/45`）。**同一件東西在不同商隊
	// 手上差價可以到兩倍多** —— 那是隨機係數，不是議價。
	Price int
	// PriceExact 為 false 代表估價還缺一項（`docs/re/46` §3）。
	// 這種貨**不賣** —— 標一個算不準的價錢比不賣更糟。
	PriceExact bool
	// Sold 為 true 代表已經被買走。
	Sold bool
}

// Merchant 是一支商隊。
type Merchant struct {
	// Size 是規模，決定形容詞。
	Size int
	// Level 是商隊等級，掉寶生成器拿它當品質基準。
	Level int
	// CurseChance 是每件貨被詛咒的百分比機率。
	CurseChance int
	// Wares 是他們帶的 7–10 件貨。
	//
	// **生成時未鑑定，列出來給玩家看的時候標成已鑑定**
	// （原版 `0x1d6c8`，見 `docs/re/44` §2）—— 商人願意讓你看清楚，
	// 但那不改變東西本身可能帶詛咒的事實。
	Wares []MerchantWare
}

// Adjective 回傳這支商隊的形容詞（原文）。
func (m Merchant) Adjective() string { return MerchantAdjective(m.Size) }

// RollMerchant 生出一支商隊的貨，連同各自的開價。
//
// 每件貨各自擲一次詛咒（`rnd(100) < 詛咒機率`），中的那件附魔取負且
// **一定沒有效果** —— 詛咒品在 GenerateLoot 裡就被擋掉效果了。
//
// 開價的順序照原版（`0x1d6c8`–`0x1d737`）：**先標成已鑑定，再估價** ——
// 已鑑定會讓估價多一項，所以順序不能對調。
func RollMerchant(r *rng.RNG, t *gamedata.Tables, items *gamedata.ItemTable,
	size, level int) Merchant {

	m := Merchant{
		Size:        size,
		Level:       level,
		CurseChance: MerchantCurseChance(r),
	}
	n := MerchantWareCount(r)
	for i := 0; i < n; i++ {
		typ := LootItemType(r)
		item, err := items.ByIndex(typ)
		if err != nil {
			continue
		}
		cursed := r.Roll(100) < m.CurseChance
		slot := GenerateLoot(r, t, item, typ, level, cursed)
		slot.Identified = true

		value, exact := ItemValue(item.Price, slot)
		m.Wares = append(m.Wares, MerchantWare{
			Item:       slot,
			Price:      MerchantPrice(r, value),
			PriceExact: exact,
		})
	}
	return m
}

// BuyFromMerchant 買下第 i 件貨。
//
// **算不準價錢的貨不賣**（`PriceExact == false`）—— 估價還缺一項
// （`docs/re/46` §3），標一個湊出來的數字給玩家比不賣更糟。
func BuyFromMerchant(m *Merchant, i int, members []Character, gold int) TradeResult {
	if m == nil || i < 0 || i >= len(m.Wares) {
		return TradeResult{Reason: "沒有這一件", Gold: gold}
	}
	w := &m.Wares[i]
	switch {
	case w.Sold:
		return TradeResult{Reason: "這件已經賣掉了", Gold: gold}
	case !w.PriceExact:
		return TradeResult{Reason: "商人說不出這件的價錢", Gold: gold}
	case w.Price > gold:
		return TradeResult{Reason: "金幣不夠", Gold: gold, Slot: -1}
	}
	for n := range members {
		slot := members[n].FreeSlot()
		if slot < 0 {
			continue
		}
		members[n].Inventory[slot] = w.Item
		w.Sold = true
		return TradeResult{OK: true, Gold: gold - w.Price, Slot: slot, Member: n}
	}
	return TradeResult{Reason: "全隊的道具欄都滿了", Gold: gold}
}
