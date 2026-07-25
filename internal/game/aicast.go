package game

// AI 施法的效果分派：選定法術之後**要打誰**。
//
// 原版在 `138d:0865` 的 `JMP 0x1000:4557` 跳進一張 17 項的跳表
// （表在 `138d:0c65`，17 個 word，dispatcher 緊接在 `138d:0c87`），
// 索引是效果類型 `ds:0x4e2e`。從 DEMON.INT 直接讀出來的表：
//
//	效果 1        → 138d:0868   以某個玩家為中心的 5×5 範圍
//	效果 2        → 138d:09fd   單體，玩家側
//	效果 3–7      → 138d:0a4d   單體，**依 K 的正負決定打自己人還是打玩家**
//	效果 0x0b     → 138d:0b29   單體，玩家側（先掃過一遍確認有人可打）
//	效果 0x0e     → 138d:0adc   單體，玩家側
//	效果 0x10     → 138d:0bef   單體，玩家側（挑一次就算，不重挑）
//	其餘（0、8、9、10、12、13、15）→ 138d:0c63，什麼都不做
//
// # 挑目標的共同骨架
//
// 每一支都是「`rnd(15) − 1` 隨機挑一個槽位、驗證、不合格再挑」。驗證條件
// 依效果而異，但共通的兩條是 `X != 0`（在場上）與陣營（見 side.go）。
//
// # 最要緊的一條：K 的正負決定打哪一邊
//
// 效果 3–7 是唯一會分流的一支（`138d:0a4d`）：
//
//	0a4d  CMP [0x4e30],0x0 / JLE 0x0a8c   ; K <= 0 → 打玩家
//	      ; K > 0（增益）
//	0a7c  CMP [BX+0x4ec4],0x1 / JG  retry ; 目標狀態必須 <= 1
//	0a83  CMP [BX+0x4ed4],0xa / JG  retry ; 目標必須在怪物側
//	      ; K <= 0（傷害／減損）
//	0ab4  CMP [BX+0x4ed4],0xa / JL  retry ; 目標必須在玩家側
//
// `0x4e30` 就是法術記錄的 K（`docs/re/09` §4 早就記著「有號係數，正負決定
// 增益／減損方向」）。**缺這一段的後果是可以直接在畫面上看到的**：怪物法師
// 會對玩家施放「力量術」之類的增益。
//
// 其餘六支沒有 K 分流，一律打玩家側 —— 也就是說怪物只有效果 3–7 的法術
// 會拿來加持自己人。
//
// # 效果 1 的範圍與否決
//
// 效果 1 先隨機挑一個**玩家側**單位當中心（`0x0868`–`0x0895`），然後掃全部
// 15 個槽位，用 `|Δx| < 3 && |Δy| < 3` 決定誰在範圍內 —— 5×5 的方框，
// 中心那一格算在內。施法者自己若落在框內也算一個己方命中（`0x08c2`–`0x090e`）。
//
// 掃完之後（`0x09ae`）：
//
//	09ae  MOV AX,[BP-0xe] / SHL AX,1 / CMP AX,[BP-0x10] / JG 中止
//
// 也就是**己方命中數 × 2 > 敵方命中數就放棄**，而且會把投入的法力
// 退回去（`0x09d9`：`ADD [BX+0x4ec2],AX`）。這條讓怪物不會把自己炸死。
//
// 己方命中的計數還有一個例外（`0x097e`–`0x0996`）：法術的 `School`
// （`ds:0x4e2c`）等於 4 時，己方單位裡種族／元素類型是 4、7、10 的
// **不算誤傷**。那組 {4, 7, 10} 與噴吐的免疫組是同一組（見 breath.go），
// 但兩邊的觸發條件不同（那邊看施放者類型 10，這邊看法術 School 4），
// 所以只照抄，不推廣成一條通則。
//
// # 尚未接上
//
// 各效果實際造成什麼（`0x4c1d`／`0x47dd`／`0x498c`／`0x4875`／`0x46d4`／
// `0x4586` 六個 far call）走的是與玩家共用的效果套用路徑，這裡不重複實作。

// AIAreaRadius 是效果 1 的方框半徑：`|Δ| < 3` → 5×5，中心含在內。
const AIAreaRadius = 2

// aiAreaElement 是「己方誤傷不計」那條例外的法術 School 值（0x097e）。
const aiAreaElement = 4

