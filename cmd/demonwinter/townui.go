package main

import (
	"fmt"
	"strconv"
	"strings"

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

	// member 是設施內的隊員游標（治療所／神殿／公會／學院共用）。
	member int
	// college 是學院清單的游標（一座城鎮最多三間）。
	college int

	// amount 非 nil 時「輸入數量」的小框開著（捐獻、買糧）。
	amount *amountInput
	// sell 非 nil 時市集的出售選擇器開著。
	sell *sellPicker

	// message 是最近一次操作的結果。
	message string

	// worldSite 為 true 時這不是城鎮，是世界地圖上的獨立神殿／學院
	// （`worldsiteui.go`）。它只有一項設施，所以 Esc 要直接回地圖 ——
	// 退回「設施選單」會停在一個只有一項的清單上，看起來像卡住了。
	worldSite bool
}

// facilityKeys 是設施的熱鍵，順序同 game.AllFacilities。
var facilityKeys = []ebiten.Key{
	ebiten.KeyM, ebiten.KeyH, ebiten.KeyI,
	ebiten.KeyG, ebiten.KeyC, ebiten.KeyD, ebiten.KeyB,
	// 學院是本作補的第八項，熱鍵自己選 L（Learn）。
	ebiten.KeyL,
}

// townSourceFile 是城鎮名稱翻譯目錄的 key，與 dwstrings 產生時一致。
const townSourceFile = "TOWN.TXT"

// townName 回傳城鎮的顯示名稱（已翻譯）。索引是城鎮編號減 1。
//
// `Number == 0` 是**世界地圖上的獨立設施**（`worldsiteui.go` 造的臨時記錄），
// 不是城鎮 —— 它沒有編號，直接用 Name。不擋掉的話索引會變 −1，
// 而翻譯目錄拿 −1 查不到就退回空字串，抬頭會變成一片空白。
func (a *app) townName(t gamedata.Town) string {
	if t.Number == 0 {
		return t.Name
	}
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
	// 疊在設施上的兩個子畫面自己吃掉輸入（含 Esc），不然按一次 Esc
	// 會連設施一起退掉 —— ESC 只退一層。
	if t.amount != nil {
		if t.amount.update() {
			t.amount = nil
		}
		return nil
	}
	if t.sell != nil {
		a.updateSellPicker(t)
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if t.worldSite {
			// 地圖上的設施只有一項，直接離開。
			a.town = nil
			a.message = fmt.Sprintf("離開%s", t.visit.Town.Name)
			return nil
		}
		t.facility = nil
		t.message = ""
		return nil
	}

	switch *t.facility {
	case game.FacilityInn:
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			a.restAtInn()
			return nil
		}
	case game.FacilityHealers:
		a.moveMemberCursor(t)
		if inpututil.IsKeyJustPressed(ebiten.KeyH) {
			a.healCurrent(t)
		}
	case game.FacilityPub:
		if inpututil.IsKeyJustPressed(ebiten.KeyB) {
			a.openRationInput(t)
		}
	case game.FacilityChurch:
		a.moveMemberCursor(t)
		if inpututil.IsKeyJustPressed(ebiten.KeyP) {
			a.prayCurrent(t)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyD) {
			a.openDonateInput(t)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyV) {
			a.convertCurrent(t)
		}
	case game.FacilityGuild:
		a.moveMemberCursor(t)
		if inpututil.IsKeyJustPressed(ebiten.KeyL) {
			a.levelUpCurrent(t)
		}
	case game.FacilityCollege:
		a.updateCollege(t)
	case game.FacilityDocks:
		if inpututil.IsKeyJustPressed(ebiten.KeyB) {
			a.buyShip(t)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			a.repairShip(t)
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
		if inpututil.IsKeyJustPressed(ebiten.KeyB) {
			a.buyCurrent(t)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyS) {
			if len(a.members) == 0 {
				t.message = "隊伍是空的"
				return nil
			}
			t.sell = &sellPicker{slot: -1}
		}
	}
	return nil
}

// 市集表格的欄寬（排版格）。表頭與資料列共用，改一個就兩邊一起動。
const (
	marketNameCells  = 18
	marketPriceCells = 7
)

