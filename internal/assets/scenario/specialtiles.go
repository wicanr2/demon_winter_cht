package scenario

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// 子地圖的特殊格清單（`nSS.DAT`，每張地城一份）。
//
// 這是「走到某一格會發生什麼」的來源（`docs/re/77`、`docs/re/78`）。
// 一度被當成 `EXITS.DAT`（`docs/re/05` §1.3）與開場序列的繪製指令
// （`docs/formats/resource-index.md` §3.3）—— 兩個都已推翻。
//
// 檔案佈局是**兩張表往中間長**：
//
//	偏移 0 起往後：3-byte 記錄 (X, Y, attr)，X == 0 表示表尾
//	偏移 510 起往前：2-byte 座標對，供類別 4（傳送）取目的地
//	attr = (類別 << 5) | 值
//
// 傳送的配對規則是**序號**：第 k 筆類別 4 的記錄用第 k 對座標
// （k 從檔尾往前數）。原版 `3SS.DAT` 的資料**看起來**像在暗示「配最近的那一個」，
// 但那是原版資料本身的怪異之處，照序號實作才是還原（見 `docs/re/77` §4）。
//
// ⚠ **這個檔案是存檔的一部分**（`docs/re/78`）：事件觸發後會就地改寫記錄，
// 離開子地圖時整份存回，開新遊戲時從 `ALL_SS.DAT` 重建。
// 所以它跟 `PARTY.DAT` 同一個等級 —— 絕不可寫回 `workplace/orig/`。
const (
	// SpecialTileFileSize 是 nSS.DAT 的大小。原版載入／存回都用這個長度
	// （`0x16064` 與 `0x14857` 的 `0x1ff`），不是 512。
	SpecialTileFileSize = 511

	specialRecordLen = 3
	specialOffX      = 0
	specialOffY      = 1
	specialOffAttr   = 2

	// attr 的切法：高 3 bit 是類別、低 5 bit 是值（`0x1726b` 的 `/0x20`、`%0x20`）。
	specialClassShift = 5
	specialValueMask  = 0x1f

	// 檔尾反向座標對：第 k 對在 `0x1ff - (k*2 + 2)`（`0x1729a`）。
	specialTailBase = SpecialTileFileSize
	specialTailStep = 2
)

// 類別（`docs/re/83` §1 把最後一塊補上，分派在 `FUN_25be_0263` ＝ `0x19a43`）：
//
//	0    → 這一格沒事（一次性事件用掉自己之後長這樣，見 SpecialTile.Dead）
//	1、2 → 回一個 1-based 序號，那是事件敘述的索引
//	3、6 → 陷阱（"A trap!"）；出貨資料裡沒有 6
//	4    → 傳送，目的地取自檔尾反向表
//	5    → 地點劇情事件，值 = docs/re/65 那張 16 格表的 case 編號
const (
	SpecialClassEventA   = 1
	SpecialClassEventB   = 2
	SpecialClassTrap     = 3
	SpecialClassTeleport = 4
	// SpecialClassLocationPlot 是主線的地點劇情格。判定條件在原版是
	// `cmp ds:0x5c62, 5`（`0x19f15`），不通過就走預設的房間文字路徑。
	SpecialClassLocationPlot = 5
	// SpecialClassTrapAlt 是**已被 `L` 標記過的陷阱**（`docs/re/91` §2）：
	// 察覺陷阱之後 `attr += 0x60`，類別 3 就變成 6；觸發時多一道
	// `Roll(2)` 的迴避 —— 手冊「經過時可能不會觸發」的精確值就是 50%。
	//
	// ⚠ 這裡原本寫「出貨資料裡一筆都沒有，程式路徑留著、資料沒用到」，
	// **那是錯的**：`1SS.DAT` 的 (11,17) 就是一筆。`nSS.DAT` 是存檔
	// （`docs/re/78`）、出貨的存檔又是玩過的（`docs/re/87`）——
	// 那一格是 1988 年那位玩家自己標記下來的。
	// 釘在 `TestShippedDataHasANoticedTrap`。
	SpecialClassTrapAlt = 6

	// specialClassAdvance 是「狀態推進」的加量：類別 +3（`0x1788e` 的 `+0x60`）。
	specialClassAdvance = 3
	// specialAdvanceLimit 是推進的守衛（`attr < 0xc1` 才加）。
	// 它的效果是**只能推進一次** —— 加完必然 >= 這個值。
	specialAdvanceLimit = 0xc1
	// specialVisitedValue 是「已造訪」標記寫進低 5 bit 的值
	//（`0x1a616`：`(類別 << 5) | 1`）。
	specialVisitedValue = 1
)

