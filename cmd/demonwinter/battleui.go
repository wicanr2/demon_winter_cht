package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/audio/pcspeaker"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 玩家戰鬥指令。
//
// **熱鍵沿用原版**（A 攻擊、T 驅散不死、D 閃避、P 祈禱、L 汲取、? 檢視、ESC 結束回合），
// 只把顯示文字換成中文。玩過原版的人不必重學鍵位，這是中文化該有的樣子。
var playerCommands = []struct {
	key    ebiten.Key
	label  string
	action game.Action
}{
	{ebiten.KeyA, "A 攻擊", game.ActionAttack},
	{ebiten.KeyC, "C 施法", game.ActionCast},
	{ebiten.KeyU, "U 使用道具", game.ActionUseItem},
	{ebiten.KeyT, "T 驅散不死", game.ActionTurnUndead},
	{ebiten.KeyP, "P 祈禱", game.ActionPray},
	{ebiten.KeyL, "L 汲取法力", game.ActionLeech},
	{ebiten.KeyD, "D 閃避", game.ActionDodge},
	{ebiten.KeyS, "S 音效開關", game.ActionSound},
	{ebiten.KeyEscape, "Esc 結束回合", game.ActionEndTurn},
}

// updateBattle 推進戰鬥。
//
// 玩家單位輪到時等玩家下指令；怪物與召喚物由簡單 AI 代打，按空白鍵逐步執行，
// 方便肉眼核對每一步。
func (a *app) updateBattle() error {
	// 選點要排在回合派發之前。**施法會立刻結束回合**（Spend 的 endsTurn），
	// 所以游標打開時「目前單位」已經換成下一個了 ——
	// 把它放進 updatePlayerTurn 裡永遠等不到按鍵。
	if a.aoe != nil {
		return a.updateAOECursor()
	}

	if out := a.battle.Outcome(); out != game.Ongoing {
		if !inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			return nil
		}
		if out == game.Victory {
			a.logf("怪物全滅")
		} else {
			a.logf("隊伍全滅")
		}
		a.battle = nil
		return nil
	}

	cur := a.battle.Current()
	if cur == nil {
		a.battle.BeginRound()
		a.logf("── 第 %d 回合 ──", a.battle.Round())
		return nil
	}

	if cur.IsPlayer {
		return a.updatePlayerTurn(cur)
	}
	if !inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return nil
	}
	a.monsterTurn(cur)
	return nil
}

// spellMenu 是施法選單的狀態。
type spellMenu struct {
	caster *game.Unit
	// entries 是這名施法者付得起、且不是佔位記錄的法術。
	entries []spellEntry
	cursor  int
}

// spellEntry 是選單上的一項。
type spellEntry struct {
	index int
	name  string
	spell gamedata.Spell
}

// openSpellMenu 列出施法者目前選得到的法術。
//
// **只列付得起的**：原版在選單外先檢查 SP 是否 >= M，不足就顯示
// 「not enough points」。列出來又不能選比較容易讓玩家困惑，
// 但完全藏起來又看不到目標，所以列出全部、把付不起的標灰。
func (a *app) openSpellMenu(u *game.Unit) {
	m := &spellMenu{caster: u}
	for i := 0; i < a.tables.NumSpells(); i++ {
		sp, err := a.tables.Spell(i)
		if err != nil || sp.Empty() {
			continue
		}
		name, err := a.strings.SpellName(i)
		if err != nil {
			continue
		}
		m.entries = append(m.entries, spellEntry{
			index: i,
			name:  a.tr.Event(spellSourceFile, i, name),
			spell: sp,
		})
	}
	a.spells = m
}

// spellSourceFile 是法術名稱翻譯目錄的 key，與 dwstrings 產生時一致。
const spellSourceFile = "FILES.DTT"

