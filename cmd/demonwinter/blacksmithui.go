package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 鐵匠鋪（地點劇情 case 4，原版 `0x1a0f9`）。
//
// 老巨魔鐵匠打了一把「為了要殺 Xeres 的人」準備的武器，白送。
// 一輪遊戲只有一次 —— 閘門是 trailer `+0xb7`，也就是劇情道具旗標陣列
// （`+0xb3`）的第 4 格（`docs/re/65` §3.2）。
//
// 原版的順序：印場景敘述 → 等 `Continue` → 印三句台詞 → 等 `Continue`
// → 問「哪個角色來拿」→ 塞進第一個空槽。這裡把前兩段併成一個文字框
// （本專案的文字框本來就會分頁），選人另開一頁。

// blacksmithScreen 是「哪個角色來拿」那一頁。
type blacksmithScreen struct{ cursor int }

// openBlacksmith 是 case 4 的入口。
func (a *app) openBlacksmith() {
	if game.PlotGiftTaken(a.save, game.PlotGiftBlacksmith) {
		// 拿過了就整格沒反應 —— 原版 `0x1a105` 直接回 2。
		return
	}
	a.box = ui.NewMixedTextBox(a.tr.UI("blacksmith.scene",
		"你走進一間鐵匠鋪。鐵匠是個年邁的巨魔，一邊的眼睛永遠閉著，"+
			"臉孔扭曲得嚇人。他先是一驚，隨即咧嘴笑了起來。")+
		"\n\n" +
		a.tr.UI("blacksmith.line1", "「我一直在打一把新武器。") + "\n" +
		a.tr.UI("blacksmith.line2", "　它是為了那些想殺掉 Xeres 的人") + "\n" +
		a.tr.UI("blacksmith.line3", "　準備的！」"))
	a.blacksmith = &blacksmithScreen{}
	a.trace.note("鐵匠鋪：開場")
}

func (a *app) updateBlacksmith() error {
	b := a.blacksmith
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		// **不給旗標。** 走開就還能再回來拿，原版的閘門只在成功時才設。
		a.blacksmith = nil
		a.trace.note("鐵匠鋪：沒拿就走了")
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		b.cursor = (b.cursor + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		b.cursor = (b.cursor - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.takeBlacksmithGift(b.cursor)
	}
	return nil
}

func (a *app) takeBlacksmithGift(member int) {
	if member < 0 || member >= len(a.members) {
		return
	}
	res := game.TakePlotGift(a.save, &a.members[member], game.PlotGiftBlacksmith)
	if res.Full {
		// 欄位滿了 → 旗標沒動，等一下還能再拿。畫面留著。
		a.message = a.tr.UI("dungeon.noroom", "放不下了")
		a.trace.note("鐵匠鋪：%s 道具欄滿了", a.members[member].Name)
		return
	}
	a.blacksmith = nil
	if !res.OK {
		return
	}
	a.message = fmt.Sprintf(a.tr.UI("blacksmith.taken", "%s 收下了那把武器"),
		a.members[member].Name)
	a.trace.note("鐵匠鋪：%s 第 %d 格拿到武器", a.members[member].Name, res.Slot)
}

func (a *app) drawBlacksmith(dst *ebiten.Image) {
	b := a.blacksmith
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	line(a.tr.UI("blacksmith.who", "誰來收下？"))
	line("")
	for i := range a.members {
		c := &a.members[i]
		line(fmt.Sprintf("%s%s　空 %d 格",
			memberMark(b.cursor, i), c.Name, freeSlots(c)))
	}
	line("")
	line(a.tr.UI("dungeon.keys", "↑↓：選擇　Enter：確定　Esc：返回"))
}
