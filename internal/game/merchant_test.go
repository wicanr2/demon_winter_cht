package game

import (
	"testing"

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
		for _, w := range m.Wares {
			if w.Identified {
				t.Fatal("商隊的貨不該是已鑑定的")
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
		for _, w := range m.Wares {
			if w.Enchant < 0 {
				t.Fatalf("詛咒機率 0 的商隊卻有詛咒品：%+v", w)
			}
		}
	}
}
