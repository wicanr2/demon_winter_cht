package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 遊戲內手札（`F2`）。
//
// 原版沒有這個畫面 —— 種族、職業、神祇、地形、陷阱那些查詢用的資料
// 在 1988 年是印在紙本手冊裡的（當年的慣例，兼作防拷）。三十幾年後那本
// 手冊多半不在玩家手上，所以搬進遊戲。
//
// **這是本專案自己加的，不是還原原版。** 所以鍵位放在 `F2` ——
// 與 `F1`（建立角色）同一組「一看就不在原版裡」的位置，
// 不去佔用原版有意義的字母鍵。
//
// 內容與格式見 `internal/manual`。

// manualLinesPerPage 是一頁塞得下的行數。
// 扣掉抬頭與底部的操作提示。
const manualLinesPerPage = 18

type manualScreen struct {
	// section 是目前選到／正在讀的章節索引。
	section int
	// reading 為 true 時在讀內文，false 時在目錄。
	reading bool
	// top 是內文捲動的起始行。
	top int
}

// openManual 打開手札。
func (a *app) openManual() {
	if a.manual.Len() == 0 {
		a.message = a.tr.UI("manual.unavailable", "手札讀不到（assets/manual/）")
		return
	}
	a.manualUI = &manualScreen{}
}

func (a *app) updateManual() error {
	m := a.manualUI
	// ESC 只退一層：讀內文時回目錄，在目錄才收起手札。
	// 照 esc-cancel-f10-quit-autosave —— ESC 永遠不會結束遊戲。
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if m.reading {
			m.reading = false
			m.top = 0
		} else {
			a.manualUI = nil
		}
		return nil
	}

	if !m.reading {
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			m.section = (m.section + 1) % a.manual.Len()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			m.section = (m.section - 1 + a.manual.Len()) % a.manual.Len()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			m.reading = true
			m.top = 0
		}
		return nil
	}

	sec := a.manual.At(m.section)
	maxTop := len(sec.Lines) - manualLinesPerPage
	if maxTop < 0 {
		maxTop = 0
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		m.top++
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		m.top--
	case inpututil.IsKeyJustPressed(ebiten.KeySpace),
		inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
		m.top += manualLinesPerPage
	case inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
		m.top -= manualLinesPerPage
	}
	if m.top > maxTop {
		m.top = maxTop
	}
	if m.top < 0 {
		m.top = 0
	}
	return nil
}

func (a *app) drawManual(dst *ebiten.Image) {
	m := a.manualUI
	y := ui.LineHeight
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX*2, y)
		y += ui.LineHeight
	}

	if !m.reading {
		line(a.tr.UI("manual.title", "手札"))
		line("")
		for i, t := range a.manual.Titles() {
			mark := "　"
			if i == m.section {
				mark = "→"
			}
			line(mark + t)
		}
		line("")
		line(a.tr.UI("manual.toc.keys", "↑↓ 選擇　Enter 翻開　Esc 收起"))
		return
	}

	sec := a.manual.At(m.section)
	line(fmt.Sprintf(a.tr.UI("manual.reading.header", "手札　%s"), sec.Title))
	line("")
	for i := m.top; i < len(sec.Lines) && i < m.top+manualLinesPerPage; i++ {
		line(sec.Lines[i])
	}

	// 捲動提示畫在固定位置，不跟著內文長度跑 —— 不然短的章節提示會浮在中間。
	y = layout.CanvasHeight - ui.LineHeight*2
	if len(sec.Lines) > manualLinesPerPage {
		line(fmt.Sprintf(a.tr.UI("manual.reading.scroll", "第 %d–%d 行／共 %d　空白鍵翻頁　Esc 回目錄"),
			m.top+1, min(m.top+manualLinesPerPage, len(sec.Lines)), len(sec.Lines)))
	} else {
		line(a.tr.UI("manual.reading.back", "Esc 回目錄"))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
