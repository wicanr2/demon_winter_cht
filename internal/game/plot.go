package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"

// 睡覺時的劇情推進（原版 `1000:0206`，見 `docs/re/80`）。
//
// **冬之魔是在玩家睡覺的時候降臨的。** 每睡一晚只推進一段，
// 而且順序固定 —— 原版每一段都以 `return` 結束，不會一晚跑兩段。

// PlotDream 是這一晚要播的夢，`NoDream` 表示沒有。
type PlotDream int

const (
	// NoDream 這一晚什麼都沒發生。
	NoDream PlotDream = -1
	// DreamWarning 是第一場夢：馬利馮的警告（`T.TXT` 第 0 頁）。
	DreamWarning PlotDream = 0
	// DreamArrival 是第二場夢：冬之魔降臨（第 1 頁）。
	DreamArrival PlotDream = 1
	// DreamGodsDead 是第三場夢：諸神已死（第 2 頁）。
	// 這一晚同時把神殿變成廢墟、把全隊的信仰清空。
	DreamGodsDead PlotDream = 2
)

// 劇情階段（隊伍欄位 `+0xb9`）。
const (
	// PlotBeforeArrival 是冬之魔還沒降臨。**兩份原版存檔的起始值都是 0。**
	//
	// 把 0 推到 1 的那道寫入**找到了**（2026-07-27，`docs/re/101` §3）：
	// 地點劇情 case 8 ——**拿到恆世寶珠**。原版是靠「送道具」常式的
	// 旗標寫入 `party[0xb3 + param] = 1`，param 6 剛好落在 `+0xb9`，
	// 也就是這個欄位。`docs/re/81` 從馬利馮預言的第一行
	// `The Orb of Evertime now is yours` 推測寫入端在寶珠事件裡，推測正確。
	//
	// 所以引擎不把寶珠當第 7 格旗標，而是**直接推進這個階段**
	// （`TakeOrbOfEvertime`）—— 同一個 byte 不要有兩個名字。
	PlotBeforeArrival byte = 0
	// PlotArrivalDue 是「下次睡覺就降臨」。
	PlotArrivalDue byte = 1
	// PlotArrived 是已降臨，世界開始惡化。
	PlotArrived byte = 2
)

// firstDreamMonth 是第一場夢的月份門檻：**月份要大於 3**
//（原版 `cmpb $0x3, +0x9d / jbe`）。月從 0 起算，所以最快是第五個月。
const firstDreamMonth = 3

// TempleRuinsValue 是神殿全毀時寫進 `+0xba` 的值。
const TempleRuinsValue = 0xff

// PlotState 是劇情推進要讀寫的隊伍欄位。
//
// 用一個小結構而不是直接吃存檔，是為了讓規則層不必認識存檔格式 ——
// 這一層只管「條件成立時該改什麼」。
type PlotState struct {
	Month      byte
	Stage      byte
	FirstDream byte
	TempleRuin byte
}

// AdvancePlotOnSleep 依原版的順序推進一段劇情，回傳該播哪一場夢。
//
// 呼叫端要在「這一晚沒有被打斷」之後才呼叫（原版 `1000:026d` 那條
// 回傳 −1 的路徑會直接離開紮營選單，不推進劇情）。
//
// 第三場夢會連帶要求呼叫端清掉全隊的薩滿技能、司祭技能與信奉的神祇 ——
// 那是角色層級的欄位，不在 PlotState 裡，回傳 `WipeFaith` 通知。
func AdvancePlotOnSleep(p *PlotState) (dream PlotDream, wipeFaith bool) {
	switch {
	case p.Month > firstDreamMonth && p.FirstDream == 0:
		p.FirstDream = 1
		return DreamWarning, false

	case p.Stage == PlotArrivalDue:
		p.Stage = PlotArrived
		return DreamArrival, false

	case p.Stage == PlotArrived && p.TempleRuin == 0:
		p.TempleRuin = TempleRuinsValue
		return DreamGodsDead, true
	}
	return NoDream, false
}

// WipeFaith 清掉一名角色的薩滿技能、司祭技能與信奉的神祇。
//
// 這是**永久剝奪**：那兩個技能旗標在整支原版執行檔裡只有這一處寫入
//（`docs/re/79` §3），沒有任何路徑能還原。驅散不死、祈禱、驅邪從此失效。
//
// 移植時不能「順手修掉」—— 那會改變遊戲的難度曲線與敘事。
func WipeFaith(c *Character) {
	c.Skills[gamedata.SkillShaman] = false
	c.Skills[gamedata.SkillPriesthood] = false
	c.Deity = 0
}
