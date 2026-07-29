package gfx

import (
	"image"
	"testing"
)

func TestModernizeTilesetPreservesTopology(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 2, 1))
	frame.SetRGBA(0, 0, EGAColor(GamePalette[0]))
	frame.SetRGBA(1, 0, EGAColor(GamePalette[15]))
	src := &Tileset{set: NormalTiles, mode: ModeEGA, w: 2, h: 1, frames: []*image.RGBA{frame}}
	got := ModernizeTileset(src)
	if got.Len() != 1 || got.Mode() != ModeEGA {
		t.Fatalf("拓樸／模式漂移：len=%d mode=%v", got.Len(), got.Mode())
	}
	if w, h := got.FrameSize(); w != 2 || h != 1 {
		t.Fatalf("frame = %dx%d", w, h)
	}
	if c := got.Tile(0).RGBAAt(1, 0); c != modernEGAPalette[15] {
		t.Fatalf("亮色映射 = %#v，預期 %#v", c, modernEGAPalette[15])
	}
	if src.Tile(0).RGBAAt(1, 0) == got.Tile(0).RGBAAt(1, 0) {
		t.Fatal("轉換就地改壞來源，或沒有套用新色盤")
	}
}