// updateSpellMenu 處理施法選單的按鍵。
func (a *app) updateSpellMenu(u *game.Unit) error {
	m := a.spells
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.spells = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		m.cursor = (m.cursor + 1) % len(m.entries)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		m.cursor = (m.cursor - 1 + len(m.entries)) % len(m.entries)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.castSelected(u)
	}
	return nil
}

// castSelected 施放游標指的法術。
func (a *app) castSelected(u *game.Unit) {
	e := a.spells.entries[a.spells.cursor]

	// 原版的順序：先看法力夠不夠（SP < M → not enough points），
	// 再看行動點。兩個都不足時顯示法力那一則 —— 玩家先想知道的是那個。
	if u.CurrentSP < e.spell.M {
		a.logf("%s 的法力不足以施放%s（需要 %d 點）", u.Name, e.name, e.spell.M)
		return
	}
	// 原版會問「How many S.P.」讓玩家決定投入多少 —— 投得多效果強，
	// 但法力用完就沒了。先問，行動點等確認後才扣：在輸入框按 Esc 反悔
	// 不該算掉一次行動。
	a.spInput = &spPrompt{
		caster: u, entry: e,
		input: game.NewSPInput(e.spell.M, u.CurrentSP),
	}
}

// castWithSP 是玩家在輸入框確認投入點數之後真正施法。
func (a *app) castWithSP(u *game.Unit, e spellEntry, sp int) {
	if _, ok := a.battle.Spend(game.ActionCast); !ok {
		a.logf("%s 行動點不足", u.Name)
		return
	}
	a.spells = nil
	u.CurrentSP -= sp

	// 範圍法術要先選中心點。原版也是先跳游標（FUN_138d_3fc9）再套效果。
	if e.spell.Effect == game.EffectAOE {
		a.aoe = &aoeCursor{caster: u, entry: e, sp: sp}
		a.aoeX, a.aoeY = u.X, u.Y
		return
	}

	target := a.battle.TargetInFront(u)
	a.applySpell(u, target, e, sp)
}

// spPrompt 是「投入多少法力」的輸入框畫面。
//
// 數字本身的規則在 game.SPInput（純邏輯、測得到），這裡只負責接按鍵與畫。
type spPrompt struct {
	caster *game.Unit
	entry  spellEntry
	input  *game.SPInput
}

func (a *app) updateSPPrompt() error {
	p := a.spInput

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		// 退回施法選單，行動點還沒扣，什麼都沒發生。
		a.spInput = nil
		return nil

	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		caster, entry, amount := p.caster, p.entry, p.input.Amount()
		a.spInput = nil
		a.castWithSP(caster, entry, amount)
		return nil

	case inpututil.IsKeyJustPressed(ebiten.KeyBackspace):
		p.input.Backspace()
		return nil

	case inpututil.IsKeyJustPressed(ebiten.KeyUp), inpututil.IsKeyJustPressed(ebiten.KeyRight):
		p.input.Adjust(1)
		return nil

	case inpututil.IsKeyJustPressed(ebiten.KeyDown), inpututil.IsKeyJustPressed(ebiten.KeyLeft):
		p.input.Adjust(-1)
		return nil
	}

	if d := pressedDigit(); d >= 0 {
		// pressedDigit 回傳 0–8 對應鍵盤 1–9、9 對應 0。
		digit := d + 1
		if digit == 10 {
			digit = 0
		}
		p.input.AppendDigit(digit)
	}
	return nil
}

func (a *app) drawSPPrompt(dst *ebiten.Image) {
	p := a.spInput
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	line(fmt.Sprintf("%s 施放%s", p.caster.Name, p.entry.name))
	line("")
	line(fmt.Sprintf("投入多少法力？　%d", p.input.Amount()))
	line("")
	line(fmt.Sprintf("可投入 %d–%d（目前法力 %d）",
		p.input.Min(), p.input.Max(), p.caster.CurrentSP))
	line("")
	line("數字鍵：輸入　↑↓：加減　Backspace：退格")
	line("Enter：施放　Esc：取消")
	line("")
	line("※ 投入越多效果越強，法力用完就沒了")
}

