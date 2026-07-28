package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 城鎮設施的操作層。
//
// 價格早就算得出來，但在這一輪之前**每個設施都只是一張價目表** ——
// 看得到治療要多少錢，卻治不了。這裡把五個設施接成真的能用：
// 治療所、酒館、神殿、公會、市集出售。
//
// ⚠ 這裡原本寫「碼頭還是只有報價，船隻陣列還沒解出來」——**已經過期**：
// `internal/assets/scenario/ship.go` 解完了，買船與修船都在 `townui.go` 接上。
// 手冊列的八個設施現在全部可用（盤點見 `docs/manual-coverage.md` §4）。

// amountInput 是「輸入一個數字」的小輸入框。
//
// 捐獻與買糧都要問數量，原版也都是先印提示再讀一行數字。
type amountInput struct {
	prompt string
	digits string
	// max 是允許的最大值，超過就不再吃新的數字。
	max int
	// apply 在按下 Enter 時執行。
	apply func(n int)
}

// value 回傳目前輸入的數字。
func (in *amountInput) value() int {
	n := 0
	for _, r := range in.digits {
		n = n*10 + int(r-'0')
	}
	return n
}

// update 處理數字輸入。回傳 true 代表輸入框關掉了。
func (in *amountInput) update() bool {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		return true
	case inpututil.IsKeyJustPressed(ebiten.KeyBackspace):
		if n := len(in.digits); n > 0 {
			in.digits = in.digits[:n-1]
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if in.value() > 0 {
			in.apply(in.value())
		}
		return true
	}
	for d := 0; d <= 9; d++ {
		if !inpututil.IsKeyJustPressed(ebiten.Key0 + ebiten.Key(d)) {
			continue
		}
		next := in.digits + string(rune('0'+d))
		// 開頭的 0 沒有意義，而且會讓「最多幾位數」的判斷難寫。
		if next == "0" {
			continue
		}
		n := 0
		for _, r := range next {
			n = n*10 + int(r-'0')
		}
		if n <= in.max {
			in.digits = next
		}
	}
	return false
}

// 多收一個 *app 才拿得到 translator（呼叫端只有 townui.go 一處）。
func (in *amountInput) draw(a *app, line func(string)) {
	line(in.prompt)
	line("")
	shown := in.digits
	if shown == "" {
		shown = "_"
	}
	line("　" + shown)
	line("")
	line(a.tr.UI("townsvc.amount.keys", "數字鍵輸入　Backspace：刪除　Enter：確定　Esc：取消"))
}

// sellPicker 是出售的兩層選擇器：先選人、再選要賣哪一格。
type sellPicker struct {
	member int
	// slot 為 -1 代表還在選人。
	slot int
}

// --- 設施內的隊員游標 ---

// moveMemberCursor 處理 ↑↓ 選隊員，回傳是否有動過。
func (a *app) moveMemberCursor(t *townScreen) bool {
	n := len(a.members)
	if n == 0 {
		return false
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		t.member = (t.member + 1) % n
		return true
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		t.member = (t.member - 1 + n) % n
		return true
	}
	return false
}

// currentMember 回傳設施內游標指的隊員，隊伍是空的則回 nil。
func (a *app) currentMember(t *townScreen) *game.Character {
	if t.member < 0 || t.member >= len(a.members) {
		return nil
	}
	return &a.members[t.member]
}

// memberMark 是隊員清單的游標記號。
func memberMark(cur, i int) string {
	if cur == i {
		return " > "
	}
	return "   "
}

// --- 治療所 ---

func (a *app) healCurrent(t *townScreen) {
	c := a.currentMember(t)
	if c == nil {
		return
	}
	svc, res := game.Heal(t.visit.Economy, c, a.gold())
	if !res.OK {
		if res.Cost > 0 {
			t.message = fmt.Sprintf(a.tr.UI("townsvc.heal.cost_note", "%s（%s 要 %d 金）"),
				res.Reason, a.healerServiceName(svc), res.Cost)
			return
		}
		t.message = res.Reason
		return
	}
	a.setGold(res.Gold)
	t.message = fmt.Sprintf(a.tr.UI("townsvc.heal.done", "%s %s，付 %d 金（剩 %d）"),
		c.Name, a.healerServiceName(svc), res.Cost, res.Gold)
}

