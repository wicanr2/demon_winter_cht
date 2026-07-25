package main

import (
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 開場標題畫面。
//
// 用的是 `OPEN.PIE`（608×336，2026-07-26 解開，見 docs/formats/graphics.md §5）。
//
// ⚠ **原版找的是 `TITLE.PIC`，那個檔不在這份 dump 裡。** `OPEN.*` 這兩個檔名
// 在執行檔裡零命中 —— 原版在這份資料上根本開不了場（DOSBox 截圖的「開場卡住」
// 就是這個原因）。本重製版直接用 `OPEN.PIE`：美術是對的，
// 「原版怎麼載入標題」則仍未解。

// loadTitle 讀開場畫面。讀不到就回 nil —— 沒有標題畫面不該讓遊戲開不起來。
func loadTitle(dataDir string) *ebiten.Image {
	data, err := os.ReadFile(filepath.Join(dataDir, "OPEN.PIE"))
	if err != nil {
		return nil
	}
	img, err := gfx.DecodeTitleScreen(data)
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

// updateTitle 等玩家按鍵離開標題畫面。
func (a *app) updateTitle() error {
	if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
		a.title = nil
	}
	return nil
}

// drawTitle 把標題畫面置中畫在畫布上。
func (a *app) drawTitle(dst *ebiten.Image) {
	b := a.title.Bounds()
	x := (layout.CanvasWidth - b.Dx()) / 2
	y := (layout.CanvasHeight - b.Dy()) / 2
	ui.DrawImageAt(dst, a.title, x, y)

	// 提示放在圖下方的黑邊裡，不蓋到美術。
	a.font.Draw(dst, "按任意鍵開始", layout.BoxPadX,
		layout.CanvasHeight-ui.LineHeight-2)
}
