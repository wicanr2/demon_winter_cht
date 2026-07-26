package scenario

// 船隻陣列（trailer `+0x22`，10 格 × 6 bytes）。
//
// `docs/re/19` §4.3 只寫到「全域 struct 裡 10 格 × 6 bytes、船體值在記錄 +0x25」，
// 其餘欄位標成未解，碼頭因此一直只能報價。這一輪把讀取端與**寫入端**都讀完了，
// 六個欄位全部定案（見 `docs/re/28`）：
//
//	+0  X 座標
//	+1  Y 座標
//	+2  子地圖編號
//	+3  船體值。**0 代表這一格是空的**，滿值 75
//	+4  原版放船時一律寫 2，語意未解
//	+5  從沒被寫過，兩份原版存檔都是 0
//
// 判定「這一格有沒有船」用的是**船體值而不是座標**（`2aed:13cc`
// `cmpb $0x0, es:0x25(bx,si)`）—— 座標 (0,0) 是合法位置，拿它當空槽標記會誤判。
const (
	shipArrayOffset = 0x22
	shipRecordLen   = 6
	shipCount       = 10

	shipOffX      = 0
	shipOffY      = 1
	shipOffMapID  = 2
	shipOffHull   = 3
	shipOffUnk4   = 4
	shipOffUnk5   = 5
	shipUnk4Value = 2
)

// ShipMaxHull 是船體滿值。放新船時寫 0x4b，修船也修到這個值。
const ShipMaxHull = 75

// Ship 是一艘船。
type Ship struct {
	X, Y  byte
	MapID byte
	// Hull 是船體值，0 代表這一格沒有船。
	Hull byte
	// Unknown4 原版放船時一律寫 2；Unknown5 從沒被寫過。
	// 兩個都保留原值，不猜語意。
	Unknown4, Unknown5 byte
}

// Exists 回報這一格有沒有船。
func (s Ship) Exists() bool { return s.Hull > 0 }

// NewShip 造一艘停在指定位置的新船，船體滿值。
//
// `Unknown4` 照原版填 2 —— 那個 byte 兩艘原版船也都是 2，
// 不填的話新船與原版船在那個欄位不一致，而沒人知道誰在讀它。
func NewShip(x, y, mapID byte) Ship {
	return Ship{X: x, Y: y, MapID: mapID, Hull: ShipMaxHull, Unknown4: shipUnk4Value}
}

func parseShips(trailer []byte) [shipCount]Ship {
	var out [shipCount]Ship
	for i := range out {
		b := trailer[shipArrayOffset+i*shipRecordLen:]
		out[i] = Ship{
			X:        b[shipOffX],
			Y:        b[shipOffY],
			MapID:    b[shipOffMapID],
			Hull:     b[shipOffHull],
			Unknown4: b[shipOffUnk4],
			Unknown5: b[shipOffUnk5],
		}
	}
	return out
}

func encodeShips(trailer []byte, ships [shipCount]Ship) {
	for i, s := range ships {
		b := trailer[shipArrayOffset+i*shipRecordLen:]
		b[shipOffX] = s.X
		b[shipOffY] = s.Y
		b[shipOffMapID] = s.MapID
		b[shipOffHull] = s.Hull
		b[shipOffUnk4] = s.Unknown4
		b[shipOffUnk5] = s.Unknown5
	}
}
