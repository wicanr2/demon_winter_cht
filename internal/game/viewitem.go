package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 鑑物 View Items（技能 28，原版動作 `0x10` ＝ `222f:34d2` ＝ `0x193c2`）
//
// 手冊：「鑑物 —— 可得知物品未來可能的用途。」
//
// 那句話在機器碼裡的意思很具體：它讀的是地城道具的 **`+4` 欄**
// （`0x1949b` 的 `les bx, ds:0x55c8[j]`，`0x55c8 − 0x55b8 = 4 筆` ＝ `+4`），
// 也就是「這件東西要**搭配哪一件**才有用」。所以鑑物不是鑑定價值，
// 是**解謎提示系統** —— 直接告訴你這一格的東西該配什麼。
//
// **它的前置是地城道具系統（A2）**，`docs/re/93` §2 當時寫「引擎連地城道具
// 都沒有，接上去只會得到一個空選單」。A2 做完了，所以這一支現在接得上。

// SkillViewItem 是鑑物（`docs/re/21` §1 的 id 28，角色記錄 `+0xe4`）。
const SkillViewItem gamedata.SkillID = 28

// ViewItemFailDie／ViewItemFailFace 是失敗判定：`Roll(3) == 2` 就失敗
// （`0x19443`–`0x19450`）。**不是 1/3 的成功率，是 1/3 的失敗率**，
// 而且**失敗照樣扣掉一次額度**（`inc` 在 `0x1943e`，排在擲點之前）。
const (
	ViewItemFailDie  = 3
	ViewItemFailFace = 2
)

// ViewItemResult 是一次鑑物的結果。
type ViewItemResult struct {
	// NoSkill 代表隊伍裡沒有人會鑑物。原版**什麼訊息都不印**就返回
	// （與觀室一樣，`0x19401`）。
	NoSkill bool
	// Exhausted 代表今天三次用完了（原版 `Your psychic powers are weak`）。
	Exhausted bool
	// Failed 代表擲點失敗（原版 ds:0x267a `Fails`）。
	Failed bool
	// Ready 為真時才輪到玩家選一件腳下的東西。
	Ready bool
}

// BeginViewItem 跑完鑑物的三道前置：有沒有人會、額度、擲點。
//
// **拆成兩段是因為中間要讓玩家選東西。** 原版的順序是
// 「檢查技能 → 檢查額度 → `+1` → 擲點 → 印 `View item:` → 選一件」，
// 前四步沒有玩家輸入，全部在這裡做完；選完之後的判讀走 ViewItemHint。
//
// uses 指向 trailer `+0xae`（`scenario.ViewItemUses`），與觀室的 `+0xad`
// 是**兩個獨立的計數**，睡覺時一起清 0（`ResetPsychicUses`）。
func BeginViewItem(r *rng.RNG, party []Character, uses *byte) ViewItemResult {
	if !partyHasSkill(party, SkillViewItem) {
		return ViewItemResult{NoSkill: true}
	}
	if uses == nil || *uses >= PsychicUsesPerDay {
		return ViewItemResult{Exhausted: true}
	}
	// **先扣額度再擲點**，照原版的順序：失敗也用掉一次。
	*uses++
	if r != nil && r.Roll(ViewItemFailDie) == ViewItemFailFace {
		return ViewItemResult{Failed: true}
	}
	return ViewItemResult{Ready: true}
}

// ViewItemHint 回傳第 index 件的 `+4` 欄 —— 「要搭配哪一件」。
//
// 第二個回傳值為 false 時代表 `+4` 是空的，原版印
// `You can discern nothing`（ds:0x268b）；有值時印
// `An image of %s`（ds:0x26a3）＋ `comes to you`（ds:0x26b2）。
//
// **回傳的是道具名字，不是索引**，因為 `+4` 欄本來就存名字
// （與 `U` 的比對用的是同一欄，`docs/re/95` §3.10）。
func ViewItemHint(items gamedata.DungeonItems, index int) (string, bool) {
	if index < 0 || index >= len(items) || items[index].UseWith == "" {
		return "", false
	}
	return items[index].UseWith, true
}

// partyHasSkill 回報隊伍裡有沒有活著的人會這項技能。
//
// **死掉的人不算** —— 觀室那一支也是這樣（`docs/re/93`）。
// 兩個靈視技能共用這一支，不要各自抄一份掃隊迴圈。
func partyHasSkill(party []Character, id gamedata.SkillID) bool {
	for i := range party {
		if party[i].Status != scenario.StatusDead && party[i].HasSkill(id) {
			return true
		}
	}
	return false
}
