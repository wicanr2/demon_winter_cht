package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 夢境畫面（原版 `FUN_1000_0339(n)` → 顯示 `T.TXT` 第 n−1 頁，見 `docs/re/82`）。
//
// 原版是整頁蓋掉畫面、印完等一個鍵。這裡照做 ——
// 這幾段是主線最重要的敘事，不該塞進走路畫面下方那個小訊息框。

// openDream 播第 page 場夢（0／1／2）。
func (a *app) openDream(page int) {
	if a.dreamText == nil || a.dreamText.Page(page) == nil {
		// 讀不到就別擋著 —— 劇情狀態已經推進過了，畫面缺一段總比卡住好。
		return
	}
	a.dreamPage = page
}

func (a *app) updateDream() error {
	if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
		a.dreamPage = -1
	}
	return nil
}

func (a *app) drawDream(dst *ebiten.Image) {
	y := ui.LineHeight
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX*2, y)
		y += ui.LineHeight
	}
	// 原版斷行是照它的 40 欄畫面，這裡是 37 欄 —— 要再斷一次，
	// 不然字尾會被切掉（見 storyLines）。
	for _, l := range storyLines(a.dreamText.Page(a.dreamPage)) {
		line(l)
	}
	line("　")
	line("（按任意鍵）")
}
