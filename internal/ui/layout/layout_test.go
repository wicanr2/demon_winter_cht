package layout

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
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

// 名冊一次列五個人、每人三行，加上標題與提示共 18 行，要塞得進地圖高度。
func TestLayout_RosterFitsAboveTextBox(t *testing.T) {
	const rosterLines = 1 + 5*3 + 2
	if h := rosterLines * textlayout.CellHeight; h > MapHeight {
		t.Errorf("名冊需要 %d 像素，超過文字視窗上方可用的 %d", h, MapHeight)
	}
}

// 圖塊放大後一格 32×32：中文 16×16 的兩倍，兩者的像素格對得齊。
func TestLayout_TileScaleMatchesTextGrid(t *testing.T) {
	if gfx.TileWidth*TileScale != 2*textlayout.CellHeight {
		t.Errorf("圖塊放大後 %d 像素，預期是行高 %d 的兩倍",
			gfx.TileWidth*TileScale, textlayout.CellHeight)
	}
}
