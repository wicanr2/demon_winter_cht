package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 候選表的形狀：17 列、每列 1–5 個、效果索引都在法術表範圍內。
//
// 這一條同時釘住「表就是 17 列」—— 第 18 列起是字串資料，多讀一列
// 會拿到 count=171 這種明顯不對的值。
func TestLootEffectPools_Shape(t *testing.T) {
	if len(lootEffectPools) != LootClassCount {
		t.Fatalf("候選表 %d 列，預期 %d", len(lootEffectPools), LootClassCount)
	}
	for i, pool := range lootEffectPools {
		if len(pool) < 1 || len(pool) > 5 {
			t.Errorf("第 %d 列有 %d 個候選，每列最多 5 個", i, len(pool))
		}
		for _, e := range pool {
			if e < 0 || e >= gamedata.NumSpellRecords {
				t.Errorf("第 %d 列的效果索引 %d 超出法術表 0–%d",
					i, e, gamedata.NumSpellRecords-1)
			}
		}
	}
}

// **每一件真實道具的類別欄位都算得出合法類別。**
//
// 這是 `docs/re/25` 當初擔心的那件事：「挑到 0 減 1 會索引到表外」。
// 兩條路徑各自保證非零，所以不會 —— 拿 30 件真貨全部跑一遍證明。
func TestLootEffectClass_AllRealItemsStayInRange(t *testing.T) {
	items := loadItems(t)
	r := rng.NewWithSeed(7)
	for i, it := range items.All() {
		for n := 0; n < 200; n++ {
			c := LootEffectClass(r, it.EffectClasses)
			if c < 1 || c > LootClassCount {
				t.Fatalf("道具 %d（%s）類別 %v 算出 %d，超出 1–%d",
					i, it.Name, it.EffectClasses, c, LootClassCount)
			}
		}
	}
}

// 第一個欄位非 0 → 候選清單；為 0 → 排除清單。
func TestLootEffectClass_InclusionVsExclusion(t *testing.T) {
	r := rng.NewWithSeed(11)

	// 雕像 [15,15,15,15]：四個都一樣，結果一定是 15。
	for n := 0; n < 50; n++ {
		if c := LootEffectClass(r, [4]int{15, 15, 15, 15}); c != 15 {
			t.Fatalf("候選清單全是 15，卻算出 %d", c)
		}
	}

	// 武器 [0,10,8,9]：排除 10、8、9，其餘 14 種都可能。
	seen := map[int]bool{}
	for n := 0; n < 3000; n++ {
		c := LootEffectClass(r, [4]int{0, 10, 8, 9})
		if c == 10 || c == 8 || c == 9 {
			t.Fatalf("排除清單裡的 %d 被選中了", c)
		}
		seen[c] = true
	}
	if len(seen) != LootClassCount-3 {
		t.Errorf("排除三種後應該還有 %d 種可能，只出現 %d 種",
			LootClassCount-3, len(seen))
	}
}

// 護甲與飾品的附魔減半：(n+1)/2。
func TestLootEnchant_HalvedForNonWeapons(t *testing.T) {
	r := rng.NewWithSeed(3)
	maxWeapon, maxArmor := 0, 0
	for n := 0; n < 5000; n++ {
		if v := LootEnchant(r, 10, 0); v > maxWeapon {
			maxWeapon = v
		}
		if v := LootEnchant(r, 10, 12); v > maxArmor {
			maxArmor = v
		}
	}
	if maxWeapon <= maxArmor {
		t.Errorf("武器最高附魔 %d、護甲 %d —— 護甲應該明顯低一截",
			maxWeapon, maxArmor)
	}
	// 附魔上限是等級（迴圈只跑到 level 次）。
	if maxWeapon > 10 {
		t.Errorf("10 級掉出 +%d，附魔不該超過等級", maxWeapon)
	}
}

