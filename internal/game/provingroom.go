package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 十間試煉室（地點劇情 case 9，地圖 2，原版 `1990:1cf5` ＝ `0x0f1f5`，
// `docs/re/101` §2）
//
// 一個職業一間。走進去之後那間房子只認得那個職業的人 ——
// 原版把 3×3 陣型整張備份到 `+0x80`，清成空的，
// **再只把符合職業的人填回去**，然後開戰。所以那是一場一個人（或幾個人）
// 的單挑，隊伍其他人站在旁邊看。
//
// 十間全過之後，地圖 2 (42,28) 那一格才會給出恆世寶珠（case 8）。
//
// # 三張表都對得上
//
//	原版 `ds:0x0cae`／`0x0cc2` 的十組座標  ＝ `2SS.DAT` 類別 5 值 9 的十筆（10/10）
//	原版 `ds:0x17e1` 的十個職業名          ＝ `gamedata.Class` 的列舉順序（10/10）
//	原版 `ds:0x0cd6` 的十個參數           ＝ 怪物編號或「特殊分支」的負數標記

// ProvingTrial 是一間試煉室要玩家做什麼。
type ProvingTrial int

const (
	// ProvingFight：打一場（大多數職業）。
	ProvingFight ProvingTrial = iota
	// ProvingTaunt：盜賊那間 —— 只印一句「他要是踩了那個陷阱可就丟臉了」。
	ProvingTaunt
	// ProvingBlessing：靈視者那間 —— 「讓他在將來的日子裡證明自己」。
	ProvingBlessing
	// ProvingLore：學者那間 —— 給一段**符文密語提示**。
	ProvingLore
)

// provingRoom 是一間試煉室。
type provingRoom struct {
	X, Y  int
	Class gamedata.Class
	// Colour 是房間被光灌滿時的顏色（原版 `ds:0x0c86` 的十個遠指標）。
	Colour string
	Trial  ProvingTrial
	// Monsters 是要打的怪（Trial 不是 ProvingFight 時為空）。
	Monsters []int
}

// provingClericHorde 是牧師那間的七隻怪（`0x0f5c1`–`0x0f5ea`）。
//
// 原版寫法是「`+0x16 + i = 0x11` 跑五次，再把 `+0x1b`／`+0x1c` 寫成 0x16」，
// 也就是 **5 隻 Zombie ＋ 2 隻 Ghost**，`+0xa6`（隻數）＝ 7。
// 一個牧師對七隻不死生物 —— 這是全十間裡唯一的多打一。
var provingClericHorde = []int{17, 17, 17, 17, 17, 22, 22}

// ProvingRooms 是十間試煉室，索引 ＝ 職業 id ＝ 原版三張表的共同索引。
//
// 怪物那一欄的來源是 `ds:0x0cd6`（`[67, 11, 52, 95, −1, −2, 88, 77, −3, −4]`）：
// **正數是怪物編號，負數是「走特殊分支」的標記**，原版拿 `abs()` 之後
// 用 `1 <= v <= 4` 分流（`0x0f4ce` 呼叫的 `2000:06aa` 就是個 abs）。
// 所以那張表其實同時編碼了兩件事，靠正負號區分 —— 照抄這個結構。
//
// 怪物名是用驗過的 `MONSTER.DAT` 載入器查出來的，不是手算位移：
// **Monk 對上 Karate master、Cleric 對上僵屍與鬼魂** —— 這兩格就足以
// 證明索引沒有偏移。
var ProvingRooms = [scenario.ProvingRoomCount]provingRoom{
	{49, 12, gamedata.Ranger, "green", ProvingFight, []int{67}},    // Cave bear
	{35, 37, gamedata.Paladin, "silver", ProvingFight, []int{11}},   // Evil spirit
	{49, 5, gamedata.Barbarian, "brown", ProvingFight, []int{52}},   // Large dragon
	{56, 31, gamedata.Monk, "violet", ProvingFight, []int{95}},      // Karate master
	{35, 12, gamedata.Cleric, "white", ProvingFight, provingClericHorde},
	{42, 13, gamedata.Thief, "grey", ProvingTaunt, nil},
	{49, 37, gamedata.Wizard, "crimson", ProvingFight, []int{88}},   // Imp
	{42, 5, gamedata.Sorcerer, "black", ProvingFight, []int{77}},    // Stalker
	{28, 31, gamedata.Visionary, "blue", ProvingBlessing, nil},
	{35, 5, gamedata.Scholar, "beige", ProvingLore, nil},
}

