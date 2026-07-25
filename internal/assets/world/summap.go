package world

import (
	"fmt"
	"os"
)

// sumMapSegmentCount 是 SUM.MAP 內打包的子地圖數量：23 個變長 RLE 區塊
// 首尾相接、無 padding 串接而成（已驗證，見 docs/re/03-audio-and-resources.md
// §3.2：23 筆 size 加總 = 15,743，等於 SUM.MAP 實際檔案大小）。
const sumMapSegmentCount = 23

// sumMapIDs、sumMapSizes 是 23 個子地圖的 ID／位元組長度表，數值取自
// DEMON.INT 反組譯結果（位址 31f0:2488 = ID 表、31f0:24b6 = size 表，
// 均為 int16／uint16、23 筆，見 docs/re/03-audio-and-resources.md §3.2）。
//
// 寫死成常數表而非執行期從 SUM.MAP 本身推導，理由：
//  1. 這兩張表本身就位在 DEMON.INT（程式邏輯所在的執行檔），不在 SUM.MAP
//     資料檔內——SUM.MAP 檔案本身完全不含自我描述的 index／size 資訊，
//     沒有「從資料推導」這個選項；唯一的資料來源就是反組譯讀出的常數。
//  2. 這組數字已有可重跑、獨立的正確性驗證（size 總和精確等於檔案位元組數，
//     機率上不可能是巧合，見 docs/re/03 §3.2），比「每次執行期重新解析
//     DEMON.INT」更穩健，也不需要在這個套件裡額外依賴 DEMON.INT 檔案。
var sumMapIDs = [sumMapSegmentCount]int{
	12, 13, 2, 21, 22, 23, 25, 33, 34, 35, 36, 4, 41, 43, 44, 45, 51, 52, 54, 55, 56, 64, 66,
}

var sumMapSizes = [sumMapSegmentCount]int{
	412, 430, 1592, 253, 344, 497, 552, 1283, 1817, 1531, 72, 1741,
	387, 964, 1046, 242, 470, 362, 275, 175, 549, 431, 318,
}

// sumMapTotalSize 是 23 筆 size 加總，等於 SUM.MAP 應有的檔案大小（15,743）。
// LoadSumMap 用它驗證表格與實際檔案是否吻合。
func sumMapTotalSize() int {
	total := 0
	for _, s := range sumMapSizes {
		total += s
	}
	return total
}

// SumMap 是 SUM.MAP 解壓後的 23 個子地圖集合。
//
// 建立方式一律透過 LoadSumMap，零值不可用。
type SumMap struct {
	segments map[int]*Map // key = 子地圖 ID（非陣列索引）
	order    []int        // 依 SUM.MAP 內部排列順序記錄 ID，供 IDs() 回傳穩定順序
}

// IDs 回傳全部子地圖 ID，順序與 SUM.MAP 檔案內的排列順序一致。
// 目前已知的 23 個 ID 涵蓋 2、4、12、13、21〜23、25、33〜36、41、43〜45、
// 51、52、54〜56、64、66——其中 2 與 4 沒有獨立的 .MAP 檔（見
// docs/formats/town-and-map.md §5 第 5 點），只能透過 SUM.MAP 取得。
func (s *SumMap) IDs() []int {
	out := make([]int, len(s.order))
	copy(out, s.order)
	return out
}

// Segment 依 ID（不是陣列索引）取得已解壓的子地圖。ok=false 表示
// 這份 SUM.MAP 沒有這個 ID。
func (s *SumMap) Segment(id int) (*Map, bool) {
	m, ok := s.segments[id]
	return m, ok
}