// 充能三種：種類 1 已用次數是 0xFF、種類 2 上限 >= 100 —— 這兩個正好
// 就是睡覺常式跳過充能的兩個條件。
func TestLootCharges_MatchesRestExceptions(t *testing.T) {
	r := rng.NewWithSeed(5)

	total, used := LootCharges(r, ChargeUnlimitedUses, 5, 16, 10)
	if used != restNeverRecharge {
		t.Errorf("種類 1 的已用次數 %d，預期 %d（過夜不充能的哨兵）",
			used, restNeverRecharge)
	}
	if total < 1 {
		t.Errorf("種類 1 的上限 %d，至少要 1", total)
	}

	for n := 0; n < 200; n++ {
		total, used = LootCharges(r, ChargeManyUses, 5, 19, 10)
		if total < restRechargeMaxTotal {
			t.Fatalf("種類 2 的上限 %d，應該 >= %d（過夜不充能的另一個條件）",
				total, restRechargeMaxTotal)
		}
		if used != 0 {
			t.Fatalf("種類 2 的已用次數應該是 0，得到 %d", used)
		}
	}

	// 兩種藥瓶固定 200（1-based 15／26 → 0-based 14／25，見 docs/re/48 §2）。
	for _, typ := range []int{chargeFixedVial - 1, lootFixedTypeVial - 1} {
		if total, _ := LootCharges(r, ChargeManyUses, 5, typ, 10); total != chargeManyFixed {
			t.Errorf("型別 %d 的次數 %d，預期 %d", typ, total, chargeManyFixed)
		}
	}

	// 種類 3 是個位數，而且**強度越高次數越少**。
	if total, _ := LootCharges(r, ChargeFewUses, 5, 0, 24); total > 2 {
		t.Errorf("強度 24 的種類 3 次數 %d，應該被 −24/8 壓下來", total)
	}
	// 減成負的時候照原版取低位元組（−2 → 254），不留負數在記憶體裡 ——
	// 存檔寫的是一個 byte，記憶體與磁碟不一致才是真的難查。
	for n := 0; n < 200; n++ {
		total, _ := LootCharges(r, ChargeFewUses, 20, 0, 40)
		if total < 0 || total > 255 {
			t.Fatalf("次數 %d 不在 byte 範圍內", total)
		}
	}
}

// 生成的道具：效果索引與強度要一起成立。
//
// 強度必須付得起該效果的最低法力，否則原版會整組重擲 ——
// 生出「有效果卻用不了」的道具是最難察覺的一種錯。
func TestGenerateLoot_StrengthCoversEffectCost(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(13)

	withEffect := 0
	for n := 0; n < 4000; n++ {
		typ := n % items.Len()
		it, err := items.ByIndex(typ)
		if err != nil {
			t.Fatal(err)
		}
		slot := GenerateDrop(r, tb, it, typ, 12)

		if slot.Type != byte(typ) {
			t.Fatalf("道具型別 %d，預期 %d", slot.Type, typ)
		}
		if slot.Power == 0 {
			continue // 沒過門檻的平凡裝備
		}
		withEffect++

		sp, err := tb.Spell(slot.Effect)
		if err != nil {
			t.Fatalf("效果索引 %d 查不到法術記錄：%v", slot.Effect, err)
		}
		if sp.M > slot.Power {
			t.Fatalf("效果 %d 的最低法力 %d 大於強度 %d —— 這件道具用不了",
				slot.Effect, sp.M, slot.Power)
		}
		// **不能要求「有效果就一定可用」。** 充能種類 3（武器／護甲／
		// 冠冕／勳章）的次數是 `3 − 強度/8`，高強度會算出 0 ——
		// 那類道具的效果本來就是裝備著被動生效，不走 Use 選單。
		if slot.Total > 0 && !slot.Usable() {
			t.Fatalf("有次數卻不可用：%+v", slot)
		}
	}
	if withEffect == 0 {
		t.Error("跑了 4000 次一件有效果的都沒生出來，門檻算錯了")
	}
}

// 被詛咒的道具：附魔為負，而且**一定沒有效果**。
//
// 詛咒是生成器自己擲的（`rnd(10) == 10`，只有武器與護甲會中），
// 呼叫端沒得指定 —— 所以這裡是撈一大堆樣本再驗那條不變式。
func TestGenerateLoot_CursedHasNoEffect(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(17)

	negative, resisted := 0, 0
	for n := 0; n < 3000; n++ {
		it, _ := items.ByIndex(0)
		slot := GenerateDrop(r, tb, it, 0, 12)
		if slot.Enchant >= 0 {
			continue
		}
		negative++
		if slot.Power != 0 || slot.Effect != 0 {
			t.Fatalf("被詛咒的道具長出了效果：%+v", slot)
		}
		if slot.ExorciseResist != 0 {
			resisted++
		}
	}
	if negative == 0 {
		t.Error("3000 次都沒擲出負附魔，詛咒那條沒生效")
	}
	// 51 − rnd(5×等級) 有機會剛好是 0（12 級時 rnd(60) 擲到 51），
	// 所以不能要求每一件都非零，只能要求整體上寫得出來。
	if resisted == 0 {
		t.Error("一件詛咒品都沒有驅邪成功率")
	}
}