// buyCurrent 買下游標指的商品。
//
// 付的是議價後的價格（TownVisit.Price）。買到的是**沒有效果的平凡裝備** ——
// 原版有效果的道具是掉寶生成的，店裡不賣（見 game.Buy 與 docs/re/25）。
func (a *app) buyCurrent(t *townScreen) {
	item, err := a.items.ByIndex(t.cursor)
	if err != nil {
		return
	}
	if t.visit.HaggleState(t.cursor).Refused() {
		t.message = "商人不肯賣這件給你"
		return
	}
	price := t.visit.Price(t.cursor, item.Price)
	res := game.Buy(a.members, a.gold(), price, t.cursor)
	if !res.OK {
		t.message = res.Reason
		return
	}
	a.setGold(res.Gold)
	t.message = fmt.Sprintf("%s 買下%s，付 %d 金（剩 %d）",
		a.members[res.Member].Name,
		a.tr.Event(itemSourceFile, t.cursor, item.Name), price, res.Gold)
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

	// 這張表要與 game.AllFacilities 等長 —— 加了設施卻忘了加標籤，
	// 下面那行 labels[i] 會直接 panic（加學院時踩過一次）。
	labels := []string{"M 市集", "H 治療所", "I 旅店", "G 城鎮公會",
		"C 神殿", "D 碼頭", "B 酒館", "L 學院"}
	for i, f := range game.AllFacilities {
		if i >= len(labels) {
			break
		}
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

// restAtInn 在旅店睡一晚。
//
// 規則在 game.Rest（原版 2aed:03e4）。旅店那條分支不吃糧食、回復比紮營多，
// 而且**讀到的常式沒有扣金錢**，所以這裡不收費。
func (a *app) restAtInn() {
	if !a.clock.CanSleep() {
		a.town.message = "現在睡不著（原版：You are restless）"
		return
	}
	res := game.Rest(a.rng, game.RestInn, a.members, a.clock, nil)
	// 睡一晚就清掉隊伍層級的每日旗標（與紮營同一條規則）。
	// 治療水池的額度也在同一段補回 7（原版 `0x1eee6`，`docs/re/90` §2）。
	game.ResetPoolDrinks(a.save)
	a.save.ViewedLandToday = false

	msg := fmt.Sprintf("睡了 %d 個時辰，%d 日 %d 時醒來",
		res.Hours, a.clock.Day(), a.clock.Hour())
	if len(res.Died) > 0 {
		names := make([]string, 0, len(res.Died))
		for _, i := range res.Died {
			names = append(names, a.members[i].Name)
		}
		msg += "　" + strings.Join(names, "、") + " 沒有醒來"
	}
	a.town.message = msg
}

func (a *app) drawFacility(dst *ebiten.Image, line func(string)) {
	t := a.town
	v := t.visit
	e := v.Economy

	line(fmt.Sprintf("%s — %s", a.townName(v.Town), game.FacilityName(*t.facility)))
	line("")

	// 疊在設施上的子畫面自己畫完就結束，不要把價目表也一起畫出來。
	if t.amount != nil {
		t.amount.draw(line)
		return
	}
	if t.sell != nil {
		a.drawSellPicker(t, line)
		return
	}

	switch *t.facility {
	case game.FacilityMarket:
		a.drawMarket(t, line)

	case game.FacilityHealers:
		line(fmt.Sprintf("治療　每點傷害 %d 金", e.HealRate()))
		line(fmt.Sprintf("解毒　%d 金", e.UnpoisonRate()))
		line(fmt.Sprintf("解除束縛　每級 %d 金", e.UnbindRate()))
		line(fmt.Sprintf("復活　每級 %d 金", e.ResurrectRate()))
		line("")
		for i, c := range a.members {
			mark := memberMark(t.member, i)
			svc, cost := e.HealerQuote(game.UnitStatus(c.Status), c.Level,
				c.BindLevel, c.MaxHP-c.CurrentHP)
			if svc == game.HealerNone {
				line(fmt.Sprintf("%s%s 狀態良好", mark, textlayout.PadCells(c.Name, 8)))
				continue
			}
			line(fmt.Sprintf("%s%s %s %d 金", mark,
				textlayout.PadCells(c.Name, 8), healerServiceName(svc), cost))
		}
		line("")
		line("↑↓：選擇隊員　H：接受治療")

	case game.FacilityPub:
		line(fmt.Sprintf("糧食　每份 %d 金（一次可買 %d–%d 份）",
			e.RationUnitPrice(), game.MinRations, game.MaxRations))
		line(fmt.Sprintf("隊伍目前 %d 份", a.save.Rations))
		line("")
		line("B：買糧")

	case game.FacilityDocks:
		if v.HasDocks() {
			line(fmt.Sprintf("買船　%d 金", e.ShipPrice()))
		} else {
			line("這座城鎮沒有船可買")
		}
		line(fmt.Sprintf("修船　每點船體 %d 金（滿值 %d）",
			e.RepairPrice(game.ShipMaxHull-1), game.ShipMaxHull))
		line("")
		if i := game.FindShipNear(&a.save.Ships, a.party.X(), a.party.Y()); i >= 0 {
			line(fmt.Sprintf("腳邊有一艘船，船體 %d／%d",
				a.save.Ships[i].Hull, game.ShipMaxHull))
		} else {
			line("腳邊沒有船")
		}
		line("")
		line("B：買船　R：修船")

	case game.FacilityChurch:
		line(fmt.Sprintf("供奉 %s　捐獻 1 金換 1 點經驗",
			a.deityName(v.Town.Facilities.Church)))
		line("")
		for i, c := range a.members {
			line(fmt.Sprintf("%s%s 信 %s　成功率 %d%%　祈禱 %d 金",
				memberMark(t.member, i), textlayout.PadCells(c.Name, 8),
				textlayout.PadCells(a.deityName(c.Deity), 8),
				c.PrayChance, game.PrayCost(c.Level)))
		}
		line("")
		line(fmt.Sprintf("改宗要 %s 的智力點數，不收金幣",
			a.skillName(game.DeityOrder(v.Town.Facilities.Church))))
		line("↑↓：選擇隊員　P：祈禱　D：捐獻　V：改宗")

	case game.FacilityGuild:
		line("升級　免費")
		line("")
		for i, c := range a.members {
			need := ""
			if ok, short := c.CanLevelUp(); ok {
				need = "可升級"
			} else if short > 0 {
				need = fmt.Sprintf("還差 %d", short)
			} else {
				need = "已達頂級"
			}
			line(fmt.Sprintf("%s%s %2d 級　經驗 %-8d %s",
				memberMark(t.member, i), textlayout.PadCells(c.Name, 8),
				c.Level, c.Experience, need))
		}
		line("")
		line("↑↓：選擇隊員　L：升級")

	case game.FacilityCollege:
		a.drawCollege(t, line)

	case game.FacilityInn:
		line(fmt.Sprintf("目前 %d 時（睡覺要在 15–24 時之間）", a.clock.Hour()))
		line("")
		if a.clock.CanSleep() {
			line("R：睡一晚（HP +2、法力 +10）")
		} else {
			line("現在睡不著。")
		}
		line("")
		line("※ 讀到的休息常式沒有扣金錢，所以這裡是免費的")
	}

	line("")
	if t.message != "" {
		// 訊息可以多行（升級的成長明細一行放不下，硬塞會超出畫布被裁掉）。
		for _, l := range strings.Split(t.message, "\n") {
			line(l)
		}
		line("")
	}
	if t.worldSite {
		line("Esc：離開")
	} else {
		line("Esc：回到設施選單")
	}
}

func (a *app) drawMarket(t *townScreen, line func(string)) {
	line(fmt.Sprintf("鑑定　%d 金", t.visit.Economy.IdentifyPrice()))
	line("")

	// 欄名放表頭，不要每列重複。
	// 中文字模 16×15 塞在 16 像素行高裡只剩 1 像素間隙，整欄相同的密集字
	// （每列都寫「買」「賣」）會黏成一片糊，看不出是哪個字。
	//
	// 表頭與資料列都用同一組欄寬常數排 —— 手動數空白排出來的表頭，
	// 只要品名欄寬一改就會歪掉。
	// 兩個價格欄的數值是**右靠齊**的，欄名也要右靠齊 —— 欄名用 PadCells
	// （左靠齊）就會比數字往左偏 3 格，看起來像兩欄各自對不上自己的標題。
	line(textlayout.PadCells("   商品", marketNameCells) +
		textlayout.PadCellsLeft("買價", marketPriceCells) +
		textlayout.PadCellsLeft("賣價", marketPriceCells))

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
		row := mark + textlayout.PadCells(name, marketNameCells-len([]rune(mark)))
		sell := textlayout.PadCellsLeft(
			strconv.Itoa(t.visit.Economy.SellPrice(item.Price)), marketPriceCells)
		if t.visit.HaggleState(i).Refused() {
			line(row + textlayout.PadCellsLeft("拒賣", marketPriceCells) + sell)
			continue
		}
		line(row + textlayout.PadCellsLeft(
			strconv.Itoa(t.visit.Price(i, item.Price)), marketPriceCells) + sell)
	}
	line("")
	line(fmt.Sprintf("↑↓：選商品　H：議價　B：買下　S：出售　（金幣 %d）", a.gold()))
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
