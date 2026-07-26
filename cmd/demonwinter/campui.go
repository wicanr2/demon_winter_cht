package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 紮營畫面。
//
// 原版的紮營選單有 13 個選項（見 `docs/re/26` §1）：
// Reorder／Sleep／Identify／Worship／Xorcise／View land／Trade／Drop／
// Equip／Use／Hunt／Cast／Quit。**這裡只做了規則已經解出來的那幾個** ——
// 睡覺與打獵。其餘標成「尚未實作」列出來，不假裝沒有。
//
// **進入鍵是本作自己選的 `R`**：原版用哪個鍵沒查（手冊那句「按 P 鍵」指的是
// 紮營畫面裡的另一個功能，不是進入鍵），而 `P` 在本作已經是隊伍名冊。
type campScreen struct {
	// message 是最近一次操作的結果。
	message string
	// hunter 是打獵選單的游標；-1 代表選單沒開。
	hunter int
}

// openCamp 進入紮營畫面。
func (a *app) openCamp() {
	a.camp = &campScreen{hunter: -1}
}

func (a *app) updateCamp() error {
	c := a.camp

	// 打獵要先選人。
	if c.hunter >= 0 {
		return a.updateHuntPicker()
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.camp = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyS):
		a.campSleep()
	case inpututil.IsKeyJustPressed(ebiten.KeyH):
		if len(a.members) == 0 {
			c.message = "隊伍是空的"
			return nil
		}
		c.hunter = 0
	}
	return nil
}

// campSleep 在野外睡一晚。
func (a *app) campSleep() {
	c := a.camp
	if !a.clock.CanSleep() {
		c.message = "現在睡不著（原版：You are restless）"
		return
	}
	rations := int(a.save.Rations)
	res := game.Rest(a.rng, game.RestCamp, a.members, a.clock, &rations)
	a.save.Rations = byte(rations)

	// 紮營會把光源重設成火把（原版 2aed:040c）。
	a.torch = game.RestCampTorch

	msg := fmt.Sprintf("睡了 %d 個時辰，%d 日 %d 時醒來",
		res.Hours, a.clock.Day(), a.clock.Hour())
	switch {
	case res.Starved:
		msg += "　沒有糧食，全隊受餓"
	case res.AteFood:
		msg += fmt.Sprintf("　吃掉一份糧食（剩 %d）", rations)
	}
	for _, i := range res.Died {
		msg += "　" + a.members[i].Name + " 沒有醒來"
	}
	c.message = msg
}

func (a *app) updateHuntPicker() error {
	c := a.camp
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		c.hunter = -1
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		c.hunter = (c.hunter + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		c.hunter = (c.hunter - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		rations := int(a.save.Rations)
		res := game.Hunt(a.rng, &a.members[c.hunter], &rations)
		a.save.Rations = byte(rations)
		c.hunter = -1

		switch {
		case res.Reason != "":
			c.message = res.Reason
		case res.Gained == 0:
			c.message = "這趟打獵一無所獲"
		default:
			c.message = fmt.Sprintf("打到 %d 份糧食（共 %d 份）", res.Gained, rations)
		}
	}
	return nil
}

func (a *app) drawCamp(dst *ebiten.Image) {
	c := a.camp
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	line(fmt.Sprintf("紮營　%d 月 %d 日 %d 時　糧食 %d 份",
		a.clock.Month(), a.clock.Day(), a.clock.Hour(), a.save.Rations))
	line("")

	if c.hunter >= 0 {
		line("誰去打獵？")
		line("")
		for i, m := range a.members {
			mark := "   "
			if i == c.hunter {
				mark = " > "
			}
			note := ""
			if !m.Skills[game.SkillHunting] {
				note = "（不會狩獵）"
			}
			line(fmt.Sprintf("%s%s%s", mark,
				textlayout.PadCells(m.Name, 10), note))
		}
		line("")
		line("↑↓：選擇　Enter：出發　Esc：取消")
		return
	}

	if a.clock.CanSleep() {
		line("  S 睡覺")
	} else {
		line("  S 睡覺（現在睡不著，要 15–24 時）")
	}
	line("  H 打獵")
	line("")
	line("Esc：收帳篷")
	line("")
	if c.message != "" {
		line(c.message)
		line("")
	}
	line("※ 原版的紮營選單有 13 項，這裡只做了規則已解出的")
	line("　 睡覺與打獵兩項，其餘見 docs/re/26")
}
