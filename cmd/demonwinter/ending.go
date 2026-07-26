package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 結局畫面（原版 `1000:3575` ＝ `0x07175`，見 `docs/re/61` §2）。
//
// 原版那支 853 bytes 的函式在 IMPRISON 成功後被呼叫，裡面的字串是：
//
//	ds:0x066a  "CONGRATULATIONS! You have won Demon's Winter.
//	            I hope you have enjoyed your adventure."
//	0x7321     "%s followed in the footsteps of %s"
//	0x73c4     "%s %s"
//
// 那兩個 `%s` 的格式字串**還沒讀出參數是什麼**（推測是隊員名與某個
// 傳承對象），所以這裡只呈現確定的那一段祝賀詞 ——
// 寧可少一行，不要編一個原版沒有的結局敘述。

// updateEnding 等玩家按鍵。結局是終點，按任意鍵離開遊戲。
func (a *app) updateEnding() error {
	if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
		return ebiten.Termination
	}
	return nil
}

// drawEnding 畫破關畫面。
func (a *app) drawEnding(dst *ebiten.Image) {
	y := layout.CanvasHeight/2 - ui.LineHeight*3
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX*4, y)
		y += ui.LineHeight
	}

	line("恭喜！")
	line("")
	line("你通關了《冬之魔》。")
	line("")
	line("惡魔已被禁錮，漫長的冬天終於要過去了。")
	line("希望這趟旅程沒有辜負你的時間。")
	line("")
	line("")
	line("按任意鍵離開")
}
