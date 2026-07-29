package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 這一組測試釘的是**擲點的值域**，不是某一次的結果。
//
// 值域是原版運算元的直接推論（`docs/re/68`／`91`），而且**七項都與手冊
// 對得上** —— 手冊是比反組譯更高一階的 oracle（`CONTEXT.md` §4），
// 所以這裡斷言的是「兩個獨立來源都指向的那個區間」。

// trapRNG 給第 i 次取樣一個**散布在整個模數上**的種子。
//
// 直接拿 1、2、3… 當種子是沒有用的：LCG 推進一步是 `state × 125 mod 2796203`，
// 小種子推完還是很小，於是 `Roll(n)` 的第一擲**永遠是 1**。
// 第一版就是這樣寫的，結果「齊射支數 2–2」「警報倒數 1–1」
// 「已注意的陷阱 400 次全躲過」—— 三個測試同時說謊，而程式是對的。
func trapRNG(i int) *rng.RNG {
	return rng.NewWithSeed(uint32(1 + i*(rng.Modulus/samples)))
}

// samples 是每一項的取樣次數。除法要整除得夠開，種子才鋪得滿模數。
const samples = 400

func tough(n int) []Character {
	p := make([]Character, n)
	for i := range p {
		p[i] = Character{Name: "隊員", CurrentHP: 9999, MaxHP: 9999}
	}
	return p
}

func totalDamage(res TrapResult) int {
	sum := 0
	for _, h := range res.Hits {
		sum += h.Damage
	}
	return sum
}

// 七種陷阱的傷害值域。跑夠多次把每一種擲點組合都掃到，
// 再斷言「觀察到的最小／最大值就是預期的邊界」——
// 不是「落在範圍內」，那種斷言鬆到抓不出差一。
func TestTrapDamageRanges(t *testing.T) {
	cases := []struct {
		name     string
		c        TrapCase
		min, max int
	}{
		// 手冊：「1-4 點傷害（護甲無效）」
		{"毒針", TrapPoisonNeedle, 1, 4},
		// 手冊：「50% 機率跌落，一旦跌落會受到 1-6 點傷害」
		{"竹籤坑", TrapPunjiPit, 0, 6},
		// 毒坑與竹籤坑數值完全相同（`docs/re/91` §3）
		{"毒坑", TrapPoisonPit, 0, 6},
		// 手冊：「長矛是飛鏢的加強版，造成 2-7 點傷害」，2–6 支
		{"長矛", TrapSpears, 0, 7 * 6},
		// 手冊：「射出 2-6 支飛鏢…造成 1-3 點傷害」
		{"飛鏢", TrapDarts, 0, 3 * 6},
	}
	for _, tc := range cases {
		lo, hi := 1<<30, -1
		for seed := 1; seed <= samples; seed++ {
			r := trapRNG(seed)
			d := totalDamage(SpringTrap(r, tc.c, false, tough(5)))
			if d < lo {
				lo = d
			}
			if d > hi {
				hi = d
			}
		}
		if lo != tc.min {
			t.Errorf("%s 的最小傷害 = %d，預期 %d", tc.name, lo, tc.min)
		}
		if hi > tc.max {
			t.Errorf("%s 的最大傷害 = %d，超過上界 %d", tc.name, hi, tc.max)
		}
	}
}

// 長矛比飛鏢多擲一顆 4 面骰 —— 這是兩者**唯一**的差別（`0x19d52`）。
//
// 用同一個種子跑：長矛的單發傷害必定 >= 飛鏢，而且會出現 > 3 的值
// （飛鏢的上限就是 3）。分開釘，因為「同一支函式帶不同參數」最容易
// 在重構時被合成一條而漏掉那顆骰子。
func TestSpearIsDartPlusOneDie(t *testing.T) {
	maxDart, maxSpear := 0, 0
	for seed := 1; seed <= samples; seed++ {
		for _, h := range SpringTrap(trapRNG(seed), TrapDarts, false, tough(5)).Hits {
			if h.Damage > maxDart {
				maxDart = h.Damage
			}
		}
		for _, h := range SpringTrap(trapRNG(seed), TrapSpears, false, tough(5)).Hits {
			if h.Damage > maxSpear {
				maxSpear = h.Damage
			}
		}
	}
	if maxDart != 3 {
		t.Errorf("飛鏢單發最大 = %d，預期 3（Roll(3)）", maxDart)
	}
	if maxSpear != 7 {
		t.Errorf("長矛單發最大 = %d，預期 7（Roll(3)+Roll(4)）", maxSpear)
	}
}

