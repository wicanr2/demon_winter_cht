package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 買船與修船。
//
// 這兩項原本只報價不執行 —— 缺的是船隻記錄的格式。這一輪把讀取端與寫入端
// 都讀完了（`docs/re/28`），六個欄位全部定案，於是可以真的放船與修船。
//
// # 船不是「你的」
//
// 原版沒有記載歸屬：修船只看**腳邊有沒有船**，買船只是往海上多放一艘。
// 世界一開始就有兩艘（原版存檔的第 0 與第 9 格），任何人走到旁邊都能修。

// ShipSlots 是船隻陣列的格數。
const ShipSlots = 10

// shipNearRange 是「在腳邊」的判定範圍：X 與 Y 的差都要小於 2。
//
// 原版是 `abs(ship.X − party.X) < 2`（`2aed:115a` 的 `cmp ax,2 / jge 跳過`），
// 也就是含斜角的 3×3。
const shipNearRange = 2

// 海面的兩個 tile。放船的目標格必須是其中之一（`2aed:1337`）。
//
// 兩個值是同一種地形的兩種浪花，載入地圖時隨機摻著鋪 —— 放船的判定
// 兩個都認，正好是「tile 0x62 是海不是牆」的又一個獨立佐證。
const (
	tileOceanA = 0x14
	tileOceanB = 0x62
)

// shipPlacementOrder 是找位置放船的四個方向，順序照原版
// （`2aed:10a5` 起連續四次呼叫）：西、東、北、南。
var shipPlacementOrder = [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

// ShipResult 是一次買船／修船的結果。
type ShipResult struct {
	OK     bool
	Reason string
	Gold   int
	Cost   int
	// Slot 是動到的那一格（買船是放進哪一格，修船是修了哪一艘）。
	Slot int
}

// FindShipNear 找出停在 (x, y) 腳邊的船，回傳格號；沒有回 −1。
//
// **原版在這裡不比對子地圖編號**，只比座標 —— 放船時比對（避免兩艘疊在
// 同一格），修船時卻不比。照著做。
//
// 唯一的偏離：**跳過空格**。原版對 10 格一律讀座標，空格的 (0,0) 在隊伍
// 走到地圖角落時會被判成「腳邊有船」，接著把船體修成 75 —— 憑空生出一艘船。
// 那是原版的疏漏，不是玩法，不照抄。
func FindShipNear(ships *[ShipSlots]scenario.Ship, x, y int) int {
	for i := range ships {
		s := ships[i]
		if !s.Exists() {
			continue
		}
		if absInt(int(s.X)-x) < shipNearRange && absInt(int(s.Y)-y) < shipNearRange {
			return i
		}
	}
	return -1
}

// RepairShip 修好腳邊的船。
//
//	費用 = (75 − 目前船體值) × 城鎮經濟係數 ÷ 2
//
// 滿血的船不收錢也不修（原版印 "Your ship looks fine to me..."）。
func RepairShip(e Economy, ships *[ShipSlots]scenario.Ship, x, y, gold int) ShipResult {
	i := FindShipNear(ships, x, y)
	if i < 0 {
		return ShipResult{Reason: "這裡沒有船可以修", Gold: gold, Slot: -1}
	}
	cost := e.RepairPrice(int(ships[i].Hull))
	if cost == 0 {
		return ShipResult{Reason: "你的船看起來好得很", Gold: gold, Slot: i}
	}
	if cost > gold {
		return ShipResult{Reason: "金幣不夠", Gold: gold, Cost: cost, Slot: i}
	}
	ships[i].Hull = scenario.ShipMaxHull
	return ShipResult{OK: true, Gold: gold - cost, Cost: cost, Slot: i}
}

// BuyShip 買一艘船並停到隊伍旁邊的海面上。
//
// 依序試西、東、北、南四格，每一格要通過兩道檢查：
//
//   - 那一格是海面（tile 0x14 或 0x62）
//   - 同一張子地圖的同一格還沒有別的船
//
// 四格都不行 → 「碼頭滿了」。有位置但 10 格全滿 → 「你已經有 10 艘船了」。
// **錢在成功放船之後才扣**（`2aed:148a` 的 32-bit SUB/SBB 在放完船那條路徑上）。
//
// tileAt 回傳指定座標的地形值，由呼叫端從目前的地圖取。
func BuyShip(ships *[ShipSlots]scenario.Ship, tileAt func(x, y int) byte,
	x, y, mapID, gold, price int) ShipResult {

	if price > gold {
		return ShipResult{Reason: "金幣不夠", Gold: gold, Cost: price, Slot: -1}
	}

	full := false
	for _, d := range shipPlacementOrder {
		tx, ty := x+d[0], y+d[1]
		if t := tileAt(tx, ty); t != tileOceanA && t != tileOceanB {
			continue
		}
		if shipAt(ships, tx, ty, mapID) >= 0 {
			continue
		}
		slot := freeShipSlot(ships)
		if slot < 0 {
			// 有水位可停但陣列滿了 —— 原版這兩種情況的訊息是分開的。
			full = true
			continue
		}
		ships[slot] = scenario.NewShip(byte(tx), byte(ty), byte(mapID))
		return ShipResult{OK: true, Gold: gold - price, Cost: price, Slot: slot}
	}
	if full {
		return ShipResult{Reason: "你已經有 10 艘船了", Gold: gold, Cost: price, Slot: -1}
	}
	return ShipResult{Reason: "碼頭滿了，附近沒有水位可以停船",
		Gold: gold, Cost: price, Slot: -1}
}

// shipAt 找出停在指定座標與子地圖的船，沒有回 −1。
func shipAt(ships *[ShipSlots]scenario.Ship, x, y, mapID int) int {
	for i := range ships {
		s := ships[i]
		if !s.Exists() {
			continue
		}
		if int(s.X) == x && int(s.Y) == y && int(s.MapID) == mapID {
			return i
		}
	}
	return -1
}

// freeShipSlot 回傳第一個空格。**空格的判定是船體值為 0**，不是座標為 0 ——
// (0,0) 是合法座標。
func freeShipSlot(ships *[ShipSlots]scenario.Ship) int {
	for i := range ships {
		if !ships[i].Exists() {
			return i
		}
	}
	return -1
}
