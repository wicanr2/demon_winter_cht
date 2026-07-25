// Package ui 是呈現層：把 assets 解出來的影像轉成 Ebiten 可畫的資源，
// 並提供文字與框線等基本繪製。
//
// assets 層只回傳標準 image.RGBA，不認識 Ebiten；規則層 game 不認識畫面。
// Ebiten 的相依集中在這一層與 cmd。
package ui

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

// EGA 字型（ASC.FNE）的字元尺寸與字表起點。見 docs/spec/09-fonts.md。
//
// 這裡用 EGA 版而不是 CGA 版，是因為**CGA 的 ASC.FNT 目前解不出可讀字形**
// （現行解碼器輸出是雜訊，見 workplace/dump/gfx/font-asc-fnt-atlas-zoom4x.png）。
// CGA 字型解開後這裡要補上，兩套都是要保存的數位文物。
const (
	GlyphWidth  = 16
	GlyphHeight = 14
	firstGlyph  = 0x20
)

// Font 是已上傳成 Ebiten 材質的點陣字型。
type Font struct {
	glyphs []*ebiten.Image
}

// LoadEGAFont 讀取 ASC.FNE／GOT.FNE 並轉成可繪製的字型。
//
// EGA 字型是 1bpp，前景色由呼叫端指定。
func LoadEGAFont(path string, fg, bg color.RGBA) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ui: 讀取字型 %s 失敗: %w", path, err)
	}
	imgs, err := gfx.DecodeEGAFont(data, fg, bg)
	if err != nil {
		return nil, fmt.Errorf("ui: 解碼字型 %s 失敗: %w", path, err)
	}
	f := &Font{glyphs: make([]*ebiten.Image, len(imgs))}
	for i, g := range imgs {
		f.glyphs[i] = ebiten.NewImageFromImage(g)
	}
	return f, nil
}

// Draw 在 (x, y) 畫一行 ASCII 文字。
//
// 只處理 0x20–0x7F；超出範圍的位元組畫成空白。中文另有 16×16 的路徑，
// 尚未實作（見 docs/spec/09-fonts.md 的中文化設計段）。
func (f *Font) Draw(dst *ebiten.Image, s string, x, y int) {
	for i := 0; i < len(s); i++ {
		ch := s[i]
		idx := int(ch) - firstGlyph
		if idx < 0 || idx >= len(f.glyphs) {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x+i*GlyphWidth), float64(y))
		dst.DrawImage(f.glyphs[idx], op)
	}
}

// Tileset 是已上傳成 Ebiten 材質的地形圖塊集。
type Tileset struct {
	src   *gfx.Tileset
	tiles []*ebiten.Image
}

// NewTileset 把解好的圖塊集轉成 Ebiten 材質。
func NewTileset(src *gfx.Tileset) *Tileset {
	t := &Tileset{src: src, tiles: make([]*ebiten.Image, src.Len())}
	for i := 0; i < src.Len(); i++ {
		t.tiles[i] = ebiten.NewImageFromImage(src.Tile(byte(i)))
	}
	return t
}

// Name 回傳這套圖塊的來源（常態或雪地）。
func (t *Tileset) Name() string { return string(t.src.Set()) }

// Tile 以 tile 值取材質，超出範圍回傳 nil。
func (t *Tileset) Tile(v byte) *ebiten.Image {
	if int(v) >= len(t.tiles) {
		return nil
	}
	return t.tiles[v]
}

// DrawImageAt 在指定像素座標畫一張材質。
func DrawImageAt(dst, src *ebiten.Image, x, y int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(src, op)
}

// StrokeRect 畫一個一像素寬的空心矩形。
func StrokeRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	line := func(x0, y0, x1, y1 int) {
		sub := dst.SubImage(image.Rect(x0, y0, x1, y1))
		if si, ok := sub.(*ebiten.Image); ok {
			si.Fill(c)
		}
	}
	line(x, y, x+w, y+1)
	line(x, y+h-1, x+w, y+h)
	line(x, y, x+1, y+h)
	line(x+w-1, y, x+w, y+h)
}
