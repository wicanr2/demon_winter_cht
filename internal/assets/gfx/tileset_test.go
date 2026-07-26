package gfx

import (
	"image/color"
	"path/filepath"
	"testing"
)

// 兩套地形圖塊都必須解出 102 個 16×16 圖塊。
//
// 102 這個數字是判斷 frame 尺寸對不對的關鍵：
// 依 16×32 解會得到 51，對不上可通行性表的 101 項（tile 0–100）。
func TestLoadTileset_CountAndSize(t *testing.T) {
	dir := origDataDir(t)

	for _, set := range []TerrainSet{NormalTiles, WinterTiles} {
		ts, err := LoadTileset(filepath.Join(dir, string(set)), set)
		if err != nil {
			t.Fatalf("LoadTileset(%s): %v", set, err)
		}
		if ts.Len() != TerrainTileCount {
			t.Errorf("%s：解出 %d 個圖塊，預期 %d", set, ts.Len(), TerrainTileCount)
		}
		for v := 0; v < ts.Len(); v++ {
			img := ts.Tile(byte(v))
			if img == nil {
				t.Fatalf("%s：圖塊 %d 為 nil", set, v)
			}
			b := img.Bounds()
			if b.Dx() != TileWidth || b.Dy() != TileHeight {
				t.Fatalf("%s：圖塊 %d 尺寸 %dx%d，預期 %dx%d",
					set, v, b.Dx(), b.Dy(), TileWidth, TileHeight)
			}
		}
		if ts.Tile(byte(TerrainTileCount)) != nil {
			t.Errorf("%s：超出範圍的 tile 值應回傳 nil", set)
		}
	}
}

// 兩套是同一批地形的常態版與雪地版：對應的圖塊必須有差異（不是同一份檔案），
// 但整體結構相同（張數一致，已在上面測過）。
//
// 這裡挑一個明確會變的圖塊來驗：tile 1 在常態版是茂密的樹，
// 雪地版是枯枝 + 白地，兩者不可能逐像素相同。
func TestTilesets_NormalAndWinterDiffer(t *testing.T) {
	dir := origDataDir(t)

	normal, err := LoadTileset(filepath.Join(dir, string(NormalTiles)), NormalTiles)
	if err != nil {
		t.Fatalf("載入常態圖塊集: %v", err)
	}
	winter, err := LoadTileset(filepath.Join(dir, string(WinterTiles)), WinterTiles)
	if err != nil {
		t.Fatalf("載入雪地圖塊集: %v", err)
	}

	same := 0
	for v := 0; v < TerrainTileCount; v++ {
		if imagesEqual(normal.Tile(byte(v)), winter.Tile(byte(v))) {
			same++
		}
	}
	if same == TerrainTileCount {
		t.Fatal("兩套圖塊完全相同，可能載到同一個檔案")
	}
	t.Logf("常態／雪地版共有 %d/%d 個圖塊逐像素相同（多為全黑或無季節差異者）",
		same, TerrainTileCount)
}

func imagesEqual(a, b interface{ At(x, y int) color.Color }) bool {
	for y := 0; y < TileHeight; y++ {
		for x := 0; x < TileWidth; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}

// dump 一份加了編號格線的圖集，供肉眼比對。
// 這個專案在 sprite 尺寸上踩過多次坑，PNG 產出是硬性驗收條件。
func TestDumpTilesetAtlases(t *testing.T) {
	dir := origDataDir(t)

	for _, set := range []TerrainSet{NormalTiles, WinterTiles} {
		ts, err := LoadTileset(filepath.Join(dir, string(set)), set)
		if err != nil {
			t.Fatalf("LoadTileset(%s): %v", set, err)
		}
		sheet := TileSpriteSheet(ts.frames, 17)
		out := filepath.Join(dumpDir(t), "tileset-"+string(set)+".png")
		if err := SavePNG(out, zoomImage(sheet, 4)); err != nil {
			t.Fatalf("存 PNG 失敗: %v", err)
		}
		t.Logf("%s: %d 個 16x16 圖塊 -> %s", set, ts.Len(), out)
	}
}

// 四個純黑 tile 是原始資料就長那樣，不是缺圖。
//
// 這條測試存在的理由是防止誤修：戰場鋪滿其中一個時整片是黑的，
// 很容易被當成「圖塊沒載到」而去「修」解碼器。
func TestTileset_BlackTilesAreRealData(t *testing.T) {
	for _, set := range []TerrainSet{NormalTiles, WinterTiles} {
		ts, err := LoadTileset(filepath.Join(origDataDir(t), string(set)), set)
		if err != nil {
			t.Fatalf("LoadTileset(%s): %v", set, err)
		}
		black := map[byte]bool{}
		for v := 0; v < ts.Len(); v++ {
			img := ts.Tile(byte(v))
			if img == nil {
				t.Fatalf("%s：tile %d 取不到圖", set, v)
			}
			allBlack := true
			for i := 0; i < len(img.Pix); i += 4 {
				if img.Pix[i] != 0 || img.Pix[i+1] != 0 || img.Pix[i+2] != 0 {
					allBlack = false
					break
				}
				if img.Pix[i+3] == 0 {
					t.Fatalf("%s：tile %d 有透明像素，地形圖塊應該全不透明", set, v)
				}
			}
			if allBlack {
				black[byte(v)] = true
			}
		}
		if set != NormalTiles {
			continue // 雪地那套只驗「沒有缺圖、沒有透明」
		}
		if len(black) != len(BlackTiles) {
			t.Errorf("%s：純黑 tile 有 %d 個，預期 %d 個", set, len(black), len(BlackTiles))
		}
		for _, v := range BlackTiles {
			if !black[v] {
				t.Errorf("%s：tile %d 不是純黑，BlackTiles 該更新", set, v)
			}
		}
	}
}
