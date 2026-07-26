package main

import (
	"fmt"

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
		out = append(out, textlayout.WrapText(
			fmt.Sprintf("%s %s", c.Name, a.winText.Fate(int(c.Class))),
			endingColumns)...)
	}
	return out
}

// endingLinesPerPage 是名單一頁的行數，留兩行給提示。
const endingLinesPerPage = 20

// drawEndingPage 畫 WIN.TXT 的一頁。超出範圍就退回祝賀詞。
func (a *app) drawEndingPage(line func(string), page int) {
	lines := a.winText.Page(page)
	if lines == nil {
		a.drawEndingFallback(line)
		return
	}
	for _, l := range lines {
		line(l)
	}
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