// LoadSumMap 解析指定路徑的 SUM.MAP：依 sumMapIDs／sumMapSizes 表切出
// 23 個變長區塊，逐一 RLE 解壓成 64×64 子地圖。
//
// 解析失敗（檔案讀不到、檔案大小與 23 筆 size 加總不符）一律回傳 error，
// 不 panic。單一區塊解壓後格數不足 4096 不視為錯誤，只會反映在該 Map 的
// FilledCount()——不過實測 23/23 個區塊都精確填滿，出現缺格就代表
// decodeRLE 又走鐘了，TestLoadSumMap_AllSegmentsFill 會擋下來。
func LoadSumMap(path string) (*SumMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("world: 讀取 %s 失敗: %w", path, err)
	}
	want := sumMapTotalSize()
	if len(data) != want {
		return nil, fmt.Errorf(
			"world: %s 長度 %d 與 23 筆子地圖 size 表加總 %d 不符，資料或表格可能已過期",
			path, len(data), want)
	}

	sm := &SumMap{
		segments: make(map[int]*Map, sumMapSegmentCount),
		order:    make([]int, 0, sumMapSegmentCount),
	}

	offset := 0
	for i := 0; i < sumMapSegmentCount; i++ {
		id := sumMapIDs[i]
		size := sumMapSizes[i]
		chunk := data[offset : offset+size]
		offset += size

		m := &Map{}
		m.filled = decodeRLE(chunk, &m.tiles)
		sm.segments[id] = m
		sm.order = append(sm.order, id)
	}

	return sm, nil
}

