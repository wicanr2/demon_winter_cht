package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"

// 施法資格：**角色只能施放自己學過的那幾系**
//
// 手冊講得很直接（`docs/manual/part-3.md`）：
//
//	按下「C」可施展法術。**角色學會的符文與吟唱法術會顯示在畫面底部。**
//
// 而學院那一節補上為什麼要分系（`part-2.md`）：
//
//	每種符文與吟唱法術都只有一間學院教授。附身、劍擊與功夫也都各自
//	只有獨一無二的學院。
//
// 引擎原本**把 43 筆法術全部列給每個人**，所以五系符文（技能 17–21）
// 與三個吟唱（12 幻術／13 附身／14 召喚）學不學都一樣 ——
// 學院收了錢卻沒有效果。這一支就是那道閘門。
//
// > 怪物那一側早就有這道閘門了（`aispell.go` 的 `aiSchoolSkill`，
// > 原版只對「被附身的玩家角色」檢查）。**兩邊用同一條換算**，
// > 差別只在吟唱系要再細分，見下。

// spellSchoolSkillBase 讓「符文系 → 技能 id」＝ `16 + 系別`
// （系別 1–5 → 技能 17–21）。與 `aispell.go` 的 `aiSchoolSkill` 同一條。
const spellSchoolSkillBase = 16

// chantSchool 是吟唱那一系的 school_id（`docs/re/15` §「school_id 對照」）。
const chantSchool = 6

// chantSkill 是吟唱系裡「哪一筆法術要哪一個技能」。
//
// **系別分不出來**：三筆的 school 都是 6，所以只能靠法術索引。
// 索引來自 `aispell.go` 記的既有結論（召喚 24、幻術 25）＋
// 第三筆 26 的效果就是 `EffectPossession`：
//
//	24  effect 15  → 召喚（技能 14）
//	25  effect 15  → 幻術（技能 12）
//	26  effect 16  → 附身（技能 13）
var chantSkill = map[int]gamedata.SkillID{
	24: gamedata.SkillSummoning,
	25: gamedata.SkillIllusion,
	26: gamedata.SkillPossession,
}

// SpellSkillFor 回傳施放第 index 筆法術需要的技能，查不到就回 ok=false。
//
// 查不到時呼叫端應該**放行**而不是擋下 —— 表裡有幾筆佔位記錄，
// 而「認不出所以不給施法」會把一個資料問題變成玩家看得見的缺功能。
func SpellSkillFor(index int, s gamedata.Spell) (gamedata.SkillID, bool) {
	if s.School == chantSchool {
		id, ok := chantSkill[index]
		return id, ok
	}
	if s.School < 1 || s.School > AISpellSchools {
		return 0, false
	}
	return gamedata.SkillID(spellSchoolSkillBase + s.School), true
}

// CanCast 回報這名角色會不會施放第 index 筆法術。
//
// **被詛咒封住的技能施不出來**：`Character.Skills` 對旗標值 2 的技能
// 本來就是 false（`docs/re/41`），所以這裡不必另外檢查 `CursedSkills`。
//
// ⚠ `c == nil` 回 false，但呼叫端**不該拿 nil 來問**。
// 召喚／幻術生物沒有角色記錄，那條路要在呼叫端就跳過這道閘門
// （見 `battleui.go` 的 `openSpellMenu`）—— 這裡回 false 是防呆，
// 不是「沒有角色就不能施法」的規則。
func CanCast(c *Character, index int, s gamedata.Spell) bool {
	if c == nil {
		return false
	}
	skill, ok := SpellSkillFor(index, s)
	if !ok {
		return true // 認不出的系別放行，見 SpellSkillFor
	}
	return c.Skills[skill]
}
