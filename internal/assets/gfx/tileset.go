package gfx

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
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
// FILES.DAT 可通行性表與所有地圖只消費 tile 0–100；最後一格 101 是
// runtime 未使用的額外水紋 frame，仍保留解碼（docs/re/118）。
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

// VideoMode 是素材版本。原版同時出貨兩套美術，副檔名不同、內容一一對應：
//
//	ModeEGA  `.SHE`  16 色，檔內 frame 16×28，**載入時水平加倍**成 32×28
//	ModeCGA  `.SHP`   4 色，frame 16×16，不加倍
//
// 這不是畫質選項，是兩個不同的原版版本 —— 完整性原則要求兩套都能跑
// （`rulebook/83`）。預設 EGA，因為那是絕大多數玩家當年看到的畫面。
type VideoMode int

const (
	// ModeEGA 是 16 色版（`.SHE` ＋ 原版自訂調色盤 GamePalette）。
	ModeEGA VideoMode = iota
	// ModeCGA 是 4 色版（`.SHP`）。
	ModeCGA
)

// EGA 地形圖塊的尺寸。檔內 16 寬，載入時逐 bit 複製兩次變成 32 寬
// （原版 `FUN_217b_0adf`／`0b19`，見 docs/formats/graphics.md §4.2）。
const (
	EGATileFileWidth = 16
	EGATileWidth     = 32
	EGATileHeight    = 28
)

// FileName 回傳這一套在該模式下的檔名。原版的素材載入器就是這樣改副檔名的
// （`FUN_1d9f_0a8b`：EGA 模式把 `.SHP` 的末字元換成 `E`）。
func (m VideoMode) FileName(set TerrainSet) string {
	if m == ModeEGA {
		return strings.TrimSuffix(string(set), "P") + "E"
	}
	return string(set)
}

// Tileset 是解好的一套地形圖塊，以 tile 值為索引。
type Tileset struct {
	set    TerrainSet
	mode   VideoMode
	w, h   int
	frames []*image.RGBA
}

// FrameSize 回傳一格的**顯示**尺寸（EGA 已含水平加倍）。
func (t *Tileset) FrameSize() (int, int) { return t.w, t.h }

// Mode 回傳這一套是哪個版本的素材。
func (t *Tileset) Mode() VideoMode { return t.mode }

// LoadTilesetMode 依 mode 載入一套地形圖塊。dir 是原版資料目錄。
//
// EGA 走 `.SHE` ＋ **GamePalette**（不是標準 16 色表 —— 那是本專案的舊錯誤，
// 見 GamePalette 的說明），並在解完之後做原版那道水平加倍。
func LoadTilesetMode(dir string, set TerrainSet, mode VideoMode) (*Tileset, error) {
	path := filepath.Join(dir, mode.FileName(set))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gfx: 讀取圖塊集 %s 失敗: %w", path, err)
	}
	var frames []*image.RGBA
	w, h := TileWidth, TileHeight
	if mode == ModeEGA {
		pal := GamePalette
		frames, err = decodeEGATiles(data, &pal)
		w, h = EGATileWidth, EGATileHeight
	} else {
		frames, err = DecodeCGASpriteSheet(data, TileWidth, TileHeight)
	}
	if err != nil {
		return nil, fmt.Errorf("gfx: 解碼圖塊集 %s 失敗: %w", path, err)
	}
	if len(frames) != TerrainTileCount {
		return nil, fmt.Errorf("gfx: %s 解出 %d 個圖塊，預期 %d",
			path, len(frames), TerrainTileCount)
	}
	return &Tileset{set: set, mode: mode, w: w, h: h, frames: frames}, nil
}

// decodeEGATiles 解 `.SHE` 並套用原版調色盤，回傳已水平加倍的 32×28 frame。
//
// **加倍放在這裡而不是繪製時**，理由與原版相同：原版是在載入時就地把緩衝區
// 展開成兩倍寬（`FUN_217b_0adf`），之後所有 blit 都對加倍後的緩衝區定址。
// 繪製時才放大會讓「一格幾像素」在兩個模式下算法不同，呈現層要多一條分支。
func decodeEGATiles(data []byte, pal *[16]byte) ([]*image.RGBA, error) {
	half, err := DecodeEGASpriteSheetPalette(data,
		EGATileFileWidth, EGATileHeight, EGAPlanesRowBlocks, pal)
	if err != nil {
		return nil, err
	}
	out := make([]*image.RGBA, len(half))
	for i, src := range half {
		dst := image.NewRGBA(image.Rect(0, 0, EGATileWidth, EGATileHeight))
		for y := 0; y < EGATileHeight; y++ {
			for x := 0; x < EGATileFileWidth; x++ {
				c := src.RGBAAt(x, y)
				dst.SetRGBA(x*2, y, c)
				dst.SetRGBA(x*2+1, y, c)
			}
		}
		out[i] = dst
	}
	return out, nil
}

// LoadTileset 從指定路徑的 .SHP 檔載入一套 16×16 CGA 地形圖塊。
//
// 保留給既有測試與只指定單一檔案路徑的呼叫端；新程式碼用 LoadTilesetMode。
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
	return &Tileset{set: set, mode: ModeCGA, w: TileWidth, h: TileHeight, frames: frames}, nil
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
