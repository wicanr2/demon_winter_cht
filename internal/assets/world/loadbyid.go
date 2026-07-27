package world

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// MapFileName 是獨立地城地圖的檔名。**只對 1／3／5 有意義** ——
// 其餘編號在 `SUM.MAP` 裡（`LoadByID` 的 switch 就是這條規則的唯一實作）。
func MapFileName(id int) string { return fmt.Sprintf("MAP%d.MAP", id) }

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
//
// # saveDir
//
// **地圖是會被改寫的**（推開家具、密語門開牆、`U` 的 `P` 動作），
// 原版改完就把整張圖寫回 `MAP%d.MAP`（`docs/re/95` §3.9）。
// 本專案不蓋原版資料目錄，改動寫在存檔目錄，所以這裡先看那邊 ——
// 與 `nSS.DAT`／`ITEMLOCB.DAT` 同一套三段優先序。
//
// `saveDir` 傳空字串代表「不要找存檔」（工具程式用，例如 `cmd/dwroute`
// 要看的是原始地圖不是某個人的進度）。
func LoadByID(saveDir, dataDir string, id int) (*Map, error) {
	switch id {
	case 1, 3, 5:
		name := MapFileName(id)
		if saveDir != "" {
			if m, err := LoadMap(filepath.Join(saveDir, name)); err == nil {
				return m, nil
			} else if !os.IsNotExist(errors.Unwrap(err)) {
				return nil, err
			}
		}
		return LoadMap(filepath.Join(dataDir, name))
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
