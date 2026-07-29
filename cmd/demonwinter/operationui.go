package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// drawOperationMenu 從 ui.json 取得復古順序與現代分組；items 只承載引擎
// 算出的 label 與 enabled 狀態，因此版面資料與規則不互相滲透。
func (a *app) drawOperationMenu(dst *ebiten.Image, layoutName string,
	items map[string]ui.MenuItem) {
	layoutData, ok := a.tr.CommandLayout(layoutName)
	if !ok {
		return // 啟動時已驗必需 layout；這裡只作防禦。
	}
	retro := make([]ui.MenuItem, 0, len(layoutData.Retro))
	for _, key := range layoutData.Retro {
		if item, exists := items[key]; exists {
			retro = append(retro, item)
		}
	}
	if a.controls == controlsRetro {
		ui.DrawMenuList(dst, a.font, retro, -1,
			layout.MenuX, layout.MenuY, layout.MenuW)
		return
	}
	modern := make([]ui.CommandGroup, 0, len(layoutData.Groups))
	for _, groupData := range layoutData.Groups {
		group := ui.CommandGroup{
			Title:  a.tr.UI(groupData.TitleKey),
			Column: groupData.Column,
		}
		for _, key := range groupData.Items {
			if item, exists := items[key]; exists {
				group.Items = append(group.Items, item)
			}
		}
		if len(group.Items) > 0 {
			modern = append(modern, group)
		}
	}
	ui.DrawModernCommandPanel(dst, a.font, modern,
		layout.StatusX, layout.MenuY, layout.StatusPixels)
}
