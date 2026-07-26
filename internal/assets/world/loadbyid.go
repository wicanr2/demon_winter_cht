package world

import (
	"fmt"
	"path/filepath"
)

// LoadByID 依子地圖編號載入地圖。
//
// 兩個來源：**1／3／5 是獨立的 `MAPn.MAP`，其餘全部在 `SUM.MAP` 裡**
// （含地圖 2、4 與世界地圖各段）。
//
// # 為什麼這支要放在 world 而不是 cmd
//
// 這條規則原本有**三份**實作：`cmd/demonwinter` 的 `loadMapArg`（只查 `SUM.MAP`）、
// `mapchange.go` 的 `loadMapByID`（有 1／3／5 特例）、以及 `cmd/dwroute` 自己抄的一份。
//
// 後果是一個實際發生過的 regression：`-map` 的預設值改成「用存檔的 MapID」
// 之後，出貨存檔的 `1` 走進了只查 `SUM.MAP` 的那一份，
// **啟動直接失敗**（`SUM.MAP 沒有子地圖 1`）。而當時只用 `-newgame`
// （MapID 34，`SUM.MAP` 有 34）驗過，沒回頭測從出貨存檔啟動那條路。
//
// 所以合成一份放在這裡 —— 有測試護著，而且兩個呼叫端不可能再漂。
func LoadByID(dataDir string, id int) (*Map, error) {
	switch id {
	case 1, 3, 5:
		return LoadMap(filepath.Join(dataDir, fmt.Sprintf("MAP%d.MAP", id)))
	}
	sm, err := LoadSumMap(filepath.Join(dataDir, "SUM.MAP"))
	if err != nil {
		return nil, err
	}
	seg, ok := sm.Segment(id)
	if !ok {
		return nil, fmt.Errorf("world: 地圖 %d 既沒有 MAP%d.MAP 也不在 SUM.MAP 裡（有的是 %v）",
			id, id, sm.IDs())
	}
	return seg, nil
}
