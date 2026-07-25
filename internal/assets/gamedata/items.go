package gamedata

import "fmt"

// itemTokensPerRecord 是每個道具在 ITEMS.DAT 裡佔用的 NUL 分隔 token 數：
// 1 個名字字串 + 7 個數字欄位。724 bytes 的原始檔案切成 240 個 token，
// 240 ÷ 8 = 30 整除無餘數，對應 30 個道具，已用真實檔案驗證過
// （見 docs/formats/game-data-tables.md 第 3 節）。
const itemTokensPerRecord = 8

// ItemKind 是道具的武器／護甲／其他分類。
//
// 這個分類**不是**檔案裡的原始欄位，是依名稱是否落在手冊已知的 8 種武器、
// 5 種護甲名單裡（見 docs/formats/game-data-tables.md 第 3.2 節）推得，
// 邏輯與 tools/parse_items.py 的 WEAPON_NAMES／ARMOR_NAMES 判斷一致。
// 這組名稱／順序本身就是「已驗證」的最強錨點：前 13 筆道具的名稱與順序
// 跟 translations/glossary.md 第 8 節「武器與護甲」原版遊戲內順序逐一比對完全一致。
type ItemKind int

const (
	// ItemKindMisc 是特殊／任務道具（冠冕、藥水瓶、戒指、法杖、寶石等 17 種）。
	ItemKindMisc ItemKind = iota
	// ItemKindWeapon 是 ITEMS.DAT 索引 0–7 的 8 種武器
	// （匕首、小斧、短劍、釘頭鎚、晨星鎚、闊劍、戰斧、雙手劍）。
	ItemKindWeapon
	// ItemKindArmor 是 ITEMS.DAT 索引 8–12 的 5 種護甲
	// （布甲、皮甲、鎖子甲、鱗甲、板甲）。
	ItemKindArmor
)

// String 回傳 ItemKind 的可讀名稱，方便測試輸出與除錯訊息。
func (k ItemKind) String() string {
	switch k {
	case ItemKindWeapon:
		return "weapon"
	case ItemKindArmor:
		return "armor"
	default:
		return "misc"
	}
}

// weaponNames 是手冊已知的 8 種武器名稱，順序即 ITEMS.DAT 索引 0–7。
// 對照 translations/glossary.md 第 8 節，逐一比對完全一致。
var weaponNames = map[string]bool{
	"dagger": true, "small axe": true, "short sword": true, "mace": true,
	"morning star": true, "broad sword": true, "battle axe": true, "2-hand sword": true,
}

// armorNames 是手冊已知的 5 種護甲名稱，順序即 ITEMS.DAT 索引 8–12。
var armorNames = map[string]bool{
	"cloth": true, "leather": true, "chain": true, "scale": true, "plate": true,
}

// Item 是 ITEMS.DAT 一筆道具記錄解析後的乾淨表示。
//
// 驗證狀態詳見 docs/formats/game-data-tables.md 第 3.3 節。
type Item struct {
	// Name 是道具名稱（例如 "dagger"、"Orb/Evertime"）。
	Name string
	// Kind 是依名稱推得的武器／護甲／其他分類，見 ItemKind 說明。
	Kind ItemKind

	// Price 售價（金幣）。已驗證：8 把武器售價隨強度遞增（匕首最便宜、
	// 雙手劍最貴），5 件護甲售價嚴格隨防護力遞增，量級與攻略裝備售價方向一致。
	Price int
	// WeaponSlot 標記是否佔用「武器手」欄位。語意未確認：8 把武器與飾品類
	// （戒指/魔杖/法杖/權杖/護身符/勳章/雕像/護符/藥膏）此欄位為真，5 件護甲
	// 與純道具（寶石/火把/提燈/惡魔水晶/恆世寶珠）為假，跟 DEMON.INT 字串
	// 「Weapon: / Armor: 」兩種提示的介面設計吻合，但邊界不完全乾淨（例如
	// vial 藥水瓶為假、salve 藥膏為真，行為不一致，無法完全排除是別的意思）。
	WeaponSlot bool
	// CategoryIndex 是原始檔案裡的道具分類索引欄位（0–3）。語意未確認：
	// 武器/護甲/crown/medallion 皆為 3；wand/staff/rod/talisman/salve 皆為 1；
	// 兩筆 vial 皆為 2；其餘（ring/gem/amulet/figurine/torch/lantern/
	// Demon Crystal/Orb-Evertime）皆為 0。四種取值分組有規律但無法對應到
	// 已知的字串表，只能算假設。
	CategoryIndex int

	// Unknown1 是原始欄位第 4 個數字（Python 參考實作稱 f3）。語意未確認：
	// 同一大類（8 把武器／5 件護甲）內部這個欄位數值完全相同（武器皆為 0，
	// 護甲皆為 12），代表不是「每種武器各自不同的傷害骰/力量需求」——那些
	// 數值查手冊是逐項不同的，跟這裡「全部武器共用同一組數字」矛盾。推測是
	// 與武器/護甲這個大類本身綁定的共用參數（可能是圖示座標或某種判定表索引）。
	Unknown1 int
	// Unknown2 是原始欄位第 5 個數字（f4）。語意未確認，同大類內數值相同
	// （武器皆為 10，護甲皆為 7）。
	Unknown2 int
	// Unknown3 是原始欄位第 6 個數字（f5）。語意未確認，同大類內數值相同
	// （武器皆為 8，護甲皆為 7）。
	Unknown3 int
	// Unknown4 是原始欄位第 7 個數字（f6）。語意未確認，同大類內數值相同
	// （武器皆為 9，護甲皆為 7）。
	Unknown4 int
}

