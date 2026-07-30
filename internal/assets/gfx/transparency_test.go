package gfx

import (
	"image"
	"image/color"
	"testing"
)

func TestTransparentBackground_RemovesOnlyOpenBackground(t *testing.T) {
	black := color.RGBA{0, 0, 0, 0xff}
	red := color.RGBA{0xff, 0, 0, 0xff}
	src := image.NewRGBA(image.Rect(0, 0, 7, 7))
	for y := 0; y < 7; y++ {
		for x := 0; x < 7; x++ {
			src.SetRGBA(x, y, black)
		}
	}
	// 紅色本體、貼身黑框，以及被本體包住的黑色內部。
	for x := 2; x <= 4; x++ {
		src.SetRGBA(x, 2, red)
		src.SetRGBA(x, 4, red)
	}
	src.SetRGBA(2, 3, red)
	src.SetRGBA(4, 3, red)

	got := TransparentBackground(src, black)
	if a := got.RGBAAt(0, 0).A; a != 0 {
		t.Fatalf("遠離人物的背景 alpha=%d，want 0", a)
	}
	if a := got.RGBAAt(3, 3).A; a != 0xff {
		t.Fatalf("封閉的黑色內部 alpha=%d，want 255", a)
	}
	if a := got.RGBAAt(1, 3).A; a != 0xff {
		t.Fatalf("貼身的一圈黑色輪廓 alpha=%d，want 255", a)
	}
	if got.RGBAAt(3, 2) != red {
		t.Fatalf("彩色本體被改寫：got %v", got.RGBAAt(3, 2))
	}
}

func TestWalkingPartyFramesBecomeTransparentInBothHistoricalModes(t *testing.T) {
	black := color.RGBA{0, 0, 0, 0xff}
	partyFrames := []byte{24, 25, 27, 28, 30, 31, 33, 34}
	for _, mode := range []VideoMode{ModeEGA, ModeCGA} {
		ts, err := LoadTilesetMode(origDataDir(t), NormalTiles, mode)
		if err != nil {
			t.Fatalf("載入 mode %d：%v", mode, err)
		}
		for _, frame := range partyFrames {
			img := TransparentBackground(ts.Tile(frame), black)
			transparent, opaque := 0, 0
			for i := 3; i < len(img.Pix); i += 4 {
				if img.Pix[i] == 0 {
					transparent++
				} else {
					opaque++
				}
			}
			if transparent == 0 || opaque == 0 {
				t.Errorf("mode %d frame %d：透明=%d、不透明=%d，應同時存在",
					mode, frame, transparent, opaque)
			}
		}
	}
}
