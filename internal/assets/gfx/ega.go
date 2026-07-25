package gfx

import (
	"fmt"
	"image"
)

// EGAPlaneLayout 列舉本套件試過的 EGA 4-plane 資料佈局假設。哪一種對，
// 要靠輸出的 PNG 肉眼比對 DOSBox 截圖決定，見 docs/formats/graphics.md
// 「肉眼比對結果」一節。
type EGAPlaneLayout int

const (
	// EGAPlanesSequential：四個 plane 各自連續一大塊
	// （plane0 全部 rows，接著 plane1 全部 rows...）。EGA 顯示卡暫存器層級
	// 就是這樣定址（sequencer map mask 選 plane、同一段 A000 位址空間），
	// 是最貼近硬體暫存器語意的假設，也是「解碼器 + 已知截圖」流程裡第一個
	// 要試的候選。
	EGAPlanesSequential EGAPlaneLayout = iota
	// EGAPlanesRowInterleaved：每一列裡 4 個 plane 的 bytes 相鄰
	// （row0: p0 p1 p2 p3, row1: p0 p1 p2 p3, ...）。部分遊戲美術工具會用
	// 這種「每列交錯」佈局方便逐列 DMA。
	EGAPlanesRowInterleaved
	// EGAPlanesRowBlocks：每一列裡 4 個 plane「各自連續一小塊」（每塊
	// rowBytes 個 byte），四塊接在一起才是一列，下一列再重複同樣的
	// 4 個區塊（row0: plane0 全部 rowBytes bytes + plane1 全部 rowBytes
	// bytes + plane2 + plane3, row1 再重複...）。
	//
	// **已驗證**（.SHE 精靈圖，見 docs/re/07-sprite-blit.md）：直接反組譯
	// `FUN_217b_07cf`（217b 段 EGA 單一 sprite blit 常式）讀出來的真實佈局。
	// 來源指標在內層 28-row 迴圈裡每列只讀 4 bytes（一個 plane 的一列），
	// 但指標步進量是 16 bytes（跳過另外 3 個 plane 的同列資料），28 列跑完
	// 一輪 plane 後，指標淨移動量剛好 +4 bytes 開始下一個 plane —— 也就是
	// 「同一列的 4 個 plane 緊接著存」，跟 EGAPlanesRowInterleaved 的差別是
	// 交錯粒度：這裡是「一整個 rowBytes 區塊」而非「單一 byte」。
	EGAPlanesRowBlocks
)

