package game

import "github.com/wicanr2/demon_winter_cht/internal/rng"

// 移動的墓碑（地點劇情 case 5，地圖 3，`docs/re/100`）
//
// 墓園是一片 11×6 的可通行墓碑（tile `0x56`）。踩上三個入口格之一
// （`0x11`）就印 `The tombstones shift before you`，然後**整片重排**：
// 從檔案重讀地圖 → 隨機挑 30 格墓碑改成阻擋（tile `0x54`）→
// 把三個入口旁邊那一格強制留通。
//
// 所以每踩一次入口就換一座迷宮，而且**永遠不會被當場封死**。
//
// > **與壓牆走廊（case 1／2）同一個路數：只改記憶體，不寫回檔案。**
// > 重排的第一步就是 `122f:28d0(3, 0)` 從 `MAP3.MAP` 重讀 ——
// > 上一次的迷宮因此消失。要是寫回去，墓園會被第一次的隨機結果鎖死。

const (
	// TombstoneOpenTile 是可以走的墓碑（`0x56`，可通行性表 raw `0x06`）。
	TombstoneOpenTile = 0x56
	// TombstoneBlockTile 是重排後擋路的那一種（`0x54`，可通行性 `0xff`）。
	//
	// ⚠ **方向與直覺相反。** `0x56`（墓碑）是**可以走的**，
	// `0x54` 才是擋路的 —— 所以這個事件是「長出 30 塊擋路的石頭」，
	// 不是「移走 30 塊墓碑」。查過可通行性表才敢這樣寫。
	TombstoneBlockTile = 0x54

	// TombstoneMapID 是墓園所在的子地圖（原版 `0x1a18c` 的 `mov ax,0x3`，
	// **寫死的 3**，不是「目前這張圖」）。
	TombstoneMapID = 3

	// 隨機範圍：`x = Roll(11) + 19`、`y = Roll(6) + 56`，而 Roll 是 1 起算，
	// 所以 x ∈ [20,30]、y ∈ [57,62] —— 正好是那片墓碑的外框。
	tombstoneRollX = 11
	tombstoneBaseX = 19
	tombstoneRollY = 6
	tombstoneBaseY = 56

	// TombstoneBlockCount 是要長出幾塊擋路石（`0x1a1f8` 的 `cmp ax,0x1e`）。
	//
	// 原版計數器從 1 起算、`<= 30` 就繼續，所以**成功 30 次**。
	TombstoneBlockCount = 30
)

// tombstoneKeepOpen 是重排完之後強制寫回 `0x56` 的三格
// （`0x1a1fd`／`0x1a207`／`0x1a211` 的 `0xf1d`／`0xe58`／`0xf54`）。
//
// 原版寫的是**線性索引**，換算用 `index = y×64 + x`
// （與壓牆那邊的 `ES:[BX + SI + 0x10]` 同一套，緩衝區第 0 格就是 tile (0,0)，
// **沒有 `MAP*.MAP` 檔頭那一個 byte**）：
//
//	0xf1d = 60×64 + 29 → (29,60)   入口 (28,60) 的東邊一格
//	0xe58 = 57×64 + 24 → (24,57)   入口 (24,58) 的北邊一格
//	0xf54 = 61×64 + 20 → (20,61)   入口 (20,60) 的南邊一格
//
// 三格各對一個入口的**相鄰格**，而且三格都落在隨機範圍內 ——
// 所以它的作用是「不管亂數怎麼擲，踩進入口之後至少有一步可以走」。
var tombstoneKeepOpen = [3]struct{ X, Y int }{
	{29, 60},
	{24, 57},
	{20, 61},
}

// TombstoneShift 把墓園重排一次。
//
// 呼叫端要**先**從檔案重讀 `MAP3.MAP`（原版 `do { load(3,0) } while (…) `），
// 這一支只做重讀之後那三段：長石頭、留通、回報。
// 分開是因為「重讀地圖」是 I/O，規則層不碰檔案。
//
// 回傳實際長出來的石頭座標，順序就是擲點順序（給軌跡與測試看的）。
func TombstoneShift(r *rng.RNG, m TileWriter) []struct{ X, Y int } {
	if r == nil || m == nil {
		return nil
	}
	out := make([]struct{ X, Y int }, 0, TombstoneBlockCount)
	// 原版是「擲點 → 不是墓碑就重擲」的無界迴圈。66 格裡要挑 30 格，
	// 到後面命中率會掉，但**永遠有得挑**（三格強制留通是在迴圈之後），
	// 所以不會空轉到底。照抄這個形狀，不改成洗牌。
	for len(out) < TombstoneBlockCount {
		x := r.Roll(tombstoneRollX) + tombstoneBaseX
		y := r.Roll(tombstoneRollY) + tombstoneBaseY
		t, err := m.TileAt(x, y)
		if err != nil || t != TombstoneOpenTile {
			continue
		}
		if m.SetTileAt(x, y, TombstoneBlockTile) != nil {
			continue
		}
		out = append(out, struct{ X, Y int }{x, y})
	}
	for _, p := range tombstoneKeepOpen {
		_ = m.SetTileAt(p.X, p.Y, TombstoneOpenTile)
	}
	return out
}