// --- 酒館 ---

func (a *app) openRationInput(t *townScreen) {
	unit := t.visit.Economy.RationUnitPrice()
	t.amount = &amountInput{
		prompt: fmt.Sprintf(a.tr.UI("townsvc.tavern.prompt", "要買幾份糧食？（每份 %d 金，最多 %d 份）"),
			unit, game.MaxRations),
		max: game.MaxRations,
		apply: func(n int) {
			rations := int(a.save.Rations)
			res := game.BuyRations(t.visit.Economy, a.gold(), rations, n)
			if !res.OK {
				t.message = res.Reason
				return
			}
			a.setGold(res.Gold)
			a.save.Rations = byte(rations + n)
			t.message = fmt.Sprintf(a.tr.UI("townsvc.tavern.done", "買了 %d 份糧食，付 %d 金（共 %d 份，剩 %d 金）"),
				n, res.Cost, a.save.Rations, res.Gold)
		},
	}
}

// --- 神殿 ---

func (a *app) openDonateInput(t *townScreen) {
	c := a.currentMember(t)
	if c == nil {
		return
	}
	t.amount = &amountInput{
		prompt: fmt.Sprintf(a.tr.UI("townsvc.temple.donate_prompt", "%s 要捐多少金幣？（1 金換 1 點經驗，身上 %d 金）"),
			c.Name, a.gold()),
		max: a.gold(),
		apply: func(n int) {
			res := game.Donate(c, a.gold(), n)
			if !res.OK {
				t.message = res.Reason
				return
			}
			a.setGold(res.Gold)
			t.message = fmt.Sprintf(a.tr.UI("townsvc.temple.donate_done", "%s 捐了 %d 金，經驗值來到 %d"),
				c.Name, n, c.Experience)
		},
	}
}

func (a *app) prayCurrent(t *townScreen) {
	c := a.currentMember(t)
	if c == nil {
		return
	}
	res := game.PrayAtTemple(c, a.gold(), t.visit.Town.Facilities.Church)
	if !res.OK {
		if res.Cost > 0 {
			t.message = fmt.Sprintf(a.tr.UI("townsvc.temple.pray_cost_note", "%s（要 %d 金）"), res.Reason, res.Cost)
			return
		}
		t.message = res.Reason
		return
	}
	a.setGold(res.Gold)
	t.message = fmt.Sprintf(a.tr.UI("townsvc.temple.pray_done", "%s 祈禱完畢，付 %d 金，呼喚成功率回到 %d%%"),
		c.Name, res.Cost, c.PrayChance)
}

// deityName 回傳神祇的顯示名稱。0 代表沒有信仰。
//
// 神祇名是 FILES.DTT 的 `[153:164]`，索引是編號減一（見 docs/re/27 §4）。
func (a *app) deityName(id int) string {
	if id == 0 {
		return a.tr.UI("townsvc.temple.no_deity", "（無）")
	}
	names := a.strings.DeityNames()
	if id < 1 || id > len(names) {
		return fmt.Sprintf(a.tr.UI("townsvc.temple.deity_fallback", "神祇 %d"), id)
	}
	return names[id-1]
}

// --- 公會 ---

