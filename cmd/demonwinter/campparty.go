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

// 紮營選單的 Party（`1000:33e8`，見 `docs/re/40`）。
//
// 原版的字串是 `Inspect character:` —— 這一項就是**攤開一個人的角色卡**，
// 不是隊伍總覽。世界地圖上的 `P`（隊伍名冊）是本作自己加的摘要，兩者並存。

// partyScreen 是角色卡的狀態。
type partyScreen struct {
	// member 是選人游標；−1 代表正在看某個人的卡。
	member int
	// showing 是正在看誰。
	showing int
}

func (a *app) openPartySheet() {
	if len(a.members) == 0 {
		a.camp.message = "隊伍是空的"
		return
	}
	a.camp.message = ""
	a.camp.party = &partyScreen{member: 0, showing: -1}
}

// updatePartySheet 推進角色卡的輸入，回報這一畫面該不該關掉。
//
// **不直接動 `a.camp`** —— 紮營的 Party 與商隊的 Inspect 是同一張卡
// （原版 `278d:2f61`，兩邊都呼叫它），呼叫端各自持有自己的 `partyScreen`。
func (a *app) updatePartySheet(p *partyScreen) (closed bool) {
	if p.showing >= 0 {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape),
			inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			p.showing = -1
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			p.showing = (p.showing + 1) % len(a.members)
			p.member = p.showing
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			p.showing = (p.showing - 1 + len(a.members)) % len(a.members)
			p.member = p.showing
		}
		return false
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		return true
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		p.member = (p.member + 1) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		p.member = (p.member - 1 + len(a.members)) % len(a.members)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		p.showing = p.member
	}
	return false
}

func (a *app) drawPartySheet(p *partyScreen, line func(string)) {
	if p.showing < 0 {
		line("要看誰？")
		line("")
		a.drawMemberList(line, p.member, nil)
		line("")
		line("↑↓：選擇　Enter：攤開　Esc：取消")
		return
	}

	c := a.members[p.showing]
	line(fmt.Sprintf("%s　%s　%s　%d 級",
		c.Name, nameOf(raceName, int(c.Race)), nameOf(className, int(c.Class)), c.Level))
	line("")

	// 五項屬性排成一行 —— 逐項換行會把技能清單擠出畫面。
	var traits []string
	for i := 0; i < gamedata.NumTraits; i++ {
		traits = append(traits, fmt.Sprintf("%s %2d", traitName(i), c.Traits[i]))
	}
	line("　" + strings.Join(traits, "　"))

	line(fmt.Sprintf("　生命 %3d/%3d　法力 %3d/%3d",
		c.CurrentHP, c.MaxHP, c.CurrentSP, c.MaxSP))
	line(fmt.Sprintf("　武器 %s　護甲 %d", a.weaponLabel(c), c.ArmorRating()))

	exp := "已達上限"
	if next := game.ExpForNextLevel(c.Level); next > 0 {
		exp = fmt.Sprintf("%d / %d", c.Experience, next)
	}
	line(fmt.Sprintf("　經驗 %s", exp))

	faith := "無信仰"
	if c.Deity > 0 {
		faith = fmt.Sprintf("%s　祈禱 %d%%", a.deityName(c.Deity), c.PrayChance)
	}
	line("　信仰 " + faith)
	if c.BindLevel > 0 {
		line(fmt.Sprintf("　束縛 %d 級", c.BindLevel))
	}

	line("")
	line("　技能")
	if names := a.learnedSkillNames(c); len(names) == 0 {
		line("　　（無）")
	} else {
		for _, row := range chunkStrings(names, 3) {
			line("　　" + strings.Join(row, "　"))
		}
	}

	line("")
	line("↑↓：換人　Enter／Esc：收起")
}

// learnedSkillNames 列出這名角色學過的技能名稱（已翻譯）。
func (a *app) learnedSkillNames(c game.Character) []string {
	var out []string
	for i := 0; i < gamedata.NumSkills; i++ {
		if c.Skills[i] {
			out = append(out, textlayout.PadCells(a.skillName(gamedata.SkillID(i)), 8))
		}
	}
	return out
}

// chunkStrings 把清單切成每列 n 個。
func chunkStrings(s []string, n int) [][]string {
	var out [][]string
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
