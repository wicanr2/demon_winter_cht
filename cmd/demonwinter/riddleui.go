package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 兩道密語謎題（`docs/re/84`）。
//
// 這是全遊戲僅有的自由文字輸入 —— 其餘互動都是選單。
// **輸入維持英文**：答案 `VOID` 與 `JESRIC` 是原版寫在資料段裡的明文，
// 中文化不能動它們，否則玩家照攻略打字會打不進去。
// 題目本身翻成中文，只有那一格輸入欄留 ASCII。

// riddleScreen 是一道謎題的作答狀態。
type riddleScreen struct {
	// plotCase 是這一題的 case 編號，決定答對之後的效果。
	plotCase int
	riddle   *game.Riddle
	// input 是玩家打到一半的答案。
	input []rune
	// result 非空時已經作答完畢，顯示結果等按鍵。
	result []string
	// done 為 true 時按任意鍵離開。
	done bool
}

// riddleKeys 是每一題在翻譯目錄裡的 key 前綴。
func riddleKeys(plotCase int) string {
	if plotCase == game.RiddleCaseSpectralPriest {
		return "riddle.priest"
	}
	return "riddle.temple"
}

// openRiddle 出題。
func (a *app) openRiddle(plotCase int) {
	r := game.RiddleFor(plotCase)
	if r == nil {
		return
	}
	a.riddle = &riddleScreen{plotCase: plotCase, riddle: r}
}

func (a *app) updateRiddle() error {
	s := a.riddle

	if s.done {
		if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
			a.riddle = nil
		}
		return nil
	}

	// ESC 只是離開作答，不是離開遊戲 —— 謎題可以再走回來重試
	// （原版也是：兩題都沒有「答錯就永久失敗」的旗標）。
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.riddle = nil
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		a.answerRiddle(s)
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.input) > 0 {
		s.input = s.input[:len(s.input)-1]
	}
	// 只收可見 ASCII。原版的讀字串函式也只吃這些，而且答案本來就是英文。
	for _, r := range ebiten.AppendInputChars(nil) {
		if r < 0x20 || r > 0x7e || len(s.input) >= s.riddle.MaxLen {
			continue
		}
		s.input = append(s.input, r)
	}
	return nil
}

// answerRiddle 判定答案並套用效果。
func (a *app) answerRiddle(s *riddleScreen) {
	k := riddleKeys(s.plotCase)
	s.done = true

	if !s.riddle.Correct(string(s.input)) {
		switch s.plotCase {
		case game.RiddleCaseSpectralPriest:
			s.result = []string{a.tr.UI(k+".wrong", s.riddle.Wrong[0])}
		case game.RiddleCaseTempleName:
			s.result = []string{
				a.tr.UI(k+".wrong0", s.riddle.Wrong[0]),
				a.tr.UI(k+".wrong1", s.riddle.Wrong[1]),
			}
			// 答錯神殿那一題會被推回一格（原版 `party+0xa2 = 0x26`）。
			a.party.TeleportTo(a.party.X(), game.TempleRejectY)
		}
		return
	}

	switch s.plotCase {
	case game.RiddleCaseSpectralPriest:
		// 答對沒有台詞，只有把牆打開。**改的是記憶體裡的地圖** ——
		// 原版就是如此，離開地城再進來牆會回到原狀。
		d := game.SpectralPriestDoor
		if err := a.tiles.SetTileAt(d.X, d.Y, game.SpectralPriestDoorOpen); err == nil {
			a.drawTiles = ditheredTiles(a.tiles, uint16(a.ditherSeed), a.save.TempleRuins)
		}
	case game.RiddleCaseTempleName:
		s.result = []string{a.tr.UI(k+".right", s.riddle.Right[0])}
	}
}

func (a *app) drawRiddle(dst *ebiten.Image) {
	s := a.riddle
	y := ui.LineHeight
	line := func(t string) {
		a.font.Draw(dst, t, layout.BoxPadX*2, y)
		y += ui.LineHeight
	}

	k := riddleKeys(s.plotCase)
	for i, orig := range s.riddle.Prompt {
		line(a.tr.UI(promptKey(k, i), orig))
	}
	line("　")

	if s.done {
		for _, t := range s.result {
			line(t)
		}
		line("　")
		line("（按任意鍵）")
		return
	}

	line(a.tr.UI("riddle.answer", "答案（英文，按 Enter 送出）：") + string(s.input) + "_")
	line("　")
	line(a.tr.UI("riddle.cancel", "Esc：離開"))
}

// promptKey 是題目第 i 行的翻譯 key。
func promptKey(prefix string, i int) string {
	return prefix + "." + string(rune('0'+i))
}
