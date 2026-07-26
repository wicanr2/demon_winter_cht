package layout

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 版面是靠幾條等式閉合的：地圖高 + 文字視窗高 = 畫布高、
// 狀態欄右緣不出界。改任何一個常數都要在這裡被擋下來，
// 不然只會在畫面上默默破版。
func TestLayout_FillsCanvas(t *testing.T) {
	if MapHeight+BoxHeight != CanvasHeight {
		t.Errorf("地圖高 %d + 文字視窗高 %d = %d，預期等於畫布高 %d",
			MapHeight, BoxHeight, MapHeight+BoxHeight, CanvasHeight)
	}
	if TextBoxTop != MapHeight {
		t.Errorf("文字視窗頂端 %d，應接在地圖下緣 %d", TextBoxTop, MapHeight)
	}
	if MapWidth >= CanvasWidth {
		t.Errorf("地圖寬 %d 已佔滿畫布寬 %d，狀態欄沒有位置", MapWidth, CanvasWidth)
	}
}

// 狀態欄至少要放得下 20 個中文字，不然那些標籤會畫到畫布外。
func TestLayout_StatusPanelFitsLabels(t *testing.T) {
	w := CanvasWidth - StatusX
	if cells := w / 16; cells < 20 {
		t.Errorf("狀態欄寬 %d 只放得下 %d 個中文字，至少要 20", w, cells)
	}
}

// 名冊一次列五個人、每人四行（含裝備），加上標題與提示共 23 行。
//
// **名冊佔整張畫布，不是只有地圖那一塊。** 開名冊時文字視窗不會同時出現，
// 所以下半部可以借來用。這條原本釘的是「每人三行、要塞進地圖高度」，
// 加了裝備那一行之後就不成立 —— 行數寫死在測試裡的話，版面改了也不會紅，
// 所以改成引用 layout 的常數。
func TestLayout_RosterFitsCanvas(t *testing.T) {
	const rosterLines = 1 + 5*RosterLinesPerMember + 2
	if h := rosterLines * textlayout.CellHeight; h > CanvasHeight {
		t.Errorf("名冊需要 %d 像素，超過畫布高度 %d", h, CanvasHeight)
	}
}

// 圖塊放大後一格 32×32：中文 16×16 的兩倍，兩者的像素格對得齊。
func TestLayout_TileScaleMatchesTextGrid(t *testing.T) {
	if gfx.TileWidth*TileScale != 2*textlayout.CellHeight {
		t.Errorf("圖塊放大後 %d 像素，預期是行高 %d 的兩倍",
			gfx.TileWidth*TileScale, textlayout.CellHeight)
	}
}

// 戰場（15×15）比視窗（9×9）大，所以畫面一定要會捲動。
//
// 這條測試釘的不是「相等」而是「視窗不大於戰場」+「視窗確實比戰場小」——
// 前者防止畫出戰場外的空格，後者提醒：只要這個關係還在，捲動就不能拿掉。
// 原版也是這樣（`FUN_222f_1404(中心−4, 中心−4)`，見 docs/re/35）。
func TestBattleViewport_IsAScrollingWindow(t *testing.T) {
	cell := gfx.TileWidth * TileScale
	cols, rows := MapWidth/cell, MapHeight/cell

	if cols > game.BattleFieldSize || rows > game.BattleFieldSize {
		t.Errorf("視窗 %d×%d 大於戰場 %d×%d",
			cols, rows, game.BattleFieldSize, game.BattleFieldSize)
	}
	if cols >= game.BattleFieldSize && rows >= game.BattleFieldSize {
		t.Error("視窗與戰場一樣大，那就不需要捲動了 —— 這條測試該改")
	}
	if cols != ViewTilesX || rows != ViewTilesY {
		t.Errorf("視窗 %d×%d 與 ViewTiles %d×%d 對不上",
			cols, rows, ViewTilesX, ViewTilesY)
	}
}
