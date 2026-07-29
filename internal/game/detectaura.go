package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 察覺靈光（精靈的天生能力，原版 `0x0e454`–`0x0e498`）
//
// 手冊講得很具體（`docs/manual/part-3.md`）：
//
//	如果隊伍中有具備察覺靈光（Detect Aura）的精靈，物品下方有時會顯示
//	「aura detected」（偵測到靈光）字樣，代表該物品可能是魔法物品；
//	**不過察覺靈光並非每次都能生效**。
//
// 攻略補上用途：「讓隊伍能分辨哪些物品有附魔」「只是讓你賣魔法戰利品
// 能多賣一點錢而已」（`docs/walkthrough/part-2.md`）。
//
// 反組譯把「有時」量化了 —— 它就掛在戰鬥結算發戰利品那一段。
//
// > 手冊說的是 Apple II 版的 `Sense magic`，DOS 版介面改叫 `Detect aura`，
// > 效果相同（`docs/manual/part-1.md` 譯註）。這一項兩版一致，
// > 沒有 `docs/re/93` 那種「手冊講的是別的平台」的問題。

const (
	// auraDie 是機率的骰面（原版 `0x0e45a` 的 `Roll(9)`）。
	auraDie = 9
	// auraBonus 是門檻的底（原版 `0x0e467` 的 `add cx,4`）。
	//
	// 判定是 `Roll(9) <= 精靈人數 + 4`，所以一個精靈是 **5/9**、
	// 兩個 6/9…… 五個精靈就必中。手冊的「並非每次都能生效」是這個。
	auraBonus = 4
)

// ElvesInParty 數隊伍裡有幾個精靈。
//
// **看種族不看技能旗標**（原版 `0x0e259` 比的是 `+0xf5 == 1`）——
// 察覺靈光是天生能力，不在那 31 格技能旗標裡。
//
// **而且死人也算。** 原版那個計數在發經驗值的迴圈裡，
// `count++` 排在「戰鬥狀態 >= 2 就跳過」之前（`0x0e261` 早於 `0x0e264`）。
// 看起來像疏漏，但沒有旁證說它是 bug，照抄。
func ElvesInParty(members []Character) int {
	n := 0
	for _, c := range members {
		if c.Race == gamedata.Elf {
			n++
		}
	}
	return n
}

// HasAura 回報一件戰利品身上有沒有「靈光」可偵測。
//
// 原版四道或條件（`0x0e472`–`0x0e48c`）：
//
//	槽[+0x02] != 0     附帶法術 A 的強度
//	槽[+0x08] != 0     效果強度
//	槽[+0x0a] != 0     特效值 A
//	槽[+0x0e] != 10    附魔（存的是 +10 偏移，所以 != 0）
//
// 注意**沒有查附帶法術 B 與特效值 B**（`+0x04`／`+0x0c`）——
// 只長 B 那一組的道具偵測不到。照抄，不要補齊。
func HasAura(slot scenario.InventorySlot) bool {
	return slot.SpellAPower != 0 || slot.Power != 0 ||
		slot.EffectValueAByte != 0 || slot.Enchant != 0
}

// DetectAura 擲一次「這件戰利品要不要標上靈光」。
//
// 順序照原版：**先擲點再看道具**（`0x0e45a` 的 Roll 在四道或條件之前）。
// 所以隊伍裡沒有精靈時**連骰都不擲** —— 亂數序列會不一樣，
// 這對 `-seed` 的重跑一致性有影響。
func DetectAura(r *rng.RNG, members []Character, slot scenario.InventorySlot) bool {
	elves := ElvesInParty(members)
	if elves == 0 || r == nil {
		return false
	}
	if r.Roll(auraDie) > elves+auraBonus {
		return false
	}
	return HasAura(slot)
}
