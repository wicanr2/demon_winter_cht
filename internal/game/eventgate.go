package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 事件格的觸發閘門（原版 `FUN_222f_0a90`，`docs/re/05` §1.5）
//
// `nSS.DAT` 的類別 1 與類別 2 都是「事件文字索引」，`docs/re/77` §4 只寫到
// 這裡就停了 —— 於是引擎一直把兩者當成同一種東西：踩到就播，播完
// `MarkVisited`，然後**下次踩到再播一次**。
//
// 兩者的差別不在事件表那一側，在**觸發閘門**：
//
//	類別 1  值 != 0 → 完全不再有反應          一次性
//	類別 2  值 != 0 → 不自動播，但武裝「重讀」  按 R 可以再讀一次
//
// 也就是說 **`nSS.DAT` 的類別編號本身就是那個分類**，不需要逐筆判斷
// 哪些事件是一次性的。判準來自出貨資料：`1SS`／`2SS` 是壓片前有人玩過
// 一段的狀態（`docs/re/78` §2），裡面**類別 2 也有值 1 的記錄**
// （1SS 九筆、2SS 三筆）—— 所以「類別 2 會被標記成已造訪」不是推論，
// 而且那條「值 != 0」的分支確實會走到。
//
// `R) Read descr.` 是主選單第 13 項（`docs/re/96` 的 15 格表），
// 對應移動迴圈 switch case 6（`222f:1282`）。

// EventGateAction 是閘門的三種結果，對應原版的三個回傳值：
// `0xfffd`（無事件）、`0x12`（一般觸發）、`0xffff`（已就地傳送）。
type EventGateAction int

const (
	EventNone EventGateAction = iota
	EventFire
	EventTeleport
)

// 重讀倒數的三個值（原版 `ds:0x4e2e` 在這條路上的用法）。
//
// ⚠ `0x4e2e` 是**跨段共用的暫存字**，全檔幾十處讀寫，在戰鬥碼裡是
// 效果類型的分派索引。同 `0x4e32` 那個坑（`docs/re/97` §5.2）——
// 位址一樣不代表語意一樣，這裡只描述移動／事件那條路。
const (
	// RereadIdle：沒有東西可以重讀。
	RereadIdle = 0
	// RereadArmed：腳下有一格看過的類別 2 事件，按 R 可以重讀。
	RereadArmed = 1
	// RereadRequested：玩家按了 R，這一次查表要放行。
	RereadRequested = 2
)

// EventGate 判斷踩在一格特殊格上會發生什麼，並回傳更新後的重讀倒數。
//
// 分支順序照原版，不要重排 —— `counter == 2` 那條在最前面，
// 所以按 R 之後**連類別 2 的「值 != 0」都會被跳過**，那正是重讀的機制。
func EventGate(class, value, counter int) (EventGateAction, int) {
	// 類別 0 ＝ 這一格沒事（一次性事件用掉自己之後長這樣）。
	if class == 0 {
		return EventNone, counter
	}
	// 類別 1 看過就結束了 —— 連 R 都讀不回來，因為倒數從沒被武裝過。
	if class == scenario.SpecialClassEventA && value != 0 {
		return EventNone, counter
	}
	switch {
	case counter == RereadRequested:
		return EventFire, counter - 1
	case class == scenario.SpecialClassEventB && value != 0:
		return EventNone, RereadArmed
	case class == scenario.SpecialClassTeleport:
		return EventTeleport, counter
	default:
		return EventFire, counter
	}
}

// TickReread 是移動迴圈每一步的倒數遞減（原版 `222f:0c0c`：
// `0x4e2e == 1` 時減一）。
//
// **只減 1 那一格**：2 是玩家剛按下 R 的請求，要留給接下來的查表。
func TickReread(counter int) int {
	if counter == RereadArmed {
		return RereadIdle
	}
	return counter
}

// RequestReread 處理 `R) Read descr.`（原版 switch case 6，`222f:1282`）。
//
// 沒武裝就什麼都不做 —— 原版直接回 `0xfffd`。回傳有沒有真的要重查。
func RequestReread(counter int) (int, bool) {
	if counter != RereadArmed {
		return counter, false
	}
	return RereadRequested, true
}