// ItemTable 是 ITEMS.DAT 解析後的查詢介面。
//
// 建立方式一律透過 LoadItemTable，零值不可用。
type ItemTable struct {
	items  []Item
	byName map[string]int
}

// LoadItemTable 解析指定路徑的 ITEMS.DAT，回傳可查詢的道具表。
//
// 解析失敗（檔案讀不到、token 數對不上 8 的倍數、數字欄位不是合法整數）
// 一律回傳 error，不 panic。
func LoadItemTable(path string) (*ItemTable, error) {
	tokens, err := readNULDelimitedTokens(path)
	if err != nil {
		return nil, err
	}
	if len(tokens)%itemTokensPerRecord != 0 {
		return nil, fmt.Errorf(
			"gamedata: %s 的 token 數 %d 不是 %d 的倍數，ITEMS.DAT 格式假設可能不適用於這個檔案",
			path, len(tokens), itemTokensPerRecord,
		)
	}

	n := len(tokens) / itemTokensPerRecord
	items := make([]Item, 0, n)
	byName := make(map[string]int, n)

	fieldLabels := []string{"price", "weapon_slot", "category", "f3", "f4", "f5", "f6"}

	for i := 0; i < n; i++ {
		chunk := tokens[i*itemTokensPerRecord : (i+1)*itemTokensPerRecord]
		name := string(chunk[0])

		nums := make([]int, 0, itemTokensPerRecord-1)
		for j, label := range fieldLabels {
			v, err := parseASCIIInt(fmt.Sprintf("%s(item #%d %s)", label, i, name), chunk[1+j])
			if err != nil {
				return nil, err
			}
			nums = append(nums, v)
		}

		kind := ItemKindMisc
		switch {
		case weaponNames[name]:
			kind = ItemKindWeapon
		case armorNames[name]:
			kind = ItemKindArmor
		}

		it := Item{
			Name:          name,
			Kind:          kind,
			Price:         nums[0],
			WeaponSlot:    nums[1] != 0,
			CategoryIndex: nums[2],
			Unknown1:      nums[3],
			Unknown2:      nums[4],
			Unknown3:      nums[5],
			Unknown4:      nums[6],
		}
		items = append(items, it)
		// 名稱重複（例如 "vial" 出現兩次）時只保留第一筆的索引，
		// 依名稱查找本來就只適用於唯一名稱；重複項目請用 All()／ByIndex()。
		if _, dup := byName[name]; !dup {
			byName[name] = i
		}
	}

	return &ItemTable{items: items, byName: byName}, nil
}

// Len 回傳道具總數（ITEMS.DAT 目前是 30）。
func (t *ItemTable) Len() int { return len(t.items) }

// All 回傳全部道具的唯讀複本，順序與檔案內原始順序一致。
func (t *ItemTable) All() []Item {
	out := make([]Item, len(t.items))
	copy(out, t.items)
	return out
}

// ByIndex 依索引（0-based，對應檔案內原始順序）取得道具。
func (t *ItemTable) ByIndex(i int) (Item, error) {
	if i < 0 || i >= len(t.items) {
		return Item{}, fmt.Errorf("gamedata: 道具索引 %d 超出範圍 [0,%d)", i, len(t.items))
	}
	return t.items[i], nil
}

// ByName 依名稱取得道具。名稱重複時（例如 "vial" 有兩筆）回傳第一筆，
// 需要全部重複項目請改用 All() 自行過濾。
func (t *ItemTable) ByName(name string) (Item, bool) {
	i, ok := t.byName[name]
	if !ok {
		return Item{}, false
	}
	return t.items[i], true
}
