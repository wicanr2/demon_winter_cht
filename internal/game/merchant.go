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

// Merchant 是一支商隊。
type Merchant struct {
	// Size 是規模，決定形容詞。
	Size int
	// Level 是商隊等級，掉寶生成器拿它當品質基準。
	Level int
	// CurseChance 是每件貨被詛咒的百分比機率。
	CurseChance int
	// Wares 是他們帶的 7–10 件貨，全部未鑑定。
	Wares []scenario.InventorySlot
}

// Adjective 回傳這支商隊的形容詞（原文）。
func (m Merchant) Adjective() string { return MerchantAdjective(m.Size) }

// RollMerchant 生出一支商隊的貨。
//
// 每件貨各自擲一次詛咒（`rnd(100) < 詛咒機率`），中的那件附魔取負且
// **一定沒有效果** —— 詛咒品在 GenerateLoot 裡就被擋掉效果了。
//
// 這裡不決定價格：**價格公式還沒解出來**，所以本專案還沒有購買介面
// （見 `docs/re/32` §4）。生成本身是完整的。
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
		m.Wares = append(m.Wares, GenerateLoot(r, t, item, typ, level, cursed))
	}
	return m
}
