package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 跨子地圖找路。
//
// **為什麼單張圖的 BFS 不夠。** 世界是 7×7 格子地圖拼起來的，
// 走到第 3／60 格會換到隔壁那張（`world.CrossEdge`）。主線的三個符印在
// 東南角的 55／56／66，起始位置在 34 —— 那是隔了兩三張圖的距離，
// 單張圖的規劃器問不出「走不走得到」，只問得出「這張圖裡走不走得到」。
//
// 這正是 `-glyphs` 旗標一直被當成「非有不可」的原因（`docs/re/64` §3）：
// 缺的不是捷徑，是**跨圖的路線規劃**（引擎那一側的邊界換圖也一直沒接，
// 見 `cmd/demonwinter/worldedge.go`）。
//
// # 判定與遊戲同一份
//
// 走不走得過去用 `gamedata.Tables.Passability`，換圖用 `world.CrossEdge` ——
// 兩支都是引擎自己在用的那一份。這支工具唯一重要的性質就是這個。
//
// # 兩個刻意的限制
//
//  1. **陸路與航海分開問**（`-sailing`）。海面在可通行表裡是擋住的，
//     所以預設規劃出來的一定是走得到的陸地。`-sailing` 打開之後海面
//     也算路，用的是 `game.IsOcean` —— 引擎 `World.Walk` 判「搭船時
//     海面變成路」用的同一支。
//
//     ⚠ `-sailing` 是**上界**不是可行路線：它假設全程都在船上，
//     沒有模擬上船（要走到船停的那一格）與下船。要問「這條路真的走得完嗎」
//     還是得實跑。它回答的是另一個問題 ——「就算有船，到得了嗎」。
//  2. **出口是強制的，不是選項。** 踩到 `EXITS.DAT` 的格子就被送到另一張圖
//     （引擎 `checkExit`），所以規劃時不能把出口格當普通路面走過去。
//  3. **落點不檢查可通行性。** 原版換圖是直接寫座標（`0x17101`
//     那幾行只設 X／Y 與 `+0xa3`），不查落點那一格。照做 ——
//     如果落點是水，玩家在原版也會被放到水上。

// wpoint 是「哪張子地圖的哪一格」。
type wpoint struct {
	mapID, x, y int
}

func (p wpoint) String() string { return fmt.Sprintf("%d:%d,%d", p.mapID, p.x, p.y) }

// mapCache 把載過的子地圖留著 —— 跨圖 BFS 會反覆問同一張圖，
// 每次重解 RLE 會讓一次規劃跑上幾十秒。
type mapCache struct {
	dataDir string
	tables  *gamedata.Tables
	maps    map[int]*world.Map
	missing map[int]bool
	// sailing 為真時海面也算路（`-sailing`）。
	sailing bool
	// exits 是 `EXITS.DAT`。踩到出口格是**強制**換圖，不是選項 ——
	// 所以規劃時也要照著走，否則排出來的路線會「從出口上面走過去」，
	// 實跑時隊伍在那一格就被送走了。
	exits *world.ExitTable

	// avoidSites 為真時繞開城鎮／神殿／學院／廢墟格。
	//
	// **為什麼要繞。** 踩上去會開啟模態畫面，而腳本後面那些方向鍵
	// 會全部落到城鎮選單裡 —— 隊伍留在原地，路線靜默歪掉，
	// 而且**完全沒有錯誤訊息**（`tools/playthrough.sh` 開頭記的第一個坑）。
	// 玩家自己走的時候也會繞，除非那裡就是目的地。
	avoidSites bool
	// goal 是終點：終點本身是城鎮格時當然要踩上去。
	goal wpoint
}

func newMapCache(dataDir string, tables *gamedata.Tables, sailing bool) *mapCache {
	c := &mapCache{
		dataDir: dataDir,
		tables:  tables,
		maps:    map[int]*world.Map{},
		missing: map[int]bool{},
		sailing: sailing,
	}
	// 讀不到就當沒有出口 —— 規劃退化成「只走世界網格」，
	// 而不是整支工具失敗。
	if t, err := world.LoadExits(filepath.Join(dataDir, "EXITS.DAT")); err == nil {
		c.exits = t
	}
	return c
}

// takeExit 看那一格是不是出口，是就回傳出口的另一端。
//
// 判別式與引擎同一條（`docs/re/85`）：**可通行表值 == 0xfd**，不是 tile 值。
// `EXITS.DAT` 沒有「來源地圖」欄位，同一組 (X,Y) 在別的地圖上不是 0xfd
// 就不算出口 —— 這也是 55 筆座標怎麼分給 26 張地圖的答案。
func (c *mapCache) takeExit(p wpoint) (wpoint, bool) {
	if c.exits == nil {
		return p, false
	}
	m := c.get(p.mapID)
	if m == nil {
		return p, false
	}
	tile, err := m.TileAt(p.x, p.y)
	if err != nil || c.tables.Passability(tile&0x7f).Raw() != exitPassability {
		return p, false
	}
	rec, ok := c.exits.Lookup(byte(p.x), byte(p.y))
	if !ok {
		return p, false
	}
	next := wpoint{int(rec.ToMap), int(rec.ToX), int(rec.ToY)}
	if c.get(next.mapID) == nil {
		return p, false
	}
	return next, true
}