// 掉落的道具是**未鑑定**的 —— 鑑定是市集另外收錢的服務。
func TestGenerateLoot_NotIdentified(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(19)
	it, _ := items.ByIndex(5)
	for n := 0; n < 100; n++ {
		if GenerateDrop(r, tb, it, 5, 8).Identified {
			t.Fatal("掉落的道具不該是已鑑定的")
		}
	}
}

// loadItems 讀真實的 ITEMS.DAT。沒有原版資料就略過。
func loadItems(t *testing.T) *gamedata.ItemTable {
	t.Helper()
	p := filepath.Join(repoRoot(t), "workplace", "orig", "demwin", "DEM_DATA", "ITEMS.DAT")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("game: 找不到 %s，略過需要真實資料的測試", p)
	}
	tb, err := gamedata.LoadItemTable(p)
	if err != nil {
		t.Fatalf("LoadItemTable: %v", err)
	}
	return tb
}

// 掉落型別只到 25 —— 火把、提燈與兩件劇情物永遠不會掉。
func TestLootItemType_ExcludesPlotItems(t *testing.T) {
	r := rng.NewWithSeed(23)
	seen := map[int]bool{}
	for n := 0; n < 5000; n++ {
		v := LootItemType(r)
		if v < 0 || v > 25 {
			t.Fatalf("掉落型別 %d 超出 0–25", v)
		}
		seen[v] = true
	}
	if len(seen) != lootTypeCount {
		t.Errorf("只出現 %d 種型別，預期 %d 種都會出現", len(seen), lootTypeCount)
	}
	for _, plot := range []int{26, 27, 28, 29} {
		if seen[plot] {
			t.Errorf("劇情物／雜貨型別 %d 不該掉出來", plot)
		}
	}
}

// 掉落機率 = 怪物等級 × 6 + 5，逐隻獨立判定。
func TestBattleDropChance(t *testing.T) {
	for _, c := range []struct{ level, want int }{
		{1, 11}, {5, 35}, {10, 65}, {16, 101},
	} {
		if got := BattleDropChance(c.level); got != c.want {
			t.Errorf("%d 級怪的掉落機率 %d%%，預期 %d%%", c.level, got, c.want)
		}
	}
}

// 一群高等怪幾乎一定有掉落，一群 1 級怪大多沒有 —— 逐隻判定的直接後果。
func TestRollBattleDrops_ScalesWithLevel(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(29)

	low, high := 0, 0
	for n := 0; n < 300; n++ {
		low += len(RollBattleDrops(r, tb, items, []int{1, 1, 1}))
		high += len(RollBattleDrops(r, tb, items, []int{15, 15, 15}))
	}
	if high <= low*3 {
		t.Errorf("1 級怪掉 %d 件、15 級怪掉 %d 件 —— 差距應該大得多", low, high)
	}
	// 16 級以上機率破百，一定每隻都掉。
	if got := len(RollBattleDrops(r, tb, items, []int{20, 20})); got != 2 {
		t.Errorf("20 級怪兩隻掉了 %d 件，機率破百應該每隻都掉", got)
	}
}

