package scenario

import (
	"fmt"
	"os"
	"path/filepath"
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
// **記錄布局程式碼側已確認**（`docs/re/95` §1）：列出腳下道具的常式
// `222f:2da5` 用 `si = i × 3` 掃這張表，比對隊伍的 `+0xa1`／`+0xa2`／`+0xa3`。
//
// **這個檔是存檔的一部分**（`docs/re/95` §3.2）：拿走一件時原版就地把
// 子地圖寫 0（`0x18397`）。所以：
//
//	ITEMLOCB.DAT  遊玩中會被改寫、要存回的那一份
//	ITEMLOCX.DAT  出廠母本（新遊戲從它重建）
//
// 與 `nSS.DAT` vs `ALL_SS.DAT` 是同一個模式（`docs/re/78`），
// 所以這一份放在 `scenario` 而不是 `gamedata` —— **會被寫回的東西集中在一起**。
// 道具的**內容**（50 件 × 6 條字串）是靜態的，留在 `gamedata.DungeonItem`。

const (
	// ItemLocFileSize 是兩個檔案的大小。
	ItemLocFileSize = 256
	// ItemLocRecordLen 是一筆記錄的長度：X, Y, 子地圖編號。
	ItemLocRecordLen = 3
	// ItemLocRecordCount 是筆數。**原版明寫的常數**：列出腳下道具的迴圈
	// 上界是 `cmp [bp-4], 0x32`（`0x18d64`，`docs/re/95` §1）。
	//
	// 不是 `256 ÷ 3 = 85` —— 那只是塞得下的上限。第 50 筆之後是原版
	// 沒清乾淨的 buffer 殘留（位元組反序、最高位被設起來的 `Provisions: Go`），
	// 永遠不會被讀到。
	ItemLocRecordCount = 50
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

// ItemLocLiveFile／ItemLocMasterFile 是兩份檔名。
const (
	ItemLocLiveFile   = "ITEMLOCB.DAT"
	ItemLocMasterFile = "ITEMLOCX.DAT"
)

// ItemLocTaken 是「這一筆已經被拿走」的子地圖值（原版寫 0，`0x18397`）。
const ItemLocTaken = 0

// ValidMapID 回報子地圖編號是不是一個真的地城。
//
// 1／3／5 是獨立的 `MAPn.MAP`，2／4 在 `SUM.MAP` 裡（`docs/re/03` §3.2）——
// 所以 **2 與 4 是合法的**，「檔案不存在」不是排除它們的理由。
// **0 也是合法值**，它的意思是「這一筆被拿走了」（`ItemLocTaken`）。
//
// 這支只給呼叫端做健全性檢查用；**解析本身不靠它決定切幾筆**
// （筆數是固定的 50，理由見 ParseItemLoc）。
func (l ItemLoc) ValidMapID() bool { return l.MapID <= 5 }

// ParseItemLoc 解析 `ITEMLOCB.DAT`。
//
// **固定切 50 筆**（`ItemLocRecordCount`），不看內容決定停在哪 ——
// 那個數字是原版明寫的（`0x18d64` 的 `cmp [bp-4], 0x32`）。
// 第 50 筆之後的 buffer 殘留根本不會被讀到。
//
// > 這裡原本的做法是「掃到子地圖不在 1–5 就停」。那在**唯讀**的前提下
// > 剛好也切出 50 筆，但拿走一件之後那一筆的子地圖會變成 0
// > （`docs/re/95` §3.2）—— 於是重讀存檔時**整張表在第一筆就截斷**。
// > 存檔往返的測試一跑就爆，而畫面上只會看到「地城道具全部消失」。
// > 教訓與記憶 `fresh-state-paths-hide-bugs` 同一條：
// > **判準不可以依賴「資料還沒被改寫過」。**
func ParseItemLoc(data []byte) (*ItemLocTable, error) {
	if len(data) != ItemLocFileSize {
		return nil, fmt.Errorf("ITEMLOC 檔案長度 = %d，預期 %d",
			len(data), ItemLocFileSize)
	}
	t := &ItemLocTable{
		Raw:     append([]byte(nil), data...),
		Records: make([]ItemLoc, ItemLocRecordCount),
	}
	for i := range t.Records {
		p := i * ItemLocRecordLen
		t.Records[i] = ItemLoc{X: data[p], Y: data[p+1], MapID: data[p+2]}
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
	if mapID == ItemLocTaken {
		return nil // 子地圖 0 代表「被拿走」，不是一張圖
	}
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
	if mapID == ItemLocTaken {
		return 0, false
	}
	for i, r := range t.Records {
		if r.MapID == mapID && r.X == x && r.Y == y {
			return i, true
		}
	}
	return 0, false
}

// --- 改寫與存回（`docs/re/95` §3.2）---

// Encode 把表寫回 256 bytes。
//
// **尾端原封不動**：第 50 筆之後是原版沒清乾淨的 buffer 殘留，
// 不是我們的資料。只覆寫前 50 筆那一段，其餘照抄 `Raw` ——
// 這與 `SpecialTiles.Encode` 保留檔尾傳送表是同一個理由。
func (t *ItemLocTable) Encode() []byte {
	out := append([]byte(nil), t.Raw...)
	for i, r := range t.Records {
		p := i * ItemLocRecordLen
		out[p], out[p+1], out[p+2] = r.X, r.Y, r.MapID
	}
	return out
}

// Take 把第 i 筆標成「已拿走」。
//
// **三個 byte 全部寫 0**，這是原版 `Take:` 那一支的作法
// （`0x199f4`–`0x19a10`：`si = j/2` ＝ `i×3`，然後連寫三個 0）。
//
// > `U` 的 `N` 動作**只清子地圖那一個 byte**（`0x1839b` 的
// > `mov es:[bx+si+2], 0`）。兩條路在原版就是不一樣的，查詢結果相同
// > （子地圖 0 對不上任何座標），但寫回檔案的位元組不同。
// > 接 `N` 的時候要照它自己那一支，別共用這一支。
//
// **不刪除記錄** —— 陣列是固定 50 格，索引就是道具的身分
// （與 `gamedata.DungeonItem` 對齊）。刪掉會讓後面每一件都換身分。
func (t *ItemLocTable) Take(i int) bool {
	if i < 0 || i >= len(t.Records) || t.Records[i].MapID == ItemLocTaken {
		return false
	}
	t.Records[i] = ItemLoc{}
	return true
}

// Drop 把第 i 件放到指定座標。
//
// 手冊：「丟棄地城道具，之後一定能在原地撿回」——
// 所以丟棄就是把那一筆的座標改成現在的位置、子地圖填回去。
// **用的是原本那一格記錄**，不是新增一筆：50 格是固定陣列，
// 而索引就是身分。
func (t *ItemLocTable) Drop(i int, x, y, mapID byte) bool {
	if i < 0 || i >= len(t.Records) || mapID == ItemLocTaken {
		return false
	}
	t.Records[i] = ItemLoc{X: x, Y: y, MapID: mapID}
	return true
}

// Taken 回報第 i 件在不在玩家手上（＝不在地圖上）。
func (t *ItemLocTable) Taken(i int) bool {
	return i >= 0 && i < len(t.Records) && t.Records[i].MapID == ItemLocTaken
}

// LoadItemLocTable 依三段優先序準備位置表，與 `LoadSpecialTileSet` 同一套：
//
//  1. **存檔目錄有 `ITEMLOCB.DAT`** → 讀那一份（進度）。
//  2. **全新開始** → 從資料目錄的 `ITEMLOCX.DAT`（母本）重建。
//  3. 母本缺檔 → 退回資料目錄的 `ITEMLOCB.DAT`。
//
// 第 2 步不能省：出廠的 `ITEMLOCB.DAT` 原則上會是玩過的狀態
// （現在兩檔相同只是因為那份資料還沒被玩過）。
func LoadItemLocTable(saveDir, dataDir string, fresh bool) (*ItemLocTable, error) {
	if !fresh {
		if t, err := readItemLoc(filepath.Join(saveDir, ItemLocLiveFile)); err == nil {
			return t, nil
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if t, err := readItemLoc(filepath.Join(dataDir, ItemLocMasterFile)); err == nil {
		return t, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return readItemLoc(filepath.Join(dataDir, ItemLocLiveFile))
}

func readItemLoc(path string) (*ItemLocTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := ParseItemLoc(data)
	if err != nil {
		return nil, fmt.Errorf("%s：%w", path, err)
	}
	return t, nil
}

// WriteItemLocTable 把表寫進存檔目錄（**不是**原版資料目錄）。
func WriteItemLocTable(dir string, t *ItemLocTable) error {
	if t == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("建立目錄 %s 失敗: %w", dir, err)
	}
	path := filepath.Join(dir, ItemLocLiveFile)
	if err := os.WriteFile(path, t.Encode(), 0o644); err != nil {
		return fmt.Errorf("寫入 %s 失敗: %w", path, err)
	}
	return nil
}
