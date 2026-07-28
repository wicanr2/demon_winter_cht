package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/audio/pcspeaker"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 玩家戰鬥指令。
//
// **熱鍵沿用原版**（A 攻擊、T 驅散不死、D 閃避、P 祈禱、L 汲取、ESC 結束回合），
// 只把顯示文字換成中文。玩過原版的人不必重學鍵位，這是中文化該有的樣子。
//
// ⚠ 這段原本把 `?` 檢視也列進去，但**它沒有綁鍵也沒有呼叫端** ——
// `game.ActionExamine` 只有定義。手冊寫的是：反白游標、顯示名稱／力量／技巧／
// 速度／護甲／武器，`←→` 切換，有怪物學識才顯示怪物屬性、有戰術才顯示牠要打誰。
// 少了這一格，**戰術與怪物學識兩個技能等於沒有效果**（`docs/manual-coverage.md` §7）。
// **`uikey` 與 `label` 是一組**：`label` 是 fallback，`uikey` 去 `ui.txt` 查。
// 這張表是套件層的變數，init 時還沒有 translator，所以不能在這裡就翻好 ——
// 翻譯發生在畫的時候（見 drawBattleCommands）。
var playerCommands = []struct {
	key    ebiten.Key
	uikey  string
	label  string
	action game.Action
}{
	{ebiten.KeyA, "battle.cmd.attack", "A 攻擊", game.ActionAttack},
	{ebiten.KeyC, "battle.cmd.cast", "C 施法", game.ActionCast},
	{ebiten.KeyU, "battle.cmd.useitem", "U 使用道具", game.ActionUseItem},
	{ebiten.KeyT, "battle.cmd.turnundead", "T 驅散不死", game.ActionTurnUndead},
	{ebiten.KeyP, "battle.cmd.pray", "P 祈禱", game.ActionPray},
	{ebiten.KeyL, "battle.cmd.leech", "L 汲取法力", game.ActionLeech},
	{ebiten.KeyD, "battle.cmd.dodge", "D 閃避", game.ActionDodge},
	{ebiten.KeyS, "battle.cmd.sound", "S 音效開關", game.ActionSound},
	{ebiten.KeyEscape, "battle.cmd.endturn", "Esc 結束回合", game.ActionEndTurn},
}

