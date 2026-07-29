// moderncoast 依原版 EGA 圖塊的水陸拓樸，產生 Modern Icon 岸線圖。
//
// 它不放大或重採樣原版美術：原版只提供每個 tile index 的水陸遮罩；
// 水面、地表與岸緣全部由 Modern Icon 素材重新合成。這讓相鄰索引仍保持
// 原版地圖拓樸，又不會把 32×28 點陣圖偽裝成現代重繪。
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

const (
	outW = 64
	outH = 56
)

func main() {
	orig := flag.String("orig", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	normalLand := flag.String("normal-land", "", "正常地表 64×56 PNG")
	normalWater := flag.String("normal-water", "", "正常水面 64×56 PNG")
	winterLand := flag.String("winter-land", "", "冬季地表 64×56 PNG")
	winterWater := flag.String("winter-water", "", "冬季水面 64×56 PNG")
	out := flag.String("out", "", "輸出目錄")
	tileList := flag.String("tiles", "1a,1d,20,3b,3c,3d,3e", "十六進位 tile 清單")
	flag.Parse()

	if *normalLand == "" || *normalWater == "" || *winterLand == "" ||
		*winterWater == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	tiles, err := parseTiles(*tileList)
	check(err)
	base, err := gfx.LoadTilesetMode(*orig, gfx.NormalTiles, gfx.ModeEGA)
	check(err)
	nLand := loadPNG(*normalLand)
	nWater := loadPNG(*normalWater)
	wLand := loadPNG(*winterLand)
	wWater := loadPNG(*winterWater)
	checkSize("normal-land", nLand)
	checkSize("normal-water", nWater)
	checkSize("winter-land", wLand)
	checkSize("winter-water", wWater)
	check(os.MkdirAll(*out, 0o755))

	for _, tile := range tiles {
		src := base.Tile(tile)
		if src == nil {
			check(fmt.Errorf("tile 0x%02x 不存在", tile))
		}
		mask := waterMask(src)
		writeCoast(filepath.Join(*out, fmt.Sprintf("normal-coast-%02x.png", tile)),
			render(mask, nLand, nWater, false))
		writeCoast(filepath.Join(*out, fmt.Sprintf("winter-coast-%02x.png", tile)),
			render(mask, wLand, wWater, true))
		fmt.Printf("0x%02x → normal/winter\n", tile)
	}
}

func parseTiles(s string) ([]byte, error) {
	var out []byte
	seen := map[byte]bool{}
	for _, raw := range strings.Split(s, ",") {
		raw = strings.TrimSpace(raw)
		raw = strings.TrimPrefix(raw, "0x")
		n, err := strconv.ParseUint(raw, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("無效 tile %q: %w", raw, err)
		}
		v := byte(n)
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("tile 清單是空的")
	}
	return out, nil
}

func loadPNG(path string) image.Image {
	f, err := os.Open(path)
	check(err)
	defer f.Close()
	img, err := png.Decode(f)
	check(err)
	return img
}

func checkSize(name string, img image.Image) {
	if img.Bounds().Dx() != outW || img.Bounds().Dy() != outH {
		check(fmt.Errorf("%s 是 %dx%d，必須是 %dx%d", name,
			img.Bounds().Dx(), img.Bounds().Dy(), outW, outH))
	}
}

// waterMask 只抽取藍／青色水域。白浪與岸沙不直接抄入新圖；
// 它們會由 render 依平滑後的水陸邊界重新繪製。
func waterMask(src image.Image) [][]float64 {
	b := src.Bounds()
	raw := make([][]float64, b.Dy())
	for y := range raw {
		raw[y] = make([]float64, b.Dx())
		for x := range raw[y] {
			r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			r8, g8, b8 := byte(r>>8), byte(g>>8), byte(bl>>8)
			if r8 < 80 && b8 >= 150 && g8 >= 70 {
				raw[y][x] = 1
			}
		}
	}
	return raw
}

func render(src [][]float64, land, water image.Image, winter bool) *image.RGBA {
	mask := make([][]float64, outH)
	for y := 0; y < outH; y++ {
		mask[y] = make([]float64, outW)
		for x := 0; x < outW; x++ {
			sx := (float64(x)+0.5)*float64(len(src[0]))/outW - 0.5
			sy := (float64(y)+0.5)*float64(len(src))/outH - 0.5
			mask[y][x] = bilinear(src, sx, sy)
		}
	}
	// 原版遮罩是 32×28 的階梯輪廓。這裡只保留其海陸拓樸，經兩次
	// 半徑 3 的 separable blur 重建成連續海灣；不能把階梯直接放大後
	// 稱為 Modern Icon。
	mask = blurMask(mask, 3, 2)

	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	for y := 0; y < outH; y++ {
		for x := 0; x < outW; x++ {
			m := smoothstep(0.34, 0.66, mask[y][x])
			lc := rgbaAt(land, x, y)
			wc := rgbaAt(water, x, y)
			c := mix(lc, wc, m)

			// 在陸側重畫 2–3 px 岸緣；冬季使用覆霜岩岸，正常版使用暖沙。
			if m < 0.55 {
				d := distanceToWater(mask, x, y, 4)
				if d >= 0 && d <= 3.25 {
					shore := color.RGBA{R: 210, G: 176, B: 94, A: 255}
					if winter {
						shore = color.RGBA{R: 205, G: 211, B: 204, A: 255}
					}
					noise := uint8((x*17 + y*29 + (x*y)%19) % 17)
					shore.R = clamp8(int(shore.R) + int(noise) - 8)
					shore.G = clamp8(int(shore.G) + int(noise)/2 - 4)
					c = mix(c, shore, 0.78*(1-d/4))
				}
			}
			dst.SetRGBA(x, y, c)
		}
	}
	return dst
}

func blurMask(src [][]float64, radius, passes int) [][]float64 {
	cur := src
	for pass := 0; pass < passes; pass++ {
		h := make([][]float64, outH)
		for y := 0; y < outH; y++ {
			h[y] = make([]float64, outW)
			for x := 0; x < outW; x++ {
				var sum, weight float64
				for dx := -radius; dx <= radius; dx++ {
					xx := x + dx
					if xx < 0 {
						xx = 0
					}
					if xx >= outW {
						xx = outW - 1
					}
					w := float64(radius + 1 - abs(dx))
					sum += cur[y][xx] * w
					weight += w
				}
				h[y][x] = sum / weight
			}
		}
		v := make([][]float64, outH)
		for y := 0; y < outH; y++ {
			v[y] = make([]float64, outW)
			for x := 0; x < outW; x++ {
				var sum, weight float64
				for dy := -radius; dy <= radius; dy++ {
					yy := y + dy
					if yy < 0 {
						yy = 0
					}
					if yy >= outH {
						yy = outH - 1
					}
					w := float64(radius + 1 - abs(dy))
					sum += h[yy][x] * w
					weight += w
				}
				v[y][x] = sum / weight
			}
		}
		cur = v
	}
	return cur
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func bilinear(a [][]float64, x, y float64) float64 {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	tx, ty := x-float64(x0), y-float64(y0)
	sample := func(px, py int) float64 {
		if px < 0 {
			px = 0
		}
		if py < 0 {
			py = 0
		}
		if px >= len(a[0]) {
			px = len(a[0]) - 1
		}
		if py >= len(a) {
			py = len(a) - 1
		}
		return a[py][px]
	}
	top := sample(x0, y0)*(1-tx) + sample(x0+1, y0)*tx
	bot := sample(x0, y0+1)*(1-tx) + sample(x0+1, y0+1)*tx
	return top*(1-ty) + bot*ty
}

func distanceToWater(mask [][]float64, x, y, radius int) float64 {
	best := -1.0
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			xx, yy := x+dx, y+dy
			if xx < 0 || yy < 0 || xx >= outW || yy >= outH || mask[yy][xx] < 0.58 {
				continue
			}
			d := math.Hypot(float64(dx), float64(dy))
			if best < 0 || d < best {
				best = d
			}
		}
	}
	return best
}

func smoothstep(a, b, v float64) float64 {
	v = math.Max(0, math.Min(1, (v-a)/(b-a)))
	return v * v * (3 - 2*v)
}

func rgbaAt(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(img.Bounds().Min.X+x, img.Bounds().Min.Y+y).RGBA()
	return color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	f := func(x, y uint8) uint8 {
		return clamp8(int(math.Round(float64(x)*(1-t) + float64(y)*t)))
	}
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func writeCoast(path string, img image.Image) {
	f, err := os.Create(path)
	check(err)
	defer f.Close()
	check(png.Encode(f, img))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "moderncoast:", err)
		os.Exit(1)
	}
}
