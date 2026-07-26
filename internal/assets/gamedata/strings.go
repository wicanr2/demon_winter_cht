package gamedata

import "fmt"

// dttStringCount 是 FILES.DTT 應該解析出的字串總數，已用真實檔案驗證過
// （見 docs/formats/resource-index.md 第 1.1 節）：5,829 bytes 的檔案是一串
// NUL 結尾字串，共 501 筆（122 筆為空字串、379 筆非空）。
const dttStringCount = 501

// StringPool 是 FILES.DTT 解析後的查詢介面，代表 Demon's Winter 的
// 遊戲主字串池（種族名、技能名、武器/護甲類型名、法術訊息、物件互動文字…等）。
//
// 建立方式一律透過 LoadStringPool，零值不可用。
type StringPool struct {
	strings []string
}

// LoadStringPool 解析指定路徑的 FILES.DTT，回傳可依索引查詢的字串池。
//
// FILES.DTT 沒有長度前綴或位移表頭，純粹是字串一個接一個排列、用 NUL
// （0x00）分隔；中段大量出現的空字串（122 個）是有意義的「留空欄位」，
// 不是雜訊，因此全部保留在索引序列裡（只有檔案結尾多出的那個空 token 會被丟棄）。
//
// 解析失敗（檔案讀不到）回傳 error，不 panic。
func LoadStringPool(path string) (*StringPool, error) {
	tokens, err := readNULDelimitedTokens(path)
	if err != nil {
		return nil, err
	}
	strs := make([]string, len(tokens))
	for i, tok := range tokens {
		strs[i] = string(tok)
	}
	return &StringPool{strings: strs}, nil
}

// Len 回傳字串池的總筆數（FILES.DTT 目前是 501，含空字串）。
func (p *StringPool) Len() int { return len(p.strings) }

// At 依索引（0-based）取得字串，可能回傳空字串（合法值，不是錯誤）。
func (p *StringPool) At(i int) (string, error) {
	if i < 0 || i >= len(p.strings) {
		return "", fmt.Errorf("gamedata: 字串池索引 %d 超出範圍 [0,%d)", i, len(p.strings))
	}
	return p.strings[i], nil
}

// All 回傳整個字串池的唯讀複本，順序與檔案內原始順序一致。
func (p *StringPool) All() []string {
	out := make([]string, len(p.strings))
	copy(out, p.strings)
	return out
}

// slice 是內部共用的區間切片輔助函式：越界或索引反轉時回傳 nil 而不 panic，
// 呼叫端（下面的已驗證分段 accessor）保證傳入範圍在合法檔案上一定成立，
// 這裡多一層防呆是為了不讓「檔案格式跟預期不符」以 panic 的方式外洩。
func (p *StringPool) slice(start, end int) []string {
	if start < 0 || end > len(p.strings) || start > end {
		return nil
	}
	out := make([]string, end-start)
	copy(out, p.strings[start:end])
	return out
}

// 下面這組是 docs/formats/resource-index.md 第 1.2 節「已驗證的分段」對應的
// 具名 accessor：每個區間都已逐項比對 translations/glossary.md 或 DEMON.INT
// 字串表確認內容與順序，不是憑檔名或位置猜測。索引區間定義見同一份文件。
//
// # 分段邊界的來源：原版的切表常式
//
// 這些邊界原本是**內容比對推出來的**，現在有了硬證據：`DEMON.INT 0x11d5d`
// 是把整個 FILES.DTT blob 切成遠指標陣列的常式（見 `docs/re/27`）。
// 它依序填 8 張表，每張的長度就寫在迴圈的比較常數裡：
//
//	5 → ds:0x54e8   種族名
//	86 → ds:0x4ccc  43 組（法術名, 命中訊息）
//	32 → ds:0x5538  技能名
//	30 → ds:0x4e38  **道具型別名（＝ ITEMS.DAT 的 30 筆）**
//	11 → ds:0x4c98  神祇名
//	303 → ds:0x55b8 場景／物件互動文字
//	22 → ds:0x50f2  **月份名**
//	12 → ds:(下一張) 幻術／召喚生物名
//
// 8 張表相加正好 501 —— 與 FILES.DTT 的字串總數一個不差。所以分段不再是
// 推測，是原版程式碼直接講的。這也修掉兩個原本猜錯的邊界（見 CONTEXT 推翻清單）。

// RaceNames 回傳 5 個種族名稱（索引 [0:5)）。已驗證：與
// translations/glossary.md 第 3 節「種族」5 項順序完全吻合。
func (p *StringPool) RaceNames() []string { return p.slice(0, 5) }

// NumSpellRecords 是法術表的記錄數，與 FILES.DAT 0x45e 起的 43 筆一致。
const NumSpellRecords = 43

// spellPairsStart 是法術「名稱＋訊息」成對字串的起點。
//
// 佈局：0–4 是五個種族名，接著 43 組 (名稱, 訊息)，然後 91 起是技能名。
// 5 + 43×2 = 91 —— 這條等式由 SkillNames 的起點反向印證，
// 而且名稱順序與 FILES.DAT 法術表逐筆對得上（0 = COLUMN OF FIRE、
// 1 = FLAME STRIKE、2 = FIRE STORM…，見 docs/re/15 §1）。
const spellPairsStart = 5

