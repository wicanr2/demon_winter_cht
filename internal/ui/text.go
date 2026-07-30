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
	"image/draw"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

// 字表起點。CGA 與 EGA 兩套字型都從 ASCII 0x20 起算。見 docs/spec/09-fonts.md。
const firstGlyph = 0x20

// 兩套字型的字元尺寸。原版依顯示卡自動擇一，兩套都要支援。
const (
	CGAGlyphWidth  = 8
	CGAGlyphHeight = 8
	EGAGlyphWidth  = 16
	EGAGlyphHeight = 14
)

// Font 是已上傳成 Ebiten 材質的點陣字型。字元尺寸隨來源而異，
// 由 Width()/Height() 取得，呼叫端不要自己假設。
type Font struct {
	atlas         *ebiten.Image
	glyphCount    int
	atlasCols     int
	width, height int
}

// Width 回傳單一字元的像素寬。
func (f *Font) Width() int { return f.width }

// Height 回傳單一字元的像素高。
func (f *Font) Height() int { return f.height }

// LoadCGAFont 讀取 ASC.FNT 並轉成可繪製的字型。
//
// CGA 字型自帶顏色（packed 2bpp），前景色不可指定。
// 檔案含兩個 bank：bank0 是一般白字黑底，bank1 是反白版（黑字亮洋紅底）。
// 這裡只取 bank0；反白版由原版用來畫置中標題。
func LoadCGAFont(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ui: 讀取字型 %s 失敗: %w", path, err)
	}
	imgs, err := gfx.DecodeCGAFont(data)
	if err != nil {
		return nil, fmt.Errorf("ui: 解碼字型 %s 失敗: %w", path, err)
	}
	if len(imgs) > gfx.CGAFontBankGlyphs {
		imgs = imgs[:gfx.CGAFontBankGlyphs]
	}
	return newFont(imgs, CGAGlyphWidth, CGAGlyphHeight), nil
}

func newFont(imgs []*image.RGBA, w, h int) *Font {
	const cols = 16
	rows := (len(imgs) + cols - 1) / cols
	rgba := image.NewRGBA(image.Rect(0, 0, cols*w, rows*h))
	for i, g := range imgs {
		x, y := (i%cols)*w, (i/cols)*h
		draw.Draw(rgba, image.Rect(x, y, x+w, y+h), g, g.Bounds().Min, draw.Src)
	}
	return &Font{
		atlas: ebiten.NewImageFromImage(rgba), glyphCount: len(imgs),
		atlasCols: cols, width: w, height: h,
	}
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
	return newFont(imgs, EGAGlyphWidth, EGAGlyphHeight), nil
}

// Draw 在 (x, y) 畫一行 ASCII 文字。
//
// 只處理 0x20–0x7F；超出範圍的位元組畫成空白。中文另有 16×16 的路徑，
// 尚未實作（見 docs/spec/09-fonts.md 的中文化設計段）。
func (f *Font) Draw(dst *ebiten.Image, s string, x, y int) {
	for i := 0; i < len(s); i++ {
		ch := s[i]
		g := f.glyphFor(rune(ch))
		if g == nil {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x+i*f.width), float64(y))
		dst.DrawImage(g, op)
	}
}

// glyphFor 取一個 ASCII 字元的材質，超出字表回傳 nil。
func (f *Font) glyphFor(ch rune) *ebiten.Image {
	idx := int(ch) - firstGlyph
	if idx < 0 || idx >= f.glyphCount {
		return nil
	}
	x, y := (idx%f.atlasCols)*f.width, (idx/f.atlasCols)*f.height
	return f.atlas.SubImage(image.Rect(x, y, x+f.width, y+f.height)).(*ebiten.Image)
}

// Tileset 是已上傳成 Ebiten 材質的地形圖塊集。
type Tileset struct {
	src         *gfx.Tileset
	tiles       []*ebiten.Image
	transparent []*ebiten.Image
}

// NewTileset 把解好的圖塊集轉成 Ebiten 材質。
func NewTileset(src *gfx.Tileset) *Tileset {
	t := &Tileset{
		src:         src,
		tiles:       make([]*ebiten.Image, src.Len()),
		transparent: make([]*ebiten.Image, src.Len()),
	}
	black := color.RGBA{0, 0, 0, 0xff}
	for i := 0; i < src.Len(); i++ {
		tile := src.Tile(byte(i))
		t.tiles[i] = ebiten.NewImageFromImage(tile)
		t.transparent[i] = ebiten.NewImageFromImage(gfx.TransparentBackground(tile, black))
	}
	return t
}

// Name 回傳這套圖塊實際載入的檔名（含模式）——狀態列靠它分辨現在跑的是
// EGA 還是 CGA 素材。回 TerrainSet 會永遠顯示 `.SHP`，看不出模式。
func (t *Tileset) Name() string { return t.src.Mode().FileName(t.src.Set()) }

// FrameSize 回傳一格的顯示尺寸。EGA 是 32×28、CGA 是 16×16 ——
// 呈現層不該假設它是正方形（原版 EGA 就不是）。
func (t *Tileset) FrameSize() (int, int) { return t.src.FrameSize() }

// Mode 回傳這一套是 EGA 還是 CGA 素材。
func (t *Tileset) Mode() gfx.VideoMode { return t.src.Mode() }

// Source 提供純 image.RGBA 的 atlas 給主題建置階段轉換。呼叫端不得改寫來源。
func (t *Tileset) Source() *gfx.Tileset { return t.src }

// Tile 以 tile 值取材質，超出範圍回傳 nil。
func (t *Tileset) Tile(v byte) *ebiten.Image {
	if int(v) >= len(t.tiles) {
		return nil
	}
	return t.tiles[v]
}

// TransparentTile 回傳去除邊界連通黑底、但保留貼身黑色輪廓的圖塊。
// 探索畫面的步行人物先畫腳下地形，再用這份材質疊上；一般地形仍用 Tile。
func (t *Tileset) TransparentTile(v byte) *ebiten.Image {
	if int(v) >= len(t.transparent) {
		return nil
	}
	return t.transparent[v]
}

// DrawImageAt 在指定像素座標畫一張材質。
func DrawImageAt(dst, src *ebiten.Image, x, y int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(src, op)
}

// DrawImageScaled 在指定像素座標畫一張放大 n 倍的材質。
//
// 一律用 FilterNearest：點陣素材被線性濾波就糊掉。
func DrawImageScaled(dst, src *ebiten.Image, x, y, n int) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(float64(n), float64(n))
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(src, op)
}

// rect 是 image.Rect 的簡寫，讓 SubImage 呼叫讀起來乾淨些。
func rect(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}

// FillRect 填一個實心矩形。
//
// 走 SubImage + Fill 與 StrokeRect 同一條路 —— 這個專案沒有引進
// vector 套件，畫面上的方塊全部靠子影像填色。
func FillRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	sub := dst.SubImage(rect(x, y, w, h))
	if si, ok := sub.(*ebiten.Image); ok {
		si.Fill(c)
	}
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
