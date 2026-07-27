package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 地圖視野（`docs/re/97` §5）
//
// 原版的地圖視窗固定 9×9，但**畫出來的範圍會往內縮**：繪圖常式
// `FUN_222f_1404` 一開始就用 `ds:0x4c86` 把四邊各縮掉那麼多格：
//
//	222f:1440  Y 上界 = 起點 + 9 − [0x4c86]
//	222f:1450  Y 起點 += [0x4c86]
//	222f:1453  X 上界 = 起點 + 9 − [0x4c86]
//	222f:1463  X 起點 += [0x4c86]
//	222f:1466  繪製緩衝區起始位移 = ds:0x2208[[0x4c86]]   ; {0,10,20,30,40,50}
//
// 縮進去的格子**留黑**（緩衝區在 `222f:1425` 先整片清成 0，而 tile 0 是純黑）。
// 所以「內縮 3」看到的是中央 3×3，其餘八成畫面是黑的 —— 那不是缺圖，是原版的照明。
//
// `ds:0x2208` 那張表 `{0, 10, 20, 30, 40, 50}` 也順帶證明繪製緩衝區的
// row stride 是 9：內縮 i 要跳 i 列再跳 i 欄 ＝ `i×9 + i` ＝ `i×10`。
//
// 內縮量由兩條**互斥**的規則決定，分界是子地圖編號 10：

// ViewSpan 是地圖視窗的邊長（原版固定 9×9，內縮之後才是實際可見範圍）。
const ViewSpan = 9

// maxViewInset 是內縮的上限。內縮 4 時只剩中央一格。
const maxViewInset = 4

// daylightInset 是**戶外**的內縮量，依小時查表（原版 `ds:0x209c`，
// 檔位移 `0x27b9c`，由 `222f:07d3`／`07f8` 寫進 `ds:0x4c86`）。
//
// 原始位元組：`02 02 02 01 01 00 00 00 00 00 00 00 00 00 01 01 02 02 03 03 …`
//
//	0–2 時    內縮 2 → 5×5   深夜將盡
//	3–4 時    內縮 1 → 7×7   黎明
//	5–13 時   內縮 0 → 9×9   白天，看得最遠
//	14–15 時  內縮 1 → 7×7   午後
//	16–17 時  內縮 2 → 5×5   黃昏
//	18 時之後 內縮 3 → 3×3   入夜
//
// **戶外不看光源**（火把在白天沒用），這是原版的分工。
var daylightInset = [...]int{
	2, 2, 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3,
}

// OutdoorSubMapMin 是「這是戶外」的子地圖下界（原版 `cmp …+0xa3, 0x0a`）。
// 10 以下是地城，10 以上是世界地圖的分塊。
const OutdoorSubMapMin = 10

// ViewInset 回傳這一刻地圖視窗四邊各要縮掉幾格。
//
// 戶外看時辰、地城看光源，兩條規則互斥（原版分別由 `222f:07d3` 與
// `222f:02dc` 寫同一個 `ds:0x4c86`，各自有子地圖編號的閘門）。
func ViewInset(mapID, hour int, light byte, members []Character) int {
	if mapID >= OutdoorSubMapMin {
		if hour < 0 || hour >= len(daylightInset) {
			// 時辰超出表的範圍：原版會讀到表外，這裡照最暗處理。
			return maxViewInset - 1
		}
		return daylightInset[hour]
	}
	return dungeonInset(light, members)
}

// dungeonInset 是地城的內縮量：`4 − 光源`，而**矮人的黑暗視覺讓光源 +1**。
//
// 原版（`222f:02a9`–`02dc`）：
//
//	[0x5c64] = 4                       ; 預設
//	子地圖 >= 10 → 跳過（走戶外那條）
//	[0x5c64] = 隊伍[+0xa7]             ; 光源強度
//	if [0x4e32] != 0 && 光源 < 4 { [0x5c64]++ }
//	[0x4c86] = 4 − [0x5c64]
//
// `ds:0x4e32` 在這支函式（`FUN_222f_0003`）裡**不是面向** —— 那是它在
// 移動程式碼裡的用途。這裡 `222f:002c`／`0053` 把它當成「隊伍裡有活著的矮人」
// 的旗標（掃五個隊員，種族 `+0xf5 == 2`、戰鬥狀態 `+0x102 <= 1`）。
// 這個位址全檔有 40 個寫入點，是典型的共用暫存字，**不能拿其他函式的語意套過來**。
//
// 所以「黑暗視覺」這個矮人天生能力的實際效果就是**地城視野半徑 +1**，
// 不是另一套照明機制。
func dungeonInset(light byte, members []Character) int {
	n := int(light)
	if n < maxViewInset && partyHasLivingDwarf(members) {
		n++
	}
	inset := maxViewInset - n
	if inset < 0 {
		inset = 0
	}
	if inset > maxViewInset {
		inset = maxViewInset
	}
	return inset
}

// partyHasLivingDwarf 回報隊伍裡有沒有還能用眼睛的矮人。
//
// 門檻照原版：戰鬥狀態 `> 1` 就不算 —— 也就是**束縛與死亡都不算，中毒還算**。
func partyHasLivingDwarf(members []Character) bool {
	for _, c := range members {
		if c.Race == gamedata.Dwarf && c.Status <= scenario.StatusPoison {
			return true
		}
	}
	return false
}

// ViewVisible 回報視窗內的第 (dx, dy) 格（0 起算）這一刻看不看得見。
func ViewVisible(dx, dy, inset int) bool {
	return dx >= inset && dx < ViewSpan-inset &&
		dy >= inset && dy < ViewSpan-inset
}
