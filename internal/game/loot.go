package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 掉寶生成（`DEMON.INT 1990:1250`–`1990:1522`，見 `docs/re/30`）。
//
// 這是「為什麼店裡買的東西都沒有效果」的答案：**效果是掉寶時長出來的**，
// `ITEMS.DAT` 只規定每一類道具能長出哪些效果。
//
// 整個流程分三段，每一段都可能提前結束：
//
//	1. 附魔：擲點決定 +1…+N，護甲與飾品減半，被詛咒的取負
//	2. 效果：兩道機率門檻都過了才有效果，再兩層擲點決定是哪一個
//	3. 次數：依道具的「充能種類」算出使用次數上限
//
// 沒過門檻的就是一件平凡裝備 —— 原版起始存檔那 11 件正是這樣。

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

// 附魔擲點的常數（`1990:1296`）。
//
//	for i := 1; i <= max(1, level); i++ {
//	    if rnd(125) < 12 − i { 附魔 = i; break }
//	}
//
// 等級越高擲的次數越多，但每一級的門檻遞減 —— 高附魔既要運氣也要等級。
const (
	enchantDie      = 125
	enchantBase     = 12
	enchantHalfFrom = 8 // 道具型別 > 8（護甲與飾品）附魔減半
)

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
	chargeManyFixed  = 200 // 戒指與火把固定 200 次
	chargeFewBase    = 3
	chargeFewDivisor = 8
	chargeFewBonusIn = 5 // rnd(5) == 1 再多一次
)

// 兩件被特別處理的道具型別（`1990:14ba`／`1990:1433`）。
const (
	itemTypeRing  = 15
	itemTypeTorch = 26
	itemTypeVial2 = 25
	// lootFixedEffect 是型別 25／26 在特定情況下被硬寫的效果索引。
	lootFixedEffect = 39
)

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

// LootEnchant 擲出附魔加成。
//
// itemType > 8 的（護甲與飾品）減半，公式是 `(附魔 + 1) / 2` ——
// 所以 +1 還是 +1、+2 也是 +1、+3 變 +2。
func LootEnchant(r *rng.RNG, level, itemType int) int {
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
	if itemType > enchantHalfFrom {
		enchant = (enchant + 1) / 2
	}
	return enchant
}

// LootCharges 依充能種類算出使用次數上限，以及已用次數的初值。
//
// 三種都對得上 `docs/re/26` §3.3 記的兩個「過夜不充能」例外 ——
// 種類 1 把已用次數寫成 0xFF、種類 2 的上限一定 >= 100，
// 正好就是睡覺常式跳過的那兩種。**兩邊獨立解出來卻剛好互相解釋**。
func LootCharges(r *rng.RNG, kind, level, itemType, strength int) (total, used int) {
	if kind == 0 {
		kind = r.Roll(3)
	}
	switch kind {
	case ChargeUnlimitedUses:
		return r.Roll(level*2) + 1, restNeverRecharge
	case ChargeManyUses:
		if itemType == itemTypeRing || itemType == itemTypeTorch {
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

// GenerateLoot 生成一件掉落的道具。
//
// level 是掉落來源的等級（怪物或寶箱），itemType 是 `ITEMS.DAT` 的索引。
// 回傳的道具**未鑑定** —— 原版的鑑定是另外付錢的服務。
//
// 兩道效果門檻沒過就回傳平凡裝備（只有附魔，沒有效果）。
// 被詛咒的道具**一定沒有效果**（`1990:1363`）。
func GenerateLoot(r *rng.RNG, t *gamedata.Tables, item gamedata.Item, itemType, level int, cursed bool) scenario.InventorySlot {
	slot := scenario.InventorySlot{Type: byte(itemType)}

	enchant := LootEnchant(r, level, itemType)
	if cursed {
		enchant = -enchant
	}
	slot.Enchant = enchant

	// 第一道門檻：機率隨等級上升。沒過就是平凡裝備。
	if r.Roll(effectGateDie) > level*effectGateAMul+effectGateAPlus {
		return slot
	}
	// 第二道門檻，以及「被詛咒的不給效果」。
	if r.Roll(effectGateDie) > level+effectGateBPlus || cursed {
		return slot
	}

	// 效果與強度要一起成立：強度不足以支付該效果的最低法力就整組重擲。
	// 原版是 `jmp 0x13bc` 跳回強度那一步之後、類別那一步之前，
	// 所以重擲的是「強度 + 類別 + 效果」三件事。
	for attempt := 0; attempt < lootRerollLimit; attempt++ {
		strength := r.Roll(level*2) + 2
		class := LootEffectClass(r, item.EffectClasses)
		pool := lootEffectPools[class-1]
		effect := pool[r.Roll(len(pool))-1]

		// 型別 25／26 的特例（`1990:1433`）。
		if itemType == itemTypeVial2 || itemType == itemTypeTorch {
			if effect == 0 {
				effect = lootFixedEffect
			}
		}
		sp, err := t.Spell(effect)
		if err == nil && sp.M > strength {
			continue // 強度不夠付這個效果的最低法力
		}

		slot.Effect = effect
		slot.Power = strength
		slot.Total, slot.Used = LootCharges(r, item.CategoryIndex, level, itemType, strength)
		return slot
	}
	// 重擲太多次就當平凡裝備。原版沒有這個上限 —— 它的迴圈理論上會一直轉，
	// 實務上低等級 + 高最低法力的組合擲不出來的機率極低。這裡加上限是
	// 不讓一顆壞掉的資料把遊戲卡死。
	return slot
}

// lootRerollLimit 是「強度配不上效果」時的重擲上限。見 GenerateLoot 結尾。
const lootRerollLimit = 64

// --- 戰鬥勝利的掉落 ---

// lootTypeCount 是掉落道具的型別上限：`rnd(26) − 1` → 0–25。
//
// **26–29（火把、提燈、惡魔水晶、恆世寶珠）永遠不會掉** ——
// 前兩件是雜貨、後兩件是劇情物，掉出來會壞掉劇情。
const lootTypeCount = 26

// LootItemType 擲出掉落道具的型別（`1990:1128`）。
func LootItemType(r *rng.RNG) int { return r.Roll(lootTypeCount) - 1 }

// 掉落機率的兩個係數（`FUN_0990_?` 的 `0xe3be` 一帶）：
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
// 型別隨機（0–25）的道具，內容走 GenerateLoot。
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
		typ := LootItemType(r)
		item, err := items.ByIndex(typ)
		if err != nil {
			continue
		}
		out = append(out, GenerateLoot(r, t, item, typ, level, false))
	}
	return out
}
