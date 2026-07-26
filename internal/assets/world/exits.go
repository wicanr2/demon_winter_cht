package world

import (
	"fmt"
	"os"
)

// EXITS.DAT ——「站在這一格會換到哪一張地圖的哪一格」。
//
// # 這個檔案被誤讀過兩次
//
// 舊註解寫「110 筆 3-byte 記錄 `[X, Y, type_byte]`」，並說那是
// `FUN_222f_1321` 實際使用的 stride。**兩件事都錯**（`docs/re/77` §3）：
//
//   - `FUN_222f_1321` 掃的是 `nSS.DAT`（511 bytes，3-byte 記錄），
//     不是 `EXITS.DAT`。當初把兩個緩衝區認成同一塊。
//   - `EXITS.DAT` 是 **55 筆 6-byte 記錄**（330 ÷ 6）。330 剛好也能被 3 整除，
//     所以錯的切法看起來自洽 —— 這是「算術對不代表切分單位對」的又一次。
//
// 欄位語意來自 `docs/re/05` §3 的 `FUN_222f_32d4` 反編譯（那一節是對的）：
//
//	while (buf[i] != 目前X || buf[i+1] != 目前Y) i += 6;
//	新子地圖 = buf[i+2];  新X = buf[i+3];  新Y = buf[i+4];  +0xaf = buf[i+5];
//
// # 為什麼可以確定切法對
//
// 出口是**成對**的：從 A 走到 B，B 那一格也有一筆走回 A，而且目的地一律
// **差一格**（走過去之後不會立刻被彈回來）。前幾對就長這樣：
//
//	(55,18) → 圖 1  (3,31)        (3,32)  → 圖 34 (55,18)
//	(55,16) → 圖 1  (2,18)        (2,18)  → 圖 34 (55,16)
//	(45,50) → 圖 2  (10,45)       (10,44) → 圖 34 (45,50)
//	(18, 7) → 圖 3  (49, 7)       (49, 7) → 圖 3  (18, 7)
//
// 6-byte 以外的切法湊不出這種互指關係 —— 見 exits_test.go 的往返測試。
const (
	exitRecordSize  = 6
	exitRecordCount = 55
	exitFileSize    = exitRecordSize * exitRecordCount // 330

	exitOffFromX = 0
	exitOffFromY = 1
	exitOffToMap = 2
	exitOffToX   = 3
	exitOffToY   = 4
	exitOffUnk   = 5
)

// ExitRecord 是一筆出口：站在 (FromX, FromY) 會被送到 ToMap 的 (ToX, ToY)。
type ExitRecord struct {
	FromX, FromY byte
	// ToMap 是目的子地圖編號（原版寫進隊伍 `+0xa3`）。
	ToMap    byte
	ToX, ToY byte
	// Unknown 是第 6 欄（原版寫進隊伍 `+0xaf`），語意未解。
	// 觀察值多為 1–5，不猜它是什麼。
	Unknown byte
}

// ExitTable 是 EXITS.DAT 解析後的 55 筆出口。
//
// 建立方式一律透過 LoadExits，零值不可用。
type ExitTable struct {
	records []ExitRecord
	byCoord map[[2]byte]int
}

// LoadExits 解析 EXITS.DAT。長度不符一律回傳 error，不 panic。
func LoadExits(path string) (*ExitTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("world: 讀取 %s 失敗: %w", path, err)
	}
	if len(data) != exitFileSize {
		return nil, fmt.Errorf("world: %s 長度 %d 不等於預期的 %d (%d 筆 * %d bytes)",
			path, len(data), exitFileSize, exitRecordCount, exitRecordSize)
	}

	records := make([]ExitRecord, exitRecordCount)
	byCoord := make(map[[2]byte]int, exitRecordCount)
	for i := range records {
		off := i * exitRecordSize
		rec := ExitRecord{
			FromX:   data[off+exitOffFromX],
			FromY:   data[off+exitOffFromY],
			ToMap:   data[off+exitOffToMap],
			ToX:     data[off+exitOffToX],
			ToY:     data[off+exitOffToY],
			Unknown: data[off+exitOffUnk],
		}
		records[i] = rec

		// 原版是線性掃描、命中第一筆就停（`docs/re/05` §3 的 while 迴圈），
		// 所以重複座標只保留第一筆；要看全部請用 All()。
		key := [2]byte{rec.FromX, rec.FromY}
		if _, dup := byCoord[key]; !dup {
			byCoord[key] = i
		}
	}

	return &ExitTable{records: records, byCoord: byCoord}, nil
}

// Len 回傳出口總數（固定 55）。
func (t *ExitTable) Len() int { return len(t.records) }

// All 回傳全部出口的複本，順序與檔案一致。
func (t *ExitTable) All() []ExitRecord {
	out := make([]ExitRecord, len(t.records))
	copy(out, t.records)
	return out
}

// Lookup 查某一格是不是出口。ok=false 表示這一格不是。
func (t *ExitTable) Lookup(x, y byte) (ExitRecord, bool) {
	i, ok := t.byCoord[[2]byte{x, y}]
	if !ok {
		return ExitRecord{}, false
	}
	return t.records[i], true
}
