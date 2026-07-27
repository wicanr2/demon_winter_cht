package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// walkthroughEnchant 是 `docs/walkthrough/part-5.md` §4.1 那張表：
// 一把**底價 0 的武器**從 +0 一路附魔到 +10，八種材質各一列的逐級費用。
//
// 底價 0 是刻意的 —— 攻略那張表只算附魔那一項的差額，
// 底價乘材質倍率的部分在新舊兩次估價裡會相減掉。
var walkthroughEnchant = map[int][10]int{
	1: {665, 1495, 2143, 2715, 3239, 3725, 4191, 4630, 5055, 5466},
	2: {630, 1416, 2030, 2572, 3069, 3529, 3970, 4386, 4789, 5178},
	3: {595, 1338, 1917, 2429, 2898, 3333, 3750, 4143, 4523, 4891},
	4: {560, 1259, 1805, 2286, 2728, 3137, 3529, 3899, 4257, 4603},
	5: {525, 1180, 1692, 2144, 2557, 2941, 3309, 3656, 3991, 4316},
	6: {490, 1102, 1579, 2000, 2387, 2745, 3088, 3411, 3725, 4027},
	7: {455, 1023, 1466, 1858, 2216, 2549, 2868, 3168, 3459, 3740},
	8: {420, 944, 1353, 1715, 2046, 2353, 2647, 2924, 3193, 3452},
}

// 逐級費用對攻略那 80 個點。
//
// **允許 ±1**，理由有兩層：
//
//   - 攻略自己那張表就不自洽 —— 原文的「總計」（逐段加總）與「0→10」
//     （整體算）差 2～3，作者自己標成捨入誤差。
//   - 原版的 pow 是 1988 年手寫的 `exp(y·ln x)`，末位與 Go 的
//     `math.Pow` 不會逐位相同。
//
// 實測 80 個點裡 **69 個完全相同、11 個差 1（而且全部同一個方向）**。
// 差 1 全同向本身就是「同一條式子、末位捨入不同」的指紋 ——
// 如果公式錯了，誤差不會這麼整齊。
//
// 對照組：把 1.7 讀成係數（舊版的 `1.7 × n × 350`）的話，
// 第一級會算出 1130 而不是 665，**差七成**。
func TestEnchantCostMatchesWalkthrough(t *testing.T) {
	exact, off1 := 0, 0
	for material, want := range walkthroughEnchant {
		for i, w := range want {
			slot := scenario.InventorySlot{
				Type: 3, MaterialClass: material, Enchant: i, Identified: true,
			}
			got := EnchantCost(0, slot, i+1)
			switch d := got - w; {
			case d == 0:
				exact++
			case d == -1 || d == 1:
				off1++
			default:
				t.Errorf("材質 %d、+%d→+%d：算出 %d，攻略 %d（差 %d）",
					material, i, i+1, got, w, d)
			}
		}
	}
	if exact+off1 != 80 {
		t.Fatalf("只對到 %d 個點（完全相同 %d、差 1 %d）", exact+off1, exact, off1)
	}
	if exact < 60 {
		t.Errorf("完全相同只有 %d 個 —— 少於 60 就該懷疑公式而不是捨入", exact)
	}
	t.Logf("80 個點：完全相同 %d、差 1 %d", exact, off1)
}

// 把 1.7 當係數的舊讀法會差七成 —— 這一條把「不是捨入問題」釘死。
func TestEnchantCostRulesOutTheCoefficientReading(t *testing.T) {
	slot := scenario.InventorySlot{Type: 3, MaterialClass: 1, Enchant: 0, Identified: true}
	got := EnchantCost(0, slot, 1)
	if got != 665 {
		t.Errorf("一般材質 +0→+1 算出 %d，攻略是 665", got)
	}
	// 係數版：trunc(1.7×1×350) = 595 → 595×19/10 = 1130。
	if got == 1130 {
		t.Error("算出 1130 —— 那是把 1.7 當係數的結果")
	}
}

// 三道閘門：只有已鑑定的武器護甲能附魔，上限 +10，不能往下調。
func TestEnchantGates(t *testing.T) {
	weapon := scenario.InventorySlot{Type: 3, MaterialClass: 1, Identified: true}
	if !Enchantable(weapon) {
		t.Error("已鑑定的武器應該可以附魔")
	}
	unidentified := weapon
	unidentified.Identified = false
	if Enchantable(unidentified) {
		t.Error("沒鑑定的不該能附魔（原版：Only identified items may be enchanted）")
	}
	gem := scenario.InventorySlot{Type: 19, Identified: true}
	if Enchantable(gem) {
		t.Error("寶石不是武器護甲，不該能附魔")
	}
	if Enchantable(scenario.InventorySlot{Type: scenario.SlotEmpty}) {
		t.Error("空槽不該能附魔")
	}

	if got := EnchantCost(0, weapon, EnchantMax+1); got != 0 {
		t.Errorf("超過上限應該回 0，得到 %d", got)
	}
	at5 := weapon
	at5.Enchant = 5
	if got := EnchantCost(0, at5, 5); got != 0 {
		t.Errorf("已經是 +5 了還報價 %d（原版印 It is already +5）", got)
	}
	if got := EnchantCost(0, at5, 3); got != 0 {
		t.Errorf("往下調應該回 0，得到 %d", got)
	}

	// 護甲的係數加倍，所以同一級同材質貴一倍上下。
	armour := scenario.InventorySlot{Type: 10, MaterialClass: 1, Identified: true}
	if EnchantCost(0, armour, 1) <= EnchantCost(0, weapon, 1) {
		t.Error("護甲的附魔應該比武器貴（係數 700 vs 350）")
	}
}

// 材質折扣是線性的每級 1/19，攻略說「大約每高一級折扣 5%」。
func TestEnchantMaterialDiscountIsLinear(t *testing.T) {
	var prev int
	for c := 1; c <= 8; c++ {
		slot := scenario.InventorySlot{
			Type: 3, MaterialClass: c, Enchant: 0, Identified: true,
		}
		got := EnchantCost(0, slot, 1)
		want := 350 * (20 - c) / 10
		if got != want {
			t.Errorf("材質 %d 的 +0→+1 是 %d，預期 %d", c, got, want)
		}
		if prev != 0 && prev-got != 35 {
			t.Errorf("材質 %d 比 %d 只便宜 %d，預期每級固定 35", c, c-1, prev-got)
		}
		prev = got
	}
}