// applySpell 套用一個法術的效果。
//
// 只實作 spec 標 READY 的四類特殊效果與通式；其餘 effect_type
// 顯示「尚未實作」而不是靜默沒反應 —— 看不出有沒有生效比沒有功能更糟。
func (a *app) applySpell(caster, target *game.Unit, e spellEntry, sp int) {
	switch e.spell.Effect {
	case game.EffectInstantDeath:
		if target == nil {
			a.logf("%s 正前方沒有目標", caster.Name)
			return
		}
		if game.CastInstantDeath(a.rng, sp, e.spell, target) {
			a.battle.Kill(target)
			a.speaker.Play(pcspeaker.EffectDeath)
			a.logf("%s 的%s殺死了 %s", caster.Name, e.name, target.Name)
		} else {
			a.logf("%s 的%s沒有奏效", caster.Name, e.name)
		}

	case game.EffectBindApply:
		if target == nil {
			a.logf("%s 正前方沒有目標", caster.Name)
			return
		}
		res := game.CastBind(a.rng, sp, e.spell, target)
		switch {
		case res.Applied:
			a.logf("%s 用%s束縛了 %s（%d 回合）",
				caster.Name, e.name, target.Name, res.Rounds)
		case res.AlreadyBound:
			a.logf("%s 已經被束縛住了", target.Name)
		default:
			a.logf("%s 抵抗了%s", target.Name, e.name)
		}

	case game.EffectBindRelease:
		if target == nil {
			target = caster
		}
		if game.CastBindRelease(sp, e.spell, target) {
			a.logf("%s 的束縛被%s解開", target.Name, e.name)
		} else {
			a.logf("%s 的力度不足以解開束縛", e.name)
		}

	case game.EffectWither:
		if target == nil {
			a.logf("%s 正前方沒有目標", caster.Name)
			return
		}
		if game.CastWither(a.rng, sp, e.spell, target) {
			a.logf("%s 的%s讓 %s 衰朽", caster.Name, e.name, target.Name)
		} else {
			a.logf("%s 的%s沒有奏效", caster.Name, e.name)
		}

	case game.EffectAOE:
		// 走到這裡代表游標已經選好中心點（見 aoeCursor）。
		hits := game.CastAOE(a.rng, a.battle, e.spell, sp, a.aoeX, a.aoeY)
		if len(hits) == 0 {
			a.logf("%s 的%s沒有波及任何人", caster.Name, e.name)
			return
		}
		a.logf("%s 施放%s（波及 %d 人）", caster.Name, e.name, len(hits))
		for _, h := range hits {
			switch {
			case h.Resisted:
				a.logf("　%s 免疫", h.Unit.Name)
			case h.Killed:
				a.speaker.Play(pcspeaker.EffectDeath)
				a.logf("　%s 倒下了", h.Unit.Name)
			case h.Delta < 0:
				a.logf("　%s 減少 %d", h.Unit.Name, -h.Delta)
			case h.Delta > 0:
				a.logf("　%s 增加 %d", h.Unit.Name, h.Delta)
			}
		}

	default:
		// 數值增減類走通式。走不通的 effect_type 才是真的沒實作。
		//
		// 治療類（K > 0）預設對自己，傷害類對正前方 —— 原版是讓玩家用游標
		// 選格子，那個游標還沒做。**這是本作的簡化，不是原版行為。**
		who := target
		if e.spell.K > 0 {
			who = caster
		}
		delta, ok := game.CastMagnitudeEffect(a.rng, sp, e.spell, who)
		if !ok {
			a.logf("%s 施放%s —— 這類效果尚未實作", caster.Name, e.name)
			return
		}
		if who == nil {
			a.logf("%s 正前方沒有目標", caster.Name)
			return
		}
		switch {
		case delta > 0:
			a.logf("%s 的%s讓 %s 增加 %d", caster.Name, e.name, who.Name, delta)
		case delta < 0:
			a.logf("%s 的%s讓 %s 減少 %d", caster.Name, e.name, who.Name, -delta)
			if !who.Alive() {
				a.battle.Kill(who)
				a.speaker.Play(pcspeaker.EffectDeath)
				a.logf("%s 倒下了", who.Name)
			}
		default:
			a.logf("%s 的%s沒有明顯效果", caster.Name, e.name)
		}
	}
}

