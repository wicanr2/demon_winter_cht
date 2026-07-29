package gfx

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
)

// LoadPNGTileset 載入逐列排列的 Modern EGA PNG atlas。
//
// PNG 必須能被 frame 尺寸整除、格數精確相符，且每個像素完全不透明。原版
// sprite 是整格覆寫而非 alpha 疊圖；讓半透明素材混進來會改變玩法辨識。
func LoadPNGTileset(path string, set TerrainSet, frameWidth, frameHeight, frames int) (*Tileset, error) {
	decoded, err := loadPNGFrames(path, frameWidth, frameHeight, frames)
	if err != nil {
		return nil, err
	}
	return &Tileset{
		set: set, mode: ModeEGA, w: frameWidth, h: frameHeight, frames: decoded,
	}, nil
}

// LoadPNGSpriteSheet 與 LoadPNGTileset 相同，但不帶地形集語意。
func LoadPNGSpriteSheet(path string, frameWidth, frameHeight, frames int) (*SpriteSheet, error) {
	decoded, err := loadPNGFrames(path, frameWidth, frameHeight, frames)
	if err != nil {
		return nil, err
	}
	return &SpriteSheet{
		mode: ModeEGA, w: frameWidth, h: frameHeight, frames: decoded,
	}, nil
}

func loadPNGFrames(path string, frameWidth, frameHeight, expected int) ([]*image.RGBA, error) {
	switch {
	case frameWidth <= 0 || frameHeight <= 0:
		return nil, fmt.Errorf("gfx: PNG atlas %s 的 frame 尺寸必須為正數", path)
	case expected <= 0:
		return nil, fmt.Errorf("gfx: PNG atlas %s 的預期格數必須為正數", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("gfx: 開啟 PNG atlas %s 失敗: %w", path, err)
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("gfx: 解碼 PNG atlas %s 失敗: %w", path, err)
	}
	b := src.Bounds()
	if b.Dx()%frameWidth != 0 || b.Dy()%frameHeight != 0 {
		return nil, fmt.Errorf(
			"gfx: PNG atlas %s 尺寸 %dx%d 不能被 frame %dx%d 整除",
			path, b.Dx(), b.Dy(), frameWidth, frameHeight)
	}
	cols, rows := b.Dx()/frameWidth, b.Dy()/frameHeight
	if got := cols * rows; got != expected {
		return nil, fmt.Errorf("gfx: PNG atlas %s 有 %d 格，預期 %d", path, got, expected)
	}

	out := make([]*image.RGBA, 0, expected)
	for frame := 0; frame < expected; frame++ {
		dst := image.NewRGBA(image.Rect(0, 0, frameWidth, frameHeight))
		ox := b.Min.X + (frame%cols)*frameWidth
		oy := b.Min.Y + (frame/cols)*frameHeight
		for y := 0; y < frameHeight; y++ {
			for x := 0; x < frameWidth; x++ {
				_, _, _, alpha := src.At(ox+x, oy+y).RGBA()
				if alpha != 0xffff {
					return nil, fmt.Errorf(
						"gfx: PNG atlas %s 第 %d 格 (%d,%d) 不是完全不透明",
						path, frame, x, y)
				}
				dst.SetRGBA(x, y, color.RGBAModel.Convert(src.At(ox+x, oy+y)).(color.RGBA))
			}
		}
		out = append(out, dst)
	}
	return out, nil
}
