package gfx

import "testing"

// 6-bit 調色盤值換算。最好認的一組是棕色：EGA 色號 6 的暫存器值是 0x14，
// 若被遮成 4 bit 會變成 0x04 = 紅色。`.PIE` 美術整片偏洋紅就是這個症狀。
func TestEGAColor_SixBitValues(t *testing.T) {
	cases := []struct {
		v    byte
		want [3]uint8
		name string
	}{
		{0x00, [3]uint8{0, 0, 0}, "黑"},
		{0x01, [3]uint8{0, 0, 170}, "藍"},
		{0x02, [3]uint8{0, 170, 0}, "綠"},
		{0x04, [3]uint8{170, 0, 0}, "紅"},
		{0x14, [3]uint8{170, 85, 0}, "棕（遮成 4 bit 會變紅）"},
		{0x3f, [3]uint8{255, 255, 255}, "亮白"},
		{0x38, [3]uint8{85, 85, 85}, "深灰"},
	}
	for _, c := range cases {
		got := EGAColor(c.v)
		if got.R != c.want[0] || got.G != c.want[1] || got.B != c.want[2] {
			t.Errorf("EGAColor(0x%02x) = (%d,%d,%d)，預期 %v — %s",
				c.v, got.R, got.G, got.B, c.want, c.name)
		}
	}
}

// 標準 EGA 預設調色盤的 16 個暫存器值，換算後要等於 EGAPalette。
//
// 這條把兩張表綁在一起：EGAColor 若算錯，用預設調色盤的素材就會變色。
func TestEGAColor_MatchesStandardPalette(t *testing.T) {
	defaults := [16]byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x14, 0x07,
		0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f,
	}
	for i, v := range defaults {
		got, want := EGAColor(v), EGAPalette[i]
		if got != want {
			t.Errorf("色號 %d（暫存器 0x%02x）= %v，標準表是 %v", i, v, got, want)
		}
	}
}
