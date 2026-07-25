package gfx

import (
	"fmt"
	"image"
)

// DecodeCGAPlanar320 把一段 CGA 2bpp、320 寬、IBM CGA mode 4/5 標準記憶體
// 佈局（偶數掃描線在前半段、奇數掃描線在後半段，各自線性排列，每列
// width/4 bytes、每 byte 4 個像素、bit 順序 MSB-first）的資料解成 RGBA。
//
// 這是 IBM CGA 顯示卡 0xB8000 framebuffer 的標準格式（非本專案臆測），
// 任何一份 CGA 硬體手冊都能查到：因為 CGA 每條掃描線只有一半的記憶體頻寬，
// 硬體把畫面切成偶數行／奇數行兩個 8000-byte 的 bank 交錯定址。
//
// 已驗證：OPEN.PIC（16000 B = 320×200÷4）用此函式解出的圖與
// workplace/dosbox/shots/05-cga-hang-open-pic.png 的開場惡魔插畫輪廓吻合。
func DecodeCGAPlanar320(data []byte, width, height int) (*image.RGBA, error) {
	rowBytes := width / 4
	halfRows := height / 2
	need := rowBytes * height
	if len(data) < need {
		return nil, fmt.Errorf("gfx: CGA data 太短: 需要 %d bytes(width=%d height=%d)，實際 %d", need, width, height, len(data))
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for row := 0; row < height; row++ {
		// 偶數行在前半段 bank，奇數行在後半段 bank；bank 內部依「螢幕上的
		// 偶/奇列序」線性排列（第 0 條偶數線在 bank 開頭，第 1 條偶數線接著…）。
		var bankOffset int
		var lineInBank int
		if row%2 == 0 {
			bankOffset = 0
			lineInBank = row / 2
		} else {
			bankOffset = rowBytes * halfRows
			lineInBank = row / 2
		}
		rowStart := bankOffset + lineInBank*rowBytes
		for col := 0; col < width; col++ {
			b := data[rowStart+col/4]
			shift := uint(6 - 2*(col%4)) // MSB-first：每 byte 4 像素，最左像素在最高位
			idx := (b >> shift) & 0x3
			img.SetRGBA(col, row, CGAPalette1High[idx])
		}
	}
	return img, nil
}

// DecodeCGASpriteSheet 把一份 .SHP 檔切成連續等尺寸 frame 逐一解碼(不做
// 偶/奇 bank 交錯——sprite frame 不是直接的硬體 framebuffer 內容，用線性
// 排列的 DecodeCGALinear320 邏輯)。frameW/frameH 是單一 frame 尺寸，用檔案
// 長度整除 frame 大小反推 frame 數。
func DecodeCGASpriteSheet(data []byte, frameW, frameH int) ([]*image.RGBA, error) {
	if frameW%4 != 0 {
		return nil, fmt.Errorf("gfx: CGA frame width 必須是 4 的倍數，得到 %d", frameW)
	}
	frameBytes := (frameW / 4) * frameH
	if frameBytes == 0 || len(data)%frameBytes != 0 {
		return nil, fmt.Errorf("gfx: %d bytes 資料無法被 frame 大小 %d(=%dx%d) 整除，尺寸假設錯誤", len(data), frameBytes, frameW, frameH)
	}
	n := len(data) / frameBytes
	frames := make([]*image.RGBA, 0, n)
	for i := 0; i < n; i++ {
		chunk := data[i*frameBytes : (i+1)*frameBytes]
		img, err := DecodeCGALinear320(chunk, frameW, frameH)
		if err != nil {
			return nil, fmt.Errorf("gfx: frame %d: %w", i, err)
		}
		frames = append(frames, img)
	}
	return frames, nil
}

// DecodeCGALinear320 是 DecodeCGAPlanar320 的「非交錯」對照版本：假設資料
// 就是單純由上到下線性排列的掃描線（不做偶/奇 bank 切換）。CGA 硬體本身
// 不會這樣存，但某些遊戲會在磁碟檔案裡先把 framebuffer 攤平成線性圖再存檔
// （載入時才重新交錯進顯存），所以拿來當候選假設之一，用截圖驗證後決定
// 用哪一版。
func DecodeCGALinear320(data []byte, width, height int) (*image.RGBA, error) {
	rowBytes := width / 4
	need := rowBytes * height
	if len(data) < need {
		return nil, fmt.Errorf("gfx: CGA data 太短: 需要 %d bytes(width=%d height=%d)，實際 %d", need, width, height, len(data))
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for row := 0; row < height; row++ {
		rowStart := row * rowBytes
		for col := 0; col < width; col++ {
			b := data[rowStart+col/4]
			shift := uint(6 - 2*(col%4))
			idx := (b >> shift) & 0x3
			img.SetRGBA(col, row, CGAPalette1High[idx])
		}
	}
	return img, nil
}
