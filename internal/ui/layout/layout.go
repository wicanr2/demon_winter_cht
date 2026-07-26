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

// 地圖視窗。9×9。
//
// **戰鬥時這個尺寸是對的** —— 原版的戰場確定是 9×9（三個獨立證據見
// game.BattleGridWidth 的說明），而戰場就畫在這個視窗裡，兩邊的一致性由
// TestBattleGrid_MatchesViewport 釘住。
//
// **走世界地圖時仍是暫定值**：原版大地圖視窗開多大還沒從原版釘出來。
// 當初取 9×9 的理由是讓右側狀態欄放得下 21 個中文字、下方放得下五行
// 文字視窗；現在多了一個理由 —— 與戰場同尺寸，兩個畫面切換時不會跳動。
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

// RosterLinesPerMember 是隊伍名冊每個成員佔幾行。
//
// 姓名／等級／職業、種族／生命、法力／未用點數、武器／護甲 —— 共四行。
// 名冊開著時文字視窗不會同時出現，所以它可以用滿整張畫布的高度。
const RosterLinesPerMember = 4

// StatusCells 是狀態欄一行放得下幾個排版格。
//
// 溢出的字會畫到畫布外被裁掉，看起來像訊息被砍一半（存檔路徑就踩過）。
// 呼叫端用 textlayout.TruncateCells 收在這個寬度內。
const StatusCells = (CanvasWidth - StatusX) / textlayout.CellWidthCJK
