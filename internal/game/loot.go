package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 掉寶生成（`DEMON.INT 1990:1107`–`1990:1596`，見 `docs/re/30`／`docs/re/48`）。
//
// 這是「為什麼店裡買的東西都沒有效果」的答案：**效果是掉寶時長出來的**，
// `ITEMS.DAT` 只規定每一類道具能長出哪些效果。
//
// 生成器依序決定七件事，每一段都可能提前結束：
//
//	1. 型別：擲 0–25，太貴的重擲（價格上限隨等級指數成長）
//	2. 材質類別：決定名稱前綴與價格倍率
//	3. 詛咒：掉落與商隊走**完全不同**的兩條路
//	4. 附魔：只有武器與護甲有，護甲減半
//	5. 效果與次數：兩道機率門檻都過了才有
//	6. 兩組特效（武器與護甲）、兩組附帶法術（只有武器）
//	7. 驅邪成功率：只寫在真的變壞的詛咒品上
//
// **所有型別比較都是 1-based 的。** 原版把型別寫進槽之後把區域變數 `+1`
// （`1990:11b6`），後面每一次比較用的都是 `型別 + 1`。這裡一律用
// `oneBased()` 明寫，不留隱式偏移（`docs/re/48` §2）。

// LootClassCount 是效果類別的數量。表在 `DEMON.INT DS:0x1941`
// （檔位移 `0x27441`），17 列 × 6 bytes，格式 `[候選數, e1 … e5]`。
// 第 18 列起就是字串資料，所以剛好 17 列，不多不少。
const LootClassCount = 17

// lootEffectPools 是每個類別的候選效果索引。
//
// 值域 0–41，全部落在法術表的 0–42 之內 —— 道具效果與法術共用同一個
// 索引空間（見 `docs/re/25` §2）。
var lootEffectPools = [LootClassCount][]int{
	{2, 10, 14},
	{1, 6, 19},
	{12, 15, 21},
	{7, 20},
	{0, 4, 31, 32},
	{13, 16},
	{3, 8, 9, 18, 22},
	{37},
	{33, 39},
	{30, 29, 28},
	{5, 11, 17},
	{35, 36},
	{34, 41},
	{40, 23},
	{24, 25},
	{26},
	{27},
}

// oneBased 把 0-based 的道具型別換成原版比較時用的值。
func oneBased(itemType int) int { return itemType + 1 }

// 型別分界（都是 1-based，見上）。分界落在
// 「武器 0–7、護甲 8–12、其餘 13–25」這條線上。
const (
	lootWeaponMax = 8  // 1-based ≤ 8 → 0-based 0–7，八把武器
	lootArmourMax = 13 // 1-based ≤ 13 → 0-based 0–12，武器加五件護甲
)

// lootRerollLimit 是各種「條件不合就重擲」的上限。
//
// 原版全部沒有上限 —— 它的迴圈理論上會一直轉，實務上擲不出來的機率極低。
// 這裡加上限是不讓一顆壞掉的資料把遊戲卡死。
const lootRerollLimit = 64

// storedOffset 是特效值與附魔在存檔裡共用的 +10 偏移。
const storedOffset = 10

// --- 1. 型別 ---

// lootTypeCount 是掉落道具的型別上限：`rnd(26) − 1` → 0–25。
//
// **26–29（火把、提燈、惡魔水晶、恆世寶珠）永遠不會掉** ——
// 前兩件是雜貨、後兩件是劇情物，掉出來會壞掉劇情。
const lootTypeCount = 26

// 型別的價格上限（`1990:1136`–`1193`）。
const (
	lootPriceBase  = 2.6 // CS:0x40e0 的 IEEE double
	lootPricePerLv = 25
)

// LootItemType 擲出掉落道具的型別（`1990:1128`），不做價格篩選。
func LootItemType(r *rng.RNG) int { return r.Roll(lootTypeCount) - 1 }

// LootPriceCap 是這個等級掉得出來的**最高底價**：`2.6^等級 + 25 × 等級`。
//
// 上一輪把這道門檻讀成「最低價格」，方向反了（`docs/re/48` §3）。
// 決定性的反證是量級：`ITEMS.DAT` 最貴的是寶石 500，而 8 級的
// `2.6^8 ≈ 2088` —— 當成下限的話 8 級以上一件都選不出來，迴圈會轉不完。
//
// 當上限就完全講得通：1 級只掉得出 27 以下的雜魚裝，7 級以後全部解鎖。
func LootPriceCap(level int) int {
	return int(float64(lootPricePerLv*level) + intPow(lootPriceBase, level))
}

