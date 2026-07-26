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

// 商隊遭遇（規則見 `docs/re/32`／`44`／`45`／`46`）。
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
}

// openMerchant 開一支商隊。
//
// 規模照原版擲（`docs/re/50`）：基準值來自存檔 `+0xaf`，
// **規模同時就是商隊等級**。原版起始存檔的基準是 1 與 4，
// 所以一開始遇到的都是小商隊 —— 那是遊戲設計，不是我們算錯。
func (a *app) openMerchant() {
	size := game.MerchantSize(a.rng, int(a.save.MerchantBase))
	a.merchant = &merchantScreen{m: game.RollMerchant(a.rng, a.tables, a.items, size)}
}

func (a *app) updateMerchant() error {
	s := a.merchant

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
	}
	return nil
}

func (a *app) buyFromMerchant(s *merchantScreen) {
	label := a.itemLabel(s.m.Wares[s.cursor].Item)
	res := game.BuyFromMerchant(&s.m, s.cursor, a.members, a.gold())
	if !res.OK {
		s.message = res.Reason
		return
	}
	a.setGold(res.Gold)
	s.message = fmt.Sprintf("買下%s，%s 收著（剩 %d 金幣）",
		label, a.members[res.Member].Name, res.Gold)
}

func (a *app) drawMerchant(dst *ebiten.Image) {
	s := a.merchant
	y := layout.StatusY
	line := func(t string) {
		a.font.Draw(dst, t, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	line(fmt.Sprintf("你看到一群%s的商人", a.merchantAdjective(s.m.Size)))
	line("")

	if !s.greeted {
		line("  G 打招呼")
		line("  I 無視他們")
		line("")
		line("Esc：走開")
		return
	}

	line(fmt.Sprintf("他們把貨攤開來看　金幣 %d", a.gold()))
	line("")
	for i, w := range s.m.Wares {
		mark := "   "
		if i == s.cursor {
			mark = " > "
		}
		price := fmt.Sprintf("%d", w.Price)
		if w.Sold {
			price = "已售出"
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
	line("↑↓：選擇　Enter：買下　Esc：走開")
}

// merchantAdjective 回傳商隊規模的形容詞（已翻譯）。
func (a *app) merchantAdjective(size int) string {
	en := game.MerchantAdjective(size)
	return a.tr.Event(merchantSourceFile, size, en)
}

// merchantSourceFile 是商隊形容詞翻譯目錄的 key。
const merchantSourceFile = "MERCHANT"
