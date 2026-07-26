package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 全隊死亡（移動分派表的動作 `0x18` ＝ `25be:000c`，見 `docs/re/58` §4）。
//
// 原版那支很短：印四行字，回傳 **−1**。11 個呼叫端散在戰鬥結算、
// 飢餓／中毒推進、事件處理那幾條路上 —— 也就是**任何一條會讓角色掉血的路
// 都要檢查一次**，不是只有戰鬥。
//
// 這一點值得記：引擎原本只在戰鬥結算裡印「隊伍全滅」的紀錄行，
// 然後就回到世界地圖 —— 一支全員陣亡的隊伍照樣可以走路、紮營、進城。
// 而畫面上沒有任何異常。

// PartyDeathLines 是原版那四行（`ds:0x274e`／`0x275c`／`0x276e`／`0x2780`）。
//
// 原版分兩段印：前兩行在第 20 列、後兩行在第 23 列
//（`ds:0x4c7c = 0x14` 與 `0x17`），中間等一次輸入。
// 中譯不照抄那個分段 —— 那是 80×25 文字模式的排版，這裡的畫面不一樣。
var PartyDeathLines = []string{
	"A cold breeze",
	"chills the air...",
	"...all characters",
	"have died.",
}

// PartyWiped 回報隊伍是不是全員陣亡。
//
// 判活的條件與 `Unit.Alive()` 一致：**HP > 0 且狀態不是死亡**。
// 只看 HP 會漏掉「HP 還有但被判死」的狀態（`scenario.StatusDead`）。
//
// **只看隊伍成員。** 召喚物與幻術生物屬玩家陣營但不是隊伍成員
// （`Battle.Outcome` 已經是這個規則，見 `battle.go` §Outcome）——
// 只剩召喚物存活時隊伍仍算全滅。這裡看的是 `[]Character`，
// 天生就不含召喚物，所以規則自動一致。
//
// 空隊伍回 false：那是**還沒建角**的狀態（`docs/re/88`：新遊戲人數從 0 起算），
// 不是死光了。回 true 會讓新遊戲一開場就宣告全隊死亡。
func PartyWiped(party []Character) bool {
	if len(party) == 0 {
		return false
	}
	for i := range party {
		if party[i].CurrentHP > 0 && party[i].Status != scenario.StatusDead {
			return false
		}
	}
	return true
}

// 為什麼 `PartyWiped` 要跟 `WriteBackParty` 一起看：
//
// 戰鬥的傷害只寫在 `Unit` 上。沒有 `WriteBackParty` 的話，
// 打輸了每個角色還是滿血，`PartyWiped` 永遠回 false ——
// **死亡畫面接得再對也不會出現**。兩者缺一都等於沒做。
