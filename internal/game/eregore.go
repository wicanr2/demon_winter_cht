package game

// 艾瑞戈爾（`FUN_25be_1ae2` ＝ `0x1b2c2`，見 `docs/re/83` §3）。
//
// 這是主線的引爆點：大祭司艾瑞戈爾自以為掌握了力量，黑鏡裡的馬利馮
// 卻在他面前捏碎春之石，冬天由此降臨。`EREGORE.TXT` 的 10 頁全部
// 用在這一場，一頁都沒有多。
//
// 觸發點是 `nSS.DAT` 類別 5、值 14 的那一格 —— **全遊戲只有一格**，
// 在地圖 1 的 (60,1)（`docs/re/83` §2）。閘門是 `party+0xbe`：談過就不再談。

// 艾瑞戈爾對話用到的頁（`EREGORE.TXT`）。
const (
	// EregoreOpening 是開場：艾瑞戈爾、黑鏡、與吊在細線上的多面鏡。
	EregoreOpening = 0
	// EregoreFinale 是馬利馮對隊伍的詛咒，也是整場的結尾。
	// 兩條路徑共用：談成的走 7→8→9，沒談成的下次見面直接跳這一頁。
	EregoreFinale = 9
)

// EregoreMonsters 是談崩時擺出來的三隻怪（`MONSTER.DAT` 索引）。
//
// 原版在收尾那段寫死 `party+0x16..0x18 = 0x39, 0x38, 0x61`，
// 並把長度 `party+0xa6` 設成 3（`0x1b486`）。
// **七條出口裡有六條會打這一場** —— 只有一條談得下去。
var EregoreMonsters = []int{0x39, 0x38, 0x61}

// EregoreOutcome 是一步走完之後要做什麼。
type EregoreOutcome int

const (
	// EregoreAsk 還要等玩家選。
	EregoreAsk EregoreOutcome = iota
	// EregoreFight 播完就開打，並立起「見過面」旗標（`party+0x99`）。
	EregoreFight
	// EregoreDone 播完就結束，並立起「談過了」旗標（`party+0xbe`）。
	EregoreDone
)

// EregoreStep 是對話的一步。
type EregoreStep struct {
	// Pages 是這一步要依序顯示的頁，可能是空的
	//（在頁 5 回答「1」那條路徑，原版什麼都不印就收場）。
	Pages []int
	// Outcome 決定播完之後怎麼走。
	Outcome EregoreOutcome
	// Choices 是 Outcome 為 EregoreAsk 時的選項數。
	Choices int
	// Next 是下一個提問節點，用它回頭呼叫 EregoreAnswer。
	Next int
}

// 提問節點就用「發問的那一頁」當編號，這樣讀 `docs/re/83` §3 的樹狀圖時
// 不必再對一次映射表。
const (
	eregoreNodeOpening = 0
	eregoreNodeWhoIs   = 2
	eregoreNodeKnights = 5
)

// StartEregore 開場。met 是 `party+0x99`：上次談崩打過一架。
//
// 兩階段是原版的設計（`0x1b2dc`）：第一次來談崩就開打並記下 `+0x99`；
// 再來一次就跳過所有問答直接播結尾。所以**無論怎麼選都會走到馬利馮那段**，
// 差別只在要不要先打一場。
func StartEregore(met bool) EregoreStep {
	if met {
		return EregoreStep{Pages: []int{EregoreFinale}, Outcome: EregoreDone}
	}
	return EregoreStep{
		Pages:   []int{EregoreOpening},
		Outcome: EregoreAsk,
		Choices: 2,
		Next:    eregoreNodeOpening,
	}
}

// EregoreAnswer 回答節點 node 的第 choice 個選項（1-based）。
//
// 越界的 choice 會回傳原節點的提問（原版是 `do { } while (ans < 1 || ans > n)`
// 的迴圈，不合法就重問，不會往前走）。
func EregoreAnswer(node, choice int) EregoreStep {
	fight := func(pages ...int) EregoreStep {
		return EregoreStep{Pages: pages, Outcome: EregoreFight}
	}
	ask := func(page, n int) EregoreStep {
		return EregoreStep{Pages: []int{page}, Outcome: EregoreAsk, Choices: n, Next: page}
	}

	switch node {
	case eregoreNodeOpening:
		switch choice {
		case 1:
			return fight(1)
		case 2:
			return ask(eregoreNodeWhoIs, 3)
		}

	case eregoreNodeWhoIs:
		switch choice {
		case 1:
			return ask(eregoreNodeKnights, 3)
		case 2:
			return fight(3)
		case 3:
			return fight(4)
		}

	case eregoreNodeKnights:
		switch choice {
		case 1:
			// 原版這條什麼都不印，直接落到收尾（`0x1b407 je`）。
			return fight()
		case 2:
			// 唯一談得下去的一條：馬利馮現身、碎片被捏碎。
			return EregoreStep{Pages: []int{7, 8, EregoreFinale}, Outcome: EregoreDone}
		case 3:
			return fight(6)
		}
	}

	// 選項不合法 —— 照原版重問同一頁。
	return ask(node, eregoreChoiceCount(node))
}

// eregoreChoiceCount 是某個提問節點的選項數。
func eregoreChoiceCount(node int) int {
	if node == eregoreNodeOpening {
		return 2
	}
	return 3
}