// updateBattle 推進戰鬥。
//
// 玩家單位輪到時等玩家下指令；怪物與召喚物由簡單 AI 代打，按空白鍵逐步執行，
// 方便肉眼核對每一步。
func (a *app) updateBattle() error {
	// 噴吐動畫只是往前推格數，**不擋輸入** —— 玩家不必等它跑完。
	if a.breath != nil {
		a.breath.frame++
		if a.breath.done() {
			a.breath = nil
		}
	}

	// 選點要排在回合派發之前。**施法會立刻結束回合**（Spend 的 endsTurn），
	// 所以游標打開時「目前單位」已經換成下一個了 ——
	// 把它放進 updatePlayerTurn 裡永遠等不到按鍵。
	if a.aoe != nil {
		return a.updateAOECursor()
	}

	// 檢視面板不花移動點也不結束回合（原版成本 0），所以它擋在
	// 回合派發之前，關掉之後輪到誰完全沒變。
	if a.examine != nil {
		return a.updateExamine()
	}

	if out := a.battle.Outcome(); out != game.Ongoing {
		// **結算要在等按鍵之前做完。** 原本擺在按下空白鍵之後，
		// 而那一幀馬上就 `a.battle = nil` 回到世界地圖 ——
		// 撿到什麼的那幾行只存在一幀，玩家永遠看不到。
		if !a.settled {
			a.settled = true
			// **戰鬥的持久後果要在這裡寫回隊伍。** 不寫的話打完就滿血
			// 復活，而且「全隊死亡」永遠不成立（`campcast.WriteBackParty`）。
			game.WriteBackParty(a.members, a.battle.Units())
			// **陣型的還原排在判勝負之前**，勝敗都要做 ——
			// 試煉室把陣型借走了（原版 `0x0e004` 也在收尾常式第一行還原）。
			a.restoreProvingFormation()
			if out == game.Victory {
				a.logLine(a.tr.UI("battle.msg.monstersdead", "怪物全滅"))
				a.awardExperience()
				a.awardGold()
				a.awardDrops()
				// 試煉室的那一場：戰勝才算過關（原版 `0x0e1bc`）。
				a.finishProvingRoom()
			} else {
				a.logLine(a.tr.UI("battle.msg.partydead", "隊伍全滅"))
			}
		}
		// 全滅就直接走死亡畫面，不要等玩家按鍵回世界地圖 ——
		// 那條路會讓一支全員陣亡的隊伍繼續走路（`deathui.go`）。
		if out == game.Defeat && a.checkPartyDeath() {
			return nil
		}
		// -autofight 時自動翻過結算畫面，等同玩家按著空白鍵。
		if !inpututil.IsKeyJustPressed(ebiten.KeySpace) && !a.autoAdvance() {
			return nil
		}
		a.battle = nil
		a.settled = false
		return nil
	}

	cur := a.battle.Current()
	if cur == nil {
		a.battle.BeginRound()
		a.logf(a.tr.UI("battle.log.round", "── 第 %d 回合 ──"), a.battle.Round())
		return nil
	}

	if cur.IsPlayer {
		if a.updateAutoFight() {
			return nil
		}
		return a.updatePlayerTurn(cur)
	}
	if !inpututil.IsKeyJustPressed(ebiten.KeySpace) && !a.autoAdvance() {
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
	// plot 非 plotNone 時，這一格不是法術而是主線動作
	// （UNCURSE／IMPRISON，見 campcast.go 的 plotAction）。
	plot plotAction
}

// openSpellMenu 列出施法者目前選得到的法術。
//
// **只列付得起的**：原版在選單外先檢查 SP 是否 >= M，不足就顯示
// 「not enough points」。列出來又不能選比較容易讓玩家困惑，
// 但完全藏起來又看不到目標，所以列出全部、把付不起的標灰。
func (a *app) openSpellMenu(u *game.Unit) {
	m := &spellMenu{caster: u}
	caster := a.characterFor(u)
	for i := 0; i < a.tables.NumSpells(); i++ {
		sp, err := a.tables.Spell(i)
		if err != nil || sp.Empty() {
			continue
		}
		// **只列他學過的那幾系**（手冊 `part-3.md`：「角色學會的符文與
		// 吟唱法術會顯示在畫面底部」）。原本 43 筆全列，於是五系符文
		// 與三個吟唱（幻術／附身／召喚）學不學都一樣。
		// caster 為 nil 代表這不是隊員（召喚／幻術生物，槽位 12–14）——
		// 牠們沒有角色記錄也就沒有技能旗標，**不套這道閘門**。
		// 手冊說召喚物「擁有其真實同類一半的法力值，因此能夠施放法術」。
		if caster != nil && !game.CanCast(caster, i, sp) {
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

// characterFor 回傳這個戰鬥單位對應的隊伍成員，不是隊員就回 nil。
//
// 配對方式與 `game.WriteBackParty` 同一條：`成員索引 = 槽位 − PlayerSlotStart`。
// 召喚／幻術生物（槽位 12–14）沒有角色記錄，回 nil。
func (a *app) characterFor(u *game.Unit) *game.Character {
	if u == nil || u.Slot < game.PlayerSlotStart || u.Slot >= game.PlayerSlotEnd {
		return nil
	}
	i := u.Slot - game.PlayerSlotStart
	if i < 0 || i >= len(a.members) {
		return nil
	}
	return &a.members[i]
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
		a.logf(a.tr.UI("battle.cast.nosp", "%s 的法力不足以施放%s（需要 %d 點）"), u.Name, e.name, e.spell.M)
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

// castWithSP 是玩家在輸入框確認投入點數之後進入選目標。
//
// **行動點與法力都還沒扣** —— 選目標的游標按 Esc 可以反悔，反悔不該
// 算掉一次行動。真正的結算在 commitSpell。
func (a *app) castWithSP(u *game.Unit, e spellEntry, sp int) {
	a.spells = nil

	// 範圍法術選中心點，其餘選單一目標。原版兩者都會跳游標
	// （單體用 FUN_138d_3fc9，回傳目標槽位，-5 代表取消）。
	a.aoe = &aoeCursor{caster: u, entry: e, sp: sp,
		area: e.spell.Effect == game.EffectAOE}
	a.aoeX, a.aoeY = u.X, u.Y
}

// commitSpell 扣行動點與法力並套用效果。游標確認之後才會走到這裡。
func (a *app) commitSpell(c *aoeCursor, target *game.Unit) {
	if _, ok := a.battle.Spend(game.ActionCast); !ok {
		a.logf(a.tr.UI("battle.msg.noap", "%s 行動點不足"), c.caster.Name)
		return
	}
	c.caster.CurrentSP -= c.sp
	a.applySpell(c.caster, target, c.entry, c.sp)
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

	line(fmt.Sprintf(a.tr.UI("battle.cast.casts", "%s 施放%s"), p.caster.Name, p.entry.name))
	line("")
	line(fmt.Sprintf(a.tr.UI("battle.sp.prompt", "投入多少法力？　%d"), p.input.Amount()))
	line("")
	line(fmt.Sprintf(a.tr.UI("battle.sp.range", "可投入 %d–%d（目前法力 %d）"),
		p.input.Min(), p.input.Max(), p.caster.CurrentSP))
	line("")
	line(a.tr.UI("battle.sp.keys1", "數字鍵：輸入　↑↓：加減　Backspace：退格"))
	line(a.tr.UI("battle.sp.keys2", "Enter：施放　Esc：取消"))
	line("")
	line(a.tr.UI("battle.sp.note", "※ 投入越多效果越強，法力用完就沒了"))
}

// applySpell 套用一個法術的效果。
//
// 只實作 spec 標 READY 的四類特殊效果與通式；其餘 effect_type
// 顯示「尚未實作」而不是靜默沒反應 —— 看不出有沒有生效比沒有功能更糟。
func (a *app) applySpell(caster, target *game.Unit, e spellEntry, sp int) {
	switch e.spell.Effect {
	case game.EffectInstantDeath:
		if target == nil {
			a.logf(a.tr.UI("battle.msg.notarget", "%s 正前方沒有目標"), caster.Name)
			return
		}
		if game.CastInstantDeath(a.rng, sp, e.spell, target) {
			a.battle.Kill(target)
			a.speaker.Play(pcspeaker.EffectDeath)
			a.logf(a.tr.UI("battle.msg.spellkilled", "%s 的%s殺死了 %s"), caster.Name, e.name, target.Name)
		} else {
			a.logf(a.tr.UI("battle.msg.spellnoeffect", "%s 的%s沒有奏效"), caster.Name, e.name)
		}

	case game.EffectBindApply:
		if target == nil {
			a.logf(a.tr.UI("battle.msg.notarget", "%s 正前方沒有目標"), caster.Name)
			return
		}
		res := game.CastBind(a.rng, sp, e.spell, target)
		switch {
		case res.Applied:
			a.logf(a.tr.UI("battle.msg.bound", "%s 用%s束縛了 %s（%d 回合）"),
				caster.Name, e.name, target.Name, res.Rounds)
		case res.AlreadyBound:
			a.logf(a.tr.UI("battle.msg.alreadybound", "%s 已經被束縛住了"), target.Name)
		default:
			a.logf(a.tr.UI("battle.msg.resisted", "%s 抵抗了%s"), target.Name, e.name)
		}

	case game.EffectBindRelease:
		if target == nil {
			target = caster
		}
		if game.CastBindRelease(sp, e.spell, target) {
			a.logf(a.tr.UI("battle.msg.unbound", "%s 的束縛被%s解開"), target.Name, e.name)
		} else {
			a.logf(a.tr.UI("battle.msg.unbindweak", "%s 的力度不足以解開束縛"), e.name)
		}

	case game.EffectWither:
		if target == nil {
			a.logf(a.tr.UI("battle.msg.notarget", "%s 正前方沒有目標"), caster.Name)
			return
		}
		if game.CastWither(a.rng, sp, e.spell, target) {
			a.logf(a.tr.UI("battle.msg.withered", "%s 的%s讓 %s 衰朽"), caster.Name, e.name, target.Name)
		} else {
			a.logf(a.tr.UI("battle.msg.spellnoeffect", "%s 的%s沒有奏效"), caster.Name, e.name)
		}

	case game.EffectPossession:
		if target == nil {
			a.logf(a.tr.UI("battle.msg.notarget", "%s 正前方沒有目標"), caster.Name)
			return
		}
		if game.Possess(a.rng, target, sp) {
			a.logf(a.tr.UI("battle.msg.possessed", "%s 的%s奪走了 %s 的心智"), caster.Name, e.name, target.Name)
		} else {
			a.logf(a.tr.UI("battle.msg.resisted", "%s 抵抗了%s"), target.Name, e.name)
		}

	case game.EffectAOE:
		// 走到這裡代表游標已經選好中心點（見 aoeCursor）。
		hits := game.CastAOE(a.rng, a.battle, e.spell, sp, a.aoeX, a.aoeY)
		if len(hits) == 0 {
			a.logf(a.tr.UI("battle.msg.aoenobody", "%s 的%s沒有波及任何人"), caster.Name, e.name)
			return
		}
		a.logf(a.tr.UI("battle.msg.aoecast", "%s 施放%s（波及 %d 人）"), caster.Name, e.name, len(hits))
		for _, h := range hits {
			switch {
			case h.Resisted:
				a.logf(a.tr.UI("battle.msg.immune.indent", "　%s 免疫"), h.Unit.Name)
			case h.Killed:
				a.speaker.Play(pcspeaker.EffectDeath)
				a.logf(a.tr.UI("battle.msg.fell.indent", "　%s 倒下了"), h.Unit.Name)
			case h.Delta < 0:
				a.logf(a.tr.UI("battle.msg.decrease.indent", "　%s 減少 %d"), h.Unit.Name, -h.Delta)
			case h.Delta > 0:
				a.logf(a.tr.UI("battle.msg.increase.indent", "　%s 增加 %d"), h.Unit.Name, h.Delta)
			}
		}

	default:
		// 數值增減類走通式。走不通的 effect_type 才是真的沒實作。
		//
		// **目標一律照呼叫端指定的**。這裡原本有一句「K > 0 就改成對自己」，
		// 那是還沒有選目標游標時的權宜；游標做好之後它變成一個安靜的 bug ——
		// 玩家用游標指了隊友要治療，卻被改成治療自己，畫面上還看不出來。
		// 只有在完全沒有目標時才退回施法者。
		who := target
		if who == nil {
			who = caster
		}
		delta, ok := game.CastMagnitudeEffect(a.rng, sp, e.spell, who)
		if !ok {
			a.logf(a.tr.UI("battle.msg.effectunimpl", "%s 施放%s —— 這類效果尚未實作"), caster.Name, e.name)
			return
		}
		if who == nil {
			a.logf(a.tr.UI("battle.msg.notarget", "%s 正前方沒有目標"), caster.Name)
			return
		}
		switch {
		case delta > 0:
			a.logf(a.tr.UI("battle.msg.buff", "%s 的%s讓 %s 增加 %d"), caster.Name, e.name, who.Name, delta)
		case delta < 0:
			a.logf(a.tr.UI("battle.msg.debuff", "%s 的%s讓 %s 減少 %d"), caster.Name, e.name, who.Name, -delta)
			if !who.Alive() {
				a.battle.Kill(who)
				a.speaker.Play(pcspeaker.EffectDeath)
				a.logf(a.tr.UI("battle.msg.fell", "%s 倒下了"), who.Name)
			}
		default:
			a.logf(a.tr.UI("battle.msg.nonoticeable", "%s 的%s沒有明顯效果"), caster.Name, e.name)
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

	line(fmt.Sprintf(a.tr.UI("battle.cast.header", "%s 施法　法力 %d"), m.caster.Name, m.caster.CurrentSP))
	line("")
	line(a.tr.UI("battle.cast.columns", "   法術            最低法力"))

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
			note = a.tr.UI("battle.cast.nosp.mark", "  法力不足")
		}
		line(fmt.Sprintf("%s%s %5d%s", mark,
			textlayout.PadCells(e.name, 14), e.spell.M, note))
	}
	line("")
	line(a.tr.UI("battle.cast.keys", "↑↓：選擇　Enter：施放　Esc：取消"))
	line(a.tr.UI("battle.cast.note", "※ 選好按 Enter 再決定投入多少法力"))
}

// aoeCursor 是範圍法術的選點狀態。
//
// 法力已經在開游標之前扣掉了 —— 原版也是先扣再選點，
// 取消選點不會退還。這裡照做，並在畫面上寫明。
type aoeCursor struct {
	caster *game.Unit
	entry  spellEntry
	sp     int

	// area 為真代表這是範圍法術，游標選的是中心點；否則是選單一目標，
	// 必須指到某個單位身上才成立。
	area bool
}

// updateAOECursor 讓玩家用方向鍵挪動游標（範圍法術是中心點，其餘是目標）。
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
	// Esc 取消：行動點與法力都還沒扣，什麼都沒發生（原版的游標回 -5）。
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.aoe = nil
		a.logLine(a.tr.UI("battle.cast.cancelled", "取消施法"))
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		c := a.aoe
		if c.area {
			a.aoe = nil
			a.commitSpell(c, nil)
			return nil
		}
		// 單體法術要真的指到一個人身上才算數。
		target := a.battle.UnitAt(a.aoeX, a.aoeY)
		if target == nil {
			a.logLine(a.tr.UI("battle.msg.emptycell", "那一格沒有人"))
			return nil
		}
		a.aoe = nil
		a.commitSpell(c, target)
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
			a.logf(a.tr.UI("battle.msg.noap", "%s 行動點不足"), u.Name)
			return nil
		}
		a.useItem(u, e)
	}
	return nil
}

// useItem 套用一件道具的效果。
//
// 道具槽 `+0x07` 是效果索引，`+0x08` 是強度，兩者一起餵進與法術**同一套**
// 效果記錄與套用路徑（原版 `17c5:19dd`–`19f2` 呼叫的 `FUN_1000_114f`
// 就是法術用的那個載入函式，見 scenario/inventory.go）。
//
// 目標暫時取正前方的敵人 —— 原版走的是與法術共用的目標挑選
// （`FUN_138d_3fc9`，會跳游標），那一段還沒接到道具這條路上。
func (a *app) useItem(u *game.Unit, e game.UsableItem) {
	sp, err := a.tables.Spell(e.Item.Effect)
	if err != nil {
		a.logf(a.tr.UI("battle.item.noeffectrecord", "%s 使用 %s，但效果 %d 查不到記錄"),
			u.Name, a.itemLabel(e.Item), e.Item.Effect)
		return
	}
	name, err := a.strings.SpellName(e.Item.Effect)
	if err != nil {
		name = fmt.Sprintf(a.tr.UI("battle.item.effect", "效果 %d"), e.Item.Effect)
	}
	entry := spellEntry{
		index: e.Item.Effect,
		name:  a.tr.Event(spellSourceFile, e.Item.Effect, name),
		spell: sp,
	}
	a.logf(a.tr.UI("battle.item.used", "%s 使用 %s"), u.Name, a.itemLabel(e.Item))
	a.applySpell(u, a.battle.TargetInFront(u), entry, e.Item.Power)

	// 用掉一次。次數用完之後這一格就不再出現在選單裡（InventorySlot.Usable）。
	if c := u2c(a.members, u); c != nil {
		c.Inventory[e.Slot].Used++
	}
}

// itemSourceFile 是道具名稱翻譯目錄的 key，與 dwstrings 產生時一致。
const itemSourceFile = "ITEMS.DAT"

// itemLabel 回傳道具的顯示名稱（已翻譯）。
func (a *app) itemLabel(it scenario.InventorySlot) string {
	// 地城道具的 17 bytes 是名字不是 ITEMS.DAT 索引 —— 拿 0xfe 去查表
	// 只會得到「道具 254」。手冊說它們在清單裡前面加 `/`。
	if it.Dungeon() {
		return a.dungeonName(it.DungeonName)
	}
	item, err := a.items.ByIndex(int(it.Type))
	if err != nil {
		return fmt.Sprintf(a.tr.UI("battle.item.itemnum", "道具 %d"), it.Type)
	}
	name := a.tr.Event(itemSourceFile, int(it.Type), item.Name)
	if it.Enchant != 0 {
		return fmt.Sprintf("%s%+d", name, it.Enchant)
	}
	return name
}

// drawItemMenu 畫使用道具選單。
func (a *app) drawItemMenu(dst *ebiten.Image) {
	m := a.useMenu
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.BoxPadX, y)
		y += ui.LineHeight
	}

	line(fmt.Sprintf(a.tr.UI("battle.item.header", "%s 使用道具"), m.caster.Name))
	line("")
	for i, e := range m.entries {
		mark := "   "
		if i == m.cursor {
			mark = " > "
		}
		state := fmt.Sprintf(a.tr.UI("battle.item.charges", "（剩 %d 次）"), e.Item.Total-e.Item.Used)
		if !e.Item.Identified {
			state = a.tr.UI("battle.item.unidentified", "（未鑑定）")
		}
		line(fmt.Sprintf("%s%s%s", mark,
			textlayout.PadCells(a.itemLabel(e.Item), 14), state))
	}
	line("")
	line(a.tr.UI("battle.item.keys", "↑↓：選擇　Enter：使用　Esc：取消"))
	line("")
	line(a.tr.UI("battle.item.note1", "※ 只列得出已裝備的武器／護甲與消耗品，而且要還有次數"))
	line(a.tr.UI("battle.item.note2", "　 原版的 Use 就是拿來觸發已裝備裝備的特殊能力"))
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
			a.logf(a.tr.UI("battle.msg.blocked", "%s 前方過不去"), u.Name)
		}
		return nil
	}
	// `?` 檢視（原版 case 10，成本 0）。實體鍵是 `/`，加不加 Shift 都收。
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) {
		a.openExamine()
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
			a.logf(a.tr.UI("battle.msg.blocked", "%s 前方過不去"), u.Name)
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
		a.logf(a.tr.UI("battle.msg.noap", "%s 行動點不足"), u.Name)
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
			a.logf(a.tr.UI("battle.msg.nosp", "%s 沒有法力"), u.Name)
			return
		}
		a.openSpellMenu(u)

	case game.ActionUseItem:
		items := u2c(a.members, u).UsableItems()
		if len(items) == 0 {
			a.logf(a.tr.UI("battle.msg.noitem", "%s 沒有可用的道具"), u.Name)
			return
		}
		a.useMenu = &itemMenu{caster: u, entries: items}

	case game.ActionAttack:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf(a.tr.UI("battle.msg.noenemyahead", "%s 正前方沒有敵人"), u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf(a.tr.UI("battle.msg.noap", "%s 行動點不足"), u.Name)
			return
		}
		a.reportAttack(u, target, a.battle.ResolveAttack(u, target, 0))

	case game.ActionTurnUndead:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf(a.tr.UI("battle.msg.noenemyahead", "%s 正前方沒有敵人"), u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf(a.tr.UI("battle.msg.noap", "%s 行動點不足"), u.Name)
			return
		}
		if game.TurnUndead(a.rng, u, target) {
			a.battle.Kill(target)
			a.logf(a.tr.UI("battle.msg.turned", "%s 驅散了 %s"), u.Name, target.Name)
		} else {
			a.logf(a.tr.UI("battle.msg.turnfailed", "%s 的驅散無效"), u.Name)
		}

	case game.ActionPray:
		if _, ok := a.battle.Spend(act); !ok {
			a.logf(a.tr.UI("battle.msg.noap", "%s 行動點不足"), u.Name)
			return
		}
		granted, next := game.Pray(a.rng, a.prayChance)
		a.prayChance = next
		if granted {
			a.logf(a.tr.UI("battle.msg.prayanswered", "%s 的祈禱得到回應"), u.Name)
		} else {
			a.logf(a.tr.UI("battle.msg.praynotanswered", "%s 的祈禱沒有回應"), u.Name)
		}

	case game.ActionLeech:
		target := a.battle.TargetInFront(u)
		if target == nil {
			a.logf(a.tr.UI("battle.msg.noenemyahead", "%s 正前方沒有敵人"), u.Name)
			return
		}
		if _, ok := a.battle.Spend(act); !ok {
			a.logf(a.tr.UI("battle.msg.noap", "%s 行動點不足"), u.Name)
			return
		}
		if ok, amount := game.Leech(a.rng, u, target.CurrentSP); ok {
			target.CurrentSP -= amount
			u.CurrentSP += amount
			a.logf(a.tr.UI("battle.msg.leeched", "%s 汲走 %s 的 %d 點法力"), u.Name, target.Name, amount)
		} else {
			a.logf(a.tr.UI("battle.msg.leechfailed", "%s 的汲取失敗"), u.Name)
		}

	case game.ActionDodge:
		bonus := a.battle.DoDodge()
		a.logf(a.tr.UI("battle.msg.dodging", "%s 進入閃避（命中率 %d）"), u.Name, game.DodgeHitModifier(bonus))

	case game.ActionSound:
		// 對應原版的 Sound on／Sound off（旗標 [0x1585]）。不耗行動點。
		a.speaker.SetEnabled(!a.speaker.Enabled())
		if a.speaker.Enabled() {
			a.logLine(a.tr.UI("battle.msg.soundon", "音效開啟"))
		} else {
			a.logLine(a.tr.UI("battle.msg.soundoff", "音效關閉"))
		}

	case game.ActionEndTurn:
		a.battle.Spend(act)
		a.logf(a.tr.UI("battle.msg.endturn", "%s 結束回合"), u.Name)
	}
}