// SpecialTile 是清單裡的一筆記錄。
type SpecialTile struct {
	X, Y byte
	// Attr 保留原始 byte。類別與值用 Class()／Value() 取，
	// 因為改寫規則（清零／推進／標記）操作的是整個 byte。
	Attr byte
}

// Class 是 attr 的高 3 bit。
func (s SpecialTile) Class() byte { return s.Attr >> specialClassShift }

// Dead 回報這筆記錄是不是**死的**（屬性整個為 0）。
//
// 兩個來源都指向同一件事：
//
//   - `Consume` 就是把屬性寫成 0（原版四處都這樣做），所以一次性事件用掉
//     自己之後長這樣。
//   - **出貨資料裡本來就有這種記錄**：`1SS.DAT` 的類別 0 共四筆
//     （(19,17)／(20,23)／(13,9)／(14,4)），四筆的屬性都剛好是 `0x00`。
//     也就是說「屬性 0」是作者用來標「這一格沒事」的寫法，不是巧合。
//
// ⚠ **原版的分派並沒有明文擋掉它**：`docs/re/83` 記的預設分支是
// 「不是類別 3／6／5 就去讀 `DATA*.TXT` 顯示房間文字」，照字面看
// 類別 0 會顯示第 0 筆文字。那條路徑本專案沒有追到底（可能還有一道
// 值為 0 的守衛），所以這裡採「死記錄就什麼都不做」——
// 那是唯一與出貨資料一致的解釋。
//
// 實跑抓到的症狀：試煉室過關（`Consume`）之後再走上去，
// 畫面跳出一段**完全無關**的房間敘述（「你走進一間詭異的停屍間…」）。
// 陷阱解除後那條路也一樣，只是一直沒人踩回去。
func (s SpecialTile) Dead() bool { return s.Attr == 0 }

// Value 是 attr 的低 5 bit。
//
// 語意隨類別而變：類別 1／2 是「已造訪」標記（序號另外從計數器來）；
// **類別 5 是地點劇情事件的 case 編號**（`docs/re/83`）；
// 類別 4 恆為 0（目的地在檔尾反向表）。
func (s SpecialTile) Value() byte { return s.Attr & specialValueMask }

// PlotCase 是類別 5 的地點劇情 case 編號，其他類別回 −1。
//
// 這個編號是**全域唯一**的：五張子地圖用掉 1–15，一次碰撞都沒有
//（`docs/re/83` §2）。所以它可以直接當事件的身分，不必再配地圖編號。
func (s SpecialTile) PlotCase() int {
	if s.Class() != SpecialClassLocationPlot {
		return -1
	}
	return int(s.Value())
}

// 地點劇情的 case 編號。只列已經接上引擎的；其餘見 `docs/re/65` §3。
const (
	// PlotCaseEregore 是艾瑞戈爾（地圖 1 的 (60,1)，全遊戲唯一一格）。
	PlotCaseEregore = 14
)

// SpecialTiles 是一張子地圖的完整清單，含檔尾的傳送目的表。
type SpecialTiles struct {
	Tiles []SpecialTile
	// Dests 是檔尾往前數的座標對，索引即「第幾筆類別 4」。
	Dests []Dest
}

// Dest 是一組傳送目的座標。
type Dest struct{ X, Y byte }

// ErrSpecialTileSize 是長度不符。
var ErrSpecialTileSize = errors.New("nSS.DAT 長度不符")

// ParseSpecialTiles 解一份 nSS.DAT。
//
// 前段讀到 X == 0 為止；檔尾的座標對讀到 (0,0) 或撞上前段為止
// —— 兩張表往中間長，中間是填充零，所以「撞上」就是停止條件。
func ParseSpecialTiles(data []byte) (*SpecialTiles, error) {
	if len(data) != SpecialTileFileSize {
		return nil, fmt.Errorf("%w：得到 %d bytes，預期 %d",
			ErrSpecialTileSize, len(data), SpecialTileFileSize)
	}

	st := &SpecialTiles{}
	end := 0
	for end+specialRecordLen <= len(data) && data[end+specialOffX] != 0 {
		st.Tiles = append(st.Tiles, SpecialTile{
			X:    data[end+specialOffX],
			Y:    data[end+specialOffY],
			Attr: data[end+specialOffAttr],
		})
		end += specialRecordLen
	}

	for k := 0; ; k++ {
		off := specialTailBase - (k*specialTailStep + 2)
		if off < end {
			break
		}
		d := Dest{X: data[off], Y: data[off+1]}
		if d.X == 0 && d.Y == 0 {
			break
		}
		st.Dests = append(st.Dests, d)
	}
	return st, nil
}

