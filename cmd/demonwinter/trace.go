package main

import (
	"fmt"
	"os"
	"strings"
)

// 試玩軌跡（`-trace`）。
//
// **這是為了 A4「全程試玩」而做的。** 在此之前所有驗收都是單點截圖：
// 開一個畫面、拍一張、確認長得對。那證明得了「這個畫面畫得出來」，
// 證明不了「從新遊戲一路走過來，這些東西串得起來」。
//
// 截圖驗收在長路徑上會騙人：xdotool 偶爾漏鍵，隊伍少走一步，
// 最後那張圖照樣是張合理的畫面 —— 只是走到了別的地方。
// 肉眼看不出差別，因為沒有對照。
//
// 所以要有軌跡：**每一次可觀察狀態改變就寫一行**，事後可以逐行核對
// 「走了幾步、到了哪、觸發了什麼」。它不改變任何規則，只是把
// 本來只存在於那一瞬間的狀態攤成時間軸。

// tracer 把狀態變化寫成一行一筆。
type tracer struct {
	f *os.File
	// last 是上一次寫出去的狀態快照。相同就不寫 ——
	// 遊戲每秒 60 幀，逐幀記錄會產生幾萬行雜訊，真正的事件淹沒在裡面。
	last string
	// n 是已寫出的行數，給 shot 命名用。
	n int
}

// newTracer 開一份軌跡檔。path 為空時回 nil（不啟用）。
func newTracer(path string) *tracer {
	if path == "" {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		// 軌跡是驗收工具，開不起來不該讓遊戲跑不動。
		fmt.Fprintf(os.Stderr, "軌跡檔開不起來（%v），這一次不記錄\n", err)
		return nil
	}
	return &tracer{f: f}
}

// note 記一件明確發生的事（不做去重）。
func (t *tracer) note(format string, args ...any) {
	if t == nil {
		return
	}
	t.n++
	fmt.Fprintf(t.f, "%4d  %s\n", t.n, fmt.Sprintf(format, args...))
}

// state 記一次狀態快照，與上一筆相同就跳過。
func (t *tracer) state(s string) {
	if t == nil || s == t.last {
		return
	}
	t.last = s
	t.n++
	fmt.Fprintf(t.f, "%4d  %s\n", t.n, s)
}

func (t *tracer) close() {
	if t == nil {
		return
	}
	_ = t.f.Close()
}

// traceState 組出目前這一幀的可觀察狀態。
//
// 只放**玩家看得到**的東西。內部旗標不放 —— 軌跡是拿來對照
// 「玩家會經歷什麼」的，塞進實作細節就變成另一種內部訊號，
// 而內部訊號正是 A4 要繞開的東西。
func (a *app) traceState() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-10s (%2d,%2d) 地圖%d", a.screenName(), a.party.X(), a.party.Y(), a.mapID)
	if a.message != "" {
		fmt.Fprintf(&b, "  訊息=%q", a.message)
	}
	// **要用 `Active()` 不是 `!= nil`。** 文字框讀完之後物件還留著，
	// 只是不再吃輸入 —— 用 `!= nil` 會讓軌跡一路標「文字框」，
	// 而驅動腳本看到它就一直送 Return 想關掉一個早就關了的東西。
	// 第一次跑全程試玩就是被這個假訊號帶偏，以為卡在事件文字上，
	// 其實真正卡住的地方在後面（隨機遭遇的戰鬥）。
	if a.box.Active() {
		fmt.Fprintf(&b, "  文字框")
	}
	if a.battle != nil {
		fmt.Fprintf(&b, "  第%d回合", a.battle.Round())
	}
	// **隊伍血量。** 加這一欄的理由：戰鬥的傷害一直沒寫回隊伍，
	// 而軌跡只記位置與畫面，所以那個 bug 在四輪試玩裡完全看不見
	// （戰鬥畫面上的血量是 `Unit` 的，是對的）。
	// 血量進軌跡之後，「打完一場仗血沒少」一眼就看得出來。
	b.WriteString("  血")
	for i := range a.members {
		if i > 0 {
			b.WriteByte('/')
		}
		fmt.Fprintf(&b, "%d", a.members[i].CurrentHP)
	}
	return b.String()
}

// screenName 是目前最上層的畫面名。順序要與 Update 的分派一致，
// 否則軌跡會指到一個其實沒在接收輸入的畫面。
func (a *app) screenName() string {
	switch {
	case a.quitting:
		return "離開確認"
	// **死亡要排在最前面**，與 `Update` 的分派順序一致。
	// 漏了它的話全隊死亡之後軌跡還寫「野外」—— 而那正是這一輪
	// 要抓的狀態。軌跡與分派順序不一致，軌跡就在說謊。
	case a.death != nil:
		return "死亡"
	case a.won:
		return "結局"
	case a.title != nil:
		return "標題"
	case a.dreamPage >= 0:
		return "夢境"
	case a.eregore != nil:
		return "艾瑞戈爾"
	case a.riddle != nil:
		return "密語"
	case a.manualUI != nil:
		return "手札"
	case a.runeBox != nil:
		return "符文"
	case a.create != nil:
		return "建角"
	case a.town != nil:
		return "城鎮"
	case a.battle != nil:
		return "戰鬥"
	case a.camp != nil:
		return "紮營"
	case a.merchant != nil:
		return "商隊"
	case a.pool != nil:
		return "水池"
	case a.dungeon != nil:
		return "地城道具"
	case a.plotGift != nil:
		return "劇情道具"
	}
	return "野外"
}