// monsterTurn 讓一隻怪物行動。
//
// 順序照原版的決策樹（見 game 套件的 ai.go）：先看噴吐、再看施法，
// 最後走近戰。
//
// 近戰的走位（轉向、逼近）是本作自己的補充 —— 原版那一段還沒讀，
// 這裡先讓怪物走得到玩家面前，戰鬥才跑得完。
func (a *app) monsterTurn(u *game.Unit) {
	if a.monsterBreathe(u) {
		return
	}
	if a.monsterCast(u) {
		return
	}
	if target := a.battle.TargetInFront(u); target != nil {
		if _, ok := a.battle.Spend(game.ActionAttack); ok {
			a.reportAttack(u, target, a.battle.ResolveAttack(u, target, 0))
			return
		}
	}

	// 咬著同一個目標打，死了才隨機換人 —— 原版就是這樣（見 game.AITarget）。
	// 每回合都挑「第一個敵人」會讓整群怪物擠在同一個隊員身上。
	target := a.battle.AITarget(u)
	if target == nil {
		a.battle.Spend(game.ActionEndTurn)
		return
	}
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

// monsterBreathe 讓會噴吐的怪物有機會噴吐，回報這一回合是否已經用掉。
//
// 原版（0x7939–0x7959）：種族／元素類型落在 8–12 且 `rnd(10) < 4` 才走這一支。
// 誤傷太多時原版會整個放棄（game.BreathPlan.Veto），那一回合就改做別的事。
func (a *app) monsterBreathe(u *game.Unit) bool {
	if u.RaceOrElement < 8 || u.RaceOrElement > 12 || a.rng.Roll(10) >= 4 {
		return false
	}
	if a.battle.BreathPlan(u).Veto() {
		return false
	}
	if _, ok := a.battle.Spend(game.ActionAttack); !ok {
		return false
	}
	cone := a.battle.BreathCone(u)
	hits := a.battle.Breathe(u)
	if len(hits) == 0 {
		return false
	}
	a.breath = &breathAnim{cells: cone}
	a.speaker.Play(pcspeaker.EffectDeath)
	a.logf(a.tr.UI("battle.msg.breath", "%s 噴出吐息（波及 %d 人）"), u.Name, len(hits))
	for _, h := range hits {
		switch {
		case h.Killed:
			a.logf(a.tr.UI("battle.msg.fell.indent", "　%s 倒下了"), h.Unit.Name)
		case h.Damage == 0:
			a.logf(a.tr.UI("battle.msg.immune.indent", "　%s 免疫"), h.Unit.Name)
		default:
			a.logf(a.tr.UI("battle.msg.breathdamage.indent", "　%s 受到 %d 點傷害"), h.Unit.Name, h.Damage)
		}
	}
	return true
}

// monsterCast 讓會法術的怪物有機會施法，回報這一回合是否已經用掉。
//
// 原版（0x799a–0x7978）：法力 > 0 且 rnd(10) > 4 才進 AI 選法術。
// 選法本身照 game.AISpellChoice（兩層擲點 + 區間表）。
//
// 技能檢查傳 nil：原版只在 `unit+0x20 == 2`（＝被魅惑的玩家角色）那一支
// 查符文系技能旗標，一般怪物不查（見 game/side.go）。這裡的施法者都是
// 一般怪物，所以不檢查是對的。
//
// 目標由效果類型與 K 的正負決定，見 game.AISpellTargetsOwnSide。
func (a *app) monsterCast(u *game.Unit) bool {
	if u.CurrentSP <= 0 || a.rng.Roll(10) <= 4 {
		return false
	}
	id, ok := game.AISpellChoice(a.rng, a.tables, u.CurrentSP, nil)
	if !ok {
		return false
	}
	sp, err := a.tables.Spell(id)
	if err != nil {
		return false
	}
	if !game.AIEffectHandled(sp.Effect) {
		return false // 跳表的 default：這個效果 AI 不會用
	}
	target := a.monsterCastTarget(u, sp)
	if target == nil {
		return false
	}
	if _, ok := a.battle.Spend(game.ActionCast); !ok {
		return false
	}

	// AI 不會把法力一次燒光 —— 投入量照原版的公式算（見 AISpellInvestment）。
	invested := game.AISpellInvestment(a.rng, u.CurrentSP, sp.M)
	u.CurrentSP -= invested

	name, err := a.strings.SpellName(id)
	if err != nil {
		name = fmt.Sprintf(a.tr.UI("battle.spell.num", "法術 %d"), id)
	}
	e := spellEntry{index: id, name: a.tr.Event(spellSourceFile, id, name), spell: sp}
	a.applySpell(u, target, e, invested)
	return true
}

// monsterCastTarget 依效果類型挑這次施法要打誰。
//
// 效果 1 是範圍：先隨機挑一個敵人當中心，數過方框裡兩邊各幾個，誤傷太多
// 就放棄（原版 0x09ae 的 `己方×2 > 敵方`）。這裡回傳的是中心那個單位，
// 範圍效果本身還沒接上，先以單體套用 —— 缺的是效果套用端，不是選目標端。
func (a *app) monsterCastTarget(u *game.Unit, sp gamedata.Spell) *game.Unit {
	if game.AIEffectIsArea(sp.Effect) {
		center := a.battle.AIPickTarget(u, false, -1)
		if center == nil {
			return nil
		}
		if a.battle.AIAreaCountAt(u, center.X, center.Y, sp.School).Veto() {
			return nil // 會炸到太多自己人
		}
		return center
	}

	ownSide := game.AISpellTargetsOwnSide(sp.Effect, sp.K)
	maxStatus := -1
	if ownSide {
		maxStatus = 1 // 0x0a7c：增益不往已經倒下的自己人身上放
	}
	return a.battle.AIPickTarget(u, ownSide, maxStatus)
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
		a.logf(a.tr.UI("battle.msg.miss", "%s 落空"), attacker.Name)
	case res.NoEffect:
		a.logf(a.tr.UI("battle.msg.noeffect", "%s 對 %s 無效"), attacker.Name, target.Name)
	case res.Killed:
		a.speaker.Play(pcspeaker.EffectDeath)
		a.logf(a.tr.UI("battle.msg.killed", "%s 擊殺 %s（%d 點）"), attacker.Name, target.Name, res.Damage)
	default:
		a.speaker.Play(pcspeaker.EffectG3)
		verb := a.tr.UI("battle.hit.normal", "命中")
		if res.Critical {
			verb = a.tr.UI("battle.hit.critical", "重擊")
		}
		a.logf(a.tr.UI("battle.msg.hit", "%s %s %s %d 點"), attacker.Name, verb, target.Name, res.Damage)
	}
}

