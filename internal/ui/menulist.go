package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// MenuItem 是復古紅底選單的一列。Label 由呼叫端翻譯完成；
// Enabled=false 時保留項目但蓋棋盤網點，讓玩家看得出功能存在但目前不可用。
type MenuItem struct {
	Label   string
	Enabled bool
}

var (
	menuRed    = color.RGBA{0xaa, 0x00, 0x00, 0xff}
	menuBlack  = color.RGBA{0x00, 0x00, 0x00, 0xff}
	menuYellow = color.RGBA{0xff, 0xff, 0x55, 0xff}
	menuGray   = color.RGBA{0xaa, 0xaa, 0xaa, 0xff}
)

// DrawMenuList 畫共用的紅底選單。cursor < 0 代表只顯示熱鍵、不畫游標；
// 這讓現行的直接熱鍵操作可以先共用同一套視覺，後續若接方向鍵選單，
// 不必重做元件。
func DrawMenuList(dst *ebiten.Image, font *MixedFont, items []MenuItem,
	cursor, x, y, w int) {
	if len(items) == 0 || w <= 0 {
		return
	}
	h := len(items) * LineHeight
	FillRect(dst, x, y, w, h, menuRed)

	for i, item := range items {
		rowY := y + i*LineHeight
		if i > 0 {
			FillRect(dst, x, rowY, w, 1, menuBlack)
		}
		if i == cursor && item.Enabled {
			FillRect(dst, x, rowY, w, LineHeight, menuYellow)
		}
		if !item.Enabled {
			// 原版不是單純調暗，而是整列蓋約 50% 棋盤網點。
			for py := 0; py < LineHeight; py += 2 {
				for px := (py / 2) & 1; px < w; px += 2 {
					FillRect(dst, x+px, rowY+py, 1, 1, menuBlack)
				}
			}
		}
		fg := menuBlack
		if !item.Enabled {
			fg = menuGray
		}
		font.DrawColor(dst, item.Label, x+4, rowY, fg)
	}
}