// passable 是這支工具的移動判定：可通行表說得過去，或者在船上且那一格是海。
func (c *mapCache) passable(m *world.Map, p point) bool {
	if walkable(m, c.tables, p) {
		return true
	}
	if !c.sailing {
		return false
	}
	tile, err := m.TileAt(p.x, p.y)
	if err != nil {
		return false
	}
	return game.IsOcean(tile & 0x7f)
}

// isSite 回報那一格踩上去會開啟模態畫面。
//
// tile 值與引擎 `game.SiteFor` 那一組相同（`docs/re/74` §1 的四筆分派 ＋ 廢墟）。
// **這裡不查廢墟旗標**：規劃時寧可多繞一格，也不要排出一條會被城鎮吃掉的路。
func (c *mapCache) isSite(p wpoint) bool {
	m := c.get(p.mapID)
	if m == nil {
		return false
	}
	tile, err := m.TileAt(p.x, p.y)
	if err != nil {
		return false
	}
	switch tile & 0x7f {
	case tileTemple, tileCollege, tileTownA, tileTownB, 0x5b:
		return true
	}
	return false
}

// get 回傳那張子地圖；不存在（7×7 裡沒存檔的那 28 格）回 nil。
func (c *mapCache) get(id int) *world.Map {
	if m, ok := c.maps[id]; ok {
		return m
	}
	if c.missing[id] {
		return nil
	}
	m, err := world.LoadByID("", c.dataDir, id)
	if err != nil {
		c.missing[id] = true
		return nil
	}
	c.maps[id] = m
	return m
}

// step 算出從 cur 往某個方向走一步的結果。ok 為 false 代表走不過去。
func (c *mapCache) step(cur wpoint, dx, dy int) (wpoint, bool) {
	m := c.get(cur.mapID)
	if m == nil {
		return cur, false
	}
	nx, ny := cur.x+dx, cur.y+dy
	if !c.passable(m, point{nx, ny}) {
		return cur, false
	}
	res := world.CrossEdge(cur.mapID, nx, ny)
	if res.Blocked {
		// 世界邊緣。原版印「船員拒絕再往前航行」並把座標退回去，
		// 等於這一步不存在。
		return cur, false
	}
	next := wpoint{res.MapID, res.X, res.Y}
	if res.Crossed && c.get(next.mapID) == nil {
		// 換到一張不存在的子地圖 —— 那格在 7×7 網格裡沒有存檔。
		// 原版會載出垃圾；規劃時當成走不過去。
		return cur, false
	}
	if c.avoidSites && next != c.goal && c.isSite(next) {
		return cur, false
	}
	// 出口排在邊界換圖**之前**判 —— 引擎的順序就是這樣
	// （回傳碼 `0x14` 在 `0x15` 之前，見 `docs/re/58` §2）。
	// 這裡在 CrossEdge 之後才問是因為兩者不會同時成立：
	// 邊界那一圈沒有出口格。
	if e, ok := c.takeExit(next); ok {
		return e, true
	}
	return next, true
}

// findWorldPath 是跨子地圖的 BFS。
func (c *mapCache) findWorldPath(start, goal wpoint) []wpoint {
	if start == goal {
		return []wpoint{start}
	}
	prev := map[wpoint]wpoint{start: start}
	queue := []wpoint{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range dirs {
			next, ok := c.step(cur, d.dx, d.dy)
			if !ok {
				continue
			}
			if _, seen := prev[next]; seen {
				continue
			}
			prev[next] = cur
			if next == goal {
				return rebuildWorld(prev, start, goal)
			}
			queue = append(queue, next)
		}
	}
	return nil
}

func rebuildWorld(prev map[wpoint]wpoint, start, goal wpoint) []wpoint {
	var rev []wpoint
	for p := goal; p != start; p = prev[p] {
		rev = append(rev, p)
	}
	rev = append(rev, start)
	out := make([]wpoint, len(rev))
	for i, p := range rev {
		out[len(rev)-1-i] = p
	}
	return out
}