// LootItemTypeFor 擲出這個等級掉得出來的道具型別。
//
// 太貴就重擲。用光重擲次數就退回最便宜的匕首 —— 那是 1 級也買得起的東西，
// 不會因為門檻算錯而憑空發財。
func LootItemTypeFor(r *rng.RNG, items *gamedata.ItemTable, level int) int {
	priceCap := LootPriceCap(level)
	for attempt := 0; attempt < lootRerollLimit; attempt++ {
		typ := LootItemType(r)
		item, err := items.ByIndex(typ)
		if err != nil {
			continue
		}
		if priceCap >= item.Price {
			return typ
		}
	}
	return 0
}

// --- 2. 材質類別 ---

// lootPlainTypes 是**不吃材質前綴**的道具型別（1-based，`1990:11b9`）。
//
// 0-based 是 8 布甲、9 皮甲、14 藥瓶、19 寶石、24 藥膏、25 藥瓶 ——
// 布與皮鍍不了銀，藥水藥膏是液體，寶石本身就是那個材質。
// 剩下的（含鎖子甲、鱗甲、板甲）都可以是銀的金的。
var lootPlainTypes = [...]int{9, 10, 15, 20, 25, 26}

// 材質類別的分界（`1990:11e1`–`1224`）。
const (
	lootMaterialDieBase = 10
	lootMaterialOffset  = 12
)

// LootMaterialClass 擲出材質類別（`+0x0f`，1–8）。
//
//	rnd(等級 + 10) → n
//	n < 9 → 1、n < 13 → 2、n < 15 → 3、n < 17 → 4、其餘 → n − 12
//
// **等級 10 是這條公式的天花板**：n 最大 `等級 + 10`，類別最大 `等級 − 2`，
// 超過 8 就讀出名稱表與倍率表之外的 bytes。原版沒有防呆，本專案鉗在 8 ——
// 照抄表外的垃圾值沒有意義，而且那些值會讓價格暴衝（`docs/re/48` §1）。
func LootMaterialClass(r *rng.RNG, itemType, level int) int {
	t1 := oneBased(itemType)
	for _, plain := range lootPlainTypes {
		if t1 == plain {
			return 1
		}
	}
	n := r.Roll(level + lootMaterialDieBase)
	var class int
	switch {
	case n < 9:
		class = 1
	case n < 13:
		class = 2
	case n < 15:
		class = 3
	case n < 17:
		class = 4
	default:
		class = n - lootMaterialOffset
	}
	if top := MaterialClassCount - 1; class > top {
		class = top
	}
	return class
}

// --- 3. 詛咒 ---

// 掉落自己擲的詛咒（`1990:124a`）：`rnd(10) == 10`，而且**只有武器與護甲**
// 會被詛咒（1-based < 14）。飾品、藥水那些不走這條。
const (
	lootCurseDie = 10
	lootCurseHit = 10
)

// --- 4. 附魔 ---

// 附魔擲點的常數（`1990:1296`）。
//
//	for i := 1; i <= max(1, level); i++ {
//	    if rnd(125) < 12 − i { 附魔 = i; break }
//	}
//
// 等級越高擲的次數越多，但每一級的門檻遞減 —— 高附魔既要運氣也要等級。
const (
	enchantDie  = 125
	enchantBase = 12
)

// LootEnchant 擲出附魔加成。
//
// **護甲減半、飾品以上根本沒有附魔**：1-based > 8（0-based ≥ 8，布甲起）
// 減半，1-based > 13（0-based ≥ 13，王冠起）連這一段都不進（`1990:126e`）。
// 減半公式是 `(附魔 + 1) / 2` —— +1 還是 +1、+2 也是 +1、+3 變 +2。
func LootEnchant(r *rng.RNG, level, itemType int) int {
	t1 := oneBased(itemType)
	if t1 > lootArmourMax {
		return 0
	}
	n := level
	if n <= 0 {
		n = 1
	}
	enchant := 0
	for i := 1; i <= n; i++ {
		if r.Roll(enchantDie) < enchantBase-i {
			enchant = i
			break
		}
	}
	if t1 > lootWeaponMax {
		enchant = (enchant + 1) / 2
	}
	return enchant
}