// SpellName 依法術索引（0–42）回傳原版英文名稱。
//
// 這個索引與 FILES.DAT 法術參數表（0x45e）的記錄編號是**同一個空間**。
func (p *StringPool) SpellName(i int) (string, error) {
	if i < 0 || i >= NumSpellRecords {
		return "", fmt.Errorf("gamedata: 法術索引 %d 超出範圍 [0,%d)", i, NumSpellRecords)
	}
	return p.At(spellPairsStart + i*2)
}

// SpellMessage 依法術索引回傳命中訊息（例如 `burnt for`）。
func (p *StringPool) SpellMessage(i int) (string, error) {
	if i < 0 || i >= NumSpellRecords {
		return "", fmt.Errorf("gamedata: 法術索引 %d 超出範圍 [0,%d)", i, NumSpellRecords)
	}
	return p.At(spellPairsStart + i*2 + 1)
}

// SkillNames 回傳 32 個技能名稱（索引 [91:123)）。已驗證：與
// translations/glossary.md 第 5 節「技能」數量與內容完全吻合，且保留遊戲
// 原始拼字 "Shamen"（glossary.md 備註明確記載此為遊戲內誤拼）。
func (p *StringPool) SkillNames() []string { return p.slice(91, 123) }

// WeaponTypeNames 回傳 8 個武器類型名稱（索引 [123:131)）。已驗證：與
// DEMON.INT 字串表中同一組武器名同序。
func (p *StringPool) WeaponTypeNames() []string { return p.slice(123, 131) }

// ArmorTypeNames 回傳 5 個護甲類型名稱（索引 [131:136)）。已驗證：與
// DEMON.INT 字串表同序。
func (p *StringPool) ArmorTypeNames() []string { return p.slice(131, 136) }

// ItemTypeNames 回傳 30 個道具型別名稱（索引 [123:153)）。
//
// **這是一整張表，長度由切表常式寫死的 30 決定**，而 ITEMS.DAT 剛好 30 筆 ——
// 所以 `ITEMS.DAT 索引 i` 的名稱就是 `123 + i`。前面兩個 accessor
// （WeaponTypeNames／ArmorTypeNames）是它的子區間，尾端的 Demon Crystal 與
// Orb/Evertime 是型別 28、29，不是另一張「神器表」（原本猜錯，已推翻）。
func (p *StringPool) ItemTypeNames() []string { return p.slice(123, 153) }

// EquipmentCategoryNames 回傳 17 個器物類別名稱（索引 [136:153)），
// 例如 crown、vial、ring、wand、staff、rod、gem、amulet、torch、lantern，
// 以及最後兩件劇情物品。是 ItemTypeNames 的尾段子區間。
func (p *StringPool) EquipmentCategoryNames() []string { return p.slice(136, 153) }

// DeityNames 回傳 11 個神祇名稱（索引 [153:164)）：Omizeh、Balmur、Gamur、
// Vemarkn、Acisc、Maldorath、Volobews、Illo、Theryni、Camear、Ancient。
//
// 用途從讀取端追出來：`DEMON.INT 0x4bea` 拿角色記錄的 `+0xf0` 減一，
// 乘 4 當索引查這張表（`ds:0x4c98`）。所以 `+0xf0` 是神祇編號、0 是「沒有」——
// 原版起始隊伍五個人都是 0。Maldorath 正是本作的魔王名，對得上。
//
// **`+0xf0` 是「長期信仰」還是「今天求到的祝福」未定案**：
// 神殿那一支（`0x1c53e`）寫入編號的同時把 `+0xeb` 設成 20，看起來像持續回合數；
// 而睡覺的夢境段（`0x3f0d`）每晚把全隊的 `+0xf0` 清成 0 —— 信仰不該過夜就沒了，
// 所以偏向「當前生效的祝福」。兩者都還沒實作。
func (p *StringPool) DeityNames() []string { return p.slice(153, 164) }

// MonthNames 回傳 22 個月份名稱（索引 [467:489)）：Ruby、Ebony、Gold、Comet、
// Spirit、Dragon、Rose、Sword、Unicorn、Metal、Lotus、Axe、Panther、Ice、
// Mandrake、Aurora、Onyx、Phoenix、Wind、Jade、Fire、Hyacinth。
//
// 讀取端在狀態列（`DEMON.INT 0x70ac`）：隊伍欄位 `+0x9d`（月）乘 4 當索引查
// `ds:0x50f2` 的遠指標表，取出名稱套進 `"Hour %d, Day %d in the Month of the %s"`。
// 所以**月是 0-based 的名稱索引**，不是序數。
//
// 這張表原本被標成「隨機物品命名詞庫」（Comet Sword 之類），**已推翻**。
func (p *StringPool) MonthNames() []string { return p.slice(467, 489) }

// IllusionSummonNames 回傳 12 個幻術／召喚生物名稱（索引 [489:501)）。
// 已驗證：與 translations/glossary.md 第 7 節「幻術／召喚生物」12 項、
// 順序、內容完全吻合。
func (p *StringPool) IllusionSummonNames() []string { return p.slice(489, 501) }
