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

// RaceNames 回傳 5 個種族名稱（索引 [0:5)）。已驗證：與
// translations/glossary.md 第 3 節「種族」5 項順序完全吻合。
func (p *StringPool) RaceNames() []string { return p.slice(0, 5) }

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

// EquipmentCategoryNames 回傳 15 個裝備／飾品類別名稱（索引 [136:151)），
// 例如 crown、vial、ring、wand、staff、rod、gem、amulet 等。內容與遊戲道具
// 類型自洽，但沒有像其他分段那樣的外部交叉比對來源，驗證強度較弱。
func (p *StringPool) EquipmentCategoryNames() []string { return p.slice(136, 151) }

// ArtifactNames 回傳 13 個特殊神器／劇情物品名稱（索引 [151:164)）。已驗證：
// 其中 "Orb/Evertime" 對應 translations/glossary.md 第 1 節劇情物品
// 「Orb of Evertime 恆世寶珠」。
func (p *StringPool) ArtifactNames() []string { return p.slice(151, 164) }

// MaterialAdjectives 回傳 22 個材質／概念形容詞（索引 [467:489)），例如
// Ruby、Ebony、Gold、Dragon、Phoenix 等。語意未確認：形態上很像老 RPG
// 常見的「隨機物品命名詞庫」（如 "Comet Sword"、"Hyacinth Ring"），但未能
// 在其他資料檔中找到直接引用此表的證據。
func (p *StringPool) MaterialAdjectives() []string { return p.slice(467, 489) }

// IllusionSummonNames 回傳 12 個幻術／召喚生物名稱（索引 [489:501)）。
// 已驗證：與 translations/glossary.md 第 7 節「幻術／召喚生物」12 項、
// 順序、內容完全吻合。
func (p *StringPool) IllusionSummonNames() []string { return p.slice(489, 501) }