// --- 5. 效果與次數 ---

// 兩道效果門檻（`1990:1308`／`1990:134c`）。兩道都要通過才會長出效果。
const (
	effectGateAMul  = 2
	effectGateAPlus = 6
	effectGateBPlus = 16
	effectGateDie   = 100
)

// 充能種類。來源是 `ITEMS.DAT` 的第三個數字欄（本專案的 `CategoryIndex`）——
// 那個欄位一直標「語意未確認」，生成端 `1990:1334` 拿它當這個用。
// 值為 0 時現場擲 `rnd(3)`。
const (
	// ChargeUnlimitedUses 把已用次數寫成 0xFF。
	//
	// **那個 255 是哨兵不是計數。** 使用判定是「上限 != 已用才可用」，
	// 而睡覺又跳過已用次數為 0xFF 的道具不充能 —— 兩條合起來的效果是
	// **這件道具永遠可用**。剛生成的道具「已經用掉 255 次」講不通，
	// 講得通的是「這一格不計次」。
	ChargeUnlimitedUses = 1
	ChargeManyUses      = 2 // 次數上限 100 以上：過夜同樣不充能
	ChargeFewUses       = 3 // 個位數次數，會充能
)

// 充能種類 2、3 的常數。
const (
	chargeManyBase   = 121
	chargeManyOffset = 5
	chargeManyFixed  = 200 // 兩種藥瓶固定 200 次
	chargeFewBase    = 3
	chargeFewDivisor = 8
	chargeFewBonusIn = 5 // rnd(5) == 1 再多一次
)

// 幾個被特別處理的型別（1-based）。
const (
	// chargeFixedVial／lootFixedTypeVial 是兩種藥瓶（0-based 14 與 25）。
	chargeFixedVial    = 15
	lootFixedTypeVial  = 26
	lootFixedTypeSalve = 25 // 藥膏（0-based 24）
	// lootFixedEffect 是藥膏與第二種藥瓶擲到 0 或 4 時被硬寫的效果，
	// 而且順便讓這件東西變成詛咒品（`1990:1433`）。
	lootFixedEffect = 39
	// lootVialBanned 是第二種藥瓶擲到就重擲的效果。
	lootVialBanned = 23
)

// LootCharges 依充能種類算出使用次數上限，以及已用次數的初值。
//
// 三種都對得上 `docs/re/26` §3.3 記的兩個「過夜不充能」例外 ——
// 種類 1 把已用次數寫成 0xFF、種類 2 的上限一定 >= 100，
// 正好就是睡覺常式跳過的那兩種。**兩邊獨立解出來卻剛好互相解釋**。
//
// 種類 2 的 200 次固定值給的是 1-based 15 與 26，也就是**兩種藥瓶**
// （0-based 14／25）。這裡原本寫成「戒指與火把」，是把 1-based 的比較
// 直接套到 0-based 型別上（`docs/re/48` §2）。
func LootCharges(r *rng.RNG, kind, level, itemType, strength int) (total, used int) {
	t1 := oneBased(itemType)
	if kind == 0 {
		kind = r.Roll(3)
	}
	switch kind {
	case ChargeUnlimitedUses:
		return r.Roll(level*2) + 1, restNeverRecharge
	case ChargeManyUses:
		if t1 == chargeFixedVial || t1 == lootFixedTypeVial {
			return chargeManyFixed, 0
		}
		return chargeManyBase - r.Roll(level+chargeManyOffset), 0
	default:
		// **強度越高次數越少，而且會減成負的。** 原版把算出來的 word
		// 用 `mov al` 存進 byte，所以 −2 會變成 254 —— 一件高強度的
		// 充能種類 3 道具反而變成幾乎用不完。這裡照著取低位元組，
		// 不是為了忠實而忠實：不取的話記憶體裡是 −2、存檔讀回來是 254，
		// 同一件道具在存檔前後行為會不一樣。
		t := chargeFewBase - strength/chargeFewDivisor
		if r.Roll(chargeFewBonusIn) == 1 {
			t++
		}
		return t & 0xff, 0
	}
}

