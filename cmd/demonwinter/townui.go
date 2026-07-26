package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// townScreen 是城鎮畫面的狀態。
type townScreen struct {
	visit *game.TownVisit

	// facility 為 nil 代表停在設施選單，否則進到某個設施內。
	facility *game.Facility

	// picking 為真代表正在選要進哪座城鎮。
	//
	// 正規入口已經是「走到世界地圖的城鎮格自動進城」（見 app.enterTownAt）。
	// 這個選單留著當偵錯用：T 鍵可以直接跳進任何一座城鎮，不用先走到那格。
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

// townSourceFile 是城鎮名稱翻譯目錄的 key，與 dwstrings 產生時一致。
const townSourceFile = "TOWN.TXT"

// townName 回傳城鎮的顯示名稱（已翻譯）。索引是城鎮編號減 1。
func (a *app) townName(t gamedata.Town) string {
	return a.tr.Event(townSourceFile, t.Number-1, t.Name)
}

// openTownPicker 打開城鎮選單（偵錯用的直接入口，見 townScreen.picking）。
func (a *app) openTownPicker() {
	a.town = &townScreen{picking: true}
}

// enterTownAt 以世界座標進城，對應原版「踩到城鎮格」的正規入口。
//
// 原版的查表迴圈沒有上界，座標不在表上就會一路讀過頭；這裡只留一行訊息。
// 會走到這條的情況是已知的：子地圖 54 有兩格城鎮 tile 不在 25 筆座標表裡
// （見 world 套件的 TestTownTiles_MostlyAccountedFor），語意還沒解。
func (a *app) enterTownAt(x, y int) {
	town, ok := a.towns.TownAt(x, y)
	if !ok {
		a.message = fmt.Sprintf("(%d,%d) 是城鎮格，但不在 25 筆城鎮座標表裡", x, y)
		return
	}
	a.town = &townScreen{visit: game.EnterTown(town, a.members)}
	a.message = fmt.Sprintf("進入%s", a.townName(town))
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
		a.message = fmt.Sprintf("離開%s", a.townName(t.visit.Town))
		return nil
	}
	for i, key := range facilityKeys {
		if !inpututil.IsKeyJustPressed(key) {
			continue
		}
		f := game.AllFacilities[i]
		if !a.town.visit.Town.Facilities.Has(int(f)) {
			continue // 沒列出來的設施，按熱鍵也不該進得去
		}
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
	line("（偵錯用。正常玩法是走到城鎮格自動進城）")
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
		line(fmt.Sprintf("%s%s 物價 %2d%s", mark,
			textlayout.PadCells(a.townName(town), 16), town.Economy, docks))
	}
	line("")
	line("↑↓：選擇　Enter：進城　Esc：取消")
}

func (a *app) drawTownMenu(dst *ebiten.Image, line func(string)) {
	v := a.town.visit
	line(fmt.Sprintf("%s　物價指數 %d", a.townName(v.Town), v.Economy.E))
	line("")

	labels := []string{"M 市集", "H 治療所", "I 旅店", "G 城鎮公會",
		"C 神殿", "D 碼頭", "B 酒館"}
	for i, f := range game.AllFacilities {
		if !v.Town.Facilities.Has(int(f)) {
			continue // 這座城鎮沒有這項設施（TOWN*.DAT 0x1ee–0x1f6）
		}
		note := ""
		if f == game.FacilityDocks && !v.HasDocks() {
			note = "（這裡沒有船）"
		}
		line("  " + labels[i] + note)
	}
	line("")
	line("Esc：離開城鎮")
}

func (a *app) drawFacility(dst *ebiten.Image, line func(string)) {
	t := a.town
	v := t.visit
	e := v.Economy

	line(fmt.Sprintf("%s — %s", a.townName(v.Town), game.FacilityName(*t.facility)))
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
				line(fmt.Sprintf("%s 狀態良好", textlayout.PadCells(c.Name, 8)))
				continue
			}
			line(fmt.Sprintf("%s %s %d 金",
				textlayout.PadCells(c.Name, 8), healerServiceName(svc), cost))
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
			line(fmt.Sprintf("%s 祈禱 %d 金",
				textlayout.PadCells(c.Name, 8), game.PrayCost(c.Level)))
		}

	case game.FacilityGuild:
		line("升級　免費")
		line("")
		for _, c := range a.members {
			line(fmt.Sprintf("%s %d 級　經驗 %d",
				textlayout.PadCells(c.Name, 8), c.Level, c.Experience))
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
		name := a.tr.Event(itemSourceFile, i, item.Name)
		// 被觸怒後的商品不能顯示價格。HagglePrice 對 s >= 1000 會算出
		// 下限 2 金 —— 那個數字看起來像「跳樓大拍賣」，實際上是商人拒賣。
		if t.visit.HaggleState(i).Refused() {
			line(fmt.Sprintf("%s%s %5s  %5d", mark, textlayout.PadCells(name, 14), "拒賣",
				t.visit.Economy.SellPrice(item.Price)))
			continue
		}
		line(fmt.Sprintf("%s%s %5d  %5d", mark, textlayout.PadCells(name, 14),
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