func (a *app) levelUpCurrent(t *townScreen) {
	c := a.currentMember(t)
	if c == nil {
		return
	}
	ok, short := c.CanLevelUp()
	if !ok {
		if short == 0 {
			t.message = fmt.Sprintf(a.tr.UI("townsvc.guild.max_level", "%s 已經是最高等級"), c.Name)
			return
		}
		t.message = fmt.Sprintf(a.tr.UI("townsvc.guild.need_exp", "公會認為 %s 還需要 %d 點經驗才能升級"),
			c.Name, short)
		return
	}
	res, err := game.LevelUp(a.rng, a.tables, c)
	if err != nil {
		t.message = fmt.Sprintf(a.tr.UI("townsvc.guild.levelup_error", "升級失敗：%v"), err)
		return
	}
	msg := fmt.Sprintf(a.tr.UI("townsvc.guild.levelup_done", "%s 升到 %d 級　生命 +%d　法力 +%d"),
		c.Name, c.Level, res.HPGain, res.SPGain)
	if res.Skipped {
		msg += a.tr.UI("townsvc.guild.trait_capped", "　（屬性已達種族上限）")
	} else {
		var gains []string
		for i, g := range res.TraitGains {
			if g > 0 {
				gains = append(gains, fmt.Sprintf("%s+%d", a.traitName(i), g))
			}
		}
		if len(gains) > 0 {
			msg += "\n"+a.tr.UI("townsvc.guild.traitgains", "　　屬性成長：") + strings.Join(gains, "　")
		}
	}
	t.message = msg
}

// --- 市集出售 ---

func (a *app) updateSellPicker(t *townScreen) {
	s := t.sell
	if s.slot < 0 {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			t.sell = nil
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			s.member = (s.member + 1) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			s.member = (s.member - 1 + len(a.members)) % len(a.members)
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			s.slot = 0
		}
		return
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		s.slot = -1
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		s.slot = (s.slot + 1) % game.InventorySlots
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		s.slot = (s.slot - 1 + game.InventorySlots) % game.InventorySlots
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.sellCurrent(t)
	}
}

func (a *app) sellCurrent(t *townScreen) {
	s := t.sell
	it := a.members[s.member].Inventory[s.slot]
	if it.Empty() {
		t.message = a.tr.UI("townsvc.sell.slot_empty", "那一格是空的")
		return
	}
	item, err := a.items.ByIndex(int(it.Type))
	if err != nil {
		t.message = a.tr.UI("townsvc.sell.unknown_item", "認不出這件道具")
		return
	}
	// 售價是基礎價的一半，**不套城鎮係數、不受議價影響**（docs/re/19 §7.4）。
	price := t.visit.Economy.SellPrice(item.Price)
	res := game.Sell(a.members, a.gold(), s.member, s.slot, price)
	if !res.OK {
		t.message = res.Reason
		return
	}
	a.setGold(res.Gold)
	t.message = fmt.Sprintf(a.tr.UI("townsvc.sell.done", "%s 賣掉%s，得 %d 金（共 %d）"),
		a.members[s.member].Name,
		a.tr.Event(itemSourceFile, int(it.Type), item.Name), price, res.Gold)
	t.sell = nil
}

func (a *app) drawSellPicker(t *townScreen, line func(string)) {
	s := t.sell
	if s.slot < 0 {
		line(a.tr.UI("townsvc.sell.who", "誰要賣東西？"))
		line("")
		for i, m := range a.members {
			line(fmt.Sprintf("%s%s", memberMark(s.member, i),
				textlayout.PadCells(m.Name, 10)))
		}
		line("")
		line(a.tr.UI("townsvc.sell.keys_confirm", "↑↓：選擇　Enter：確定　Esc：取消"))
		return
	}

	m := a.members[s.member]
	line(fmt.Sprintf(a.tr.UI("townsvc.sell.slot_prompt", "%s 要賣哪一件？（售價是底價的一半）"), m.Name))
	line("")
	for i := 0; i < game.InventorySlots; i++ {
		it := m.Inventory[i]
		name, note := a.tr.UI("townsvc.sell.item_empty", "（空）"), ""
		if !it.Empty() {
			name = a.itemLabel(it)
			if item, err := a.items.ByIndex(int(it.Type)); err == nil {
				note = fmt.Sprintf(a.tr.UI("townsvc.sell.price_note", "%d 金"), t.visit.Economy.SellPrice(item.Price))
			}
			switch i {
			case m.EquippedWeapon:
				note = a.tr.UI("townsvc.sell.weapon_locked", "（武器，不能賣）")
			case m.EquippedArmor:
				note = a.tr.UI("townsvc.sell.armor_locked", "（護甲，不能賣）")
			}
		}
		line(fmt.Sprintf("%s%s%s", memberMark(s.slot, i),
			textlayout.PadCells(name, 14), note))
	}
	line("")
	line(a.tr.UI("townsvc.sell.keys_sell", "↑↓：選擇　Enter：賣出　Esc：返回"))
}

