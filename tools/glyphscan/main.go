// glyphscan 找出三個緋紅符印在世界地圖上的座標（`docs/re/59`）。
//
// 符印圖塊是 0x63，只出現在 SUM.MAP 解壓後的子地圖裡。
// 用途是驗收：把玩家用 -map/-x/-y 直接放到符印上，才驗得到 UNCURSE。
package main

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

func main() {
	sm, err := world.LoadSumMap("workplace/orig/demwin/DEM_DATA/SUM.MAP")
	if err != nil {
		panic(err)
	}
	for _, id := range sm.IDs() {
		m, ok := sm.Segment(id)
		if !ok {
			continue
		}
		t := m.Tiles()
		for i, b := range t {
			if b == 0x63 {
				fmt.Printf("子地圖 %d (0x%02x)  X=%d Y=%d\n", id, id, i%64, i/64)
			}
		}
	}
}