// LootEffectClass 依道具的四個類別欄位決定效果類別（回傳 1–17）。
//
// **第一個欄位是 0 還是非 0，決定另外三個是排除還是候選** ——
// 這是 `docs/re/25` 當初卡住的地方，原因是把 `jne` 讀成了 `je`：
//
//	f[0] != 0 → 從四個欄位裡隨機挑一個（候選清單）
//	f[0] == 0 → rnd(17)，撞到 f[1..3] 就重擲（排除清單）
//
// 兩條路都保證結果非零，所以後面減一之後永遠落在 0–16，不會索引到表外。
// 武器的 `0,10,8,9` 是「除了 10、8、9 以外都行」，護甲的 `12,7,7,7`
// 是「12 或 7」，`15,15,15,15`（雕像）則是「一定是 15」。
func LootEffectClass(r *rng.RNG, classes [4]int) int {
	if classes[0] != 0 {
		return classes[r.Roll(4)-1]
	}
	for {
		c := r.Roll(LootClassCount)
		if c != classes[1] && c != classes[2] && c != classes[3] {
			return c
		}
	}
}

// --- 6. 兩組特效與兩組附帶法術 ---

// 特效那兩輪的常數（`1990:1926`–`1a2c`）。
const (
	// 兩輪各自的門檻：`rnd(100) − 1` 要小於等於 `等級 + 這個數`。
	condRound0Plus = 8
	condRound1Plus = 5
	// 不是詛咒品時有四成機率走「雜項」那條：條件碼 0–7，值寫死 12。
	condPlainDie   = 10
	condPlainUnder = 5
	condPlainSpan  = 8
	condPlainValue = 12
	// 另一條的條件碼是 `rnd(5) + 16` → 17–21。
	// **只有 21（`0x15`）會讓後面那個特效值真的生效。**
	condCodeDie  = 5
	condCodeBase = 16
	// 特效級數的爬升擲點：`rnd(15) − 1 < 等級 − k` 就再加一級。
	condStepDie = 15
)

// 附帶法術那兩輪的常數（`1990:1a3c`–`1b36`）。
const (
	spellRound0Plus = 3
	// 強度 = `rnd(22 × 等級 / 10) + 4`（原版的整數除法）。
	spellPowerMulNum = 22
	spellPowerMulDen = 10
	spellPowerPlus   = 4
	// 詛咒品而且強度超過 14 時，法術被硬寫成 26。
	spellCursedPower = 14
	spellCursedIndex = 26
	// 法術表 `+4` 這個欄位為 0 代表不能附在武器上；為 2 就把索引的
	// 最高位元點起來（那個旗標的用途還沒解，照抄）。
	spellW4Blocked = 0
	spellW4Flag    = 2
	spellHighBit   = 0x80
	spellCount     = 43
)

// rollConditions 擲出兩組特效（`+0x09`–`+0x0c`）。武器與護甲才有。
//
// 兩輪各自擲一次通過機率，過了就寫一組。走哪一條分兩種：
//
//   - 不是詛咒品而且擲中四成 → 條件碼 0–7、值寫死 12（不生效的雜項）
//   - 其餘 → 條件碼 17–21、值是一個隨等級爬升的級數（詛咒品取負）
//
// 只有條件碼 21 會讓值生效，而 21 對護甲是禁止的 ——
// **所以護甲永遠拿不到會生效的特效**（`1990:19b7`）。
func rollConditions(r *rng.RNG, slot *scenario.InventorySlot, t1, level int, cursed bool) {
	if t1 > lootArmourMax {
		return
	}
	for round := 0; round < 2; round++ {
		v := r.Roll(effectGateDie) - 1
		limit := level + condRound0Plus
		if round == 1 {
			limit = level + condRound1Plus
		}
		if limit < v {
			continue
		}

		if !cursed && r.Roll(condPlainDie) < condPlainUnder {
			setCondition(slot, round, r.Roll(condPlainSpan)-1, condPlainValue)
			continue
		}

		step := 1
		for i := 0; i < lootRerollLimit; i++ {
			if r.Roll(condStepDie)-1 >= level-step {
				break
			}
			step++
		}

		code := 0
		for i := 0; i < lootRerollLimit; i++ {
			idx := r.Roll(condCodeDie)
			if idx == condCodeDie && t1 > lootWeaponMax {
				continue
			}
			if round != 0 && slot.CondA == idx+condCodeBase {
				continue
			}
			code = idx + condCodeBase
			break
		}
		if code == 0 {
			continue
		}
		if cursed {
			step = -step
		}
		setCondition(slot, round, code, step+storedOffset)
	}
}

