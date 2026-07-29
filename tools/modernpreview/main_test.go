package main

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

func TestParseIndices(t *testing.T) {
	got := parseIndices("01, 14,62")
	for _, index := range []int{0x01, 0x14, 0x62} {
		if !got[index] {
			t.Fatalf("index 0x%02x missing from %#v", index, got)
		}
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
}

func TestApplyTerrainOverlays(t *testing.T) {
	dir := t.TempDir()
	wantNormal := color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}
	wantWinter := color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}
	writeSolid(t, filepath.Join(dir, "demon-23-plain.png"), wantNormal)
	writeSolid(t, filepath.Join(dir, "winter-23-plain.png"), wantWinter)
	// A candidate in the same directory must not leak into the explicitly approved set.
	writeSolid(t, filepath.Join(dir, "demon-17-coast.png"), color.RGBA{R: 0xff, A: 0xff})
	writeSolid(t, filepath.Join(dir, "winter-17-coast.png"), color.RGBA{R: 0xff, A: 0xff})

	normal := blankFrames()
	winter := blankFrames()
	applyTerrainOverlays(dir, normal, winter, map[int]bool{0x23: true})

	if got := normal[0x23].RGBAAt(0, 0); got != wantNormal {
		t.Fatalf("normal overlay = %#v, want %#v", got, wantNormal)
	}
	if got := winter[0x23].RGBAAt(0, 0); got != wantWinter {
		t.Fatalf("winter overlay = %#v, want %#v", got, wantWinter)
	}
	if got := normal[0x17].RGBAAt(0, 0); got.R != 0 {
		t.Fatalf("unapproved coast leaked into atlas: %#v", got)
	}
}

func blankFrames() []*image.RGBA {
	frames := make([]*image.RGBA, terrainFrames)
	for i := range frames {
		frames[i] = image.NewRGBA(image.Rect(0, 0, gfx.EGATileWidth, gfx.EGATileHeight))
	}
	return frames
}

func writeSolid(t *testing.T, path string, c color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, gfx.EGATileWidth, gfx.EGATileHeight))
	for y := 0; y < gfx.EGATileHeight; y++ {
		for x := 0; x < gfx.EGATileWidth; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	if err := gfx.SavePNG(path, img); err != nil {
		t.Fatal(err)
	}
}
