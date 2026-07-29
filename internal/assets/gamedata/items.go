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
	// UnusedFlag 是原始記錄的第二數值欄（0/1）。IDA 9.4 對數值表基址
	// `ds:5300` 的全檔 xref 證明 DOS 執行檔沒有讀 record+2；舊名
	// WeaponSlot 是從資料分組猜的，已撤回（docs/re/110）。
	// 保留它只為忠實解析與研究其他平台，不得據此新增玩法。
	UnusedFlag bool
	// CategoryIndex 是**充能種類**（0–3）。原本標「語意未確認」，
	// 掉寶生成端（`1990:1334`）拿它當 switch 的分支值（見 `docs/re/30`）：
	//
	//	1  已用次數寫 0xFF、上限 rnd(等級×2)+1  → 過夜不充能
	//	2  上限 121 − rnd(等級+5)（戒指與火把固定 200）→ 上限 >= 100，也不充能
	//	3  上限 3 − 強度/8，rnd(5) == 1 再 +1   → 個位數，會充能
	//	0  現場擲 rnd(3) 決定用上面哪一種
	//
	// 分組（武器/護甲/冠冕/勳章 = 3；魔杖/法杖/權杖/護符/藥膏 = 1；
	// 兩件藥水瓶 = 2；其餘 = 0）於是有了解釋：3 是裝備（次數少甚至為 0，
	// 效果靠裝備被動生效）、1 與 2 是消耗品的兩種續航模式。
	//
	// 兩個「過夜不充能」的例外（`docs/re/26` §3.3 的 `+0x05 >= 100`
	// 與 `+0x06 == 0xff`）正好對應種類 2 與種類 1 —— 兩邊獨立解出來卻互相解釋。
	CategoryIndex int

	// EffectClasses 是這件道具的四個**效果類別候選**（原始欄位 f3–f6）。
	//
	// 掉寶生成時（`1990:13e3`）從這四個裡隨機挑一個、減 1 當列索引去查
	// `DEMON.INT` `DS:0x1941` 的候選表，再從該列的候選裡挑一個效果索引，
	// 寫進道具槽的 `+0x07`。詳見 `docs/re/25` §3。
	//
	// 這解釋了為什麼「同一大類內部數值完全相同」（武器全是 `0,10,8,9`、
	// 護甲全是 `12,7,7,7`）—— 它綁的是大類的效果池，不是單件道具的屬性。
	//
	// **第一個欄位是 0 還是非 0，決定另外三個是排除還是候選**：
	//
	//	f[0] != 0 → 從四個裡隨機挑一個（候選清單）
	//	f[0] == 0 → rnd(17)，撞到 f[1..3] 就重擲（排除清單）
	//
	// 這一條原本標成「還沒解釋清楚，挑到 0 減 1 會索引到表外」——
	// 那是把 `jne` 讀成 `je` 的結果。兩條路都保證非零，減一之後永遠落在
	// 0–16。已實作於 `game.LootEffectClass`，30 件真貨全跑過。
	EffectClasses [4]int
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

	fieldLabels := []string{"price", "unused_flag", "category", "f3", "f4", "f5", "f6"}

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
			UnusedFlag:    nums[1] != 0,
			CategoryIndex: nums[2],
			EffectClasses: [4]int{nums[3], nums[4], nums[5], nums[6]},
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
