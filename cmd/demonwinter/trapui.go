package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// 地城陷阱的操作層：`L` 查看陷阱 ＋ 陷阱本體。
//
// 規則在 `internal/game/traps.go`（原版動作 `0x07` ＝ `222f:1882`
// 與 `25be:0263` 那張九格分派表）。這裡只做輸入、訊息與存檔寫回。
//
// **這一整塊在接上之前是「解完沒接」**：七種陷阱的擲點、25%／100% 的
// 偵測率、Noticed／Disarmed 兩種改寫，`docs/re/68`／`91` 全部讀出來了，
// 而引擎只印一行「有陷阱！」。連帶讓察覺陷阱（技能 10）與
// 解除陷阱（技能 11）兩個技能一直沒有效果 —— 而學院會收錢教它們。

// springTrap 觸發一格陷阱，回傳是否要中止這一步剩下的處理。
//
// 中止的唯一情況是全隊死亡。原版印完陷阱訊息之後**還是會走文字路徑**
// （`0x19ace` 之後回到共用出口），所以其餘情況都回 false。
func (a *app) springTrap(hit *scenario.SpecialHit) bool {
	noticed := hit.Tile.Class() == scenario.SpecialClassTrapAlt
	c := game.TrapCaseFor(hit.Tile.Value())
	res := game.SpringTrap(a.rng, c, noticed, a.members)

	a.message = strings.Join(a.trapLines(c, noticed, res), "\n")
	a.trace.note("陷阱 %s", game.TrapNames[c])

	if res.AlarmCountdown > 0 {
		// 警報不扣血 —— 它把遭遇倒數設成 1–5 步（`0x19eac`）。
		a.save.EncounterCountdown = byte(res.AlarmCountdown)
	}
	if res.Wiped {
		return a.checkPartyDeath()
	}
	return false
}

// trapLines 把結果組成畫面上的幾行。
//
// 原文字串留在 `game.TrapNames` 與這裡的 fallback，翻譯走 `a.tr.UI`
// 的語意化 key（與死亡畫面同一個做法）。
func (a *app) trapLines(c game.TrapCase, noticed bool, res game.TrapResult) []string {
	if res.Avoided {
		return []string{a.tr.UI("trap.avoided", "小心翼翼地避開了陷阱")}
	}

	var out []string
	if noticed {
		out = append(out,
			a.tr.UI("trap.careful", "即使再小心"),
			a.tr.UI("trap.triggered", "還是觸動了陷阱"))
	}
	out = append(out, a.tr.UI(trapNameKey(c), trapNameZH[c]))

	for _, h := range res.Hits {
		out = append(out, a.trapHitLine(c, h))
	}
	if res.AlarmCountdown > 0 {
		out = append(out, fmt.Sprintf(
			a.tr.UI("trap.alarm.steps", "%d 步之內必定遇敵"), res.AlarmCountdown))
	}
	return out
}

// trapHitLine 是一下的敘述。四種陷阱落空時說法各不相同，照原版分開。
func (a *app) trapHitLine(c game.TrapCase, h game.TrapHit) string {
	name := "隊員"
	if h.Member >= 0 && h.Member < len(a.members) {
		name = a.members[h.Member].Name
	}

	if h.Missed {
		switch c {
		case game.TrapBungeiPit, game.TrapPoisonPit:
			// 原版 `: safe`
			return name + a.tr.UI("trap.pit.safe", "：沒事")
		case game.TrapPool, game.TrapAcidPool:
			// 原版 `swims out.`
			return name + a.tr.UI("trap.pool.escape", "游了出來")
		case game.TrapSpears:
			return a.tr.UI("trap.spear.miss", "長矛落空")
		case game.TrapDarts:
			return a.tr.UI("trap.dart.miss", "飛鏢落空")
		}
		return a.tr.UI("trap.miss", "落空")
	}

	line := fmt.Sprintf("%s%s%d", name,
		a.tr.UI("trap.damage", " 受到傷害 "), h.Damage)
	if h.Died {
		line += a.tr.UI("trap.died", "，倒下了")
	}
	return line
}

// trapNameKey 是第 c 格的翻譯 key。
func trapNameKey(c game.TrapCase) string {
	return fmt.Sprintf("trap.name%d", int(c))
}

// trapNameZH 是九格的中文宣告，索引即 case 編號。
//
// `Bungei` 不譯 —— 出處在攻略與譯名表裡都查不到，
// **不憑音譯造一個專有名詞**（`docs/re/68` §5）。
var trapNameZH = [9]string{
	"有什麼東西你看不見",
	"毒針射中了",
	"一個 Bungei 陷坑！",
	"毒坑！",
	"長矛陷阱！",
	"飛鏢從牆上的孔洞射出",
	"一池水！",
	"一池酸液！",
	"觸動了警報！",
}

// lookForTraps 是 `L` 指令（手冊「地底 → 偵測與解除陷阱」）。
//
// 掃描距離是**光源強度**，所以在戶外（沒有火把時 torch 仍有值）也掃得動；
// 原版沒有把它限制在地城，這裡照做。
func (a *app) lookForTraps() {
	st := a.special[a.mapID]
	scan := game.LookForTraps(a.rng, a.members, st, a.world.Tiles(),
		a.party.X(), a.party.Y(), a.party.Facing(), int(a.torch))

	a.trapSpots = scan.Spots
	if len(scan.Spots) == 0 {
		a.message = a.tr.UI("trap.none", "沒有發現陷阱")
		a.trace.note("查看陷阱：沒有")
		return
	}

	verb := a.tr.UI("trap.noticed", "已注意")
	if scan.HasDisarm {
		verb = a.tr.UI("trap.disarmed", "已解除")
	}
	a.message = fmt.Sprintf("%s　%s %d %s",
		verb, a.tr.UI("trap.found.prefix", "找到"), len(scan.Spots),
		a.tr.UI("trap.found.suffix", "個陷阱"))
	a.trace.note("查看陷阱：%d 個（%s）", len(scan.Spots), verb)
}
