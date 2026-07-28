package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 艾瑞戈爾的場景（`docs/re/83` §3）。
//
// 這是全遊戲的引爆點：黑鏡裡的馬利馮在艾瑞戈爾面前捏碎春之石，
// 冬天從此降臨。`EREGORE.TXT` 的 10 頁全部用在這一場。
//
// 畫面照夢境那邊的作法 —— 整頁蓋掉，一次一頁。原版的 `19d1(page, mode)`
// 本來就是整頁重畫，而且這幾段是主線最長的敘事，塞進走路畫面下方那個
// 小訊息框會被切成十幾次「按鍵繼續」。

// eregoreScreen 是這一場的播放狀態。
type eregoreScreen struct {
	// step 是目前這一步（要播哪些頁、播完怎麼走）。
	step game.EregoreStep
	// shown 是 step.Pages 裡已經播到第幾頁。
	shown int
}

// locationPlot 分派 `nSS.DAT` 類別 5 的地點劇情事件。
//
// case 編號是**全域唯一**的（1–15 分佈在五張子地圖，零碰撞，
// `docs/re/83` §2），所以不必再配地圖編號。
//
// **16 格全部接上了**（`docs/re/65`／`83`／`98`–`102`）。
//
// 這裡明確報未接，不要靜默什麼都不做，那會讓人以為那一格本來就沒事。
func (a *app) locationPlot(c int) {
	switch c {
	case plotCaseMachinery:
		a.resetCrushingWalls()
	case plotCaseCrush:
		a.advanceCrushingWalls()
	case plotCaseBlacksmith:
		a.openBlacksmith()
	case plotCaseArmory:
		// 四座台座共用一個 case，靠座標算出是哪一件。
		a.openArmory(a.party.X(), a.party.Y())
	case plotCaseTombstones:
		a.shiftTombstones()
	case plotCaseBell:
		a.ringBell()
	case plotCaseNpcBed:
		a.sleepAtNpc()
	case plotCaseDemonCrystal:
		a.openDemonCrystal()
	case plotCaseProving:
		a.enterProvingRoom()
	case plotCaseOrb:
		a.openOrb()
	case plotCaseWorkshop:
		a.openWorkshop()

	case game.RiddleCaseSpectralPriest, game.RiddleCaseTempleName:
		a.openRiddle(c)

	case game.CircleOfLightCase:
		a.circleOfLightDoor()

	case scenario.PlotCaseEregore:
		if a.save.ShardShattered != 0 {
			// 談過了。原版此時回 3，等於這一格沒反應（`0x1a55f`）。
			return
		}
		a.openEregore(a.save.EregoreMet == 1)
	default:
		a.message = fmt.Sprintf(a.tr.UI("eregore.plot.unhandled", "（地點劇情 %d 還沒接，見 docs/re/65）"), c)
	}
}

// openEregore 開場。met 是隊伍欄位 `+0x99`（上次談崩打過一架）。
func (a *app) openEregore(met bool) {
	if a.eregoreText == nil {
		// 讀不到文字就別擋著 —— 旗標照樣推進，畫面缺一段總比卡在原地好。
		a.finishEregore(game.EregoreStep{Outcome: game.EregoreDone})
		return
	}
	a.eregore = &eregoreScreen{step: game.StartEregore(met)}
}

func (a *app) updateEregore() error {
	e := a.eregore

	// 沒有頁要播（頁 5 回答「1」那條原版什麼都不印）→ 直接收場。
	if len(e.step.Pages) == 0 {
		a.eregore = nil
		a.finishEregore(e.step)
		return nil
	}

	// 還沒翻到最後一頁 —— 任意鍵翻頁。
	if e.shown < len(e.step.Pages)-1 {
		if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
			e.shown++
		}
		return nil
	}

	// **停在最後一頁**：發問的那一頁要一邊顯示一邊等答案，
	// 不能先按一次鍵把它翻掉再問（原版就是同一畫面上直接輸入）。
	if e.step.Outcome != game.EregoreAsk {
		if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
			a.eregore = nil
			a.finishEregore(e.step)
		}
		return nil
	}

	// 等一個合法的選項。原版是 `do {} while` 迴圈，按錯就繼續等，
	// 不會退出也不會亂跳（`0x1b32b`）。
	for i := 1; i <= e.step.Choices; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i-1)) {
			e.step = game.EregoreAnswer(e.step.Next, i)
			e.shown = 0
			return nil
		}
	}
	return nil
}

// finishEregore 套用這一場的結果。
//
// 兩個旗標各有各的意思，不能合併：
//
//   - EregoreFight → `+0x99 = 1`（見過面）＋ 三隻怪的戰鬥。
//     下次再來就跳過所有問答直接播結尾。
//   - EregoreDone  → `+0xbe = 1`（碎片已碎）。這一格從此不再觸發，
//     而且**世界跟著壞掉** —— 城鎮變廢墟、地圖 tile 被替換。
func (a *app) finishEregore(step game.EregoreStep) {
	switch step.Outcome {
	case game.EregoreFight:
		a.save.EregoreMet = 1
		a.startBattle(game.EregoreMonsters)
	case game.EregoreDone:
		a.save.ShardShattered = 1
		// 城鎮變廢墟是同一個旗標的下游效果，畫面要跟著換。
		a.drawTiles = ditheredTiles(a.tiles, uint16(a.ditherSeed), a.save.TempleRuins)
		a.message = a.tr.UI("eregore.shattered", "春之石碎了。冬天開始了。")
	}
}

func (a *app) drawEregore(dst *ebiten.Image) {
	e := a.eregore
	y := ui.LineHeight
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX*2, y)
		y += ui.LineHeight
	}

	if len(e.step.Pages) == 0 {
		return
	}
	page := e.step.Pages[e.shown]
	for _, l := range a.storyPage("EREGORE", page, a.eregoreText.Page(page)) {
		line(l)
	}
	line("　")

	// 選項的文字就寫在那一頁裡（原版頁 2 的 "1) …" "2) …" 是內文的一部分），
	// 所以這裡只提示按哪些鍵，不重印選項。
	if e.shown == len(e.step.Pages)-1 && e.step.Outcome == game.EregoreAsk {
		if e.step.Choices == 2 {
			line(a.tr.UI("eregore.choice.two", "　按 1 或 2"))
			return
		}
		line(a.tr.UI("eregore.choice.three", "　按 1、2 或 3"))
		return
	}
	line(a.tr.UI("ending.press", "（按任意鍵）"))
}

// loadEregoreText 讀艾瑞戈爾那一場的文字。
func loadEregoreText(dir string) *scenario.StoryText {
	return loadStoryOrNil(dir, scenario.StoryEregore)
}
