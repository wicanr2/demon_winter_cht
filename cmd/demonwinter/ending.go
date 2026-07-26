package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// endingColumns 是結局名單斷行的欄寬。
//
// `WIN.TXT` 內文那幾頁原版就已經斷好行（每行約 37 字，正好是這個值）；
// 只有名單那一頁是整句，要自己斷。
//
// **單位是字格不是像素，而且這個介面把 ASCII 也畫成全形寬。**
// 第一版按「半形 8px」算成 74，結果每一行都被右緣裁掉一半 ——
// 而裁掉的是句子中段，畫面上看起來像資料壞了，不像排版問題。
// 從 layout 常數推出來就不會再算錯。
const endingColumns = (layout.CanvasWidth - layout.BoxPadX*3) / textlayout.CellWidthCJK

// storyContIndent 是續行的額外縮排。
const storyContIndent = "  "

// endingPixels 是結局畫面可用的像素寬（給 WrapMixed 用）。
const endingPixels = layout.CanvasWidth - layout.BoxPadX*3

// 結局序列（原版 `1000:3575` ＝ `0x07175`，見 `docs/re/04` §5.1）。
//
// 原版的流程：
//
//	19d1(0xffff,1)          載入 WIN.TXT
//	19d1(0,1) 等鍵          頁 0：火山崩塌、遠古者的聲音
//	19d1(1,1) 等鍵          頁 1：「你做得很好」
//	19d1(2,1) 等鍵          頁 2：世界仍在冬天、諸神已死
//	19d1(3,1)               頁 3：**提議讓你成為不朽**，等 1 或 2
//	if (答案 != 1) {
//	    19d1(5,1) 等鍵      頁 5：婉拒 —— 讓你回歸凡人
//	    逐一印出隊員的下場
//	    1361(0x66a)         "CONGRATULATIONS! You have won Demon's Winter."
//	    無窮迴圈            遊戲在此凍結，沒有「返回主選單」
//	}
//
// 答案 1（接受）走的是頁 4，那一頁自己就以 CONGRATULATIONS 作結。
//
// 這裡照著做，只有兩處刻意不同：
//
//   - **不無窮迴圈。** 原版卡在那裡是 1988 年的作法；這裡按鍵離開。
//   - 隊員下場那一段，原版格式字串是 `"%s %s"`（`docs/re/61` §2）——
//     第二個 `%s` 就是 `WIN.TXT` 第 6 頁那十段句子（`docs/re/82` §6）。
//     **「索引 ＝ 職業編號」是判讀**（十段對十個職業），原版取用的程式碼
//     還沒讀到，所以順序可能整體偏移。見 `scenario.Fate` 的註解。

// 結局各段的頁索引。
const (
	endingPageIntro   = 0
	endingPageOffer   = 3 // 這一頁問「接受還是婉拒」
	endingPageAccept  = 4
	endingPageDecline = 5
)

// endingScreen 是結局的播放狀態。
type endingScreen struct {
	// page 是目前顯示的頁。
	page int
	// credits 為 true 時在放隊員的下場名單。
	credits bool
	// creditTop 是名單捲動的起始行。五個人的下場加起來放不進一頁。
	creditTop int
	// done 為 true 時停在最後一句，按鍵離開。
	done bool
}

// updateEnding 推進結局序列。
func (a *app) updateEnding() error {
	if a.ending == nil {
		a.ending = &endingScreen{page: endingPageIntro}
	}
	e := a.ending

	// 提議那一頁要等 1 或 2，其餘等任意鍵。
	if e.page == endingPageOffer && !e.credits && !e.done {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.Key1):
			// 接受 —— 頁 4 自己就以 CONGRATULATIONS 作結，不放名單。
			e.page = endingPageAccept
			e.done = true
		case inpututil.IsKeyJustPressed(ebiten.Key2):
			e.page = endingPageDecline
		}
		return nil
	}

	if len(inpututil.AppendJustPressedKeys(nil)) == 0 {
		return nil
	}
	switch {
	case e.done:
		return ebiten.Termination
	case e.credits:
		// 名單一頁放不下五個人，翻完才結束。
		if e.creditTop+endingLinesPerPage < len(a.creditLines()) {
			e.creditTop += endingLinesPerPage
		} else {
			e.done = true
		}
	case e.page == endingPageDecline:
		// 婉拒之後才有名單：每個人各自回到自己的人生。
		e.credits = true
	default:
		e.page++
	}
	return nil
}

// drawEnding 畫結局。
func (a *app) drawEnding(dst *ebiten.Image) {
	e := a.ending
	y := ui.LineHeight
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX*2, y)
		y += ui.LineHeight
	}

	// 讀不到 WIN.TXT 就退回最精簡的祝賀 —— 破關了卻什麼都不顯示最糟。
	if a.winText == nil || a.winText.Page(endingPageIntro) == nil {
		a.drawEndingFallback(line)
		return
	}

	switch {
	case e != nil && e.credits:
		all := a.creditLines()
		for i := e.creditTop; i < len(all) && i < e.creditTop+endingLinesPerPage; i++ {
			line(all[i])
		}
		line("　")
		if e.creditTop+endingLinesPerPage < len(all) {
			line("（按任意鍵繼續）")
		} else {
			line("（按任意鍵）")
		}
		return
	case e != nil && e.done && e.page == endingPageAccept:
		a.drawEndingPage(line, endingPageAccept)
		return
	case e != nil && e.done:
		a.drawEndingFallback(line)
		return
	}

	page := endingPageIntro
	if e != nil {
		page = e.page
	}
	a.drawEndingPage(line, page)
	if page == endingPageOffer {
		line("　")
		line("　按 1 接受，按 2 婉拒")
	}
}