// setCondition 把一組（條件碼, 值）寫進第 round 組。
func setCondition(slot *scenario.InventorySlot, round, code, value int) {
	if round == 0 {
		slot.CondA, slot.EffectAByte = code, value
		return
	}
	slot.CondB, slot.EffectBByte = code, value
}

// rollItemSpells 擲出兩組附帶法術（`+0x01`–`+0x04`）。**只有武器有。**
//
// 兩輪的門檻不一樣：第一輪要 `rnd(100) − 1 <= 等級 + 3`，
// 第二輪嚴一級，要 `<= 等級`。所以第二個法術比第一個難得多。
func rollItemSpells(r *rng.RNG, t *gamedata.Tables, slot *scenario.InventorySlot,
	t1, level int, cursed bool) {

	if t1 > lootWeaponMax {
		return
	}
	for round := 0; round < 2; round++ {
		v := r.Roll(effectGateDie) - 1
		limit := level
		if round == 0 {
			limit = level + spellRound0Plus
		}
		if limit < v {
			continue
		}

		power := r.Roll(spellPowerMulNum*level/spellPowerMulDen) + spellPowerPlus

		index, flag, ok := 0, 0, false
		for i := 0; i < lootRerollLimit; i++ {
			s := r.Roll(spellCount) - 1
			if power > spellCursedPower && cursed {
				s = spellCursedIndex
			}
			sp, err := t.Spell(s)
			if err != nil || sp.M > power || sp.W4 == spellW4Blocked {
				continue
			}
			index, ok = s, true
			if sp.W4 == spellW4Flag {
				flag = spellHighBit
			}
			break
		}
		if !ok {
			continue
		}
		if round == 0 {
			slot.SpellA, slot.SpellAPower = flag+index, power
		} else {
			slot.SpellB, slot.SpellBPower = flag+index, power
		}
	}
}

// --- 7. 驅邪成功率 ---

// 驅邪成功率只寫在**真的變壞了**的詛咒品上（`1990:1b37`，只有掉落走這條）。
//
//	成功率 = 51 − rnd(5 × 等級)
//
// 值越大越好驅，所以等級越高的詛咒品越難解。
const (
	exorciseBase = 51
	exorciseMul  = 5
)

// --- 生成入口 ---

// GenerateDrop 生成一件戰鬥／寶箱掉落的道具（原版的模式 0）。
//
// 詛咒是**這裡自己擲的**：`rnd(10) == 10`，而且只有武器與護甲會中。
// 中了的話附魔取負、不給效果，最後補一個驅邪成功率。
func GenerateDrop(r *rng.RNG, t *gamedata.Tables, item gamedata.Item,
	itemType, level int) scenario.InventorySlot {

	slot, _ := generateLoot(r, t, item, itemType, level, false, 0)
	return slot
}

// GenerateWare 生成一件商隊的貨（原版的模式 1）。
//
// **商隊那條路跟掉落完全不同**（`docs/re/48` §5）：chance 擲中的貨不是被
// 詛咒，而是**跳過第一道效果門檻**。附魔不取負，也不寫驅邪成功率。
// 第二個回傳值就是原版存進 `ds:0x4e2e`、由商隊逐件記下來的那個旗標。
func GenerateWare(r *rng.RNG, t *gamedata.Tables, item gamedata.Item,
	itemType, level, chance int) (scenario.InventorySlot, bool) {

	return generateLoot(r, t, item, itemType, level, true, chance)
}