// aoeColor 是範圍法術的選取框顏色。
var aoeColor = color.RGBA{0xff, 0x55, 0x55, 0xff}

// breathColor 是噴吐動畫的填色。
//
// 原版是拿法術記錄的 school（`ds:0x4e2c`）去查一個圖塊逐格畫
// （`138d:1a59`），那個圖塊的來源還沒解；本專案先用單色方塊，
// 形狀與順序照原版（`game.BreathCone`）。
var breathColor = color.RGBA{0xff, 0xaa, 0x22, 0xff}

// breathAnim 是噴吐動畫的狀態：整個錐形逐格點亮，亮完就消失。
type breathAnim struct {
	cells []game.BreathCell
	frame int
}

// breathFramesPerCell 是每一格停留幾個 frame。
//
// **原版的每格延遲沒解出來**（`138d:1a59` 那個逐格畫的迴圈裡看不到延遲，
// 速度取決於當年的機器）。3 frame 是本專案挑的：60fps 之下 16 格的錐形
// 約 0.8 秒，看得清楚是從嘴邊往外擴，又不至於讓人等。
const breathFramesPerCell = 3

// shown 回傳目前該點亮到第幾格（含）。
func (b *breathAnim) shown() int { return b.frame / breathFramesPerCell }

// done 回報整個錐形都亮完了沒有。
func (b *breathAnim) done() bool { return b.shown() >= len(b.cells)+2 }

