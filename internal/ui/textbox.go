package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 轉出去，讓呼叫端不必同時 import 兩個套件。
const (
	TextBoxColumns   = textlayout.Columns
	TextBoxPageLines = textlayout.PageLines
)

// TextBox 是可翻頁的文字視窗，排版邏輯在 textlayout。
type TextBox = textlayout.TextBox

// NewTextBox 建立一個顯示 s 的文字視窗，依原版規格斷行分頁。
func NewTextBox(s string) *TextBox { return textlayout.NewTextBox(s) }

// Draw 把目前這一頁畫在 (x, y)，並在下方畫出提示。
func DrawTextBox(dst *ebiten.Image, b *TextBox, f *Font, x, y int, frame color.Color) {
	if !b.Active() {
		return
	}
	lines := b.Lines()

	w := (TextBoxColumns + 2) * f.Width()
	h := (TextBoxPageLines + 2) * (f.Height() + 2)
	dst.SubImage(rect(x, y, w, h)).(*ebiten.Image).Fill(color.RGBA{0, 0, 0, 0xff})
	StrokeRect(dst, x, y, w, h, frame)

	for i, ln := range lines {
		f.Draw(dst, ln, x+f.Width(), y+(f.Height()+2)*(i+1)-2)
	}

	hint := "-- more --"
	if !b.HasMore() {
		hint = "-- ok --"
	}
	f.Draw(dst, hint, x+f.Width(), y+(f.Height()+2)*(TextBoxPageLines+1)-2)
}
