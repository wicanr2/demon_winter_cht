package world

import (
	"fmt"
	"os"
)

// exitRecordSize、exitRecordCount、exitFileSize 是 EXITS.DAT 的固定佈局：
// 110 筆 3-byte 記錄 [X, Y, type_byte]，330 bytes（已由反組譯修正，見
// docs/re/05-event-triggering.md §1.4、§5：原本 town-and-map.md 假設的
// 「165 筆 2-byte 座標對」已被推翻，stride 3 才是 FUN_222f_1321 實際使用的）。
const (
	exitRecordSize  = 3
	exitRecordCount = 110
	exitFileSize    = exitRecordSize * exitRecordCount // 330
)

// ExitRecord 是 EXITS.DAT 單筆 3-byte 記錄：座標 (X, Y) + 事件類型碼。
type ExitRecord struct {
	X, Y byte
	// Type 是原始 type_byte，未拆解。Category()／SubValue() 是它的
	// 語意拆解（見 docs/re/05-event-triggering.md §1.2）。
	Type byte
}

// Category 回傳事件類別（0–7），即 type_byte / 32。
//
// 已驗證（見 docs/re/05-event-triggering.md §1.2、§1.4）：類別 4 = 傳送點
// （讀第二組座標覆寫玩家位置）；類別 1/2 會讓事件索引 +1。真實 EXITS.DAT
// 資料裡目前只出現類別 0（94 筆）、1（14 筆）、2（2 筆），類別 3/5/6/7
// （含 4）雖然引擎邏輯上支援，但這份資料集用不到（type_byte 最大值只有
// 64，不足以組出 ≥96 的高類別）。
func (r ExitRecord) Category() int { return int(r.Type) / 32 }

// SubValue 回傳子值（0–31），即 type_byte % 32。
func (r ExitRecord) SubValue() int { return int(r.Type) % 32 }

// ExitTable 是 EXITS.DAT 解析後的 110 筆記錄集合，提供依座標查詢的
// 核心介面（事件觸發鏈路：座標 → EXITS.DAT 記錄 → 事件索引，見
// docs/re/05-event-triggering.md §1）。
//
// 建立方式一律透過 LoadExits，零值不可用。
type ExitTable struct {
	records []ExitRecord
	byCoord map[[2]byte]int // (X,Y) -> records 索引
}

// LoadExits 解析指定路徑的 EXITS.DAT。
//
// 格式（已由反組譯驗證，見 docs/re/05-event-triggering.md §1.4）：
// 330 bytes = 110 筆 3-byte 記錄 [X, Y, type_byte]，X/Y 值域 1–62。
// 解析失敗（檔案讀不到、長度不符）一律回傳 error，不 panic。
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
	for i := 0; i < exitRecordCount; i++ {
		off := i * exitRecordSize
		rec := ExitRecord{X: data[off], Y: data[off+1], Type: data[off+2]}
		records[i] = rec

		// 同一座標可能出現多筆記錄（真實 EXITS.DAT 裡 110 筆只有 85 個
		// 相異座標）。原版 FUN_222f_1321 是線性掃描、命中第一筆就 break
		// （見 docs/re/05 §1.2），因此這裡只保留「第一次出現」的索引，
		// 跟原版查詢行為一致；要看全部重複記錄請用 All()。
		key := [2]byte{rec.X, rec.Y}
		if _, dup := byCoord[key]; !dup {
			byCoord[key] = i
		}
	}

	return &ExitTable{records: records, byCoord: byCoord}, nil
}

// Len 回傳記錄總數（EXITS.DAT 固定 110）。
func (t *ExitTable) Len() int { return len(t.records) }

// All 回傳全部記錄的複本，順序與檔案內原始順序一致。
func (t *ExitTable) All() []ExitRecord {
	out := make([]ExitRecord, len(t.records))
	copy(out, t.records)
	return out
}

// Lookup 查詢座標 (x, y) 有無出口／事件記錄，這是事件觸發鏈路的核心查詢
// （對應原版 FUN_222f_1321 的線性掃描比對，見 docs/re/05-event-triggering.md
// §1.2）。座標重複時回傳第一筆（與原版行為一致，見 LoadExits 說明）。
// ok=false 表示這個座標沒有記錄。
func (t *ExitTable) Lookup(x, y byte) (ExitRecord, bool) {
	i, ok := t.byCoord[[2]byte{x, y}]
	if !ok {
		return ExitRecord{}, false
	}
	return t.records[i], true
}