// drawBattlefield 畫戰鬥網格：雙方單位的位置與面向。
//
// **底圖是空的。** 手冊說戰場是「該區域的放大地圖」，但那張放大圖怎麼生成
// 還沒反組譯出來。畫格線比畫一張猜的地形圖誠實 ——
// 玩家至少看得出誰站哪一格。
// battleCam 回傳 9×9 視窗的左上角。
//
// 戰場是 15×15，視窗只有 9×9 —— 所以畫面得跟著人跑。原版也是這樣：
// `FUN_222f_1404(中心X − 4, 中心Y − 4)`，中心每步追向行動中的單位
// （`[0x50f0] += sign(單位X − [0x50f0])`，見 docs/re/35）。
//
// 這裡直接對準行動單位，不做逐格追趕 —— 追趕的動畫感在無頭截圖裡看不出來，
// 而且會讓「現在在看誰」變得不確定。夾在牆框之內，視窗不會飄到空白區。
func (a *app) battleCam() (x, y int) {
	// 視窗格數是版面常數，不從像素寬回推 —— 回推會跟著素材尺寸變，
	// 而「視窗開幾格」是規則（原版固定 9×9），與一格幾像素無關。
	cols, rows := layout.ViewTilesX, layout.ViewTilesY

	cx, cy := game.BattleCentreX, game.BattleCentreY
	if cur := a.battle.Current(); cur != nil && cur.Alive() {
		cx, cy = cur.X, cur.Y
	}
	return clampCam(cx-cols/2, cols), clampCam(cy-rows/2, rows)
}

