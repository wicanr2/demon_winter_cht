package game

// 兵器庫（地點劇情 case 3，原版 `25be:085c` ＝ `0x1a03c`，`docs/re/99`）
//
// 地圖 1 上一個長方形的四個角各有一座台座，每座放一件**不同的**劇情裝備，
// 頭上飄著幾團發光的球體。走上去會問「你要靠近嗎？」，答是就送。
//
// 四座台座共用 case 3，靠座標算出 0–3 的編號，那個編號同時是
// 「送道具」常式（`25be:11ff`）的參數與一次性旗標 `+0xb3` 的索引 ——
// 也就是說**四座台座各拿一次，互不影響**。

// armorySpot 是一座台座的座標與它的道具編號。
type armorySpot struct {
	X, Y int
	ID   PlotGiftID
}

// ArmorySpots 是四座台座（地圖 1，`1SS.DAT` 類別 5 值 3 剛好四筆）。
//
// 編號來自原版那條 if 鏈（`0x1a041`–`0x1a069`），**不是照座標排序的**：
//
//	idx = 0
//	if (X == 23 && Y == 27) idx = 1
//	if (X == 33) { idx = 2; if (Y == 31) idx++ }
//
// (23,31) 落在「什麼條件都不符」那一格，所以它拿的是 idx 0。
// 這種「預設值就是一個有效編號」的寫法看不出設計意圖，
// 但四座各對一件道具的結果是自洽的 —— 見 ArmoryGiftFor 的驗證註解。
var ArmorySpots = []armorySpot{
	{23, 31, PlotGiftArmoryChain},
	{23, 27, PlotGiftArmoryMace},
	{33, 27, PlotGiftArmoryDagger},
	{33, 31, PlotGiftArmorySword},
}

// ArmoryGiftFor 照原版的 if 鏈把座標換成道具編號。
//
// **照抄那條鏈，不查表。** 原版對「不是那四格」的座標也會回一個編號
// （預設 0），而 case 3 只掛在那四格上，所以實務上等價；
// 但寫成查表就會多出一條「查不到」的分支，那是原版沒有的狀態。
func ArmoryGiftFor(x, y int) PlotGiftID {
	id := PlotGiftArmoryChain // 0
	if x == 23 && y == 27 {
		id = PlotGiftArmoryMace // 1
	}
	if x == 33 {
		id = PlotGiftArmoryDagger // 2
		if y == 31 {
			id++ // 3
		}
	}
	return id
}
