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

	total, used := LootCharges(r, ChargeUnlimitedUses, 5, 17, 10)
	if used != restNeverRecharge {
		t.Errorf("種類 1 的已用次數 %d，預期 %d（過夜不充能的哨兵）",
			used, restNeverRecharge)
	}
	if total < 1 {
		t.Errorf("種類 1 的上限 %d，至少要 1", total)
	}

	for n := 0; n < 200; n++ {
		total, used = LootCharges(r, ChargeManyUses, 5, 20, 10)
		if total < restRechargeMaxTotal {
			t.Fatalf("種類 2 的上限 %d，應該 >= %d（過夜不充能的另一個條件）",
				total, restRechargeMaxTotal)
		}
		if used != 0 {
			t.Fatalf("種類 2 的已用次數應該是 0，得到 %d", used)
		}
	}

	// 戒指與火把固定 200。
	if total, _ := LootCharges(r, ChargeManyUses, 5, itemTypeRing, 10); total != chargeManyFixed {
		t.Errorf("戒指的次數 %d，預期 %d", total, chargeManyFixed)
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
		slot := GenerateLoot(r, tb, it, typ, 12, false)

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
func TestGenerateLoot_CursedHasNoEffect(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(17)

	negative := 0
	for n := 0; n < 500; n++ {
		it, _ := items.ByIndex(0)
		slot := GenerateLoot(r, tb, it, 0, 12, true)
		if slot.Power != 0 || slot.Effect != 0 {
			t.Fatalf("被詛咒的道具長出了效果：%+v", slot)
		}
		if slot.Enchant < 0 {
			negative++
		}
		if slot.Enchant > 0 {
			t.Fatalf("被詛咒的附魔是正的：%d", slot.Enchant)
		}
	}
	if negative == 0 {
		t.Error("500 次都沒擲出負附魔，詛咒那條沒生效")
	}
}

// 掉落的道具是**未鑑定**的 —— 鑑定是市集另外收錢的服務。
func TestGenerateLoot_NotIdentified(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(19)
	it, _ := items.ByIndex(5)
	for n := 0; n < 100; n++ {
		if GenerateLoot(r, tb, it, 5, 8, false).Identified {
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
