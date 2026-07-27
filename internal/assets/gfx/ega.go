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
					// 調色盤值是 6 bit，不能遮成 4 bit 去索引 16 色表。
					rgba = EGAColor(palette[idx])
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
// **已驗證**：檔案內的 frame 是 **16x28、224 bytes**，佈局
// `EGAPlanesRowBlocks`（frame 內逐列排列、每列 4 個 plane 各自連續一塊）。
// 呼叫方式：`DecodeEGASpriteSheet(data, 16, 28, EGAPlanesRowBlocks)`。
// 六個 .SHE 都適用，CYPHER.SHE 不是特例。
//
// 各檔 frame 數與 CGA 對應檔完全相同：COMBAT 44、CYPHER 27、DEMON 102、
// MONSTER 240、SHIP 32、WINTER 102。
//
// **frame stride 0x1c0=448 是記憶體側的值，不是檔案格式。**
// `FUN_217b_07cf` 硬編碼的 448 沒有錯，但它描述的是載入時做過
// 水平 pixel doubling 之後、緩衝區裡的 frame：
//
//	1d9f:00bf  MOV word ptr [0x5226],0xe0   ; 224 = 檔案內 frame 大小
//	1d9f:0101  SHL AX,0x1
//	1d9f:0109  MOV [0x521a],AX              ; 448 = 記憶體內 frame 大小
//
// 載入器 `FUN_1d9f_0a8b` 只在副檔名為 "shE"/"SHE" 時呼叫 `FUN_217b_0adf`
// 就地加倍（每 byte 逐 bit 複製兩次）。加倍後結構同構——每列仍是 4 個 plane，
// 只是各自從 2 bytes 變 4 bytes——所以「檔案當 16x28 解」與「記憶體當
// 32x28 解」是同一張圖。
//
// **踩過的雷**：用 32x28 解檔案會看到「一格裡 2x2 四個小圖」，那是兩個 frame
// 左右各半錯位疊起來的假象。位元組數在 16x28 與 32x28 兩種讀法下都整除，
// 3.5x 規律也都成立，**算術完全分不出來**——這是本專案在 sprite 尺寸上
// 踩的第四個坑。定案靠的是遊戲自己宣告的 `[0x5226]` 常數加上肉眼比對。
func DecodeEGASpriteSheet(data []byte, frameW, frameH int, layout EGAPlaneLayout) ([]*image.RGBA, error) {
	return DecodeEGASpriteSheetPalette(data, frameW, frameH, layout, nil)
}

// DecodeEGASpriteSheetPalette 與 DecodeEGASpriteSheet 相同，但可以指定
// 6-bit 調色盤。
//
// **地形圖塊與精靈圖一定要傳 GamePalette**，不能用標準 16 色表 ——
// `.SHE` 檔本身不帶調色盤，而原版開機時把 16 個調色盤暫存器整組換掉了
// （見 GamePalette）。用標準表解出來的顏色會系統性地錯，而且**看起來像
// 解碼失敗的雜訊**：沙地變黃、水變紅、樹幹變青、海岸線變成紅綠色塊。
// 本專案為此把一整批正確解出來的地形圖塊誤判成「雜訊格」過。
func DecodeEGASpriteSheetPalette(data []byte, frameW, frameH int,
	layout EGAPlaneLayout, palette *[16]byte) ([]*image.RGBA, error) {

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
		img, err := DecodeEGAPlanar(chunk, frameW, frameH, layout, palette)
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
// 真正解出來的佈局是 DecodeEGASpriteSheet(data, 16, 28, EGAPlanesRowBlocks)
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

// 開場標題畫面 `OPEN.PIE` 的尺寸（2026-07-26 解出）。
//
// 這是本專案最後一個未解的素材，先前記錄是「不符 3.5× 規律、6 種候選尺寸
// 全部雜訊」。解法不是再猜一輪尺寸，而是**窮舉每一種可能的列寬，
// 用「相鄰列的位元組差異」量測哪一種最像圖**：
//
//	每列 304 bytes → 43.2   ← 尖銳極小值
//	每列 303 bytes → 72.8
//	每列 305 bytes → 73.5
//
// 43.2 與兩個已知正確的對照組同級（`OPEN.PIC` 44.9、`PIC1.PIE` 55.6），
// 而左右鄰居立刻跳到 73 —— 這是「找到正確 stride」的特徵，不是碰巧。
//
// 102144 ÷ 304 = 336 列整除。每列 4 個 plane 各 76 bytes（`EGAPlanesRowBlocks`）。
//
// **它不是半寬圖。** `docs/formats/graphics.md` 記「EGA 素材一律檔案存半寬、
// 顯示時寬度 ×2」—— 全螢幕標題是例外：608×2 = 1216 超過任何 EGA 模式，
// 而且照 608 直接解出來的字母比例正常（半寬圖會橫向擠成一半）。
//
// 肉眼驗收：解出來是「DEMON'S WINTER」標題、惡魔、SSI 與 NOVOTRADE 標誌，
// 與 CGA 版 `OPEN.PIC`（320×200）是同一張美術。兩版都有大量抖色雜點，
// 那是美術本身，不是解碼問題。
const (
	TitleScreenWidth  = 608
	TitleScreenHeight = 336
)

// 人像框（`PIC1–6.PIE`／`PRIEST.PIE`／`SHAMEN.PIE`／`THANATOS.PIE`，18,160 B）。
//
// **佈局與標題畫面不同。** 人像框是 plane-major（四個 plane 各自連續一大塊），
// 標題畫面是每列 4 個 plane 各一小塊。同一個副檔名兩種佈局 ——
// 拿其中一種去解另一種會得到雜訊，而且看起來像「尺寸猜錯」。
const (
	PortraitWidth  = 144
	PortraitHeight = 252
)

// DecodePortrait 解人像框。
func DecodePortrait(data []byte) (*image.RGBA, error) {
	pal, body, err := ParsePIEPalette(data)
	if err != nil {
		return nil, err
	}
	want := PortraitWidth / 8 * PortraitHeight * 4
	if len(body) != want {
		return nil, fmt.Errorf("gfx: 人像框去掉表頭後 %d bytes，預期 %d（%d×%d）",
			len(body), want, PortraitWidth, PortraitHeight)
	}
	return DecodeEGAPlanar(body, PortraitWidth, PortraitHeight,
		EGAPlanesSequential, pal)
}

// DecodeTitleScreen 解開場標題畫面（`OPEN.PIE`）。
//
// ⚠ 這個檔**沒有被任何執行檔引用**。`DEMON.INT` 的檔名表列的是
// `TITLE.PIC`，而那個檔不在這份 dump 裡（見 docs/formats/graphics.md §5）。
// 解得出來、也確實是這款遊戲的標題畫面，但原版怎麼載入它仍是未解的。
func DecodeTitleScreen(data []byte) (*image.RGBA, error) {
	pal, body, err := ParsePIEPalette(data)
	if err != nil {
		return nil, err
	}
	want := TitleScreenWidth / 8 * 4 * TitleScreenHeight
	if len(body) != want {
		return nil, fmt.Errorf("gfx: OPEN.PIE 去掉表頭後 %d bytes，預期 %d（%d×%d，每列 4 個 plane）",
			len(body), want, TitleScreenWidth, TitleScreenHeight)
	}
	return DecodeEGAPlanar(body, TitleScreenWidth, TitleScreenHeight,
		EGAPlanesRowBlocks, pal)
}
