package main

import (
	"fmt"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// createStage 是建角流程的階段。
type createStage int

const (
	stageRace createStage = iota
	stageTraits
	stageClass
	stageName
	stageSlot
)

// createScreen 是建角畫面的狀態。
type createScreen struct {
	stage   createStage
	cursor  int
	create  *game.CharacterCreation
	name    []rune
	class   gamedata.Class
	message string

	// picked 是這一輪重擲要重擲哪幾項。
	// 英文手冊寫「該屬性（或多項屬性）」—— 可以一次勾好幾項再一起擲。
	picked [gamedata.NumTraits]bool
}

// traitNames 是五項屬性的顯示名稱，順序同 gamedata.Trait。
var traitNames = []string{"速度", "力量", "智力", "耐力", "技巧"}

// digitKeys 是 1–9 的按鍵，用來選種族／職業／屬性編號。
var digitKeys = []ebiten.Key{
	ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4,
	ebiten.KeyDigit5, ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8,
	ebiten.KeyDigit9,
}

// pressedDigit 回傳這一幀按下的數字鍵（1–9 → 0–8），沒有則回 -1。
func pressedDigit() int {
	for i, k := range digitKeys {
		if inpututil.IsKeyJustPressed(k) {
			return i
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit0) {
		return 9
	}
	return -1
}

func (a *app) openCreate() {
	a.create = &createScreen{stage: stageRace}
}

// openNewGameParty 開始「開新遊戲要建滿五個角色」的流程。
//
// 原版的建角在遊戲外（`Character Utilities`），玩家必須先造好隊伍
// 才有存檔可以載。本作沒有那支獨立程式，所以把它做成
// **新遊戲開場的一段強制流程**：五個槽位都建過才走得動。
//
// 為什麼要強制：`-newgame` 只改隊伍共用欄位（位置、時間、金幣……），
// 角色仍是出貨存檔那五個。不強制的話玩家會拿著**別人玩過的角色**
// 從正確的起點開始 —— 而畫面上看不出那五個人不是自己建的
//（`docs/re/87` §6）。
func (a *app) openNewGameParty() {
	a.newGameSlots = len(a.members)
	a.createSlot = 0
	a.openCreate()
}

// newGamePending 回報還在建角流程中。
//
// 這期間**擋掉世界地圖的輸入** —— 不擋的話玩家可以按 ESC 溜出去，
// 帶著半套自建、半套出貨的隊伍上路，那是最難察覺的一種壞狀態。
func (a *app) newGamePending() bool { return a.newGameSlots > 0 }

func (a *app) updateCreate() error {
	c := a.create

	// ESC 一律只退回上一階段，退到底就關掉建角畫面。離開遊戲走 F10。
	//
	// **新遊戲的強制流程例外**：退到底也不關 —— 五個角色沒建完就沒有隊伍，
	// 關掉只會讓玩家卡在一個沒有隊伍的世界地圖上。
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if c.stage == stageRace {
			if a.newGamePending() {
				c.message = fmt.Sprintf("開新遊戲要建滿 %d 個角色（還剩 %d 個）",
					len(a.members), a.newGameSlots)
				return nil
			}
			a.create = nil
			return nil
		}
		c.stage--
		c.message = ""
		return nil
	}

	switch c.stage {
	case stageRace:
		if d := pressedDigit(); d >= 0 && d < len(raceName) {
			cr, err := game.NewCharacterCreation(a.rng, a.tables, gamedata.Race(d))
			if err != nil {
				c.message = fmt.Sprintf("擲點失敗：%v", err)
				return nil
			}
			c.create = cr
			c.picked = [gamedata.NumTraits]bool{}
			c.stage = stageTraits
		}

	case stageTraits:
		// 數字鍵勾選要重擲的屬性，可複選。
		if d := pressedDigit(); d >= 0 && d < gamedata.NumTraits {
			c.picked[d] = !c.picked[d]
		}
		// 原版是「選好之後按 ESC 重擲」，但本專案的 ESC 一律是「退回上一層」，
		// 兩者衝突時以不丟進度的那條為準（見 esc-cancel-f10-quit-autosave）。
		// 改用 R，畫面上有標。
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			var which []gamedata.Trait
			for i, on := range c.picked {
				if on {
					which = append(which, gamedata.Trait(i))
				}
			}
			if len(which) == 0 {
				c.message = "先用數字鍵勾選要重擲的屬性"
				return nil
			}
			if err := c.create.Reroll(which); err != nil {
				c.message = err.Error()
				return nil
			}
			c.picked = [gamedata.NumTraits]bool{}
			c.message = ""
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			c.stage = stageClass
			c.cursor = 0
		}

	case stageClass:
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			c.cursor = (c.cursor + 1) % len(className)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			c.cursor = (c.cursor - 1 + len(className)) % len(className)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			c.class = gamedata.Class(c.cursor)
			c.stage = stageName
			c.name = nil
		}

	case stageName:
		a.editName(c)
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(c.name) > 0 {
			c.stage = stageSlot
			// 強制流程裡游標直接指到該建的那一個 —— 讓玩家一路按 Enter
			// 就能照順序建滿，不必自己記「剛才建到第幾個」。
			c.cursor = 0
			if a.newGamePending() {
				c.cursor = a.createSlot
			}
		}

	case stageSlot:
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			c.cursor = (c.cursor + 1) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			c.cursor = (c.cursor - 1 + len(a.members)) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			a.members[c.cursor] = c.create.Finish(string(c.name), c.class)
			a.message = fmt.Sprintf("%s 已加入隊伍第 %d 位（按 S 存檔才會留下）",
				string(c.name), c.cursor+1)
			a.create = nil
			a.advanceNewGameParty()
		}
	}
	return nil
}

