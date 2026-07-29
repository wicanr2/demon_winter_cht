// Command mapwindow 列出指定地圖座標周圍的實際 tile index。
//
// Modern Icon 美術必須逐索引重畫；不能只看截圖猜「這看起來像森林」。
// 這支工具把 9×9 runtime 視窗與索引頻率印出，供美術 manifest 與同場景驗收使用。
package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

func main() {
	data := flag.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	mapID := flag.Int("map", 34, "子地圖編號")
	x := flag.Int("x", 28, "中心 X")
	y := flag.Int("y", 50, "中心 Y")
	find := flag.String("find-tiles", "", "掃描所有地圖，找含指定十六進位 tile 的視窗")
	inventory := flag.Bool("inventory", false, "列出所有地圖實際使用的 tile、總數與第一個座標")
	minMap := flag.Int("min-map", 0, "掃描時只納入此編號以上的地圖（0 表示不限制）")
	maxMap := flag.Int("max-map", 0, "掃描時只納入此編號以下的地圖（0 表示不限制）")
	limit := flag.Int("limit", 12, "find-tiles 最多列出幾個視窗")
	flag.Parse()

	if *inventory {
		if err := printInventory(*data, *minMap, *maxMap); err != nil {
			panic(err)
		}
		return
	}
	if *find != "" {
		targets, err := parseTileSet(*find)
		if err != nil {
			panic(err)
		}
		if err := findWindows(*data, targets, *minMap, *maxMap, *limit); err != nil {
			panic(err)
		}
		return
	}

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

type tileUse struct {
	count       int
	mapID, x, y int
}

func printInventory(dataDir string, minMap, maxMap int) error {
	sm, err := world.LoadSumMap(filepath.Join(dataDir, "SUM.MAP"))
	if err != nil {
		return err
	}
	ids := append([]int{1, 3, 5}, sm.IDs()...)
	sort.Ints(ids)
	uses := map[byte]tileUse{}
	for _, id := range ids {
		if !mapInRange(id, minMap, maxMap) {
			continue
		}
		m, err := world.LoadByID("", dataDir, id)
		if err != nil {
			return err
		}
		for y := 0; y < game.MapHeight; y++ {
			for x := 0; x < game.MapWidth; x++ {
				t, err := m.TileAt(x, y)
				if err != nil {
					return err
				}
				t &= 0x7f
				u := uses[t]
				if u.count == 0 {
					u.mapID, u.x, u.y = id, x, y
				}
				u.count++
				uses[t] = u
			}
		}
	}
	keys := make([]int, 0, len(uses))
	for key := range uses {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)
	for _, key := range keys {
		u := uses[byte(key)]
		fmt.Printf("%02x count=%-7d first=map%-2d (%2d,%2d)\n",
			key, u.count, u.mapID, u.x, u.y)
	}
	return nil
}

type windowHit struct {
	mapID, x, y, score int
	counts             map[byte]int
}

func parseTileSet(s string) (map[byte]bool, error) {
	out := map[byte]bool{}
	for _, raw := range strings.Split(s, ",") {
		raw = strings.TrimPrefix(strings.TrimSpace(raw), "0x")
		n, err := strconv.ParseUint(raw, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("無效 tile %q: %w", raw, err)
		}
		out[byte(n)] = true
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("tile 清單是空的")
	}
	return out, nil
}

func findWindows(dataDir string, targets map[byte]bool, minMap, maxMap, limit int) error {
	sm, err := world.LoadSumMap(filepath.Join(dataDir, "SUM.MAP"))
	if err != nil {
		return err
	}
	ids := append([]int{1, 3, 5}, sm.IDs()...)
	sort.Ints(ids)
	var hits []windowHit
	halfX, halfY := layout.ViewTilesX/2, layout.ViewTilesY/2
	for _, id := range ids {
		if !mapInRange(id, minMap, maxMap) {
			continue
		}
		m, err := world.LoadByID("", dataDir, id)
		if err != nil {
			return err
		}
		for cy := halfY; cy < game.MapHeight-halfY; cy++ {
			for cx := halfX; cx < game.MapWidth-halfX; cx++ {
				counts := map[byte]int{}
				score := 0
				for dy := -halfY; dy <= halfY; dy++ {
					for dx := -halfX; dx <= halfX; dx++ {
						t, err := m.TileAt(cx+dx, cy+dy)
						if err != nil {
							return err
						}
						t &= 0x7f
						if targets[t] {
							counts[t]++
							score++
						}
					}
				}
				if score > 0 {
					hits = append(hits, windowHit{id, cx, cy, score, counts})
				}
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		if hits[i].mapID != hits[j].mapID {
			return hits[i].mapID < hits[j].mapID
		}
		if hits[i].y != hits[j].y {
			return hits[i].y < hits[j].y
		}
		return hits[i].x < hits[j].x
	})
	if limit < 0 || limit > len(hits) {
		limit = len(hits)
	}
	for _, h := range hits[:limit] {
		keys := make([]int, 0, len(h.counts))
		for k := range h.counts {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		fmt.Printf("map=%d center=(%d,%d) score=%d counts:", h.mapID, h.x, h.y, h.score)
		for _, k := range keys {
			fmt.Printf(" %02x=%d", k, h.counts[byte(k)])
		}
		fmt.Println()
	}
	return nil
}

func mapInRange(id, minMap, maxMap int) bool {
	return (minMap == 0 || id >= minMap) && (maxMap == 0 || id <= maxMap)
}
