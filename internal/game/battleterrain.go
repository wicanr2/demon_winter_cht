package game

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

// 戰場地形：進戰鬥時從世界地圖切一塊下來。
//
// 原版的作法（`DEMON.INT` 檔位移 0x172f4 起的函式）：
//
//  1. 把 9×9 的地形緩衝區清成 0（實際清 0x80 bytes）。
//  2. 以遭遇發生的世界座標為中心，取一個 (9−2k)×(9−2k) 的視窗，
//     貼進網格的 (k, k)。k 是視野內縮量，見 gamedata.LightInsetAt。
//  3. 每一格直接抄世界地圖的 tile：`tile = world[Y*64 + X]`（0x1737f）。
//  4. 視窗超出世界地圖邊界（X 或 Y 不在 0–63）的格子跳過，維持 0。
//
// 所以戰場不是另一份資料，就是大地圖的局部放大 —— 與手冊講的
// 「戰場是該區域的放大地圖」一致。外圈那些 0 畫出來是空白，
// 這也是「晚上看不遠」在畫面上的表現方式。
//
// 尚未實作的一個分支：0x1739a 有個條件會把 tile 0x25 換成 0x5b
// （閘門是隊伍欄位 `+0xba` > 0x7f，語意未解）。這裡照抄原 tile，
// 不猜那個條件。

// BattleTerrainSize 是戰場地形網格的邊長，與 BattleGridWidth 相同。
const BattleTerrainSize = BattleGridWidth

// BattleTerrain 是一場戰鬥的地形，索引 = y*BattleTerrainSize + x。
// 值 0 代表「空白／看不到」。
type BattleTerrain [BattleTerrainSize * BattleTerrainSize]byte

// TileAt 回傳 (x, y) 的地形值。界外回 0。
func (t *BattleTerrain) TileAt(x, y int) byte {
	if x < 0 || x >= BattleTerrainSize || y < 0 || y >= BattleTerrainSize {
		return 0
	}
	return t[y*BattleTerrainSize+x]
}

// NewBattleTerrain 以世界座標 (cx, cy) 為中心，從 m 切出戰場地形。
//
// inset 是視野內縮量 k（0 = 整個 9×9 都看得到）。超出 0–MaxLightInset
// 一律回傳 error，不默默夾住 —— 內縮量算錯會讓整個戰場變空白，
// 那種畫面看起來像「地圖沒載到」，很難查。
func NewBattleTerrain(m *world.Map, cx, cy, inset int) (*BattleTerrain, error) {
	if m == nil {
		return nil, fmt.Errorf("game: 戰場地形需要世界地圖")
	}
	if inset < 0 || inset > gamedata.MaxLightInset {
		return nil, fmt.Errorf("game: 視野內縮量 %d 超出 0–%d",
			inset, gamedata.MaxLightInset)
	}

	var t BattleTerrain
	half := BattleTerrainSize / 2

	for gy := inset; gy < BattleTerrainSize-inset; gy++ {
		for gx := inset; gx < BattleTerrainSize-inset; gx++ {
			wx, wy := cx-half+gx, cy-half+gy
			if wx < 0 || wx >= world.MapWidth || wy < 0 || wy >= world.MapHeight {
				continue // 界外維持空白
			}
			v, err := m.TileAt(wx, wy)
			if err != nil {
				continue
			}
			t[gy*BattleTerrainSize+gx] = v
		}
	}
	return &t, nil
}

// HiddenCells 回傳戰場上哪些格子被地形擋住看不到。
//
// 委給 gamedata 的遮蔽表（FILES.DAT 0x0A8），見 gamedata/sight.go。
func (t *BattleTerrain) HiddenCells(s *gamedata.SightShadow) ([]bool, error) {
	if s == nil {
		return nil, fmt.Errorf("game: 需要視線遮蔽表")
	}
	return s.HiddenCells(t[:])
}

// Visible 回傳套用視線遮蔽之後、實際畫得出來的地形。
//
// 被遮住的格子寫成 0 —— 這正是原版的作法（0x1306d：最高位是 1 的格子
// 一律寫成 0 再交給算繪），所以是「看不到」，不是「不能走」。
func (t *BattleTerrain) Visible(s *gamedata.SightShadow) (*BattleTerrain, error) {
	hidden, err := t.HiddenCells(s)
	if err != nil {
		return nil, err
	}
	out := *t
	for i, h := range hidden {
		if h {
			out[i] = 0
		}
	}
	return &out, nil
}
