package gfx

import (
	"os"
	"path/filepath"
	"testing"
)

// 開場標題畫面解得開，而且尺寸正好用完整個檔案。
func TestDecodeTitleScreen(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(origDataDir(t), "OPEN.PIE"))
	if err != nil {
		t.Skipf("找不到 OPEN.PIE：%v", err)
	}

	img, err := DecodeTitleScreen(data)
	if err != nil {
		t.Fatalf("DecodeTitleScreen: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != TitleScreenWidth || b.Dy() != TitleScreenHeight {
		t.Errorf("解出 %d×%d，預期 %d×%d",
			b.Dx(), b.Dy(), TitleScreenWidth, TitleScreenHeight)
	}

	// 尺寸必須把檔案用光。差一列就代表 stride 或高度猜錯，
	// 而那種錯誤解出來仍然「有圖」，只是整體斜掉。
	if want := 16 + TitleScreenWidth/8*4*TitleScreenHeight; len(data) != want {
		t.Errorf("檔案 %d bytes，尺寸推得的長度是 %d", len(data), want)
	}
}

// 相鄰列的相似度必須與已知正確的素材同級。
//
// 這條是找出 304 bytes/列 的那個判準本身：解錯 stride 的話，
// 相鄰列等於隨機取樣，差異會跳到 70 以上。
func TestTitleScreen_RowCoherence(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(origDataDir(t), "OPEN.PIE"))
	if err != nil {
		t.Skipf("找不到 OPEN.PIE：%v", err)
	}
	body := data[16:]
	stride := TitleScreenWidth / 8 * 4

	score := func(wb int) float64 {
		var tot, n int
		for r := 1; r < 300 && (r+1)*wb <= len(body); r++ {
			for i := 0; i < wb; i++ {
				d := int(body[(r-1)*wb+i]) - int(body[r*wb+i])
				if d < 0 {
					d = -d
				}
				tot += d
			}
			n += wb
		}
		if n == 0 {
			return 999
		}
		return float64(tot) / float64(n)
	}

	got := score(stride)
	if got > 60 {
		t.Errorf("正確 stride 的列間差異 %.1f，應與已知素材同級（< 60）", got)
	}
	// 差一個 byte 就該明顯變差 —— 證明這個極小值是尖的，不是碰巧。
	for _, off := range []int{-1, 1} {
		if near := score(stride + off); near < got+20 {
			t.Errorf("stride %d 的差異 %.1f 與正確值 %.1f 太接近，極小值不夠尖銳",
				stride+off, near, got)
		}
	}
}
