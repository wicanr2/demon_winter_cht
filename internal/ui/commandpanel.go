package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// CommandGroup 是現代命令區的一個語意分組。Column 只能是 0 或 1；
// 同欄的分組依傳入順序往下排列。
type CommandGroup struct {
	Title  string
	Column int
	Items  []MenuItem
}

var (
	commandPanelBG       = color.RGBA{0x08, 0x12, 0x1d, 0xff}
	commandTabBG         = color.RGBA{0x1d, 0x4f, 0x67, 0xff}
	commandTabLine       = color.RGBA{0x55, 0xff, 0xff, 0xff}
	commandButtonBG      = color.RGBA{0x19, 0x2a, 0x38, 0xff}
	commandButtonBorder  = color.RGBA{0x55, 0x55, 0x55, 0xff}
	commandButtonText    = color.RGBA{0xff, 0xff, 0xff, 0xff}
	commandDisabledText  = color.RGBA{0xaa, 0xaa, 0xaa, 0xff}
	commandDisabledShade = color.RGBA{0x11, 0x1b, 0x24, 0xff}
)

// DrawModernCommandPanel 以兩個固定欄與分頁式標題排列命令。
// 它不改熱鍵，只把原本擠成單一直列的提示整理成可掃讀的語意區塊。
func DrawModernCommandPanel(dst *ebiten.Image, font *MixedFont, groups []CommandGroup,
	x, y, w int) {
	if len(groups) == 0 || w < 32 {
		return
	}
	const gap = 8
	columnW := (w - gap) / 2
	columnY := [2]int{y, y}
	FillRect(dst, x, y, w, 272, commandPanelBG)

	for _, group := range groups {
		column := group.Column
		if column < 0 || column > 1 {
			column = 0
		}
		gx := x + column*(columnW+gap)
		gy := columnY[column]

		// 標題畫成突出面板的 tab：上、左、右有框，底線用亮色銜接內容。
		FillRect(dst, gx, gy, columnW, LineHeight, commandTabBG)
		StrokeRect(dst, gx, gy, columnW, LineHeight, commandButtonBorder)
		FillRect(dst, gx, gy+LineHeight-2, columnW, 2, commandTabLine)
		font.DrawColor(dst, group.Title, gx+8, gy, commandButtonText)
		gy += LineHeight + 3

		for _, item := range group.Items {
			bg, fg := commandButtonBG, commandButtonText
			if !item.Enabled {
				bg, fg = commandDisabledShade, commandDisabledText
			}
			FillRect(dst, gx, gy, columnW, LineHeight-2, bg)
			StrokeRect(dst, gx, gy, columnW, LineHeight-2, commandButtonBorder)
			font.DrawColor(dst, item.Label, gx+6, gy-1, fg)
			gy += LineHeight
		}
		columnY[column] = gy + 6
	}
}
