package game

import "github.com/wicanr2/demon_winter_cht/internal/rng"

// 紮營選單的 Xorcise（`1000:1959`，見 `docs/re/41`）。
//
// **這不是戰鬥裡的「驅散不死」。** 名字很像，做的事完全不同：
// 這是**把身上被詛咒的裝備拆下來**。
//
// # 詛咒到底做了什麼
//
// 讀完這一段才知道詛咒不只是「附魔為負」：
//
//   - 它會把技能旗標設成 **2**（學過、但用不出來）
//   - 它會動到四個「含加成」屬性欄位
//
// 驅邪成功就把兩者都還原：所有值為 2 的技能旗標清成 0，四個含加成欄位
// 各自複製回天生值。**那四對欄位一一對應**（`+0xf7←+0xf3`、`+0xf8←+0xe8`、
// `+0xfb←+0xe9`、`+0xfe←+0xea`），正好是 save.go 早就標好的
// 速度／力量／技巧／法力上限四組「天生 vs 含加成」—— 這一段等於替那四對
// 欄位做了第二次獨立確認。

// 驅邪需要的兩種技能之一（與神殿的教團技能同一組）。
// SkillShaman／SkillPriesthood 定義在 services.go。

// exorciseStatusLimit 是能驅邪的狀態上限（原版 `< 2`）。
const exorciseStatusLimit = 2

// ExorciseResult 是一次驅邪的結果。
type ExorciseResult struct {
	// OK 為 true 代表真的做了（不代表成功）。
	OK bool
	// Success 為 true 代表詛咒解開、裝備拆下來了。
	Success bool
	// Reason 是沒做成的原因。
	Reason string
	// Chance 是這一次的成功率（來自道具的 `+0x0d`）。
	Chance int
	// Freed 是解開了幾項被封住的技能。
	Freed int
}

// CanExorcise 回報 caster 能不能替 target 的第 slot 格驅邪。
//
// 四道關卡，順序照原版。最後一條是這一項最特別的地方：
// **只能對「正裝備著」的東西驅邪** —— 選單會一直重問，
// 直到你挑中對方的武器格或護甲格。放在包包裡的詛咒物碰不到。
func CanExorcise(caster, target *Character, slot int) (bool, string) {
	switch {
	case caster == nil || target == nil:
		return false, "reason.member.invalid"
	case int(caster.Status) >= exorciseStatusLimit:
		return false, "reason.exorcise.unavailable"
	case !caster.HasSkill(SkillShaman) && !caster.HasSkill(SkillPriesthood):
		return false, "reason.exorcise.no_skill"
	case caster.ExorcisedToday:
		return false, "reason.exorcise.used_today"
	case slot < 0 || slot >= InventorySlots:
		return false, "reason.slot.invalid"
	case target.Inventory[slot].Empty():
		return false, "reason.slot.empty"
	case slot != target.EquippedWeapon && slot != target.EquippedArmor:
		return false, "reason.exorcise.not_equipped"
	}
	return true, ""
}

// Exorcise 驅邪。**失敗也用掉今天的機會**（旗標在擲骰之前就設起來）。
//
// 成功率直接來自道具的 `+0x0d`（`InventorySlot.ExorciseResist`）：
// `rnd(100)` 大於它就失敗，所以那個值越大越好驅。
func Exorcise(r *rng.RNG, caster, target *Character, slot int) ExorciseResult {
	if ok, why := CanExorcise(caster, target, slot); !ok {
		return ExorciseResult{Reason: why}
	}
	caster.ExorcisedToday = true

	chance := target.Inventory[slot].ExorciseResist
	res := ExorciseResult{OK: true, Chance: chance}
	if r.Roll(100) > chance {
		return res
	}
	res.Success = true

	// 被封住的技能全部解開 —— 原版是掃 31 格、把值 2 的清成 0。
	for i := range target.CursedSkills {
		if target.CursedSkills[i] {
			target.CursedSkills[i] = false
			res.Freed++
		}
	}
	// 那一件脫下來。
	target.unequipIfSlot(slot)
	return res
}

// CursedSkillList 回傳這名角色被封住的技能編號，供介面顯示。
func (c *Character) CursedSkillList() []int {
	var out []int
	for i, on := range c.CursedSkills {
		if on {
			out = append(out, i)
		}
	}
	return out
}
