// Package rng 重現原版 Demon's Winter 的亂數產生器。
//
// 原版的核心是一個線性同餘產生器（LCG），狀態存在 real mode 的
// 記憶體位址 0x481C（低字）與 0x481E（高字），構成一個 32 位元整數：
//
//	state = (state × 125) mod 2796203
//
// 乘數 125 與模數 2796203 都由反組譯確認（見 docs/re/06-combat-system.md）。
// 模數 0x2AAAAB 是 Wagstaff 質數 (2²³+1)/3。
//
// 對應的原版指令（segment 30c2）：
//
//	30c2:000f  MOV BX,0x7d      ; 125
//	30c2:0012  MUL BX           ; 32 位元乘法，分兩次 16 位元 MUL
//	30c2:0017  MUL BX
//	30c2:0029  MOV CX,0xaaab    ; 模數低位
//	30c2:002c  MOV BX,0x2a      ; 模數高位 → 0x2AAAAB
//	30c2:004a  SUB word ptr [0x481c],AX
//	30c2:004e  SBB word ptr [0x481e],DX
//
// 這個 RNG 是整個引擎「行為對齊原版」的基石：戰鬥命中、傷害骰、
// 隨機遭遇全部依賴它。序列不一致的話，其他部分再正確也對不上原版。
package rng

import "time"

const (
	// Multiplier 是 LCG 乘數（原版 0x7D）。
	Multiplier = 125
	// Modulus 是 LCG 模數（原版 0x2AAAAB），Wagstaff 質數 (2²³+1)/3。
	Modulus = 2796203
)

// RNG 是原版亂數產生器的狀態。零值不可用，請用 New 或 NewWithSeed 建立。
type RNG struct {
	state uint32
}

// New 以目前時間當種子建立產生器。
//
// 對應原版從 DOS 系統時鐘（INT 21h, AH=2Ch）取種子的行為。
// 要做確定性對拍請改用 NewWithSeed。
func New() *RNG {
	return NewWithSeed(uint32(time.Now().UnixNano()))
}

// NewWithSeed 以指定種子建立產生器。
//
// 種子會被正規化到 [1, Modulus-1]：狀態為 0 會讓 LCG 永遠卡在 0，
// 所以 0 會被換成 1。原版的種子來自 DOS 系統時鐘（INT 21h, AH=2Ch），
// 但重現時通常會指定固定種子以便對拍。
func NewWithSeed(seed uint32) *RNG {
	s := seed % Modulus
	if s == 0 {
		s = 1
	}
	return &RNG{state: s}
}

// Next 推進狀態一步並回傳新的狀態值，範圍 [1, Modulus-1]。
func (r *RNG) Next() uint32 {
	// 原版用兩次 16 位元 MUL 湊出 32 位元乘法。state < 2796203 < 2²²，
	// 乘 125 後小於 2²⁹，用 uint64 中介不會溢位，結果與原版逐位元相同。
	r.state = uint32((uint64(r.state) * Multiplier) % Modulus)
	return r.state
}

// State 回傳目前的內部狀態，供存檔或對拍除錯用。
func (r *RNG) State() uint32 { return r.state }

// SetState 直接設定內部狀態，供讀檔或對拍除錯用。
func (r *RNG) SetState(s uint32) { r.state = s }

// Uniform 推進一步並回傳 [0, 1) 的均勻亂數。
//
// 這就是原版 `1d9f:0dd4`（→ `30c2:0006`）留在軟浮點暫存器 A 裡的值：
// LCG 推進之後**立刻除以 2796203.0**（IEEE double）。`Roll` 是它的
// 整數等價版本（見下），但有些地方（例如商隊售價的 ±40% 浮動，
// `docs/re/45`）需要那個小數本身。
//
// **這裡刻意用 float64 而不是有理數**：呼叫端接著要做的乘加也是原版的
// IEEE double 運算，用同一組型別與同一個順序才逐位元相同。
func (r *RNG) Uniform() float64 {
	return float64(r.Next()) / float64(Modulus)
}

// Roll 擲一個 n 面骰，回傳 [1, n]。
//
// 對應原版的頂層擲骰函式 FUN_1d9f_0e0b（1d9f:0e0b，全域 234 處呼叫）：
//
//	負數取絕對值；n 為 0 或 1 時直接回傳 1（不消耗亂數）；
//	否則 floor(uniform × n) + 1。
//
// 原版走的是自製軟體浮點函式庫（段 310e）：LCG 狀態推進函式（30c2:0006）
// 結尾就先把新狀態除以 2796203.0（IEEE double），Roll(n) 再把這個小數乘上 n、
// 截斷取整。這裡用整數運算取代（先乘後除，而非原版的先除後乘），兩者在有理數
// 上等價；經 docs/re/14-rng-float-equivalence.md 用數論證明並窮舉驗證
// （33,554,424 組 (state,n) 組合），兩條路徑的 floor() 結果保證逐一相同——
// 因為 Modulus 是質數、state 恆不為 0、且遊戲用到的 n 都遠小於 Modulus，
// 精確有理數 state*n/Modulus 的小數部分恆與整數邊界至少相差 1/Modulus
// （約 21.4 bit 安全邊界），而原版浮點精度下界是 IEEE double 的 53 bit，
// 捨入誤差不可能大到讓 floor() 跨過邊界。回歸測試見 rng_test.go 的
// TestRollFloatEquivalence。
func (r *RNG) Roll(n int) int {
	if n < 0 {
		n = -n
	}
	if n == 0 || n == 1 {
		return 1
	}
	return int(uint64(r.Next())*uint64(n)/uint64(Modulus)) + 1
}
