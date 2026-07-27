package game

// 壓牆走廊（地點劇情 case 1／2，地圖 1，`docs/re/65` §3、`docs/re/83` §2）
//
// 這是全遊戲唯一一個**會直接殺死全隊**的場景機關，而且兩個 case 是一組：
//
//	case 1  走廊兩端（(15,38)／(23,38)）—— 重新載入 MAP1.MAP，**牆全部復位**，
//	        印 `You hear the whirring of massive machinery`
//	case 2  走廊中段 —— 兩側的牆各往中間推進一列；
//	        推完之後隊伍腳下若變成牆 → `Your party has been crushed by the walls.`
//
// 所以玩法是：踏進走廊之後牆一步一步逼近，**要在被夾到之前跑到另一端**，
// 而另一端那一格會把牆重置。走錯方向就死在裡面。
//
// > **牆只寫在記憶體裡，這一次是真的。** case 2 只寫 `ds:0x4c94`，
// > 沒有呼叫存檔那一支（`122f:28d0`）—— 與 `M`／密語門不同
// > （那兩個會寫回 `MAPn.MAP`，`docs/re/95` §3.9）。
// > 復位靠的正是 case 1 從**檔案**重讀一次，所以「不存」是這個機關的一部分。

const (
	// CrushWallTile 是牆的 tile 值（原版 `0x19fcc` 的 `mov …,0x0d`）。
	CrushWallTile = 0x0d

	// crushScanTop／crushScanBottom 是掃描範圍：從第 35 列往下找第一列
	// 「還不是牆」的（`0x19f89` 的 `0x23`、`0x19fb0` 的 `cmp …,0x26`）。
	crushScanTop    = 0x23 // 35
	crushScanBottom = 0x26 // 38

	// crushColLeft／crushColRight 是走廊的寬度（`0x19fb6` 的 `0x10`、
	// `0x19feb` 的 `cmp …,0x15`）。
	crushColLeft  = 0x10 // 16
	crushColRight = 0x15 // 21

	// crushMirror 是上下對稱的軸：下半那一列 ＝ `crushMirror − 上半那一列`
	// （`0x19fd0` 的 `mov ax,0x4c` ＋ `sub ax,found`）。
	//
	// 35→41、36→40、37→39、38→38 —— 兩側同時往第 38 列夾。
	crushMirror = 0x4c // 76

	// crushScanCol 是掃描用的那一欄（`0x19f9e` 的 `+0x10`）。
	// 與 crushColLeft 同一個值，但**語意不同**：一個是「從哪一欄開始填」，
	// 一個是「看哪一欄判斷推到哪裡了」。原版剛好相同，分開命名免得被合併。
	crushScanCol = 0x10
)

// TileWriter 是「能改一格」的地圖。`world.Map` 滿足它。
//
// 壓牆要讀也要寫，而 `TileSource` 只有讀 —— 分成兩個介面是因為
// **絕大多數規則只該讀**，能寫的入口越少越好。
type TileWriter interface {
	TileSource
	SetTileAt(x, y int, t byte) error
}

// CrushResult 是牆推進一次的結果。
type CrushResult struct {
	// Row 是這一次被填成牆的上半列（下半列是 crushMirror − Row）。
	Row int
	// Crushed 為真代表隊伍現在站在牆裡 —— 全隊死亡。
	Crushed bool
}

// AdvanceCrushingWalls 讓兩側的牆各往中間推進一列（原版 `0x19f84`–`0x1a006`）。
//
// 演算法照抄：
//
//  1. 由第 35 列往下掃到第 38 列，找**第一列 `(16, y)` 還不是牆**的。
//     全都是牆的話 `found` 留 0 —— 那是原版的初值，不是「沒找到」的旗標。
//  2. 把 `(16..21, found)` 與 `(16..21, 76−found)` 六格全部寫成牆。
//  3. 隊伍腳下那一格現在是牆 → 被壓死。
//
// > 第 1 步那個 `found = 0` 的初值是原版的行為，**不要「修正」成
// > 「找不到就什麼都不做」**：走廊夾滿之後再踩一步，原版會去寫第 0 列
// > 與第 76 列（第 76 列超出 64×64，`SetTileAt` 會擋下來）。
// > 那是原版的邊界瑕疵，但它到不了 —— 夾滿之前隊伍早就死了或跑掉了。
func AdvanceCrushingWalls(m TileWriter, partyX, partyY int) CrushResult {
	if m == nil {
		return CrushResult{}
	}
	row := 0
	for y := crushScanTop; y <= crushScanBottom; y++ {
		t, err := m.TileAt(crushScanCol, y)
		if err != nil {
			continue
		}
		if t != CrushWallTile {
			row = y
			break
		}
	}
	for x := crushColLeft; x <= crushColRight; x++ {
		_ = m.SetTileAt(x, row, CrushWallTile)
		_ = m.SetTileAt(x, crushMirror-row, CrushWallTile)
	}

	here, err := m.TileAt(partyX, partyY)
	return CrushResult{Row: row, Crushed: err == nil && here == CrushWallTile}
}
