package gamedata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 城鎮資料。
//
// 執行期反組譯看到的是遠指標 `0x5534` 指向「當前城鎮資料表」，
// 價格公式從它的 `+0x1ed`（經濟係數）與 `+0x1f5`（船價基礎值）取值。
// **那個緩衝區就是載入的 `TOWN<n>.DAT` 檔案本身** —— 檔案剛好 512 bytes，
// 兩個位移都落在檔內。
//
// 這條對應是本專案自己確認的，兩個獨立證據：
//
//  1. `+0x1ed` 在 25 座城鎮的值落在 8–25，正好是「物價指數」該有的值域。
//  2. `+0x1f5` 只有 5 座城鎮非零（Janthrin、New Gleon、Dragontooth、
//     Asaht、Land's Edge），其餘 20 座為 0 —— 只有碼頭城鎮賣船。
//     攻略明寫「前往東北方很遠的**新格里昂**，買一艘船」，
//     New Gleon 正是那 5 座之一。
const (
	// NumTowns 是城鎮數。TOWN.TXT 前 25 個字串即 25 座城鎮名。
	NumTowns = 25

	// townRecordSize 是 TOWN<n>.DAT 的檔案大小。
	townRecordSize = 512

	// offTownEconomy／offTownShipBase 是城鎮表內的兩個已驗證欄位。
	offTownEconomy  = 0x1ed
	offTownShipBase = 0x1f5
)

// Town 是一座城鎮。
type Town struct {
	// Number 是 1–25，對應 TOWN<n>.DAT。
	Number int
	Name   string

	// Economy 是經濟係數 E，一切價格的基礎。
	Economy int
	// ShipBase 是買船價的基礎值，買船價 = ShipBase × 10。0 代表不賣船。
	ShipBase int

	// raw 是整份 512 bytes，其餘欄位語意未解，先留著供後續分析。
	raw []byte
}

// SellsShips 回報這座城鎮有沒有碼頭在賣船。
func (t Town) SellsShips() bool { return t.ShipBase > 0 }

// TownTable 是 25 座城鎮。
type TownTable struct {
	towns []Town
}

// LoadTownTable 從資料目錄讀入 TOWN.TXT 與 TOWN1..25.DAT。
//
// 缺任何一個檔就是錯誤 —— 城鎮少一座不會當掉，只會讓玩家某天走進去
// 發現物價全錯，那種缺陷比開不起來難查得多。
func LoadTownTable(dataDir string) (*TownTable, error) {
	names, err := loadTownNames(filepath.Join(dataDir, "TOWN.TXT"))
	if err != nil {
		return nil, err
	}

	t := &TownTable{}
	for i := 1; i <= NumTowns; i++ {
		path := filepath.Join(dataDir, fmt.Sprintf("TOWN%d.DAT", i))
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("gamedata: 讀取 %s 失敗: %w", path, err)
		}
		if len(raw) != townRecordSize {
			return nil, fmt.Errorf("gamedata: %s 長度 %d，預期 %d",
				path, len(raw), townRecordSize)
		}
		name := fmt.Sprintf("城鎮 %d", i)
		if i-1 < len(names) {
			name = names[i-1]
		}
		t.towns = append(t.towns, Town{
			Number:   i,
			Name:     name,
			Economy:  int(raw[offTownEconomy]),
			ShipBase: int(raw[offTownShipBase]),
			raw:      raw,
		})
	}
	return t, nil
}

// loadTownNames 讀 TOWN.TXT 的 NUL 分隔字串。
func loadTownNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gamedata: 讀取 %s 失敗: %w", path, err)
	}
	var out []string
	for _, part := range strings.Split(string(data), "\x00") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	if len(out) < NumTowns {
		return nil, fmt.Errorf("gamedata: %s 只解出 %d 個字串，至少要 %d 個城鎮名",
			path, len(out), NumTowns)
	}
	return out, nil
}

// Len 回傳城鎮數。
func (t *TownTable) Len() int { return len(t.towns) }

// All 回傳全部城鎮。
func (t *TownTable) All() []Town { return append([]Town(nil), t.towns...) }

// ByNumber 以 1–25 取城鎮。
func (t *TownTable) ByNumber(n int) (Town, error) {
	if n < 1 || n > len(t.towns) {
		return Town{}, fmt.Errorf("gamedata: 城鎮編號 %d 超出 1–%d", n, len(t.towns))
	}
	return t.towns[n-1], nil
}