// aiAreaImmuneRaces 是上面那條例外裡不算誤傷的種族／元素類型。
// 與 breathImmuneRaces 是同一組值，但觸發條件不同，所以分開寫。
var aiAreaImmuneRaces = map[int]bool{4: true, 7: true, 10: true}

// aiEffectHandled 是跳表裡有實作的效果類型。其餘走 default（什麼都不做）。
var aiEffectHandled = map[int]bool{
	1: true, 2: true, 3: true, 4: true, 5: true,
	6: true, 7: true, 0x0b: true, 0x0e: true, 0x10: true,
}

// AIEffectHandled 回報 AI 拿到這個效果類型時會不會真的做事。
func AIEffectHandled(effect int) bool { return aiEffectHandled[effect] }

// AIEffectIsArea 回報這個效果走不走範圍那一支（只有效果 1）。
func AIEffectIsArea(effect int) bool { return effect == 1 }

// AISpellTargetsOwnSide 回報 AI 這次施法要打自己人還是打玩家。
//
// 只有效果 3–7 會看 K 的正負；其餘一律打玩家側。
func AISpellTargetsOwnSide(effect, k int) bool {
	return effect >= 3 && effect <= 7 && k > 0
}

// AIPickTarget 替 AI 挑一個施法目標。
//
// ownSide 為 true 時挑與施法者同側的單位（增益），否則挑對面（傷害）。
// maxStatus 是目標狀態的上限：增益那一支要求 `<= 1`（0x0a7c），
// 傷害那一支不檢查，傳負數代表不限。
//
// 原版是無界重挑；這裡先確認真的有合格對象再挑，避免全滅時轉不出來
// —— 效果 0x0b 那一支（0x0b29）原版自己也是先掃一遍才進迴圈，同一個手法。
func (b *Battle) AIPickTarget(caster *Unit, ownSide bool, maxStatus int) *Unit {
	if caster == nil {
		return nil
	}
	want := caster.OnPlayerSide()
	if !ownSide {
		want = !want
	}

	ok := func(u *Unit) bool {
		if u == nil || !u.Alive() || u.X == 0 {
			return false
		}
		if u.OnPlayerSide() != want {
			return false
		}
		return maxStatus < 0 || int(u.Status) <= maxStatus
	}

	any := false
	for slot := 0; slot < CombatSlots; slot++ {
		if ok(b.Unit(slot)) {
			any = true
			break
		}
	}
	if !any {
		return nil
	}
	for {
		if u := b.Unit(b.rng.Roll(CombatSlots) - 1); ok(u) {
			return u
		}
	}
}

// AIAreaCount 是效果 1 掃完方框之後的兩個計數。
type AIAreaCount struct {
	// Own 是施法者這一側的命中數，**含施法者自己**。
	Own int
	// Enemy 是對面的命中數。
	Enemy int
}

// Veto 回報這次範圍法術該不該放棄。原版：己方 × 2 > 敵方就退款中止。
func (c AIAreaCount) Veto() bool { return c.Own*2 > c.Enemy }

// AIAreaCountAt 數出以 (cx, cy) 為中心的 5×5 方框裡兩邊各命中幾個。
//
// school 是法術的 School 欄位，等於 aiAreaElement 時，己方單位裡種族／
// 元素類型落在 aiAreaImmuneRaces 的不計入誤傷。
func (b *Battle) AIAreaCountAt(caster *Unit, cx, cy, school int) AIAreaCount {
	var c AIAreaCount
	if caster == nil {
		return c
	}
	// 施法者自己先算：原版是獨立的一段（0x08c2–0x090e），不走底下的迴圈。
	if inAIArea(cx, cy, caster.X, caster.Y) {
		c.Own++
	}
	for slot := 0; slot < CombatSlots; slot++ {
		u := b.Unit(slot)
		if u == nil || u == caster || !u.Alive() || u.X == 0 {
			continue
		}
		if !inAIArea(cx, cy, u.X, u.Y) {
			continue
		}
		if u.OnPlayerSide() != caster.OnPlayerSide() {
			c.Enemy++
			continue
		}
		if school == aiAreaElement && aiAreaImmuneRaces[u.RaceOrElement] {
			continue // 同屬性，不算誤傷
		}
		c.Own++
	}
	return c
}

// inAIArea 判斷 (x, y) 在不在以 (cx, cy) 為中心的方框內。
func inAIArea(cx, cy, x, y int) bool {
	return absInt(x-cx) <= AIAreaRadius && absInt(y-cy) <= AIAreaRadius
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
