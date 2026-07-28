package ui

import (
	"image"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

// SpriteSheet 是已上傳到 Ebiten 的原版戰鬥動畫表。
type SpriteSheet struct {
	src   *gfx.SpriteSheet
	atlas *ebiten.Image
	cols  int
	w, h  int
}

func NewSpriteSheet(src *gfx.SpriteSheet) *SpriteSheet {
	const cols = 16
	w, h := src.FrameSize()
	rows := (src.Len() + cols - 1) / cols
	rgba := image.NewRGBA(image.Rect(0, 0, cols*w, rows*h))
	for i := 0; i < src.Len(); i++ {
		x, y := (i%cols)*w, (i/cols)*h
		draw.Draw(rgba, image.Rect(x, y, x+w, y+h), src.Frame(i),
			src.Frame(i).Bounds().Min, draw.Src)
	}
	return &SpriteSheet{
		src: src, atlas: ebiten.NewImageFromImage(rgba),
		cols: cols, w: w, h: h,
	}
}

func (s *SpriteSheet) Mode() gfx.VideoMode { return s.src.Mode() }

func (s *SpriteSheet) Frame(i int) *ebiten.Image {
	if s == nil || i < 0 || i >= s.src.Len() {
		return nil
	}
	x, y := (i%s.cols)*s.w, (i/s.cols)*s.h
	return s.atlas.SubImage(image.Rect(x, y, x+s.w, y+s.h)).(*ebiten.Image)
}
