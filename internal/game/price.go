package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 道具估價（`278d:1c1b`，見 `docs/re/44`）。
//
// 這是商隊售價那條線的核心。六項全部解出來了，見 ItemValue。

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

// 強度加價的三條分支（`docs/re/46` §3）。三個係數都是「浮點常數乘出整數」
// 的同一個套路：`0.9`、`1.07 × 200 = 214`。
const (
	// 無限次數（`+0x06 == 0xff`）：`5 × 強度² × 次數上限 × 0.9`。
	chargeUnlimitedMul   = 5
	chargeUnlimitedScale = 0.9
	// 次數上限 > 100：`500 × 強度² ÷ (上限 − 100)`，整數除法。
	chargeManyMul   = 500
	chargeManySplit = 100
	// 其餘：`上限 × 強度 × 1.07 × 200`。
	chargeFewMul = 214

	// slotUsedUnlimited 是「無限次數」的哨兵。
	slotUsedUnlimited = 0xff
)

// chargeBonus 是估價的第三項：強度帶來的加價。
//
// 三條分支的分界不是隨便切的：`上限 > 100` 那條要除以 `上限 − 100`，
// **分界正好把除數 0 與負數擋在外面**。守衛與除數對得起來，
// 這是判讀正確的旁證之一。
func chargeBonus(slot scenario.InventorySlot) int {
	p := slot.Power
	switch {
	case slot.Used == slotUsedUnlimited:
		return int(float64(chargeUnlimitedMul*p*p*slot.Total) * chargeUnlimitedScale)
	case slot.Total > chargeManySplit:
		return chargeManyMul * p * p / (slot.Total - chargeManySplit)
	default:
		return slot.Total * p * chargeFewMul
	}
}

// 估價的第四、五項：兩組特效值（`278d:1e13`／`1e97`，見 `docs/re/49`）。
//
//	n = 槽[+0x0a] − 10
//	加價 = trunc(1.4 × |n| × 250) × sign(n)
//
// `+0x0c` 那一組一模一樣。這兩個 byte 是掉寶生成器寫的特效級數
// （`docs/re/48` §6.1），**詛咒品是負的，所以會扣分**。
//
// **這裡不能用「1.4 × 250 = 350」的整數捷徑。** 原版先算 `1.4 × |n|`
// 再乘 250，浮點誤差在第一步就進去了：`n = 3` 時算出來是 1049 不是 1050。
// 第二、三項可以用整數（270／214）是因為它們**先乘整數再乘小數**，
// 順序不同結果就不同 —— 同一個「小數乘出整數」的樣子，安全性不一樣。
const (
	effectValueBase = 1.4
	effectValueMul  = 250.0
)

// 估價的第六項：附魔（`278d:1eeb`）。
//
//	加價 = trunc(1.7 × |附魔| × (武器 350／護甲 700))
//
// **取的是絕對值而且沒有補回符號** —— 負附魔（詛咒品）在這一項照樣加分。
// 型別 13 以上（王冠、飾品、藥水）完全沒有這一項，與掉寶生成器
// 「飾品以上不長附魔」剛好對得起來（`docs/re/48` §4）。
const (
	enchantValueBase   = 1.7
	enchantValueWeapon = 350.0
	enchantValueArmour = 700.0
	// 型別分界（這支常式直接讀槽 `+0x00`，是 0-based）。
	valueWeaponMax = 7
	valueArmourMax = 12
)

// effectValueTerm 算一組特效值的加價。raw 是槽裡的原始 byte。
func effectValueTerm(raw int) int {
	if raw == 0 {
		return 0
	}
	n := raw - storedOffset
	magnitude := float64(n)
	if magnitude < 0 {
		magnitude = -magnitude
	}
	v := int(effectValueBase * magnitude * effectValueMul)
	if n < 0 {
		return -v
	}
	return v
}

// enchantValueTerm 算附魔的加價。
func enchantValueTerm(slot scenario.InventorySlot) int {
	if int(slot.Type) > valueArmourMax {
		return 0
	}
	coefficient := enchantValueWeapon
	if int(slot.Type) > valueWeaponMax {
		coefficient = enchantValueArmour
	}
	magnitude := float64(slot.Enchant)
	if magnitude < 0 {
		magnitude = -magnitude
	}
	return int(enchantValueBase * magnitude * coefficient)
}

// ItemValue 回傳道具的估價。
//
// 六項全部解出來了（`docs/re/44`／`46`／`49`）：
//
//  1. 底價 × 材質倍率
//  2. 已鑑定的話加 `(兩組附帶法術的強度相加) × 270`
//  3. 強度不為 0 的話加 chargeBonus（三條分支）
//  4. 特效值 A 不為 0 的話加 `trunc(1.4 × |n| × 250) × sign(n)`
//  5. 特效值 B 同上
//  6. 附魔（武器 ×350、護甲 ×700，取絕對值）
//
// 最後兩道鉗制：**有驅邪成功率（＝掉落生的詛咒品）就一文不值**，
// 算出負數也歸零。
func ItemValue(basePrice int, slot scenario.InventorySlot) int {
	value := ItemValueBase(basePrice, slot)
	if slot.Identified {
		value += (slot.SpellAPower + slot.SpellBPower) * identifiedBonusMul
	}
	if slot.Power != 0 {
		value += chargeBonus(slot)
	}
	value += effectValueTerm(slot.EffectAByte)
	value += effectValueTerm(slot.EffectBByte)
	value += enchantValueTerm(slot)

	if slot.ExorciseResist != 0 {
		return 0
	}
	if value < 0 {
		return 0
	}
	return value
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