// drawSpellMenu 畫施法選單。
func (a *app) drawSpellMenu(dst *ebiten.Image) {
	m := a.spells
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	line(fmt.Sprintf("%s 施法　法力 %d", m.caster.Name, m.caster.CurrentSP))
	line("")
	line("   法術            最低法力")

	const window = 14
	start := m.cursor - window/2
	if start < 0 {
		start = 0
	}
	if start+window > len(m.entries) {
		start = len(m.entries) - window
	}
	if start < 0 {
		start = 0
	}
	for i := start; i < start+window && i < len(m.entries); i++ {
		e := m.entries[i]
		mark := "   "
		if i == m.cursor {
			mark = " > "
		}
		note := ""
		if m.caster.CurrentSP < e.spell.M {
			note = "  法力不足"
		}
		line(fmt.Sprintf("%s%-14s %5d%s", mark, e.name, e.spell.M, note))
	}
	line("")
	line("↑↓：選擇　Enter：施放　Esc：取消")
	line("※ 選好按 Enter 再決定投入多少法力")
}

// aoeCursor 是範圍法術的選點狀態。
//
// 法力已經在開游標之前扣掉了 —— 原版也是先扣再選點，
// 取消選點不會退還。這裡照做，並在畫面上寫明。
type aoeCursor struct {
	caster *game.Unit
	entry  spellEntry
	sp     int
}

// updateAOECursor 讓玩家用方向鍵挪動 5×5 的中心點。
func (a *app) updateAOECursor() error {
	for _, kf := range keyFacing {
		if !inpututil.IsKeyJustPressed(kf.key) {
			continue
		}
		dx, dy := kf.f.Delta()
		nx, ny := a.aoeX+dx, a.aoeY+dy
		if nx >= game.BattleGridMinX && nx < game.BattleGridWidth &&
			ny >= 0 && ny < game.BattleGridHeight {
			a.aoeX, a.aoeY = nx, ny
		}
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		c := a.aoe
		a.aoe = nil
		a.applySpell(c.caster, nil, c.entry, c.sp)
	}
	return nil
}

// itemMenu 是使用道具選單的狀態。
type itemMenu struct {
	caster  *game.Unit
	entries []game.UsableItem
	cursor  int
}

// u2c 找出戰鬥單位對應的隊伍成員。
//
// 用姓名對 —— 戰鬥單位是從角色複製出來的，兩者之間沒有指標關聯。
func u2c(members []game.Character, u *game.Unit) *game.Character {
	for i := range members {
		if members[i].Name == u.Name {
			return &members[i]
		}
	}
	return &game.Character{}
}

// updateItemMenu 處理使用道具選單的按鍵。
func (a *app) updateItemMenu(u *game.Unit) error {
	m := a.useMenu
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.useMenu = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		m.cursor = (m.cursor + 1) % len(m.entries)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		m.cursor = (m.cursor - 1 + len(m.entries)) % len(m.entries)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		e := m.entries[m.cursor]
		a.useMenu = nil
		if _, ok := a.battle.Spend(game.ActionUseItem); !ok {
			a.logf("%s 行動點不足", u.Name)
			return nil
		}
		// **道具的效果索引欄位還沒在存檔格式裡定位到。**
		// 反組譯是 `FUN_1000_114f(item.effect_index)` 載入 5-word 效果記錄，
		// 但那個欄位在 17 bytes 的槽裡對不到位置。照實說，不要瞎猜一個欄位
		// 然後產生看起來合理、其實亂來的效果。
		a.logf("%s 使用 %s —— 道具效果索引尚未定位，無效果",
			u.Name, a.itemLabel(e.Item))
	}
	return nil
}

