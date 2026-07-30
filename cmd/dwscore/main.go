// Command dwscore 產生《冬之魔》宣傳片專用的 72 秒 MIDI 配樂。
//
// 這不是 1988 原版 BGM。旋律只取原作陣亡音效 B3-A3-B3-C4-G3-C4 的輪廓，
// 其餘和聲、配器、節奏與三幕結構均為 remake 宣傳片的新編曲。
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
)

const (
	ppq          = 480
	microseconds = 750000 // 80 BPM
	ticksPerSec  = 640
	totalSeconds = 72
)

type event struct {
	tick     int
	priority int
	data     []byte
}

type score struct{ events []event }

func (s *score) add(tick, priority int, data ...byte) {
	s.events = append(s.events, event{tick: tick, priority: priority, data: data})
}

func (s *score) program(ch, program, volume, pan byte) {
	s.add(0, 0, 0xc0|ch, program)
	s.add(0, 0, 0xb0|ch, 7, volume)
	s.add(0, 0, 0xb0|ch, 10, pan)
}

func tick(sec float64) int { return int(sec*ticksPerSec + 0.5) }

func (s *score) note(ch, pitch, velocity byte, start, duration float64) {
	begin, end := tick(start), tick(start+duration)
	s.add(begin, 2, 0x90|ch, pitch, velocity)
	s.add(end, 1, 0x80|ch, pitch, 0)
}

func (s *score) chord(ch byte, pitches []byte, velocity byte, start, duration float64) {
	for _, p := range pitches {
		s.note(ch, p, velocity, start, duration)
	}
}

