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

	// MapFrameX/Y 是雙線框左上角；MapFramePad 是框到內容的內距。
	// 世界與戰場都必須走這組原點，否則切換畫面時框會留在原處、
	// 內容卻跳動。頂端留一列給金幣／糧食／日期。
	MapFrameX   = 8
	MapFrameY   = textlayout.CellHeight
	MapFramePad = 4
	MapOriginX  = MapFrameX + MapFramePad
	MapOriginY  = MapFrameY + MapFramePad

	// LogX/Y/W 是世界訊息與戰鬥紀錄的左下區域。EGA 框高 260，
	// 底端在 276，留 12 px 後從 288 開始。
	LogX = 8
	LogY = 288
	LogW = 296

	// MenuX/Y/W 是世界、戰鬥、營地與城鎮共用的選單面板。
	// 中文標籤較英文短，保留寬度給熱鍵與可用狀態，不把面板縮窄。
	MenuX = 480
	MenuY = 128
	MenuW = 160
)

// StatusX 是右欄左緣。地圖框右緣在 304，再留 16 px 分隔。
const StatusX = 320

// StatusY 是狀態欄第一行的頂端；第 0 列留給全寬抬頭。
const StatusY = textlayout.CellHeight

// BoxHeight 是文字視窗的高度：5 行疏排內文 + 一行提示 + 上下內距。
// 事件框是 modal，為了中文行距會覆蓋地圖底端一小段，但永遠貼齊畫布下緣。
const BoxHeight = textlayout.PageLines*uiProseLineHeight +
	textlayout.CellHeight + 2*BoxPadY

// layout 不依賴 ui 套件；此值須與 ui.ProseLineHeight 保持一致。
const uiProseLineHeight = 20

// 文字視窗的內距：左右各一個中文格、上下各半格。
const (
	BoxPadX = textlayout.CellWidthCJK
	BoxPadY = textlayout.CellHeight / 2
)

// TextBoxTop 是 modal 文字視窗頂端；視窗永遠與畫布下緣切齊。
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

// StatusPixels 是狀態欄的可用像素寬，給 textlayout.WrapMixed 用。
//
// 有了 StatusCells 為什麼還要這個：`WrapMixed` 是按像素算的
// （它要處理半形與全形混排），拿格數餵它會斷在錯的地方。
const StatusPixels = CanvasWidth - StatusX
