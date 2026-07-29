package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 紮營選單的 Identify（`1000:1e46`，見 `docs/re/37`）。
//
// 掉寶生成出來的東西是**未鑑定**的：效果與附魔都看不到。要看清楚就得
// 在營地花一天研究它 —— 而且不同種類的東西要不同的學識技能。
//
// # 三種學識，各管一類
//
// 這是本專案少見的「反組譯與資料表互相印證」的例子：程式碼查的是角色記錄
// `+0xde`／`+0xdf`／`+0xe0` 三個 byte，技能旗標陣列從 `+0xc8` 起算，
// 所以是技能 22／23／24 —— 而技能表那三格正好是
// **Weapon lore／Potion lore／Item lore**。兩邊各自解出來，對得上。

// 三種學識技能。
const (
	SkillWeaponLore gamedata.SkillID = 22
	SkillPotionLore gamedata.SkillID = 23
	SkillItemLore   gamedata.SkillID = 24
)

// 道具型別到學識技能的分界（`1000:1eb8`–`0x1ee2`）。
const (
	// 型別 0–12 是武器與護甲。
	loreWeaponMaxType = 12
	// 型別 14、24、25 歸藥劑學識；13 與 15–23 歸物品學識。
	lorePotionTypeA = 14
	lorePotionTypeB = 24
	lorePotionTypeC = 25
	loreItemLowType = 13
)

// LoreSkillFor 回傳鑑定這一型道具需要哪一種學識。
//
// 三個範圍剛好把 0–25 切乾淨：
//
//	0–12   武器學識
//	13     物品學識
//	14     藥劑學識
//	15–23  物品學識
//	24–25  藥劑學識
//
// 原版是兩段獨立的條件式（各自可以把「不會」旗標設起來），範圍剛好不重疊。
func LoreSkillFor(itemType byte) gamedata.SkillID {
	switch {
	case itemType <= loreWeaponMaxType:
		return SkillWeaponLore
	case itemType == lorePotionTypeA, itemType == lorePotionTypeB, itemType == lorePotionTypeC:
		return SkillPotionLore
	default:
		return SkillItemLore
	}
}

// identifyChance 的兩個係數：成功率 = 智力 × 9 ÷ 2（整數除法）。
//
// 智力 22 就到 99%，一般起始角色（智力 10 上下）大約四成五。
const (
	identifyChanceMul = 9
	identifyChanceDiv = 2
)

// IdentifyChance 回傳這名角色的鑑定成功率（百分比）。
func IdentifyChance(intellect int) int {
	return intellect * identifyChanceMul / identifyChanceDiv
}

// identifyStatusLimit 是能鑑定的狀態上限（原版 `< 3`）。
// 正常 0、中毒 1、還有一個 2 —— 3 以上（束縛、死亡）就不能研究東西了。
const identifyStatusLimit = 3

// IdentifyResult 是一次鑑定的結果。
type IdentifyResult struct {
	// OK 為 true 代表這一次真的動手研究了（不代表成功）。
	OK bool
	// Success 為 true 代表鑑定成功，道具已標記為已鑑定。
	Success bool
	// Reason 是沒動手的原因。
	Reason string
	// Chance 是這一次的成功率（百分比），供介面顯示。
	Chance int
}

// CanIdentify 回報這名角色現在能不能鑑定第 slot 格，不能的話給原因。
//
// 四道關卡，順序照原版：狀態 → 本日是否已研究過 → 那一格能不能鑑定
// → 有沒有對應的學識。
func CanIdentify(c *Character, slot int) (bool, string) {
	if c.Status >= scenario.CombatStatus(identifyStatusLimit) {
		return false, "reason.identify.unavailable"
	}
	if c.IdentifiedToday {
		return false, "reason.identify.used_today"
	}
	if slot < 0 || slot >= InventorySlots {
		return false, "reason.slot.invalid"
	}
	it := c.Inventory[slot]
	switch {
	case it.Empty():
		return false, "reason.slot.empty"
	case it.Identified:
		return false, "reason.identify.already"
	case !c.HasSkill(LoreSkillFor(it.Type)):
		return false, "reason.identify.unsupported"
	}
	return true, ""
}

// Identify 花掉今天的研究機會，試著鑑定第 slot 格。
//
// **失敗也算用掉一天**（原版在擲骰之前就把 `+0xed` 設成 1）。
// 那個旗標由睡覺清掉（`2aed:0513` 同一個迴圈，與打獵的 `+0xef` 並排）——
// 兩段程式合起來就是「每天一次」。
func Identify(r *rng.RNG, c *Character, slot int) IdentifyResult {
	if ok, why := CanIdentify(c, slot); !ok {
		return IdentifyResult{Reason: why}
	}
	c.IdentifiedToday = true

	chance := IdentifyChance(c.Traits[gamedata.Intellect])
	res := IdentifyResult{OK: true, Chance: chance}
	if r.Roll(100) < chance {
		c.Inventory[slot].Identified = true
		res.Success = true
	}
	return res
}

// IdentifiableSlots 列出這名角色身上還沒鑑定、而且他看得懂的那幾格。
//
// 介面用它把「選了也沒用」的格子先標出來 —— 原版是選下去才告訴你，
// 但原版一天只能試一次，讓人白白浪費一天不是體貼的作法。
func (c *Character) IdentifiableSlots() []int {
	var out []int
	for i := 0; i < InventorySlots; i++ {
		it := c.Inventory[i]
		if it.Empty() || it.Identified {
			continue
		}
		if !c.HasSkill(LoreSkillFor(it.Type)) {
			continue
		}
		out = append(out, i)
	}
	return out
}
