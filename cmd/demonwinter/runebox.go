package main

import (
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 符文密語的顯示（`docs/re/72`、`docs/re/02` §2.4）。
//
// 原版把 `%` 開頭的事件文字用 `CYPHER.SHP` 的 27 個符文字形畫出來，
// 9 欄一列（`FUN_25be_18fa`：`local_6%9+1, local_6/9+1`）。
// 玩家看到的是天書 —— **那是設計意圖**，不是缺陷：
// 圖書室會給一組已知答案（`YMROS IS MINE`），玩家由此推出符文↔字母的
// 對照，之後各處的密語才解讀得出來（攻略 `part-4` §55／§141）。
//
// 所以中文化**不翻符文**（`docs/re/72` §7 定案）——
// 翻掉它等於把謎題刪掉。中文只用來說明「這是符文密語，要自己對照」。

// loadRuneFont 讀符文字型。讀不到回 nil —— 沒有字型不該讓遊戲開不起來，
// 退化成「只顯示中文說明」比整個崩掉好。
func loadRuneFont(dataDir string) []*ebiten.Image {
	data, err := os.ReadFile(filepath.Join(dataDir, "CYPHER.SHP"))
	if err != nil {
		return nil
	}
	frames, err := gfx.DecodeCGASpriteSheet(data, 16, 16)
	if err != nil || len(frames) < scenario.RuneGlyphCount {
		return nil
	}
	out := make([]*ebiten.Image, len(frames))
	for i, f := range frames {
		out[i] = ebiten.NewImageFromImage(f)
	}
	return out
}

// runeScreen 是符文密語的畫面狀態。
type runeScreen struct {
	glyphs []int
}

// openRuneBox 開符文畫面。字型缺失時只顯示說明文字。
func (a *app) openRuneBox(text string) {
	a.runeBox = &runeScreen{glyphs: scenario.RuneGlyphs(text)}
}

func (a *app) updateRuneBox() error {
	if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
		a.runeBox = nil
	}
	return nil
}

// drawRuneBox 把符文畫成 9 欄的網格，下面附中文說明。
func (a *app) drawRuneBox(dst *ebiten.Image) {
	r := a.runeBox
	const cell = 16
	x0, y0 := layout.BoxPadX*2, layout.StatusY

	for i, g := range r.glyphs {
		if g <= 0 || a.runeFont == nil || g >= len(a.runeFont) {
			continue // 空白 glyph 與非法字元都不畫
		}
		col, row := i%scenario.RuneGridCols, i/scenario.RuneGridCols
		ui.DrawImageAt(dst, a.runeFont[g], x0+col*cell, y0+row*cell)
	}

	rows := (len(r.glyphs) + scenario.RuneGridCols - 1) / scenario.RuneGridCols
	y := y0 + rows*cell + ui.LineHeight
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}
	line(a.tr.UI("rune.header"))
	line("")
	line(a.tr.UI("rune.hint1"))
	line(a.tr.UI("rune.hint2"))
	line(a.tr.UI("rune.hint3"))
	line("")
	line(a.tr.UI("rune.dismiss"))
}
