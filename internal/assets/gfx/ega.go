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
// **踩過的雷**：一開始假設每個 frame 自己是獨立的「plane0 整塊+plane1 整塊
// +...」（跟 PIE 全螢幕圖一樣的佈局），套用在 .SHE 上輸出全是雜訊。
// 肉眼比對後改成 DecodeEGASpriteSheetGlobalPlanes（4 個 plane 是整個檔案
// 等級分塊，不是每個 frame 各自分塊）才解對，細節見 docs/formats/graphics.md。
// 這個函式保留給「frame 內自帶四平面」的假設（若某些 .SHE 檔案其實是這種
// 佈局，仍可能用得到），預設請用 DecodeEGASpriteSheetGlobalPlanes。
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
// 再依 frameH 切成等高的 frame。**已驗證**：對 MONSTER.SHE/COMBAT.SHE 等
// 檔案用這個函式解出清楚的怪物剪影，DecodeEGASpriteSheet(逐 frame 四平面)
// 則是雜訊，兩者對照就是本專案排除錯誤佈局假設的直接證據。
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
