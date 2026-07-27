package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 紮營選單的 View land（規則在 internal/game/viewland.go，出處 docs/re/38）。
//
// 這是原版唯一的「看地圖」手段 —— 平常畫面上只有腳邊那 9×9。
// 爬上高處之後可以用方向鍵把視窗推開去看遠處，**整隊一天只能看一次**。

// viewLandScreen 是觀地檢視的狀態。
type viewLandScreen struct {
	// member 是選人游標；−1 代表已經在看地圖了。
	member int
	// x, y 是視窗中心的世界座標。
	x, y int
}

func (a *app) openViewLand() {
	if len(a.members) == 0 {
		a.camp.message = "隊伍是空的"
		return
	}
	a.camp.message = ""
	a.camp.viewLand = &viewLandScreen{member: 0}
}

func (a *app) updateViewLand() error {
	v := a.camp.viewLand

	if v.member >= 0 {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			a.camp.viewLand = nil
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			v.member = (v.member + 1) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			v.member = (v.member - 1 + len(a.members)) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			c := &a.members[v.member]
			ok, why := game.CanViewLand(c, int(a.mapID), a.save.ViewedLandToday)
			if !ok {
				a.camp.message = why
				return nil
			}
			a.save.ViewedLandToday = true
			v.x, v.y = game.ViewLandOrigin(a.party.X(), a.party.Y())
			v.member = -1
			a.camp.message = fmt.Sprintf("%s 爬上高處張望", c.Name)
		}
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.camp.viewLand = nil
		return nil
	}
	for _, kf := range keyFacing {
		if inpututil.IsKeyJustPressed(kf.key) {
			v.x, v.y = game.ViewLandStep(v.x, v.y, kf.f)
		}
	}
	return nil
}

func (a *app) drawViewLand(dst *ebiten.Image, line func(string)) {
	v := a.camp.viewLand

	if v.member >= 0 {
		line("誰爬上去看？")
		line("")
		a.drawMemberList(line, v.member, func(i int) string {
			if !a.members[i].HasSkill(game.SkillViewLand) {
				return "（不會觀地）"
			}
			return ""
		})
		line("")
		line("↑↓：選擇　Enter：確定　Esc：取消")
		if a.camp.message != "" {
			line("")
			line(a.camp.message)
		}
		return
	}

	// 地圖畫在左邊的地圖視窗裡，文字排在右邊的狀態欄 ——
	// 與世界地圖畫面同一套版面，切換時不會跳動。
	a.drawMapWindow(dst, v.x, v.y)

	y := layout.StatusY
	right := func(s string) {
		a.font.Draw(dst, textlayout.TruncateCells(s, layout.StatusCells),
			layout.StatusX, y)
		y += ui.LineHeight
	}
	right("觀地")
	right("")
	right(fmt.Sprintf("座標 %d,%d", v.x, v.y))
	right(fmt.Sprintf("子地圖 %d", a.mapID))
	right("")
	right("方向鍵：張望")
	right("Esc：下來")
}

// drawMapWindow 以 (cx, cy) 為中心把世界地圖畫進地圖視窗，中心標一個框。
//
// 與 drawWorld 幾乎相同，差別只在中心可以離開隊伍所在的格子。
func (a *app) drawMapWindow(dst *ebiten.Image, cx, cy int) {
	ts := a.tileset()
	halfX, halfY := layout.ViewTilesX/2, layout.ViewTilesY/2
	cellW, cellH, scale := a.tileMetrics()

	for dy := 0; dy < layout.ViewTilesY; dy++ {
		for dx := 0; dx < layout.ViewTilesX; dx++ {
			mx, my := cx-halfX+dx, cy-halfY+dy
			if mx < 0 || mx >= game.MapWidth || my < 0 || my >= game.MapHeight {
				continue
			}
			img := ts.Tile(a.drawTiles[my*game.MapWidth+mx] & 0x7f)
			if img == nil {
				continue
			}
			ui.DrawImageScaled(dst, img, dx*cellW, dy*cellH, scale)
		}
	}
	ui.StrokeRect(dst, halfX*cellW, halfY*cellH, cellW, cellH, markerColor)
}
