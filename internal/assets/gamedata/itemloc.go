package gamedata

import (
	"fmt"
	"os"
)

// `ITEMLOCB.DAT`／`ITEMLOCX.DAT` — 地城道具的擺放位置
//
// 手冊「物品」一節說的地城道具（鐵砧、花朵、金鑰匙…）就是靠這張表擺進地圖的。
// 格式在 `docs/formats/town-and-map.md` §4 早就解過（**這一份不是新發現，
// 是把已有的判讀接進引擎**），這裡補上程式碼這一側的兩個錨點：
//
//   - **誰載入**：通用資源載入迴圈（`0x11fc2`）用 `ds:0x14c9` 那張
//     六筆檔名遠指標表（`party.dat` / `demon.shp` / **`itemlocb.dat`** /
//     `files.dtt` / `files.dat` / `winter.shp`），索引 2。
//   - **載到哪**：資源 arena 的 256-byte 槽（`docs/re/22` §1 的索引 2）——
//     檔案剛好 256 bytes，兩邊對得上。
//
// ⚠ **記錄布局本身仍是「檢視資料推出來的」**，還沒追到逐欄消費它的程式碼。
// 兩份檔案逐 byte 相同（同一個 md5），推測是主檔／備份。

const (
	// ItemLocFileSize 是兩個檔案的大小。
	ItemLocFileSize = 256
	// ItemLocRecordLen 是一筆記錄的長度：X, Y, 子地圖編號。
	ItemLocRecordLen = 3
	// ItemLocRecordCount 是塞得下的筆數（256 ÷ 3 = 85，尾端剩 1 byte）。
	ItemLocRecordCount = ItemLocFileSize / ItemLocRecordLen
)

// ItemLoc 是一筆地城道具位置。
type ItemLoc struct {
	X, Y  byte
	MapID byte
}

// Empty 回報這是不是「這一格沒放東西」的空記錄。
//
// `(0,0)` 在真實資料裡出現三次，夾在有效記錄中間 ——
// 不是結尾標記，是**佔位**（`docs/formats/town-and-map.md` §4.2）。
func (l ItemLoc) Empty() bool { return l.X == 0 && l.Y == 0 }

// ItemLocTable 是解析結果。
type ItemLocTable struct {
	// Records 是**有效**的記錄（含空位佔位），依檔案順序。
	Records []ItemLoc
	// Raw 保留整份 256 bytes —— 尾端那段不是記錄（見 Parse 的說明）。
	Raw []byte
}

// itemLocValidMap 回報子地圖編號是不是一個真的地城。
//
// 1／3／5 是獨立的 `MAPn.MAP`，2／4 在 `SUM.MAP` 裡（`docs/re/03` §3.2）——
// 所以 **2 與 4 是合法的**，「檔案不存在」不是排除它們的理由。
func itemLocValidMap(id byte) bool { return id >= 1 && id <= 5 }

// ParseItemLoc 解析 `ITEMLOCB.DAT`。
//
// **尾端不是記錄。** 真實資料在第 50 筆之後接的是一段沒清乾淨的
// buffer 殘留，再往後全是 `0xff`。判準是子地圖編號：
// 一碰到不在 1–5 的值就停 —— 這與「掃到 `0xff` 才停」不同，
// 因為殘留段的第一個 byte 不是 `0xff`（實際是 `0xa0`）。
//
// > 那段殘留其實是**位元組反序、且最高位被設起來**的一句文字
// > （`ef c7 a0 ba …` 逐 byte `& 0x7f` 再反著讀 ＝ `Provisions: Go`）。
// > 認出它只是為了確定「那不是資料」，不必解釋它為什麼在那裡。
func ParseItemLoc(data []byte) (*ItemLocTable, error) {
	if len(data) != ItemLocFileSize {
		return nil, fmt.Errorf("ITEMLOC 檔案長度 = %d，預期 %d",
			len(data), ItemLocFileSize)
	}
	t := &ItemLocTable{Raw: append([]byte(nil), data...)}
	for i := 0; i < ItemLocRecordCount; i++ {
		p := i * ItemLocRecordLen
		rec := ItemLoc{X: data[p], Y: data[p+1], MapID: data[p+2]}
		if !itemLocValidMap(rec.MapID) {
			break
		}
		// 座標超出 64×64 也代表已經走進殘留段。
		if rec.X >= 64 || rec.Y >= 64 {
			break
		}
		t.Records = append(t.Records, rec)
	}
	return t, nil
}

// LoadItemLoc 從檔案讀。
func LoadItemLoc(path string) (*ItemLocTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取 %s：%w", path, err)
	}
	return ParseItemLoc(data)
}

// OnMap 回傳某張子地圖上的所有位置。
func (t *ItemLocTable) OnMap(mapID byte) []ItemLoc {
	var out []ItemLoc
	for _, r := range t.Records {
		if r.MapID == mapID {
			out = append(out, r)
		}
	}
	return out
}

// At 回報某張子地圖的某一格有沒有道具，並給出它在表裡的索引。
//
// 索引可能就是道具的身分（`ds:0x55c8` 那張敘述指標表用的也是一個索引），
// **但那條還沒接上** —— 見 `docs/re/94` §3。
func (t *ItemLocTable) At(mapID, x, y byte) (int, bool) {
	for i, r := range t.Records {
		if r.MapID == mapID && r.X == x && r.Y == y {
			return i, true
		}
	}
	return 0, false
}