// clampCam 把視窗夾在牆框之內（含牆，牆是戰場的一部分）。
func clampCam(v, span int) int {
	lo, hi := game.BattleWallLow, game.BattleWallHigh-span+1
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (a *app) drawBattlefield(dst *ebiten.Image) {
	cellW, cellH, scale := a.tileMetrics()
	cur := a.battle.Current()
	camX, camY := a.battleCam()

	// 先鋪地形。地形是 3×3 個世界 tile 各放大 5×5 拼出來的
	// （docs/re/36），外圍那圈牆也是地形的一部分。
	ts := a.tileset()
	cols, rows := layout.ViewTilesX, layout.ViewTilesY
	for vy := 0; vy < rows; vy++ {
		for vx := 0; vx < cols; vx++ {
			// 畫不畫要看座標，**不能看地形值是不是 0** —— 0 是合法的
			// 世界 tile，大地圖上到處都是（見 game.InArena 的說明）。
			tx, ty := camX+vx, camY+vy
			if a.battleTerrain != nil && game.InArena(tx, ty) {
				v := a.battleTerrain.TileAt(tx, ty)
				if img := ts.Tile(v & 0x7f); img != nil {
					ui.DrawImageScaled(dst, img,
						layout.MapOriginX+vx*cellW, layout.MapOriginY+vy*cellH, scale)
				}
			}
		}
	}

	if a.breath != nil {
		// 逐格點亮，順序照原版的掃描 —— 看起來是從嘴邊往外擴。
		for i, c := range a.breath.cells {
			if i > a.breath.shown() {
				break
			}
			ui.FillRect(dst,
				layout.MapOriginX+(c.X-camX)*cellW,
				layout.MapOriginY+(c.Y-camY)*cellH,
				cellW, cellH, breathColor)
		}
	}

	if a.aoe != nil {
		if a.aoe.area {
			// 5×5 的效果範圍畫出來，玩家才知道會掃到誰（包含自己人）。
			r := game.AOERadius
			ui.StrokeRect(dst,
				layout.MapOriginX+(a.aoeX-r-camX)*cellW,
				layout.MapOriginY+(a.aoeY-r-camY)*cellH,
				(2*r+1)*cellW, (2*r+1)*cellH, aoeColor)
		}
		ui.StrokeRect(dst,
			layout.MapOriginX+(a.aoeX-camX)*cellW,
			layout.MapOriginY+(a.aoeY-camY)*cellH,
			cellW, cellH, aoeColor)
	}

	for _, u := range a.battle.Units() {
		if !u.Alive() {
			continue
		}
		x := layout.MapOriginX + (u.X-camX)*cellW
		y := layout.MapOriginY + (u.Y-camY)*cellH
		// 上下界用**實際畫出來的**格數算，不用 layout.MapHeight ——
		// EGA 一格 28 高，9 格是 252 而不是 288，用後者會讓單位標記
		// 畫到地圖框外面。
		if x < layout.MapOriginX || x >= layout.MapOriginX+layout.ViewTilesX*cellW ||
			y < layout.MapOriginY || y >= layout.MapOriginY+layout.ViewTilesY*cellH {
			continue
		}
		a.drawBattleUnitSprite(dst, u, x, y, scale)
		if u == cur {
			ui.StrokeRect(dst, x, y, cellW, cellH, markerColor)
		} else if u.IsPlayer {
			ui.StrokeRect(dst, x, y, cellW, cellH, partyColor)
		} else {
			ui.StrokeRect(dst, x, y, cellW, cellH, enemyColor)
		}
		// 檢視游標。原版是**反白**（`ds:0x5192 = 0xff` 之後重畫那一格，
		// 與 `L` 查看陷阱的白框同一個機制）；這裡先用白框，
		// 因為文字反白要動到字型繪製那一層。
		if a.examine != nil && u.Slot == a.examine.slot() {
			ui.StrokeRect(dst, x, y, cellW, cellH, trapMarkerColor)
		}
	}
	drawMapFrame(dst, cellW, cellH)
}

// drawBattleUnitSprite 把原版的兩條戰鬥圖像路徑接回來：
//
//   - 隊員：COMBAT.SHP/SHE 的職業／面向 glyph（17c5:0def–0e29）
//   - 怪物與召喚物：MONSTER.SHP/SHE，每種外觀 8 幀
//
// 兩者都整格覆寫地形，包含素材原有的黑底；這與原版把圖塊索引直接寫進
// 戰場緩衝的做法一致。
func (a *app) drawBattleUnitSprite(dst *ebiten.Image, u *game.Unit, x, y, scale int) {
	if u.Slot >= game.PlayerSlotStart && u.Slot < game.PlayerSlotEnd {
		member := u.Slot - game.PlayerSlotStart
		if member >= 0 && member < len(a.members) {
			frame := 0x14 + (u.Facing&3)*2
			class := int(a.members[member].Class)
			switch {
			case class > 5:
				frame += 8
			case class > 2:
				frame += 0x10
			}
			if img := a.combatSprites.Frame(frame); img != nil {
				ui.DrawImageScaled(dst, img, x, y, scale)
				return
			}
		}
	}

	// 每組八幀＝四個方向各兩個姿勢。素材順序肉眼與原版載入單位共同確認：
	// 南 0/1、西 2/3、東 4/5、北 6/7。
	pair := [...]int{6, 4, 0, 2}
	frame := u.SpriteIndex*8 + pair[u.Facing&3]
	if a.battle != nil {
		frame += a.battle.Round() & 1
	}
	if img := a.monsterSprites.Frame(frame); img != nil {
		ui.DrawImageScaled(dst, img, x, y, scale)
	}
}

// drawBattle 畫戰鬥狀態：行動點、單位血量、最近幾行紀錄、可用指令。
func (a *app) drawBattle(dst *ebiten.Image) {
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	// 檢視面板佔滿整個側欄 —— 原版也是把戰場資訊換掉，不是疊一個小框。
	if a.examine != nil {
		a.drawExamine(line)
		return
	}

	cur := a.battle.Current()
	if cur != nil {
		line(fmt.Sprintf(a.tr.UI("battle.header.actor", "%s 行動　%d 點"), cur.Name, a.battle.Points()))
	}

	// 戰鬥紀錄已移到地圖下方；右欄只留行動者與單位名單。
	const headerLines = 1
	rows := (layout.MenuY-layout.StatusY)/ui.LineHeight - headerLines
	units := a.battle.Units()
	hidden := 0
	if len(units) > rows {
		// 「另有 N 個」自己也佔一列，先替它留位，不能讓最後一列
		// 與 y=MenuY 的紅底選單互相覆蓋。
		rows--
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
			state = a.tr.UI("battle.unit.dead", " 陣亡")
		}
		// 中文名要按「格」截、按「格」補 —— `name[:8]` 會切在字元中間，
		// `%-8s` 依位元組補會讓數字欄歪掉（見 textlayout.PadCells）。
		line(fmt.Sprintf("%s%s%3d/%-3d%s", tag,
			textlayout.PadCells(u.Name, 8), u.HP, u.MaxHP, state))
	}
	if hidden > 0 {
		line(fmt.Sprintf(a.tr.UI("battle.unit.more", " …另有 %d 個單位"), hidden))
	}

}