// itemLabel 回傳道具的顯示名稱。
func (a *app) itemLabel(it scenario.InventorySlot) string {
	item, err := a.items.ByIndex(int(it.Type))
	if err != nil {
		return fmt.Sprintf("道具 %d", it.Type)
	}
	if it.Enchant != 0 {
		return fmt.Sprintf("%s%+d", item.Name, it.Enchant)
	}
	return item.Name
}

// drawItemMenu 畫使用道具選單。
func (a *app) drawItemMenu(dst *ebiten.Image) {
	m := a.useMenu
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	line(fmt.Sprintf("%s 使用道具", m.caster.Name))
	line("")
	for i, e := range m.entries {
		mark := "   "
		if i == m.cursor {
			mark = " > "
		}
		state := ""
		if !e.Item.Identified {
			state = "（未鑑定）"
		}
		line(fmt.Sprintf("%s%-14s%s", mark, a.itemLabel(e.Item), state))
	}
	line("")
	line("↑↓：選擇　Enter：使用　Esc：取消")
	line("")
	line("※ 只列得出已裝備的武器／護甲與消耗品 —— 原版的 Use")
	line("　 就是拿來觸發已裝備裝備的特殊能力")
	line("※ 道具的效果索引欄位尚未在存檔格式中定位，選了不會有效果")
}

// updatePlayerTurn 處理玩家單位的一次按鍵。
func (a *app) updatePlayerTurn(u *game.Unit) error {
	if a.useMenu != nil {
		return a.updateItemMenu(u)
	}
	// 投入點數的輸入框疊在施法選單上面，要先處理。
	if a.spInput != nil {
		return a.updateSPPrompt()
	}
	if a.spells != nil {
		return a.updateSpellMenu(u)
	}

	// 轉向與前進：方向鍵轉到該面向（每次轉一格），Enter 前進。
	for _, kf := range keyFacing {
		if !inpututil.IsKeyJustPressed(kf.key) {
			continue
		}
		a.faceToward(u, kf.f)
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if !a.battle.Step(u) {
			a.logf("%s 前方過不去", u.Name)
		}
		return nil
	}

	for _, c := range playerCommands {
		if !inpututil.IsKeyJustPressed(c.key) {
			continue
		}
		a.runPlayerAction(u, c.action)
		return nil
	}
	return nil
}

// faceToward 把單位轉向指定方位，一次按鍵只轉一格。
//
// 原版的轉向鍵是「順時針／逆時針／迴轉」各 1 點，不是「直接面向某方」。
// 這裡讓方向鍵挑最短路徑轉一格，按鍵直覺但成本與原版一致。
func (a *app) faceToward(u *game.Unit, want game.Facing) {
	cur := game.Facing(u.Facing)
	if cur == want {
		if !a.battle.Step(u) {
			a.logf("%s 前方過不去", u.Name)
		}
		return
	}

	act := game.ActionTurnCW
	switch {
	case cur.Reverse() == want:
		act = game.ActionAboutFace
	case cur.CCW() == want:
		act = game.ActionTurnCCW
	}
	if !a.battle.TurnTo(u, act) {
		a.logf("%s 行動點不足", u.Name)
	}
}

