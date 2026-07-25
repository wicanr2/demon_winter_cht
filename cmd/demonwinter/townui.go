package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// townScreen 是城鎮畫面的狀態。
type townScreen struct {
	visit *game.TownVisit

	// facility 為 nil 代表停在設施選單，否則進到某個設施內。
	facility *game.Facility

	// picking 為真代表正在選要進哪座城鎮。
	//
	// **這是暫時的入口。** 原版靠「走到世界地圖的城鎮格」進城，
	// 但「座標 → 城鎮編號」的對照還沒解出來（見 docs/spec/08-town-economy.md
	// 未解表）。在那之前用選單讓玩家進得去，功能才驗得到。
	picking bool
	cursor  int

	// message 是最近一次操作的結果。
	message string
}

// facilityKeys 是設施的熱鍵，順序同 game.AllFacilities。
var facilityKeys = []ebiten.Key{
	ebiten.KeyM, ebiten.KeyH, ebiten.KeyI,
	ebiten.KeyG, ebiten.KeyC, ebiten.KeyD, ebiten.KeyB,
}

// openTownPicker 打開城鎮選單。
func (a *app) openTownPicker() {
	a.town = &townScreen{picking: true}
}

func (a *app) updateTown() error {
	t := a.town

	if t.picking {
		return a.updateTownPicker(t)
	}
	if t.facility != nil {
		return a.updateFacility(t)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.town = nil
		a.message = fmt.Sprintf("離開%s", t.visit.Town.Name)
		return nil
	}
	for i, key := range facilityKeys {
		if !inpututil.IsKeyJustPressed(key) {
			continue
		}
		f := game.AllFacilities[i]
		t.facility = &f
		t.message = ""
		return nil
	}
	return nil
}

func (a *app) updateTownPicker(t *townScreen) error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		t.cursor = (t.cursor + 1) % a.towns.Len()
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		t.cursor = (t.cursor - 1 + a.towns.Len()) % a.towns.Len()
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.town = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter),
		inpututil.IsKeyJustPressed(ebiten.KeySpace):
		town, err := a.towns.ByNumber(t.cursor + 1)
		if err != nil {
			return nil
		}
		t.visit = game.EnterTown(town, a.members)
		t.picking = false
	}
	return nil
}

// updateFacility 處理設施內的操作。
func (a *app) updateFacility(t *townScreen) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		t.facility = nil
		t.message = ""
		return nil
	}

	switch *t.facility {
	case game.FacilityDocks:
		if inpututil.IsKeyJustPressed(ebiten.KeyB) {
			if !t.visit.HasDocks() {
				t.message = "這裡沒有船可賣"
				return nil
			}
			t.message = fmt.Sprintf("一艘船要 %d 金幣", t.visit.Economy.ShipPrice())
		}
	case game.FacilityMarket:
		// 議價：對游標指的商品談一次價。
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			t.cursor = (t.cursor + 1) % a.items.Len()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			t.cursor = (t.cursor - 1 + a.items.Len()) % a.items.Len()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyH) {
			a.haggleCurrent(t)
		}
	}
	return nil
}

// haggleCurrent 對游標指的商品議價一次。
func (a *app) haggleCurrent(t *townScreen) {
	s := t.visit.HaggleState(t.cursor)
	if s.Refused() {
		t.message = "商人不肯再賣這件給你"
		return
	}

	out, next := game.Haggle(a.rng, s)
	t.visit.SetHaggleState(t.cursor, next)
	switch out {
	case game.HaggleSuccess:
		t.message = "商人讓步了"
	case game.HaggleUnmoved:
		// 這一步是死路：s 之後 >= 100，下次議價必定觸怒對方。
		t.message = "商人不為所動（再談就會惹惱他）"
	case game.HaggleOffended:
		t.message = "你惹惱了商人，這件他不賣了"
	}
}

func (a *app) drawTown(dst *ebiten.Image) {
	t := a.town
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	switch {
	case t.picking:
		a.drawTownPicker(dst, line)
	case t.facility != nil:
		a.drawFacility(dst, line)
	default:
		a.drawTownMenu(dst, line)
	}
}

func (a *app) drawTownPicker(dst *ebiten.Image, line func(string)) {
	t := a.town
	line("進入哪座城鎮？")
	line("（原版是走到城鎮格進城，座標對照尚未解出）")
	line("")

	// 一次只列游標附近幾座，畫布放不下 25 座。
	const window = 12
	start := t.cursor - window/2
	if start < 0 {
		start = 0
	}
	if start+window > a.towns.Len() {
		start = a.towns.Len() - window
	}
	for i := start; i < start+window && i < a.towns.Len(); i++ {
		town, err := a.towns.ByNumber(i + 1)
		if err != nil {
			continue
		}
		mark := "   "
		if i == t.cursor {
			mark = " > "
		}
		docks := ""
		if town.SellsShips() {
			docks = "　碼頭"
		}
		line(fmt.Sprintf("%s%-16s 物價 %2d%s", mark, town.Name, town.Economy, docks))
	}
	line("")
	line("↑↓：選擇　Enter：進城　Esc：取消")
}

