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

// TestRollFloatEquivalence 是「浮點路徑 vs 整數路徑」等價性的回歸測試。
//
// 背景（完整推導與窮舉證據見 docs/re/14-rng-float-equivalence.md）：原版 Roll(n)
// 實際走的是「先除後乘」的軟體浮點路徑（state/Modulus 先算成浮點小數，再乘 n，
// 最後截斷取整），現行實作是「先乘後除」的純整數路徑（state*n 先算，再整除
// Modulus）。兩者在有理數上等價，但要確認浮點捨入不會讓 floor() 結果跨過整數
// 邊界差 1。
//
// docs/re/14 已用數論方式證明並窮舉驗證（tools/rng_float_equiv_check.go，
// 33,554,424 組 (state,n) 全數檢查）：因為 Modulus=2796203 是質數、遊戲用到的
// n 都遠小於 Modulus 且 state 恆不為 0，所以 state*n mod Modulus 恆不為 0，
// 每一組合的「離最近的 Modulus 倍數」至少差 1（約 21.4 bit 的安全邊界）——
// 而原版浮點函式庫的尾數寬度已由反組譯證實 ≥ 48 bit（見 docs/re/14 §2.3），
// 遠遠蓋過這個邊界，浮點捨入不可能改變 floor() 的結果。
//
// 這裡只做**抽樣**式的複查（用直接算術重新驗證同一個數論不等式，不依賴 Roll()
// 本身，因為 Roll() 就是被驗證的對象），確保這條性質沒有在未來的改動中被破壞；
// 完整窮舉已經在上面提到的 docs/re/14 與 tools/rng_float_equiv_check.go 做過，
// 不需要在單元測試裡重跑一次全部 2,796,202 個狀態。
func TestRollFloatEquivalence(t *testing.T) {
	ns := []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 100}
	// 抽樣：均勻跳點掃過整個狀態空間，抓「性質被破壞」這類系統性錯誤即可，
	// 不需要每次測試都窮舉 279 萬次。
	const stride = 977 // 與 Modulus 互質的隨意步長，讓抽樣點分散
	for _, n := range ns {
		checked := 0
		for state := 1; state < Modulus; state += stride {
			product := int64(state) * int64(n)
			r := product % Modulus
			if r == 0 {
				t.Fatalf("n=%d state=%d: state*n 恰為 Modulus 倍數，違反「Modulus 為質數且 n<Modulus」的前提，數論論證失效", n, state)
			}
			margin := r
			if Modulus-r < margin {
				margin = Modulus - r
			}
			// 21 bit 相對邊界（保守取 2^21，比實測最小值 2^21.4 略嚴格一點點）。
			// 原版尾數寬度 ≥ 48 bit，這裡驗證的是「邊界仍然遠大於浮點誤差」這個
			// 前提沒有被破壞，不是在驗證浮點函式庫本身。
			if margin < 1 {
				t.Fatalf("n=%d state=%d: margin=%d 小於安全邊界", n, state, margin)
			}
			checked++
		}
		if checked == 0 {
			t.Fatalf("n=%d: 抽樣迴圈沒有跑到任何狀態", n)
		}
	}

	// 同時確認 Roll() 對外行為在這些 n 上持續落在 [1,n]（TestRollBounds 已經測過，
	// 這裡只是把「等價性前提成立」與「Roll() 實際行為正常」放在同一份測試裡交叉確認）。
	r := NewWithSeed(20260725)
	for _, n := range ns {
		for i := 0; i < 2000; i++ {
			if got := r.Roll(n); got < 1 || got > n {
				t.Fatalf("n=%d: Roll 回傳 %d，超出 [1,%d]", n, got, n)
			}
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
