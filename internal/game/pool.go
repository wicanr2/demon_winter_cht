package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 治療水池（tile `0x35`，原版動作 `0x11` ＝ `222f:37c4`，`docs/re/90`）
//
// **`docs/re/05` 把這個 tile 記成「寫死的阻擋（推測：撞牆）」，那是錯的。**
// 隊伍確實不會走上去（回傳碼發生在寫座標之前），但**會發生事**：
// 選一名角色喝水，回 1–4 點 HP，然後回到迴圈再問一次，
// 直到玩家取消或今天的額度用完。
//
// 額度是隊伍共用的每日次數，睡覺補回 7（存檔 `+0xaa`，`scenario.PoolDrinks`）。
// 與觀地（一天一次）同一組，只是額度不同。
//
// ⚠ **與手冊「地底 → 陷阱」的水池陷阱是兩件事**：那個是掉進去、
// 每回合 `Roll(3)−1` 傷害（`docs/re/68`，`TrapPool`）。名字撞在一起，
// 數值不能互相搬。

const (
	// PoolHealDie 是每一口回復的點數（`0x19769`，1–4）。
	PoolHealDie = 4
	// PoolDrinksPerDay 是睡一覺之後的額度（`0x1eee6` 寫死的 7）。
	PoolDrinksPerDay = 7
)

// PoolSip 是喝一口的結果。
type PoolSip struct {
	// Empty 代表今天的額度已經用完（原版印 `The pool is empty`）。
	Empty bool
	// Member 是喝水的隊員索引。
	Member int
	// Healed 是實際回復的點數。**0 是合法結果** —— 滿血的人喝下去
	// 鉗成 0，而原版照樣印一行 `He is healed 0`，額度也照扣。
	Healed int
}

// DrinkFromPool 讓一名隊員喝一口水池。
//
// drinks 是剩餘額度的指標，**扣減在擲點之前**（原版 `0x1974a` 的 `dec`
// 就排在選人之後、擲點之前）—— 所以滿血的人喝一口也會消耗一次。
//
// 回傳的 Empty 為真時什麼都沒發生，額度也沒動。
func DrinkFromPool(r *rng.RNG, party []Character, who int, drinks *byte) PoolSip {
	if drinks == nil || *drinks == 0 {
		return PoolSip{Empty: true}
	}
	if who < 0 || who >= len(party) {
		return PoolSip{Empty: true}
	}
	*drinks--

	c := &party[who]
	heal := r.Roll(PoolHealDie)
	// 鉗到上限。原版比的是 `最大 HP < 目前 + 擲點`，成立才改成差額。
	if c.MaxHP < c.CurrentHP+heal {
		heal = c.MaxHP - c.CurrentHP
	}
	if heal < 0 {
		heal = 0
	}
	c.CurrentHP += heal
	return PoolSip{Member: who, Healed: heal}
}

// PoolCanDrink 回報這一格還喝不喝得到（只看額度，不看誰）。
func PoolCanDrink(drinks byte) bool { return drinks > 0 }

// ResetPoolDrinks 是睡覺時把額度補回 7（原版 `0x1eee6`，就在
// `You sleep.` 那支裡，與清光源、清每日旗標同一段）。
//
// **不是「進到新的一口池子就回滿」** —— 那是隊伍層級的每日額度。
func ResetPoolDrinks(s *scenario.SaveGame) {
	if s == nil {
		return
	}
	s.PoolDrinks = PoolDrinksPerDay
}
