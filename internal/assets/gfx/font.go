package gfx

import (
	"fmt"
	"image"
	"image/color"
)

// 字型格式已解出（2026-07-25），詳細反組譯證據見 docs/re/17-font-format.md。
// 兩份繪字常式：
//   - CGA 路徑：FUN_1d9f_0eeb（*(int*)0x19fb==0 分支）→ FUN_217b_025a，
//     直接把字型緩衝區的位元組 MOVSW 搬進 CGA 硬體 framebuffer（ES:0xB800）。
//   - EGA/圖形路徑：FUN_1d9f_0eeb（else 分支）→ FUN_1d9f_0eb1 →
//     FUN_217b_097c，用 EGA GC Write Mode 2 逐列画出字元。
//
// 兩者都從同一個全域字型緩衝區指標 DAT_31f0_5488（far pointer）取資料，
// 字元碼→位移的算式在兩條路徑都是「(char_code-0x20)*每字bytes數」——字表從
// ASCII 0x20（空白）開始，不含控制字元。

// FontFirstChar 是字型表對應的第一個 ASCII 字元碼（0x20 = 空白）。
// 已驗證：FUN_1d9f_0eeb 兩條路徑都对 char_code 做 -0x20 才查表
// （CGA 路徑 `param_3*0x10-0x200` = `(char-0x20)*16`；
// EGA 路徑 `addw $0xffe0,char` = `char-0x20`，兩者共用同一張字表）。
const FontFirstChar = 0x20

// --- CGA 字型（.FNT） ---

const (
	cgaFontHeaderBytes = 1  // 檔頭 1 byte，內容目前觀察到是 0x00，用途未查出
	cgaGlyphWidth      = 8  // px
	cgaGlyphHeight     = 8  // px
	cgaGlyphBytes      = 16 // 8 rows * 2 bytes/row
)

// DecodeCGAFont 解 `.FNT`（CGA 版字型）。
//
// 已驗證（見 docs/re/17-font-format.md §2）：反組譯 FUN_217b_025a
// （217b:025a）逐指令核對出的佈局——
//   - 每字 8 寬 x 8 高，16 bytes/字（8 rows * 2 bytes/row）
//   - 每列 2 bytes 是「同一列的 2bpp PLANAR 資料」：byte0 是該列 8 個像素的
//     bit0 平面，byte1 是 bit1 平面（不是 chunky「左 4px+右 4px」），
//     MSB-first（bit7 = 最左像素）。
//     `color_index = bit0(byte0,bit) | (bit1(byte1,bit) << 1)`。
//   - 字表從 ASCII 0x20（空白）開始依序排列，檔案開頭 1 byte 是本函式
//     呼叫鏈沒有用到的欄位（觀察值恒為 0x00，可能是留給別的載入器共用的
//     header，用途未查出，不影響字形本身）。
//   - 顏色用 CGAPalette1High（黑/亮青/亮洋紅/白）——同一份調色盤在
//     `docs/formats/graphics.md` 已對其他 CGA 素材驗證過。
//
// 回傳字元 glyph 陣列（依 ASCII 0x20 起遞增排列）與涵蓋的字元數。
func DecodeCGAFont(data []byte) ([]*image.RGBA, error) {
	if len(data) <= cgaFontHeaderBytes {
		return nil, fmt.Errorf("gfx: CGA 字型資料太短(%d bytes)，至少需要 %d bytes(header)+1 個字元", len(data), cgaFontHeaderBytes)
	}
	body := data[cgaFontHeaderBytes:]
	n := len(body) / cgaGlyphBytes
	if n == 0 {
		return nil, fmt.Errorf("gfx: CGA 字型資料不足一個字元(%d bytes < %d)", len(body), cgaGlyphBytes)
	}
	glyphs := make([]*image.RGBA, n)
	for g := 0; g < n; g++ {
		chunk := body[g*cgaGlyphBytes : (g+1)*cgaGlyphBytes]
		img := image.NewRGBA(image.Rect(0, 0, cgaGlyphWidth, cgaGlyphHeight))
		for row := 0; row < cgaGlyphHeight; row++ {
			b0 := chunk[row*2]
			b1 := chunk[row*2+1]
			for col := 0; col < cgaGlyphWidth; col++ {
				shift := uint(7 - col) // MSB-first
				bit0 := (b0 >> shift) & 1
				bit1 := (b1 >> shift) & 1
				idx := bit0 | (bit1 << 1)
				img.SetRGBA(col, row, CGAPalette1High[idx])
			}
		}
		glyphs[g] = img
	}
	return glyphs, nil
}

