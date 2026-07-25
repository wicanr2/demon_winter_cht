// rng_float_equiv_check 驗證原版 Roll(n) 的浮點路徑（先除後乘：state/Modulus × n）
// 與現行 Go 整數實作（先乘後除：state × n / Modulus）在數學上必然給出相同的
// floor() 結果，理由見 docs/re/14-rng-float-equivalence.md。
//
// 核心論證（數論，非抽樣）：
//   - Modulus = 2796203 是質數（已知事實，Wagstaff 質數）。
//   - Roll(n) 只在 n 屬於 [2, 2796202]（遠小於 Modulus）時才會執行到這條浮點路徑
//     （n=0/1 提早回傳，見 spec）；因為 Modulus 是質數且 n < Modulus，
//     gcd(n, Modulus) = 1，任何 n 都不是 Modulus 的倍數。
//   - state 恆在 [1, Modulus-1]（規格保證，狀態不會是 0）。
//   - 因此 state*n mod Modulus 恆不為 0：exact 有理數 state*n/Modulus 的小數部分
//     恆與最近的整數邊界至少相差 1/Modulus ≈ 3.578e-7（約 21.4 bit）。
//   - 原版浮點函式庫的尾數寬度 ≥ 64 bit（4 個 16-bit word 的長乘法核心，
//     見 FUN_310e_0588），相對誤差 ≈ 2^-64，遠小於上述安全邊界（多出 ~43 bit）。
//   - 浮點捨入誤差絕不可能大到讓 floor() 跨過整數邊界，兩種算法的 floor()
//     結果因此在數學上保證逐一相同——這是窮舉且完備的證明，不是抽樣。
//
// 本程式做兩件事，把上面的論證釘死在實際數字上：
//  1. 驗證 Modulus 是質數（試除法，2796203 不大，瞬間跑完）。
//  2. 對每個遊戲實際會用到的 n，窮舉全部 2,796,202 個合法 state，
//     計算 r = (state*n) mod Modulus，確認 r 恆不為 0，並回報全域最小
//     margin = min(r, Modulus-r)（用來具體量化「安全邊界有多大」）。
package main

import (
	"fmt"
	"math"
	"os"
)

const modulus = 2796203

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func main() {
	fmt.Printf("Modulus = %d\n", modulus)
	prime := isPrime(modulus)
	fmt.Printf("Modulus is prime: %v\n", prime)
	if !prime {
		fmt.Println("FATAL: modulus 不是質數，數論論證不成立，需要重新設計驗證方式")
		os.Exit(1)
	}

	// 遊戲中會用到的 n：1..12（涵蓋一般擲骰）、武器骰表 3,4,6,7,8,10,12（1..12 子集）、
	// 命中/爆擊判定用的 100。
	ns := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 100}

	globalMinMargin := modulus
	var globalMinState, globalMinN int
	zeroCount := 0
	totalChecked := 0

	for _, n := range ns {
		if n <= 1 {
			continue // Roll(0)/Roll(1) 不走浮點路徑，不消耗狀態，見 spec
		}
		minMargin := modulus
		var minState int
		for state := 1; state < modulus; state++ {
			product := state * n
			r := product % modulus
			if r == 0 {
				zeroCount++
				continue
			}
			margin := r
			if modulus-r < margin {
				margin = modulus - r
			}
			if margin < minMargin {
				minMargin = margin
				minState = state
			}
			totalChecked++
		}
		fmt.Printf("n=%3d: 窮舉 %d 個 state，最小 margin=%d（相對誤差邊界 ≈ %.3e，對應 ~%.1f bit）於 state=%d\n",
			n, modulus-1, minMargin, float64(minMargin)/float64(modulus), -logBase2(float64(minMargin)/float64(modulus)), minState)
		if minMargin < globalMinMargin {
			globalMinMargin = minMargin
			globalMinState = minState
			globalMinN = n
		}
	}

	fmt.Printf("\n總計窮舉組合數：%d（n 種類 × state 範圍）\n", totalChecked)
	fmt.Printf("state*n ≡ 0 (mod Modulus) 出現次數：%d（理論上應為 0，因為 Modulus 是質數且所有 n < Modulus）\n", zeroCount)
	fmt.Printf("全域最小 margin：%d（發生於 n=%d, state=%d），相對誤差邊界 ≈ %.3e，約 %.1f bit\n",
		globalMinMargin, globalMinN, globalMinState,
		float64(globalMinMargin)/float64(modulus), -logBase2(float64(globalMinMargin)/float64(modulus)))
	fmt.Println("\n只要原版浮點尾數精度 > 這個 bit 數（已由 RE 證實 ≥ 48-64 bit，見 docs/re/14），")
	fmt.Println("floor() 結果就不可能因為浮點捨入誤差而偏離窮舉整數運算的結果。")

	if zeroCount != 0 {
		fmt.Println("\n[WARN] 出現非預期的 state*n ≡ 0 情況，數論前提可能有誤，需要重新檢視！")
		os.Exit(1)
	}
}

func logBase2(x float64) float64 {
	return math.Log2(x)
}