// ProvingLoreHint 是學者那間給的符文密語（原版 `ds:0x0cea` 那個遠指標）。
//
// **定案不翻**（worklist D2）：符文是要玩家自己建對照表的解謎機制，
// 而這一句的答案要用英文輸入。點是原版的間隔符。
const ProvingLoreHint = "...USE....FACETED..MIRROR.IN..DARK....CHAPEL"

// ProvingRoomAt 回傳這個座標是第幾間試煉室，不是就回 −1。
//
// 原版是掃十格比對 X 與 Y（`0x0f213`–`0x0f254`），**最後一筆命中的算**
// （迴圈沒有 break）。十組座標互不相同，所以等價於「找到就回」。
func ProvingRoomAt(x, y int) int {
	idx := -1
	for i, r := range ProvingRooms {
		if r.X == x && r.Y == y {
			idx = i
		}
	}
	return idx
}

// ProvingEntry 是走進一間試煉室之後的結果。
type ProvingEntry int

const (
	// ProvingRunTrial：隊伍裡有能上場的人，照 Trial 進行。
	ProvingRunTrial ProvingEntry = iota
	// ProvingFreePass：隊伍裡**根本沒有**這個職業 → 「既然你沒有…我就讓你過。」
	ProvingFreePass
	// ProvingComeBackWhenWell：有這個職業但全都倒了 → 「等你的…好了再來。」
	// 這一條會**把隊伍趕出去**，而且不給過關旗標。
	ProvingComeBackWhenWell
)

// ProvingEjectTo 是「等你的人好了再來」那條路把隊伍丟到的座標
// （`0x0f405` 的 `+0xa1 = 0x2a`、`0x0f40f` 的 `+0xa2 = 0x13`）。
var ProvingEjectTo = struct{ X, Y int }{0x2a, 0x13} // (42,19)

// ProvingFighters 回傳這間試煉室能上場的成員索引，以及**有沒有倒下的同職業**。
//
// 原版的判定（`0x0f376`–`0x0f3bd`）：
//
//	職業 != 這間的職業        → 跳過，兩邊都不算
//	職業相同但戰鬥狀態 >= 2   → 算進 others（束縛與死亡）
//	職業相同且狀態 <= 1       → 能上場（正常與中毒都算）
//
// 門檻與矮人黑暗視覺那邊一樣是 `>= 2`（見 PartyHasDarkVision）——
// 兩處獨立比較，剛好同一個值，不要合併。
func ProvingFighters(idx int, members []Character) (fighters []int, downed int) {
	if idx < 0 || idx >= len(ProvingRooms) {
		return nil, 0
	}
	want := ProvingRooms[idx].Class
	for i, c := range members {
		if c.Class != want {
			continue
		}
		if c.Status > scenario.StatusPoison {
			downed++
			continue
		}
		fighters = append(fighters, i)
	}
	return fighters, downed
}

// EnterProvingRoom 判斷走進第 idx 間之後走哪一條路。
func EnterProvingRoom(idx int, members []Character) (ProvingEntry, []int) {
	fighters, downed := ProvingFighters(idx, members)
	switch {
	case len(fighters) > 0:
		return ProvingRunTrial, fighters
	case downed > 0:
		return ProvingComeBackWhenWell, nil
	default:
		return ProvingFreePass, nil
	}
}

// ProvingFormation 排出只有那幾個人上場的 3×3 陣型。
//
// 原版把整張表清成空的，再從**格 3 開始**逐一填
// （`0x0f396` 先 `matched++` 才 `party[+0x02 + matched] = i`，
// 所以第一個人落在格 3 而不是格 0）。格 3–5 是中間那一排 D E F。
func ProvingFormation(fighters []int) Formation {
	var f Formation
	f.Clear()
	for n, member := range fighters {
		cell := provingFirstCell + n
		if cell >= FormationCells {
			break
		}
		f[cell] = byte(member)
	}
	return f
}

// provingFirstCell 是第一個上場的人落在第幾格（原版 `+0x02 + 1`）。
const provingFirstCell = 3

// PassProvingRoom 把第 idx 間標記成過關。
func PassProvingRoom(s *scenario.SaveGame, idx int) {
	if s == nil || idx < 0 || idx >= len(s.ProvingPassed) {
		return
	}
	s.ProvingPassed[idx] = 1
}

// ProvingRoomsCleared 回報十間是不是全過了。
//
// 這就是地點劇情 case 8（恆世寶珠）的閘門：原版數「還有幾間沒過」，
// 不為 0 就什麼都不給（`0x1a288`–`0x1a2af`）。
func ProvingRoomsCleared(s *scenario.SaveGame) bool {
	if s == nil {
		return false
	}
	for _, v := range s.ProvingPassed {
		if v == 0 {
			return false
		}
	}
	return true
}
