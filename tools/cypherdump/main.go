// cypherdump 把 CYPHER.SHP 解成 PNG（`docs/re/72`）。
//
// 1728 bytes ÷ 64 = 27 個 16×16 frame，推測是符文字型（26 字母 + 句點）。
// 依本專案硬規則，視覺產物一律 dump 出來肉眼比對，不接受「解碼沒報錯」。
package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

func main() {
	data, err := os.ReadFile("workplace/orig/demwin/DEM_DATA/CYPHER.SHP")
	if err != nil {
		panic(err)
	}
	frames, err := gfx.DecodeCGASpriteSheet(data, 16, 16)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d frames\n", len(frames))
	sheet := gfx.TileSpriteSheet(frames, 9)
	f, err := os.Create("workplace/dump/persist/82-cypher-font.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, sheet); err != nil {
		panic(err)
	}
}
