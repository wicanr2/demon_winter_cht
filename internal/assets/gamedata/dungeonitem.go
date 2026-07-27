package gamedata

import "fmt"

// 地城道具的資料層（`docs/re/95`）
//
// 手冊「物品」一節說的地城道具（鐵砧、花朵、金鑰匙…）由兩份資料合成：
//
//   - **在哪** —— `ITEMLOCB.DAT`，50 筆 `(X, Y, 子地圖)`（`docs/re/94`）
//   - **是什麼** —— `FILES.DTT` 的第 164–463 條，**每件連續 6 條**
//
// 「6 條一件」不是數出來的，是原版明寫的：列出腳下道具的那支常式
// （`222f:2da5` ＝ `0x18c95`）每處理一件就 `add [bp-6], 6`，
// 而位置表的迴圈上限是 `cmp [bp-4], 0x32` ＝ **50**。
// 50 × 6 ＝ 300，正好用完 `docs/re/27` 第 6 張表宣告的 303 條裡的前 300 條。

const (
	// DungeonItemFirstString 是第一件道具的第一條字串在 `FILES.DTT` 的索引。
	// 與 `docs/re/27` §2 第 6 張表的起點相同。
	DungeonItemFirstString = 164
	// DungeonItemFields 是每件佔幾條（原版 `0x18cd6` 的 `add …, 6`）。
	DungeonItemFields = 6
	// DungeonItemCount 是件數（原版 `0x18d64` 的 `cmp …, 0x32`）。
	// **與 `ITEMLOCB.DAT` 的 50 筆是同一個數字**，兩邊靠索引對齊。
	DungeonItemCount = 50
	// DungeonItemsPerTile 是同一格最多列幾件（原版 `0x18d6a` 的 `cmp …, 9`）。
	//
	// ⚠ 手冊寫「每個位置最多只能放 10 件物品」，執行檔的選單上限是 9。
	// 沒有複核過，兩個數字都留著。
	DungeonItemsPerTile = 9
)

// DungeonItem 是一件地城道具的六個欄位。
//
// **拿到手之後它住在角色的道具槽裡**（`docs/re/95` §3.1）：17-byte 的槽
// 對地城道具是另一種解讀 —— `[0]` 是型別 `0xfe`／`0xff`、`[1..16]` 是名字。
// 與一般道具（`docs/re/25` 的效果／強度欄）是同一塊記憶體的兩種用法。
// 手冊說地城道具在清單裡前面加 `/`，那個記號的來源就是這兩個型別值。
type DungeonItem struct {
	// Name 是道具名（`+0`）。50 件全部非空。
	Name string
	// Immovable 非空代表**拿不走**（`+1`）。
	// 內容要嘛是一句理由（`It is too heavy`），要嘛只有 `*`（拿不走但沒話說）。
	// 25 件有值 —— 剛好一半是家具類。
	Immovable string
	// Look 是檢視敘述（`+2`）。48 件有值。
	Look string
	// MoveResult 是**推開之後**發生什麼（`+3`）。
	// `*` 代表推得動但沒事；數字串疑似 `XXYY…`（在那個座標開一條路），
	// 例如 `Old bookcase` 的 `461100` 對上它的檢視敘述
	// 「It has been moved to reveal a passage」。**判讀，未讀消費端。**
	MoveResult string
	// UseWith 是**用什麼才有效**，內容是另一件道具的名字（`+4`）。
	// 18 件有值，而且每一個都能在這 50 件裡找到同名的一件。
	//
	// **鑑物讀的就是這一欄**（`docs/re/93` §2 讀 `[bx+0x55c8]` ＝ `+4`）——
	// 手冊「鑑物：可得知物品未來可能的用途」對上了。
	UseWith string
	// UseResult 是用對之後發生什麼（`+5`），**首字元是動作碼**。
	// 見 DungeonItemAction。
	UseResult string
}

