package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func TestMaterialMultiplier_Table(t *testing.T) {
	want := []int{0, 1, 2, 5, 20, 35, 50, 60, 75}
	for i, w := range want {
		if got := MaterialMultiplier(i); got != w {
			t.Errorf("類別 %d 的倍率 %d，預期 %d", i, got, w)
		}
	}
	if MaterialMultiplier(-1) != 0 || MaterialMultiplier(MaterialClassCount) != 0 {
		t.Error("越界的類別應該回 0")
	}
}

// 起始存檔的裝備都是類別 1（×1）—— 估價就等於底價。
func TestItemValueBase_PlainGearIsBasePrice(t *testing.T) {
	slot := scenario.InventorySlot{Type: 3, MaterialClass: 1}
	if got := ItemValueBase(30, slot); got != 30 {
		t.Errorf("平凡裝備估價 %d，預期 30", got)
	}
}

func TestItemValueBase_ScalesWithClass(t *testing.T) {
	cases := map[int]int{0: 0, 1: 30, 3: 150, 8: 2250}
	for class, want := range cases {
		slot := scenario.InventorySlot{Type: 3, MaterialClass: class}
		if got := ItemValueBase(30, slot); got != want {
			t.Errorf("類別 %d 的估價 %d，預期 %d", class, got, want)
		}
	}
}

// 原版存檔裡每一件實物的材質類別都是 1 —— 這條同時釘住讀檔路徑。
func TestItemValueBase_RealSaveIsAllClassOne(t *testing.T) {
	save := loadParty(t)
	seen := 0
	for i := range save.Characters {
		for _, it := range save.Characters[i].Inventory {
			if it.Empty() {
				continue
			}
			seen++
			if it.MaterialClass != 1 {
				t.Errorf("%s 身上有材質類別 %d 的東西（型別 %d）",
					save.Characters[i].Name, it.MaterialClass, it.Type)
			}
		}
	}
	if seen == 0 {
		t.Fatal("原版存檔應該有裝備")
	}
}

// 已鑑定的加價 = (+0x02 + +0x04) × 270。225 × 1.2 剛好是整數。
func TestItemValue_IdentifiedBonus(t *testing.T) {
	slot := scenario.InventorySlot{
		Type: 3, MaterialClass: 1, Identified: true,
		SpellAPower: 2, SpellBPower: 3,
	}
	got, exact := ItemValue(30, slot)
	if !exact {
		t.Error("強度 0 的道具應該算得出確切售價")
	}
	if want := 30 + 5*270; got != want {
		t.Errorf("估價 %d，預期 %d", got, want)
	}
}

// 未鑑定就沒有第二項。
func TestItemValue_UnidentifiedSkipsBonus(t *testing.T) {
	slot := scenario.InventorySlot{Type: 3, MaterialClass: 1, SpellAPower: 2, SpellBPower: 3}
	if got, _ := ItemValue(30, slot); got != 30 {
		t.Errorf("未鑑定的估價 %d，預期 30", got)
	}
}

// 強度加價的三條分支。
func TestItemValue_ChargeBonusBranches(t *testing.T) {
	base := func(power, total, used int) scenario.InventorySlot {
		return scenario.InventorySlot{
			Type: 3, MaterialClass: 1, Power: power, Total: total, Used: used,
		}
	}
	cases := []struct {
		name string
		slot scenario.InventorySlot
		want int
	}{
		{"無限次數", base(4, 10, 0xff), int(float64(5*4*4*10) * 0.9)},
		{"上限 > 100", base(4, 150, 0), 500 * 4 * 4 / 50},
		{"一般次數", base(4, 10, 0), 10 * 4 * 214},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := ItemValue(30, tc.slot)
			if want := 30 + tc.want; got != want {
				t.Errorf("估價 %d，預期 %d", got, want)
			}
		})
	}
}

// 上限剛好 101 時除數是 1 —— 分界正好把除以 0 擋在外面。
func TestItemValue_ManyChargesBoundary(t *testing.T) {
	at := func(total int) int {
		v, _ := ItemValue(0, scenario.InventorySlot{
			Type: 3, MaterialClass: 1, Power: 2, Total: total,
		})
		return v
	}
	if got, want := at(101), 500*2*2/1; got != want {
		t.Errorf("上限 101 的加價 %d，預期 %d", got, want)
	}
	if got, want := at(100), 100*2*214; got != want {
		t.Errorf("上限 100 走的是另一條，加價 %d，預期 %d", got, want)
	}
}

// 第四項還沒解 —— +0x0a 不為 0 就不能說算得準。
func TestItemValue_EffectAMakesItInexact(t *testing.T) {
	slot := scenario.InventorySlot{Type: 3, MaterialClass: 1, EffectAByte: 12}
	if _, exact := ItemValue(30, slot); exact {
		t.Error("+0x0a 不為 0 還缺第四項，不該說算得準")
	}
	slot.EffectAByte = 0
	if _, exact := ItemValue(30, slot); !exact {
		t.Error("+0x0a 為 0 時前三項就是全部，應該算得準")
	}
}

// 商隊開價落在估價的 [0.6, 1.4) 之間 —— ±40% 的浮動。
func TestMerchantPrice_StaysWithinSpread(t *testing.T) {
	r := rng.NewWithSeed(7)
	const value = 1000
	lo, hi := value, 0
	for i := 0; i < 5000; i++ {
		p := MerchantPrice(r, value)
		if p < int(float64(value)*0.6) || p >= int(float64(value)*1.4) {
			t.Fatalf("開價 %d 超出 [0.6, 1.4) × %d", p, value)
		}
		if p < lo {
			lo = p
		}
		if p > hi {
			hi = p
		}
	}
	// 五千次應該把兩端都掃到 —— 否則係數的算法八成錯了。
	if lo > 610 || hi < 1390 {
		t.Errorf("五千次的範圍只有 %d–%d，預期接近 600–1399", lo, hi)
	}
}

// 估價 0 的東西開價也是 0，不會因為浮動變出錢來。
func TestMerchantPrice_ZeroStaysZero(t *testing.T) {
	r := rng.NewWithSeed(1)
	for i := 0; i < 100; i++ {
		if p := MerchantPrice(r, 0); p != 0 {
			t.Fatalf("估價 0 卻開價 %d", p)
		}
	}
}
