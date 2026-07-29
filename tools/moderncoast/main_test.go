package main

import (
	"image"
	"image/color"
	"testing"
)

func TestParseTilesHexAndDeduplicate(t *testing.T) {
	got, err := parseTiles("17, 0x1a,17,3e")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x17, 0x1a, 0x3e}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestWaterMaskUsesBlueTopologyOnly(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 1))
	src.SetRGBA(0, 0, color.RGBA{0, 170, 255, 255})
	src.SetRGBA(1, 0, color.RGBA{255, 170, 0, 255})
	src.SetRGBA(2, 0, color.RGBA{255, 255, 255, 255})
	got := waterMask(src)
	if got[0][0] != 1 || got[0][1] != 0 || got[0][2] != 0 {
		t.Fatalf("unexpected mask: %v", got[0])
	}
}

func TestRenderProducesOpaqueRuntimeFrame(t *testing.T) {
	src := make([][]float64, 28)
	for y := range src {
		src[y] = make([]float64, 32)
		for x := range src[y] {
			if x < 16 {
				src[y][x] = 1
			}
		}
	}
	land := image.NewRGBA(image.Rect(0, 0, outW, outH))
	water := image.NewRGBA(image.Rect(0, 0, outW, outH))
	for y := 0; y < outH; y++ {
		for x := 0; x < outW; x++ {
			land.SetRGBA(x, y, color.RGBA{80, 120, 40, 255})
			water.SetRGBA(x, y, color.RGBA{5, 30, 90, 255})
		}
	}
	got := render(src, land, water, false)
	if got.Bounds() != image.Rect(0, 0, outW, outH) {
		t.Fatalf("bounds = %v", got.Bounds())
	}
	for y := 0; y < outH; y++ {
		for x := 0; x < outW; x++ {
			if got.RGBAAt(x, y).A != 255 {
				t.Fatalf("pixel (%d,%d) alpha = %d", x, y, got.RGBAAt(x, y).A)
			}
		}
	}
}
