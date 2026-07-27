package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 戰鬥中的 `?` 檢視（動作 case 10，`138d:24e0` → `17c5:1056` ＝ `0xc8a6`）
//
// **這一格是 `docs/manual-coverage.md` §7 列的第二個「解完沒接」**：
// `ActionExamine` 的成本（0 移動點）早就定義了，卻**零呼叫端、沒綁鍵** ——
// 連帶讓戰術（技能 7）與怪物學識（技能 25）兩個技能完全沒有效果。
//
// 版面逐項對過原版的格式字串：
//
//	c958  名稱                        遠指標表 ds:0x5194[單位]
//	c97b  狀態（非正常才印）           ds:0x15ec[狀態]；束縛再加 "%s>%d" 帶等級
//	ca4a  "%s:    %3d"  力量           unit+0x4ec0
//	ca88  "%s:    %3d"  技巧           unit+0x4ec6
//	cac6  "%s:    %3d"  速度           unit+0x4eb8
//	cb04  "Armor: %3d pts."            unit+0x4ebe
//	cb3a  武器名稱                     unit+0x4ebc → 106a:000a 查名
//	cb7f  "HP: %3d  SP:%3d"            **只有槽位 > 11**（召喚／幻術）
//	cbd9  牠要打誰的名稱               **只有隊伍裡有人會戰術**
//
// 兩道技能閘門是從同一段的兩個負偏移認出來的：`es:[bx+si-0x63b]` 與
// `es:[bx+si-0x64d]` 差 `0x12` ＝ 18，而技能旗標裡怪物學識(25)與戰術(7)
// 的間距正好也是 18（`0xe1 − 0xcf`）。兩個閘門因此定案。

// 檢視面板要用到的兩個技能（`docs/re/21` §1）。
const (
	// SkillTactics 戰術：顯示每隻怪物打算攻擊誰。
	SkillTactics gamedata.SkillID = 7
	// SkillMonsterLore 怪物學識：沒有它就看不到怪物的屬性。
	SkillMonsterLore gamedata.SkillID = 25
)

// ExamineCard 是檢視面板上要顯示的內容。
//
// 這是**資料不是文字** —— 標籤與翻譯留給顯示層，
// 與死亡畫面、陷阱訊息同一個分工。
type ExamineCard struct {
	Name string
	// Slot 是單位槽位（怪物 0–6、隊伍 7–11、召喚 12–14）。
	Slot int

	// Status 非正常時才顯示。
	Status UnitStatus
	// BindLevel 是束縛的剩餘回合。原版把它接在狀態後面印成 `狀態>等級`，
	// **只在狀態值介於 1 與 5 之間**（也就是三個束縛等級）才印。
	BindLevel int
	// ShowBindLevel 依原版的 `1 < 狀態 < 5` 判定。
	ShowBindLevel bool

	// Stats 為 false 時整組屬性都不顯示 —— 看怪物而隊伍裡沒人會怪物學識。
	Stats                  bool
	Strength, Skill, Speed int
	Armor                  int
	// WeaponIndex 帶符號（負數＝附毒，見 Unit.WeaponIndex）。
	// **名稱留給顯示層查** —— 武器名表在 `StringPool.WeaponTypeNames()`，
	// 規則層拿不到它，硬塞一份進來就是第二份實作。
	WeaponIndex int

	// ShowHPSP 只在召喚／幻術生物上為真（原版 `槽位 > 11`）。
	ShowHPSP  bool
	HP, MaxSP int
	SP        int

	// TargetName 是「牠打算攻擊誰」。空字串代表不顯示
	// —— 沒有人會戰術，或這個單位沒有記住目標。
	TargetName string
}

// PartyHasSkill 回報隊伍裡有沒有人會某項技能（死人不算）。
//
// **與 `LookForTraps` 的門檻不同**：那裡照原版要求 `狀態 <= 1`
// （被束縛就不能拆陷阱），檢視面板只排除死亡。不要合成一支
// 「通用的隊伍技能查詢」—— 兩邊的條件是原版分別寫死的。
func PartyHasSkill(party []Character, s gamedata.SkillID) bool {
	for i := range party {
		if party[i].Status != scenario.StatusDead && party[i].HasSkill(s) {
			return true
		}
	}
	return false
}

// ExamineUnit 組出一個單位的檢視卡。
//
// **參數是槽位不是索引。** 走 `b.Unit(slot)` 查，不自己切陣列 ——
// `AITargetSlot` 存的也是槽位，兩邊用同一個查法才不會錯位
// （戰鬥傷害寫回那次就是槽位／索引混用踩出來的）。
// party 用來查兩項技能；nil 代表兩個閘門都關著。
func ExamineUnit(b *Battle, slot int, party []Character) ExamineCard {
	u := b.Unit(slot)
	if u == nil {
		return ExamineCard{}
	}

	card := ExamineCard{
		Name:      u.Name,
		Slot:      u.Slot,
		Status:    u.Status,
		BindLevel: u.BindRounds,
		// 原版的條件是 `1 < 狀態 < 5` —— 三個束縛等級，中毒與死亡都不帶數字。
		ShowBindLevel: u.Status > StatusPoison && u.Status < StatusDead,
	}

	// 怪物的屬性要有人會怪物學識才看得到。隊伍成員與自己召喚的生物一律看得到。
	card.Stats = u.IsPlayer || PartyHasSkill(party, SkillMonsterLore)
	if card.Stats {
		card.Strength, card.Skill, card.Speed = u.Strength, u.Skill, u.Speed
		card.Armor = u.Armor
		card.WeaponIndex = u.WeaponIndex
	}

	// 召喚／幻術生物多一行 HP／SP（原版 `槽位 > 11`）。
	if u.Slot >= SummonSlotStart {
		card.ShowHPSP = true
		card.HP, card.SP, card.MaxSP = u.HP, u.CurrentSP, u.MaxSP
	}

	// 戰術：顯示牠記住的攻擊目標。第一回合之前沒有目標（`noAITarget`），
	// 那時候這一行就是空的 —— 原版也一樣，欄位還沒填。
	if PartyHasSkill(party, SkillTactics) {
		if t := b.Unit(u.AITargetSlot); t != nil {
			card.TargetName = t.Name
		}
	}
	return card
}

// ExamineOrder 是 `?` 面板上 Continue／Back 走訪的順序：**槽位由小到大**，
// 跳過空槽。原版的選單就是 `Continue`／`Back`／`Quit` 三個鍵（C／B／Q，
// 字串在 `ds:0x0adb`／`0x0ae4`／`0x0ae9`），不是手冊寫的 `←`／`→` ——
// 那是 Apple II 版的說法（`docs/manual/part-3`），DOS 版換成了選單。
func ExamineOrder(b *Battle) []int {
	var out []int
	for slot := 0; slot < CombatSlots; slot++ {
		if u := b.Unit(slot); u != nil && u.Status != StatusDead {
			out = append(out, slot)
		}
	}
	return out
}
