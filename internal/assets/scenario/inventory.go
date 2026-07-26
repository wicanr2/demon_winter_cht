package scenario

// 道具槽的已解欄位（17 bytes，見 docs/formats/game-data-tables.md §1.3）。
//
//	+0x00  道具型別 = ITEMS.DAT 索引，0xFF 空槽
//	+0x05  總次數（上限）
//	+0x06  已用次數（與 +0x05 相等代表用完）
//	+0x07  效果索引 —— 與法術共用同一張效果記錄表
//	+0x08  效果強度，同時當「可不可用」旗標（0 = 不可用）
//	+0x09  條件旗標 A，值 0x15 時啟用 +0x0a
//	+0x0a  武器特效值 A（以 +10 偏移儲存）
//	+0x0b  條件旗標 B，值 0x15 時啟用 +0x0c
//	+0x0c  武器特效值 C（以 +10 偏移儲存）
//	+0x0e  附魔加成（以 +10 偏移儲存，10 = 無附魔）
//	+0x10  已鑑定旗標
//
// `+0x05`–`+0x08` 是本輪新解的，來源是「使用道具」那一支
// （`FUN_17c5_18ab`，Ghidra `17c5:18ab`）：
//
//	17c5:1976  CMP byte ES:[BX+0x8],0x0 / JZ 跳過   ; 強度 0 → 這格不可用
//	17c5:197d  AL = ES:[BX+0x5]
//	17c5:1981  CMP AL,ES:[BX+0x6] / JZ 跳過         ; 次數用完 → 這格不可用
//
// **哪一邊是上限、哪一邊是計數，是從睡眠常式反推的**：`2aed:0471` 在睡覺時
// 把 `+0x06` 清成 0（限 `+0x05 < 100` 且 `+0x06 != 0xff` 的道具）。
// 清「已用次數」＝過夜充能，說得通；清「上限」則會讓道具永久失效。
// 本專案一度把兩者標反 —— 只看使用端那道 `CMP` 分不出來，要找到寫入端才行。
//	17c5:19dd  AL = ES:[BX+0x7]  → 效果索引
//	17c5:19e6  AL = ES:[BX+0x8]  → 效果強度
//	17c5:19ef  PUSH 效果索引 / CALLF 0x1000:114f    ; 載入 5-word 效果記錄
//
// **`FUN_1000_114f` 就是法術用的那個載入函式**（`docs/re/09` §4.1）：
// 它拿索引去 `[0x4e28]` 讀 `索引×10` 的 5 個 word，寫進 `0x4e2c`–`0x4e34`。
// 一張表、一個載入函式、一個索引空間 —— 所以**道具的效果索引與法術 id
// 是同一個命名空間**，`+0x07` 是幾就等於施放第幾號法術的效果。
//
// 原版起始存檔裡每一件都是 `+0x07 = +0x08 = 0` 的平凡武具，
// 所以那些道具在原版的 Use 選單裡本來就選不到。
const (
	slotType       = 0x00
	slotTotal      = 0x05
	slotUsed       = 0x06
	slotEffect     = 0x07
	slotPower      = 0x08
	slotCondA      = 0x09
	slotEffectA    = 0x0a
	slotCondB      = 0x0b
	slotEffectB    = 0x0c
	slotEnchant    = 0x0e
	slotIdentified = 0x10

	// slotEmpty 是空槽的型別值。
	slotEmpty = 0xff

	// effectCondEnabled 是「下一個位元組有效」的條件值。
	effectCondEnabled = 0x15

	// storedOffset 是特效值與附魔共用的儲存偏移：存的是實際值 +10。
	storedOffset = 10
)

// InventorySlot 是一格裝備／道具。
type InventorySlot struct {
	// Type 是 ITEMS.DAT 的記錄索引。空槽是 0xFF。
	Type byte
	// Enchant 是附魔加成（已扣掉 +10 的儲存偏移）。
	Enchant int
	// WeaponEffect 是武器特效值（已扣掉 +10）。兩個條件旗標都沒啟用時為 0。
	WeaponEffect int
	// Identified 是已鑑定旗標。
	Identified bool

	// Effect 是效果索引，與法術 id 同一個命名空間（見上）。
	Effect int
	// Power 是效果強度，等同施法時「投入多少法力」那個參數。
	// **0 代表這件道具沒有可用效果**。
	Power int
	// Total 是使用次數上限，Used 是已用次數。兩者相等代表用完。
	// **過夜會把 Used 歸零**（見 game.Rest）。
	Total, Used int
}

// Usable 回報這件道具在戰鬥中選不選得到。
//
// 原版的兩道條件（`17c5:1976`／`17c5:1981`）：強度為 0、或次數用完，
// 這一格就不會出現在 Use 選單裡。裝備類另有「必須是已裝備的那一件」
// 的限制，那條在 game.UsableItems。
func (s InventorySlot) Usable() bool {
	return !s.Empty() && s.Power != 0 && s.Used != s.Total
}

// Empty 回報這格是不是空的。
func (s InventorySlot) Empty() bool { return s.Type == slotEmpty }

// parseInventorySlot 解出一格的已解欄位。
//
// **特效值有兩組（A/B），各自有一個條件旗標。** 原版是
// `if (slot[+0x09] == 0x15) 讀 slot[+0x0a]`，B 組同理。
// 兩組都啟用時相加 —— 一件武器可以帶兩個特效。
func parseInventorySlot(raw []byte) InventorySlot {
	if len(raw) < inventorySlotLen {
		return InventorySlot{Type: slotEmpty}
	}
	out := InventorySlot{
		Type:       raw[slotType],
		Identified: raw[slotIdentified] != 0,
	}
	// 空槽的其餘欄位沒有意義，不要解讀成「附魔 −10」。
	if out.Empty() {
		return out
	}
	out.Effect = int(raw[slotEffect])
	out.Power = int(raw[slotPower])
	out.Total = int(raw[slotTotal])
	out.Used = int(raw[slotUsed])
	out.Enchant = int(raw[slotEnchant]) - storedOffset
	if raw[slotCondA] == effectCondEnabled {
		out.WeaponEffect += int(raw[slotEffectA]) - storedOffset
	}
	if raw[slotCondB] == effectCondEnabled {
		out.WeaponEffect += int(raw[slotEffectB]) - storedOffset
	}
	return out
}
