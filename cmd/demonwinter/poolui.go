package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 治療水池（tile `0x35`，原版動作 `0x11` ＝ `222f:37c4`）。
//
// **接上之前引擎把它當牆** —— `docs/re/05` 推測成「寫死的阻擋」，
// 而 `docs/re/90` 讀完發現那是一口可以喝的治療水池。
// 規則在 `internal/game/pool.go`，這裡只做選人與訊息。
//
// 原版的流程是一個迴圈：問「哪個角色要喝」→ 喝一口 → 印回復幾點 →
// **回去再問一次**，直到玩家取消或今天的額度用完。
// 這個畫面照做，所以喝完不會自動關掉。

// poolScreen 是水池的選人畫面。
type poolScreen struct {
	cursor int
	// lines 是這一輪已經喝過的紀錄，累積顯示（原版是逐行印在同一個框裡）。
	lines []string
}

// openPool 打開水池畫面。**隊伍不會走上那一格** ——
// 原版的回傳碼發生在寫座標之前（`docs/re/90` §4）。
func (a *app) openPool() {
	if len(a.members) == 0 {
		return
	}
	a.pool = &poolScreen{}
	a.trace.note("水池：剩 %d 口", a.save.PoolDrinks)
}

func (a *app) updatePool() error {
	p := a.pool
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.pool = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		p.cursor = (p.cursor + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		p.cursor = (p.cursor - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.drinkFromPool()
	}
	return nil
}

func (a *app) drinkFromPool() {
	p := a.pool
	sip := game.DrinkFromPool(a.rng, a.members, p.cursor, &a.save.PoolDrinks)
	if sip.Empty {
		// 原版在迴圈頂端就印這一行然後回傳 —— 額度用完畫面就結束。
		a.message = a.tr.UI("pool.empty")
		a.pool = nil
		a.trace.note("水池：乾了")
		return
	}
	// 原版的字串是 `He is healed %d`。**0 是合法結果**（滿血的人喝一口
	// 照樣扣一次額度），所以這一行不做「沒回血就不印」的優化。
	p.lines = append(p.lines, fmt.Sprintf(
		a.tr.UI("pool.healed"),
		a.members[sip.Member].Name, sip.Healed))
	a.trace.note("水池：%s 回復 %d，剩 %d 口",
		a.members[sip.Member].Name, sip.Healed, a.save.PoolDrinks)
}

func (a *app) drawPool(dst *ebiten.Image) {
	p := a.pool
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	line(a.tr.UI("pool.title"))
	line(fmt.Sprintf(a.tr.UI("pool.left"), a.save.PoolDrinks))
	line("")
	line(a.tr.UI("pool.who"))
	for i := range a.members {
		c := &a.members[i]
		line(fmt.Sprintf("%s%s　%d/%d",
			memberMark(p.cursor, i), c.Name, c.CurrentHP, c.MaxHP))
	}
	line("")
	for _, l := range p.lines {
		line(l)
	}
	line("")
	line(a.tr.UI("pool.keys"))
}