// editName 處理姓名輸入。只收 ASCII —— 存檔的姓名欄是 12 bytes 的
// NUL 結尾字串，塞中文會在 12 bytes 內只放得下五個字，而且原版讀不回來。
func (a *app) editName(c *createScreen) {
	const maxName = 11 // 12 bytes 扣掉結尾的 NUL

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(c.name) > 0 {
		c.name = c.name[:len(c.name)-1]
	}
	for _, r := range ebiten.AppendInputChars(nil) {
		if r < 0x20 || r > 0x7e || len(c.name) >= maxName {
			continue
		}
		c.name = append(c.name, r)
	}
}

func (a *app) drawCreate(dst *ebiten.Image) {
	c := a.create
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	line("建立角色")
	line("")

	switch c.stage {
	case stageRace:
		line("選擇種族：")
		for i, n := range raceName {
			line(fmt.Sprintf("  %d  %s", i+1, n))
		}
		line("")
		line("數字鍵：選擇　Esc：取消")

	case stageTraits:
		a.drawTraitStage(c, line)

	case stageClass:
		line("選擇職業：")
		for i, n := range className {
			mark := "   "
			if i == c.cursor {
				mark = " > "
			}
			line(fmt.Sprintf("%s%s", mark, n))
		}
		line("")
		line("↑↓：選擇　Enter：確定　Esc：回上一步")

	case stageName:
		line(fmt.Sprintf("%s %s", nameOf(raceName, int(c.create.Race)),
			nameOf(className, int(c.class))))
		line("")
		line(fmt.Sprintf("姓名：%s_", string(c.name)))
		line("")
		line("※ 姓名只收英數字：存檔的姓名欄是 12 bytes，")
		line("　 中文塞不下，原版也讀不回來")
		line("")
		line("Enter：確定　Backspace：刪除　Esc：回上一步")

	case stageSlot:
		line("放進隊伍第幾位？（會覆蓋原本的角色）")
		line("")
		for i, m := range a.members {
			mark := "   "
			if i == c.cursor {
				mark = " > "
			}
			line(fmt.Sprintf("%s%d  %-10s %d 級 %s", mark, i+1, m.Name, m.Level,
				nameOf(className, int(m.Class))))
		}
		line("")
		line("↑↓：選擇　Enter：確定　Esc：回上一步")
	}

	if c.message != "" {
		line("")
		line(c.message)
	}
}

func (a *app) drawTraitStage(c *createScreen, line func(string)) {
	line(fmt.Sprintf("種族：%s　剩餘重擲 %d 次",
		nameOf(raceName, int(c.create.Race)), c.create.RerollsLeft()))
	line("")
	// 表頭與資料列用同一組欄寬產生 —— 手動數空格對齊，改一次格式就歪一次。
	// 混排字型每個字元都是一格，所以 Go 的 %-4s（按 rune 計）剛好等於 4 格。
	const traitRow = "%s%s %-4s %4s %10s%s"
	line(fmt.Sprintf(traitRow, "   ", " ", "屬性", "擲出", "該種族平均", ""))

	for i, n := range traitNames {
		tr := gamedata.Trait(i)
		avg, err := c.create.RaceAverage(tr)
		if err != nil {
			continue
		}
		mark := "   "
		if c.picked[i] {
			mark = " * "
		}
		// 手冊建議低於 6 就重擲。標出來，但不擋玩家重擲比較高的值。
		advice := ""
		if c.create.BelowAdvice(tr) {
			advice = "  ← 建議重擲"
		}
		line(fmt.Sprintf(traitRow, mark, strconv.Itoa(i+1), n,
			strconv.Itoa(c.create.Traits[i]), strconv.Itoa(avg), advice))
	}

	line("")
	line("數字鍵：勾選要重擲的屬性（可複選）")
	line("R：重擲勾選的項目　Enter：接受並選職業")
	line("")
	line("※ 原版是選好按 ESC 重擲；本作 ESC 一律是「退回上一步」，")
	line("　 所以改用 R，免得按錯就退出建角")
}

// advanceNewGameParty 在強制流程裡接著建下一個角色。
func (a *app) advanceNewGameParty() {
	if !a.newGamePending() {
		return
	}
	a.newGameSlots--
	if a.newGameSlots == 0 {
		a.message = "隊伍組好了。按 S 存檔。"
		return
	}
	a.createSlot++
	a.openCreate()
}
