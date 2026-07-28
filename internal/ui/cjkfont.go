package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/assets/cjk"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// ASCII 字模的放大倍率，依來源決定：
//
//	CGA `ASC.FNT`  8×14  → ×2（放大兩倍才與中文同高）
//	EGA `ASC.FNE` 16×14  → ×1
//
// **兩者的前進量都是 16**（`textlayout.CellWidthASC`），所以換字型不動版面。
//
// ⚠ ×2 那條路會讓英文的像素是 2×2、中文是 1×1，同一行字兩種密度，
// 看起來一粗一細。EGA 那套是原版在 EGA 模式下**實際使用**的字模，
// 1×1，與倚天字模同密度 —— 這是預設路徑。CGA 那條留著是因為
// `-video cga` 還在，而且它是原版 CGA 模式的真實樣子。
const (
	asciiScaleCGA = 2
	asciiScaleEGA = 1
)

// LineHeight 是混排文字的行高，等於中文格高。
const LineHeight = textlayout.CellHeight

// MixedFont 是可同時排中文與 ASCII 的字型。
//
// 中文走倚天點陣（16×15，在 16×16 格內置中），
// ASCII 走原版 CGA 字型（8×8 放大兩倍）。
type MixedFont struct {
	cjk   *cjk.Font
	ascii *Font
	// asciiScale 由字模尺寸決定，見上面的常數。
	asciiScale int
	// asciiOffsetY 讓字模在 16 高的格子裡垂直置中。
	asciiOffsetY int

	// cache 存已上傳成材質的中文字模，避免每幀重建。
	cache map[rune]*ebiten.Image
	// fg 是中文的前景色。倚天字模是 1bpp，顏色由這裡決定。
	fg color.RGBA

	// missing 記錄取不到字模的字，供品質檢查。
	missing map[rune]bool

	// fullWidthASCII 開著時，英數改走**倚天的全形英數**（Big5 A2AF 起）
	// 而不是原版的點陣字模。見 UseFullWidthASCII。
	fullWidthASCII bool
}

// UseFullWidthASCII 切換英數要用倚天的全形字還是原版的字模。
//
// **為什麼需要這個選項。** 原版的 ASCII 字模（CGA `ASC.FNT` 與 EGA
// `ASC.FNE`）都是粗筆畫的顯示體：`ASC.FNE` 的 `A` 直筆有 4 px 寬，
// 而倚天漢字的筆畫是 1–2 px。兩者放在同一行，英文看起來又黑又重、
// 中文看起來很細，是**筆畫重量**的差異，不只是像素密度。
//
// 倚天自己的全形英數（`Ａ` = Big5 `A2CF`）是 2 px 筆畫、16×15，
// 與漢字同一套設計、同一個尺寸 —— 開起來混排就一致了，
// **而且前進量仍是 16，版面完全不動**。
//
// 代價是英數不再是原版的字形。這是一個取捨不是修正：
// 忠實度那一側選原版字模，一致性那一側選全形。
func (m *MixedFont) UseFullWidthASCII(on bool) { m.fullWidthASCII = on }

// fullWidthOf 把半形英數換成全形（`A` → `Ａ`，差 0xFEE0）。表外的字回傳 0。
//
// **只換字母與數字，標點不換。** 全形標點在倚天裡是照中文排版設計的，
// 位置與字面都不一樣：`．` 是**置中的點**不是基線句點，所以
// `DEMON.SHE` 會變成 `DEMON・SHE`（實跑抓到）。`，` 同理偏高。
// 標點筆畫細、面積小，維持原版字模看不太出來重量差；
// 字母數字面積大、出現頻繁，那才是「一粗一細」的來源。
//
// 空白也不換 —— 兩邊都是空的，換了沒有差別。
func fullWidthOf(ch rune) rune {
	switch {
	case ch >= '0' && ch <= '9',
		ch >= 'A' && ch <= 'Z',
		ch >= 'a' && ch <= 'z':
		return ch + 0xFEE0
	}
	return 0
}

// NewMixedFont 建立混排字型。
//
// ascii 收 CGA 8×8（放大兩倍）或 EGA 16×14（原尺寸）兩種；
// 判準是**放大後的寬度必須等於 `CellWidthASC`**，不合就報錯 ——
// 排版格寬是全專案的共用前提，讓一個尺寸不合的字模靜默通過，
// 症狀會變成十幾個畫面「有點不齊」而不是一個看得見的錯誤。
func NewMixedFont(c *cjk.Font, ascii *Font, fg color.RGBA) (*MixedFont, error) {
	var scale int
	switch {
	case ascii.Width() == CGAGlyphWidth && ascii.Height() == CGAGlyphHeight:
		scale = asciiScaleCGA
	case ascii.Width() == EGAGlyphWidth && ascii.Height() == EGAGlyphHeight:
		scale = asciiScaleEGA
	default:
		return nil, fmt.Errorf("ui: 混排認得 CGA %dx%d 與 EGA %dx%d，得到 %dx%d",
			CGAGlyphWidth, CGAGlyphHeight, EGAGlyphWidth, EGAGlyphHeight,
			ascii.Width(), ascii.Height())
	}
	if w := ascii.Width() * scale; w != textlayout.CellWidthASC {
		return nil, fmt.Errorf("ui: 字模放大後寬 %d，排版格寬是 %d",
			w, textlayout.CellWidthASC)
	}
	return &MixedFont{
		cjk:          c,
		ascii:        ascii,
		asciiScale:   scale,
		asciiOffsetY: (textlayout.CellHeight - ascii.Height()*scale) / 2,
		cache:        map[rune]*ebiten.Image{},
		fg:           fg,
		missing:      map[rune]bool{},
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
// 中文字模 16×15 在 16×16 格內垂直置中；ASCII 同理（EGA 16×14 上下各留 1）。
// 取不到字模的字畫成空格並記進 missing，不中斷渲染。
func (m *MixedFont) Draw(dst *ebiten.Image, s string, x, y int) int {
	for _, ch := range s {
		if ch < 0x80 {
			// 全形英數走中文那條路。取不到字模就掉回原版字模 ——
			// Big5 的全形區沒有涵蓋每一個 ASCII，靜默畫成空白比較糟。
			if m.fullWidthASCII {
				if w := fullWidthOf(ch); w != 0 {
					if g := m.glyph(w); g != nil {
						op := &ebiten.DrawImageOptions{}
						ox := (textlayout.CellWidthCJK - cjk.GlyphWidth) / 2
						oy := (textlayout.CellHeight - cjk.GlyphHeight) / 2
						op.GeoM.Translate(float64(x+ox), float64(y+oy))
						dst.DrawImage(g, op)
						x += textlayout.CellWidthASC
						continue
					}
					// 掉回原版字模時把它從 missing 移掉，
					// 否則品質檢查會被一整批英數洗版。
					delete(m.missing, w)
				}
			}
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			op.GeoM.Scale(float64(m.asciiScale), float64(m.asciiScale))
			op.GeoM.Translate(float64(x), float64(y+m.asciiOffsetY))
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
