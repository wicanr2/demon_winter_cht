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
