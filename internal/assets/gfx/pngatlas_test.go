package gfx

import (
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"
)

func writePNGAtlas(t *testing.T, img image.Image) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "atlas.png")
	if err := SavePNG(path, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadPNGAtlasSlicesRowMajor(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	colours := []color.RGBA{
		{R: 10, A: 255}, {G: 20, A: 255}, {B: 30, A: 255}, {R: 40, G: 40, A: 255},
	}
	for i, c := range colours {
		ox, oy := (i%2)*2, i/2
		for y := 0; y < 1; y++ {
			for x := 0; x < 2; x++ {
				img.SetRGBA(ox+x, oy+y, c)
			}
		}
	}
	path := writePNGAtlas(t, img)

	tiles, err := LoadPNGTileset(path, NormalTiles, 2, 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	if tiles.Len() != 4 {
		t.Fatalf("格數 = %d，預期 4", tiles.Len())
	}
	for i, want := range colours {
		if got := tiles.Tile(byte(i)).RGBAAt(0, 0); got != want {
			t.Errorf("格 %d = %#v，預期 %#v", i, got, want)
		}
	}
}

func TestLoadPNGAtlasRejectsWrongGeometryCountAndAlpha(t *testing.T) {
	t.Run("geometry", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 3, 2))
		if _, err := LoadPNGSpriteSheet(writePNGAtlas(t, img), 2, 1, 3); err == nil ||
			!strings.Contains(err.Error(), "不能被") {
			t.Fatalf("錯誤 = %v，預期尺寸整除錯誤", err)
		}
	})
	t.Run("count", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 4, 2))
		for y := 0; y < 2; y++ {
			for x := 0; x < 4; x++ {
				img.SetRGBA(x, y, color.RGBA{A: 255})
			}
		}
		if _, err := LoadPNGSpriteSheet(writePNGAtlas(t, img), 2, 1, 3); err == nil ||
			!strings.Contains(err.Error(), "有 4 格") {
			t.Fatalf("錯誤 = %v，預期格數錯誤", err)
		}
	})
	t.Run("alpha", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 2, 1))
		img.SetRGBA(0, 0, color.RGBA{R: 1, A: 255})
		img.SetRGBA(1, 0, color.RGBA{R: 1, A: 254})
		if _, err := LoadPNGSpriteSheet(writePNGAtlas(t, img), 2, 1, 1); err == nil ||
			!strings.Contains(err.Error(), "不是完全不透明") {
			t.Fatalf("錯誤 = %v，預期 alpha 錯誤", err)
		}
	})
}
