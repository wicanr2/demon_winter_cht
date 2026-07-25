package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"

// Facility 是城鎮的一類設施。
//
// 七個字串來自 `DEMON.INT`（`Marketplace`／`Healers`／`Rest`／`Town guild`／
// `Church`／`Docks`／`Pub`），設施本身的規則見 docs/spec/08-town-economy.md。
//
// **哪座城鎮有哪些設施還沒解出來。** `TOWN*.DAT` 的 28 種類型碼與這 7 個字串
// 對不起來（見 docs/formats/town-and-map.md §1.5），要對照執行檔的分派表才能定案。
// 在那之前一律當成七種都有 —— 少列設施會讓玩家以為功能不存在，
// 多列只是進去發現沒東西買。
type Facility int

const (
	FacilityMarket Facility = iota
	FacilityHealers
	FacilityInn
	FacilityGuild
	FacilityChurch
	FacilityDocks
	FacilityPub
)

// FacilityName 回傳設施的中文名稱。
func FacilityName(f Facility) string {
	switch f {
	case FacilityMarket:
		return "市集"
	case FacilityHealers:
		return "治療所"
	case FacilityInn:
		return "旅店"
	case FacilityGuild:
		return "城鎮公會"
	case FacilityChurch:
		return "神殿"
	case FacilityDocks:
		return "碼頭"
	case FacilityPub:
		return "酒館"
	}
	return "?"
}

// AllFacilities 依原版字串順序列出七種設施。
var AllFacilities = []Facility{
	FacilityMarket, FacilityHealers, FacilityInn, FacilityGuild,
	FacilityChurch, FacilityDocks, FacilityPub,
}

// TownVisit 是一次進城，把城鎮資料、價格體系與議價狀態綁在一起。
type TownVisit struct {
	Town    gamedata.Town
	Economy Economy

	// haggle 是 30 件商品各自的議價狀態，離城即重置
	// （狀態存在城鎮駐留期間，不是永久的）。
	haggle []HaggleState
}

// EnterTown 建立一次進城。
//
// 議價初值取決於隊伍裡有沒有人學過說服，所以要在進城當下決定，
// 不能預先算好 —— 隊伍組成會變。
func EnterTown(t gamedata.Town, party []Character) *TownVisit {
	return &TownVisit{
		Town:    t,
		Economy: Economy{E: t.Economy, ShipBase: t.ShipBase},
		haggle:  NewHaggleStates(PartyHasPersuasion(party)),
	}
}

// HaggleState 取某件商品的議價狀態。
func (v *TownVisit) HaggleState(item int) HaggleState {
	if item < 0 || item >= len(v.haggle) {
		return 0
	}
	return v.haggle[item]
}

// SetHaggleState 更新某件商品的議價狀態。
func (v *TownVisit) SetHaggleState(item int, s HaggleState) {
	if item < 0 || item >= len(v.haggle) {
		return
	}
	v.haggle[item] = s
}

// Price 回傳某件商品目前的售價：標價套用經濟係數後再套議價折扣。
func (v *TownVisit) Price(item int, basePrice int) int {
	return HagglePrice(v.Economy.BuyPrice(basePrice), v.HaggleState(item))
}

// HasDocks 回報這座城鎮能不能買船。
//
// 目前用「船價基礎值非零」當判準 —— 25 座裡只有 5 座非零，
// 與「只有碼頭城鎮賣船」吻合（攻略指名的新格里昂在其中）。
// 這是本專案的推論，不是從執行檔的設施表讀出來的。
func (v *TownVisit) HasDocks() bool { return v.Town.SellsShips() }
