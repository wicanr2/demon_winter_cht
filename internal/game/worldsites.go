package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"

// 世界地圖上的獨立神殿與學院（`docs/re/86`）。
//
// 這兩處**不在城鎮裡**，是地圖上自己的 tile：`0x25` 神殿、`0x26` 學院
//（`docs/re/74` §1 的四筆 tile 分派）。
//
// 兩張表都內嵌在 `DEMON.INT`（不是資料檔），所以照本專案的慣例
// 抄成 Go 原始碼（同 `weaponDamageDice`、`aicast` 那幾張）。
//
// **兩張表都沒有「哪張地圖」欄位。** 分辨方式跟 `EXITS.DAT` 一樣
//（`docs/re/85`）：靠腳下的 tile —— 只有踩在 `0x25`／`0x26` 上才會查表，
// 所以同一組座標在別的地圖上不會誤中。實測 35 筆學院座標有 34 筆
// 恰好落在一個地圖段的 `0x26` 上，**0 筆落在兩個以上**。

// collegeSite 是一座世界地圖上的學院。
type collegeSite struct {
	X, Y  byte
	Skill gamedata.SkillID
}

// collegeSites 是 `ds:0x342d` 的 35 筆三元組（X, Y, 技能）。
//
// 步長 3、上限 105（`0x1c6e3` 的 `+= 3`、`0x1c6e7` 的 `cmp 0x69`），
// 105 ÷ 3 ＝ 35。**表長度看迴圈邊界，不要猜哨兵** ——
// `docs/re/74` 曾經寫「20 座」，因為 #19 的技能欄是 `0` 被當成結束標記，
// 而 `0` 是劍擊（技能 id 0），不是哨兵。
//
// #20 與 #1 的三個欄位完全相同。迴圈找到第一筆就跳出，
// 所以 #20 永遠選不到 —— **照抄，不要去重**：去重會改變「第一筆」是哪一筆。
var collegeSites = []collegeSite{
	{18, 48, 1}, {14, 37, 1}, {28, 25, 2}, {21, 32, 3}, {53, 36, 4},
	{54, 20, 4}, {54, 57, 4}, {45, 6, 6}, {49, 40, 8}, {23, 36, 22},
	{40, 25, 24}, {13, 54, 25}, {46, 7, 26}, {8, 9, 27}, {20, 46, 28},
	{23, 42, 29}, {15, 43, 30}, {17, 9, 10}, {19, 12, 11}, {29, 37, 0},
	{14, 37, 1}, {29, 39, 5}, {19, 29, 8}, {28, 30, 12}, {21, 11, 13},
	{44, 34, 14}, {18, 33, 17}, {25, 58, 18}, {37, 19, 19}, {34, 23, 21},
	{52, 29, 29}, {36, 39, 30}, {51, 4, 20}, {17, 38, 12}, {45, 49, 8},
}

// CollegeSkillAt 回傳站在 (x, y) 的學院教哪個技能。
//
// 呼叫端要先確認腳下 tile 是 `0x26` —— 這張表只有座標，
// 不查 tile 就會在別的地圖上誤中。
func CollegeSkillAt(x, y int) (gamedata.SkillID, bool) {
	for _, c := range collegeSites {
		if int(c.X) == x && int(c.Y) == y {
			return c.Skill, true
		}
	}
	return 0, false
}

// templeSite 是一座世界地圖上的神殿。
type templeSite struct {
	X, Y byte
	// Deity 是 1–10 的神祇編號，直接對得上 `FILES.DTT` 的神祇名表。
	Deity int
}

// templeSites 是 `ds:0x310d` 的 19 筆四元組。
//
// 步長 4、上限 `0x4c`（76），76 ÷ 4 ＝ 19（`0x1be74`／`0x1be78`）。
//
// 原始記錄是 `(X, Y, a, b)`，而程式算的是 `idx = a*2 − b`
//（`0x1be5d`–`0x1be6c`）。資料裡 `a ∈ 1..5`、`b ∈ {0,1}`，
// 所以 `idx ∈ 1..10` 且**每一組 (a,b) 對到唯一的 idx** ——
// 那就是神祇編號。這裡直接存算好的 idx，不留 a／b：
// 拆成兩欄只是原版的資料壓縮，語意上是一個數。
//
// **自我驗證**：城鎮那個入口把城鎮記錄的值直接當 `idx` 用
//（`0x1be1b` 那條 `!= 0xff` 的路徑），兩個入口餵的是同一個索引空間。
// 而 `FILES.DTT` 的神祇名表有 11 筆，容得下 1–10。
var templeSites = []templeSite{
	{44, 51, 1}, {55, 8, 3}, {37, 37, 3}, {14, 15, 3}, {45, 15, 5},
	{22, 53, 7}, {51, 26, 7}, {48, 30, 7}, {18, 26, 9},
	{49, 11, 2}, {24, 15, 2}, {42, 36, 4}, {51, 40, 4}, {42, 17, 6},
	{50, 42, 6}, {9, 25, 8}, {33, 33, 10}, {10, 16, 10}, {41, 6, 10},
}

// TempleDeityAt 回傳站在 (x, y) 的神殿屬於哪位神祇（1–10），沒有回 0。
//
// 同 CollegeSkillAt：呼叫端要先確認腳下 tile 是 `0x25`。
func TempleDeityAt(x, y int) int {
	for _, t := range templeSites {
		if int(t.X) == x && int(t.Y) == y {
			return t.Deity
		}
	}
	return 0
}

// WorldCollegeCount／WorldTempleCount 給測試與文件用。
func WorldCollegeCount() int { return len(collegeSites) }
func WorldTempleCount() int  { return len(templeSites) }