// Encode 把清單寫回 511 bytes 的原始佈局。
//
// 存回時必須是原版的長度與佈局，因為原版自己還會讀它
//（本專案的存檔要能被原版讀，這是移植的基本要求）。
func (st *SpecialTiles) Encode() []byte {
	out := make([]byte, SpecialTileFileSize)
	for i, t := range st.Tiles {
		off := i * specialRecordLen
		if off+specialRecordLen > len(out) {
			break
		}
		out[off+specialOffX] = t.X
		out[off+specialOffY] = t.Y
		out[off+specialOffAttr] = t.Attr
	}
	for k, d := range st.Dests {
		off := specialTailBase - (k*specialTailStep + 2)
		if off < 0 || off+1 >= len(out) {
			break
		}
		out[off] = d.X
		out[off+1] = d.Y
	}
	return out
}

// SpecialHit 是一次查表命中的結果，對應原版寫進
// `ds:0x52f4`／`0x52f6`／`0x5c62` 的三個值。
type SpecialHit struct {
	// Index 是記錄在 Tiles 裡的位置，改寫時要用它定位。
	Index int
	Tile  SpecialTile
	// EventIndex 是事件敘述的索引。
	//
	// **每一種類別都有值**（原版 `0x17265` 對所有命中都寫 `ds:0x52f4 = local_c`），
	// 只有類別 1／2 額外 +1（`0x17291`）。所以類別 0 拿到的是 0-based 的計數、
	// 類別 1／2 拿到的是 1-based —— 這個不對稱是原版的，不是筆誤。
	EventIndex int
	// Dest 是類別 4 的傳送目的地；Teleport 為 false 時無意義。
	Dest     Dest
	Teleport bool
}

// Lookup 拿座標查表，沒命中回 nil。
//
// 序號的算法照原版：掃描途中**只有類別 1／2 會讓計數器前進**
//（`0x172df`），類別 4 走另一個計數器（`0x172e7`）當傳送目的表的索引。
// 兩個計數器互不相干 —— 共用一個會讓兩種查詢互相污染。
func (st *SpecialTiles) Lookup(x, y byte) *SpecialHit {
	eventSeq, teleSeq := 0, 0
	for i, t := range st.Tiles {
		if t.X == x && t.Y == y {
			hit := &SpecialHit{Index: i, Tile: t, EventIndex: eventSeq}
			switch t.Class() {
			case SpecialClassEventA, SpecialClassEventB:
				hit.EventIndex++
			case SpecialClassTeleport:
				if teleSeq < len(st.Dests) {
					hit.Dest = st.Dests[teleSeq]
					hit.Teleport = true
				}
			}
			return hit
		}
		switch t.Class() {
		case SpecialClassEventA, SpecialClassEventB:
			eventSeq++
		case SpecialClassTeleport:
			teleSeq++
		}
	}
	return nil
}

// Consume 把一筆記錄的屬性清零 —— 一次性事件用掉自己。
// 屬性 0 的類別是 0，之後不會再命中任何分支（原版四處都這樣做）。
func (st *SpecialTiles) Consume(index int) {
	if index < 0 || index >= len(st.Tiles) {
		return
	}
	st.Tiles[index].Attr = 0
}

// Advance 把類別 +3（原版 `attr += 0x60`），並照原版的守衛只推進一次。
// 回傳有沒有真的動。
func (st *SpecialTiles) Advance(index int) bool {
	if index < 0 || index >= len(st.Tiles) {
		return false
	}
	if st.Tiles[index].Attr >= specialAdvanceLimit {
		return false
	}
	st.Tiles[index].Attr += specialClassAdvance << specialClassShift
	return true
}

// MarkVisited 把低 5 bit 設成 1、類別留著（原版 `(類別 << 5) | 1`）。
// 記錄照樣命中、照樣分派 —— 這是「看過了」不是「用掉了」。
func (st *SpecialTiles) MarkVisited(index int) {
	if index < 0 || index >= len(st.Tiles) {
		return
	}
	st.Tiles[index].Attr =
		st.Tiles[index].Class()<<specialClassShift | specialVisitedValue
}

// ALLSSBlockSize 是 ALL_SS.DAT 裡每個區塊的大小。
// 前 511 bytes 是 nSS.DAT 的內容，第 512 個 byte 是區塊標記。
const ALLSSBlockSize = 512

// SpecialTileMapCount 是有特殊格清單的子地圖數（`1SS.DAT`–`5SS.DAT`）。
const SpecialTileMapCount = 5