// runPlayerAction 執行一個玩家指令。
//
// **先檢查前置條件再扣點。** 沒有目標的攻擊、非不死系的驅散，
// 都應該是「什麼都沒發生」而不是「白花三點」。
func (a *app) runPlayerAction(u *game.Unit, act game.Action) {
	switch act {
	case game.ActionCast:
		if u.CurrentSP <= 0 {
			a.logf("%s 沒有法力", u.Name)
			return
		}
		a.openSpellMenu(u)

	case game.ActionUseItem:
		items := u2c(a.members, u).UsableItems()
		if len(items) == 0 {
			a.logf("%s 沒有可用的道具", u.Name)
			return
		}
		a.useMenu = &itemMenu{caster: u, entries: items}

	case game.ActionAttack:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf("%s 正前方沒有敵人", u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf("%s 行動點不足", u.Name)
			return
		}
		a.reportAttack(u, target, a.battle.ResolveAttack(u, target, 0))

	case game.ActionTurnUndead:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf("%s 正前方沒有敵人", u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf("%s 行動點不足", u.Name)
			return
		}
		if game.TurnUndead(a.rng, u, target) {
			a.battle.Kill(target)
			a.logf("%s 驅散了 %s", u.Name, target.Name)
		} else {
			a.logf("%s 的驅散無效", u.Name)
		}

	case game.ActionPray:
		if _, ok := a.battle.Spend(act); !ok {
			a.logf("%s 行動點不足", u.Name)
			return
		}
		granted, next := game.Pray(a.rng, a.prayChance)
		a.prayChance = next
		if granted {
			a.logf("%s 的祈禱得到回應", u.Name)
		} else {
			a.logf("%s 的祈禱沒有回應", u.Name)
		}

	case game.ActionLeech:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf("%s 正前方沒有敵人", u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf("%s 行動點不足", u.Name)
			return
		}
		if ok, amount := game.Leech(a.rng, u, target.CurrentSP); ok {
			target.CurrentSP -= amount
			u.CurrentSP += amount
			a.logf("%s 汲走 %s 的 %d 點法力", u.Name, target.Name, amount)
		} else {
			a.logf("%s 的汲取失敗", u.Name)
		}

	case game.ActionDodge:
		bonus := a.battle.DoDodge()
		a.logf("%s 進入閃避（命中率 %d）", u.Name, game.DodgeHitModifier(bonus))

	case game.ActionSound:
		// 對應原版的 Sound on／Sound off（旗標 [0x1585]）。不耗行動點。
		a.speaker.SetEnabled(!a.speaker.Enabled())
		if a.speaker.Enabled() {
			a.logf("音效開啟")
		} else {
			a.logf("音效關閉")
		}

	case game.ActionEndTurn:
		a.battle.Spend(act)
		a.logf("%s 結束回合", u.Name)
	}
}

// monsterTurn 讓一隻怪物行動。
//
// AI 很簡單：前方有人就打，沒有就轉向最近的敵人，都不行就結束回合。
// 這不是原版行為 —— 原版的怪物 AI 尚未反組譯，這裡先讓戰鬥能跑完。
func (a *app) monsterTurn(u *game.Unit) {
	if target := a.battle.TargetInFront(u); target != nil {
		if _, ok := a.battle.Spend(game.ActionAttack); ok {
			a.reportAttack(u, target, a.battle.ResolveAttack(u, target, 0))
			return
		}
	}

	enemies := a.battle.Enemies(u)
	if len(enemies) == 0 {
		a.battle.Spend(game.ActionEndTurn)
		return
	}
	target := a.battle.Unit(enemies[0])
	if want, ok := stepToward(u, target); ok && game.Facing(u.Facing) != want {
		if a.battle.TurnTo(u, turnActionToward(game.Facing(u.Facing), want)) {
			return
		}
	}
	if a.battle.Step(u) {
		return
	}
	a.battle.Spend(game.ActionEndTurn)
}