// drawBattleCommands 把可用指令畫在文字視窗的位置。
func (a *app) drawBattleCommands(dst *ebiten.Image) {
	cur := a.battle.Current()
	y := a.logTop() + layout.BoxPadY

	if a.battle.Outcome() != game.Ongoing {
		a.font.Draw(dst, a.tr.UI("battle.over.keys", "戰鬥結束　空白鍵：繼續"), layout.BoxPadX, y)
		return
	}
	if a.aoe != nil {
		what, note := a.tr.UI("battle.aim.single", "選目標"), a.tr.UI("battle.aim.single.note", "※ 指到誰就對誰施放，Esc 可以反悔")
		if a.aoe.area {
			what, note = a.tr.UI("battle.aim.area", "選 5×5 的中心"), a.tr.UI("battle.aim.area.note", "※ 範圍內敵我都會被波及，Esc 可以反悔")
		}
		a.font.Draw(dst, fmt.Sprintf(a.tr.UI("battle.aim.keys", "%s：%s　方向鍵：移動　Enter：施放　Esc：取消"),
			a.aoe.entry.name, what), layout.BoxPadX, y)
		a.font.Draw(dst, note, layout.BoxPadX, y+ui.LineHeight)
		return
	}
	if cur == nil || !cur.IsPlayer {
		a.font.Draw(dst, a.tr.UI("battle.monster.keys", "空白鍵：讓對方行動"), layout.BoxPadX, y)
		return
	}

	// 付不起的指令仍保留在紅底選單，以棋盤網點表示不可用。
	items := make([]ui.MenuItem, len(playerCommands))
	for i, c := range playerCommands {
		items[i] = ui.MenuItem{
			Label:   a.tr.UI(c.uikey, c.label),
			Enabled: a.battle.CanAct(c.action),
		}
	}
	ui.DrawMenuList(dst, a.font, items, -1,
		layout.MenuX, layout.MenuY, layout.MenuW)
	a.font.Draw(dst, a.tr.UI("battle.move.keys", "方向鍵：轉向／前進　Enter：前進　? 檢視"),
		layout.StatusX, layout.MenuY+len(items)*ui.LineHeight+ui.LineHeight)
}