// DungeonItemAction 是 `UseResult` 的首字元動作碼。
//
// **已確認**：分派表在 `0x186fd`（`docs/re/95` §3）——
// `cmp ax,0x4e/0x50/0x53/0x54` 四格，`'D'` 在前面用 `cmp al,0x44` 先擋掉。
// 內容從第二個字元起算（原版 `0x18285` 的 `inc`）。
type DungeonItemAction byte

const (
	// ActionNone 代表 `UseResult` 是空的（這件道具用不出東西）。
	ActionNone DungeonItemAction = 0
	// ActionDescribe `D`：印一段敘述。多半是主線提示
	// （`Man in trance` 說出 Qoorik 的入口、`Papyrus` 寫出 Asaht）。
	ActionDescribe DungeonItemAction = 'D'
	// ActionBecome `N`：這一格的道具**變成另一件**，內容是新道具的名字
	// （`Man in glass` → `Man in trance`、`Red dust` → `Bag/red dust`）。
	ActionBecome DungeonItemAction = 'N'
	// ActionTeleport `T`：傳送。參數是 **`XXYYMM`，逐 2 位數 `atoi`**
	// （`0x183f8`–`0x18441`：分別寫進隊伍的 `+0xa1`／`+0xa2`／`+0xa3`）。
	// `T564603` ＝ (56,46) 子地圖 3。
	ActionTeleport DungeonItemAction = 'T'
	// ActionPassage `P`：同樣三組 2 位數 `atoi`（`0x1849d`），但**不寫隊伍座標**。
	// 目的地未讀 —— 參數形狀與 MoveResult 的數字相同，疑似改地圖 tile。
	ActionPassage DungeonItemAction = 'P'
	// ActionStory `S`：參數只有一位數（`S1`／`S2`）。處理在 `0x185b2`，**未讀**。
	// `S2` 掛在 `Circle light` 上 —— 那正是主線的光之環。
	ActionStory DungeonItemAction = 'S'
)

// Action 取出動作碼。
func (it DungeonItem) Action() DungeonItemAction {
	if it.UseResult == "" {
		return ActionNone
	}
	return DungeonItemAction(it.UseResult[0])
}

// ActionParam 取出動作碼後面的參數（可能是文字，也可能是數字串）。
func (it DungeonItem) ActionParam() string {
	if it.UseResult == "" {
		return ""
	}
	return it.UseResult[1:]
}

// CanTake 回報這件道具帶不帶得走。
func (it DungeonItem) CanTake() bool { return it.Immovable == "" }

// DungeonItems 是全部 50 件，索引與 `ITEMLOCB.DAT` 的記錄索引對齊。
type DungeonItems []DungeonItem

// LoadDungeonItems 從已載入的字串池切出 50 件。
func LoadDungeonItems(p *StringPool) (DungeonItems, error) {
	need := DungeonItemFirstString + DungeonItemCount*DungeonItemFields
	if p.Len() < need {
		return nil, fmt.Errorf("FILES.DTT 只有 %d 條，切不出 %d 件地城道具（需要 %d 條）",
			p.Len(), DungeonItemCount, need)
	}
	out := make(DungeonItems, DungeonItemCount)
	for i := range out {
		base := DungeonItemFirstString + i*DungeonItemFields
		f := [DungeonItemFields]string{}
		for k := range f {
			s, err := p.At(base + k)
			if err != nil {
				return nil, fmt.Errorf("第 %d 件的第 %d 條：%w", i, k, err)
			}
			f[k] = s
		}
		out[i] = DungeonItem{
			Name: f[0], Immovable: f[1], Look: f[2],
			MoveResult: f[3], UseWith: f[4], UseResult: f[5],
		}
	}
	return out, nil
}

// ByName 依名稱找一件，回傳索引。`UseWith` 存的是名字，要靠這個換回索引。
func (d DungeonItems) ByName(name string) (int, bool) {
	for i, it := range d {
		if it.Name == name {
			return i, true
		}
	}
	return 0, false
}