// 齊射是 2–6 支，每一支各自判定命中。
func TestVolleyCount(t *testing.T) {
	lo, hi := 1<<30, 0
	for seed := 1; seed <= samples; seed++ {
		n := len(SpringTrap(trapRNG(seed), TrapDarts, false, tough(5)).Hits)
		if n < lo {
			lo = n
		}
		if n > hi {
			hi = n
		}
	}
	if lo != 2 || hi != 6 {
		t.Errorf("齊射支數 = %d–%d，預期 2–6（Roll(5)+1）", lo, hi)
	}
}

// 酸池就是水池每輪多 2 點。**傷害 0 是合法結果** —— 水池的
// Roll(3)−1 可以是 0，原版照樣印一行 `Drowns. 0 damage`。
func TestAcidPoolIsPoolPlusTwo(t *testing.T) {
	poolMax, acidMin, acidMax := 0, 1<<30, 0
	for seed := 1; seed <= samples; seed++ {
		for _, h := range SpringTrap(trapRNG(seed), TrapPool, false, tough(5)).Hits {
			if !h.Missed && h.Damage > poolMax {
				poolMax = h.Damage
			}
		}
		for _, h := range SpringTrap(trapRNG(seed), TrapAcidPool, false, tough(5)).Hits {
			if h.Missed {
				continue
			}
			if h.Damage < acidMin {
				acidMin = h.Damage
			}
			if h.Damage > acidMax {
				acidMax = h.Damage
			}
		}
	}
	if poolMax != 2 {
		t.Errorf("水池單輪最大 = %d，預期 2（Roll(3)−1）", poolMax)
	}
	if acidMin != 2 || acidMax != 4 {
		t.Errorf("酸池單輪 = %d–%d，預期 2–4（水池 +2）", acidMin, acidMax)
	}
}

// 水池會一輪一輪擲到脫困或死亡 —— 手冊「每回合有 33% 機率脫困」。
// 選人只做一次，所以整串 Hits 的 Member 必須全部相同。
func TestPoolRepeatsOnSameVictim(t *testing.T) {
	rounds := 0
	for seed := 1; seed <= samples; seed++ {
		res := SpringTrap(trapRNG(seed), TrapPool, false, tough(5))
		if len(res.Hits) == 0 {
			t.Fatalf("seed %d：水池一輪都沒跑", seed)
		}
		if len(res.Hits) > rounds {
			rounds = len(res.Hits)
		}
		who := res.Hits[0].Member
		for _, h := range res.Hits {
			if h.Member != who {
				t.Fatalf("seed %d：水池換人了（%d → %d）—— 選人在迴圈外只做一次",
					seed, who, h.Member)
			}
		}
		// 最後一輪一定是脫困（或死亡）。
		last := res.Hits[len(res.Hits)-1]
		if !last.Missed && !last.Died {
			t.Fatalf("seed %d：水池沒有出口，最後一輪既沒脫困也沒倒下", seed)
		}
	}
	if rounds < 2 {
		t.Error("水池從來沒有跑超過一輪 —— 那就不是「每回合 33% 脫困」")
	}
}

