// Package music 合成 Demon's Winter remake 的原創背景配樂。
// 原版 DOS 沒有 BGM；這些旋律是 remake 新編曲，不是還原素材。
package music

import (
	"encoding/binary"
	"math"
)

type Scene int

const (
	Silent Scene = iota
	Exploration
	Sanctuary
	Battle
	Finale
)

const SampleRate = 44100

type note struct {
	semitone int
	beats    float64
}
type score struct {
	bpm          float64
	root         int
	melody, bass []note
}

var scores = map[Scene]score{
	Exploration: {82, 45,
		[]note{{0, 1}, {3, .5}, {5, .5}, {7, 1}, {3, 1}, {0, 1}, {-2, 1}, {0, 2}, {0, 1}, {5, .5}, {7, .5}, {10, 1}, {7, 1}, {5, 1}, {3, 1}, {0, 2}},
		[]note{{0, 2}, {-5, 2}, {-2, 2}, {-7, 2}, {0, 2}, {-5, 2}, {-2, 2}, {-7, 2}}},
	Sanctuary: {68, 48,
		[]note{{0, 1}, {4, 1}, {7, 2}, {9, 1}, {7, 1}, {4, 2}, {2, 1}, {5, 1}, {9, 2}, {7, 1}, {5, 1}, {4, 2}},
		[]note{{0, 4}, {-5, 4}, {-3, 4}, {-5, 4}}},
	Battle: {126, 43,
		[]note{{0, .5}, {0, .5}, {3, 1}, {5, .5}, {3, .5}, {0, 1}, {7, .5}, {7, .5}, {8, 1}, {7, .5}, {5, .5}, {3, 1}, {0, .5}, {3, .5}, {5, 1}, {10, .5}, {8, .5}, {7, 1}, {5, 1}, {3, 1}},
		[]note{{0, 1}, {0, 1}, {-2, 1}, {-5, 1}, {0, 1}, {0, 1}, {3, 1}, {-2, 1}}},
	Finale: {96, 41,
		[]note{{0, 1}, {1, 1}, {5, 1}, {8, 1}, {7, 2}, {5, 1}, {1, 1}, {0, 1}, {5, 1}, {8, 1}, {12, 1}, {11, 2}, {8, 1}, {7, 1}},
		[]note{{0, 2}, {-1, 2}, {-5, 2}, {-4, 2}, {0, 2}, {-1, 2}, {-5, 2}, {-7, 2}}},
}

// Render 產生可循環的 16-bit 單聲道 PCM。波形完全由程式生成，不使用取樣庫。
func Render(scene Scene, volume float64) []byte {
	s, ok := scores[scene]
	if !ok || volume <= 0 {
		return nil
	}
	if volume > 1 {
		volume = 1
	}
	beat := float64(SampleRate) * 60 / s.bpm
	melody := renderLine(s.melody, s.root+12, beat, .62)
	bass := renderLine(s.bass, s.root-12, beat, .50)
	n := max(len(melody), len(bass))
	out := make([]byte, n*2)
	for i := 0; i < n; i++ {
		var v float64
		if i < len(melody) {
			v += melody[i]
		}
		if i < len(bass) {
			v += bass[i]
		}
		sample := int16(math.MaxInt16 * volume * .55 * math.Tanh(v))
		binary.LittleEndian.PutUint16(out[i*2:], uint16(sample))
	}
	return out
}

func renderLine(line []note, root int, beat, gain float64) []float64 {
	total := 0
	for _, n := range line {
		total += int(n.beats * beat)
	}
	out := make([]float64, total)
	pos := 0
	for _, n := range line {
		count := int(n.beats * beat)
		freq := 440 * math.Pow(2, float64(root+n.semitone-69)/12)
		for i := 0; i < count; i++ {
			phase := math.Mod(float64(i)*freq/SampleRate, 1)
			env := 1.0
			edge := 35 * SampleRate / 1000
			if i < edge {
				env = float64(i) / float64(edge)
			}
			if tail := count - i; tail < edge {
				env *= float64(tail) / float64(edge)
			}
			out[pos+i] = (1 - 4*math.Abs(phase-.5)) * env * gain
		}
		pos += count
	}
	return out
}