// 材質類別：六個豁免型別永遠是 1，其餘落在 1–8。
func TestLootMaterialClass(t *testing.T) {
	r := rng.NewWithSeed(101)

	// 1-based 的豁免清單換回 0-based：布甲、皮甲、藥瓶、寶石、藥膏、藥瓶。
	for _, typ := range []int{8, 9, 14, 19, 24, 25} {
		for n := 0; n < 200; n++ {
			if c := LootMaterialClass(r, typ, 12); c != 1 {
				t.Fatalf("型別 %d 應該永遠是類別 1，得到 %d", typ, c)
			}
		}
	}

	// 其餘型別：值域 1–8，而且高等級擲得出高類別。
	seen := map[int]bool{}
	for n := 0; n < 20000; n++ {
		c := LootMaterialClass(r, 0, 10)
		if c < 1 || c > MaterialClassCount-1 {
			t.Fatalf("類別 %d 超出 1–%d", c, MaterialClassCount-1)
		}
		seen[c] = true
	}
	for c := 1; c <= 8; c++ {
		if !seen[c] {
			t.Errorf("10 級擲了兩萬次都沒出現類別 %d", c)
		}
	}

	// 低等級擲不出貴材質：等級 1 時 n 最大 11，只可能是類別 1 或 2。
	for n := 0; n < 2000; n++ {
		if c := LootMaterialClass(r, 0, 1); c > 2 {
			t.Fatalf("1 級擲出類別 %d，公式最多只到 2", c)
		}
	}
}

// 價格上限：2.6^等級 + 25×等級。低等怪掉不出貴東西。
func TestLootPriceCap(t *testing.T) {
	// 匕首 2、雙手劍 100、寶石 500（ITEMS.DAT 的底價）。
	cases := []struct {
		level, atLeast, below int
	}{
		{1, 2, 100},   // 1 級：買得起匕首，買不起雙手劍
		{4, 100, 500}, // 4 級：雙手劍可以，寶石還不行
		{7, 500, 0},   // 7 級：全部解鎖
	}
	for _, c := range cases {
		got := LootPriceCap(c.level)
		if got < c.atLeast {
			t.Errorf("%d 級的上限 %d，應該至少蓋得住 %d", c.level, got, c.atLeast)
		}
		if c.below != 0 && got >= c.below {
			t.Errorf("%d 級的上限 %d，不該蓋到 %d", c.level, got, c.below)
		}
	}
}

// 型別篩選：低等級只挑得出便宜貨。
func TestLootItemTypeFor_RespectsCap(t *testing.T) {
	items := loadItems(t)
	r := rng.NewWithSeed(103)
	cap1 := LootPriceCap(1)
	for n := 0; n < 2000; n++ {
		typ := LootItemTypeFor(r, items, 1)
		it, err := items.ByIndex(typ)
		if err != nil {
			t.Fatal(err)
		}
		if it.Price > cap1 {
			t.Fatalf("1 級挑到 %s（底價 %d），上限是 %d", it.Name, it.Price, cap1)
		}
	}
}

// 附魔的三段式：武器全額、護甲減半、飾品以上沒有。
func TestLootEnchant_ThreeTiers(t *testing.T) {
	r := rng.NewWithSeed(107)
	for n := 0; n < 3000; n++ {
		// 王冠（13）以上完全沒有附魔。
		for _, typ := range []int{13, 15, 19, 25} {
			if v := LootEnchant(r, 12, typ); v != 0 {
				t.Fatalf("型別 %d 不該有附魔，得到 +%d", typ, v)
			}
		}
	}
	// 布甲（8）屬於護甲，要減半 —— 這正是偏一格的那一格。
	maxCloth, maxSword := 0, 0
	for n := 0; n < 5000; n++ {
		if v := LootEnchant(r, 12, 8); v > maxCloth {
			maxCloth = v
		}
		if v := LootEnchant(r, 12, 7); v > maxSword {
			maxSword = v
		}
	}
	if maxCloth >= maxSword {
		t.Errorf("布甲最高 +%d、雙手劍 +%d —— 布甲是護甲，應該被減半",
			maxCloth, maxSword)
	}
}

// 詛咒只出現在武器與護甲上。
func TestGenerateDrop_CurseOnlyOnGear(t *testing.T) {
	tb, items := loadTables(t), loadItems(t)
	r := rng.NewWithSeed(109)
	for _, typ := range []int{13, 15, 19, 22} {
		it, _ := items.ByIndex(typ)
		for n := 0; n < 500; n++ {
			slot := GenerateDrop(r, tb, it, typ, 12)
			if slot.Enchant < 0 || slot.ExorciseResist != 0 {
				t.Fatalf("型別 %d 不該被詛咒：%+v", typ, slot)
			}
		}
	}
}
