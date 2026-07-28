// dwroute 在地圖上找一條路，輸出 `tools/playthrough.sh` 吃得下的按鍵腳本。
//
// **這不是作弊捷徑。** 它只做路線規劃 —— 隊伍照樣一步一步走過去，
// 每一格照樣觸發它該觸發的東西。A4 的規則是「不用 debug 捷徑走完主線」
// （`docs/re/64` §3），查地圖找路跟玩家拿張攻略地圖是同一件事。
//
// # 為什麼是 Go 而不是 Python
//
// 前一版是 `tools/route.py`，自己解 `MAP*.MAP` 的檔頭、自己讀
// `FILES.DAT` 的可通行表。兩個問題：
//
//  1. **已經出過一次錯**：檔頭偏移寫成 0（實際是 1），整張圖錯開一格。
//     症狀不明顯 —— 路大致合理，只是偶爾穿牆。
//  2. **`SUM.MAP` 是 RLE 壓縮的**（世界地圖與地圖 2、4 都在裡面）。
//     Python 再寫一份解壓，遲早會與 Go 那份不一致 ——
//     那時候「路規劃錯」與「解壓錯」就分不開了。
//
// 所以搬進 Go，直接用引擎自己的 `world.LoadMap`／`world.LoadSumMap`／
// `gamedata.Tables`。**規劃用的判定與遊戲用的判定是同一份程式碼**，
// 這是這支工具唯一重要的性質。
//
// # 用法
//
//	dwroute -map MAP1.MAP -from 9,32 -to 23,31     # 獨立地城檔
//	dwroute -map 34 -from 55,8 -to 46,7            # SUM.MAP 的段
//	dwroute -map 34 -list-sites                    # 列出這張圖上的地點格
//	dwroute -map MAP1.MAP -from 9,32 -reachable-exits
//	dwroute -world -from 34:28,50 -to 55:47,22     # 跨子地圖（會自己換圖）
//	dwroute -world-reach -from 34:28,50            # 走得到哪幾張子地圖
//	dwroute -world-reach -from 34:28,50 -sailing   # 有船的話到得了哪裡
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

// 方向 → (dx, dy, xdotool 鍵名)。Y 向下為正（螢幕座標）。
var dirs = []struct {
	dx, dy int
	key    string
}{
	{0, -1, "Up"}, {0, 1, "Down"}, {-1, 0, "Left"}, {1, 0, "Right"},
}

func main() {
	dataDir := flag.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	mapArg := flag.String("map", "MAP1.MAP", "地圖：檔名（MAP1.MAP）或 SUM.MAP 的段編號（34）")
	from := flag.String("from", "", "起點 X,Y")
	to := flag.String("to", "", "終點 X,Y")
	listSites := flag.Bool("list-sites", false, "列出這張圖上的城鎮／神殿／學院／出口格")
	reachExits := flag.Bool("reachable-exits", false, "列出從起點走得到的出口，依步數排序")
	worldMode := flag.Bool("world", false,
		"跨子地圖找路。-from／-to 改成 地圖:X,Y（例如 34:28,50）")
	worldReach := flag.Bool("world-reach", false,
		"列出從 -from（地圖:X,Y）走得到的所有子地圖")
	sailing := flag.Bool("sailing", false,
		"跨圖模式：海面也算路（假設全程在船上，是上界不是可行路線）")
	flag.Parse()

	// -world／-world-reach 不先載單一張圖 —— 它們自己按需載，
	// 而 -map 在這兩個模式下沒有意義（起點的地圖編號寫在 -from 裡）。
	if *worldMode || *worldReach {
		tables, err := gamedata.LoadTables(filepath.Join(*dataDir, "FILES.DAT"))
		if err != nil {
			fatal(fmt.Errorf("讀 FILES.DAT：%w", err))
		}
		if *worldReach {
			runWorldReach(worldDataDir(*dataDir), tables, *from, *sailing)
			return
		}
		runWorldRoute(worldDataDir(*dataDir), tables, *from, *to, *sailing)
		return
	}

	m, err := loadMap(*dataDir, *mapArg)
	if err != nil {
		fatal(err)
	}
	tables, err := gamedata.LoadTables(filepath.Join(*dataDir, "FILES.DAT"))
	if err != nil {
		fatal(fmt.Errorf("讀 FILES.DAT：%w", err))
	}

	if *listSites {
		printSites(m, tables)
		return
	}

	fx, fy, err := parseCoord(*from)
	if err != nil {
		fatal(fmt.Errorf("-from：%w", err))
	}

	if *reachExits {
		printReachableExits(*dataDir, m, tables, fx, fy)
		return
	}

	tx, ty, err := parseCoord(*to)
	if err != nil {
		fatal(fmt.Errorf("-to：%w", err))
	}

	path := findPath(m, tables, fx, fy, tx, ty)
	if path == nil {
		fatal(fmt.Errorf("(%d,%d) 走不到 (%d,%d)", fx, fy, tx, ty))
	}
	fmt.Printf("# %s：(%d,%d) → (%d,%d)，%d 步\n", *mapArg, fx, fy, tx, ty, len(path)-1)
	for _, line := range toScript(path) {
		fmt.Println(line)
	}
}

// loadMap 依 -map 的形式載入：純數字走 `world.LoadByID`，其餘當檔名。
//
// **編號 → 地圖的規則只有一份**，在 `world.LoadByID` ——
// 這支原本自己抄了一份，而重複的那份正是上一次 regression 的來源。
func loadMap(dataDir, arg string) (*world.Map, error) {
	id, err := strconv.Atoi(arg)
	if err != nil {
		return world.LoadMap(filepath.Join(dataDir, arg))
	}
	// 存檔目錄傳空字串：路線工具要看的是原始地圖，不是某個人的進度。
	return world.LoadByID("", dataDir, id)
}

