package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 七個形容詞蓋十種規模，2/3、6/7、8/9 各共用一個。
func TestMerchantAdjective(t *testing.T) {
	for _, c := range []struct {
		size int
		want string
	}{
		{0, "ragged looking"},
		{1, "poor looking"},
		{2, "travelling"}, {3, "travelling"},
		{4, "adventuring"},
		{5, "well dressed"},
		{6, "upper class"}, {7, "upper class"},
		{8, "wealthy"}, {9, "wealthy"},
		// 超出範圍鉗住，不 panic。
		{-5, "ragged looking"}, {99, "wealthy"},
	} {
		if got := MerchantAdjective(c.size); got != c.want {
			t.Errorf("規模 %d → %q，預期 %q", c.size, got, c.want)
		}
	}
}

// 詛咒機率 = rnd(120) − 80，負的鉗成 0。
//
// 值域 0–40 意味著**最壞的商隊也只有四成的貨是詛咒品**，
// 而且超過一半的商隊完全乾淨（rnd 擲到 80 以下就是 0）。
func TestMerchantCurseChance_Range(t *testing.T) {
	r := rng.NewWithSeed(31)
	clean, max := 0, 0
	const n = 4000
	for i := 0; i < n; i++ {
		c := MerchantCurseChance(r)
		if c < 0 || c > merchantCurseDie-merchantCurseOffset {
			t.Fatalf("詛咒機率 %d 超出 0–%d", c, merchantCurseDie-merchantCurseOffset)
		}
		if c == 0 {
			clean++
		}
		if c > max {
			max = c
		}
	}
	if clean*3 < n {
		t.Errorf("完全乾淨的商隊只有 %d/%d —— 應該超過一半", clean, n)
	}
	if max < 35 {
		t.Errorf("最高詛咒機率只到 %d%%，應該接近 40%%", max)
	}
}

// 一支商隊帶 7–10 件貨。
func TestMerchantWareCount(t *testing.T) {
	r := rng.NewWithSeed(37)
	seen := map[int]bool{}
	for i := 0; i < 2000; i++ {
		n := MerchantWareCount(r)
		if n < 7 || n > 10 {
			t.Fatalf("件數 %d 超出 7–10", n)
		}
		seen[n] = true
	}
	if len(seen) != 4 {
		t.Errorf("只出現 %d 種件數，7–10 四種都該出現", len(seen))
	}
}

// 商隊的貨全部未鑑定；詛咒品附魔為負且沒有效果。
func TestRollMerchant_WaresAreUnidentified(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(41)

	cursed, withEffect := 0, 0
	for n := 0; n < 400; n++ {
		m := RollMerchant(r, tb, items, 6, 14)
		if len(m.Wares) < 7 || len(m.Wares) > 10 {
			t.Fatalf("帶了 %d 件貨", len(m.Wares))
		}
		for _, ware := range m.Wares {
			w := ware.Item
			if !w.Identified {
				t.Fatal("列出來的貨應該標成已鑑定（docs/re/44 §2）")
			}
			if ware.Price <= 0 {
				t.Fatalf("開價 %d 不合理：%+v", ware.Price, w)
			}
			if w.Enchant < 0 {
				cursed++
				if w.Power != 0 || w.Effect != 0 {
					t.Fatalf("詛咒品不該有效果：%+v", w)
				}
			}
			if w.Power != 0 {
				withEffect++
			}
		}
	}
	if cursed == 0 {
		t.Error("400 支商隊一件詛咒品都沒有，詛咒那條沒生效")
	}
	if withEffect == 0 {
		t.Error("400 支商隊一件有效果的都沒有，生成那條沒生效")
	}
}

// 詛咒機率 0 的商隊一件詛咒品都不該有。
func TestRollMerchant_CleanCaravanHasNoCurse(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(43)

	for n := 0; n < 500; n++ {
		m := RollMerchant(r, tb, items, 3, 10)
		if m.CurseChance != 0 {
			continue
		}
		for _, ware := range m.Wares {
			if ware.Item.Enchant < 0 {
				t.Fatalf("詛咒機率 0 的商隊卻有詛咒品：%+v", ware.Item)
			}
		}
	}
}

// 買一件貨：扣錢、進包包、標成已賣。
func TestBuyFromMerchant_Success(t *testing.T) {
	tb, items := loadTables(t), loadItems(t)
	m := RollMerchant(rng.NewWithSeed(11), tb, items, 5, 8)

	i := -1
	for n, w := range m.Wares {
		if w.PriceExact {
			i = n
			break
		}
	}
	if i < 0 {
		t.Skip("這一支商隊沒有算得出價錢的貨")
	}
	want := m.Wares[i].Item
	price := m.Wares[i].Price

	members := []Character{*campChar("買家")}
	res := BuyFromMerchant(&m, i, members, price+100)
	if !res.OK {
		t.Fatalf("應該買得成：%s", res.Reason)
	}
	if res.Gold != 100 {
		t.Errorf("剩 %d 金幣，預期 100", res.Gold)
	}
	if members[0].Inventory[res.Slot] != want {
		t.Errorf("拿到 %+v，預期 %+v", members[0].Inventory[res.Slot], want)
	}
	if !m.Wares[i].Sold {
		t.Error("那一件應該標成已賣")
	}
	if res := BuyFromMerchant(&m, i, members, 10000); res.OK {
		t.Error("同一件不該買得到第二次")
	}
}

func TestBuyFromMerchant_Refusals(t *testing.T) {
	full := *campChar("滿", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	base := MerchantWare{Item: scenario.InventorySlot{Type: 3}, Price: 50, PriceExact: true}

	cases := []struct {
		name   string
		ware   MerchantWare
		gold   int
		member Character
		reason string
	}{
		{"金幣不夠", base, 10, *campChar("窮"), "金幣不夠"},
		{"算不出價錢", MerchantWare{Item: base.Item, Price: 50}, 999, *campChar("A"),
			"商人說不出這件的價錢"},
		{"已經賣掉", MerchantWare{Item: base.Item, Price: 50, PriceExact: true, Sold: true},
			999, *campChar("A"), "這件已經賣掉了"},
		{"包包滿了", base, 999, full, "全隊的道具欄都滿了"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Merchant{Wares: []MerchantWare{tc.ware}}
			members := []Character{tc.member}
			res := BuyFromMerchant(&m, 0, members, tc.gold)
			if res.OK {
				t.Fatal("預期擋下來")
			}
			if res.Reason != tc.reason {
				t.Errorf("理由 %q，預期 %q", res.Reason, tc.reason)
			}
		})
	}
	if res := BuyFromMerchant(nil, 0, nil, 100); res.OK {
		t.Error("nil 商隊不該買得到東西")
	}
}
