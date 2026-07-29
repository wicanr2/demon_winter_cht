package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 開新遊戲的起始狀態（`docs/re/87`）。
//
// **原版出貨的 `PARTY.DAT` 不是新遊戲，是玩過的存檔。**
// 九個欄位與建角程式寫死的起始值不同（糧食 8 vs 20、日 17 vs 8、
// 時 13 vs 5、位置 (9,32) 圖 1 vs (28,50) 圖 34、金幣 65 vs 75……），
// 而且 Menhir 的血是 15/27。
//
// 這件事很重要，因為之前所有試玩都是從那份存檔開始的 ——
// **等於從遊戲中段、地城深處開始玩**。真正的遊戲是從**世界地圖**開始的。
//
// 來源是建角程式（`Character Utilities`，`0x14567` ＝ `206a:02c7`）
// 裡的初始化段，`0x148c8`–`0x1497f` 逐行抄下來。

// 新遊戲的起始值。全部來自 `0x148c8`–`0x1497f` 的立即數。
const (
	// NewGameRations 是起始糧食（`0x148cc`：`+0x9b = 0x14`）。
	NewGameRations = 20
	// NewGameDay 是起始日（`0x14908`：`+0x9e = 8`）。月不寫，留 0（紅玉月）。
	NewGameDay = 8
	// NewGameHour 是起始時辰（`0x14912`：`+0x9f = 5`）。
	NewGameHour = 5
	// NewGameTimeCounter 是起始步數計數（`0x1491c`：`+0xa0 = 1`）。
	NewGameTimeCounter = 1
	// NewGameX／NewGameY 是起始座標（`0x148d6`／`0x148e0`）。
	NewGameX = 28
	NewGameY = 50
	// NewGameMapID 是起始地圖（`0x148ea`：`+0xa3 = 0x22` ＝ 34）。
	//
	// **34 是世界地圖的一段，不是地城。** 出貨存檔的 1 是地城 ——
	// 這一個 byte 就是「從中段開始」與「從頭開始」的差別。
	NewGameMapID = 34
	// NewGameFacing 是起始朝向（`0x148f4`：`+0xa4 = 1`）。
	NewGameFacing = 1
	// NewGameLight 是起始光源（`0x148fe`：`+0xa7 = 1`）。
	NewGameLight = 1
	// NewGameGold 是起始金幣（`0x14942`：`+0x0a = 0x4b`，高位字寫 0）。
	NewGameGold = 75
	// NewGameMerchantBase 是商隊規模基準（`0x1494c`：`+0xaf = 1`）。
	NewGameMerchantBase = 1

	// NewGameEncounterMin／Max 是起始遭遇倒數。
	// 原版是 `Roll(5) + 0xe`（`0x14922`–`0x14933`）；Roll 值域 1–5，
	// 所以是 15–19，不是把 raw RNG 當成 0-based 得出的 14–18。
	NewGameEncounterMin = 15
	NewGameEncounterMax = 19
)

// 起始那艘船（`0x14956`–`0x1497a` 寫的是船隻陣列第 9 格）。
//
// **船體值是 67 不是滿值 75**（`0x14971`：`+0x5b = 0x43`）——
// 一開始就有一艘半舊的船。移植時不要「順手補滿」，那會改變玩家
// 第一次要不要花錢修船的決定。
const (
	NewGameShipSlot  = 9
	NewGameShipX     = 13
	NewGameShipY     = 51
	NewGameShipMapID = 1
	NewGameShipHull  = 67
)

// newGameFormationEmpty 是陣型格的空值。
//
// 原版把 trailer `+0x00`–`+0x08` 九格全部寫 `0xff`（`0x14984` 的迴圈，
// 上限 9）。`0xff` 是「這一格沒人」——**不是 0**。
const newGameFormationEmpty = 0xff

// ApplyNewGame 把存檔改成「剛開新遊戲」的狀態。
//
// 角色本身不動 —— 建角是另一件事（玩家用 `F1` 逐個換掉五個槽位）。
// 這裡只設隊伍共用的那些欄位。
//
// encounterCountdown 由呼叫端給（原版是 `Roll(5) + 14`），
// 這樣才擲得出可重現的值；超出 15–19 會被夾回範圍。
func ApplyNewGame(s *scenario.SaveGame, encounterCountdown int) {
	for i := range s.Formation {
		s.Formation[i] = newGameFormationEmpty
	}
	s.Gold = NewGameGold
	s.Rations = NewGameRations
	s.Day = NewGameDay
	s.Hour = NewGameHour
	s.TimeCounter = NewGameTimeCounter
	s.PositionX = NewGameX
	s.PositionY = NewGameY
	s.MapID = NewGameMapID
	s.Facing = NewGameFacing
	s.LightSource = NewGameLight
	// **治療水池的額度刻意不設。** 原版的新遊戲初始化沒有寫 `+0xaa`
	// （全檔只有三處碰它：水池的比較與遞減、以及睡覺補回 7，
	// `docs/re/90` §2），所以剛開局是 0 —— 第一次睡覺之前喝不到水池。
	// 不要「順手補滿」，那會改變玩家第一晚要不要紮營的決定。
	s.MerchantBase = NewGameMerchantBase

	if encounterCountdown < NewGameEncounterMin {
		encounterCountdown = NewGameEncounterMin
	}
	if encounterCountdown > NewGameEncounterMax {
		encounterCountdown = NewGameEncounterMax
	}
	s.EncounterCountdown = byte(encounterCountdown)

	// **隊伍人數歸零。** 原版的建角程式每造好一個角色就 `inc trailer[+0x9a]`
	// （`0x15109`），所以一支還沒建角的隊伍是 0 人。
	// 出貨存檔是 5，不歸零的話「建到一半就存檔」會寫出「5 人」但其中
	// 幾個還是別人的角色 —— 而讀回來完全看不出來。
	s.PartySize = 0

	// 船隻陣列先全部清空，再放那一艘 —— 出貨存檔裡有兩艘船，
	// 只覆蓋第 9 格的話會把別人玩剩的船帶進新遊戲。
	for i := range s.Ships {
		s.Ships[i] = scenario.Ship{}
	}
	s.Ships[NewGameShipSlot] = scenario.Ship{
		X: NewGameShipX, Y: NewGameShipY, MapID: NewGameShipMapID,
		Hull: NewGameShipHull, Unknown4: 2,
	}
}
