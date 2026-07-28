package main

import (
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
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
//
// **原版美術不動。** 花體 `DEMON'S WINTER` logo、移植署名
// （Judit Buczolich／Laszlo Fazekas）、SSI 與 Novotrade 的標記
// 都是 1988 年的歷史紀錄，重繪成中文等於把它們塗掉
// （`rulebook/83` 保全歷史、`rulebook/93` 素材用原版）。
//
// 中文標題改放在**圖上方的黑邊**：玩家一開遊戲就看得到「冬之魔」，
// 原版畫面也一格都沒動。
func (a *app) drawTitle(dst *ebiten.Image) {
	b := a.title.Bounds()
	x := (layout.CanvasWidth - b.Dx()) / 2
	y := (layout.CanvasHeight - b.Dy()) / 2
	ui.DrawImageAt(dst, a.title, x, y)

	// 「冬之魔」是 1990 年軟體世界代理版的官方中文標題，
	// 不是本專案另取的譯名（`translations/glossary.md` §23）。
	zhTitle := a.tr.UI("title.name", "冬之魔")
	tw := textlayout.TextWidth(zhTitle)
	a.font.Draw(dst, zhTitle, (layout.CanvasWidth-tw)/2, (y-ui.LineHeight)/2)

	// 提示放在圖下方的黑邊裡，不蓋到美術。
	a.font.Draw(dst, a.tr.UI("title.press", "按任意鍵開始"), layout.BoxPadX,
		layout.CanvasHeight-ui.LineHeight-2)
}