// DecodeEGAPlanar 把 4-plane EGA 資料解成 RGBA。width 必須是 8 的倍數
// （每個 plane byte 對應 8 個像素）。palette 若為 nil 用標準 EGAPalette。
//
// 待驗證：這只是候選解碼器之一，plane 佈局、bit 順序都可能猜錯——
// 對錯要看輸出 PNG 是不是雜訊，並與 DOSBox 截圖比對。
func DecodeEGAPlanar(data []byte, width, height int, layout EGAPlaneLayout, palette *[16]byte) (*image.RGBA, error) {
	if width%8 != 0 {
		return nil, fmt.Errorf("gfx: EGA width 必須是 8 的倍數，得到 %d", width)
	}
	rowBytes := width / 8
	planeSize := rowBytes * height
	need := planeSize * 4
	if len(data) < need {
		return nil, fmt.Errorf("gfx: EGA data 太短: 需要 %d bytes(width=%d height=%d 4-plane)，實際 %d", need, width, height, len(data))
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	getPlaneByte := func(plane, row, col int) byte {
		switch layout {
		case EGAPlanesSequential:
			return data[plane*planeSize+row*rowBytes+col]
		case EGAPlanesRowInterleaved:
			return data[row*rowBytes*4+col*4+plane]
		case EGAPlanesRowBlocks:
			return data[row*rowBytes*4+plane*rowBytes+col]
		default:
			return 0
		}
	}

	for row := 0; row < height; row++ {
		for col := 0; col < rowBytes; col++ {
			var p [4]byte
			for plane := 0; plane < 4; plane++ {
				p[plane] = getPlaneByte(plane, row, col)
			}
			for bit := 0; bit < 8; bit++ {
				shift := uint(7 - bit)
				idx := byte(0)
				for plane := 0; plane < 4; plane++ {
					if (p[plane]>>shift)&1 != 0 {
						idx |= 1 << uint(plane)
					}
				}
				x := col*8 + bit
				var rgba = EGAPalette[idx]
				if palette != nil {
					rgba = EGAPalette[palette[idx]&0x0f]
				}
				img.SetRGBA(x, row, rgba)
			}
		}
	}
	return img, nil
}

// DecodeEGASpriteSheet 把一份 .SHE 檔案切成連續等尺寸 frame，逐一解碼。
// frameW/frameH 是單一 frame 的像素尺寸，資料裡沒有 frame 數量欄位——
// 檔案長度整除 (frameW/8*frameH*4) 即為 frame 數，除不盡就回傳 error
// （代表尺寸假設錯誤）。
//
// **已驗證**（2026-07-25，見 docs/re/07-sprite-blit.md）：直接反組譯
// `FUN_217b_07cf`（217b 段 EGA 單一 sprite blit 常式）讀出真實佈局是
// **frameW=32、frameH=28、layout=EGAPlanesRowBlocks**——frame 內部逐列排列、
// 每列 4 個 plane 各自連續一塊（不是逐 frame 四平面各自整塊，也不是整檔
// 四平面分塊）。呼叫方式：
// `DecodeEGASpriteSheet(data, 32, 28, EGAPlanesRowBlocks)`。
//
// **踩過的雷**：早期（2026-07-25 上午）在沒有讀 217b 段反組譯前，靠「檔案
// 大小整除、跟 CGA frame 數對上」試過 8 種佈局假設（frame 內自帶四平面
// sequential／整檔四平面 sequential／逐列交錯 1-byte 粒度／16x56 高度
// 縮放／32x16 不縮放…），全部輸出雜訊，細節見 docs/formats/graphics.md
// §4.2。真正的答案要讀 `217b:07cf` 的原始指令（不是 decompile，decompile
// 對 word/byte 指標單位有歧義）才對得出來：來源指標在 28-row 內層迴圈裡
// 每列只讀 4 bytes 卻步進 16 bytes，28 列跑完淨移動 -444(0x1bc)+448(0x1c0)
// = +4 bytes 才切到下一個 plane——這代表「同一列的 4 個 plane 緊接著存」，
// 且 sprite 尺寸是 32×28（不是先前用位元組數整除猜的 16×56），兩者總像素數
// 剛好都是 896，這正是「除法對得上、佈局/尺寸猜錯」的第三個案例（CGA
// 全螢幕圖偶奇交錯、CGA sprite 寬高對調之後的第三次教訓）。
func DecodeEGASpriteSheet(data []byte, frameW, frameH int, layout EGAPlaneLayout) ([]*image.RGBA, error) {
	if frameW%8 != 0 {
		return nil, fmt.Errorf("gfx: EGA frame width 必須是 8 的倍數，得到 %d", frameW)
	}
	frameBytes := (frameW / 8) * frameH * 4
	if frameBytes == 0 || len(data)%frameBytes != 0 {
		return nil, fmt.Errorf("gfx: %d bytes 資料無法被 frame 大小 %d(=%dx%d 4-plane) 整除，尺寸假設錯誤", len(data), frameBytes, frameW, frameH)
	}
	n := len(data) / frameBytes
	frames := make([]*image.RGBA, 0, n)
	for i := 0; i < n; i++ {
		chunk := data[i*frameBytes : (i+1)*frameBytes]
		img, err := DecodeEGAPlanar(chunk, frameW, frameH, layout, nil)
		if err != nil {
			return nil, fmt.Errorf("gfx: frame %d: %w", i, err)
		}
		frames = append(frames, img)
	}
	return frames, nil
}

// DecodeEGASpriteSheetGlobalPlanes 把整份 .SHE 當成「一張很高的 4-plane
// 圖」解碼（4 個 plane 是整個檔案等級的四大塊，不是逐 frame 各自四平面），
// 再依 frameH 切成等高的 frame。
//
// **這個函式對應的假設已被排除**（2026-07-25 更正，見
// docs/re/07-sprite-blit.md）：此處先前的文件註解誤植為「已驗證對
// MONSTER.SHE/COMBAT.SHE 解出清楚剪影」，但實際上這個假設（連同這裡的
// EGAPlanesSequential 佈局）跟其餘 7 種試過的假設一樣，輸出的都是雜訊，
// 見 docs/formats/graphics.md §4.2「整檔 4-plane sequential(global)」列。
// 真正解出來的佈局是 DecodeEGASpriteSheet(data, 32, 28, EGAPlanesRowBlocks)
// ——4 個 plane 的分塊粒度是「每一列」，不是「整個檔案」。保留這個函式僅供
// 對照／回歸測試用，**不要在新程式碼裡使用**。
func DecodeEGASpriteSheetGlobalPlanes(data []byte, frameW, frameH int) ([]*image.RGBA, error) {
	if frameW%8 != 0 {
		return nil, fmt.Errorf("gfx: EGA frame width 必須是 8 的倍數，得到 %d", frameW)
	}
	rowBytes := frameW / 8
	planeSize := len(data) / 4
	if planeSize == 0 || len(data)%4 != 0 || planeSize%rowBytes != 0 {
		return nil, fmt.Errorf("gfx: %d bytes 資料無法均分成 4 個 plane、每列 %d bytes，尺寸假設錯誤", len(data), rowBytes)
	}
	totalHeight := planeSize / rowBytes
	if totalHeight%frameH != 0 {
		return nil, fmt.Errorf("gfx: 整檔總高度 %d 列無法被 frame 高度 %d 整除，尺寸假設錯誤", totalHeight, frameH)
	}
	whole, err := DecodeEGAPlanar(data, frameW, totalHeight, EGAPlanesSequential, nil)
	if err != nil {
		return nil, err
	}
	n := totalHeight / frameH
	frames := make([]*image.RGBA, 0, n)
	for i := 0; i < n; i++ {
		f := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
		for y := 0; y < frameH; y++ {
			for x := 0; x < frameW; x++ {
				f.SetRGBA(x, y, whole.RGBAAt(x, i*frameH+y))
			}
		}
		frames = append(frames, f)
	}
	return frames, nil
}

// ParsePIEPalette 嘗試把 .PIE 檔開頭多出來的 16 bytes 當作「調色盤索引表」
// 解析：第 i 個 byte 是「邏輯色 i 對應到哪個實體 EGA 色碼（0-15）」，
// 對應 EGA 硬體的 Palette Register（AC index 0x00-0x0F 各存一個 6-bit
// 色碼，這裡假設只用到低 4 位、相容 16 色標準色盤）。
//
// 待驗證：也可能這 16 bytes 根本不是調色盤（比如是圖框尺寸/位置 header）。
// 用 PNG 輸出比對截圖决定。
func ParsePIEPalette(data []byte) (*[16]byte, []byte, error) {
	if len(data) < 16 {
		return nil, nil, fmt.Errorf("gfx: .PIE 檔太短，不足 16 bytes 表頭")
	}
	var pal [16]byte
	copy(pal[:], data[:16])
	return &pal, data[16:], nil
}
