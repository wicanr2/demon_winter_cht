package gfx

import (
	"fmt"
	"image"
	"image/color"
)

// DecodeMonoFont8x12 把 1bpp、8×12 點陣字型資料（VGA BIOS 字型的標準存法：
// 每個字元 12 bytes，每 byte 一列、MSB-first、1=前景）解成一張 glyph atlas
// PNG，方便肉眼驗證字型解對了沒有(能不能認出英文字母形狀)。cols 決定
// atlas 每列放幾個字元。
func DecodeMonoFont8x12(data []byte, glyphCount, cols int) (*image.RGBA, error) {
	const gw, gh = 8, 12
	need := glyphCount * gh
	if len(data) < need {
		return nil, fmt.Errorf("gfx: 字型資料太短: 需要 %d bytes(%d 字元 × %d rows)，實際 %d", need, glyphCount, gh, len(data))
	}
	rows := (glyphCount + cols - 1) / cols
	img := image.NewRGBA(image.Rect(0, 0, cols*gw, rows*gh))
	fg := color.RGBA{0xff, 0xff, 0xff, 0xff}
	bg := color.RGBA{0x00, 0x00, 0x00, 0xff}
	for g := 0; g < glyphCount; g++ {
		gx := (g % cols) * gw
		gy := (g / cols) * gh
		for row := 0; row < gh; row++ {
			b := data[g*gh+row]
			for bit := 0; bit < gw; bit++ {
				on := (b>>(uint(7-bit)))&1 != 0
				c := bg
				if on {
					c = fg
				}
				img.SetRGBA(gx+bit, gy+row, c)
			}
		}
	}
	return img, nil
}
