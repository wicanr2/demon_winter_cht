package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 回復量是 1–4（`Roll(4)`），與陷阱那組一樣**釘邊界不釘區間**。
func TestPoolHealRange(t *testing.T) {
	lo, hi := 1<<30, 0
	for i := 1; i <= samples; i++ {
		p := []Character{{Name: "甲", CurrentHP: 1, MaxHP: 9999}}
		drinks := byte(PoolDrinksPerDay)
		sip := DrinkFromPool(trapRNG(i), p, 0, &drinks)
		if sip.Healed < lo {
			lo = sip.Healed
		}
		if sip.Healed > hi {
			hi = sip.Healed
		}
	}
	if lo != 1 || hi != PoolHealDie {
		t.Errorf("回復量 = %d–%d，預期 1–%d", lo, hi, PoolHealDie)
	}
}

// 鉗到上限，**而且滿血的人喝一口照樣扣一次額度**。
//
// 這一條單獨釘，因為它是最容易被「順手優化」掉的行為：
// 原版的 `dec` 排在選人之後、擲點之前（`0x1974a`），所以沒回到血
// 也一樣消耗。改成「回復 0 就不扣」看起來比較合理，但那不是原版。
func TestPoolClampsAndStillSpendsADrink(t *testing.T) {
	p := []Character{{Name: "甲", CurrentHP: 20, MaxHP: 20}}
	drinks := byte(3)

	sip := DrinkFromPool(trapRNG(1), p, 0, &drinks)
	if sip.Healed != 0 {
		t.Errorf("滿血的人回復了 %d 點", sip.Healed)
	}
	if p[0].CurrentHP != 20 {
		t.Errorf("滿血的人變成 %d 點", p[0].CurrentHP)
	}
	if drinks != 2 {
		t.Errorf("額度 = %d，預期 2 —— 沒回到血也要扣一次", drinks)
	}
}

// 差一點滿血時只補到上限。
func TestPoolHealsAtMostToMax(t *testing.T) {
	for i := 1; i <= samples; i++ {
		p := []Character{{Name: "甲", CurrentHP: 19, MaxHP: 20}}
		drinks := byte(PoolDrinksPerDay)
		DrinkFromPool(trapRNG(i), p, 0, &drinks)
		if p[0].CurrentHP > 20 {
			t.Fatalf("第 %d 次喝超過上限：%d/20", i, p[0].CurrentHP)
		}
	}
}

// 額度用完就是 `The pool is empty`，而且什麼都不動。
func TestPoolEmpty(t *testing.T) {
	p := []Character{{Name: "甲", CurrentHP: 1, MaxHP: 20}}
	drinks := byte(0)

	sip := DrinkFromPool(trapRNG(1), p, 0, &drinks)
	if !sip.Empty {
		t.Error("額度 0 卻還喝得到")
	}
	if p[0].CurrentHP != 1 {
		t.Errorf("空水池還是回了血：%d", p[0].CurrentHP)
	}
	if PoolCanDrink(0) {
		t.Error("PoolCanDrink(0) 應該是 false")
	}
}

// 額度是**隊伍共用**的每日次數：七口就是七口，換誰喝都一樣。
func TestPoolDrinksAreSharedByTheParty(t *testing.T) {
	p := []Character{
		{Name: "甲", CurrentHP: 1, MaxHP: 99},
		{Name: "乙", CurrentHP: 1, MaxHP: 99},
	}
	drinks := byte(PoolDrinksPerDay)
	for i := 0; i < PoolDrinksPerDay; i++ {
		if sip := DrinkFromPool(trapRNG(i+1), p, i%2, &drinks); sip.Empty {
			t.Fatalf("第 %d 口就空了", i+1)
		}
	}
	if drinks != 0 {
		t.Errorf("喝了 %d 口之後額度 = %d，預期 0", PoolDrinksPerDay, drinks)
	}
	if !DrinkFromPool(trapRNG(1), p, 0, &drinks).Empty {
		t.Error("第八口還喝得到 —— 額度是隊伍共用的，不是每人七口")
	}
}

// 睡覺補回 7（原版 `0x1eee6`，就在 `You sleep.` 那支裡）。
func TestResetPoolDrinks(t *testing.T) {
	s := &scenario.SaveGame{PoolDrinks: 0}
	ResetPoolDrinks(s)
	if s.PoolDrinks != PoolDrinksPerDay {
		t.Errorf("睡覺之後 = %d，預期 %d", s.PoolDrinks, PoolDrinksPerDay)
	}
	ResetPoolDrinks(nil) // 不 panic
}

// 越界的隊員索引當成喝不到，額度不動。
func TestPoolOutOfRange(t *testing.T) {
	p := []Character{{Name: "甲", CurrentHP: 1, MaxHP: 20}}
	drinks := byte(PoolDrinksPerDay)
	for _, i := range []int{-1, 1, 99} {
		if !DrinkFromPool(rng.NewWithSeed(1), p, i, &drinks).Empty {
			t.Errorf("索引 %d 竟然喝得到", i)
		}
	}
	if drinks != PoolDrinksPerDay {
		t.Errorf("越界卻扣了額度：剩 %d", drinks)
	}
}

// tile 0x35 的觸發方式是水池，**不是「寫死的阻擋」**（`docs/re/05` 已推翻）。
func TestPoolTileTrigger(t *testing.T) {
	if got := TriggerFor(0x35); got != TriggerPool {
		t.Errorf("tile 0x35 → %d，預期 TriggerPool(%d)", got, TriggerPool)
	}
}