// stepToward 回傳朝目標靠近該面向哪一邊。差距大的那個軸優先。
func stepToward(u, target *game.Unit) (game.Facing, bool) {
	dx, dy := target.X-u.X, target.Y-u.Y
	if dx == 0 && dy == 0 {
		return 0, false
	}
	if absInt(dx) >= absInt(dy) {
		if dx > 0 {
			return game.East, true
		}
		return game.West, true
	}
	if dy > 0 {
		return game.South, true
	}
	return game.North, true
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// turnActionToward 挑一個轉向動作，往目標面向靠近一格。
func turnActionToward(cur, want game.Facing) game.Action {
	switch {
	case cur.CW() == want:
		return game.ActionTurnCW
	case cur.CCW() == want:
		return game.ActionTurnCCW
	default:
		return game.ActionAboutFace
	}
}

// reportAttack 把一次攻擊的結果寫進紀錄並播對應音效。
//
// 音效編號照反組譯查到的呼叫端（見 docs/re/03 §1.5）：
// 未命中依交戰距離用 1 或 4、命中依武器類型用 5 或 8、陣亡放那段旋律。
// 這裡先用「近戰」那一支（1／5）—— 遠端攻擊的判別條件（欄位 0x4ed4）
// 還沒接進戰鬥單位。
func (a *app) reportAttack(attacker, target *game.Unit, res game.AttackResult) {
	switch {
	case !res.Hit:
		a.speaker.Play(pcspeaker.EffectC3)
		a.logf("%s 落空", attacker.Name)
	case res.NoEffect:
		a.logf("%s 對 %s 無效", attacker.Name, target.Name)
	case res.Killed:
		a.speaker.Play(pcspeaker.EffectDeath)
		a.logf("%s 擊殺 %s（%d 點）", attacker.Name, target.Name, res.Damage)
	default:
		a.speaker.Play(pcspeaker.EffectG3)
		verb := "命中"
		if res.Critical {
			verb = "重擊"
		}
		a.logf("%s %s %s %d 點", attacker.Name, verb, target.Name, res.Damage)
	}
}

var facingArrow = []string{"↑", "→", "↓", "←"}

// gridColor 是戰場格線的顏色，刻意比行動游標暗。
var gridColor = color.RGBA{0x40, 0x40, 0x40, 0xff}

// aoeColor 是範圍法術的選取框顏色。
var aoeColor = color.RGBA{0xff, 0x55, 0x55, 0xff}

// drawBattlefield 畫戰鬥網格：雙方單位的位置與面向。
//
// **底圖是空的。** 手冊說戰場是「該區域的放大地圖」，但那張放大圖怎麼生成
// 還沒反組譯出來。畫格線比畫一張猜的地形圖誠實 ——
// 玩家至少看得出誰站哪一格。
func (a *app) drawBattlefield(dst *ebiten.Image) {
	cell := gfx.TileWidth * layout.TileScale
	cur := a.battle.Current()

	// 先鋪地形，格線疊在上面。地形是大地圖的局部放大，看不到的格子是空的
	// （夜間視野縮小、被樹石擋住），那些地方就只剩格線。
	ts := a.tileset()
	for gy := 0; gy < layout.MapHeight/cell; gy++ {
		for gx := game.BattleGridMinX; gx < layout.MapWidth/cell; gx++ {
			if a.battleTerrain != nil {
				if v := a.battleTerrain.TileAt(gx, gy); v != 0 {
					if img := ts.Tile(v & 0x7f); img != nil {
						ui.DrawImageScaled(dst, img, gx*cell, gy*cell, layout.TileScale)
					}
				}
			}
			ui.StrokeRect(dst, gx*cell, gy*cell, cell, cell, gridColor)
		}
	}

	if a.aoe != nil {
		// 5×5 的效果範圍畫出來，玩家才知道會掃到誰（包含自己人）。
		r := game.AOERadius
		ui.StrokeRect(dst, (a.aoeX-r)*cell, (a.aoeY-r)*cell,
			(2*r+1)*cell, (2*r+1)*cell, aoeColor)
		ui.StrokeRect(dst, a.aoeX*cell, a.aoeY*cell, cell, cell, aoeColor)
	}

	for _, u := range a.battle.Units() {
		if !u.Alive() {
			continue
		}
		x, y := u.X*cell, u.Y*cell
		if x < 0 || x >= layout.MapWidth || y < 0 || y >= layout.MapHeight {
			continue
		}
		mark := "怪"
		if u.IsPlayer {
			mark = "我"
		}
		a.font.Draw(dst, mark, x, y)
		a.font.Draw(dst, facingArrow[u.Facing&3], x+16, y)
		if u == cur {
			ui.StrokeRect(dst, x, y, cell, cell, markerColor)
		}
	}
}

// drawBattle 畫戰鬥狀態：行動點、單位血量、最近幾行紀錄、可用指令。
func (a *app) drawBattle(dst *ebiten.Image) {
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	cur := a.battle.Current()
	line(fmt.Sprintf("戰鬥　第 %d 回合", a.battle.Round()))
	if cur != nil {
		line(fmt.Sprintf("%s 行動　%d 點", cur.Name, a.battle.Points()))
	}

	// 單位最多 15 個，加上抬頭與紀錄會超出地圖高度。先算出能放幾行，
	// 剩下的留給戰鬥紀錄 —— 紀錄比完整名單重要，看不到最後一擊很難受。
	const headerLines = 2
	rows := layout.MapHeight/ui.LineHeight - headerLines - battleLogLines - 1
	units := a.battle.Units()
	hidden := 0
	if len(units) > rows {
		hidden = len(units) - rows
		units = units[:rows]
	}

	for _, u := range units {
		tag := " "
		if u == cur {
			tag = ">"
		}
		state := ""
		if !u.Alive() {
			state = " 陣亡"
		}
		name := u.Name
		if len(name) > 8 {
			name = name[:8]
		}
		line(fmt.Sprintf("%s%-8s%3d/%-3d%s", tag, name, u.HP, u.MaxHP, state))
	}
	if hidden > 0 {
		line(fmt.Sprintf(" …另有 %d 個單位", hidden))
	}

	line("")
	// 紀錄依面板寬度斷行。不斷行的話長名字會把「命中 12 點」的尾巴
	// 畫到畫布外，看起來像少了一個字。
	width := layout.CanvasWidth - layout.StatusX
	for _, s := range a.log {
		for _, ln := range textlayout.WrapMixed(s, width) {
			line(ln)
		}
	}
}

// drawBattleCommands 把可用指令畫在文字視窗的位置。
func (a *app) drawBattleCommands(dst *ebiten.Image) {
	cur := a.battle.Current()
	y := layout.TextBoxTop + layout.BoxPadY

	if a.battle.Outcome() != game.Ongoing {
		a.font.Draw(dst, "戰鬥結束　空白鍵：繼續", layout.BoxPadX, y)
		return
	}
	if a.aoe != nil {
		a.font.Draw(dst, fmt.Sprintf(
			"%s：選 5×5 的中心　方向鍵：移動　Enter：施放",
			a.aoe.entry.name), layout.BoxPadX, y)
		a.font.Draw(dst, "※ 法力已扣，範圍內敵我都會被波及",
			layout.BoxPadX, y+ui.LineHeight)
		return
	}
	if cur == nil || !cur.IsPlayer {
		a.font.Draw(dst, "空白鍵：讓對方行動", layout.BoxPadX, y)
		return
	}

	// 付不起的指令仍然列出來但標成不可用 —— 藏起來會讓玩家以為功能不存在。
	x := layout.BoxPadX
	for i, c := range playerCommands {
		label := c.label
		if !a.battle.CanAct(c.action) {
			label = "(" + label + ")"
		}
		if i == 3 {
			x = layout.BoxPadX
			y += ui.LineHeight
		}
		x = a.font.Draw(dst, label+"  ", x, y)
	}
	a.font.Draw(dst, "方向鍵：轉向／前進　Enter：前進",
		layout.BoxPadX, y+ui.LineHeight)
}
