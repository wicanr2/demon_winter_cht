package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

// 紮營選單的 View land（`1000:097b`，見 `docs/re/38`）。
//
// 爬上高處看看四周 —— 打開一張可以捲動的世界地圖。這是原版唯一的
// 「看地圖」手段：平常畫面上只有腳邊那 9×9。
//
// 三道門檻，缺一不可：**得會觀地、今天還沒看過、而且人在大地圖上**。

// SkillViewLand 是觀地技能（角色記錄 `+0xe2`，技能旗標從 `+0xc8` 起算）。
//
// 26 這個編號與技能表對得上 —— `FILES.DTT` 切出來的第 26 項就是
// `View land`，與這個選項同名（`docs/re/37` §2 記過同樣的交叉印證）。
const SkillViewLand gamedata.SkillID = 26

// viewLandStatusLimit 是能觀地的狀態上限（原版 `< 2`）。
// 比鑑定嚴一格：中毒（1）還能研究道具，但爬不上高處。
const viewLandStatusLimit = 2

// viewLandMinMapID 是能觀地的最小子地圖編號。
//
// 原版寫的是 `party[+0xa3] > 10`，也就是 **>= 11** ——
// 而 `world.SubMapMinID` 正好是 11（世界格 (1,1) 的編號）。
// 兩邊是各自推出來的，分界一致：**地城與室內看不到地形**。
const viewLandMinMapID = world.SubMapMinID

// CanViewLand 回報這名角色現在能不能觀地，不能的話給原因。
//
// usedToday 是隊伍層級的旗標（存檔 trailer `+0xac`）—— 觀地**整隊一天
// 只能用一次**，不是每人一次。鑑定與打獵那兩個旗標則是每人各自一份。
func CanViewLand(c *Character, mapID int, usedToday bool) (bool, string) {
	switch {
	case c == nil:
		return false, "reason.member.invalid"
	case c.Status >= viewLandStatusLimit:
		return false, "reason.viewland.unavailable"
	case !c.HasSkill(SkillViewLand):
		return false, "reason.viewland.no_skill"
	case usedToday:
		return false, "reason.viewland.used_today"
	case mapID < viewLandMinMapID:
		return false, "reason.viewland.location"
	}
	return true, ""
}

// ViewLandOrigin 回傳觀地檢視一開始的世界座標（隊伍所在的那一格）。
//
// 原版把 `party+0xa1`／`+0xa2` 直接當起點，再用 `FUN_222f_1404(x−4, y−4)`
// 畫 9×9 的視窗 —— 與戰場攝影機是同一支常式。
func ViewLandOrigin(px, py int) (x, y int) { return px, py }

// ViewLandStep 把觀地檢視的游標移動一格並夾在地圖之內。
func ViewLandStep(x, y int, f Facing) (int, int) {
	dx, dy := f.Delta()
	return clampMap(x + dx), clampMap(y + dy)
}

func clampMap(v int) int {
	if v < 0 {
		return 0
	}
	if v >= MapWidth {
		return MapWidth - 1
	}
	return v
}
