package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// NewMixedTextBox 建立一個中英混排、依畫布寬度斷行的文字視窗。
func NewMixedTextBox(s string) *TextBox {
	return textlayout.NewMixedTextBox(s, textlayout.MixedColumns)
}

// DrawMixedTextBox 把目前這一頁畫在 (x, y)，並在下方畫出翻頁提示。
//
// 寬高一律取自 layout —— 與 WrapMixed 的斷行寬度同源，
// 才不會出現「排版說塞得下、畫出來卻超出框」。
func DrawMixedTextBox(dst *ebiten.Image, b *TextBox, f *MixedFont, x, y int, frame color.Color) {
	if !b.Active() {
		return
	}

	w := textlayout.MixedColumns + 2*layout.BoxPadX
	dst.SubImage(rect(x, y, w, layout.BoxHeight)).(*ebiten.Image).Fill(color.RGBA{0, 0, 0, 0xff})
	StrokeRect(dst, x, y, w, layout.BoxHeight, frame)

	f.DrawProseLines(dst, b.Lines(), x+layout.BoxPadX, y+layout.BoxPadY)

	hint := "－ 續 －"
	if !b.HasMore() {
		hint = "－ 完 －"
	}
	f.Draw(dst, hint, x+layout.BoxPadX,
		y+layout.BoxPadY+textlayout.PageLines*ProseLineHeight)
}
