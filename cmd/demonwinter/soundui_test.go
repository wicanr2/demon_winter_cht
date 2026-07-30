package main

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/audio/music"
)

type soundSpy struct {
	enabled      bool
	played       []int
	musicEnabled bool
	scene        music.Scene
}

func (s *soundSpy) Play(id int)                { s.played = append(s.played, id) }
func (s *soundSpy) SetEnabled(on bool)         { s.enabled = on }
func (s *soundSpy) Enabled() bool              { return s.enabled }
func (s *soundSpy) SetMusic(scene music.Scene) { s.scene = scene }
func (s *soundSpy) SetMusicEnabled(on bool)    { s.musicEnabled = on }
func (s *soundSpy) MusicEnabled() bool         { return s.musicEnabled }

func TestSoundOutputIsObservableWithoutAudioDevice(t *testing.T) {
	spy := &soundSpy{enabled: true}
	a := &app{speaker: spy}

	a.playSound(8)
	a.playSound(-1)
	if len(spy.played) != 2 || spy.played[0] != 8 || spy.played[1] != -1 {
		t.Fatalf("播放序列 = %v，預期 [8 -1]", spy.played)
	}
	if on := a.toggleSound(); on {
		t.Fatal("第一次切換後應關閉音效")
	}
	if on := a.toggleSound(); !on {
		t.Fatal("第二次切換後應開啟音效")
	}
}

func TestNilSoundOutputIsSafe(t *testing.T) {
	a := &app{}
	a.playSound(8)
	if a.toggleSound() {
		t.Fatal("沒有音效輸出時不應回報已開啟")
	}
}
