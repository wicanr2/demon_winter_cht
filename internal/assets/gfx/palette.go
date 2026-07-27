// Package gfx 解碼 Demon's Winter 的 CGA/EGA 美術素材（畫面 PIC/PIE、
// 精靈圖 SHP/SHE、字型 FNT/FNE），輸出成標準 RGBA 影像供 Ebiten 渲染或
// 存成 PNG 供人眼比對驗證。
//
// 這個套件的驗收標準不是「解碼不回傳 error」，而是輸出的 PNG 要能對上
// DOSBox 原版截圖（workplace/dosbox/shots/*.png）。細節與已驗證/假設
// 的區分見 docs/formats/graphics.md。
package gfx

import "image/color"

// EGAPalette 是標準 IBM EGA/VGA 16 色調色盤（RGBI 電平：0x00/0x55/0xAA/0xFF），
// 索引即 EGA 4-bit 色碼（bit0=Blue, bit1=Green, bit2=Red, bit3=Intensity）。
// 這是 IBM 官方電路電平，不是本專案臆測——EGA/VGA 相容顯示卡的預設調色盤暫存器
// 出廠值就是這 16 組，任何一份 EGA/VGA 硬體手冊或 DOSBox 原始碼的
// `int10_vga.cpp` 均可查證。
var EGAPalette = [16]color.RGBA{
	{0x00, 0x00, 0x00, 0xff}, // 0 黑 black
	{0x00, 0x00, 0xaa, 0xff}, // 1 藍 blue
	{0x00, 0xaa, 0x00, 0xff}, // 2 綠 green
	{0x00, 0xaa, 0xaa, 0xff}, // 3 青 cyan
	{0xaa, 0x00, 0x00, 0xff}, // 4 紅 red
	{0xaa, 0x00, 0xaa, 0xff}, // 5 洋紅 magenta
	{0xaa, 0x55, 0x00, 0xff}, // 6 棕 brown
	{0xaa, 0xaa, 0xaa, 0xff}, // 7 淺灰 light gray
	{0x55, 0x55, 0x55, 0xff}, // 8 深灰 dark gray
	{0x55, 0x55, 0xff, 0xff}, // 9 淺藍 light blue
	{0x55, 0xff, 0x55, 0xff}, // 10 淺綠 light green
	{0x55, 0xff, 0xff, 0xff}, // 11 淺青 light cyan
	{0xff, 0x55, 0x55, 0xff}, // 12 淺紅 light red
	{0xff, 0x55, 0xff, 0xff}, // 13 淺洋紅 light magenta
	{0xff, 0xff, 0x55, 0xff}, // 14 黃 yellow
	{0xff, 0xff, 0xff, 0xff}, // 15 白 white
}

// CGAPalette1High 是 CGA 320×200 圖形模式（mode 4/5）的 palette 1、
// high-intensity 調色盤：{黑, 青, 洋紅, 白}。這是 IBM CGA 硬體的固定四色組
// 之一（不可自訂成任意色，只能在兩組固定調色盤 × 高/低亮度間選）。
// 已用 DOSBox 截圖驗證（workplace/dosbox/shots/05-cga-hang-open-pic.png
// 肉眼可見黑/青/洋紅/白四色，與此表相符），見 docs/formats/graphics.md。
var CGAPalette1High = [4]color.RGBA{
	{0x00, 0x00, 0x00, 0xff}, // 0 黑
	{0x55, 0xff, 0xff, 0xff}, // 1 高亮青 light cyan
	{0xff, 0x55, 0xff, 0xff}, // 2 高亮洋紅 light magenta
	{0xff, 0xff, 0xff, 0xff}, // 3 白
}

// EGAColor 把 6-bit EGA 調色盤暫存器的值換成 RGB。
//
// **調色盤值是 6 bit，不是 4 bit。** EGA 的 Attribute Controller 每個
// 調色盤暫存器存 `r g b R G B` 六個位元：低三位是主要色（各 2/3 強度），
// 高三位是次要色（各 1/3 強度）。合起來是 64 色裡的一色。
//
//	R = 主R×170 + 次r×85
//	G = 主G×170 + 次g×85
//	B = 主B×170 + 次b×85
//
// 先前的實作把值 `&0x0f` 之後拿去索引 16 色標準色盤 —— 那會把高三位
// 整組丟掉。最好認的症狀是**棕色變紅色**：EGA 的棕（色號 6）是 `0x14`，
// 遮成 4 bit 變 `0x04` = 紅。`.PIE` 美術的膚色與土色就是這樣整片偏成洋紅。
func EGAColor(v byte) color.RGBA {
	c := func(primary, secondary uint) uint8 {
		var n uint
		if primary != 0 {
			n += 170
		}
		if secondary != 0 {
			n += 85
		}
		return uint8(n)
	}
	return color.RGBA{
		R: c(uint(v>>2)&1, uint(v>>5)&1),
		G: c(uint(v>>1)&1, uint(v>>4)&1),
		B: c(uint(v>>0)&1, uint(v>>3)&1),
		A: 0xff,
	}
}

// GamePalette 是**原版自己設進 EGA 調色盤暫存器的 16 個值**（EGA 6-bit），
// 不是標準 16 色表。
//
// 標準表只是顯示卡的出廠值；這款遊戲開機時把它整組換掉了，所以拿標準表
// 去解 `.SHE`／`.PIE` 素材，顏色會系統性地錯（沙地變黃、水變紅、樹幹變青）。
//
// 出處（`docs/formats/graphics.md` §3.2b）：
//
//	表本體    `DEMON.INT` 檔位移 0x26fb7 ＝ `ds:0x14b7`，16 bytes 全部 ≤ 0x3f
//	設定常式  `2cdc:16b7`：`for i := 0; i < 0x10; i++` 逐格呼叫
//	          `217b:0a81`（INT 10h trampoline）＝ `INT 10h AH=10h AL=00h`
//	呼叫端    `1d9f:1ac1`：`PUSH DS / MOV AX,0x14b7 / PUSH AX / CALLF 2cdc:16b7`
//	閘門      `CMP word ptr [0x19fb],0`＝**只有 EGA 模式才設**
//
// 驗收：把原版 DOSBox EGA 原生截圖（640×350）出現的顏色逐一取出來，
// 12 種主要顏色與本表 **12/12 全中**，含只在地城出現的 `#FF0055`（索引 14）。
var GamePalette = [16]byte{
	0x00, // 0  #000000 黑
	0x3c, // 1  #FF5555 淺紅
	0x02, // 2  #00AA00 綠（樹葉）
	0x26, // 3  #FFAA00 沙／平原（畫面上占比最高）
	0x0b, // 4  #00AAFF 水
	0x0a, // 5  #00AA55 深綠
	0x01, // 6  #0000AA 藍
	0x07, // 7  #AAAAAA 淺灰（石、泥縫）
	0x30, // 8  #555500 暗橄欖
	0x1b, // 9  #00FFFF 亮青
	0x27, // 10 #FFAAAA 膚色
	0x04, // 11 #AA0000 紅
	0x30, // 12 #555500 暗橄欖（與 8 重複，原版就是這樣）
	0x36, // 13 #FFFF00 黃
	0x2c, // 14 #FF0055 深紅（磚）
	0x3f, // 15 #FFFFFF 白
}
