package ui

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/wicanr2/demon_winter_cht/internal/audio/pcspeaker"
)

// SampleRate 是輸出取樣率。原版是 PC speaker（沒有取樣率的概念），
// 這裡挑一個現代音效卡通用的值來重建方波。
const SampleRate = 44100

// Speaker 播放原版的 PC speaker 音效。
//
// 波形在 pcspeaker 合成（純函式），這一層只負責推到音效裝置。
//
// **開裝置一律在背景做，主迴圈永遠不等它。** 沒有音效裝置的環境
// （容器、無頭 CI、沒裝音效卡的機器）開起來會一直卡住 ——
// 遊戲整個凍結比沒有聲音嚴重得多。ready 沒亮之前所有播放都是 no-op。
type Speaker struct {
	// ready 由背景 goroutine 在 players 準備好之後設起來。
	// 用 atomic 發布，主迴圈只在它為 1 時才碰 players。
	ready   atomic.Bool
	players map[int]*audio.Player

	// enabled 對應原版的 `[0x1585]` 音效開關（戰鬥選單有 Sound off 指令）。
	enabled atomic.Bool
}

// allEffects 是原版跳表涵蓋的全部效果編號。
var allEffects = []int{
	pcspeaker.EffectDeath,
	pcspeaker.EffectC3, pcspeaker.EffectD3, pcspeaker.EffectE3, pcspeaker.EffectF3,
	pcspeaker.EffectG3, pcspeaker.EffectA3, pcspeaker.EffectB3, pcspeaker.EffectC4,
}

// NewSpeaker 建立音效播放器。
//
// volume 是 0–1；**0 代表完全不碰音效裝置**（回傳 nil，所有方法都是 no-op），
// 無頭測試與截圖用得上。原版沒有音量控制，這個參數純粹是體貼。
func NewSpeaker(volume float64) *Speaker {
	if volume <= 0 || !audioDeviceAvailable() {
		return nil
	}

	s := &Speaker{players: map[int]*audio.Player{}}
	s.enabled.Store(true)

	// 波形合成很快，但開音效裝置可能很慢，所以整段丟到背景。
	go func() {
		ctx := audio.NewContext(SampleRate)
		players := make(map[int]*audio.Player, len(allEffects))
		for _, id := range allEffects {
			notes := pcspeaker.Effect(id)
			if len(notes) == 0 {
				continue
			}
			p, err := ctx.NewPlayer(bytes.NewReader(
				pcspeaker.Render(notes, SampleRate, volume)))
			if err != nil {
				return // 開不起來就一直靜音，ready 不會亮
			}
			players[id] = p
		}
		s.players = players
		s.ready.Store(true)
	}()

	return s
}

// SetEnabled 開關音效，對應原版戰鬥選單的 Sound on／Sound off。
func (s *Speaker) SetEnabled(on bool) {
	if s == nil {
		return
	}
	s.enabled.Store(on)
	if !on {
		s.Stop()
	}
}

// Enabled 回報音效是否開著。沒有音效裝置時一律回 false。
func (s *Speaker) Enabled() bool { return s != nil && s.enabled.Load() }

// Play 播放一個效果。裝置還沒準備好、或根本沒有裝置時什麼都不做。
//
// **同一個效果重播時從頭開始，不疊音。** 原版只有一個喇叭通道，
// 疊起來會變成完全不同的聲音。
func (s *Speaker) Play(id int) {
	if s == nil || !s.enabled.Load() || !s.ready.Load() {
		return
	}
	p, ok := s.players[id]
	if !ok {
		return
	}
	// 原版只有一條 PC speaker 通道；新效果會取代目前聲音，不能讓不同
	// effect 的 audio.Player 疊在一起變成原版不存在的和弦。
	for _, other := range s.players {
		other.Pause()
	}
	if err := p.Rewind(); err != nil {
		return
	}
	p.Play()
}

// Stop 停掉所有正在播的音效。
func (s *Speaker) Stop() {
	if s == nil || !s.ready.Load() {
		return
	}
	for _, p := range s.players {
		p.Pause()
	}
}

// audioDeviceAvailable 回報這台機器有沒有可用的音效裝置。
//
// **必須在建立 audio context 之前檢查。** Ebiten 的音效 context 會掛進遊戲
// 主迴圈；容器或沒有音效卡的機器上建起來會讓整個遊戲凍住 ——
// 那比沒有聲音嚴重得多，而且看起來像當機不像缺功能。
// 實測就是這樣：headless 容器裡跑到攻擊那一步整個卡死。
//
// Linux 上判準是 /dev/snd 底下有沒有裝置節點；其他平台一律當成有
// （macOS／Windows 都有系統級的音效抽象層，不會因為沒有硬體就卡住）。
func audioDeviceAvailable() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	entries, err := os.ReadDir("/dev/snd")
	if err != nil {
		return false
	}
	for _, e := range entries {
		// controlC0／pcmC0D0p 之類的節點才算數，空目錄不算。
		if strings.HasPrefix(e.Name(), "pcm") || strings.HasPrefix(e.Name(), "control") {
			return true
		}
	}
	return false
}
