package scenario

// 道具槽的已解欄位（17 bytes，見 docs/formats/game-data-tables.md §1.3）。
//
//	+0x00  道具型別 = ITEMS.DAT 索引，0xFF 空槽
//	+0x01  附帶法術 A 的索引（最高位元 0x80 是旗標）
//	+0x02  附帶法術 A 的強度
//	+0x03  附帶法術 B 的索引
//	+0x04  附帶法術 B 的強度
//	+0x05  總次數（上限）
//	+0x06  已用次數（與 +0x05 相等代表用完）
//	+0x07  效果索引 —— 與法術共用同一張效果記錄表
//	+0x08  效果強度，同時當「可不可用」旗標（0 = 不可用）
//	+0x09  條件旗標 A，值 0x15 時啟用 +0x0a
//	+0x0a  武器特效值 A（以 +10 偏移儲存）
//	+0x0b  條件旗標 B，值 0x15 時啟用 +0x0c
//	+0x0c  武器特效值 B（以 +10 偏移儲存）
//	+0x0d  驅邪成功率（只有掉落生的詛咒品才寫）
//	+0x0e  附魔加成（以 +10 偏移儲存，10 = 無附魔）
//	+0x0f  材質／品質類別，決定名稱前綴與價格倍率
//	+0x10  已鑑定旗標
//
// 17 個 byte 到這裡全部有名字了。`+0x01`–`+0x04` 與 `+0x09`–`+0x0c`
// 是讀通掉寶生成器（`docs/re/48`）之後才對上的 —— 那支常式是這些欄位
// **唯一的寫入端**，看它寫什麼進去比看讀取端猜語意可靠得多。
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
//
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
	slotType = 0x00
	// `+0x01`–`+0x04` 是**兩組「附帶法術」**（`docs/re/48` §6）：
	// 各一個位元組的法術索引（最高位元 0x80 是旗標）加一個位元組的強度。
	// 掉寶生成器只給**武器**長這個，一次最多兩個。
	//
	// 估價把兩個強度相加乘 270 當已鑑定加價（`docs/re/44` §3）——
	// 那個「語意未解的兩個 byte」就是這裡的強度。
	slotSpellA      = 0x01
	slotSpellAPower = 0x02
	slotSpellB      = 0x03
	slotSpellBPower = 0x04
	slotTotal       = 0x05
	slotUsed        = 0x06
	slotEffect      = 0x07
	slotPower       = 0x08
	slotCondA       = 0x09
	slotEffectA     = 0x0a
	slotCondB       = 0x0b
	slotEffectB     = 0x0c

	// slotExorcise 是驅邪成功率（`1000:19c8`：`rnd(100) > 它` 就失敗）。
	// 值越大越好驅。這個 byte 一度標在「語意未解」那一排。
	slotExorcise   = 0x0d
	slotEnchant    = 0x0e
	slotIdentified = 0x10

	// slotEmpty 是空槽的型別值。
	slotEmpty = 0xff

	// slotMaterialClass 是**材質／品質類別**（`+0x0f`）。它有兩個讀取端：
	//
	//   - 名稱：掉寶結算用它查一張名稱表（`docs/re/30`）
	//   - 價格：估價常式拿它查倍率表 `ds:0x18b3`
	//     = `{0, 1, 2, 5, 20, 35, 50, 60, 75}`（`docs/re/44`）
	//
	// 所以「金匕首」貴在這個 byte 上 —— 同一個型別，倍率差 75 倍。
	// 兩份原版存檔裡每一件實物都是 1（×1 = 原價），與「起始裝備都是
	// 平凡貨」一致。新造一格時照樣填 1。
	//
	// 唯一的例外是 Wopple 已清空的第 2 格留著 2 —— 原版「交出道具」
	// 只把型別寫成 0xFF，其餘 bytes 不動，那個 2 是前一件的殘值。
	slotMaterialClass        = 0x0f
	slotMaterialClassDefault = 1

	// effectCondEnabled 是「下一個位元組有效」的條件值。
	effectCondEnabled = 0x15

	// storedOffset 是特效值與附魔共用的儲存偏移：存的是實際值 +10。
	storedOffset = 10
)