// awardExperience 發放戰鬥勝利的經驗值（`docs/re/56`）。
//
// 總經驗是每隻打倒的怪物的 `MONSTER.DAT` 經驗值相加，**除以隊伍人數**
// 之後每人各拿一份 —— 原版的訊息就叫 "Exp per chr"。
// 被束縛或死亡的隊員拿不到，而且**那一份直接消失**（分母是全隊人數）。
func (a *app) awardExperience() {
	total := 0
	for _, u := range a.battle.Units() {
		if u == nil || u.IsPlayer || u.Alive() {
			continue
		}
		total += u.Experience
	}
	if total == 0 {
		return
	}
	statuses := make([]game.UnitStatus, len(a.members))
	for i := range a.members {
		statuses[i] = game.UnitStatus(a.members[i].Status)
	}
	if per := game.AwardBattleExp(a.members, statuses, total); per > 0 {
		a.logf(a.tr.UI("battle.drop.exp", "每人獲得 %d 點經驗"), per)
	}
}

// awardGold 發放戰鬥勝利的金幣（`docs/re/56` §3）。
//
// 每隻怪物各出 `1.7^level + Roll(2.1^level) + 3`。原版掃的是全部怪物
// 單位而不是死掉的那些，這裡照做 —— 勝利的條件就是全滅，等價。
//
// 金幣進隊伍共有的錢包（存檔 `+0x0a`，4 bytes），封頂 `0x00FFFFFF`。
func (a *app) awardGold() {
	var levels []int
	for _, u := range a.battle.Units() {
		if u == nil || u.IsPlayer {
			continue
		}
		levels = append(levels, u.Level)
	}
	gold := game.BattleGold(a.rng, levels)
	if gold <= 0 {
		return
	}
	a.setGold(game.CapValue(a.save.Gold + gold))
	a.logf(a.tr.UI("battle.drop.gold", "撿到 %d 枚金幣"), gold)
}

// awardDrops 發放戰鬥勝利的戰利品。
//
// 每隻死掉的怪物各自擲一次（`rnd(100) <= 等級×6 + 5`，見 game.RollBattleDrops），
// 中的那隻掉一件隨機型別的道具。**掉的東西是未鑑定的**，市集才鑑定得出來。
//
// 隊伍道具欄滿了就掉在地上 —— 原版印 "No more room!"，這裡照做。
func (a *app) awardDrops() {
	var levels []int
	for _, u := range a.battle.Units() {
		if u == nil || u.IsPlayer || u.Alive() {
			continue
		}
		levels = append(levels, u.Level)
	}
	drops := game.RollBattleDrops(a.rng, a.tables, a.items, levels)
	if len(drops) == 0 {
		return
	}
	for _, slot := range drops {
		member, idx := a.freeInventorySlot()
		if member < 0 {
			a.logf(a.tr.UI("battle.drop.noroom", "撿到%s，但沒有人放得下了"), a.itemLabel(slot))
			continue
		}
		a.members[member].Inventory[idx] = slot
		// 察覺靈光（精靈天生能力）：有機率在戰利品後面標一句
		// 「偵測到靈光」。**擲點在看道具之前**，見 game.DetectAura。
		aura := ""
		if game.DetectAura(a.rng, a.members, slot) {
			aura = "　" + a.tr.UI("battle.aura", "（偵測到靈光）")
		}
		a.logf(a.tr.UI("battle.drop.picked", "%s 撿到%s%s"), a.members[member].Name, a.itemLabel(slot), aura)
	}
}

// freeInventorySlot 找出全隊第一個空的道具格，找不到回 (-1, -1)。
func (a *app) freeInventorySlot() (member, slot int) {
	for i := range a.members {
		if s := a.members[i].FreeSlot(); s >= 0 {
			return i, s
		}
	}
	return -1, -1
}
