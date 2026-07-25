// Package layout 是中文化後的畫面版面：畫布尺寸、地圖視窗、狀態欄、文字視窗的位置。
//
// 刻意不依賴 Ebiten —— Ebiten 在 init 期就要求顯示器，
// 版面常數混進去就會讓「版面有沒有破」這件事在無頭環境下測不了。
// 繪製由 ui 套件負責，這裡只有座標算術。
package layout

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 畫布。中文化後拉到 640×400（見 docs/spec/09-fonts.md）：
// 中文需要 16×16 點陣才可讀，英文 8×8 放大兩倍後與中文同高，兩者可以混排。
const (
	CanvasWidth  = 640
	CanvasHeight = 400
)

// TileScale 是圖塊的放大倍率。原版圖塊 16×16，放大後一格 32×32，
// 正好是中文格的兩倍，像素格對得齊。
const TileScale = 2

// 地圖視窗。**格數仍是暫定值**，尚未對齊原版版面 —— 取 9×9
// 是為了讓右側狀態欄放得下 21 個中文字、下方放得下五行文字視窗。
const (
	ViewTilesX = 9
	ViewTilesY = 9

	MapWidth  = ViewTilesX * gfx.TileWidth * TileScale
	MapHeight = ViewTilesY * gfx.TileHeight * TileScale
)

// StatusX 是狀態欄左緣，留 8 像素與地圖分開。
const StatusX = MapWidth + 8

// StatusY 是狀態欄第一行的頂端。
const StatusY = 4

// BoxHeight 是文字視窗的高度：5 行內文 + 一行提示 + 上下內距。
const BoxHeight = (textlayout.PageLines+1)*textlayout.CellHeight + 2*BoxPadY

// 文字視窗的內距：左右各一個中文格、上下各半格。
const (
	BoxPadX = textlayout.CellWidthCJK
	BoxPadY = textlayout.CellHeight / 2
)

// TextBoxTop 是文字視窗頂端。它與畫布下緣切齊 ——
// 版面靠這條等式閉合，改任一個常數都會被 layout 的測試擋下來。
const TextBoxTop = CanvasHeight - BoxHeight
