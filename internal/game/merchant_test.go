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

// 「跳過第一道效果門檻」的機率 = rnd(120) − 80，負的鉗成 0。
//
// 值域 0–40，而且**超過一半的商隊擲到 0**（rnd 擲到 80 以下就歸零）。
// 這個參數以前被讀成詛咒機率，見 `docs/re/48` §5。
func TestMerchantEffectChance_Range(t *testing.T) {
	r := rng.NewWithSeed(31)
	clean, max := 0, 0
	const n = 4000
	for i := 0; i < n; i++ {
		c := MerchantEffectChance(r)
		if c < 0 || c > merchantEffectDie-merchantEffectOffset {
			t.Fatalf("機率 %d 超出 0–%d", c, merchantEffectDie-merchantEffectOffset)
		}
		if c == 0 {
			clean++
		}
		if c > max {
			max = c
		}
	}
	if clean*3 < n {
		t.Errorf("擲到 0 的商隊只有 %d/%d —— 應該超過一半", clean, n)
	}
	if max < 35 {
		t.Errorf("最高只到 %d%%，應該接近 40%%", max)
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

	guaranteed, withEffect := 0, 0
	for n := 0; n < 400; n++ {
		m := RollMerchant(r, tb, items, 9)
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
				t.Fatalf("商隊的貨不該有負附魔：%+v", w)
			}
			if ware.Guaranteed {
				guaranteed++
			}
			if w.Power != 0 {
				withEffect++
			}
		}
	}
	if guaranteed == 0 {
		t.Error("400 支商隊一件「保證帶效果」都沒擲到")
	}
	if withEffect == 0 {
		t.Error("400 支商隊一件有效果的都沒有，生成那條沒生效")
	}
}

// **商隊完全不會生出詛咒品。** 生成器的詛咒判定只在掉落模式跑，
// 商隊那條路連負附魔與驅邪成功率都不會出現（`docs/re/48` §5）。
func TestRollMerchant_NeverCursed(t *testing.T) {
	tb := loadTables(t)
	items := loadItems(t)
	r := rng.NewWithSeed(43)

	for n := 0; n < 500; n++ {
		m := RollMerchant(r, tb, items, 3)
		for _, ware := range m.Wares {
			if ware.Item.Enchant < 0 {
				t.Fatalf("商隊的貨出現負附魔：%+v", ware.Item)
			}
			if ware.Item.ExorciseResist != 0 {
				t.Fatalf("商隊的貨被寫了驅邪成功率：%+v", ware.Item)
			}
		}
	}
}

// 買一件貨：扣錢、進包包、標成已賣。
func TestBuyFromMerchant_Success(t *testing.T) {
	tb, items := loadTables(t), loadItems(t)
	m := RollMerchant(rng.NewWithSeed(11), tb, items, 5)

	const i = 0
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
	base := MerchantWare{Item: scenario.InventorySlot{Type: 3}, Price: 50}

	cases := []struct {
		name   string
		ware   MerchantWare
		gold   int
		member Character
		reason string
	}{
		{"金幣不夠", base, 10, *campChar("窮"), "金幣不夠"},
		{"已經賣掉", MerchantWare{Item: base.Item, Price: 50, Sold: true},
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

// 規模 = clamp(基準 + rnd(3) − 2, ≤ 9)，而且規模就是等級。
func TestMerchantSize(t *testing.T) {
	r := rng.NewWithSeed(53)
	seen := map[int]bool{}
	for n := 0; n < 3000; n++ {
		got := MerchantSize(r, 4)
		if got < 3 || got > 5 {
			t.Fatalf("基準 4 擲出 %d，應該落在 3–5", got)
		}
		seen[got] = true
	}
	if len(seen) != 3 {
		t.Errorf("三種結果只出現 %d 種", len(seen))
	}
	// 上限鉗在 9，下限（本專案補的）鉗在 0。
	for n := 0; n < 200; n++ {
		if got := MerchantSize(r, 20); got != MerchantMaxSize {
			t.Fatalf("基準 20 擲出 %d，應該鉗成 %d", got, MerchantMaxSize)
		}
		if got := MerchantSize(r, 0); got < 0 {
			t.Fatalf("基準 0 擲出負數 %d", got)
		}
	}
	// 規模就是等級。
	m := RollMerchant(rng.NewWithSeed(59), loadTables(t), loadItems(t), 6)
	if m.Level != m.Size {
		t.Errorf("等級 %d != 規模 %d", m.Level, m.Size)
	}
}

// 商隊議價：第一次一定成功、降 6%、翻臉之後那件貨不賣。
func TestHaggleWithMerchant(t *testing.T) {
	tb, items := loadTables(t), loadItems(t)
	m := RollMerchant(rng.NewWithSeed(83), tb, items, 8)
	if len(m.Wares) == 0 {
		t.Fatal("這支商隊沒有貨")
	}
	r := rng.NewWithSeed(84)

	// 商隊的議價一律從 0 開始 —— 不吃市集那個說服技能的初值加成。
	for i, w := range m.Wares {
		if w.Haggle != 0 {
			t.Fatalf("第 %d 件的議價初值是 %d，商隊應該一律 0", i, w.Haggle)
		}
	}

	list := m.Wares[0].Price
	outcome, ok := HaggleWith(r, &m, 0)
	if !ok || outcome != HaggleSuccess {
		t.Fatalf("第一次議價應該一定成功，得到 %v（ok=%v）", outcome, ok)
	}
	if got, want := m.Wares[0].WarePrice(), HagglePrice(list, 1); got != want {
		t.Errorf("殺價後要價 %d，預期 %d（標價 %d 打 6%%）", got, want, list)
	}

	// 一路議到翻臉，那一件就買不到了。
	for n := 0; n < 200 && !m.Wares[0].Haggle.Refused(); n++ {
		HaggleWith(r, &m, 0)
	}
	if !m.Wares[0].Haggle.Refused() {
		t.Fatal("議了 200 次都沒翻臉，機率算式不對")
	}
	res := BuyFromMerchant(&m, 0, []Character{*campChar("買家")}, 999999)
	if res.OK {
		t.Error("翻臉之後那件不該買得到")
	}
}
