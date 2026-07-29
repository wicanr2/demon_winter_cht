package gfx

import (
	"image"
	"image/color"
)

// modernEGAPalette 是第三套可選主題的受限色盤。它不改 frame 編號、尺寸或
// 透明規則，只把原版 EGA 的 16 個硬體色映到較柔和、層次較清楚的現代像素色。
// 因此 102 個地形索引與所有動畫索引仍能逐格對照原版。
var modernEGAPalette = [16]color.RGBA{
	{0x12, 0x15, 0x1d, 0xff}, // 0 黑
	{0xe4, 0x4b, 0x3f, 0xff}, // 1 淺紅
	{0x4f, 0xa8, 0x62, 0xff}, // 2 樹葉綠
	{0xd8, 0xb4, 0x5a, 0xff}, // 3 沙／平原
	{0x4f, 0x78, 0xb8, 0xff}, // 4 水
	{0x25, 0x68, 0x49, 0xff}, // 5 深綠
	{0x1d, 0x36, 0x63, 0xff}, // 6 深藍
	{0xb7, 0xb4, 0xaa, 0xff}, // 7 石灰
	{0x6f, 0x70, 0x3b, 0xff}, // 8 暗橄欖
	{0x66, 0xc9, 0xd4, 0xff}, // 9 亮青
	{0xe0, 0xa6, 0x87, 0xff}, // 10 膚色
	{0x82, 0x35, 0x3c, 0xff}, // 11 深紅
	{0x6f, 0x70, 0x3b, 0xff}, // 12 暗橄欖（原版重複色）
	{0xed, 0xce, 0x68, 0xff}, // 13 黃
	{0x9f, 0x35, 0x4d, 0xff}, // 14 磚紅
	{0xe8, 0xdf, 0xc8, 0xff}, // 15 羊皮紙白
}

func modernizeImage(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			c := src.RGBAAt(x, y)
			best, bestDistance := 0, int(^uint(0)>>1)
			for i, raw := range GamePalette {
				p := EGAColor(raw)
				dr, dg, db := int(c.R)-int(p.R), int(c.G)-int(p.G), int(c.B)-int(p.B)
				d := dr*dr + dg*dg + db*db
				if d < bestDistance {
					best, bestDistance = i, d
				}
			}
			dst.SetRGBA(x, y, modernEGAPalette[best])
		}
	}
	return dst
}

// ModernizeTileset 建立保持索引與 EGA 32×28 尺寸的現代調色版本。
func ModernizeTileset(src *Tileset) *Tileset {
	frames := make([]*image.RGBA, len(src.frames))
	for i, frame := range src.frames {
		frames[i] = modernizeImage(frame)
	}
	return &Tileset{set: src.set, mode: ModeEGA, w: src.w, h: src.h, frames: frames}
}

// ModernizeSpriteSheet 與 ModernizeTileset 相同，但用於戰鬥／怪物／船 atlas。
func ModernizeSpriteSheet(src *SpriteSheet) *SpriteSheet {
	frames := make([]*image.RGBA, len(src.frames))
	for i, frame := range src.frames {
		frames[i] = modernizeImage(frame)
	}
	return &SpriteSheet{mode: ModeEGA, w: src.w, h: src.h, frames: frames}
}
