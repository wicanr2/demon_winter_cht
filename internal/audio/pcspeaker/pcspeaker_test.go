package pcspeaker

import (
	"math"
	"testing"
)

// 八個單音必須精確落在 C 大調自然音階 C3–C4。
//
// 這是「除頻值抄對了」最硬的一道驗收：抄錯任何一個 byte，
// 頻率就不會剛好落在音階上。誤差容許 0.1 Hz，那是整數 divisor 的量化誤差。
func TestSingleTones_MatchCMajorScale(t *testing.T) {
	want := []struct {
		id   int
		name string
		hz   float64
	}{
		{EffectC3, "C3", 130.81},
		{EffectD3, "D3", 146.83},
		{EffectE3, "E3", 164.81},
		{EffectF3, "F3", 174.61},
		{EffectG3, "G3", 196.00},
		{EffectA3, "A3", 220.00},
		{EffectB3, "B3", 246.94},
		{EffectC4, "C4", 261.63},
	}
	for _, w := range want {
		notes := Effect(w.id)
		if len(notes) != 1 {
			t.Fatalf("效果 %d 應是單音，得到 %d 個音符", w.id, len(notes))
		}
		got := notes[0].Frequency()
		if math.Abs(got-w.hz) > 0.1 {
			t.Errorf("效果 %d（%s）= %.2f Hz，預期 %.2f Hz", w.id, w.name, got, w.hz)
		}
		if notes[0].Duration != singleToneTicks {
			t.Errorf("效果 %d 長度 %d tick，原版是 %d", w.id, notes[0].Duration, singleToneTicks)
		}
	}
}

// 一個 tick 約 54.9 ms。這個數字決定所有音效的快慢，抄錯會整體變速。
func TestTickDuration(t *testing.T) {
	ms := TickSeconds * 1000
	if ms < 54.8 || ms > 55.0 {
		t.Errorf("一個 tick = %.2f ms，預期約 54.9 ms", ms)
	}
}

// 死亡旋律：11 個音符、B3 起 C4 收、全長約 1.5 秒。
func TestDeathMelody(t *testing.T) {
	notes := Effect(EffectDeath)
	if len(notes) != 11 {
		t.Fatalf("死亡旋律有 %d 個音符，預期 11（第 12 筆 0xffff 是結束哨兵，不算音符）",
			len(notes))
	}

	if f := notes[0].Frequency(); math.Abs(f-246.94) > 0.1 {
		t.Errorf("起音 %.2f Hz，預期 B3 246.94", f)
	}
	if f := notes[len(notes)-1].Frequency(); math.Abs(f-261.63) > 0.1 {
		t.Errorf("收尾 %.2f Hz，預期 C4 261.63", f)
	}
	if d := Duration(notes); d < 1.4 || d > 1.7 {
		t.Errorf("全長 %.2f 秒，預期約 1.5 秒", d)
	}

	// 休止符不發聲但佔時間。
	rests := 0
	for _, n := range notes {
		if n.Divisor == 0 {
			if n.Frequency() != 0 {
				t.Error("休止符不該有頻率")
			}
			if n.Seconds() <= 0 {
				t.Error("休止符還是要佔時間")
			}
			rests++
		}
	}
	if rests != 5 {
		t.Errorf("休止符 %d 個，預期 5", rests)
	}
}

// 未知的效果編號回傳 nil，不是 panic 也不是隨便給一個音。
func TestEffect_UnknownIsSilent(t *testing.T) {
	for _, id := range []int{-2, 9, 10, 99} {
		if notes := Effect(id); notes != nil {
			t.Errorf("效果 %d 不存在，應回傳 nil，得到 %v", id, notes)
		}
	}
	if notes := Effect(EffectStop); notes != nil {
		t.Errorf("停止效果應回傳 nil，得到 %v", notes)
	}
}

// 合成出來的長度要對得上音符時長。
func TestRender_SampleCount(t *testing.T) {
	const rate = 44100
	notes := Effect(EffectC3)
	pcm := Render(notes, rate, 0.5)

	wantSamples := int(Duration(notes) * rate)
	gotSamples := len(pcm) / 2 // 16-bit 單聲道
	if gotSamples != wantSamples {
		t.Errorf("合成出 %d 個取樣，預期 %d", gotSamples, wantSamples)
	}
}

// 波形必須是方波：取樣值只有 +peak 與 −peak 兩種。
//
// PC speaker 只能開／關，用正弦波聽起來太柔，不是那個年代的聲音。
func TestRender_IsSquareWave(t *testing.T) {
	pcm := Render(Effect(EffectC3), 44100, 1.0)

	seen := map[int16]int{}
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int16(pcm[i]) | int16(pcm[i+1])<<8
		seen[v]++
	}
	if len(seen) != 2 {
		t.Errorf("方波應只有兩種取樣值，實際有 %d 種", len(seen))
	}
	for v := range seen {
		if v != math.MaxInt16 && v != -math.MaxInt16 {
			t.Errorf("取樣值 %d 不是 ±峰值", v)
		}
	}
}

// 數出過零次數反推頻率，確認合成出來的真的是那個音高。
//
// 只檢查音符表對不夠 —— 合成迴圈把週期算錯的話，
// 表是對的、聽起來卻不對。
func TestRender_ActualPitch(t *testing.T) {
	const rate = 44100
	notes := Effect(EffectA3) // 220 Hz
	pcm := Render(notes, rate, 1.0)

	transitions := 0
	var prev int16
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int16(pcm[i]) | int16(pcm[i+1])<<8
		if i > 0 && v != prev {
			transitions++
		}
		prev = v
	}
	// 一個週期有兩次轉換（高→低、低→高）。
	cycles := float64(transitions) / 2
	got := cycles / Duration(notes)
	if math.Abs(got-220) > 3 {
		t.Errorf("合成波形實測 %.1f Hz，預期約 220 Hz", got)
	}
}

// 休止段落必須全是靜音。
func TestRender_RestsAreSilent(t *testing.T) {
	pcm := Render([]Note{{Divisor: 0, Duration: 3}}, 44100, 1.0)
	if len(pcm) == 0 {
		t.Fatal("休止仍要產生對應長度的靜音取樣")
	}
	for i := range pcm {
		if pcm[i] != 0 {
			t.Fatalf("休止段落第 %d byte 不是 0", i)
		}
	}
}

func TestRender_Degenerate(t *testing.T) {
	if got := Render(Effect(EffectC3), 0, 1.0); got != nil {
		t.Error("取樣率 0 應回傳 nil")
	}
	if got := Render(nil, 44100, 1.0); len(got) != 0 {
		t.Error("沒有音符時不該產生取樣")
	}
	// 音量超出範圍要被鉗住，不能溢位成雜訊。
	loud := Render(Effect(EffectC3), 44100, 5.0)
	normal := Render(Effect(EffectC3), 44100, 1.0)
	if len(loud) != len(normal) {
		t.Error("音量不該影響長度")
	}
	for i := range loud {
		if loud[i] != normal[i] {
			t.Error("音量 > 1 應被鉗到 1")
			break
		}
	}
}
