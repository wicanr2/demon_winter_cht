// Package pcspeaker 重現原版的 PC speaker 音效。
//
// **原版沒有配樂。** 全檔唯一的聲音相關 port I/O 是 `OUT 0x42`／`OUT 0x43`／
// `IN/OUT 0x61`，也就是 8253 計時器 channel 2 加上喇叭閘 —— PC speaker 的標準操作。
// AdLib／MIDI 在反組譯中零命中（見 `docs/re/03-audio-and-resources.md` §1.1）。
// 所以「音樂還原」的正確範圍是忠實重現這幾段音效序列，不是自製配樂。
//
// 這個套件只做**波形合成**，不碰播放裝置 —— 合成是純函式，
// 無頭環境下測得到；播放由 ui 層負責。
package pcspeaker

import "math"

// PITFrequency 是 8253/8254 計時器的輸入頻率。
//
// 發聲頻率 = PITFrequency / divisor。原版寫進 port 0x42 的就是 divisor。
const PITFrequency = 1193182.0

// TickHz 是 INT 1Ch 的觸發頻率。音符長度以 tick 計。
//
// 一個 tick ≈ 54.9 ms。原版把音符播放常式掛在 INT 1Ch 上，
// 每次中斷推進一格（見 docs/re/03 §1.2–1.3）。
const TickHz = 1193182.0 / 65536.0

// TickSeconds 是一個 tick 的秒數。
const TickSeconds = 1.0 / TickHz

// Note 是一筆音符記錄：頻率除頻值 + 持續 tick 數。
//
// Divisor 為 0 代表休止（不發聲但佔時間）。
type Note struct {
	Divisor  int
	Duration int
}

// Frequency 回傳這個音符的頻率（Hz）。休止回傳 0。
func (n Note) Frequency() float64 {
	if n.Divisor <= 0 {
		return 0
	}
	return PITFrequency / float64(n.Divisor)
}

// Seconds 回傳這個音符的長度（秒）。
func (n Note) Seconds() float64 { return float64(n.Duration) * TickSeconds }

// 效果編號。沿用原版跳表的 effect_id，不另外編號。
const (
	// EffectDeath 是唯一的多音符效果：11 個音符的短旋律，全長約 1.5 秒。
	// 呼叫端是「清空單位 HP 與座標」的函式 —— 也就是單位陣亡。
	EffectDeath = -1
	// EffectStop 停止發聲。
	EffectStop = 0
)

// 八個單音的效果編號。頻率精確對上 C 大調自然音階 C3–C4
// （誤差全在 0.1 Hz 內，是整數 divisor 的量化誤差，不可能是巧合）。
const (
	EffectC3 = 1
	EffectD3 = 2
	EffectE3 = 3
	EffectF3 = 4
	EffectG3 = 5
	EffectA3 = 6
	EffectB3 = 7
	EffectC4 = 8
)

// singleToneDivisors 是 effect_id 1–8 的除頻值，逐 byte 從原版反組譯核對過。
var singleToneDivisors = map[int]int{
	EffectC3: 0x23a2,
	EffectD3: 0x1fbe,
	EffectE3: 0x1c48,
	EffectF3: 0x1ab1,
	EffectG3: 0x17c8,
	EffectA3: 0x1530,
	EffectB3: 0x12e0,
	EffectC4: 0x11d0,
}

// singleToneTicks 是八個單音的長度。原版全部是 3 tick。
const singleToneTicks = 3

// deathMelody 是 effect_id −1 的 11 個音符（原版第 12 筆是 0xffff 結束哨兵，不算音符）。
//
// B3 起音、C4 收尾，中段穿插短促的 A3／B3／C4／G3。
var deathMelody = []Note{
	{0x12e0, 6}, // B3
	{0x0000, 4}, // 休止
	{0x1530, 2}, // A3
	{0x0000, 1},
	{0x12e0, 2}, // B3
	{0x0000, 1},
	{0x11d0, 2}, // C4
	{0x0000, 1},
	{0x17c8, 2}, // G3
	{0x0000, 1},
	{0x11d0, 6}, // C4
}

// Effect 回傳一個效果編號對應的音符序列。
//
// 未知的編號回傳 nil（原版是 `CMP AX,0xa ; JNC skip`，超出範圍直接跳過）。
func Effect(id int) []Note {
	if id == EffectDeath {
		return append([]Note(nil), deathMelody...)
	}
	if id == EffectStop {
		return nil
	}
	div, ok := singleToneDivisors[id]
	if !ok {
		return nil
	}
	return []Note{{Divisor: div, Duration: singleToneTicks}}
}

// Render 把一段音符序列合成成 16-bit 單聲道 PCM（little-endian）。
//
// PC speaker 只能開／關，所以波形是**方波**不是正弦波 ——
// 用正弦波聽起來會太柔，不是那個年代的聲音。
//
// amplitude 是 0–1 的音量。原版沒有音量控制（喇叭只有開關），
// 這裡留一個參數純粹是為了不要吵到玩家。
func Render(notes []Note, sampleRate int, amplitude float64) []byte {
	if sampleRate <= 0 {
		return nil
	}
	if amplitude < 0 {
		amplitude = 0
	}
	if amplitude > 1 {
		amplitude = 1
	}
	peak := int16(float64(math.MaxInt16) * amplitude)

	var out []byte
	for _, n := range notes {
		samples := int(n.Seconds() * float64(sampleRate))
		freq := n.Frequency()

		for i := 0; i < samples; i++ {
			var v int16
			if freq > 0 {
				// 方波：一個週期的前半高、後半低。
				period := float64(sampleRate) / freq
				if math.Mod(float64(i), period) < period/2 {
					v = peak
				} else {
					v = -peak
				}
			}
			out = append(out, byte(v), byte(v>>8))
		}
	}
	return out
}

// Duration 回傳一段音符序列的總長度（秒）。
func Duration(notes []Note) float64 {
	total := 0.0
	for _, n := range notes {
		total += n.Seconds()
	}
	return total
}
