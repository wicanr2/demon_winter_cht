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
	in := flag.String("in", "workplace/orig/demwin/DEM_DATA/DEMON.SHP", "")
	out := flag.String("out", "workplace/dump/gfx/tileatlas-demon-shp.png", "")
	flag.Parse()
	data, err := os.ReadFile(*in)
	if err != nil { panic(err) }
	frames, err := gfx.DecodeCGASpriteSheet(data, 16, 16)
	if err != nil { panic(err) }
	const cols, zoom = 12, 3
	cw, ch := 16*zoom*2+4, 16*zoom+12
	rows := (len(frames) + cols - 1) / cols
	atlas := image.NewRGBA(image.Rect(0, 0, cols*cw, rows*ch))
	draw.Draw(atlas, atlas.Bounds(), &image.Uniform{color.RGBA{40, 40, 40, 255}}, image.Point{}, draw.Src)
	for i, f := range frames {
		ox, oy := (i%cols)*cw+2, (i/cols)*ch+10
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				c := f.At(x, y)
				for zy := 0; zy < zoom; zy++ {
					for zx := 0; zx < zoom*2; zx++ {
						atlas.Set(ox+x*zoom*2+zx, oy+y*zoom+zy, c)
					}
				}
			}
		}
		drawNum(atlas, ox, oy-9, i)
	}
	f, _ := os.Create(*out)
	defer f.Close()
	png.Encode(f, atlas)
	fmt.Printf("%d 格 → %s\n", len(frames), *out)
}

var digits = [10][5]string{
	{"111","101","101","101","111"},{"010","010","010","010","010"},
	{"111","001","111","100","111"},{"111","001","111","001","111"},
	{"101","101","111","001","001"},{"111","100","111","001","111"},
	{"111","100","111","101","111"},{"111","001","001","001","001"},
	{"111","101","111","101","111"},{"111","101","111","001","111"},
}

func drawNum(dst *image.RGBA, x, y, n int) {
	for k, r := range fmt.Sprintf("%d", n) {
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
