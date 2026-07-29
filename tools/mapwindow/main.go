// Command mapwindow 列出指定地圖座標周圍的實際 tile index。
//
// Modern Icon 美術必須逐索引重畫；不能只看截圖猜「這看起來像森林」。
// 這支工具把 9×9 runtime 視窗與索引頻率印出，供美術 manifest 與同場景驗收使用。
package main

import (
	"flag"
	"fmt"
	"sort"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

func main() {
	data := flag.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	mapID := flag.Int("map", 34, "子地圖編號")
	x := flag.Int("x", 28, "中心 X")
	y := flag.Int("y", 50, "中心 Y")
	flag.Parse()

	m, err := world.LoadByID("", *data, *mapID)
	if err != nil {
		panic(err)
	}
	halfX, halfY := layout.ViewTilesX/2, layout.ViewTilesY/2
	counts := make(map[byte]int)
	fmt.Printf("map=%d center=(%d,%d) view=%dx%d\n", *mapID, *x, *y,
		layout.ViewTilesX, layout.ViewTilesY)
	for dy := 0; dy < layout.ViewTilesY; dy++ {
		for dx := 0; dx < layout.ViewTilesX; dx++ {
			mx, my := *x-halfX+dx, *y-halfY+dy
			if mx < 0 || mx >= game.MapWidth || my < 0 || my >= game.MapHeight {
				fmt.Print(" --")
				continue
			}
			t, err := m.TileAt(mx, my)
			if err != nil {
				panic(err)
			}
			t &= 0x7f
			counts[t]++
			fmt.Printf(" %02x", t)
		}
		fmt.Println()
	}
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	fmt.Print("counts:")
	for _, k := range keys {
		fmt.Printf(" %02x=%d", k, counts[byte(k)])
	}
	fmt.Println()
}
