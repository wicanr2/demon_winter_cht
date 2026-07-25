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

// Roll 擲一個 n 面骰，回傳 [1, n]。
//
// 對應原版的頂層擲骰函式 FUN_1d9f_0e0b（1d9f:0e0b，全域 234 處呼叫）：
//
//	負數取絕對值；n 為 0 或 1 時直接回傳 1（不消耗亂數）；
//	否則 floor(uniform × n) + 1。
//
// 原版把狀態轉成浮點數再乘 n，用的是自製的軟體浮點函式庫（段 310e，
// 模擬 80 位元擴充精度），不是硬體 x87。這裡用整數運算取代：
// 對 n 遠小於 Modulus 的情況（遊戲中最大是 RNG(100)）兩者結果一致，
// 且避免了浮點捨入帶來的平台差異。
func (r *RNG) Roll(n int) int {
	if n < 0 {
		n = -n
	}
	if n == 0 || n == 1 {
		return 1
	}
	return int(uint64(r.Next())*uint64(n)/uint64(Modulus)) + 1
}
