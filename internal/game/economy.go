package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 經驗值與金幣的共用封頂。儲存是 4 bytes，但數值封在 24 bit。
const ValueCap = 0x00FFFFFF

// CapValue 套用 0x00FFFFFF 封頂。
func CapValue(v int) int {
	if v > ValueCap {
		return ValueCap
	}
	if v < 0 {
		return 0
	}
	return v
}

// Economy 是一個城鎮的價格體系。
//
// 一切價格的基礎是經濟係數 E（城鎮資料表 +0x1ed，1 byte）。
// **全部整數除法。** 只有兩處不看 E：碼頭買船與市集賣價。
type Economy struct {
	// E 是城鎮經濟係數。
	E int
	// ShipBase 是城鎮表 +0x1f5，買船價 = ShipBase × 10。
	ShipBase int
}

// BuyPrice 回傳市集標價：商品基礎價 × E / 10。
//
// 商品基礎價全遊戲共用一份；城鎮只決定賣哪些商品與 E。
func (e Economy) BuyPrice(basePrice int) int { return basePrice * e.E / 10 }

// SellPrice 回傳市集售價：基礎價的一半，**不套用 E**。
//
// 買價套 E、賣價不套 —— 這是原版的不對稱設計，要照做。
func (e Economy) SellPrice(basePrice int) int { return basePrice / 2 }

// IdentifyPrice 回傳鑑定費用：E × 5，與道具無關。
func (e Economy) IdentifyPrice() int { return e.E * 5 }

// RationUnitPrice 回傳酒館糧食單價：E / 5，**下限 1**。
func (e Economy) RationUnitPrice() int {
	if p := e.E / 5; p > 1 {
		return p
	}
	return 1
}

// 糧食一次可買的數量範圍。
const (
	MinRations = 1
	MaxRations = 200
)

// ShipPrice 回傳買船價格：城鎮表 +0x1f5 × 10。**不看 E。**
func (e Economy) ShipPrice() int { return e.ShipBase * 10 }

// ShipMaxHull 是船體滿值，固定 75。
const ShipMaxHull = 75

// RepairPrice 回傳修船費用：(75 − 目前船體值) × E / 2。
// 滿血時為 0，呼叫端應顯示「你的船看起來很好」而不是收 0 元。
func (e Economy) RepairPrice(hull int) int {
	if hull >= ShipMaxHull {
		return 0
	}
	return (ShipMaxHull - hull) * e.E / 2
}

// 治療所四項服務的費率。
func (e Economy) HealRate() int      { return e.E / 5 }      // 每點傷害
func (e Economy) UnpoisonRate() int  { return e.E * 4 }      // 固定，不乘任何欄位
func (e Economy) UnbindRate() int    { return 47 * e.E / 5 } // 每級
func (e Economy) ResurrectRate() int { return e.E * 10 }     // 每級

// HealerService 是治療所依角色狀態選出的服務。
type HealerService int

const (
	// HealerNone 角色狀態正常且滿血，不需要服務。
	HealerNone HealerService = iota
	HealerHeal
	HealerUnpoison
	HealerUnbind
	HealerResurrect
)

// HealerQuote 依角色狀態決定服務項目與費用。
//
// 死亡優先於束縛、束縛優先於中毒、都沒有才看傷勢。
//
// **解束縛乘的是束縛等級（角色記錄 +0xec），不是角色等級。** 畫面標籤
// 印的是 `Unbind %3d/lvl`，那個 lvl 指的是束縛法術的等級 —— 本專案一度
// 拿角色等級去乘，與 `docs/re/19` §5.2 的 `char.field(+0xec) × 費率` 不符。
func (e Economy) HealerQuote(status UnitStatus, level, bindLevel, damage int) (HealerService, int) {
	switch {
	case status == StatusDead:
		return HealerResurrect, level * e.ResurrectRate()
	case status >= StatusBindLow:
		return HealerUnbind, bindLevel * e.UnbindRate()
	case status == StatusPoison:
		return HealerUnpoison, e.UnpoisonRate()
	case damage > 0:
		return HealerHeal, damage * e.HealRate()
	default:
		return HealerNone, 0
	}
}

// CollegeGoldCost 回傳學院學一項技能的金幣費用。
//
//	費用 = points × (5 × points + 25)
//
// points 是該技能對該職業的智慧點數成本（1–10），
// 對應費用是 30、70、120、180、250、330、420、520、630、750。
func CollegeGoldCost(points int) int { return points * (5*points + 25) }

// TempleDonation 把捐獻換成經驗值。1:1，無倍率，封頂 0x00FFFFFF。
func TempleDonation(exp, amount int) int { return CapValue(exp + amount) }

// PrayCost 回傳神殿祈禱回復好感度的費用：角色等級 × 50。
func PrayCost(level int) int { return level * 50 }

// FavorMax 是好感度的滿值。已達滿值時神殿會回「已經處於良好關係」。
const FavorMax = 20

// --- 市集議價 ---

// HaggleState 是單一商品的議價狀態。30 件商品各有一個。
//
// 初值：隊伍中有人已學「說服」→ 0，否則 1。
type HaggleState int

// hagglePermanentRefusal 是被觸怒後寫入的值，該商品從此拒賣。
const hagglePermanentRefusal = 1000

// NewHaggleStates 依隊伍是否有人會說服，決定 30 件商品的議價初值。
func NewHaggleStates(hasPersuasion bool) []HaggleState {
	init := HaggleState(1)
	if hasPersuasion {
		init = 0
	}
	out := make([]HaggleState, 30)
	for i := range out {
		out[i] = init
	}
	return out
}

// PartyHasPersuasion 回報隊伍中是否有人已學說服（技能 id 9）。
func PartyHasPersuasion(party []Character) bool {
	for i := range party {
		if party[i].HasSkill(gamedata.SkillPersuasion) {
			return true
		}
	}
	return false
}

// HaggleOutcome 是一次議價的結果。
type HaggleOutcome int

const (
	// HaggleSuccess 議價成功，折扣再加一級。
	HaggleSuccess HaggleOutcome = iota
	// HaggleUnmoved 商人不為所動，價格不變。
	//
	// **這一步是死路**：失敗後 s >= 100，下次議價 s×10 >= 1000
	// 必定大於 Roll(100) 的上限 → 一定觸怒對方並永久拒賣。
	HaggleUnmoved
	// HaggleOffended 商人被冒犯，該商品從此拒賣。
	HaggleOffended
)

// Haggle 執行一次議價，回傳結果與更新後的狀態。
func Haggle(r *rng.RNG, s HaggleState) (HaggleOutcome, HaggleState) {
	if r.Roll(100) < int(s)*10 {
		return HaggleOffended, hagglePermanentRefusal
	}
	if r.Roll(100) < int(s)*15 {
		if s < 100 {
			s += 100
		}
		return HaggleUnmoved, s
	}
	return HaggleSuccess, s + 1
}

// Refused 回報這件商品是否已被永久拒賣。
func (s HaggleState) Refused() bool { return s >= hagglePermanentRefusal }

// HagglePrice 依議價狀態算出實際售價。
//
//	n = s；n == −1 → 0；n > 99 → n −= 100
//	價格 = max(2, 標價 − 標價 × n × 6%)
//
// 每次成功議價打掉標價的 6%，下限 2 金幣。
func HagglePrice(listPrice int, s HaggleState) int {
	n := int(s)
	if n == -1 {
		n = 0
	}
	if n > 99 {
		n -= 100
	}
	price := listPrice - listPrice*n*6/100
	if price < 2 {
		return 2
	}
	return price
}
