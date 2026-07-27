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
	"path/filepath"
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
	// （整份檔案就是完整的 tile 陣列）；SUM.MAP 解壓出的子地圖實測也是
	// 23/23 全數 4096。這個欄位留著是為了讓「解壓沒解完」在
	// FilledCount() 上看得見，而不是靜靜生出一張缺角的地圖。
	//
	// 舊註解曾寫「SUM.MAP 子地圖可能小於 4096，RLE 讀完就停」——那是
	// decodeRLE 兩個 bug（次數 0 沒當成 256、沒跳過區塊首 byte）造成的
	// 假象，不是格式特性，已推翻。
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

// SetTileAt 就地改寫座標 (x, y) 的 tile。
//
// 原版有幾處事件會直接改寫地圖緩衝區：密語謎題答對時把牆打開
// （`docs/re/84`：`map[0x48b] = 0`，`0x48b` ＝ (11,18)）、
// 推開家具（`Move:`）、`U` 的 `P` 動作。
//
// > ⚠ 這裡原本寫「**改的是記憶體，不是檔案**，所以離開地城再進來牆會回到
// > 原狀，那是 1988 年的行為，照抄」。**那句話是錯的。** 三處改完 tile
// > 之後都緊接著 `122f:28d0(子地圖, 1)`，那一支會把整張地圖
// > （`0x1001` bytes）寫回 `MAP%d.MAP`（`docs/re/95` §3.9）。
// > 密語門那一處在 `0x1a383`，參數是 `(5, 1)` —— **牆是真的開著的**。
//
// **這一支仍然只動記憶體。** 寫回檔案是呼叫端的事（`SaveMap`），
// 因為存到哪裡是專案政策：原版蓋回自己的資料目錄，本專案寫存檔目錄。
func (m *Map) SetTileAt(x, y int, t byte) error {
	if x < 0 || x >= MapWidth || y < 0 || y >= MapHeight {
		return fmt.Errorf("world: 座標 (%d,%d) 超出地圖範圍 [0,%d)x[0,%d)", x, y, MapWidth, MapHeight)
	}
	m.tiles[y*MapWidth+x] = t
	return nil
}

// Tiles 回傳整份 tile 陣列的複本（row-major，索引 = y*MapWidth+x），
// 長度固定 4096。回傳複本是為了避免呼叫端改到 Map 的內部狀態。
func (m *Map) Tiles() [mapTileCount]byte { return m.tiles }

// Encode 把地圖寫回 `MAPn.MAP` 的 4097-byte 形式。
//
// **header 原樣寫回。** 它的語意還沒解（MAP1=0x00、MAP3=0x97、MAP5=0x09），
// 所以不能用 0 補 —— 那會讓寫過的存檔與原版檔案在第一個 byte 就不同。
// `SUM.MAP` 解出來的子地圖沒有這個欄位（固定 0），而那些地圖也不會走到
// 這條路（見 SaveMap）。
func (m *Map) Encode() []byte {
	out := make([]byte, mapFileSize)
	out[0] = m.header
	copy(out[1:], m.tiles[:])
	return out
}

// SaveMap 把地圖寫成 `MAPn.MAP`。
//
// **只給 1／3／5 用。** 原版的存檔那一支（`122f:28d0` 參數非 0）不分編號
// 一律寫 `MAP%d.MAP`，但**載入那一支只有 1／3／5 讀這個檔名**
// （`0x187e4` 的 `cmp ax,1/3/5`），其餘走 `SUM.MAP`。
// 所以在地圖 2／4 推開家具，原版會留下一個永遠不會被讀的 `MAP2.MAP` ——
// 改動就這樣消失了。**那是原版的洞，不是格式**，這裡不複製它：
// 呼叫端只對 1／3／5 存。
//
// 實務上碰不到：三件推得動的家具在地圖 1／3，兩個 `P` 動作也在 1／3，
// 密語門在 5。
func SaveMap(path string, m *Map) error {
	if m == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("world: 建立目錄失敗: %w", err)
	}
	if err := os.WriteFile(path, m.Encode(), 0o644); err != nil {
		return fmt.Errorf("world: 寫入 %s 失敗: %w", path, err)
	}
	return nil
}

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
