package world

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

// 把 26 張子地圖畫成 PNG（城鎮格標紅框），輸出到 workplace/dump/maps/。
//
// 不是斷言，是給人看的。地圖這種東西「解得出數字」和「畫出來是張地圖」
// 差很遠：SUM.MAP 的 RLE 解錯 256 格，格數、值域、size 加總全部照樣通過，
// 畫出來也還是有海岸線有陸塊，只是整張圖從某一欄開始往左滑。本專案的
// 硬規則是視覺產物一律 dump PNG 肉眼比對，這支就是那個入口。
func TestDumpWorldMaps(t *testing.T) {
	dir := origDataDir(t)
	outDir := filepath.Join(repoRoot(t), "workplace", "dump", "maps")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ts, err := gfx.LoadTileset(filepath.Join(dir, string(gfx.NormalTiles)), gfx.NormalTiles)
	if err != nil {
		t.Fatal(err)
	}

	render := func(name string, tiles []byte) {
		img := image.NewRGBA(image.Rect(0, 0, MapWidth*gfx.TileWidth, MapHeight*gfx.TileHeight))
		for y := 0; y < MapHeight; y++ {
			for x := 0; x < MapWidth; x++ {
				tile := ts.Tile(tiles[y*MapWidth+x])
				if tile == nil {
					continue
				}
				at := image.Rect(x*gfx.TileWidth, y*gfx.TileHeight,
					(x+1)*gfx.TileWidth, (y+1)*gfx.TileHeight)
				draw.Draw(img, at, tile, image.Point{}, draw.Src)
			}
		}
		// 城鎮格標紅框，方便對照 gamedata.townSites。
		red := color.RGBA{255, 0, 0, 255}
		for y := 0; y < MapHeight; y++ {
			for x := 0; x < MapWidth; x++ {
				if !gamedata.IsTownTile(tiles[y*MapWidth+x]) {
					continue
				}
				x0, y0 := x*gfx.TileWidth, y*gfx.TileHeight
				for i := 0; i < gfx.TileWidth; i++ {
					img.Set(x0+i, y0, red)
					img.Set(x0+i, y0+gfx.TileHeight-1, red)
					img.Set(x0, y0+i, red)
					img.Set(x0+gfx.TileWidth-1, y0+i, red)
				}
			}
		}
		if err := gfx.SavePNG(filepath.Join(outDir, name+".png"), img); err != nil {
			t.Fatal(err)
		}
	}

	for _, n := range []string{"MAP1.MAP", "MAP3.MAP", "MAP5.MAP"} {
		m, err := LoadMap(filepath.Join(dir, n))
		if err != nil {
			t.Fatal(err)
		}
		tiles := m.Tiles()
		render(fmt.Sprintf("map%c", n[3]), tiles[:])
	}
	sm, err := LoadSumMap(filepath.Join(dir, "SUM.MAP"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range sm.IDs() {
		m, _ := sm.Segment(id)
		tiles := m.Tiles()
		render(fmt.Sprintf("map%d", id), tiles[:])
	}
	t.Logf("26 張子地圖已輸出到 %s", outDir)
}
