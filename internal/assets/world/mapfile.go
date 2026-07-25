// Package world 解析 Demon's Winter 的地圖與世界資料檔：
// 獨立地城地圖（MAP1/3/5.MAP）、打包地圖（SUM.MAP，含 RLE 解壓）、
// 出口／事件座標表（EXITS.DAT）、城鎮定義（TOWN1..25.DAT）。
//
// 規格來源見 docs/formats/town-and-map.md（結構總覽）與
// docs/re/03-audio-and-resources.md §3（SUM.MAP 的 RLE 解壓演算法，
// 反組譯自 FUN_25be_17fe，25be:17fe）、docs/re/05-event-triggering.md
// （EXITS.DAT 的 3-byte 記錄格式，反組譯自 FUN_222f_1321，222f:1321）。
package world

import (
	"fmt"
	"os"
)

// MapWidth、MapHeight 是每張地圖固定的格數。原版 MAP1/3/5.MAP 與
// SUM.MAP 打包的子地圖一律是 64×64（已驗證，見 docs/formats/town-and-map.md §2.1）。
const (
	MapWidth  = 64
	MapHeight = 64

	mapTileCount = MapWidth * MapHeight // 4096
	mapFileSize  = 1 + mapTileCount     // 4097：1 byte header + 64*64 tiles
)

// Map 是一張 64×64 地圖／地城的乾淨表示。
//
// 建立方式一律透過 LoadMap（獨立 .MAP 檔）或 SumMap.Segment（SUM.MAP
// 內打包的子地圖），零值不可用。
type Map struct {
	// header 是獨立 .MAP 檔案開頭的 1 byte（已驗證存在，語意未解——
	// MAP1=0x00、MAP3=0x97、MAP5=0x09，不是簡單的 map_id，見
	// docs/formats/town-and-map.md §2.1）。SUM.MAP 解壓出的子地圖沒有
	// 這個欄位，固定為 0。
	header byte
	tiles  [mapTileCount]byte

	// filled 是實際被寫入非預設值的格數。獨立 .MAP 檔一律等於 4096
	// （整份檔案就是完整的 tile 陣列）。SUM.MAP 解壓出的子地圖可能小於
	// 4096——RLE 資料讀完就停止，不保證填滿整張 64×64 網格
	// （見 docs/re/03-audio-and-resources.md §3.3 最後一段），未填到的格子
	// 維持零值 0（與 MAP1.MAP 裡「0 = 未映射區域/邊界外」的觀察一致）。
	filled int
}

// Header 回傳獨立 .MAP 檔開頭的 1 byte 欄位。SUM.MAP 解壓出的子地圖
// 沒有這個欄位，固定回傳 0。
func (m *Map) Header() byte { return m.header }

// FilledCount 回傳實際被寫入的格數（見 filled 欄位說明）。
// 呼叫端可用它判斷這張地圖是否「填滿」整個 64×64 網格。
func (m *Map) FilledCount() int { return m.filled }

// TileAt 回傳座標 (x, y) 的 tile 值。座標超出 [0, MapWidth) × [0, MapHeight)
// 範圍時回傳 error，不 panic。
func (m *Map) TileAt(x, y int) (byte, error) {
	if x < 0 || x >= MapWidth || y < 0 || y >= MapHeight {
		return 0, fmt.Errorf("world: 座標 (%d,%d) 超出地圖範圍 [0,%d)x[0,%d)", x, y, MapWidth, MapHeight)
	}
	return m.tiles[y*MapWidth+x], nil
}

// Tiles 回傳整份 tile 陣列的複本（row-major，索引 = y*MapWidth+x），
// 長度固定 4096。回傳複本是為了避免呼叫端改到 Map 的內部狀態。
func (m *Map) Tiles() [mapTileCount]byte { return m.tiles }

// LoadMap 解析指定路徑的獨立地城地圖檔（MAP1.MAP／MAP3.MAP／MAP5.MAP）。
//
// 格式（已驗證，見 docs/formats/town-and-map.md §2.1）：4097 bytes =
// 1 byte header + 64×64 = 4096 bytes 的 tile 陣列，row-major
// （offset 1+y*64+x）。解析失敗一律回傳 error，不 panic。
func LoadMap(path string) (*Map, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("world: 讀取 %s 失敗: %w", path, err)
	}
	if len(data) != mapFileSize {
		return nil, fmt.Errorf("world: %s 長度 %d 不等於預期的 %d (1 + %d*%d)",
			path, len(data), mapFileSize, MapWidth, MapHeight)
	}
	m := &Map{header: data[0], filled: mapTileCount}
	copy(m.tiles[:], data[1:])
	return m, nil
}