// SlotEmpty 是空槽的型別值，給 game 層清空用。
const SlotEmpty = slotEmpty

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

	// MaterialClass 是材質／品質類別（`+0x0f`），決定名稱前綴與價格倍率。
	MaterialClass int

	// SpellA／SpellB 是兩組附帶法術的原始位元組（`+0x01`／`+0x03`），
	// SpellAPower／SpellBPower 是各自的強度（`+0x02`／`+0x04`）。
	// 只有武器會長這個，見 `docs/re/48` §6。
	SpellA, SpellAPower int
	SpellB, SpellBPower int

	// CondA／CondB 是兩組特效的條件旗標（`+0x09`／`+0x0b`），
	// EffectAByte／EffectBByte 是各自的**原始值**（沒扣掉 storedOffset）。
	//
	// `WeaponEffect` 是「條件旗標為 0x15 時把值算進去」的衍生結果，
	// 拆不回來；估價的第四項要的是 `EffectAByte` 這個原始 byte
	// （`docs/re/46` §4）。四個 byte 都留原值才寫得回存檔。
	CondA, EffectAByte int
	CondB, EffectBByte int

	// ExorciseResist 是驅邪成功率（`+0x0d`）。紮營選單的 Xorcise
	// 擲 `rnd(100)`，大於這個值就失敗 —— **越大越好驅**，
	// 所以嚴格說是「好驅程度」不是抗性（見 `docs/re/41`）。
	ExorciseResist int
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
	out.ExorciseResist = int(raw[slotExorcise])
	out.MaterialClass = int(raw[slotMaterialClass])
	out.SpellA = int(raw[slotSpellA])
	out.SpellAPower = int(raw[slotSpellAPower])
	out.SpellB = int(raw[slotSpellB])
	out.SpellBPower = int(raw[slotSpellBPower])
	out.CondA = int(raw[slotCondA])
	out.EffectAByte = int(raw[slotEffectA])
	out.CondB = int(raw[slotCondB])
	out.EffectBByte = int(raw[slotEffectB])
	out.Enchant = int(raw[slotEnchant]) - storedOffset
	if raw[slotCondA] == effectCondEnabled {
		out.WeaponEffect += int(raw[slotEffectA]) - storedOffset
	}
	if raw[slotCondB] == effectCondEnabled {
		out.WeaponEffect += int(raw[slotEffectB]) - storedOffset
	}
	return out
}

// encodeInto 把已解欄位寫回一格的原始 bytes。
//
// `+0x01`–`+0x04` 與 `+0x09`–`+0x0c` 原本標「語意未解、留原值」，
// 掉寶生成器讀通之後（`docs/re/48`）都有了名字，改成照欄位寫回 ——
// 讀出來是什麼就寫回什麼，逐位元組往返仍然相同。
//
// `WeaponEffect` 仍然不寫：它是兩組「條件旗標＋特效值」算出來的**衍生值**，
// 拆不回去（3 = 3+0 還是 1+2？）。要改武器特效請動 CondA／EffectAByte 那四個。
//
// 空槽只寫型別。這是照原版做的：交出道具時它也只把 `+0x00` 寫成 0xFF，
// 剩下的 bytes 就留在那裡（兩份原版存檔都看得到這種殘值）。
func (s InventorySlot) encodeInto(raw []byte) {
	if len(raw) < inventorySlotLen {
		return
	}
	raw[slotType] = s.Type
	if s.Empty() {
		return
	}
	raw[slotEffect] = byte(s.Effect)
	raw[slotPower] = byte(s.Power)
	raw[slotTotal] = byte(s.Total)
	raw[slotUsed] = byte(s.Used)
	raw[slotExorcise] = byte(s.ExorciseResist)
	raw[slotMaterialClass] = byte(s.MaterialClass)
	raw[slotSpellA] = byte(s.SpellA)
	raw[slotSpellAPower] = byte(s.SpellAPower)
	raw[slotSpellB] = byte(s.SpellB)
	raw[slotSpellBPower] = byte(s.SpellBPower)
	raw[slotCondA] = byte(s.CondA)
	raw[slotEffectA] = byte(s.EffectAByte)
	raw[slotCondB] = byte(s.CondB)
	raw[slotEffectB] = byte(s.EffectBByte)
	raw[slotEnchant] = byte(s.Enchant + storedOffset)
	if s.Identified {
		raw[slotIdentified] = 1
	} else {
		raw[slotIdentified] = 0
	}
}

// newSlotRaw 造一格全新的原始 bytes，給「這一格換了另一件東西」用。
//
// 換了東西就不能沿用舊 bytes：舊道具的次數、附魔、武器特效會整批
// 黏到新道具身上。全部歸零再填 `+0x0f`，其餘由 encodeInto 覆蓋。
func newSlotRaw() []byte {
	raw := make([]byte, inventorySlotLen)
	raw[slotMaterialClass] = slotMaterialClassDefault
	return raw
}
