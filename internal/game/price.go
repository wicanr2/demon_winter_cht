package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 道具估價（`278d:1c1b`，見 `docs/re/44`）。
//
// 這是商隊售價那條線的核心。**目前只解出第一項** —— 底價乘上材質倍率；
// 另外兩項（鑑定加價、效果加價）還沒讀完，見 §「沒解的部分」。

// materialMultiplier 是材質／品質類別（道具槽 `+0x0f`）的價格倍率。
//
// 表在 `ds:0x18b3`，九個 byte：`00 01 02 05 14 23 32 3c 4b`。
// 級距是 0、1、2、5、20、35、50、60、75 —— **最高一級是原價的 75 倍**。
// 同一把匕首，材質類別不同就從幾枚金幣變成一筆財產。
//
// 類別 0 的倍率是 0：那一級的東西不值錢（也可能是「沒有材質」的佔位）。
var materialMultiplier = [...]int{0, 1, 2, 5, 20, 35, 50, 60, 75}

// MaterialClassCount 是材質類別的數量。
const MaterialClassCount = len(materialMultiplier)

// MaterialMultiplier 回傳材質類別的價格倍率。越界回 0（不值錢）。
func MaterialMultiplier(class int) int {
	if class < 0 || class >= MaterialClassCount {
		return 0
	}
	return materialMultiplier[class]
}

// ItemValueBase 回傳估價的第一項：`ITEMS.DAT 的底價 × 材質倍率`。
func ItemValueBase(basePrice int, slot scenario.InventorySlot) int {
	return basePrice * MaterialMultiplier(slot.MaterialClass)
}

// identifiedBonusMul 是已鑑定道具的加價係數。
//
// 原版是 `(槽+0x02 + 槽+0x04) × 225`，再乘上一個 `push` 進去的 double 1.2
// （`1c80`–`1ca1`）。**225 × 1.2 = 270 剛好是整數** —— 用浮點繞一圈只是
// 編譯器把 `* 1.2` 直譯的結果，這裡直接用 270，值完全相同。
const identifiedBonusMul = 270

// ItemValue 回傳道具的估價，以及這個數字**準不準**。
//
// 原版的公式有三項（`docs/re/44` §3）：
//
//  1. 底價 × 材質倍率
//  2. 已鑑定的話加 `(槽+0x02 + 槽+0x04) × 270`
//  3. **強度不為 0 的話**再加一段與 `5 × 強度²` 有關的浮點項 —— 還沒解
//
// 所以 `exact` 只在強度為 0 時為 true。呼叫端**不該把 exact=false 的數字
// 當價錢用** —— 那是缺了一整項的下界，不是估計值。
func ItemValue(basePrice int, slot scenario.InventorySlot) (value int, exact bool) {
	value = ItemValueBase(basePrice, slot)
	if slot.Identified {
		value += (slot.Unknown02 + slot.Unknown04) * identifiedBonusMul
	}
	return value, slot.Power == 0
}

// 商隊售價（`0x1d6e1`–`0x1d727`，見 `docs/re/45`）。
//
// 商人不會照估價賣 —— 每一件貨在列出來的時候乘上一個隨機係數：
//
//	售價 = trunc( 估價 × (uniform() × 0.8 + 0.6) )
//
// 也就是 **[0.6, 1.4) 的 ±40% 浮動**。同一件東西在不同商隊手上差價可以到兩倍多。
const (
	merchantPriceSpread = 0.8
	merchantPriceFloor  = 0.6
)

// MerchantPrice 把估價換算成商隊的開價。
//
// **用 float64 而不是有理數**：原版就是這三步 IEEE double 運算
// （`uniform × 0.8`、`+ 0.6`、`× 估價`，最後截斷），同型別同順序才逐位元相同。
func MerchantPrice(r *rng.RNG, value int) int {
	scale := r.Uniform()*merchantPriceSpread + merchantPriceFloor
	return int(float64(value) * scale)
}
