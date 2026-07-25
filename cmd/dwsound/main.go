// Command dwsound 把原版的 PC speaker 音效輸出成 WAV，供用耳朵驗收。
//
// 音效沒有「截圖」這種東西。頻率表對不對可以寫測試，
// 但「聽起來像不像 1988 年那台 PC」只能真的播出來聽。
//
// 用法：
//
//	dwsound -out workplace/sound
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/demon_winter_cht/internal/audio/pcspeaker"
)

// effects 是要輸出的效果，附上反組譯查到的使用情境。
var effects = []struct {
	id   int
	name string
	note string
}{
	{pcspeaker.EffectDeath, "death", "單位陣亡（唯一的多音符旋律，約 1.5 秒）"},
	{pcspeaker.EffectC3, "c3", "攻擊未命中（近戰）"},
	{pcspeaker.EffectD3, "d3", "音階 D3"},
	{pcspeaker.EffectE3, "e3", "音階 E3"},
	{pcspeaker.EffectF3, "f3", "攻擊未命中（遠端）"},
	{pcspeaker.EffectG3, "g3", "攻擊命中（近戰）"},
	{pcspeaker.EffectA3, "a3", "音階 A3"},
	{pcspeaker.EffectB3, "b3", "音階 B3"},
	{pcspeaker.EffectC4, "c4", "攻擊命中（遠端）／扣血提示"},
}

func main() {
	outDir := flag.String("out", "workplace/sound", "WAV 輸出目錄")
	rate := flag.Int("rate", 44100, "取樣率")
	volume := flag.Float64("volume", 0.5, "音量 0–1")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	for _, e := range effects {
		notes := pcspeaker.Effect(e.id)
		if len(notes) == 0 {
			continue
		}
		pcm := pcspeaker.Render(notes, *rate, *volume)
		path := filepath.Join(*outDir, fmt.Sprintf("%02d-%s.wav", e.id+1, e.name))
		if err := writeWAV(path, pcm, *rate); err != nil {
			fatal(err)
		}
		fmt.Printf("%-24s %5.2f 秒　%s\n", filepath.Base(path),
			pcspeaker.Duration(notes), e.note)
	}
	fmt.Printf("\n輸出到 %s\n", *outDir)
}

// writeWAV 寫出 16-bit 單聲道 WAV。
func writeWAV(path string, pcm []byte, sampleRate int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	const (
		channels      = 1
		bitsPerSample = 16
	)
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8

	w := func(v any) {
		if err == nil {
			err = binary.Write(f, binary.LittleEndian, v)
		}
	}
	f.WriteString("RIFF")
	w(uint32(36 + len(pcm)))
	f.WriteString("WAVEfmt ")
	w(uint32(16))            // fmt 區塊長度
	w(uint16(1))             // PCM
	w(uint16(channels))      //
	w(uint32(sampleRate))    //
	w(uint32(byteRate))      //
	w(uint16(blockAlign))    //
	w(uint16(bitsPerSample)) //
	f.WriteString("data")
	w(uint32(len(pcm)))
	if err != nil {
		return err
	}
	_, err = f.Write(pcm)
	return err
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "dwsound:", err)
	os.Exit(1)
}
