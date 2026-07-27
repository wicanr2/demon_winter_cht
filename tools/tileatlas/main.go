// 把 DEMON.SHE／WINTER.SHE 的 102 格畫成標了索引的 atlas，
// 用來與 DOSBox 原版實機截圖逐格肉眼比對（rulebook/64 的截圖 oracle）。
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

func main() {
	in := flag.String("in", "workplace/orig/demwin/DEM_DATA/DEMON.SHE", "來源 .SHE")
	out := flag.String("out", "workplace/dump/gfx/tileatlas.png", "輸出 PNG")
	cols := flag.Int("cols", 12, "每列幾格")
	zoom := flag.Int("zoom", 3, "放大倍率")
	dbl := flag.Bool("double", true, "水平加倍（原版載入時做的事，顯示寬 32）")
	rev := flag.Bool("revplanes", false, "把 plane 順序反過來組 index（測試 I,R,G,B 佈局）")
	gamePal := flag.Bool("gamepal", false, "用原版自己設的 16 色調色盤（ds:0x14b7）而不是標準 EGA 表")
	flag.Parse()

	data, err := os.ReadFile(*in)
	if err != nil {
		panic(err)
	}
	// 原版開機時用 INT 10h AH=10h AL=00h 逐格設的 16 個調色盤暫存器，
	// 表在 DEMON.INT 的 ds:0x14b7（檔位移 0x26fb7），值是 EGA 6-bit。
	gamePalette := [16]byte{0x00, 0x3c, 0x02, 0x26, 0x0b, 0x0a, 0x01, 0x07,
		0x30, 0x1b, 0x27, 0x04, 0x30, 0x36, 0x2c, 0x3f}

	var frames []*image.RGBA
	if *gamePal {
		frames, err = decodeWithPalette(data, 16, 28, gamePalette)
	} else if *rev {
		frames, err = decodeReversed(data, 16, 28)
	} else {
		frames, err = gfx.DecodeEGASpriteSheet(data, 16, 28, gfx.EGAPlanesRowBlocks)
	}
	if err != nil {
		panic(err)
	}
	fw, fh := 16, 28
	if *dbl {
		fw = 32
	}
	cw, ch := fw**zoom+4, fh**zoom+12
	rows := (len(frames) + *cols - 1) / *cols
	atlas := image.NewRGBA(image.Rect(0, 0, *cols*cw, rows*ch))
	draw.Draw(atlas, atlas.Bounds(), &image.Uniform{color.RGBA{40, 40, 40, 255}}, image.Point{}, draw.Src)

	for i, f := range frames {
		ox, oy := (i%*cols)*cw+2, (i / *cols)*ch+10
		b := f.Bounds()
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				c := f.At(b.Min.X+x, b.Min.Y+y)
				xs := x
				if *dbl {
					xs = x * 2
				}
				w := 1
				if *dbl {
					w = 2
				}
				for zy := 0; zy < *zoom; zy++ {
					for zx := 0; zx < *zoom*w; zx++ {
						atlas.Set(ox+xs**zoom+zx, oy+y**zoom+zy, c)
					}
				}
			}
		}
		drawNum(atlas, ox, oy-9, i)
	}
	f, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, atlas); err != nil {
		panic(err)
	}
	fmt.Printf("%d 格 → %s（每格顯示 %dx%d，放大 %dx）\n", len(frames), *out, fw, fh, *zoom)
}

// decodeWithPalette 用指定的 6-bit EGA 調色盤解 .SHE。
func decodeWithPalette(data []byte, w, h int, pal [16]byte) ([]*image.RGBA, error) {
	rowBytes := w / 8
	frameBytes := rowBytes * h * 4
	if len(data)%frameBytes != 0 {
		return nil, fmt.Errorf("%d 不能被 frame 大小 %d 整除", len(data), frameBytes)
	}
	var out []*image.RGBA
	for i := 0; i+frameBytes <= len(data); i += frameBytes {
		img, err := gfx.DecodeEGAPlanar(data[i:i+frameBytes], w, h, gfx.EGAPlanesRowBlocks, &pal)
		if err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, nil
}

// decodeReversed 與 gfx.DecodeEGASpriteSheet 唯一的差別是 index 的 bit 順序：
// plane 0 進 bit 3 而不是 bit 0。用來檢驗檔案的 plane 順序是 B,G,R,I 還是 I,R,G,B。
func decodeReversed(data []byte, w, h int) ([]*image.RGBA, error) {
	rowBytes := w / 8
	frameBytes := rowBytes * h * 4
	if len(data)%frameBytes != 0 {
		return nil, fmt.Errorf("%d 不能被 frame 大小 %d 整除", len(data), frameBytes)
	}
	var out []*image.RGBA
	for i := 0; i+frameBytes <= len(data); i += frameBytes {
		d := data[i : i+frameBytes]
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for row := 0; row < h; row++ {
			for col := 0; col < rowBytes; col++ {
				var p [4]byte
				for plane := 0; plane < 4; plane++ {
					p[plane] = d[row*rowBytes*4+plane*rowBytes+col]
				}
				for bit := 0; bit < 8; bit++ {
					shift := uint(7 - bit)
					idx := byte(0)
					for plane := 0; plane < 4; plane++ {
						if (p[plane]>>shift)&1 != 0 {
							idx |= 1 << uint(3-plane)
						}
					}
					img.SetRGBA(col*8+bit, row, gfx.EGAPalette[idx])
				}
			}
		}
		out = append(out, img)
	}
	return out, nil
}

// 3x5 點陣數字，只為了標索引，不引外部字型。
var digits = [10][5]string{
	{"111", "101", "101", "101", "111"}, {"010", "010", "010", "010", "010"},
	{"111", "001", "111", "100", "111"}, {"111", "001", "111", "001", "111"},
	{"101", "101", "111", "001", "001"}, {"111", "100", "111", "001", "111"},
	{"111", "100", "111", "101", "111"}, {"111", "001", "001", "001", "001"},
	{"111", "101", "111", "101", "111"}, {"111", "101", "111", "001", "111"},
}

func drawNum(dst *image.RGBA, x, y, n int) {
	s := fmt.Sprintf("%d", n)
	for k, r := range s {
		d := digits[r-'0']
		for row := 0; row < 5; row++ {
			for col := 0; col < 3; col++ {
				if d[row][col] == '1' {
					dst.Set(x+k*4+col, y+row, color.RGBA{255, 255, 0, 255})
				}
			}
		}
	}
}