func generateLoot(r *rng.RNG, t *gamedata.Tables, item gamedata.Item,
	itemType, level int, merchant bool, chance int) (scenario.InventorySlot, bool) {

	t1 := oneBased(itemType)
	slot := scenario.InventorySlot{Type: byte(itemType)}
	slot.MaterialClass = LootMaterialClass(r, itemType, level)

	cursed, flagged := false, false
	if merchant {
		flagged = r.Roll(effectGateDie) < chance
	} else {
		cursed = r.Roll(lootCurseDie) >= lootCurseHit && t1 <= lootArmourMax
	}

	enchant := LootEnchant(r, level, itemType)
	if cursed {
		enchant = -enchant
	}
	slot.Enchant = enchant

	// 第一道門檻。**擲中的商隊貨直接跳過**，其餘沒過就到此為止 ——
	// 連特效與附帶法術那兩段都不跑，這件就是徹底的平凡貨。
	if !(merchant && flagged) {
		if r.Roll(effectGateDie) > level*effectGateAMul+effectGateAPlus {
			return slot, flagged
		}
	}

	// 充能種類要在第二道門檻**之前**決定：原版就是這個順序，
	// 亂數的消耗順序跟著它走。
	kind := item.CategoryIndex
	if kind == 0 {
		kind = r.Roll(3)
	}

	// 第二道門檻，以及「被詛咒的不給效果」。沒過就直接跳到特效那一段。
	if r.Roll(effectGateDie) <= level+effectGateBPlus && !cursed {
		effect, strength, gotCursed, ok := rollEffect(r, t, item, t1, level)
		if gotCursed {
			cursed = true
		}
		if ok {
			slot.Effect = effect
			slot.Power = strength
			slot.Total, slot.Used = LootCharges(r, kind, level, itemType, strength)
		}
	}

	rollConditions(r, &slot, t1, level, cursed)
	rollItemSpells(r, t, &slot, t1, level, cursed)

	if !merchant && cursed && lootDamaged(slot) {
		// 原版把算出來的 word 用 `mov al` 存進一個 byte，所以 11 級以上
		// 減成負的時候會繞回 247 附近 —— **反而變成一驅就掉**。
		// 照著取低位元組，理由與 LootCharges 同一條：不取的話記憶體是負數、
		// 存檔讀回來是大正數，同一件道具在存檔前後行為會不一樣。
		slot.ExorciseResist = (exorciseBase - r.Roll(exorciseMul*level)) & 0xff
	}
	return slot, flagged
}

// lootDamaged 回報這件道具是不是真的變壞了（`1990:1b3f`–`1b6c`）。
//
// 五個 byte 全是 0、附魔又沒有變負，代表詛咒擲中了卻什麼都沒發生 ——
// 那種東西不必給驅邪成功率。
func lootDamaged(slot scenario.InventorySlot) bool {
	return slot.SpellAPower != 0 || slot.SpellBPower != 0 || slot.Power != 0 ||
		slot.EffectAByte != 0 || slot.EffectBByte != 0 || slot.Enchant < 0
}

// rollEffect 擲出效果與強度。
//
// **強度只擲一次。** 效果配不上強度時原版跳回的是類別那一步
// （`1990:13bc`），強度那一行在它前面 —— 重擲的是「類別 + 效果」兩件事。
func rollEffect(r *rng.RNG, t *gamedata.Tables, item gamedata.Item,
	t1, level int) (effect, strength int, cursed, ok bool) {

	strength = r.Roll(level*2) + 2
	for attempt := 0; attempt < lootRerollLimit; attempt++ {
		class := LootEffectClass(r, item.EffectClasses)
		pool := lootEffectPools[class-1]

		picked := false
		for inner := 0; inner < lootRerollLimit; inner++ {
			effect = pool[r.Roll(len(pool))-1]
			if t1 == lootFixedTypeSalve || t1 == lootFixedTypeVial {
				if effect == 0 || effect == 4 {
					effect = lootFixedEffect
					cursed = true
				}
			}
			if t1 == lootFixedTypeVial && effect == lootVialBanned {
				continue
			}
			picked = true
			break
		}
		if !picked {
			continue
		}
		if sp, err := t.Spell(effect); err == nil && sp.M > strength {
			continue // 強度不夠付這個效果的最低法力
		}
		return effect, strength, cursed, true
	}
	return 0, 0, cursed, false
}

// --- 戰鬥勝利的掉落 ---

// 掉落機率的兩個係數（`0xe3be` 一帶）：
//
//	if rnd(100) <= 怪物等級×6 + 5 → 這一隻掉一件
//
// 逐隻獨立判定，所以打贏一群低等怪也可能一件都沒有。
const (
	dropChanceMul  = 6
	dropChancePlus = 5
)

// BattleDropChance 回傳一隻怪物掉東西的百分比機率。
func BattleDropChance(monsterLevel int) int {
	return monsterLevel*dropChanceMul + dropChancePlus
}

