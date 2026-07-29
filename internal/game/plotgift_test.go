package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func giftSave() *scenario.SaveGame { return &scenario.SaveGame{} }

func TestBlacksmithGiftGoesIntoAFreeSlot(t *testing.T) {
	s, c := giftSave(), taker()
	res := TakePlotGift(s, c, PlotGiftBlacksmith)
	if !res.OK || res.Slot != 0 {
		t.Fatalf("拿不到：%+v", res)
	}
	got := c.Inventory[0]
	// 逐欄位對原版（`0x1ab33`–`0x1ab61`）。
	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"型別", int(got.Type), 0x05},
		{"附帶法術 A", got.SpellA, 0x94},
		{"附帶法術 A 強度", got.SpellAPower, 0x04},
		{"驅邪成功率", got.ExorciseResist, 0x14},
		{"附魔", got.Enchant, -1},
		{"材質類別", got.MaterialClass, 0x08},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d，預期 %d", tc.name, tc.got, tc.want)
		}
	}
	if !got.Identified {
		t.Error("劇情送的道具應該是已鑑定的")
	}
}

// **一輪遊戲只有一次。** 旗標在 trailer `+0xb3 + 4` ＝ `+0xb7`。
func TestBlacksmithGiftIsOneShot(t *testing.T) {
	s, c := giftSave(), taker()
	TakePlotGift(s, c, PlotGiftBlacksmith)
	if !PlotGiftTaken(s, PlotGiftBlacksmith) {
		t.Fatal("拿過了卻沒記旗標")
	}
	if res := TakePlotGift(s, taker(), PlotGiftBlacksmith); res.OK {
		t.Error("同一件劇情道具拿了兩次")
	}
}

// **道具欄滿了就整件事不算** —— 旗標不能先記，不然那件道具永遠拿不到了。
func TestBlacksmithGiftDoesNotBurnTheFlagWhenFull(t *testing.T) {
	s, c := giftSave(), taker()
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: 1}
	}
	res := TakePlotGift(s, c, PlotGiftBlacksmith)
	if res.OK || !res.Full {
		t.Fatalf("欄位滿了卻回 %+v", res)
	}
	if PlotGiftTaken(s, PlotGiftBlacksmith) {
		t.Error("放不下卻把旗標燒掉了 —— 那件道具永遠拿不到了")
	}
}

// 沒讀出來的 id 一律不給（寧可少給，不要憑空生道具）。
//
// **param 0–5 都解出來了**（`docs/re/99`、`101`），只剩 6 不在表內 ——
// 而 6 是刻意不收：它的旗標落在 `+0xb9`（劇情階段），要由 case 8
// 直接推進階段，不是多一格旗標（`docs/re/101` §3.2）。
func TestUnknownPlotGiftGivesNothing(t *testing.T) {
	s, c := giftSave(), taker()
	for _, id := range []PlotGiftID{6, -1, 99} {
		if res := TakePlotGift(s, c, id); res.OK {
			t.Errorf("未讀出的 id %d 卻發了道具", id)
		}
	}
}

