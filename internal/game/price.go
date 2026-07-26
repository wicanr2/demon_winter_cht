package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

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

// ItemValueBase 回傳道具的基礎估價：`ITEMS.DAT 的底價 × 材質倍率`。
//
// **這不是完整的售價。** 原版還會加上兩項（`docs/re/44` §3）：
//
//   - 已鑑定的話加 `(槽+0x02 + 槽+0x04) × 225 × 1.2`
//   - 強度不為 0 的話再加一段與 `5 × 強度²` 有關的軟浮點項
//
// 那兩項的浮點運算順序還沒讀完，所以這裡只給第一項。
// 起始存檔的十件裝備 `+0x02`／`+0x04`／強度全是 0、材質類別都是 1 ——
// 對它們來說 `ItemValueBase` 就是完整售價，剛好等於底價。
func ItemValueBase(basePrice int, slot scenario.InventorySlot) int {
	return basePrice * MaterialMultiplier(slot.MaterialClass)
}
