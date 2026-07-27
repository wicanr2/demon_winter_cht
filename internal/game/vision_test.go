package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func dwarf(status scenario.CombatStatus) Character {
	return Character{Name: "矮人", Race: gamedata.Dwarf, Status: status, CurrentHP: 10}
}

func human() Character {
	return Character{Name: "人類", Race: gamedata.Human, CurrentHP: 10}
}

// 戶外看時辰、完全不看光源 —— 這是原版兩條規則互斥的那一半。
func TestViewInsetOutdoorFollowsTheClock(t *testing.T) {
	for _, c := range []struct{ hour, want int }{
		{0, 2}, {2, 2}, {3, 1}, {4, 1}, {5, 0}, {13, 0}, {14, 1}, {15, 1},
		{16, 2}, {17, 2}, {18, 3}, {23, 3},
	} {
		// 光源給 0 與 3 兩種，戶外都不該有差別。
		for _, light := range []byte{0, 3} {
			got := ViewInset(34, c.hour, light, []Character{human()})
			if got != c.want {
				t.Errorf("戶外 %d 時（光源 %d）：內縮 %d，預期 %d",
					c.hour, light, got, c.want)
			}
		}
	}
}

// 地城看光源、完全不看時辰 —— 互斥的另一半。
func TestViewInsetDungeonIgnoresTheClock(t *testing.T) {
	for _, hour := range []int{5, 12, 23} {
		if got := ViewInset(1, hour, 2, []Character{human()}); got != 2 {
			t.Errorf("地城 %d 時：內縮 %d，預期 2（光源 2 → 4−2）", hour, got)
		}
	}
}

// 矮人的黑暗視覺 ＝ 地城光源 +1。
//
func TestDwarfDarkVisionAddsOneToDungeonLight(t *testing.T) {
	noDwarf := []Character{human(), human()}
	withDwarf := []Character{human(), dwarf(scenario.StatusNormal)}

	if got := ViewInset(1, 13, 0, noDwarf); got != 4 {
		t.Errorf("光源 0、沒有矮人：內縮 %d，預期 4（只剩中央一格）", got)
	}
	if got := ViewInset(1, 13, 0, withDwarf); got != 3 {
		t.Errorf("光源 0、有矮人：內縮 %d，預期 3（3×3）", got)
	}
	// 光源已經到 4 就不再加（原版的 `CMP AX,4 / JGE` 閘門）。
	if got := ViewInset(1, 13, 4, withDwarf); got != 0 {
		t.Errorf("光源 4、有矮人：內縮 %d，預期 0", got)
	}
	// 出貨存檔的實機對照：光源 1、**沒有矮人** → 內縮 3 → 3×3，
	// 與 DOSBox 原版在地圖 1 的 (9,32) 抓到的畫面逐格相符。
	// （這一條原本寫成「光照 0 ＋ 三個矮人」—— 那是手寫 PARTY.DAT parse
	// 漂掉的結果，用驗過的 scenario.LoadSaveGame 讀出來才是這組值。）
	if got := ViewInset(1, 13, 1, noDwarf); got != 3 {
		t.Errorf("出貨存檔（光源 1、無矮人）：內縮 %d，預期 3", got)
	}
}

// 門檻照原版 `+0x102 > 1`：束縛與死亡的矮人不算，中毒的還算。
func TestDwarfDarkVisionNeedsALivingDwarf(t *testing.T) {
	for _, c := range []struct {
		status scenario.CombatStatus
		counts bool
		label  string
	}{
		{scenario.StatusNormal, true, "正常"},
		{scenario.StatusPoison, true, "中毒"},
		{2, false, "束縛一級"},
		{scenario.StatusDead, false, "死亡"},
	} {
		inset := ViewInset(1, 13, 0, []Character{dwarf(c.status)})
		got := inset == 3
		if got != c.counts {
			t.Errorf("%s的矮人：內縮 %d（算不算：%v，預期 %v）",
				c.label, inset, got, c.counts)
		}
	}
}

// 內縮把可見範圍收成中央的正方形，而且不會收成負數。
func TestViewVisibleWindow(t *testing.T) {
	for _, c := range []struct{ inset, wantSpan int }{
		{0, 9}, {1, 7}, {2, 5}, {3, 3}, {4, 1},
	} {
		n := 0
		for dy := 0; dy < ViewSpan; dy++ {
			for dx := 0; dx < ViewSpan; dx++ {
				if ViewVisible(dx, dy, c.inset) {
					n++
				}
			}
		}
		if n != c.wantSpan*c.wantSpan {
			t.Errorf("內縮 %d：看得見 %d 格，預期 %d×%d",
				c.inset, n, c.wantSpan, c.wantSpan)
		}
		// 中央那一格永遠看得見（隊伍自己站的地方）。
		if !ViewVisible(ViewSpan/2, ViewSpan/2, c.inset) {
			t.Errorf("內縮 %d：中央那一格看不見", c.inset)
		}
	}
}