// toWorldScript 把跨圖路線壓成 `rep N 方向`，換圖處插一行註解。
//
// **換圖不需要按鍵** —— 走到邊界格就自動換了。註解是給讀腳本的人對照用的，
// 因為腳本裡連續的 `rep 40 Right` 看不出中間跨了兩張圖。
func toWorldScript(path []wpoint) []string {
	var out []string
	runKey, runN := "", 0
	flush := func() {
		if runKey != "" {
			out = append(out, fmt.Sprintf("rep %d %s", runN, runKey))
		}
		runKey, runN = "", 0
	}
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		key := ""
		if b.mapID != a.mapID {
			// 換圖那一步的方向從**編號差**看，座標已經 wrap 過了。
			switch b.mapID - a.mapID {
			case -10:
				key = "Left"
			case 10:
				key = "Right"
			case -1:
				key = "Up"
			case 1:
				key = "Down"
			}
		} else {
			for _, d := range dirs {
				if d.dx == b.x-a.x && d.dy == b.y-a.y {
					key = d.key
				}
			}
		}
		if key != runKey {
			flush()
			runKey = key
		}
		runN++
		if b.mapID != a.mapID {
			flush()
			out = append(out, fmt.Sprintf("# → 子地圖 %d 的 (%d,%d)", b.mapID, b.x, b.y))
		}
	}
	flush()
	return out
}

// parseWorldCoord 讀 `34:28,50`（子地圖編號 : X , Y）。
func parseWorldCoord(s string) (wpoint, error) {
	i := strings.Index(s, ":")
	if i < 0 {
		return wpoint{}, fmt.Errorf("要 地圖:X,Y 的形式（例如 34:28,50），得到 %q", s)
	}
	id, err := strconv.Atoi(strings.TrimSpace(s[:i]))
	if err != nil {
		return wpoint{}, fmt.Errorf("地圖編號：%w", err)
	}
	x, y, err := parseCoord(s[i+1:])
	if err != nil {
		return wpoint{}, err
	}
	return wpoint{id, x, y}, nil
}

// runWorldRoute 是 `-world` 的入口。
func runWorldRoute(dataDir string, tables *gamedata.Tables, fromArg, toArg string, sailing bool) {
	start, err := parseWorldCoord(fromArg)
	if err != nil {
		fatal(fmt.Errorf("-from：%w", err))
	}
	goal, err := parseWorldCoord(toArg)
	if err != nil {
		fatal(fmt.Errorf("-to：%w", err))
	}
	c := newMapCache(dataDir, tables, sailing)
	c.avoidSites, c.goal = true, goal
	path := c.findWorldPath(start, goal)
	if path == nil {
		fatal(fmt.Errorf("%s 走不到 %s（%s）", start, goal, routeMode(sailing)))
	}
	crossings := 0
	for i := 1; i < len(path); i++ {
		if path[i].mapID != path[i-1].mapID {
			crossings++
		}
	}
	fmt.Printf("# %s → %s：%d 步，跨圖 %d 次\n", start, goal, len(path)-1, crossings)
	for _, line := range toWorldScript(path) {
		fmt.Println(line)
	}
}

// runWorldReach 列出從起點**走得到的所有子地圖**，附最短步數。
//
// 這是問「A6 的路線存在嗎」最直接的一問：主線要到 55／56／66，
// 它們在不在名單裡，一眼看得出來。
func runWorldReach(dataDir string, tables *gamedata.Tables, fromArg string, sailing bool) {
	start, err := parseWorldCoord(fromArg)
	if err != nil {
		fatal(fmt.Errorf("-from：%w", err))
	}
	c := newMapCache(dataDir, tables, sailing)
	dist := map[wpoint]int{start: 0}
	best := map[int]int{start.mapID: 0}
	queue := []wpoint{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range dirs {
			next, ok := c.step(cur, d.dx, d.dy)
			if !ok {
				continue
			}
			if _, seen := dist[next]; seen {
				continue
			}
			dist[next] = dist[cur] + 1
			if b, ok := best[next.mapID]; !ok || dist[next] < b {
				best[next.mapID] = dist[next]
			}
			queue = append(queue, next)
		}
	}
	var ids []int
	for id := range best {
		ids = append(ids, id)
	}
	for i := range ids {
		for j := i + 1; j < len(ids); j++ {
			if best[ids[j]] < best[ids[i]] {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	fmt.Printf("從 %s 走得到 %d 張子地圖（%s）：\n", start, len(ids), routeMode(sailing))
	for _, id := range ids {
		fmt.Printf("  %4d 步 → 子地圖 %d\n", best[id], id)
	}
}

// routeMode 是輸出裡那句「用什麼判定算的」。
func routeMode(sailing bool) string {
	if sailing {
		return "含航海，全程假設在船上"
	}
	return "只算陸路"
}

// worldDataDir 只是讓 -world 也吃得到 -data 的預設值。
func worldDataDir(dataDir string) string { return filepath.Clean(dataDir) }
