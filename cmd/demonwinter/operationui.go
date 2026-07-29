package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// drawOperationMenu 保證復古模式仍走原版紅色單欄；現代模式才使用兩欄與
// tab 式語意標題。兩邊吃同一批 MenuItem，因此可用／停用判定不會分叉。
func (a *app) drawOperationMenu(dst *ebiten.Image, retro []ui.MenuItem,
	modern []ui.CommandGroup) {
	if a.controls == controlsRetro {
		ui.DrawMenuList(dst, a.font, retro, -1,
			layout.MenuX, layout.MenuY, layout.MenuW)
		return
	}
	ui.DrawModernCommandPanel(dst, a.font, modern,
		layout.StatusX, layout.MenuY, layout.StatusPixels)
}

func commandGroup(title string, column int, items []ui.MenuItem) ui.CommandGroup {
	return ui.CommandGroup{Title: title, Column: column, Items: items}
}
