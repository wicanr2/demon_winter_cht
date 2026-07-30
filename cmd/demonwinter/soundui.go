package main

import "github.com/wicanr2/demon_winter_cht/internal/audio/music"

// soundOutput 是遊戲規則／介面與實際音效裝置之間的最小邊界。
//
// 正式執行使用 ui.Speaker；測試可注入 spy，不必開音效卡。這讓「哪個情境
// 送出哪個原版 effect id」成為可觀測行為，而不只測一支孤立的對照函式。
type soundOutput interface {
	Play(int)
	SetEnabled(bool)
	Enabled() bool
	SetMusic(music.Scene)
	SetMusicEnabled(bool)
	MusicEnabled() bool
}

func (a *app) syncMusic() {
	if a.speaker == nil {
		return
	}
	scene := music.Exploration
	switch {
	case a.death != nil || a.won:
		scene = music.Finale
	case a.battle != nil || a.sea != nil:
		scene = music.Battle
	case a.town != nil || a.camp != nil || a.merchant != nil || a.title != nil:
		scene = music.Sanctuary
	}
	a.speaker.SetMusic(scene)
}

func (a *app) toggleMusic() bool {
	if a.speaker == nil {
		return false
	}
	a.speaker.SetMusicEnabled(!a.speaker.MusicEnabled())
	return a.speaker.MusicEnabled()
}

func (a *app) playSound(id int) {
	if a.speaker != nil {
		a.speaker.Play(id)
	}
}

// toggleSound 切換原版的 Sound on／off，並回傳切換後狀態。
func (a *app) toggleSound() bool {
	if a.speaker == nil {
		return false
	}
	a.speaker.SetEnabled(!a.speaker.Enabled())
	return a.speaker.Enabled()
}
