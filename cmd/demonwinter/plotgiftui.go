package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 劇情送道具的畫面 —— 鐵匠鋪（case 4）與兵器庫（case 3）共用
// （原版也共用 `25be:11ff` 那一支，`docs/re/65` §3.2、`docs/re/99`）。
//
// 兩者的差別只在**開場**：鐵匠先播三句台詞就直接選人；
// 兵器庫多一道「你要靠近嗎？」的是非題，答不就什麼都不會發生。
// 選人與塞道具那一段一模一樣，所以只留一份。

// plotGiftScreen 是「哪個角色來拿」那一頁。
type plotGiftScreen struct {
	// id 是要送的那件道具，同時是一次性旗標的索引。
	id game.PlotGiftID
	// ask 為真時停在是非題那一頁（兵器庫），還沒進到選人。
	ask bool
	// cursor 是選人游標。
	cursor int
}

// openBlacksmith 是地點劇情 case 4 的入口。
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
	a.plotGift = &plotGiftScreen{id: game.PlotGiftBlacksmith}
	a.trace.note("鐵匠鋪：開場")
}

// armoryItemName 是四座台座上那件道具的名字（`ds:0x27cc` 的四個遠指標）。
//
// 原版把它嵌進一句 `%s%s%s`：前綴（場景敘述）＋ 這個名字 ＋ 後綴（球體那句）。
// 譯名對齊 `docs/walkthrough/part-4.md` §3.2 列的那四樣
// （釘頭鎚／銀鏈／水晶匕首／短劍）—— 玩家是拿攻略對畫面的。
func (a *app) armoryItemName(id game.PlotGiftID) string {
	switch id {
	case game.PlotGiftArmoryChain:
		return a.tr.UI("armory.item.chain", "一件銀製鏈甲")
	case game.PlotGiftArmoryMace:
		return a.tr.UI("armory.item.mace", "一把釘頭鎚")
	case game.PlotGiftArmoryDagger:
		return a.tr.UI("armory.item.dagger", "一把水晶匕首")
	case game.PlotGiftArmorySword:
		return a.tr.UI("armory.item.sword", "一把冰藍色的短劍")
	}
	return ""
}

// openArmory 是地點劇情 case 3 的入口：四座台座之一。
//
// 原版先組出敘述（`0x1a08f` 的 `sprintf("%s%s%s")`）再問
// `Do you approach?`（`0x1a0cc`）；答不就直接回 2，**旗標不動**。
func (a *app) openArmory(x, y int) {
	id := game.ArmoryGiftFor(x, y)
	if game.PlotGiftTaken(a.save, id) {
		// 拿過了 —— 原版 `0x1a083` 檢查 `+0xb3 + idx`，不為 0 就回 2。
		return
	}
	a.box = ui.NewMixedTextBox(a.tr.UI("armory.scene",
		"你走進一間兵器庫，裡頭堆滿了陳舊的武器與甲冑。房間中央的台座上"+
			"放著一件特別的東西：")+
		a.armoryItemName(id)+
		a.tr.UI("armory.spheres",
			"。幾團發光的球體懸在台座上方，似乎很不歡迎你的到來。"))
	a.plotGift = &plotGiftScreen{id: id, ask: true}
	a.trace.note("兵器庫：(%d,%d) 台座 %d", x, y, id)
}

func (a *app) updatePlotGift() error {
	g := a.plotGift
	if g.ask {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyY):
			// 原版答是之後才印「球體亮了起來」（`0x1aa02`），
			// 那一句只有兵器庫這條路會印。
			g.ask = false
			a.message = a.tr.UI("armory.brighter", "球體的光亮了起來。")
			a.trace.note("兵器庫：靠近")
		case inpututil.IsKeyJustPressed(ebiten.KeyN),
			inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			// **旗標不動。** 走開還能再回來（原版答不只是 return 2）。
			a.plotGift = nil
			a.trace.note("兵器庫：沒靠近")
		}
		return nil
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		// **不給旗標。** 走開就還能再回來拿，原版的閘門只在成功時才設。
		a.plotGift = nil
		a.trace.note("劇情道具：沒拿就走了")
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		g.cursor = (g.cursor + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		g.cursor = (g.cursor - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.takePlotGift(g.cursor)
	}
	return nil
}

func (a *app) takePlotGift(member int) {
	g := a.plotGift
	if member < 0 || member >= len(a.members) {
		return
	}
	res := game.TakePlotGift(a.save, &a.members[member], g.id)
	if res.Full {
		// 欄位滿了 → 旗標沒動，等一下還能再拿。畫面留著。
		a.message = a.tr.UI("dungeon.noroom", "放不下了")
		a.trace.note("劇情道具：%s 道具欄滿了", a.members[member].Name)
		return
	}
	a.plotGift = nil
	if !res.OK {
		return
	}
	// 名字本身就帶量詞（「一把釘頭鎚」），所以中間不留空白。
	a.message = fmt.Sprintf(a.tr.UI("plotgift.taken", "%s 收下了%s"),
		a.members[member].Name, a.plotGiftLabel(g.id))
	a.trace.note("劇情道具：%s 第 %d 格拿到 %d 號", a.members[member].Name, res.Slot, g.id)
}

// plotGiftLabel 是訊息裡那件道具怎麼稱呼。
func (a *app) plotGiftLabel(id game.PlotGiftID) string {
	if id == game.PlotGiftBlacksmith {
		return a.tr.UI("blacksmith.sword", "那把武器")
	}
	return a.armoryItemName(id)
}

func (a *app) drawPlotGift(dst *ebiten.Image) {
	g := a.plotGift
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	if g.ask {
		line(a.tr.UI("armory.approach", "你要靠近嗎？"))
		line("")
		line(a.tr.UI("armory.keys", "Y：靠近　N／Esc：離開"))
		return
	}
	line(a.tr.UI("plotgift.who", "誰來收下？"))
	line("")
	for i := range a.members {
		c := &a.members[i]
		line(fmt.Sprintf("%s%s　空 %d 格",
			memberMark(g.cursor, i), c.Name, freeSlots(c)))
	}
	line("")
	line(a.tr.UI("dungeon.keys", "↑↓：選擇　Enter：確定　Esc：返回"))
}
