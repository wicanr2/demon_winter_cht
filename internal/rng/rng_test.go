package rng

import "testing"

// TestLCGMatchesReferenceArithmetic 驗證我們的實作與原版的 32 位元
// 「乘 125、減 q×模數」算術等價。
//
// 原版用兩次 16 位元 MUL 湊 32 位元乘法，再用 SUB/SBB 做 32 位元借位減法。
// 這裡直接模擬那組指令的行為，與 Next() 對拍。
func TestLCGMatchesReferenceArithmetic(t *testing.T) {
	// 模擬原版指令序列：state 拆成 lo/hi 兩個 16 位元字。
	ref := func(state uint32) uint32 {
		lo := uint32(state & 0xFFFF)
		hi := uint32(state >> 16)
		// MUL BX(125) 兩次，湊出 32 位元乘積。
		p := lo * Multiplier
		newLo := p & 0xFFFF
		newHi := (p >> 16) + hi*Multiplier
		prod := (newHi << 16) | newLo
		// SUB/SBB：減去 q × 模數。
		q := prod / Modulus
		return prod - q*Modulus
	}

	r := NewWithSeed(1)
	state := uint32(1)
	for i := 0; i < 100000; i++ {
		state = ref(state)
		got := r.Next()
		if got != state {
			t.Fatalf("第 %d 步不一致：Next()=%d，參考算術=%d", i, got, state)
		}
	}
}

// TestStateStaysInRange 確認狀態永遠落在 [1, Modulus-1]。
// 狀態變成 0 會讓 LCG 死鎖在 0，是最致命的失效模式。
func TestStateStaysInRange(t *testing.T) {
	r := NewWithSeed(12345)
	for i := 0; i < 1000000; i++ {
		s := r.Next()
		if s == 0 || s >= Modulus {
			t.Fatalf("第 %d 步狀態越界：%d", i, s)
		}
	}
}

// TestZeroSeedNormalised 確認種子 0 會被正規化，不會讓產生器死鎖。
func TestZeroSeedNormalised(t *testing.T) {
	r := NewWithSeed(0)
	if r.State() == 0 {
		t.Fatal("種子 0 未被正規化，LCG 會永遠停在 0")
	}
	if got := r.Next(); got == 0 {
		t.Fatal("推進後狀態為 0")
	}
}

// TestRollBounds 驗證 Roll 的邊界行為與原版一致。
func TestRollBounds(t *testing.T) {
	r := NewWithSeed(999)

	// n = 0 或 1 直接回 1，且不消耗亂數（原版在這條路徑上不推進狀態）。
	before := r.State()
	for _, n := range []int{0, 1} {
		if got := r.Roll(n); got != 1 {
			t.Errorf("Roll(%d) = %d，預期 1", n, got)
		}
	}
	if r.State() != before {
		t.Error("Roll(0)/Roll(1) 不應推進 RNG 狀態")
	}

	// 負數取絕對值。
	if got := r.Roll(-1); got != 1 {
		t.Errorf("Roll(-1) = %d，預期 1", got)
	}

	// 一般範圍：戰鬥用到的 RNG(100) 與各種武器傷害骰。
	for _, n := range []int{3, 4, 6, 7, 8, 10, 12, 100} {
		for i := 0; i < 20000; i++ {
			v := r.Roll(n)
			if v < 1 || v > n {
				t.Fatalf("Roll(%d) 回傳 %d，超出 [1,%d]", n, v, n)
			}
		}
	}
}

// TestRollDistribution 檢查分布沒有明顯偏斜。
// 這不是嚴謹的統計檢定，只是抓「整段落在某一半」這類實作錯誤。
func TestRollDistribution(t *testing.T) {
	r := NewWithSeed(4242)
	const n, iter = 6, 600000
	counts := make([]int, n+1)
	for i := 0; i < iter; i++ {
		counts[r.Roll(n)]++
	}
	expected := iter / n
	for face := 1; face <= n; face++ {
		diff := counts[face] - expected
		if diff < 0 {
			diff = -diff
		}
		// 容許 5% 偏差。
		if diff > expected/20 {
			t.Errorf("面 %d 出現 %d 次，期望約 %d，偏差過大", face, counts[face], expected)
		}
	}
}

// TestSequenceReproducible 相同種子必須產生相同序列。
// 對拍原版時要靠這個性質固定測試條件。
func TestSequenceReproducible(t *testing.T) {
	a, b := NewWithSeed(777), NewWithSeed(777)
	for i := 0; i < 10000; i++ {
		if x, y := a.Next(), b.Next(); x != y {
			t.Fatalf("第 %d 步序列分歧：%d vs %d", i, x, y)
		}
	}
}

// TestPeriodIsLong 確認在合理步數內狀態不會回到起點。
// 乘數 125 對模數 2796203 若不是原根，週期會短到影響遊戲體感。
func TestPeriodIsLong(t *testing.T) {
	const seed = 1
	r := NewWithSeed(seed)
	for i := 1; i <= 100000; i++ {
		if r.Next() == seed {
			t.Fatalf("週期只有 %d 步，太短", i)
		}
	}
}