// SplitAllSS 把 ALL_SS.DAT 切成五份 nSS.DAT 的內容。
//
// 這就是原版開新遊戲做的事（`0x14845` 的迴圈）：讀 2560 bytes，
// 每 512 取前 511 寫成 `1SS.DAT`…`5SS.DAT`。
// **母本唯讀、工作副本才會被改寫** —— 所以新遊戲一定要走這一步，
// 否則會沿用上一次遊玩留下的狀態（原版出廠的 `1SS`／`2SS` 就是髒的）。
func SplitAllSS(data []byte) ([][]byte, error) {
	want := ALLSSBlockSize * SpecialTileMapCount
	if len(data) != want {
		return nil, fmt.Errorf("ALL_SS.DAT 長度不符：得到 %d bytes，預期 %d",
			len(data), want)
	}
	out := make([][]byte, 0, SpecialTileMapCount)
	for i := 0; i < SpecialTileMapCount; i++ {
		block := data[i*ALLSSBlockSize : i*ALLSSBlockSize+SpecialTileFileSize]
		out = append(out, append([]byte(nil), block...))
	}
	return out, nil
}

// SpecialTileFileName 是子地圖 n 的清單檔名（原版 `sprintf("%dSS.DAT", n)`）。
func SpecialTileFileName(mapID int) string {
	return fmt.Sprintf("%dSS.DAT", mapID)
}

// LoadSpecialTileSet 準備五張子地圖的特殊格清單。
//
// 三條來源，優先序照原版的語意：
//
//  1. **存檔目錄裡有** → 讀那一份。清單會被事件就地改寫，所以它是進度的一部分
//     （`docs/re/78`），跟 `PARTY.DAT` 同一個等級。
//  2. **全新開始**（fresh）→ 從 `ALL_SS.DAT`（母本）切五份。這就是原版建角時
//     做的事（`0x14845` 的迴圈）。**不能省這一步** —— 原版出廠的 `1SS.DAT`／
//     `2SS.DAT` 是壓片前玩到一半的狀態（29 處差異），直接沿用等於玩家一開局
//     就繼承別人走過的痕跡。
//  3. 母本缺檔 → 退回讀資料目錄的 `nSS.DAT`（就是那份髒的，但有總比沒有好）。
//
// 缺檔一律不算錯：大地圖與城鎮本來就沒有清單（原版只在子地圖 < 10 才載入），
// 查不到就是「這張圖沒有特殊格」。
func LoadSpecialTileSet(saveDir, dataDir string, fresh bool) (map[int]*SpecialTiles, error) {
	if !fresh {
		st, err := readSpecialDir(saveDir)
		if err != nil {
			return nil, err
		}
		if len(st) > 0 {
			return st, nil
		}
	}

	raw, err := os.ReadFile(filepath.Join(dataDir, "ALL_SS.DAT"))
	switch {
	case err == nil:
		blocks, err := SplitAllSS(raw)
		if err != nil {
			return nil, err
		}
		out := make(map[int]*SpecialTiles, len(blocks))
		for i, b := range blocks {
			st, err := ParseSpecialTiles(b)
			if err != nil {
				return nil, fmt.Errorf("ALL_SS.DAT 區塊 %d：%w", i, err)
			}
			out[i+1] = st
		}
		return out, nil
	case os.IsNotExist(err):
		return readSpecialDir(dataDir)
	default:
		return nil, err
	}
}

// readSpecialDir 從一個目錄讀 `1SS.DAT`–`5SS.DAT`，缺檔跳過。
func readSpecialDir(dir string) (map[int]*SpecialTiles, error) {
	out := make(map[int]*SpecialTiles)
	for id := 1; id <= SpecialTileMapCount; id++ {
		path := filepath.Join(dir, SpecialTileFileName(id))
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		st, err := ParseSpecialTiles(raw)
		if err != nil {
			return nil, fmt.Errorf("%s：%w", path, err)
		}
		out[id] = st
	}
	return out, nil
}

// WriteSpecialTileSet 把清單寫進指定目錄（存檔目錄，**不是**原版資料目錄）。
//
// 清單是進度的一部分：不寫出去的話「這個一次性事件已經觸發過」會在
// 關掉遊戲時消失，而且畫面上完全看不出來 —— 下次進同一個地城，用掉的事件又活了。
func WriteSpecialTileSet(dir string, set map[int]*SpecialTiles) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("建立目錄 %s 失敗: %w", dir, err)
	}
	for id, st := range set {
		if st == nil {
			continue
		}
		path := filepath.Join(dir, SpecialTileFileName(id))
		if err := os.WriteFile(path, st.Encode(), 0o644); err != nil {
			return fmt.Errorf("寫入 %s 失敗: %w", path, err)
		}
	}
	return nil
}
