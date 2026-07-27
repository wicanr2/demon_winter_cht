package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 地圖視野的裁切（`docs/re/97` §5）
//
// **光照等級本身在 clock.go**（`LightLevel`／`LightAtHour`／`DungeonLight`，
// 連同 `ds:0x209c` 那張時辰表）—— 那一份早就寫好了，這裡只負責
// 「等級 → 畫面上哪幾格看得見」。第一版曾經在這個檔案裡把時辰表與
// `4 − 光源` 重抄一遍，那是 one-implementation-per-rule 的典型漂移。
//
// 原版的地圖視窗固定 9×9，但繪圖常式 `FUN_222f_1404` 一開始就用
// `ds:0x4c86`（＝光照等級）把四邊各縮掉那麼多格：
//
//	222f:1425  繪製緩衝區（ds:0x514e）整片清 0        ; tile 0 是純黑
//	222f:1440  Y 上界 = [BP+8] + 9 − [0x4c86]
//	222f:1450  [BP+8] += [0x4c86]                    ; Y 起點往內移
//	222f:1453  X 上界 = [BP+6] + 9 − [0x4c86]
//	222f:1463  [BP+6] += [0x4c86]                    ; X 起點往內移
//	222f:1466  起始緩衝位移 = ds:0x2208[[0x4c86]]      ; {0, 10, 20, 30, 40, 50}
//
// 縮進去的格子**留黑**。所以等級 3 看到的是中央 3×3，其餘八成畫面是黑的
// —— 那不是缺圖，是原版的照明。
//
// `ds:0x2208` 那張 `{0,10,20,30,40,50}` 順帶證明繪製緩衝區的 row stride 是 9：
// 內縮 i 要跳 i 列再跳 i 欄 ＝ `i×9 + i` ＝ `i×10`。

// ViewSpan 是地圖視窗的邊長（原版固定 9×9，內縮之後才是實際可見範圍）。
const ViewSpan = 9

// OutdoorSubMapMin 是「這是戶外」的子地圖下界（原版 `cmp …+0xa3, 0x0a`）。
// 10 以下是地城，10 以上是世界地圖的分塊。
const OutdoorSubMapMin = 10

// ViewInset 回傳這一刻地圖視窗四邊各要縮掉幾格。
//
// 戶外看時辰、地城看光源，兩條規則互斥（原版分別由 `222f:07d3` 與
// `222f:02dc` 寫同一個 `ds:0x4c86`，各自有子地圖編號的閘門）。
// **戶外不看光源**（火把在白天沒用），**地城不看時辰**。
func ViewInset(mapID, hour int, torch byte, members []Character) int {
	if mapID >= OutdoorSubMapMin {
		return int(LightAtHour(hour))
	}
	return int(DungeonLight(int(torch), PartyHasDarkVision(members)))
}

// PartyHasDarkVision 回報隊伍裡有沒有還能用眼睛的矮人。
//
// 這就是原版 `ds:0x4e32` 在 `FUN_222f_0003` 裡的語意（`222f:002c`／`0053`）：
//
//	[0x4e32] = 0
//	for i := 0; i < 5; i++ {
//	    if 角色[+0xf5] != 2 { continue }     // 種族 2 ＝ 矮人
//	    if 角色[+0x102] > 1 { continue }     // 戰鬥狀態 > 1 不算
//	    [0x4e32] = 1
//	}
//
// 門檻照原版 `> 1`：**束縛與死亡都不算，中毒還算**。
// 效果是地城光源 +1，也就是視野半徑 +1（見 DungeonLight）。
func PartyHasDarkVision(members []Character) bool {
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