func main() {
	out := flag.String("out", "workplace/promo/score/demon-winter-trailer.mid", "輸出 MIDI")
	flag.Parse()

	var s score
	// GM：低音提琴、大提琴、法國號、合唱、弦樂群；第 10 聲道是打擊樂。
	s.program(0, 43, 92, 52)
	s.program(1, 42, 102, 76)
	s.program(2, 60, 108, 64)
	s.program(3, 52, 88, 64)
	s.program(4, 48, 92, 64)
	compose(&s)

	if err := os.MkdirAll(dir(*out), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*out, midi(s.events), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("宣傳片配樂 MIDI：%s（%d 秒，80 BPM）\n", *out, totalSeconds)
}

func compose(s *score) {
	// 第一幕：冰封檔案。只有低音持續音與極稀疏的心跳。
	s.note(0, 35, 44, 0, 7.7) // B1
	s.note(0, 30, 38, 4, 3.7) // F#1
	for _, at := range []float64{2.0, 4.7, 7.1} {
		drum(s, 36, 44, at, .12)
	}

	// 陣亡旋律輪廓，壓低八度並拉長，成為歷史／旅程主題。
	motif := []byte{47, 45, 47, 48, 43, 48} // B2 A2 B2 C3 G2 C3
	for _, at := range []float64{8, 15, 22, 29, 36, 43, 49} {
		playMotif(s, 1, motif, at, 58)
	}

	// 第二幕：世界甦醒。四組小調和聲逐層加入。
	chords := [][]byte{
		{47, 50, 54}, // Bm
		{43, 47, 50}, // G
		{45, 49, 52}, // A
		{42, 46, 49}, // F#
	}
	for i, at := range []float64{8, 15, 22, 29, 36, 43, 50} {
		c := chords[i%len(chords)]
		s.chord(4, c, byte(42+i*3), at, 6.7)
		s.note(0, c[0]-12, byte(52+i*3), at, 6.7)
	}
	for at := 22.0; at < 42; at += 1.5 {
		drum(s, 41, 40, at, .10)
	}
	for at := 36.0; at < 57; at += .75 {
		vel := byte(48 + int(at-36)/2)
		drum(s, 36, vel, at, .10)
		if int((at-36)/.75)%4 == 2 {
			drum(s, 45, vel-8, at, .10)
		}
	}

	// 戰火逼近：弦樂固定音型與號角短句。
	ost := []byte{47, 54, 50, 54, 43, 50, 47, 50}
	for at := 42.0; at < 57; at += 4 {
		for i, p := range ost {
			s.note(4, p, 66, at+float64(i)*.5, .38)
		}
	}
	for _, at := range []float64{45, 51, 55} {
		s.note(2, 47, 68, at, .65)
		s.note(2, 50, 62, at+.75, .65)
		s.note(2, 54, 72, at+1.5, 1.1)
	}

	// 57–60 秒：決斷前抽空，只留心跳與低頻。
	s.note(0, 23, 48, 57, 3) // B0
	drum(s, 36, 62, 57.4, .14)
	drum(s, 36, 72, 59.0, .14)

	// 60–66 秒：完整回答。銅管用放大的原作輪廓，64 秒為最大命中重音。
	climax := []struct {
		p byte
		t float64
		d float64
	}{
		{59, 60, .72}, {57, 60.8, .72}, {59, 61.6, .72},
		{60, 62.4, .72}, {55, 63.2, .72}, {60, 64, 1.85},
	}
	for _, n := range climax {
		s.note(2, n.p, 108, n.t, n.d)
		s.note(3, n.p-12, 82, n.t, n.d)
	}
	s.chord(4, []byte{47, 50, 54, 59}, 92, 60, 4)
	s.chord(4, []byte{48, 52, 55, 60}, 100, 64, 2)
	for at := 60.0; at < 66; at += .5 {
		drum(s, 36, 90, at, .12)
		if int((at-60)*2)%2 == 1 {
			drum(s, 49, 72, at, .18)
		}
	}
	drum(s, 57, 118, 64, .35)
	drum(s, 49, 112, 64, .35)

	// 片尾不是大調勝利：C 與 B 並存，留下故事尚未終結的懸念。
	s.chord(3, []byte{48, 55, 59}, 62, 66, 5.4)
	s.chord(2, []byte{36, 43, 47}, 54, 66, 5.4)
	s.note(0, 35, 44, 66, 5.8)
	s.note(1, 60, 58, 70.7, .75)
}

func playMotif(s *score, ch byte, notes []byte, start float64, velocity byte) {
	offsets := []float64{0, 1.2, 2.0, 2.8, 3.6, 4.5}
	lengths := []float64{.9, .55, .55, .55, .65, 1.5}
	for i, p := range notes {
		s.note(ch, p, velocity, start+offsets[i], lengths[i])
	}
}

func drum(s *score, pitch, velocity byte, at, duration float64) {
	s.note(9, pitch, velocity, at, duration)
}

func midi(events []event) []byte {
	events = append(events,
		event{tick: 0, priority: -2, data: []byte{0xff, 0x51, 0x03,
			byte((microseconds >> 16) & 0xff), byte((microseconds >> 8) & 0xff),
			byte(microseconds & 0xff)}},
		event{tick: totalSeconds * ticksPerSec, priority: 9, data: []byte{0xff, 0x2f, 0x00}},
	)
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].tick != events[j].tick {
			return events[i].tick < events[j].tick
		}
		return events[i].priority < events[j].priority
	})
	var track bytes.Buffer
	last := 0
	for _, e := range events {
		track.Write(varlen(e.tick - last))
		track.Write(e.data)
		last = e.tick
	}
	var out bytes.Buffer
	out.WriteString("MThd")
	binary.Write(&out, binary.BigEndian, uint32(6))
	binary.Write(&out, binary.BigEndian, uint16(0))
	binary.Write(&out, binary.BigEndian, uint16(1))
	binary.Write(&out, binary.BigEndian, uint16(ppq))
	out.WriteString("MTrk")
	binary.Write(&out, binary.BigEndian, uint32(track.Len()))
	out.Write(track.Bytes())
	return out.Bytes()
}

func varlen(v int) []byte {
	buf := []byte{byte(v & 0x7f)}
	for v >>= 7; v > 0; v >>= 7 {
		buf = append(buf, byte(v&0x7f)|0x80)
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return buf
}

func dir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "dwscore:", err)
	os.Exit(1)
}