// 兵器庫四座台座：座標 → 編號 → 道具規格，逐格釘死。
//
// 這張表的每一列都有**兩個獨立來源**互相印證（`docs/re/99` §3）：
// 原版寫死的位元組，與 `docs/walkthrough/part-4.md` §3.2 列的四樣道具。
func TestArmoryGifts(t *testing.T) {
	for _, c := range []struct {
		x, y     int
		want     PlotGiftID
		typ      byte
		material int
		enchant  int
		label    string
	}{
		{23, 31, PlotGiftArmoryChain, 0x0a, 3, 0, "銀鏈甲（型別 10 ＝ chain、材質 3 ＝ Silver）"},
		{23, 27, PlotGiftArmoryMace, 0x03, 0, 0, "釘頭鎚（型別 3 ＝ mace、無材質前綴）"},
		{33, 27, PlotGiftArmoryDagger, 0x00, 5, 1, "水晶匕首 +1（型別 0 ＝ dagger、材質 5 ＝ Crystal）"},
		{33, 31, PlotGiftArmorySword, 0x02, 0, 0, "冰藍短劍（型別 2 ＝ short sword）"},
	} {
		id := ArmoryGiftFor(c.x, c.y)
		if id != c.want {
			t.Errorf("(%d,%d) → 編號 %d，預期 %d", c.x, c.y, id, c.want)
			continue
		}
		spec, ok := plotGiftSpec(id)
		if !ok {
			t.Errorf("%s：編號 %d 沒有規格", c.label, id)
			continue
		}
		if spec.Type != c.typ || spec.MaterialClass != c.material || spec.Enchant != c.enchant {
			t.Errorf("%s：型別 %#02x 材質 %d 附魔 %d，預期 %#02x/%d/%d",
				c.label, spec.Type, spec.MaterialClass, spec.Enchant,
				c.typ, c.material, c.enchant)
		}
		if !spec.Identified {
			t.Errorf("%s：共用前置段寫 +0x10 = 1，應該是已鑑定", c.label)
		}
	}

	// 冰藍短劍的「冰藍」是附帶法術 15（寒顫，Ice 系）不是材質前綴。
	sword, _ := plotGiftSpec(PlotGiftArmorySword)
	if sword.SpellA != 0x0f || sword.SpellAPower != 4 {
		t.Errorf("冰藍短劍的附帶法術 %d 強度 %d，預期 15／4",
			sword.SpellA, sword.SpellAPower)
	}
	// 釘頭鎚的常駐效果是速度 +2；0x12 是類型，0x0c − 10 是數值。
	mace, _ := plotGiftSpec(PlotGiftArmoryMace)
	if mace.EffectTypeA != scenario.EquipmentEffectSpeed || mace.EffectValueAByte != 0x0c {
		t.Errorf("釘頭鎚的特效位元組 %#02x/%#02x，預期 0x12／0x0c",
			mace.EffectTypeA, mace.EffectValueAByte)
	}
	if got := mace.EquipmentBonus(scenario.EquipmentEffectSpeed); got != 2 {
		t.Errorf("釘頭鎚速度加成 = %d，預期 2", got)
	}

	// 四座各自獨立：拿了一座不影響其他三座。
	s, c := giftSave(), taker()
	if res := TakePlotGift(s, c, PlotGiftArmoryDagger); !res.OK {
		t.Fatalf("拿匕首失敗：%+v", res)
	}
	for _, other := range []PlotGiftID{
		PlotGiftArmoryChain, PlotGiftArmoryMace, PlotGiftArmorySword,
	} {
		if PlotGiftTaken(s, other) {
			t.Errorf("拿了匕首卻把編號 %d 也標成拿過了", other)
		}
	}
}

// 旗標要進存檔，不然關掉重開又能拿一次。
func TestPlotGiftFlagsRoundTrip(t *testing.T) {
	s := giftSave()
	s.PlotGifts[PlotGiftBlacksmith] = 1
	if len(s.PlotGifts) != scenario.PlotGiftCount {
		t.Fatalf("旗標陣列 %d 格，預期 %d", len(s.PlotGifts), scenario.PlotGiftCount)
	}
	if !PlotGiftTaken(s, PlotGiftBlacksmith) {
		t.Error("旗標設了卻說沒拿過")
	}
}

// 惡魔水晶（case 7 ＝ 跳表 param 5）：型別 ＝ `param + 0x17` ＝ 28，
// 而 `ITEMS.DAT` 第 28 件就是 `Demon Crystal`（`docs/re/101` §4）。
func TestDemonCrystalSpec(t *testing.T) {
	spec, ok := plotGiftSpec(PlotGiftDemonCrystal)
	if !ok {
		t.Fatal("惡魔水晶沒有規格")
	}
	if spec.Type != 28 {
		t.Errorf("型別 %d，預期 28（0x17 + param 5）", spec.Type)
	}
	if !spec.Identified {
		t.Error("共用前置段寫 +0x10 = 1，應該是已鑑定")
	}
	// 劇情道具不是裝備：沒有附帶法術、沒有效果、沒有材質。
	if spec.SpellA != 0 || spec.Power != 0 || spec.MaterialClass != 0 {
		t.Errorf("惡魔水晶不該有效果欄位：%+v", spec)
	}
	// 旗標是 `+0xb3 + 5` ＝ `+0xb8`，而且落在 PlotGiftCount 之內。
	if int(PlotGiftDemonCrystal) >= scenario.PlotGiftCount {
		t.Errorf("編號 %d 超出旗標陣列 %d 格",
			PlotGiftDemonCrystal, scenario.PlotGiftCount)
	}
}
