package gfx

import (
	"fmt"
	"image"
	"os"
)

// 地形圖塊的尺寸。CGA 精靈圖 .SHP 的 frame 一律是 16×16。
//
// 這個數字是肉眼比對定案的：依 16×32 解碼時，每個 frame 裡是「兩個完整圖形
// 上下疊著」；依 16×16 解碼則每格自成一體。位元組數在兩種讀法下都整除，
// 算術分不出來 —— 見 docs/formats/graphics.md「.SHP frame 尺寸」。
const (
	TileWidth  = 16
	TileHeight = 16
)

// BlackTiles 是 DEMON.SHP 裡**整格純黑**的四個 tile。
//
// 它們不是解碼失敗，也不是缺圖 —— `Tile()` 回傳的是一張正常的 16×16、
// 每個像素都不透明，只是顏色全黑。隊伍站在這種地形上開打時，
// 整片戰場會是黑的，那是忠實呈現原始資料。
//
// 記在這裡是為了防止「修好」它：看到一片黑很容易以為是圖塊沒載到。
var BlackTiles = [...]byte{0, 17, 86, 92}

// TerrainTileCount 是地形圖塊集的圖塊數。
//
// DEMON.SHP／WINTER.SHP 各 6528 bytes ÷ 64 bytes/frame = 102。
// 與 FILES.DAT 可通行性表的有效範圍（tile 0–100，101 項）相符，
// 最後一格 101 未被該表涵蓋。
const TerrainTileCount = 102

// TerrainSet 指定要載入哪一套地形圖塊。
//
// 兩套內容一一對應：同一個 tile 值在兩套裡是同一種地形，
// WinterTiles 是它的雪地版本（枯枝、白地、雪山）。
type TerrainSet string

const (
	// NormalTiles 是常態地表（DEMON.SHP）。
	NormalTiles TerrainSet = "DEMON.SHP"
	// WinterTiles 是雪地版地表（WINTER.SHP）。
	WinterTiles TerrainSet = "WINTER.SHP"
)

// Tileset 是解好的一套地形圖塊，以 tile 值為索引。
type Tileset struct {
	set    TerrainSet
	frames []*image.RGBA
}

// LoadTileset 從指定路徑的 .SHP 檔載入一套 16×16 地形圖塊。
func LoadTileset(path string, set TerrainSet) (*Tileset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gfx: 讀取圖塊集 %s 失敗: %w", path, err)
	}
	frames, err := DecodeCGASpriteSheet(data, TileWidth, TileHeight)
	if err != nil {
		return nil, fmt.Errorf("gfx: 解碼圖塊集 %s 失敗: %w", path, err)
	}
	if len(frames) != TerrainTileCount {
		return nil, fmt.Errorf("gfx: %s 解出 %d 個圖塊，預期 %d",
			path, len(frames), TerrainTileCount)
	}
	return &Tileset{set: set, frames: frames}, nil
}

// Set 回傳這套圖塊屬於常態還是雪地。
func (t *Tileset) Set() TerrainSet { return t.set }

// Len 回傳圖塊數。
func (t *Tileset) Len() int { return len(t.frames) }

// Tile 以 tile 值取出圖塊。超出範圍回傳 nil，由呼叫端決定怎麼畫。
func (t *Tileset) Tile(v byte) *image.RGBA {
	if int(v) >= len(t.frames) {
		return nil
	}
	return t.frames[v]
}
