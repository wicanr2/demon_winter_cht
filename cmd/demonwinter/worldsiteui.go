package main

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 世界地圖上的神殿與學院（`docs/re/86`）。
//
// 原版是**同一支函式、兩個入口**（`docs/re/74` §5）：城鎮設施把城鎮記錄的
// 欄位寫進 `party+0xa8` 再呼叫，走地圖則寫 `0xff` 讓函式自己用座標查表。
//
// 這裡照同樣的分工：查完表之後**造一份只有那一項設施的 `gamedata.Town`**，
// 然後開一般的城鎮畫面。所有服務邏輯（祈禱、捐獻、改宗、學技能）都是
// 現成的，一行都不必重寫 —— 而且行為與城鎮裡的那份保證一致，
// 因為它們就是同一份程式碼。
//
// **這不是「假裝進城」。** 原版的世界地圖神殿與學院本來就與城鎮的
// 共用同一支處理函式；差別只在參數從哪來。

// openWorldTemple 開世界地圖上的神殿。
func (a *app) openWorldTemple(x, y int) {
	deity := game.TempleDeityAt(x, y)
	if deity == 0 {
		// tile 是神殿但表裡沒有這一格 —— 說清楚，不要靜默。
		// （原版此時 `+0xa8` 留在 `0xff`，後續行為未追。）
		a.message = fmt.Sprintf("(%d,%d) 是神殿格，但不在 19 筆神殿表裡", x, y)
		return
	}
	name := fmt.Sprintf("%s的神殿", a.deityName(deity))
	a.openWorldSite(name, game.FacilityChurch, gamedata.TownFacilities{Church: deity})
	a.message = "進入" + name
}

// openWorldCollege 開世界地圖上的學院。
func (a *app) openWorldCollege(x, y int) {
	skill, ok := game.CollegeSkillAt(x, y)
	if !ok {
		a.message = fmt.Sprintf("(%d,%d) 是學院格，但不在 35 筆學院表裡", x, y)
		return
	}
	name := fmt.Sprintf("%s學院", a.skillName(skill))
	a.openWorldSite(name, game.FacilityCollege,
		gamedata.TownFacilities{Colleges: []int{int(skill)}})
	a.message = "進入" + name
}

// openWorldSite 用一份臨時的城鎮記錄開單一設施。
//
// `Number` 留 0 是「不是城鎮」的標記（`townName` 據此改用 Name，
// 否則索引會變 −1、抬頭一片空白）。
// `Economy` 給 1：學費與捐獻都要拿它算，給 0 會讓價格全變 0
// （**免費的學院不是還原，是 bug**）。
func (a *app) openWorldSite(name string, facility game.Facility, f gamedata.TownFacilities) {
	town := gamedata.Town{Name: name, Economy: worldSiteEconomy, Facilities: f}
	visit := game.EnterTown(town, a.members)
	a.town = &townScreen{visit: visit, facility: &facility, worldSite: true}
}

// worldSiteEconomy 是世界地圖設施的經濟係數。
//
// 城鎮的 E 來自 `TOWN<n>.DAT`；地圖上的設施沒有那份記錄。
// **原版怎麼定價還沒追**（`docs/re/86` §4 列為未解），
// 這裡先用 1 —— 學費公式 `points*(5*points+25)` 不吃 E，
// 所以學院的價格與原版一致；受影響的只有捐獻換經驗那一項。
const worldSiteEconomy = 1
