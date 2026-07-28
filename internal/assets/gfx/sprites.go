package gfx

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
)

// SpriteSheet 是戰鬥單位使用的原版 .SHP/.SHE 動畫表。
// EGA frame 已和地形一樣在載入時水平加倍成 32×28。
type SpriteSheet struct {
	mode   VideoMode
	w, h   int
	frames []*image.RGBA
}

func (s *SpriteSheet) Mode() VideoMode       { return s.mode }
func (s *SpriteSheet) FrameSize() (int, int) { return s.w, s.h }
func (s *SpriteSheet) Len() int              { return len(s.frames) }

func (s *SpriteSheet) Frame(i int) *image.RGBA {
	if i < 0 || i >= len(s.frames) {
		return nil
	}
	return s.frames[i]
}

// LoadSpriteSheetMode 載入 MONSTER/COMBAT 這類沒有副檔名的素材基名。
// frame 保留不透明黑底：原版就是把 sprite 索引寫進地圖緩衝後整格覆寫，
// 不是以透明遮罩疊在地形上。
func LoadSpriteSheetMode(dir, base string, mode VideoMode) (*SpriteSheet, error) {
	ext := ".SHP"
	if mode == ModeEGA {
		ext = ".SHE"
	}
	path := filepath.Join(dir, strings.TrimSuffix(base, filepath.Ext(base))+ext)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gfx: 讀取 sprite %s 失敗: %w", path, err)
	}

	var frames []*image.RGBA
	w, h := TileWidth, TileHeight
	if mode == ModeEGA {
		pal := GamePalette
		frames, err = decodeEGATiles(data, &pal)
		w, h = EGATileWidth, EGATileHeight
	} else {
		frames, err = DecodeCGASpriteSheet(data, TileWidth, TileHeight)
	}
	if err != nil {
		return nil, fmt.Errorf("gfx: 解碼 sprite %s 失敗: %w", path, err)
	}
	return &SpriteSheet{mode: mode, w: w, h: h, frames: frames}, nil
}