// traitName 依索引取屬性名（表在 createui.go）。
func (a *app) traitName(i int) string { return a.label(traitNames, i) }

// --- 碼頭 ---

// buyShip 買一艘船，停到隊伍旁邊的海面上。
//
// 原版是先問「Buy ?」再放船，放不下就退回；這裡把兩步合成一次按鍵，
// 因為放不下的時候本來就不扣錢。
func (a *app) buyShip(t *townScreen) {
	if !t.visit.HasDocks() {
		t.message = a.tr.UI("townsvc.dock.no_ship", "這座城鎮沒有船可買")
		return
	}
	price := t.visit.Economy.ShipPrice()
	res := game.BuyShip(&a.save.Ships, a.tileAt,
		a.party.X(), a.party.Y(), a.mapID, a.gold(), price)
	if !res.OK {
		t.message = fmt.Sprintf(a.tr.UI("townsvc.dock.buy_cost_note", "%s（船價 %d 金）"), res.Reason, price)
		return
	}
	a.setGold(res.Gold)
	t.message = fmt.Sprintf(a.tr.UI("townsvc.dock.buy_done", "船與船員都備妥了，付 %d 金（剩 %d）"), res.Cost, res.Gold)
}

// repairShip 修好腳邊的船。
func (a *app) repairShip(t *townScreen) {
	res := game.RepairShip(t.visit.Economy, &a.save.Ships,
		a.party.X(), a.party.Y(), a.gold())
	if !res.OK {
		if res.Cost > 0 {
			t.message = fmt.Sprintf(a.tr.UI("townsvc.dock.repair_cost_note", "%s（修好要 %d 金）"), res.Reason, res.Cost)
			return
		}
		t.message = res.Reason
		return
	}
	a.setGold(res.Gold)
	t.message = fmt.Sprintf(a.tr.UI("townsvc.dock.repair_done", "修好了，付 %d 金（剩 %d）"), res.Cost, res.Gold)
}

// tileAt 回傳地圖座標的地形值，超出範圍回 0。
//
// 供買船的水位判定用 —— 走的是 a.tiles（檔案解出來的原樣），不是
// a.drawTiles（摻過浪花的顯示用副本）。兩個海面 tile 放船都認，
// 所以其實摻不摻都一樣，但規則判定一律走 tiles 是本專案的既有分工。
func (a *app) tileAt(x, y int) byte {
	t, err := a.tiles.TileAt(x, y)
	if err != nil {
		return 0
	}
	return t
}

// --- 學院 ---
//
// 一座城鎮最多有三間學院（`TOWN*.DAT` 的三個槽），每間只教一種技能。
// 兩層游標：上下選學院、Tab 換要學的人、L 學下去。

func (a *app) updateCollege(t *townScreen) {
	colleges := t.visit.Town.Facilities.Colleges
	if len(colleges) == 0 {
		return
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		t.college = (t.college + 1) % len(colleges)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		t.college = (t.college - 1 + len(colleges)) % len(colleges)
	case inpututil.IsKeyJustPressed(ebiten.KeyTab):
		if n := len(a.members); n > 0 {
			t.member = (t.member + 1) % n
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyL):
		a.learnCurrent(t)
	}
}

func (a *app) learnCurrent(t *townScreen) {
	colleges := t.visit.Town.Facilities.Colleges
	c := a.currentMember(t)
	if c == nil || t.college < 0 || t.college >= len(colleges) {
		return
	}
	skill := gamedata.SkillID(colleges[t.college])
	res, err := game.LearnSkill(a.tables, c, a.gold(), skill)
	if err != nil {
		t.message = fmt.Sprintf(a.tr.UI("townsvc.college.learn_error", "學院出錯：%v"), err)
		return
	}
	if !res.OK {
		t.message = res.Reason
		if res.Cost > 0 {
			t.message += fmt.Sprintf(a.tr.UI("townsvc.college.cost_note", "（學費 %d 金）"), res.Cost)
		}
		return
	}
	a.setGold(res.Gold)
	t.message = fmt.Sprintf(a.tr.UI("townsvc.college.learn_done", "%s 學會了%s，付 %d 金（剩 %d）"),
		c.Name, a.skillName(skill), res.Cost, res.Gold)
}