// creditLines 把五個人的下場攤成可以直接畫的行。
//
// 每個人的下場是一整句長文，**一定要斷行** —— 直接印會被畫面右緣裁掉，
// 而裁掉的是句子中段，看起來像資料壞了不像排版問題。
// 而且五個人加起來一頁也放不下，所以要能捲。兩個問題都是截圖才看出來的。
func (a *app) creditLines() []string {
	var out []string
	for i, c := range a.members {
		if i > 0 {
			out = append(out, "　")
		}
		// **中文要用 WrapMixed，不能用 WrapText。**
		// `WrapText` 按空白斷詞，中文整句沒有空白 → 被當成一個超長「單字」
		// 獨佔一行，然後被畫面右緣裁掉。英文原文剛好有空白，所以第一版
		// 只有英文看起來正常，接上中譯才炸開。
		out = append(out, textlayout.WrapMixed(
			fmt.Sprintf("%s %s", c.Name, a.fateFor(int(c.Class))),
			endingPixels)...)
	}
	return out
}

// fateFor 取某個職業的結局句子，有中譯就用中譯。
//
// **這裡不能直接叫 `winText.Fate()`** —— 那讀的是英文原檔。
// 名單那一頁在翻譯目錄裡是「一頁一條、十行對十個職業」，
// 所以要照同樣的方式切。第一版忘了接，畫面上其他頁都中文了，
// 只有名單還是英文 —— 而那正好是最後一幕。
func (a *app) fateFor(class int) string {
	if zh := a.tr.UI(storyKeyWinFates, ""); zh != "" {
		lines := strings.Split(strings.TrimRight(zh, "\n"), "\n")
		if class >= 0 && class < len(lines) {
			return lines[class]
		}
	}
	return a.winText.Fate(class)
}

// storyKeyWinFates 是結局名單那一頁的翻譯 key。
const storyKeyWinFates = "story.WIN.6"

// endingLinesPerPage 是名單一頁的行數，留兩行給提示。
const endingLinesPerPage = 20

// drawEndingPage 畫 WIN.TXT 的一頁。超出範圍就退回祝賀詞。
func (a *app) drawEndingPage(line func(string), page int) {
	lines := a.winText.Page(page)
	if lines == nil {
		a.drawEndingFallback(line)
		return
	}
	for _, l := range a.storyPage("WIN", page, lines) {
		line(l)
	}
}

// storyPage 取一頁劇情文字，有中譯就用中譯。
//
// 譯文以**整頁**為單位（`dwstrings story`）—— 原版的行是照它 40 欄畫面斷的，
// 中文的斷點本來就不一樣，逐行對譯會逼譯者遷就英文的斷點。
// 所以譯文自己帶換行，這裡照它的換行切開就好，不再套 storyLines 的重斷。
func (a *app) storyPage(file string, page int, src []string) []string {
	if zh := a.tr.UI(fmt.Sprintf("story.%s.%d", file, page), ""); zh != "" {
		return strings.Split(zh, "\n")
	}
	return storyLines(src)
}

// storyLines 把劇情文字的一頁調整成這個畫面放得下的行。
//
// **原版已經斷過行了，但它的畫面是 40 欄、這裡是 37 欄** ——
// 差那三欄，馬利馮預言裡就有四行被切掉字尾（`claw upon y`、`be hear`、
// `rise up in glor`）。切在字中間，看起來像資料壞了。
//
// 續行沿用原行的縮排：那段預言靠縮排分句，續行頂格會把層次打散。
func storyLines(src []string) []string {
	var out []string
	for _, l := range src {
		indent := l[:len(l)-len(strings.TrimLeft(l, " 　"))]
		body := l[len(indent):]
		width := endingColumns - len([]rune(indent)) - len(storyContIndent)
		if width < 8 {
			width = 8
		}
		wrapped := textlayout.WrapText(body, width)
		if len(wrapped) == 0 {
			out = append(out, l)
			continue
		}
		for i, w := range wrapped {
			// 續行多縮兩格。那段預言靠縮排分句，續行頂格會被讀成
			// 新的一句（「the」「crystal」「glory」單獨一行看起來就是）。
			if i > 0 {
				out = append(out, indent+storyContIndent+w)
				continue
			}
			out = append(out, indent+w)
		}
	}
	return out
}

// drawEndingFallback 是最後一段祝賀。
//
// 原版是寫死在 `ds:0x066a` 的英文字串，這裡用中文 ——
// 它不在 `WIN.TXT` 裡（`docs/re/61` §2），所以本來就要自己給。
func (a *app) drawEndingFallback(line func(string)) {
	line("恭喜！")
	line("　")
	line("你通關了《冬之魔》。")
	line("　")
	line("惡魔已被禁錮，漫長的冬天終於要過去了。")
	line("希望這趟旅程沒有辜負你的時間。")
	line("　")
	line("　")
	line("按任意鍵離開")
}

// loadWinText 讀結局文字。讀不到不算錯 —— 退回精簡祝賀就好。
func loadWinText(dir string) *scenario.StoryText {
	st, err := scenario.LoadStoryText(dir, scenario.StoryWin)
	if err != nil {
		return nil
	}
	return st
}

// loadStoryOrNil 讀一個分頁劇情文字檔，讀不到就回 nil。
func loadStoryOrNil(dir string, m scenario.StoryMode) *scenario.StoryText {
	st, err := scenario.LoadStoryText(dir, m)
	if err != nil {
		return nil
	}
	return st
}
