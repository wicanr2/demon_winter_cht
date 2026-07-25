package world

import "fmt"

// 世界子地圖網格。
//
// 世界是 7×7 格的子地圖拼成的，每格自己是一張 64×64 的地圖。
// **子地圖編號 = 欄×10 + 列**（欄是十位數、列是個位數），欄與列都從 1 數到 7，
// 所以合法編號是 11–77。編號小於 10 的不是世界地圖，是地城／室內。
//
// # 怎麼確認的
//
// 走到子地圖邊界時的換圖程式碼（`DEMON.INT` 0x16fec–0x17114）。四個方向
// 各一段，形狀完全對稱：
//
//	往西  X == 3   → 欄 == 1 就擋住，否則 map_id -= 10、X ← 59
//	往東  X == 60  → 欄 == 7 就擋住，否則 map_id += 10、X ← 4
//	往北  Y == 3   → 列 == 1 就擋住，否則 map_id -= 1、 Y ← 59
//	往南  Y == 60  → 列 == 7 就擋住，否則 map_id += 1、 Y ← 4
//
// 「X 方向動十位數、Y 方向動個位數」直接把兩個軸釘死，不必再靠接邊統計
// 或攻略的方位敘述去猜。擋住的條件同時給出網格範圍：欄 1–7、列 1–7。
//
// 判斷欄／列的指令是 `idiv 10`：商是欄（`0x16ffd`、`0x170e2`），
// 餘數是列（`0x1704c`、`0x17097`）。
//
// # 順帶確認的兩件事
//
//   - **隊伍欄位 `+0xa3` 是目前的子地圖編號**。上面四段都在改它，
//     而別處用 `>= 10` 當「在戶外還是在地城」的判斷（例如戰場視野要不要
//     查時辰表，見 gamedata.LightInsetAt）—— 剛好對上「10 以下是地城」。
//   - **一張子地圖走得到的範圍是 4–59**，不是 0–63。四周各三格是換圖用的
//     邊界，踩到就被送到隔壁那張圖的另一側。
const (
	// SubMapMinAxis／SubMapMaxAxis 是網格的欄／列範圍（含端點）。
	SubMapMinAxis = 1
	SubMapMaxAxis = 7

	// SubMapMinID 是最小的世界子地圖編號。小於這個值的是地城／室內，
	// 不在世界網格上。
	SubMapMinID = SubMapMinAxis*10 + SubMapMinAxis // 11

	// WalkMin／WalkMax 是子地圖內走得到的座標範圍（含端點）。
	WalkMin = 4
	WalkMax = 59

	// edgeLow／edgeHigh 是觸發換圖的兩個座標值。
	edgeLow  = 3
	edgeHigh = 60
)

// IsWorldSubMap 回報這個編號是不是世界地圖上的子地圖（而非地城／室內）。
func IsWorldSubMap(id int) bool {
	return id >= SubMapMinID && SubMapColumn(id) <= SubMapMaxAxis &&
		SubMapRow(id) >= SubMapMinAxis && SubMapRow(id) <= SubMapMaxAxis
}

// SubMapColumn 取子地圖編號的欄（十位數，對應世界的 X 軸）。
func SubMapColumn(id int) int { return id / 10 }

// SubMapRow 取子地圖編號的列（個位數，對應世界的 Y 軸）。
func SubMapRow(id int) int { return id % 10 }

// SubMapID 由欄、列組出子地圖編號。
func SubMapID(col, row int) (int, error) {
	if col < SubMapMinAxis || col > SubMapMaxAxis ||
		row < SubMapMinAxis || row > SubMapMaxAxis {
		return 0, fmt.Errorf("world: 欄列 (%d,%d) 超出 %d–%d",
			col, row, SubMapMinAxis, SubMapMaxAxis)
	}
	return col*10 + row, nil
}

// CrossResult 是一次邊界換圖的結果。
type CrossResult struct {
	// Crossed 為真代表真的換了一張圖。
	Crossed bool
	// Blocked 為真代表走到世界邊緣，過不去 —— 座標與編號都不變。
	Blocked bool
	// MapID、X、Y 是換圖之後的位置（沒換就是原值）。
	MapID, X, Y int
}

// CrossEdge 處理「走到子地圖邊界」。
//
// 座標落在 edgeLow／edgeHigh 時換到隔壁那張圖並把座標送到另一側；
// 已經在世界邊緣就回 Blocked。其餘情況原樣回傳、Crossed 為 false。
//
// 只處理世界子地圖 —— 地城（編號 < 11）沒有這套換圖規則，直接原樣回傳。
func CrossEdge(mapID, x, y int) CrossResult {
	res := CrossResult{MapID: mapID, X: x, Y: y}
	if !IsWorldSubMap(mapID) {
		return res
	}
	col, row := SubMapColumn(mapID), SubMapRow(mapID)

	switch {
	case x == edgeLow: // 往西
		if col == SubMapMinAxis {
			res.Blocked = true
			return res
		}
		res.MapID, res.X = mapID-10, WalkMax

	case x == edgeHigh: // 往東
		if col == SubMapMaxAxis {
			res.Blocked = true
			return res
		}
		res.MapID, res.X = mapID+10, WalkMin

	case y == edgeLow: // 往北
		if row == SubMapMinAxis {
			res.Blocked = true
			return res
		}
		res.MapID, res.Y = mapID-1, WalkMax

	case y == edgeHigh: // 往南
		if row == SubMapMaxAxis {
			res.Blocked = true
			return res
		}
		res.MapID, res.Y = mapID+1, WalkMin

	default:
		return res
	}

	res.Crossed = true
	return res
}