func (a *app) drawTownMenu(dst *ebiten.Image, line func(string)) {
	v := a.town.visit
	line(fmt.Sprintf("%s　物價指數 %d", v.Town.Name, v.Economy.E))
	line("")

	labels := []string{"M 市集", "H 治療所", "I 旅店", "G 城鎮公會",
		"C 神殿", "D 碼頭", "B 酒館"}
	for i, f := range game.AllFacilities {
		note := ""
		if f == game.FacilityDocks && !v.HasDocks() {
			note = "（這裡沒有船）"
		}
		line("  " + labels[i] + note)
	}
	line("")
	line("Esc：離開城鎮")
	line("")
	// 哪座城鎮實際有哪些設施還沒解出來，先照實說，不要讓玩家以為是 bug。
	line("※ 城鎮設施清單尚未從原版解出，目前七種都列出")
}

func (a *app) drawFacility(dst *ebiten.Image, line func(string)) {
	t := a.town
	v := t.visit
	e := v.Economy

	line(fmt.Sprintf("%s — %s", v.Town.Name, game.FacilityName(*t.facility)))
	line("")

	switch *t.facility {
	case game.FacilityMarket:
		a.drawMarket(t, line)

	case game.FacilityHealers:
		line(fmt.Sprintf("治療　每點傷害 %d 金", e.HealRate()))
		line(fmt.Sprintf("解毒　%d 金", e.UnpoisonRate()))
		line(fmt.Sprintf("解除束縛　每級 %d 金", e.UnbindRate()))
		line(fmt.Sprintf("復活　每級 %d 金", e.ResurrectRate()))
		line("")
		for _, c := range a.members {
			svc, cost := e.HealerQuote(game.StatusNormal, c.Level, c.MaxHP-c.CurrentHP)
			if svc == game.HealerNone {
				line(fmt.Sprintf("%-8s 狀態良好", c.Name))
				continue
			}
			line(fmt.Sprintf("%-8s %s %d 金", c.Name, healerServiceName(svc), cost))
		}

	case game.FacilityPub:
		line(fmt.Sprintf("糧食　每份 %d 金（一次可買 %d–%d 份）",
			e.RationUnitPrice(), game.MinRations, game.MaxRations))

	case game.FacilityDocks:
		if v.HasDocks() {
			line(fmt.Sprintf("買船　%d 金", e.ShipPrice()))
		} else {
			line("這座城鎮沒有船可買")
		}
		line(fmt.Sprintf("修船　每點船體 %d 金（滿值 %d）",
			e.RepairPrice(game.ShipMaxHull-1), game.ShipMaxHull))
		line("")
		line("B：詢問船價")

	case game.FacilityChurch:
		line("捐獻　1 金換 1 點經驗")
		line("")
		for _, c := range a.members {
			line(fmt.Sprintf("%-8s 祈禱 %d 金", c.Name, game.PrayCost(c.Level)))
		}

	case game.FacilityGuild:
		line("升級　免費")
		line("")
		for _, c := range a.members {
			line(fmt.Sprintf("%-8s %d 級　經驗 %d", c.Name, c.Level, c.Experience))
		}

	case game.FacilityInn:
		line("休息回復法力。原版的休息與時間推進尚未接上。")
	}

	line("")
	if t.message != "" {
		line(t.message)
		line("")
	}
	line("Esc：回到設施選單")
}

func (a *app) drawMarket(t *townScreen, line func(string)) {
	line(fmt.Sprintf("鑑定　%d 金", t.visit.Economy.IdentifyPrice()))
	line("")

	// 欄名放表頭，不要每列重複。
	// 中文字模 16×15 塞在 16 像素行高裡只剩 1 像素間隙，整欄相同的密集字
	// （每列都寫「買」「賣」）會黏成一片糊，看不出是哪個字。
	line("   商品             買價   賣價")

	const window = 8
	start := t.cursor - window/2
	if start < 0 {
		start = 0
	}
	if start+window > a.items.Len() {
		start = a.items.Len() - window
	}
	for i := start; i < start+window && i < a.items.Len(); i++ {
		item, err := a.items.ByIndex(i)
		if err != nil {
			continue
		}
		mark := "   "
		if i == t.cursor {
			mark = " > "
		}
		// 被觸怒後的商品不能顯示價格。HagglePrice 對 s >= 1000 會算出
		// 下限 2 金 —— 那個數字看起來像「跳樓大拍賣」，實際上是商人拒賣。
		if t.visit.HaggleState(i).Refused() {
			line(fmt.Sprintf("%s%-14s %5s  %5d", mark, item.Name, "拒賣",
				t.visit.Economy.SellPrice(item.Price)))
			continue
		}
		line(fmt.Sprintf("%s%-14s %5d  %5d", mark, item.Name,
			t.visit.Price(i, item.Price), t.visit.Economy.SellPrice(item.Price)))
	}
	line("")
	line("↑↓：選商品　H：議價")
}

func healerServiceName(s game.HealerService) string {
	switch s {
	case game.HealerHeal:
		return "治療"
	case game.HealerUnpoison:
		return "解毒"
	case game.HealerUnbind:
		return "解除束縛"
	case game.HealerResurrect:
		return "復活"
	}
	return ""
}

// 讓編譯器確認這幾個型別有被用到。
var _ = gamedata.Town{}
