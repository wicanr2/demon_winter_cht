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
// **他們的貨是掉寶生成器生出來的**，這是商隊與市集最大的差別 ——
// 市集賣的是沒有效果的平凡裝備（安全但無趣），商隊賣的可能帶效果。
//
// 商隊那條路**不會生出詛咒品**：生成器的詛咒判定只在掉落模式跑
// （`docs/re/48` §5）。以前這裡寫「可能被詛咒」，是把商隊那個機率
// 參數的用途讀錯了。

// 商隊規模對應的形容詞索引表（`ds:0x3612`）。索引是規模、值是形容詞編號。
//
// 規模 2 與 3 共用「travelling」、6 與 7 共用「upper class」、
// 8 與 9 共用「wealthy」—— 七個形容詞蓋十種規模。
var merchantAdjectiveIndex = [MerchantMaxSize + 1]int{0, 1, 2, 2, 3, 4, 5, 5, 6, 6}

// MerchantMaxSize 是商隊規模的上限。原版算出來之後鉗在 9（`0x1d567`）。
const MerchantMaxSize = 9

// 規模的擲點（`0x1d555`，見 `docs/re/50`）。
//
//	規模 = clamp(基準 + rnd(3) − 2, ≤ 9)
//
// 基準來自存檔的 `+0xaf`（戶外會被地圖記錄自帶的參數覆蓋，那個來源沒解）。
const (
	merchantSizeDie    = 3
	merchantSizeOffset = 2
)

// MerchantSize 依基準值擲出這一支商隊的規模。
//
// **規模同時就是商隊等級** —— 原版是同一個區域變數，算完規模之後直接
// 當等級傳給掉寶生成器（`0x1d6ad`）。所以「衣衫襤褸的商隊」賣的東西
// 一定也差，兩件事本來就是一回事。
//
// 原版只鉗上限；下限這裡補一個 0，免得基準值太小時算出負數去索引
// 形容詞表（原版會讀到表前面的 bytes）。
func MerchantSize(r *rng.RNG, base int) int {
	size := base + r.Roll(merchantSizeDie) - merchantSizeOffset
	if size > MerchantMaxSize {
		return MerchantMaxSize
	}
	if size < 0 {
		return 0
	}
	return size
}

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
	// 這個機率 = rnd(120) − 80，負的鉗成 0，值域 0–40，
	// **而且有一半以上的商隊擲到 0**。
	//
	// 它以前被讀成「詛咒機率」，讀錯了（`docs/re/48` §5）：生成器在商隊
	// 模式下拿它擲中的貨**跳過第一道效果門檻**，帶效果的機率因此大增
	// （第二道門檻還在，所以不是「一定有」）。
	// 附魔不取負、效果不被擋、驅邪成功率也不寫 —— 一件壞事都沒發生。
	merchantEffectDie    = 120
	merchantEffectOffset = 80

	// 貨物件數 = rnd(4) + 6 → 7–10 件。
	merchantWaresDie  = 4
	merchantWaresBase = 6
)

// MerchantEffectChance 擲出這一支商隊「跳過第一道效果門檻」的百分比機率
// （已鉗在 0 以上）。
func MerchantEffectChance(r *rng.RNG) int {
	c := r.Roll(merchantEffectDie) - merchantEffectOffset
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
	// Sold 為 true 代表已經被買走。
	Sold bool
	// Haggle 是這一件的議價狀態（`docs/re/52` §3）。
	//
	// **商隊的議價一律從 0 開始** —— 原版開畫面時把 31 格計數器全部清 0
	// （`0x1d52e`），不吃市集那個「隊伍裡有人會說服就給初值」的加成。
	Haggle HaggleState
	// Guaranteed 是原版逐件記下來的那個旗標（`ds:0x4e2e`）：這一件跳過了
	// 第一道效果門檻。**商隊後續拿它做什麼還沒讀**（`docs/re/48` §5），
	// 所以目前只保存不使用。
	Guaranteed bool
}