// 毒針**沒有命中判定**。這是攻略特別點名它的唯一原因，
// 也是「察覺陷阱／解除陷阱大多沒意義，除非被毒針扎中」那句話的來源。
func TestNeedleNeverMisses(t *testing.T) {
	for seed := 1; seed <= samples; seed++ {
		res := SpringTrap(trapRNG(seed), TrapPoisonNeedle, false, tough(5))
		if len(res.Hits) != 1 {
			t.Fatalf("seed %d：毒針打了 %d 下，預期 1", seed, len(res.Hits))
		}
		if res.Hits[0].Missed {
			t.Fatalf("seed %d：毒針落空了 —— 它沒有命中判定", seed)
		}
	}
}

// 警報不扣血，它把遭遇倒數設成 1–5 步。
func TestAlarmSetsCountdownAndDealsNoDamage(t *testing.T) {
	lo, hi := 1<<30, 0
	for seed := 1; seed <= samples; seed++ {
		p := tough(5)
		res := SpringTrap(trapRNG(seed), TrapAlarm, false, p)
		if len(res.Hits) != 0 {
			t.Fatalf("seed %d：警報打了人 —— 它是整組裡唯一不碰 HP 的", seed)
		}
		if res.AlarmCountdown < lo {
			lo = res.AlarmCountdown
		}
		if res.AlarmCountdown > hi {
			hi = res.AlarmCountdown
		}
	}
	if lo != 1 || hi != 5 {
		t.Errorf("警報倒數 = %d–%d，預期 1–5（Roll(5)）", lo, hi)
	}
}

// 「已注意」的格子有 50% 完全不觸發。沒注意到的必定觸發。
func TestNoticedTrapCanBeAvoided(t *testing.T) {
	avoided, fired := 0, 0
	for seed := 1; seed <= samples; seed++ {
		if SpringTrap(trapRNG(seed), TrapPoisonNeedle, true, tough(5)).Avoided {
			avoided++
		} else {
			fired++
		}
	}
	if avoided == 0 || fired == 0 {
		t.Fatalf("已注意的陷阱：躲過 %d 次、觸發 %d 次 —— 應該兩者都有", avoided, fired)
	}
	for seed := 1; seed <= samples; seed++ {
		if SpringTrap(trapRNG(seed), TrapPoisonNeedle, false, tough(5)).Avoided {
			t.Fatalf("seed %d：沒注意到的陷阱不該有迴避", seed)
		}
	}
}

// 死人不會再被扣血，而且陷阱**不篩掉死人再選** ——
// 選到死人這一下就空放。篩掉會改變機率分布。
func TestTrapCanTargetTheDeadAndWastes(t *testing.T) {
	wasted := false
	for seed := 1; seed <= samples && !wasted; seed++ {
		p := []Character{
			{Name: "活", CurrentHP: 9999, MaxHP: 9999},
			{Name: "死", CurrentHP: 0, MaxHP: 20, Status: scenario.StatusDead},
		}
		res := SpringTrap(trapRNG(seed), TrapPoisonNeedle, false, p)
		if res.Hits[0].Member == 1 {
			if res.Hits[0].Damage != 0 || !res.Hits[0].Missed {
				t.Fatalf("seed %d：打到死人卻造成 %d 點傷害", seed, res.Hits[0].Damage)
			}
			if p[1].CurrentHP != 0 {
				t.Fatalf("seed %d：死人的 HP 被改成 %d", seed, p[1].CurrentHP)
			}
			wasted = true
		}
	}
	if !wasted {
		t.Skip("這批種子沒選到死人 —— 沒驗到，但也沒失敗")
	}
}

// 陷阱打死人之後要能宣告全隊死亡。**在這之前這條路是斷的** ——
// 陷阱只印一行訊息，不扣血，所以 `PartyWiped` 永遠不會因為陷阱成立。
func TestTrapCanWipeTheParty(t *testing.T) {
	wiped := false
	for seed := 1; seed <= samples && !wiped; seed++ {
		p := []Character{{Name: "殘血", CurrentHP: 1, MaxHP: 20}}
		res := SpringTrap(trapRNG(seed), TrapPoisonNeedle, false, p)
		if res.Wiped {
			if !PartyWiped(p) {
				t.Fatalf("seed %d：回報全滅但 PartyWiped 說沒有", seed)
			}
			wiped = true
		}
	}
	if !wiped {
		t.Error("一人隊伍被毒針必中扎了 200 次都沒死 —— 死亡路徑沒接上")
	}
}

