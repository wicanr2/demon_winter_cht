package game

import "testing"

// 表長度照迴圈邊界，不照猜的哨兵。
//
// `docs/re/74` 曾經把學院寫成「20 座」，因為 #19 的技能欄是 `0`
// 被當成表結束標記 —— 而 `0` 是劍擊（技能 id 0），不是哨兵。
// 正確的長度來自 `0x1c6e7` 的 `cmp [bp-4], 0x69`（105 ÷ 3 ＝ 35）。
//
// 神殿同理：`0x1be78` 的 `cmp 0x4c`（76 ÷ 4 ＝ 19）。
func TestWorldSiteTableLengths(t *testing.T) {
	if got := WorldCollegeCount(); got != 35 {
		t.Errorf("學院筆數 = %d，預期 35（105 ÷ 3，迴圈上限 0x69）", got)
	}
	if got := WorldTempleCount(); got != 19 {
		t.Errorf("神殿筆數 = %d，預期 19（76 ÷ 4，迴圈上限 0x4c）", got)
	}
}

// #20 與 #1 是完全相同的一筆，而查表「找到第一筆就跳出」。
//
// 釘住這個是因為它看起來像資料錯誤，會很想去重 ——
// 但去重會改變「第一筆」是哪一筆，而原版永遠選不到 #20。
func TestCollegeDuplicateIsPreserved(t *testing.T) {
	if collegeSites[1] != collegeSites[20] {
		t.Fatalf("#1 %v 與 #20 %v 應該完全相同", collegeSites[1], collegeSites[20])
	}
	// 查 (14,37) 要拿到 #1，不是 #20 —— 兩者技能相同，所以這條測的是
	// 「掃描順序沒被改成從後往前」。
	skill, ok := CollegeSkillAt(14, 37)
	if !ok {
		t.Fatal("(14,37) 應該有學院")
	}
	if skill != collegeSites[1].Skill {
		t.Errorf("(14,37) 教 %d，預期 %d", skill, collegeSites[1].Skill)
	}
}

// 神殿的神祇編號要落在 1–10。
//
// 編碼是 `idx = a*2 − b`，資料裡 `a ∈ 1..5`、`b ∈ {0,1}`
// （`0x1be5d`–`0x1be6c`）。落在範圍外就表示我抄表時算錯了 ——
// 而症狀是「進了神殿但神祇名字是『神祇 13』」，不會有人立刻聯想到抄錯。
func TestTempleDeitiesInRange(t *testing.T) {
	seen := map[int]int{}
	for i, s := range templeSites {
		if s.Deity < 1 || s.Deity > 10 {
			t.Errorf("神殿 #%d (%d,%d) 的神祇 = %d，超出 1–10",
				i, s.X, s.Y, s.Deity)
		}
		seen[s.Deity]++
	}
	// 五個奇數（b=1）與五個偶數（b=0）都要出現 ——
	// 只出現一半的話表示 `a*2 − b` 那個算式我讀反了。
	odd, even := 0, 0
	for d := range seen {
		if d%2 == 1 {
			odd++
		} else {
			even++
		}
	}
	if odd != 5 || even != 5 {
		t.Errorf("神祇編號的奇偶分佈 = 奇 %d／偶 %d，預期各 5（a ∈ 1..5 × b ∈ {0,1}）",
			odd, even)
	}
}

// 查不到的座標要回「沒有」，不能回第一筆。
func TestWorldSiteLookupMisses(t *testing.T) {
	if _, ok := CollegeSkillAt(0, 0); ok {
		t.Error("(0,0) 沒有學院，卻查到了")
	}
	if got := TempleDeityAt(0, 0); got != 0 {
		t.Errorf("(0,0) 沒有神殿，卻回了神祇 %d", got)
	}
}