// decodeRLE 解壓一個 SUM.MAP 子地圖區塊，寫入 out（固定 64×64=4096 格，
// 呼叫端負責預先歸零）。回傳實際被寫入的格數；用真實 SUM.MAP 實測
// 23/23 個區塊都精確等於 4096。
//
// 演算法反組譯自 FUN_25be_17fe（Ghidra 位址 25be:17fe = DEMON.INT 檔位移
// 0x1afde，見 docs/re/03-audio-and-resources.md §3.3）：
//
//	逐 byte 掃描來源資料，直到讀完 param_1-1 個 byte 為止（原函式的
//	迴圈判斷式在 0x1b0b6：「param_1 - 1 <= 讀取游標」就結束）：
//	  - byte 最高位 = 1：RLE 記錄，2 bytes。byte&0x7f 是 tile 值，
//	    下一個 byte 是重複次數，寫入該值 count 次。
//	  - byte 最高位 = 0：literal 記錄，1 byte。這個 byte 本身就是
//	    tile 值，只寫入一次。
//
// 格式與已知限制（誠實記錄，未強行湊測試）：
//
//  1. **重複次數 0 代表 256，不是 0。** 原函式的重複迴圈是 do-while，
//     不是 while——先寫一格，再 `dec` 次數，最後才比對是否為 0：
//
//     1b01b  loop: inc  word [bp-0x4]      ; 已寫格數++
//     1b048        cmp  word [bp-0x4],0x1000
//     1b04d        jne  1b057              ; 寫滿 4096 就收工
//     1b057        dec  byte [bp-0x7]      ; 次數--
//     1b05a        add  word [bp-0x6],0x40 ; 游標推進
//     1b06a        cmp  byte [bp-0x7],0x0
//     1b06e        jne  loop
//
//     次數 byte = 0 時，`dec` 後變 0xff，要繞完整整 256 圈才回到 0。
//     這是 8 位元計數器唯一能表達「256 連」的方式，也是 RLE 常見手法。
//     照字面當成 0 圈解，每碰到一次就少寫 256 格——症狀是後續 tile 全部
//     往前位移 256 格（欄優先游標下 = 往左 4 欄），而且解出來還是「像
//     地圖」，肉眼與格數檢查都抓不到。抓到它的是城鎮座標表交叉比對
//     （見 §3）。
//
//  2. **每個區塊的第一個 byte 不是 RLE 資料，要跳過。**
//     原程式讀檔時把區塊寫進 `[ds:0x4c90]+1`（0x188b5 的 `add ax,0x1`），
//     解壓時從 `[ds:0x4c94]` 起讀。實測上這一個 byte 一定得跳過：跳過後
//     23/23 個區塊都精確填滿 4096 格，不跳過則全部卡在 4097-256k 格。
//     這個 byte 的語意未解，值域 0x01–0x09（區段 2 與 4 例外，都是 0xa3），
//     與獨立 .MAP 檔開頭那 1 byte header（MAP1=0x00、MAP3=0x97、MAP5=0x09）
//     疑似同一種東西。
//
//  3. 上面兩點的**驗收依據不是「解得出東西」，是城鎮座標表**：
//     DEMON.INT 的 25 筆城鎮 (X,Y) 表（DS:0x2f86，見 gamedata 的 townSites）
//     是從機器碼獨立讀出來的，與 SUM.MAP 無關。修正前 25 個座標上一個
//     城鎮 tile（0x2e／0x64）都沒有；兩點都修好之後，24 個座標精確命中
//     SUM.MAP 的城鎮 tile，第 25 個（海盜灣）命中 MAP1.MAP——25/25 全中。
//     兩份互相獨立的資料能對到這種程度，不會是湊出來的。
//
//  4. **尚未實作：tile 0x14 的條件替換。** 原函式在寫入前會檢查
//     `tile == 0x14`，成立就呼叫 15be:179f，回傳 1 時改寫成 0x62
//     （0x1b01e／0x1b075 兩個分支都有）。那個判斷式是什麼還沒追，
//     推測與雪地版地形（WINTER.SHP）有關。本函式先照實寫入 0x14，
//     不猜條件——真要還原季節變化時要回頭補這一段。
//
//  5. **寫入游標是 column-major（逐欄填），不是循序**。
//     docs/re/03 §3.3 把游標描述成「每寫一格 +64，對 4096 取餘」，
//     那個公式漏了進位：gcd(64,4096)=64，逐字實作只會在 64 個位置上
//     打轉，丟掉 98% 的資料。真正的邏輯在原始指令裡（25be:187a 起）：
//
//     25be:187a  ADD word ptr [BP+-0x6],0x40    ; cursor += 64
//     25be:187e  CMP word ptr [BP+-0x6],0x1000  ; cmp cursor, 4096
//     25be:1883  JL   0x746a                    ; 未越界就跳過進位
//     25be:1885  ADD word ptr [BP+-0x6],0xf001  ; cursor += -4095
//
//     `0xf001` 當有號 16 位元是 -4095，也就是 `-4096 + 1`：游標走到
//     欄底時回到頂端並右移一欄。這是標準的 column-major 走訪。
//
//     這點很容易錯得無聲無息：改用 row-major 循序游標一樣能填滿 4096
//     格、一樣通過「size 加總／值域／格數」這類測試，畫出來也還是像
//     地圖（有房間有走廊），但整張圖是沿對角線轉置的。屬於「資料對
//     但顯示錯」，測試抓不到，只有對照原版畫面才看得出來。
func decodeRLE(src []byte, out *[mapTileCount]byte) int {
	const (
		colStep  = MapWidth         // 往下一列 = +64
		wrapBack = mapTileCount - 1 // 越界時回頂端並右移一欄 = -4095
	)

	cursor := 0
	filled := 0
	// 跳過區塊的第一個 byte（見說明 §2）。
	i := 1
	n := len(src)

	// 原函式每寫一格就推進游標一次，所以計數用 filled 而非 cursor。
	write := func(v byte) {
		out[cursor] = v
		filled++
		cursor += colStep
		if cursor >= mapTileCount {
			cursor -= wrapBack
		}
	}

	for i < n-1 && filled < mapTileCount {
		b := src[i]
		if b&0x80 != 0 {
			// RLE 記錄：高位元=1，低 7 位是 tile 值，下一 byte 是重複次數。
			val := b & 0x7f
			// 次數 0 代表 256：原函式是 do-while，dec 後才判斷（見說明 §1）。
			count := int(src[i+1])
			if count == 0 {
				count = 256
			}
			i += 2
			for c := 0; c < count && filled < mapTileCount; c++ {
				write(val)
			}
		} else {
			// literal 記錄：這個 byte 本身就是 tile 值。
			write(b)
			i++
		}
	}

	return filled
}