type point struct{ x, y int }

// walkable 用引擎的判定：可通行表值不是 0xff 就走得過去。
//
// **只做大地圖／地城那條規則**（深度 0）。子地圖的判定是反過來的
// （`docs/re/22` §2：非 0xff 代表踏出子地圖），但那是「離開」不是「走路」，
// 規劃路線時用不到。
func walkable(m *world.Map, t *gamedata.Tables, p point) bool {
	tile, err := m.TileAt(p.x, p.y)
	if err != nil {
		return false
	}
	return !t.Passability(tile & 0x7f).Blocked()
}

// findPath 是 BFS。回傳含起點終點的座標串列，走不到回 nil。
func findPath(m *world.Map, t *gamedata.Tables, fx, fy, tx, ty int) []point {
	start, goal := point{fx, fy}, point{tx, ty}
	if start == goal {
		return []point{start}
	}
	prev := map[point]point{start: start}
	queue := []point{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range dirs {
			next := point{cur.x + d.dx, cur.y + d.dy}
			if _, seen := prev[next]; seen || !walkable(m, t, next) {
				continue
			}
			prev[next] = cur
			if next == goal {
				return rebuild(prev, start, goal)
			}
			queue = append(queue, next)
		}
	}
	return nil
}

func rebuild(prev map[point]point, start, goal point) []point {
	var rev []point
	for p := goal; p != start; p = prev[p] {
		rev = append(rev, p)
	}
	rev = append(rev, start)
	out := make([]point, len(rev))
	for i, p := range rev {
		out[len(rev)-1-i] = p
	}
	return out
}

// toScript 把座標串列壓成 `rep N 方向`。
func toScript(path []point) []string {
	var out []string
	runKey, runN := "", 0
	for i := 1; i < len(path); i++ {
		dx, dy := path[i].x-path[i-1].x, path[i].y-path[i-1].y
		key := ""
		for _, d := range dirs {
			if d.dx == dx && d.dy == dy {
				key = d.key
			}
		}
		if key == runKey {
			runN++
			continue
		}
		if runKey != "" {
			out = append(out, fmt.Sprintf("rep %d %s", runN, runKey))
		}
		runKey, runN = key, 1
	}
	if runKey != "" {
		out = append(out, fmt.Sprintf("rep %d %s", runN, runKey))
	}
	return out
}

// 地點 tile（`docs/re/74` §1 的四筆分派 + 廢墟）。
const (
	tileTemple  = 0x25
	tileCollege = 0x26
	tileTownA   = 0x2e
	tileTownB   = 0x64
)

// exitPassability 是「這一格是出口」的可通行表值（`docs/re/85`）。
const exitPassability = 0xfd

// printSites 列出這張圖上所有值得走過去的格子。
//
// 出口與地點在畫面上沒有任何標記（tile 外觀有九種變體），
// 玩家靠紙本地圖，驗收時靠這個。
func printSites(m *world.Map, t *gamedata.Tables) {
	for y := 0; y < world.MapHeight; y++ {
		for x := 0; x < world.MapWidth; x++ {
			tile, err := m.TileAt(x, y)
			if err != nil {
				continue
			}
			label := ""
			switch tile & 0x7f {
			case tileTemple:
				label = "神殿"
			case tileCollege:
				label = "學院"
			case tileTownA, tileTownB:
				label = "城鎮"
			}
			if label == "" && t.Passability(tile&0x7f).Raw() == exitPassability {
				label = "出口"
			}
			if label != "" {
				// **座標不留空白**：這一行會被 shell 迴圈拿去餵 `-to`，
				// `%2d` 的補空白會讓 `(46, 7)` 被拆成兩個欄位。
				// 踩過一次，浪費了一輪對照。
				fmt.Printf("%s\t%d,%d\ttile=0x%02x\n", label, x, y, tile&0x7f)
			}
		}
	}
}

// printReachableExits 列出從起點走得到的出口，附步數與目的地。
func printReachableExits(dataDir string, m *world.Map, t *gamedata.Tables, fx, fy int) {
	table, err := world.LoadExits(filepath.Join(dataDir, "EXITS.DAT"))
	if err != nil {
		fatal(fmt.Errorf("讀 EXITS.DAT：%w", err))
	}
	type hit struct {
		steps int
		rec   world.ExitRecord
	}
	var hits []hit
	for _, r := range table.All() {
		tile, err := m.TileAt(int(r.FromX), int(r.FromY))
		if err != nil || t.Passability(tile&0x7f).Raw() != exitPassability {
			// 這一筆不屬於這張圖（判別式是可通行表值，見 `docs/re/85`）。
			continue
		}
		p := findPath(m, t, fx, fy, int(r.FromX), int(r.FromY))
		if p == nil {
			continue
		}
		hits = append(hits, hit{len(p) - 1, r})
	}
	// 步數少的先列。
	for i := range hits {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].steps < hits[i].steps {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
	if len(hits) == 0 {
		fmt.Printf("從 (%d,%d) 走不到這張圖上的任何出口\n", fx, fy)
		return
	}
	fmt.Printf("從 (%d,%d) 走得到的出口：\n", fx, fy)
	for _, h := range hits {
		fmt.Printf("  %3d 步 → (%2d,%2d) 通往圖 %d 的 (%d,%d)\n",
			h.steps, h.rec.FromX, h.rec.FromY, h.rec.ToMap, h.rec.ToX, h.rec.ToY)
	}
}

func parseCoord(s string) (int, int, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("要 X,Y 兩個數字，得到 %q", s)
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "!! %v\n", err)
	os.Exit(1)
}