// Merchant 是一支商隊。
type Merchant struct {
	// Size 是規模，決定形容詞。
	Size int
	// Level 是商隊等級，掉寶生成器拿它當品質基準。
	// **與 Size 永遠相同** —— 原版是同一個變數（`docs/re/50`）。
	Level int
	// EffectChance 是每件貨「跳過第一道效果門檻」的百分比機率（見上）。
	EffectChance int
	// Wares 是他們帶的 7–10 件貨。
	//
	// **生成時未鑑定，列出來給玩家看的時候標成已鑑定**
	// （原版 `0x1d6c8`，見 `docs/re/44` §2）—— 商人願意讓你看清楚，
	// 而且看清楚的那個值也是估價的依據。
	Wares []MerchantWare
}

// Adjective 回傳這支商隊的形容詞（原文）。
func (m Merchant) Adjective() string { return MerchantAdjective(m.Size) }

// RollMerchant 生出一支商隊的貨，連同各自的開價。
//
// 每件貨各自擲一次（`rnd(100) < EffectChance`），
// 中的那件跳過第一道效果門檻。原版把結果逐件記在自己的陣列裡
// （`ds:0x4e2e` 進出一次），這裡收在 `MerchantWare.Guaranteed`。
//
// 開價的順序照原版（`0x1d6c8`–`0x1d737`）：**先標成已鑑定，再估價** ——
// 已鑑定會讓估價多一項，所以順序不能對調。
func RollMerchant(r *rng.RNG, t *gamedata.Tables, items *gamedata.ItemTable,
	size int) Merchant {

	m := Merchant{
		Size:         size,
		Level:        size,
		EffectChance: MerchantEffectChance(r),
	}
	n := MerchantWareCount(r)
	for i := 0; i < n; i++ {
		typ := LootItemTypeFor(r, items, m.Level)
		item, err := items.ByIndex(typ)
		if err != nil {
			continue
		}
		slot, guaranteed := GenerateWare(r, t, item, typ, m.Level, m.EffectChance)
		slot.Identified = true

		m.Wares = append(m.Wares, MerchantWare{
			Item:       slot,
			Price:      MerchantPrice(r, ItemValue(item.Price, slot)),
			Guaranteed: guaranteed,
		})
	}
	return m
}

// WarePrice 回傳這一件目前的要價（開價套上議價折扣）。
//
// 議價的規則與市集共用同一段程式（`docs/re/52` §2 的 case 4），
// 所以這裡直接用 `HagglePrice` —— 每成功一次打掉 6%，下限 2 金。
func (w MerchantWare) WarePrice() int { return HagglePrice(w.Price, w.Haggle) }

// HaggleWith 對第 i 件貨議價一次。
//
// **第一次一定成功**（狀態 0 → 兩道門檻都是 0）。之後越議越危險：
// 翻臉的代價是**那一件貨報廢**，不是整支商隊走人。
func HaggleWith(r *rng.RNG, m *Merchant, i int) (HaggleOutcome, bool) {
	if m == nil || i < 0 || i >= len(m.Wares) {
		return HaggleOffended, false
	}
	w := &m.Wares[i]
	if w.Sold || w.Haggle.Refused() {
		return HaggleOffended, false
	}
	outcome, next := Haggle(r, w.Haggle)
	w.Haggle = next
	return outcome, true
}

// BuyFromMerchant 買下第 i 件貨。
//
// 估價六項全解之後（`docs/re/49`）就沒有「說不出價」這回事了 ——
// 那個狀態是公式缺一項時的自保，補齊就拿掉。
func BuyFromMerchant(m *Merchant, i int, members []Character, gold int) TradeResult {
	if m == nil || i < 0 || i >= len(m.Wares) {
		return TradeResult{Reason: "沒有這一件", Gold: gold}
	}
	w := &m.Wares[i]
	switch {
	case w.Sold:
		return TradeResult{Reason: "這件已經賣掉了", Gold: gold}
	case w.Haggle.Refused():
		return TradeResult{Reason: "商人不肯賣這件給你了", Gold: gold}
	case w.WarePrice() > gold:
		return TradeResult{Reason: "金幣不夠", Gold: gold, Slot: -1}
	}
	for n := range members {
		slot := members[n].FreeSlot()
		if slot < 0 {
			continue
		}
		members[n].Inventory[slot] = w.Item
		w.Sold = true
		return TradeResult{OK: true, Gold: gold - w.WarePrice(), Slot: slot, Member: n}
	}
	return TradeResult{Reason: "全隊的道具欄都滿了", Gold: gold}
}
