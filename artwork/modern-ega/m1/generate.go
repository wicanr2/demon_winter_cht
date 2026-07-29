//go:build ignore

// M1-B bounded runtime study.
//
// Every output pixel is authored by the primitives below. The generator reads no
// original game data and contains no copied bitmap. Run:
//
//	go run artwork/modern-ega/m1/generate.go
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

const (
	w = 32
	h = 28
)

var p = struct {
	void, water0, water1, water2 color.RGBA
	earth0, earth1, earth2       color.RGBA
	grass0, grass1, grass2       color.RGBA
	trunk0, trunk1               color.RGBA
	stone0, stone1, stone2       color.RGBA
	snow0, snow1, snow2          color.RGBA
	roof0, roof1, gold           color.RGBA
	cloth, skin, steel           color.RGBA
}{
	void:   rgba(0x12, 0x15, 0x1d),
	water0: rgba(0x13, 0x35, 0x59), water1: rgba(0x1f, 0x58, 0x7b), water2: rgba(0x66, 0xc9, 0xd4),
	earth0: rgba(0x6c, 0x4c, 0x2d), earth1: rgba(0xaa, 0x78, 0x3f), earth2: rgba(0xd8, 0xb4, 0x5a),
	grass0: rgba(0x25, 0x68, 0x49), grass1: rgba(0x4f, 0xa8, 0x62), grass2: rgba(0x83, 0xc7, 0x72),
	trunk0: rgba(0x3b, 0x2a, 0x28), trunk1: rgba(0x82, 0x4d, 0x35),
	stone0: rgba(0x38, 0x47, 0x56), stone1: rgba(0x6f, 0x7f, 0x8b), stone2: rgba(0xb7, 0xb4, 0xaa),
	snow0: rgba(0x6f, 0x8b, 0xa3), snow1: rgba(0xb5, 0xc9, 0xd5), snow2: rgba(0xe8, 0xdf, 0xc8),
	roof0: rgba(0x62, 0x24, 0x31), roof1: rgba(0xb6, 0x39, 0x42), gold: rgba(0xed, 0xce, 0x68),
	cloth: rgba(0x4f, 0x78, 0xb8), skin: rgba(0xe0, 0xa6, 0x87), steel: rgba(0xd8, 0xdd, 0xdd),
}

type tileSpec struct {
	file, label string
	index       int
	draw        func(bool, int) *image.RGBA
	variant     int
}

func main() {
	specs := []tileSpec{
		{"plain", "plain", 0x23, plain, 0},
		{"deep-water-a", "deep water A", 0x14, deepWater, 0},
		{"deep-water-b", "deep water B", 0x62, deepWater, 1},
		{"coast-nw-corner", "NW coast corner", 0x17, coastNW, 0},
		{"forest", "forest", 0x01, forest, 0},
		{"mountain", "mountain", 0x63, mountain, 0},
		{"town", "town", 0x2e, town, 0},
		{"party-north-a", "party north A", 0x1e, partyNorth, 0},
		{"party-north-b", "party north B", 0x1f, partyNorth, 1},
	}
	outDir := filepath.Join("artwork", "modern-ega", "m1", "tiles")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}
	for _, winter := range []bool{false, true} {
		season := "demon"
		if winter {
			season = "winter"
		}
		for _, s := range specs {
			img := s.draw(winter, s.variant)
			path := filepath.Join(outDir, fmt.Sprintf("%s-%02x-%s.png", season, s.index, s.file))
			writePNG(path, img)
		}
	}
	writeSheet(filepath.Join("artwork", "modern-ega", "m1", "m1-b-contact-sheet.png"), specs)
	writeContinuityProof(filepath.Join("artwork", "modern-ega", "m1", "m1-b-continuity-proof.png"))
}

func rgba(r, g, b byte) color.RGBA { return color.RGBA{r, g, b, 0xff} }

func baseLand(winter bool) color.RGBA {
	if winter {
		return p.snow1
	}
	return p.earth2
}

func blank(c color.RGBA) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return dst
}

func rect(dst *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	draw.Draw(dst, image.Rect(x0, y0, x1, y1), &image.Uniform{c}, image.Point{}, draw.Src)
}

