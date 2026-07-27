package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 通用的是非題那一頁。
//
// 原版好幾個地點劇情都走同一支 `1000:b566`（印完問句等 Y/N），
// 回傳非 0 就是「是」。這裡只留一份，別在每個 case 裡各寫一次。

// confirmScreen 是一道是非題。
type confirmScreen struct {
	prompt string
	// yes／no 兩邊都可以是 nil（＝什麼都不做就關掉）。
	yes, no func()
}

// askConfirm 開一道是非題。
func (a *app) askConfirm(prompt string, yes, no func()) {
	a.confirm = &confirmScreen{prompt: prompt, yes: yes, no: no}
}

func (a *app) updateConfirm() error {
	c := a.confirm
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyY):
		a.confirm = nil
		if c.yes != nil {
			c.yes()
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyN),
		inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.confirm = nil
		if c.no != nil {
			c.no()
		}
	}
	return nil
}

func (a *app) drawConfirm(dst *ebiten.Image) {
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}
	line(a.confirm.prompt)
	line("")
	line(a.tr.UI("confirm.keys", "Y：是　N／Esc：否"))
}
