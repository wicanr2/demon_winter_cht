package gfx

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
)

// SavePNG 把解碼結果存成 PNG，供人眼比對 DOSBox 截圖用。目錄不存在會自動建立。
func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// TileSpriteSheet 把一系列同尺寸的 frame 依 cols 欄排成一張 sprite sheet，
// 方便一次肉眼看完整份 .SHP/.SHE 檔的所有 frame。
func TileSpriteSheet(frames []*image.RGBA, cols int) *image.RGBA {
	if len(frames) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	fw := frames[0].Bounds().Dx()
	fh := frames[0].Bounds().Dy()
	rows := (len(frames) + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*fw, rows*fh))
	for i, f := range frames {
		ox := (i % cols) * fw
		oy := (i / cols) * fh
		for y := 0; y < fh; y++ {
			for x := 0; x < fw; x++ {
				sheet.SetRGBA(ox+x, oy+y, f.RGBAAt(x, y))
			}
		}
	}
	return sheet
}

// zoomImage 把影像放大 factor 倍(最近鄰內插),方便肉眼細看小 sprite。
func zoomImage(src *image.RGBA, factor int) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*factor, b.Dy()*factor))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.RGBAAt(b.Min.X+x, b.Min.Y+y)
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					out.SetRGBA(x*factor+dx, y*factor+dy, c)
				}
			}
		}
	}
	return out
}