func px(dst *image.RGBA, x, y int, c color.RGBA) {
	if image.Pt(x, y).In(dst.Bounds()) {
		dst.SetRGBA(x, y, c)
	}
}

func hline(dst *image.RGBA, y, x0, x1 int, c color.RGBA) {
	for x := x0; x <= x1; x++ {
		px(dst, x, y, c)
	}
}

func plain(winter bool, _ int) *image.RGBA {
	dst := blank(baseLand(winter))
	// No mark enters the two-pixel continuity band. Broad 2×1 clusters are
	// deliberately lower contrast than actors; there is no repeated bright grass confetti.
	dark, light := p.earth1, p.grass1
	if winter {
		dark, light = p.snow0, p.snow2
	}
	for _, q := range [][2]int{{5, 6}, {18, 5}, {11, 15}, {25, 19}, {5, 23}} {
		rect(dst, q[0], q[1], q[0]+3, q[1]+1, dark)
	}
	for _, q := range [][2]int{{23, 9}, {7, 18}, {17, 23}} {
		rect(dst, q[0], q[1], q[0]+2, q[1]+1, light)
	}
	return dst
}

func deepWater(_ bool, variant int) *image.RGBA {
	dst := blank(p.water0)
	// A two-pixel quiet border makes all four ports identical in A and B. Wave
	// phases may differ inside the tile without creating a hard color seam.
	ys := []int{5, 15, 25}
	if variant != 0 {
		ys = []int{8, 18}
	}
	for _, y := range ys {
		hline(dst, y, 2, 29, p.water1)
		hline(dst, y+1, 5, 12, p.water2)
		hline(dst, y+1, 21, 27, p.water2)
	}
	return dst
}

func coastNW(winter bool, _ int) *image.RGBA {
	dst := blank(baseLand(winter))
	// Exact ports:
	// top and left = water; right changes water→land at y=10/11;
	// bottom changes water→land at x=10/11. This is a NW water corner only.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x+y < 20 {
				dst.SetRGBA(x, y, p.water0)
			} else if x+y < 23 {
				dst.SetRGBA(x, y, p.water1)
			}
		}
	}
	for x, y := 2, 17; x <= 17; x, y = x+3, y-3 {
		px(dst, x, y, p.water2)
		px(dst, x+1, y, p.water2)
	}
	return dst
}

func forest(winter bool, _ int) *image.RGBA {
	dst := blank(baseLand(winter))
	leaf0, leaf1, leaf2 := p.grass0, p.grass1, p.grass2
	if winter {
		leaf0, leaf1, leaf2 = p.stone0, p.snow0, p.snow2
	}
	rect(dst, 14, 11, 18, 25, p.trunk0)
	rect(dst, 17, 12, 20, 24, p.trunk1)
	// One large crown, three-value lighting, two-pixel land continuity band.
	rect(dst, 7, 6, 25, 16, leaf0)
	rect(dst, 10, 3, 22, 19, leaf0)
	rect(dst, 5, 9, 27, 14, leaf0)
	rect(dst, 9, 5, 21, 12, leaf1)
	rect(dst, 7, 9, 15, 14, leaf1)
	rect(dst, 11, 5, 17, 9, leaf2)
	return dst
}

func mountain(winter bool, _ int) *image.RGBA {
	dst := blank(baseLand(winter))
	dark, mid, light := p.stone0, p.stone1, p.stone2
	if winter {
		dark, mid, light = p.stone0, p.snow0, p.snow2
	}
	// 4 px quiet border preserves plain adjacency. Large asymmetric faces survive 32×28.
	for y := 4; y <= 24; y++ {
		span := (y - 3) * 11 / 10
		for x := 16 - span; x <= 16+span; x++ {
			c := dark
			if x < 16 {
				c = mid
			}
			if x < 12+(y-4)/3 {
				c = light
			}
			px(dst, x, y, c)
		}
	}
	return dst
}

