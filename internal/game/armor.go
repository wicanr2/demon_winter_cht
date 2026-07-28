package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 護甲：防護點數表與職業穿著上限
//
// 兩張表都內嵌在 `DEMON.INT` 的資料段，不在 `ITEMS.DAT` 裡。
// `docs/re/06` §「護甲最終數值來源」把前者列成「已驗證有此機制、
// 細節未解……表格內容本次未逐一核對」，並在 §545 掛成待辦；這裡補完。

// armorPoints 是護甲防護點數表（`31f0:16c5`，7 個 byte）。
//
// 索引是 **`ITEMS.DAT 型別 − 7`**（`17c5:0bb9` 的 `ADD AX,0xfff9`），
// 所以涵蓋型別 7–13：
//
//	型別  7  雙手劍   0   ← 佔位，護甲槽放不進武器
//	型別  8  布甲     1
//	型別  9  皮甲     2
//	型別 10  鎖子甲   4
//	型別 11  鱗甲     5
//	型別 12  板甲     6
//	型別 13  王冠     3   ← 有值但穿不上，見 ClassArmorMax 下方的說明
//
// **不是「型別 − 7」那條等差**。原本的實作照手冊語感寫成 1–5 的線性換算，
// 鎖子甲以上每件都少 1 點。表本身是硬證據：布甲 1、皮甲 2 之後跳到 4。
//
// 皮甲 = 2 這一格還有旁證：硬化皮膚技能加的就是 2 點，而手冊說那項技能
// 「額外提供皮甲等級的防護」（`docs/manual/part-2.md`）。兩邊自洽。
//
// > ⚠ 手冊 `part-3.md` §護甲舉例「穿板甲的戰士受到 7 點傷害，護甲會吸收
// > 5 點」——照這張表板甲是 6 不是 5。手冊只在**同平台**時壓過反組譯
// > （`CONTEXT.md` §oracle 優先序），而這是出貨的 DOS 資料，以表為準。
var armorPoints = [7]int{0, 1, 2, 4, 5, 6, 3}

// armorPointsBase 是型別換算成表索引要減掉的數（`17c5:0bb9`）。
const armorPointsBase = 7

// ArmorPoints 回傳一件護甲的防護點數，含附魔。
//
// 附魔是**直接加上去**的（`17c5:0bbf`–`0bd4`：讀 `+0x1a`＝槽位的
// `+0x0e`，與表值相加後 `ADD AX,0xfff6` 扣掉那個 +10 偏移）。
// 附魔 +2 的鎖子甲就是 6 點，跟板甲同級。
func ArmorPoints(slot scenario.InventorySlot) int {
	if slot.Empty() {
		return 0
	}
	i := int(slot.Type) - armorPointsBase
	if i < 0 || i >= len(armorPoints) {
		return 0
	}
	return armorPoints[i] + slot.Enchant
}

// armorClassMax 是各職業穿得上的最重護甲型別（`31f0:03de`，每職業一個 word，
// 索引＝角色記錄 `+0xf6` 的職業 id）。
//
// 換裝那一段先擋型別範圍（`1000:2820`：`< 8` 或 `> 12` 直接不理），
// 再查這張表；`表值 < 型別` 就印 `You're the wrong class.`（`ds:0x044a`）。
//
// 三處與手冊對得上，等於同時驗了表位址與職業排序：
//
//	巫師 8 布甲    「巫師被禁止穿著除最輕型以外的護甲」（part-1.md）
//	靈視者 10 鎖子甲「可穿著重至鎖子甲的護甲」（同上）
//	盜賊 10 鎖子甲  「不會穿著可能妨礙技能施展的重型護甲」（同上）
//
// 遊俠那格是 14 —— 超出護甲型別上界，實質等於不限制。
var armorClassMax = [gamedata.NumClasses]int{
	gamedata.Ranger:    14,
	gamedata.Paladin:   12,
	gamedata.Barbarian: 12,
	gamedata.Monk:      9,
	gamedata.Cleric:    10,
	gamedata.Thief:     10,
	gamedata.Wizard:    8,
	gamedata.Sorcerer:  8,
	gamedata.Visionary: 10,
	gamedata.Scholar:   9,
}

// ClassArmorMax 回傳某職業穿得上的最重護甲型別。
//
// 職業 id 超出範圍時回 armorLastIndex（不限制）——**認不出來就放行**，
// 免得把資料問題變成玩家看得見的缺功能。
func ClassArmorMax(c gamedata.Class) int {
	if c < 0 || int(c) >= gamedata.NumClasses {
		return armorLastIndex
	}
	return armorClassMax[c]
}

// ClassCanWear 回報這個職業穿不穿得上這件護甲。
//
// 只管職業限制，型別範圍由 CanEquipAsArmor 負責。
func ClassCanWear(c gamedata.Class, it scenario.InventorySlot) bool {
	return int(it.Type) <= ClassArmorMax(c)
}
