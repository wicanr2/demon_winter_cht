package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/assets/cjk"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// asciiScale 是英文字模的放大倍率。CGA 字型是 8×8，放大兩倍後與中文同高。
const asciiScale = 2

// LineHeight 是混排文字的行高，等於中文格高。
const LineHeight = textlayout.CellHeight

// MixedFont 是可同時排中文與 ASCII 的字型。
//
// 中文走倚天點陣（16×15，在 16×16 格內置中），
// ASCII 走原版 CGA 字型（8×8 放大兩倍）。
type MixedFont struct {
	cjk   *cjk.Font
	ascii *Font

	// cache 存已上傳成材質的中文字模，避免每幀重建。
	cache map[rune]*ebiten.Image
	// fg 是中文的前景色。倚天字模是 1bpp，顏色由這裡決定。
	fg color.RGBA

	// missing 記錄取不到字模的字，供品質檢查。
	missing map[rune]bool
}

// NewMixedFont 建立混排字型。ascii 必須是 CGA 8×8 那一套。
func NewMixedFont(c *cjk.Font, ascii *Font, fg color.RGBA) (*MixedFont, error) {
	if ascii.Width() != CGAGlyphWidth || ascii.Height() != CGAGlyphHeight {
		return nil, fmt.Errorf("ui: 混排需要 CGA 8x8 字型，得到 %dx%d",
			ascii.Width(), ascii.Height())
	}
	return &MixedFont{
		cjk:     c,
		ascii:   ascii,
		cache:   map[rune]*ebiten.Image{},
		fg:      fg,
		missing: map[rune]bool{},
	}, nil
}

// glyph 取得（必要時建立）一個中文字的材質。
func (m *MixedFont) glyph(ch rune) *ebiten.Image {
	if img, ok := m.cache[ch]; ok {
		return img
	}
	src, ok := m.cjk.Glyph(ch)
	if !ok {
		m.missing[ch] = true
		m.cache[ch] = nil
		return nil
	}

	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if src.AlphaAt(x, y).A != 0 {
				rgba.SetRGBA(x, y, m.fg)
			}
		}
	}
	img := ebiten.NewImageFromImage(rgba)
	m.cache[ch] = img
	return img
}

// Draw 在 (x, y) 畫一行中英混排文字，回傳排完後的 x 座標。
//
// 中文字模 16×15 在 16×16 格內垂直置中；ASCII 8×8 放大兩倍後填滿 8×16。
// 取不到字模的字畫成空格並記進 missing，不中斷渲染。
func (m *MixedFont) Draw(dst *ebiten.Image, s string, x, y int) int {
	for _, ch := range s {
		if ch < 0x80 {
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			op.GeoM.Scale(asciiScale, asciiScale)
			op.GeoM.Translate(float64(x), float64(y))
			if g := m.ascii.glyphFor(ch); g != nil {
				dst.DrawImage(g, op)
			}
			x += textlayout.CellWidthASC
			continue
		}

		if g := m.glyph(ch); g != nil {
			op := &ebiten.DrawImageOptions{}
			// 字模 16×15、格子 16×16 → 垂直置中偏移 (16−15)/2 = 0（向下取整）。
			// 寫成算式而不是常數，換 24×24 字模時不用改這裡。
			ox := (textlayout.CellWidthCJK - cjk.GlyphWidth) / 2
			oy := (textlayout.CellHeight - cjk.GlyphHeight) / 2
			op.GeoM.Translate(float64(x+ox), float64(y+oy))
			dst.DrawImage(g, op)
		}
		x += textlayout.CellWidthCJK
	}
	return x
}

// Missing 回傳目前為止取不到字模的字。
//
// **fallback 數量是品質指標**：一大批字掉進這裡時，
// 先懷疑索引公式或漏帶 SPCFONT，不要無腦補字型。
func (m *MixedFont) Missing() []rune {
	out := make([]rune, 0, len(m.missing))
	for ch := range m.missing {
		out = append(out, ch)
	}
	return out
}

// DrawLines 逐行畫出一段已斷好行的混排文字。
func (m *MixedFont) DrawLines(dst *ebiten.Image, lines []string, x, y int) {
	for i, ln := range lines {
		m.Draw(dst, ln, x, y+i*textlayout.CellHeight)
	}
}