// case 編號超出 0–8 一律當 case 0，不猜。
func TestTrapCaseForOutOfRange(t *testing.T) {
	for _, v := range []byte{9, 0x10, 0x1f} {
		if got := TrapCaseFor(v); got != TrapUnknown {
			t.Errorf("值 %d → case %d，預期 %d（未讀的那一格）", v, got, TrapUnknown)
		}
	}
	for v := byte(0); v <= 8; v++ {
		if got := TrapCaseFor(v); byte(got) != v {
			t.Errorf("值 %d → case %d，應該原樣對應", v, got)
		}
	}
}

// --- `L` 查看陷阱 ---

// corridor 是一條往東的走道：y 固定，x 從 1 起都是 tile 0x11（可能有記錄），
// 到 wall 那一格變成牆。用 0x11 而不是 0x00 鋪走道，是因為只有 0x11
// 才會去查特殊格清單 —— 這樣才驗得到「掃到底」而不是「第一格就停」。
type corridor struct{ wallX int }

func (c corridor) TileAt(x, y int) (byte, error) {
	if x >= c.wallX {
		return 0x62, nil // 隨便一個非走道 tile
	}
	return tileEventGateA, nil
}

func trapTilesAt(coords ...[2]int) *scenario.SpecialTiles {
	st := &scenario.SpecialTiles{}
	for _, c := range coords {
		st.Tiles = append(st.Tiles, scenario.SpecialTile{
			X: byte(c[0]), Y: byte(c[1]),
			Attr: scenario.SpecialClassTrap<<5 | byte(TrapPoisonNeedle),
		})
	}
	return st
}

func withSkill(s gamedata.SkillID) Character {
	c := Character{Name: "盜賊", CurrentHP: 10, MaxHP: 10}
	c.Skills[s] = true
	return c
}

// 有察覺陷阱技能就是 100%：掃描範圍內的每一格都要找到。
func TestLookForTrapsWithDetectFindsAll(t *testing.T) {
	st := trapTilesAt([2]int{1, 5}, [2]int{2, 5}, [2]int{3, 5})
	scan := LookForTraps(trapRNG(1), []Character{withSkill(SkillDetectTraps)},
		st, corridor{wallX: 10}, 0, 5, East, 3)

	if !scan.HasDetect || scan.HasDisarm {
		t.Fatalf("技能判定錯了：detect=%v disarm=%v", scan.HasDetect, scan.HasDisarm)
	}
	if len(scan.Spots) != 3 {
		t.Fatalf("找到 %d 格，預期 3（有技能就是 100%%）", len(scan.Spots))
	}
	// 沒有解除陷阱 → 三格都變成「已注意」＝ 類別 6。
	for i, tile := range st.Tiles {
		if got := tile.Class(); got != scenario.SpecialClassTrapAlt {
			t.Errorf("第 %d 格類別 = %d，預期 %d（已注意）",
				i, got, scenario.SpecialClassTrapAlt)
		}
	}
}

// 有解除陷阱技能 → 整格清成 0，之後不是陷阱了。
func TestLookForTrapsWithDisarmClearsTheTile(t *testing.T) {
	st := trapTilesAt([2]int{1, 5})
	party := []Character{withSkill(SkillDetectTraps), withSkill(SkillDisarmTraps)}
	scan := LookForTraps(trapRNG(1), party, st, corridor{wallX: 10}, 0, 5, East, 3)

	if len(scan.Spots) != 1 || !scan.Spots[0].Disarmed {
		t.Fatalf("預期解除一格，得到 %+v", scan.Spots)
	}
	if st.Tiles[0].Attr != 0 {
		t.Errorf("解除後 attr = %#x，預期 0", st.Tiles[0].Attr)
	}
}