func town(winter bool, _ int) *image.RGBA {
	dst := blank(baseLand(winter))
	// Town fills 24×21, not a miniature concept-painting. Door is the focal read.
	rect(dst, 5, 10, 27, 25, p.stone0)
	rect(dst, 7, 11, 25, 24, p.stone1)
	for y := 4; y <= 11; y++ {
		x0 := 16 - (y-4)*2
		x1 := 16 + (y-4)*2
		rect(dst, x0, y, x1+1, y+1, p.roof0)
		if y < 9 {
			rect(dst, x0+2, y, x1, y+1, p.roof1)
		}
	}
	rect(dst, 13, 16, 20, 24, p.void)
	rect(dst, 15, 17, 18, 24, p.trunk1)
	rect(dst, 8, 14, 11, 18, p.gold)
	return dst
}

func partyNorth(winter bool, variant int) *image.RGBA {
	dst := blank(p.void)
	if winter {
		rect(dst, 0, 25, 32, 28, p.snow0)
	}
	// 18×24 actor, centered on x=16 with feet at y=25. Variant shifts one leg,
	// not the anchor. This fixes the direct-downscale study's undersized actor.
	rect(dst, 12, 3, 20, 8, p.skin)
	rect(dst, 11, 8, 21, 18, p.cloth)
	rect(dst, 8, 9, 11, 19, p.steel)
	rect(dst, 21, 9, 24, 19, p.steel)
	rect(dst, 9, 9, 11, 11, p.gold)
	rect(dst, 21, 9, 23, 11, p.gold)
	if variant == 0 {
		rect(dst, 12, 18, 16, 25, p.stone1)
		rect(dst, 17, 18, 21, 24, p.stone1)
	} else {
		rect(dst, 11, 18, 15, 24, p.stone1)
		rect(dst, 17, 18, 22, 25, p.stone1)
	}
	// North-facing helmet/crest is wider than the face and reads at native size.
	rect(dst, 11, 2, 21, 4, p.steel)
	rect(dst, 15, 0, 18, 3, p.roof1)
	return dst
}

func writeSheet(path string, specs []tileSpec) {
	const scale = 4
	cellW, cellH := w*scale, h*scale
	dst := image.NewRGBA(image.Rect(0, 0, len(specs)*cellW, 2*cellH))
	for row, winter := range []bool{false, true} {
		for col, s := range specs {
			src := s.draw(winter, s.variant)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					c := src.RGBAAt(x, y)
					rect(dst, col*cellW+x*scale, row*cellH+y*scale,
						col*cellW+(x+1)*scale, row*cellH+(y+1)*scale, c)
				}
			}
		}
	}
	writePNG(path, dst)
}

func writeContinuityProof(path string) {
	const scale = 4
	// Four 3×3 patches: DEMON plain, WINTER plain, alternating ocean,
	// and a land patch with forest/mountain/town surrounded by plain.
	dst := image.NewRGBA(image.Rect(0, 0, 4*3*w*scale, 3*h*scale))
	drawTile := func(src *image.RGBA, gx, gy int) {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := src.RGBAAt(x, y)
				rect(dst, (gx*w+x)*scale, (gy*h+y)*scale,
					(gx*w+x+1)*scale, (gy*h+y+1)*scale, c)
			}
		}
	}
	for patch := 0; patch < 4; patch++ {
		for y := 0; y < 3; y++ {
			for x := 0; x < 3; x++ {
				var src *image.RGBA
				switch patch {
				case 0:
					src = plain(false, 0)
				case 1:
					src = plain(true, 0)
				case 2:
					src = deepWater(false, (x+y)&1)
				default:
					src = plain(false, 0)
					if x == 1 && y == 0 {
						src = forest(false, 0)
					} else if x == 1 && y == 1 {
						src = mountain(false, 0)
					} else if x == 1 && y == 2 {
						src = town(false, 0)
					}
				}
				drawTile(src, patch*3+x, y)
			}
		}
	}
	writePNG(path, dst)
}

func writePNG(path string, img image.Image) {
	if filepath.Base(filepath.Dir(path)) == "tiles" {
		if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
			panic(fmt.Sprintf("%s: bounds = %v, want 32x28", path, img.Bounds()))
		}
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a != 0xffff {
					panic(fmt.Sprintf("%s: translucent pixel at %d,%d", path, x, y))
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	fmt.Println(path)
}