// --- EGA 字型（.FNE） ---

const (
	egaGlyphWidth  = 16 // px
	egaGlyphHeight = 14 // px
	egaGlyphBytes  = 28 // 14 rows * 2 bytes/row
)

// DecodeEGAFont 解 `.FNE`（EGA 版字型，含花體 GOT.FNE）。
//
// 已驗證（見 docs/re/17-font-format.md §3）：反組譯 FUN_217b_097c
// （217b:097c，14 列迴圈、EGA Graphics Controller Write Mode 2）逐指令
// 核對出的佈局——
//   - 每字 16 寬 x 14 高，28 bytes/字（14 rows * 2 bytes/row）
//   - 每列 2 bytes 合起來當一個 16-bit MSB-first 遮罩（bit15=最左像素），
//     1bpp（1=前景、0=背景），不含顏色——EGA 路徑用 Write Mode 2 在繪製時
//     才指定前景/背景色（呼叫端傳入的 `0xff00`＝白底黑字 或 `0xb`＝
//     高亮青底黑字，對應選單反白效果，見文件 §3.4），不是字型檔本身的資料。
//   - 字表從 ASCII 0x20（空白）開始，**無 header**（2688 = 96×28 整除，
//     96 = 0x20~0x7F 標準可印 ASCII 範圍，不含控制字元也不含延伸字元）。
//
// fg/bg 由呼叫端指定（對應原引擎繪製時的前景/背景色，不是字型檔案本身
// 內建的顏色）。
func DecodeEGAFont(data []byte, fg, bg color.RGBA) ([]*image.RGBA, error) {
	n := len(data) / egaGlyphBytes
	if n == 0 {
		return nil, fmt.Errorf("gfx: EGA 字型資料不足一個字元(%d bytes < %d)", len(data), egaGlyphBytes)
	}
	glyphs := make([]*image.RGBA, n)
	rowBytes := egaGlyphWidth / 8
	for g := 0; g < n; g++ {
		chunk := data[g*egaGlyphBytes : (g+1)*egaGlyphBytes]
		img := image.NewRGBA(image.Rect(0, 0, egaGlyphWidth, egaGlyphHeight))
		for row := 0; row < egaGlyphHeight; row++ {
			for col := 0; col < egaGlyphWidth; col++ {
				b := chunk[row*rowBytes+col/8]
				bit := (b >> uint(7-col%8)) & 1
				c := bg
				if bit != 0 {
					c = fg
				}
				img.SetRGBA(col, row, c)
			}
		}
		glyphs[g] = img
	}
	return glyphs, nil
}

// GlyphForChar 依 ASCII 字元碼從 glyphs 陣列（DecodeCGAFont/DecodeEGAFont
// 的回傳值，從 FontFirstChar 起遞增排列）取出對應的 glyph。
// ch 超出字表涵蓋範圍時回傳 nil, false。
func GlyphForChar(glyphs []*image.RGBA, ch byte) (*image.RGBA, bool) {
	if ch < FontFirstChar {
		return nil, false
	}
	idx := int(ch) - FontFirstChar
	if idx < 0 || idx >= len(glyphs) {
		return nil, false
	}
	return glyphs[idx], true
}

// RenderText 把一段 ASCII 字串排成一張橫向影像，方便肉眼比對遊戲截圖裡的
// UI 文字（如 "Walk"、"DEMON'S WINTER"）。查不到的字元(超出字表範圍)畫成
// 該 glyph 寬度的全背景空白，不中斷渲染。
func RenderText(glyphs []*image.RGBA, s string, glyphW, glyphH int, bg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, glyphW*len(s), glyphH))
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < glyphH; y++ {
			img.SetRGBA(x, y, bg)
		}
	}
	for i := 0; i < len(s); i++ {
		g, ok := GlyphForChar(glyphs, s[i])
		if !ok {
			continue
		}
		ox := i * glyphW
		for y := 0; y < glyphH; y++ {
			for x := 0; x < glyphW; x++ {
				img.SetRGBA(ox+x, y, g.RGBAAt(x, y))
			}
		}
	}
	return img
}
