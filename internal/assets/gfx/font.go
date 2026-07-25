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
	cgaGlyphWidth  = 8  // px
	cgaGlyphHeight = 8  // px
	cgaGlyphBytes  = 16 // 8 rows * 2 bytes/row（每 byte 4 個 2bpp 像素）

	// CGAFontBankGlyphs 是 `.FNT` 每個 bank 的字元數。
	// `.FNT` 有兩個 bank：bank0 = 一般（白字黑底）、bank1 = 反白
	// （黑字洋紅底），各 96 字（ASCII 0x20–0x7F）。
	CGAFontBankGlyphs = 96
	// CGAFontBankBytes 是單一 bank 的位元組數（96 × 16 = 1536）。
	CGAFontBankBytes = CGAFontBankGlyphs * cgaGlyphBytes
)

// DecodeCGAFont 解 `.FNT`（CGA 版字型）。
//
// 佈局（2026-07-25 重解，推翻先前「1-byte header + 逐列雙平面」的錯誤斷言，
// 見 docs/re/17-font-format.md §2）。證據來自 workplace/ghidra/export/
// disassembly.asm 的原始指令，不是 decompile：
//
//	【每字 16 bytes、字表從 0x20 起、bank 選擇】1d9f:0f1e 起的 CGA 分支
//	    1d9f:0f1e  MOV AX,[0x53b2]        ; 3072 = 檔案資料長度（開機時寫死）
//	    1d9f:0f25  MOV CX,0x2 / MOV BX,0  ; → 32-bit 除法 helper → 1536
//	    1d9f:0f32  MOV AX,[0x1ff0]        ; 反白旗標（0/1）
//	    …          32×32 乘法             ; 1536 * flag  = bank 起始位移
//	    1d9f:0f4e  MOV AX,[BP+0xa]        ; 字元碼
//	    1d9f:0f51  MOV CX,0x4 / SHL AX,CL ; char * 16    → 每字 16 bytes
//	    1d9f:0f56  ADD AX,0xfe00          ; -0x200 = -(0x20*16) → 字表起於 0x20
//	    1d9f:0f74  CALLF 0x2000:1a0a      ; = 217b:025a（blit）
//
//	【每列 2 bytes、8 列、來源線性】217b:025a
//	    217b:0283  MOVSW / ADD DI,0x4e / ADD SI,0x2   ; 迴圈 CX=4
//	    217b:029a  ADD DI,0x2000                       ; 切到奇數掃描線 bank
//	    217b:02a7  MOVSW / ADD DI,0x4e / ADD SI,0x2   ; 迴圈 CX=4
//	  第一組（SI 不偏移）讀來源 0,4,8,12 → 螢幕列 0,2,4,6；
//	  第二組（SI+2）讀 2,6,10,14 → 螢幕列 1,3,5,7。
//	  兩組合起來「螢幕列 r ← 來源位移 2r」——來源是**線性**的，
//	  framebuffer 的偶奇交錯在函式內部產生，不存在於檔案裡。
//	  每列只搬 2 bytes（一次 MOVSW），CGA mode 4 每 byte 4 像素 → 8 px 寬。
//
//	【packed 2bpp，不是 planar】每列 2 bytes 被 MOVSW 原封搬進 0xB800
//	  framebuffer 相鄰的兩個 byte，而 CGA mode 4 的 framebuffer 定義就是
//	  「每 byte 4 個像素、2 bits/像素、最左像素在最高位」。所以字型檔的
//	  byte0 = 該列左邊 4 個像素、byte1 = 右邊 4 個像素。
//	  這也跟繪字格寬一致：1d9f:0f6c `MOV CX,[BP+8] / SHL CX,1`
//	  （字元欄 × 2 bytes）、1d9f:0f66 `MOV AX,0x140`（字元列 × 320 bytes
//	  = 4 個 field row = 8 條掃描線）。
//
//	【沒有 header】檔案 3073 bytes = 3072 資料 + 1 個結尾 0x1A（DOS EOF），
//	  資料從位移 0 開始。開機常數 [0x53b2]=0xc00(3072) 就是不含那個 0x1A 的
//	  長度；EGA 版 .FNE 同理是 0xa80(2688) 且檔案剛好 2688 bytes。
//
//	【bank1 = 反白】[0x1ff0] 是反白旗標：1d9f:12ea 設 1、1d9f:1357 設 0，
//	  中間畫的是「選單反白項目」。同一支函式在 1d9f:12c7/12ce 依 CGA/EGA
//	  先填底色 0xb（EGA 亮青）或 0xaa（CGA：0xAA = 四個 2bpp 像素全為
//	  色號 2 = 亮洋紅）。實際 dump 出的 bank1 正是「色號 0 的字 + 色號 2 的
//	  底」，與 0xAA 填色完全吻合；bank0 則只用色號 0 與 3。
//
// 回傳 192 個 glyph：索引 [0,96) 是一般字（ASCII 0x20 起遞增），
// [96,192) 是同一批字的反白版。GlyphForChar 取的是一般字那組。
func DecodeCGAFont(data []byte) ([]*image.RGBA, error) {
	n := len(data) / cgaGlyphBytes
	if n == 0 {
		return nil, fmt.Errorf("gfx: CGA 字型資料不足一個字元(%d bytes < %d)", len(data), cgaGlyphBytes)
	}
	glyphs := make([]*image.RGBA, n)
	for g := 0; g < n; g++ {
		chunk := data[g*cgaGlyphBytes : (g+1)*cgaGlyphBytes]
		img := image.NewRGBA(image.Rect(0, 0, cgaGlyphWidth, cgaGlyphHeight))
		for row := 0; row < cgaGlyphHeight; row++ {
			for col := 0; col < cgaGlyphWidth; col++ {
				b := chunk[row*2+col/4]
				shift := uint(6 - 2*(col%4)) // MSB-first：每 byte 4 像素
				idx := (b >> shift) & 0x3
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