// skillSourceFile 是技能名稱翻譯目錄的 key，與 dwstrings 產生時一致。
const skillSourceFile = "SKILLS"

// skillName 回傳技能的顯示名稱（已翻譯）。原文來自 FILES.DTT 的技能名段。
func (a *app) skillName(s gamedata.SkillID) string {
	// 種族天生能力（偽 id 31–34）的名字不在 FILES.DTT 的技能表裡，
	// 另外走 RACESKILL 目錄（見 gamedata.RaceSkillName）。
	if en := gamedata.RaceSkillName(s); en != "" {
		return a.tr.Event(raceSkillSourceFile, int(s)-int(gamedata.SkillRegeneration), en)
	}
	names := a.strings.SkillNames()
	if int(s) < 0 || int(s) >= len(names) {
		return fmt.Sprintf(a.tr.UI("townsvc.skill.fallback", "技能 %d"), s)
	}
	return a.tr.Event(skillSourceFile, int(s), names[s])
}

// raceSkillSourceFile 是種族天生能力譯名目錄的 key。
const raceSkillSourceFile = "RACESKILL"

func (a *app) drawCollege(t *townScreen, line func(string)) {
	colleges := t.visit.Town.Facilities.Colleges
	c := a.currentMember(t)
	if len(colleges) == 0 || c == nil {
		line(a.tr.UI("townsvc.college.none", "這座城鎮沒有學院"))
		return
	}

	remaining, err := c.RemainingSkillPoints(a.tables)
	if err != nil {
		line(fmt.Sprintf(a.tr.UI("townsvc.college.points_error", "算不出剩餘點數：%v"), err))
		return
	}
	line(fmt.Sprintf(a.tr.UI("townsvc.college.header", "學生：%s（%s）　剩餘智力點數 %d"),
		c.Name, a.label(className, int(c.Class)), remaining))
	line("")

	for i, id := range colleges {
		skill := gamedata.SkillID(id)
		points, err := a.tables.SkillCost(skill, c.Class)
		if err != nil {
			continue
		}
		note := fmt.Sprintf(a.tr.UI("townsvc.college.cost_points", "%d 點　%d 金"), points, game.CollegeGoldCost(points))
		if c.HasSkill(skill) {
			note = a.tr.UI("townsvc.college.learned", "已學會")
		}
		line(fmt.Sprintf("%s%s%s", memberMark(t.college, i),
			textlayout.PadCells(a.skillName(skill), 14), note))
	}
	line("")
	line(a.tr.UI("townsvc.college.keys", "↑↓：選學院　Tab：換學生　L：學下去"))
}

// convertCurrent 讓游標指的隊員改信這座神殿的神。
//
// 改宗不收金幣，收的是智力點數 —— 訊息要講清楚，不然玩家會以為免費。
func (a *app) convertCurrent(t *townScreen) {
	c := a.currentMember(t)
	if c == nil {
		return
	}
	deity := t.visit.Town.Facilities.Church
	res, err := game.ConvertAtTemple(a.tables, c, a.gold(), deity)
	if err != nil {
		t.message = fmt.Sprintf(a.tr.UI("townsvc.temple.convert_error", "改宗出錯：%v"), err)
		return
	}
	if !res.OK {
		t.message = res.Reason
		return
	}
	t.message = fmt.Sprintf(a.tr.UI("townsvc.temple.convert_done", "%s 改信 %s，學會了%s（花 %d 點智力）"),
		c.Name, a.deityName(deity), a.skillName(game.DeityOrder(deity)), res.Cost)
}
