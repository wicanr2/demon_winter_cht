package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 戰鬥中的 `?` 檢視（原版動作 case 10 → `17c5:1056`，規則在 `internal/game/examine.go`）。
//
// **接上之前這一格是空的**：`game.ActionExamine` 只有定義，零呼叫端、沒綁鍵，
// 而 `battleui.go` 的註解還寫著「熱鍵沿用原版（含 `?` 檢視）」。
// 連帶讓戰術（技能 7）與怪物學識（技能 25）兩個技能完全沒有效果。
//
// 走訪用 **C 繼續／B 返回／Q 離開** —— 那是原版 DOS 版真正的選單
// （字串在 `ds:0x0adb`／`0x0ae4`／`0x0ae9`，與市集同一組）。
// 手冊寫的 `←`／`→` 是 Apple II 版的說法；這裡兩種都收，
// 方向鍵是給現代玩家的方便，不影響原版鍵位可用。

// examineView 是檢視面板的狀態。
type examineView struct {
	// order 是可看的槽位，由小到大。
	order []int
	// at 是 order 裡的位置。
	at int
}

// openExamine 打開檢視面板。**不花移動點**（原版成本 0）。
func (a *app) openExamine() {
	if a.battle == nil {
		return
	}
	order := game.ExamineOrder(a.battle)
	if len(order) == 0 {
		return
	}
	a.examine = &examineView{order: order}
	a.trace.note("檢視：%d 個單位", len(order))
}

func (a *app) updateExamine() error {
	v := a.examine
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		inpututil.IsKeyJustPressed(ebiten.KeyQ):
		a.examine = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyC),
		inpututil.IsKeyJustPressed(ebiten.KeyRight):
		v.at = (v.at + 1) % len(v.order)
	case inpututil.IsKeyJustPressed(ebiten.KeyB),
		inpututil.IsKeyJustPressed(ebiten.KeyLeft):
		v.at = (v.at - 1 + len(v.order)) % len(v.order)
	}
	return nil
}

// examineSlot 是游標目前指的槽位。戰場繪製拿它畫反白框。
func (v *examineView) slot() int { return v.order[v.at] }

func (a *app) drawExamine(line func(string)) {
	card := game.ExamineUnit(a.battle, a.examine.slot(), a.members)

	line(card.Name)
	if s := a.examineStatusLine(card); s != "" {
		line(s)
	}
	line("")

	if card.Stats {
		line(fmt.Sprintf("%s：%3d", a.tr.UI("examine.strength"), card.Strength))
		line(fmt.Sprintf("%s：%3d", a.tr.UI("examine.skill"), card.Skill))
		line(fmt.Sprintf("%s：%3d", a.tr.UI("examine.speed"), card.Speed))
		line(fmt.Sprintf("%s：%3d %s",
			a.tr.UI("examine.armor"), card.Armor,
			a.tr.UI("examine.armor.unit")))
		line(fmt.Sprintf("%s：%s",
			a.tr.UI("examine.weapon"), a.weaponName(card.WeaponIndex)))
	} else {
		// 原版在沒有怪物學識時整組屬性都不印。這裡多說一句為什麼 ——
		// 不然玩家只會看到一片空白，以為是壞掉了。
		line(a.tr.UI("examine.nolore"))
	}

	if card.ShowHPSP {
		line(fmt.Sprintf("%s：%3d　%s：%3d",
			a.tr.UI("examine.hp"), card.HP,
			a.tr.UI("examine.sp"), card.SP))
	}
	if card.TargetName != "" {
		line("")
		line(fmt.Sprintf(a.tr.UI("examine.target"), card.TargetName))
	}

	line("")
	line(fmt.Sprintf("%d / %d", a.examine.at+1, len(a.examine.order)))
	line(a.tr.UI("examine.keys"))
}

// examineStatusLine 是狀態那一行。束縛才帶等級（原版 `1 < 狀態 < 5`）。
func (a *app) examineStatusLine(card game.ExamineCard) string {
	if card.Status == game.StatusNormal {
		return ""
	}
	name := a.tr.UI(examineStatusKey(card.Status))
	if card.ShowBindLevel {
		return fmt.Sprintf("%s>%d", name, card.BindLevel)
	}
	return name
}

func examineStatusKey(s game.UnitStatus) string {
	return fmt.Sprintf("examine.status%d", int(s))
}

// examineStatusZH 是六種狀態的中文。索引就是原版的狀態值。
// ui:dynamic examine.status —— 由 examineStatusKey(s) 查表。
func examineStatusZH(s game.UnitStatus) string {
	switch s {
	case game.StatusPoison:
		return "中毒"
	case game.StatusBindLow, 3, game.StatusBindHigh:
		return "被束縛"
	case game.StatusDead:
		return "已死亡"
	}
	return ""
}

// weaponName 把武器索引換成名稱。
//
// **索引帶符號**：負數代表附毒（`Unit.WeaponIndex`），取絕對值才是表索引。
//
// 名表走 `StringPool.WeaponTypeNames()`（`FILES.DTT` 的 `[123:131)`），
// 而**翻譯走 `ITEMS.DAT` 那本目錄** —— 兩張表的前八筆是同一組名字，
// `gamedata` 的測試就在釘這件事（`strings_test.go` 逐項比對）。
// 沿用市集與紮營同一條翻譯路徑，不另外開一本武器目錄。
func (a *app) weaponName(idx int) string {
	poisoned := idx < 0
	if poisoned {
		idx = -idx
	}
	names := a.strings.WeaponTypeNames()
	name := a.tr.UI("examine.weapon.none")
	if idx > 0 && idx <= len(names) {
		// 武器類型 idx（1 起算）對應 ITEMS.DAT 索引 idx−1。
		name = a.tr.Event(itemSourceFile, idx-1, names[idx-1])
	}
	if poisoned {
		name += a.tr.UI("examine.weapon.poison")
	}
	return name
}
