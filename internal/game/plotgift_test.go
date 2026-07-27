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
func TestUnknownPlotGiftGivesNothing(t *testing.T) {
	s, c := giftSave(), taker()
	for _, id := range []PlotGiftID{0, 1, 2, 3, 5, -1, 99} {
		if res := TakePlotGift(s, c, id); res.OK {
			t.Errorf("未讀出的 id %d 卻發了道具", id)
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