// RollBattleDrops 依戰敗怪物的等級擲出戰利品。
//
// **每隻怪物各自判定**（`rnd(100) <= 等級×6 + 5`），中的那隻掉一件
// 型別隨機的道具 —— 型別還要過價格上限那一關，所以低等怪掉不出貴東西。
//
// 掉落的道具**是未鑑定的**，而且可能被詛咒 —— 戰場上撿到的東西
// 本來就不知道好壞，這是原版的設計，不是缺漏。
func RollBattleDrops(r *rng.RNG, t *gamedata.Tables, items *gamedata.ItemTable,
	monsterLevels []int) []scenario.InventorySlot {

	var out []scenario.InventorySlot
	for _, level := range monsterLevels {
		if r.Roll(100) > BattleDropChance(level) {
			continue
		}
		typ := LootItemTypeFor(r, items, level)
		item, err := items.ByIndex(typ)
		if err != nil {
			continue
		}
		out = append(out, GenerateDrop(r, t, item, typ, level))
	}
	return out
}

// --- 戰鬥勝利的經驗值（`1990:0d0e` 一帶，見 `docs/re/56`）---

// expBoundStatus 是「拿不到經驗值」的狀態下界。
//
// 原版是 `if 角色[+0x102] >= 2 → 跳過`（`0xe264`）。角色記錄 `+0x102`
// 是戰鬥狀態的**單一列舉值**：0 正常／1 中毒／2–4 束縛三級／5 死亡
// （`docs/re/26`）。所以**中毒照樣拿經驗，被束縛或死亡就沒有** ——
// 分界剛好切在「還能自己行動」與「不能」之間。
const expBoundStatus = 2

// BattleExpPerCharacter 回傳每人分到多少經驗值。
//
// **總經驗除以隊伍人數**（`0xe235` 的有號 32-bit 除法）——
// 訊息「Exp per chr」的 per chr 就是這麼來的。**分母是全隊人數，
// 不是拿得到經驗的人數**：有人被束縛或死亡時，那一份不會分給活著的人，
// 直接消失。
func BattleExpPerCharacter(total, partySize int) int {
	if partySize <= 0 {
		return 0
	}
	return total / partySize
}

// AwardBattleExp 把經驗值發給隊伍，回傳每人實際拿到多少。
//
// 封頂 `0x00FFFFFF`（`0xe289` 的 `cmp [+0xc6],0xff`），與 CapValue 同一個上限。
func AwardBattleExp(party []Character, statuses []UnitStatus, total int) int {
	per := BattleExpPerCharacter(total, len(party))
	if per == 0 {
		return 0
	}
	for i := range party {
		if i < len(statuses) && statuses[i] >= expBoundStatus {
			continue // 束縛或死亡：這一份就消失
		}
		party[i].Experience = CapValue(party[i].Experience + per)
	}
	return per
}

// --- 戰鬥勝利的金幣（`0xe2bf`–`0xe3ab`，見 `docs/re/56`）---

// 金幣的兩個底數在同一個常數池，中間只隔 8 bytes：
// `1.7` 在 `1990:40f0`、`2.1` 在 `1990:40e8`。
//
// 指數是怪物的 `MONSTER.DAT` level（1–10）。原版讀的是 `unit+0x1a`，
// 那一欄對玩家單位是附魔加成、對怪物是 level —— 兩張屬性表是同一套
// schema，12 隻召喚生物與同名怪物有 96 個數值零誤差（`docs/re/57`）。
const (
	goldPowBase  = 1.7
	goldRollBase = 2.1
	// goldPerUnit 是每隻怪物固定加的 3（`0xe327` 連三個 inc ax）。
	goldPerUnit = 3
)

// intPow 是 `1990:0a83`：acc 從 1.0 起跳，連乘 exp 次。
//
// 指數一律是整數 —— 全檔沒有任何非整數次方的證據，這個引擎也沒有
// log／exp（`docs/re/55` §2）。
func intPow(base float64, exp int) float64 {
	p := 1.0
	for i := 0; i < exp; i++ {
		p *= base
	}
	return p
}

// BattleGold 擲出這一場戰鬥的金幣總額。
//
// 每隻怪物各出 `1.7^level + Roll(2.1^level) + 3`，`Roll` 回傳 1..n。
// 量級：1 級怪 5–6 枚、5 級怪 18–57 枚、10 級怪 205–1871 枚。
//
// 原版掃的是**全部怪物單位**，不是「死掉的那些」（迴圈邊界是
// `party+0xa6` 怪物數）。勝利的條件就是全滅，所以實務上等價。
func BattleGold(r *rng.RNG, monsterLevels []int) int {
	total := 0
	for _, lv := range monsterLevels {
		base := int(intPow(goldPowBase, lv))
		spread := int(intPow(goldRollBase, lv))
		total += base + r.Roll(spread) + goldPerUnit
	}
	return total
}