// 沒有技能就是 25%：掃很多次，找到的比例要明顯不是 0 也不是 1。
//
// 這裡刻意**不**斷言精確比例 —— 斷言比例就是在測 RNG 而不是測規則。
// 要釘的是「有技能與沒技能走的是不同分支」。
func TestLookForTrapsWithoutSkillIsChancy(t *testing.T) {
	found, total := 0, samples
	for i := 1; i <= total; i++ {
		st := trapTilesAt([2]int{1, 5})
		scan := LookForTraps(trapRNG(i), []Character{{Name: "路人", CurrentHP: 10}},
			st, corridor{wallX: 10}, 0, 5, East, 3)
		found += len(scan.Spots)
	}
	if found == 0 || found == total {
		t.Fatalf("沒有技能時找到 %d/%d —— 應該是機率不是必然", found, total)
	}
}

// 掃描距離就是光源強度，而且撞到牆就停。
func TestLookForTrapsStopsAtLightAndWall(t *testing.T) {
	far := trapTilesAt([2]int{5, 5})
	scan := LookForTraps(trapRNG(1), []Character{withSkill(SkillDetectTraps)},
		far, corridor{wallX: 10}, 0, 5, East, 3)
	if len(scan.Spots) != 0 {
		t.Errorf("光源 3 卻掃到 x=5 的陷阱 —— 掃描距離沒有跟著光源")
	}

	behind := trapTilesAt([2]int{4, 5})
	scan = LookForTraps(trapRNG(1), []Character{withSkill(SkillDetectTraps)},
		behind, corridor{wallX: 3}, 0, 5, East, 10)
	if len(scan.Spots) != 0 {
		t.Errorf("牆在 x=3，卻掃到 x=4 的陷阱 —— 撞牆沒有停")
	}
}

// 被束縛或死掉的角色不算數（原版 `+0x102 <= 1`）。
func TestLookForTrapsIgnoresIncapacitatedMembers(t *testing.T) {
	bound := withSkill(SkillDetectTraps)
	bound.Status = scenario.StatusBound1
	dead := withSkill(SkillDisarmTraps)
	dead.Status = scenario.StatusDead

	scan := LookForTraps(trapRNG(1), []Character{bound, dead},
		trapTilesAt([2]int{1, 5}), corridor{wallX: 10}, 0, 5, East, 3)
	if scan.HasDetect || scan.HasDisarm {
		t.Errorf("被束縛／死亡的人不該提供技能：detect=%v disarm=%v",
			scan.HasDetect, scan.HasDisarm)
	}

	// 中毒還是能用（原版的門檻是 <= 1，中毒剛好是 1）。
	poisoned := withSkill(SkillDetectTraps)
	poisoned.Status = scenario.StatusPoison
	scan = LookForTraps(trapRNG(1), []Character{poisoned},
		trapTilesAt([2]int{1, 5}), corridor{wallX: 10}, 0, 5, East, 3)
	if !scan.HasDetect {
		t.Error("中毒的人應該還能察覺陷阱（門檻是 <= 1，不是 == 0）")
	}
}

// 「已注意」只能推進一次 —— 掃兩次不會變成類別 9。
func TestNoticedAdvancesOnlyOnce(t *testing.T) {
	st := trapTilesAt([2]int{1, 5})
	party := []Character{withSkill(SkillDetectTraps)}
	for i := 0; i < 3; i++ {
		LookForTraps(trapRNG(i+1), party, st, corridor{wallX: 10}, 0, 5, East, 3)
	}
	if got := st.Tiles[0].Class(); got != scenario.SpecialClassTrapAlt {
		t.Errorf("掃三次之後類別 = %d，預期停在 %d", got, scenario.SpecialClassTrapAlt)
	}
}
