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

// 商隊遭遇（規則見 `docs/re/32`／`44`／`45`／`46`／`49`／`50`）。
//
// **這個畫面是原版的子集。** 原版有七個選項
// （Purchase / View mind / Continue / Back / Haggle / Inspect / Quit，
// 見 `docs/re/52`），這裡只有 Purchase 與離開：
//
//   - Continue／Back 是貨單翻頁，640×400 一次列得完，沒有對應物
//   - Haggle 的三條分支解了，**降價幅度沒解**，湊一個等於自己設計遊戲
//   - Inspect／View mind 完全沒讀
//
// 路上遇到一群商人，打招呼就能看他們的貨。與市集最大的差別是
// **貨是掉寶生成器生出來的** —— 可能帶效果，也可能被詛咒。
//
// **進入鍵 `M` 是偵錯用的**：商隊是**事件動作**不是隨機遭遇
// （`docs/re/50`），要接上事件表才會自己出現。與 `B 測試戰鬥`、
// `T 進入城鎮` 同一個性質 —— 沒有這個鍵就沒辦法驗這個畫面。

// merchantScreen 是商隊畫面的狀態。
type merchantScreen struct {
	m game.Merchant
	// greeted 為 false 時停在「打招呼還是無視」。
	greeted bool
	// cursor 是貨物清單的游標。
	cursor int
	// message 是最近一次操作的結果。
	message string
	// mindRead 記錄 View mind 用過了沒（原版一支商隊只能用一次）。
	mindRead bool
	// party 是 Inspect 打開的角色卡（`docs/re/52` §2 的 case 13）。
	//
	// 原版的 Inspect 選一名角色再呼叫 `278d:2f61(角色, 3)` ——
	// **與紮營選單的 Party 是同一張卡**，所以這裡直接共用 partyScreen。
	party *partyScreen
}

// openMerchant 開一支商隊。
//
// 規模照原版擲（`docs/re/50`）：基準值來自存檔 `+0xaf`，
// **規模同時就是商隊等級**。原版起始存檔的基準是 1 與 4，
// 所以一開始遇到的都是小商隊 —— 那是遊戲設計，不是我們算錯。
func (a *app) openMerchant() {
	size := game.MerchantSize(a.rng, int(a.save.MerchantBase))
	m := game.RollMerchant(a.rng, a.tables, a.items, size)
	if a.debugMerchantLies >= 0 {
		m = game.RollMerchantWithLies(a.rng, a.tables, a.items, size, a.debugMerchantLies)
	}
	a.merchant = &merchantScreen{m: m}
}

func (a *app) updateMerchant() error {
	s := a.merchant

	if s.party != nil {
		if a.updatePartySheet(s.party) {
			s.party = nil
		}
		return nil
	}

	if !s.greeted {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape),
			inpututil.IsKeyJustPressed(ebiten.KeyI):
			a.merchant = nil
		case inpututil.IsKeyJustPressed(ebiten.KeyG),
			inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			s.greeted = true
		}
		return nil
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.merchant = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		s.cursor = (s.cursor + 1) % len(s.m.Wares)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		s.cursor = (s.cursor - 1 + len(s.m.Wares)) % len(s.m.Wares)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.buyFromMerchant(s)
	case inpututil.IsKeyJustPressed(ebiten.KeyH):
		a.haggleWithMerchant(s)
	case inpututil.IsKeyJustPressed(ebiten.KeyV):
		a.viewMerchantMind(s)
	case inpututil.IsKeyJustPressed(ebiten.KeyI):
		if len(a.members) > 0 {
			s.party = &partyScreen{member: 0, showing: -1}
		}
	}
	return nil
}

// viewMerchantMind 讀商人的心（原版的 View mind，`docs/re/53`）。
//
// **一支商隊只能用一次** —— 原版用過再按就印 "Only usable once"。
func (a *app) viewMerchantMind(s *merchantScreen) {
	if s.mindRead {
		s.message = a.tr.UI("merchant.mind.once")
		return
	}
	s.mindRead = true
	switch n := game.ViewMind(a.rng, &s.m); n {
	case 0:
		s.message = a.tr.UI("merchant.mind.nothing")
	default:
		s.message = fmt.Sprintf(a.tr.UI("merchant.mind.lies"), n)
	}
}

// haggleWithMerchant 對游標上那一件議價一次（原版的 Haggle，`docs/re/52` §3）。
func (a *app) haggleWithMerchant(s *merchantScreen) {
	label := a.itemLabel(s.m.Wares[s.cursor].Item)
	outcome, ok := game.HaggleWith(a.rng, &s.m, s.cursor)
	if !ok {
		s.message = a.tr.UI("merchant.haggle.refused")
		return
	}
	switch outcome {
	case game.HaggleSuccess:
		s.message = fmt.Sprintf(a.tr.UI("merchant.haggle.success"),
			label, s.m.Wares[s.cursor].WarePrice())
	case game.HaggleUnmoved:
		s.message = a.tr.UI("merchant.haggle.unmoved")
	default:
		s.message = fmt.Sprintf(a.tr.UI("merchant.haggle.offended"), label)
	}
}

func (a *app) buyFromMerchant(s *merchantScreen) {
	label := a.itemLabel(s.m.Wares[s.cursor].Item)
	res := game.BuyFromMerchant(&s.m, s.cursor, a.members, a.gold())
	if !res.OK {
		s.message = a.reasonText(res.Reason)
		return
	}
	a.setGold(res.Gold)
	s.message = fmt.Sprintf(a.tr.UI("merchant.buy.done"),
		label, a.members[res.Member].Name, res.Gold)
}

func (a *app) drawMerchant(dst *ebiten.Image) {
	s := a.merchant
	y := layout.StatusY
	line := func(t string) {
		a.font.Draw(dst, t, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	if s.party != nil {
		a.drawPartySheet(s.party, line)
		return
	}

	line(fmt.Sprintf(a.tr.UI("merchant.intro"), a.merchantAdjective(s.m.Size)))
	line("")

	if !s.greeted {
		line(a.tr.UI("merchant.greet.hail"))
		line(a.tr.UI("merchant.greet.ignore"))
		line("")
		line(a.tr.UI("merchant.greet.leave"))
		return
	}

	line(fmt.Sprintf(a.tr.UI("merchant.wares.header"), a.gold()))
	line("")
	for i, w := range s.m.Wares {
		mark := "   "
		if i == s.cursor {
			mark = " > "
		}
		price := fmt.Sprintf("%d", w.WarePrice())
		switch {
		case w.Sold:
			price = a.tr.UI("merchant.wares.sold")
		case w.Exposed:
			price = a.tr.UI("merchant.wares.lied")
		case w.Haggle.Refused():
			price = a.tr.UI("merchant.wares.refused")
		}
		note := ""
		if w.Item.Enchant != 0 {
			note = fmt.Sprintf("%+d", w.Item.Enchant)
		}
		line(fmt.Sprintf("%s%s%s%s", mark,
			textlayout.PadCells(a.itemLabel(w.Item), 14),
			textlayout.PadCells(price, 8), note))
	}
	line("")
	if s.message != "" {
		line(s.message)
		line("")
	}
	line(a.tr.UI("merchant.keys"))
}

// merchantAdjective 回傳商隊規模的形容詞（已翻譯）。
func (a *app) merchantAdjective(size int) string {
	en := game.MerchantAdjective(size)
	return a.tr.Event(merchantSourceFile, size, en)
}

// merchantSourceFile 是商隊形容詞翻譯目錄的 key。
const merchantSourceFile = "MERCHANT"
