package game

// 噴吐攻擊。
//
// 原版是 `FUN_138d_17b8`（Ghidra `138d:17b8` = DEMON.INT 檔位移 `0x8c88`，
// 錨點字串 `breathes!`）。怪物 AI 在種族／元素類型落在 8–12 時有 30% 機率
// 走這一支（見 ai.go）。整個函式 1244 bytes，分成**兩趟掃描**：
//
//	第一趟（0x1805–0x196a）  數出兩邊各命中幾個，決定要不要放棄
//	否決檢查（0x196d）      己方×2 > 敵方、或敵方 0 → 中止，回傳碼 3
//	第二趟（0x1a15–0x1c8b）  逐格往外掃、畫動畫、對命中的單位扣血
//
// # 範圍的形狀
//
// 兩趟用不同的寫法描述同一個錐形，這本身就是一組交叉驗證。
//
// 第一趟以**施放者面前那一格**為起點（0x17dd–0x17f0：拿朝向查方向增量表
// 加到施放者座標上），四個方向各一段寫死展開。以朝北為例（0x18be–0x18d4）：
//
//	目標.Y <= 起點.Y            ; 不能在起點後面（等於是可以的）
//	目標.Y >  起點.Y − 4        ; 深度 4，**含起點那一格**
//	|Δx|  <= |Δy|               ; 錐形
//
// 第二趟直接以**施放者自己**為原點，用兩層迴圈掃（0x1a04／0x1c71–0x1c8b）：
//
//	a = 1 … 4                   ; 沿噴吐方向的距離
//	c = −(a−1) … +(a−1)         ; 橫向偏移，|c| <= a − 1
//
// 兩者等價：`a = along + 1`，於是 `|c| <= a − 1 = along`。深度都是 4 格。
//
// # 同族免疫
//
// 出現兩次，條件等價但寫法不同：
//
//   - 第一趟（0x1829–0x1852）：**施放者**種族是 10 且目標種族是 4／7／10 時，
//     這個目標整個跳過，連命中數都不算。
//   - 第二趟（0x1b5f–0x1b8c）：**元素**是 8 且目標種族是 4／7／10 時，
//     傷害歸零（仍然會走完命中流程）。
//
// 元素由施放者種族換來（見 BreathElement），種族 10 ⇒ 元素 8，所以兩處一致。
//
// 對照 `MONSTER.DAT` 就知道這是什麼：種族 10 是**冰龍**，種族 7 是冰系生物
// （冰元素、冰惡魔、極地熊、寒冬狼、雪人、雪巨人、守衛者）——
// **冰的吐息傷不了冰**。火龍（8）與風龍（9）沒有對應分支，這個不對稱是
// 原版本來就有的，不補。
//
// # 傷害
//
//	138d:1b4b  AX = 施放者當前 HP（unit+0x06）
//	138d:1b4f  CX = 3 / IDIV CX
//	138d:1b56  CALLF rnd(AX)
//
// 也就是 `rnd(施放者當前HP ÷ 3)` —— **與投入、等級、目標護甲都無關**，
// 只看噴吐者自己還剩多少血。受傷的龍噴得比較弱。
//
// 扣血在 0x1c1f；`目標.HP > 傷害` 才活得下來（0x1bb5 的 `JG`），
// 所以傷害等於 HP 就死。

// BreathDepth 是噴吐涵蓋幾格深，**含面前那一格**。
const BreathDepth = 4

// breathImmuneRaces 是「同族互不傷害」的那組種族／元素類型。
var breathImmuneRaces = map[int]bool{4: true, 7: true, 10: true}

// breathImmuneCaster 是觸發上面那條免疫的施放者種族。
const breathImmuneCaster = 10

// breathImmuneElement 是同一條免疫在第二趟掃描用的元素值（＝種族 10 換來的）。
const breathImmuneElement = 8

// BreathElement 把噴吐者的種族換成噴吐的元素（0x1992–0x19e8）。
//
//	種族 8  → 元素 6
//	種族 9  → 元素 7
//	種族 10 → 元素 8
//	種族 11 → 元素 rnd(3) + 5，也就是 6、7、8 隨機一個
//
// 種族 12 沒有對應分支，但**那是打不到的**：`MONSTER.DAT` 裡落在 8–12 的
// 只有四隻，全是龍 —— 火龍 8、風龍 9、冰龍 10、巨龍 11。AI 那道
// `8 <= 種族 <= 12` 只是抓得寬一點，元素表已經涵蓋所有真的會出現的值。
// 巨龍擲 `rnd(3)+5` 代表牠三種吐息隨機來一種。
//
// 真要餵 12 進來，原版會沿用 `ds:0x4e2c` 上一趟掃描留下的殘值；
// 這裡回 ok=false，由呼叫端當成「沒有元素」處理。
func BreathElement(r RollSource, race int) (int, bool) {
	switch race {
	case 8:
		return 6, true
	case 9:
		return 7, true
	case 10:
		return 8, true
	case 11:
		if r == nil {
			return 0, false
		}
		return r.Roll(3) + 5, true
	}
	return 0, false
}

// BreathDamage 回傳一次噴吐對單一目標造成的傷害。
//
// `rnd(施放者當前HP ÷ 3)`（整數除法）。HP 不到 3 時商為 0，原版會拿 0 去擲，
// 這裡直接回 0。
func BreathDamage(r RollSource, casterHP int) int {
	n := casterHP / 3
	if r == nil || n <= 0 {
		return 0
	}
	return r.Roll(n)
}

// BreathPlan 是噴吐在真的打下去之前算出來的東西。
type BreathPlan struct {
	// Targets 是會被波及的單位，不分敵我。
	Targets []*Unit
	// Own／Enemy 是兩邊各命中幾個，用來判斷要不要放棄。
	Own, Enemy int
}

// Veto 回報這次噴吐該不該放棄（0x196d–0x197b）。
//
// 兩個條件：一個敵人都沒打到，或**己方命中數 × 2 > 敵方命中數**。
// 與 AI 範圍法術的否決是同一條規則（見 aicast.go），怪物不會為了噴一個人
// 把同伴一起燒掉。
func (p BreathPlan) Veto() bool { return p.Enemy == 0 || p.Own*2 > p.Enemy }

// BreathTargets 回傳這次噴吐會波及哪些單位。
//
// 起點是施放者面前那一格；範圍是從該格往正前方深度 BreathDepth 的錐形。
// 不分敵我 —— 原版掃全部 15 個槽位，只跳過不在場上的（X == 0）與同族免疫的。
func (b *Battle) BreathTargets(caster *Unit) []*Unit {
	return b.BreathPlan(caster).Targets
}

// BreathPlan 掃出波及範圍並數出兩邊的命中數。
func (b *Battle) BreathPlan(caster *Unit) BreathPlan {
	var p BreathPlan
	if caster == nil {
		return p
	}
	dx, dy := Facing(caster.Facing).Delta()
	originX, originY := caster.X+dx, caster.Y+dy

	for slot := 0; slot < CombatSlots; slot++ {
		u := b.Unit(slot)
		if u == nil || !u.Alive() {
			continue // 原版看的是 X == 0 的哨兵，等價於「不在場上」
		}
		if caster.RaceOrElement == breathImmuneCaster &&
			breathImmuneRaces[u.RaceOrElement] {
			continue
		}
		if !inBreathCone(originX, originY, dx, dy, u.X, u.Y) {
			continue
		}
		p.Targets = append(p.Targets, u)
		// 原版數的是陣營 `>= 11`／`< 11`（0x18ff），不是「與施放者不同邊」。
		// 噴吐者一定在怪物側，所以兩種寫法等價。
		if u.OnPlayerSide() {
			p.Enemy++
		} else {
			p.Own++
		}
	}
	return p
}

// Breathe 執行一次噴吐：算範圍、判否決、逐一扣血。
//
// 回傳被打到的單位與各自受的傷害；放棄時回 nil。倒下的單位已經結算過。
func (b *Battle) Breathe(caster *Unit) []BreathHit {
	if caster == nil {
		return nil
	}
	plan := b.BreathPlan(caster)
	if plan.Veto() {
		return nil
	}
	element, hasElement := BreathElement(b.rng, caster.RaceOrElement)

	var out []BreathHit
	for _, u := range plan.Targets {
		dmg := BreathDamage(b.rng, caster.HP)
		if hasElement && element == breathImmuneElement &&
			breathImmuneRaces[u.RaceOrElement] {
			dmg = 0
		}
		hit := BreathHit{Unit: u, Damage: dmg}
		if dmg >= u.HP {
			b.Kill(u)
			hit.Killed = true
		} else {
			u.HP -= dmg
		}
		out = append(out, hit)
	}
	return out
}

// BreathHit 是噴吐打在單一目標上的結果。
type BreathHit struct {
	Unit   *Unit
	Damage int
	Killed bool
}

// inBreathCone 判斷 (tx, ty) 是否落在從 (ox, oy) 朝 (dx, dy) 的錐形內。
//
// along 是沿噴吐方向的距離，across 是垂直方向的偏移。
// 命中條件：`0 <= along < BreathDepth` 且 `|across| <= along`。
//
// **起點那一格算命中**（along = 0，此時 across 只能是 0）——
// 原版排除的是「在起點後面」（0x18c5 的 `JG`、0x18e7 的 `JL`），
// 等於起點的情況不會被排掉。
func inBreathCone(ox, oy, dx, dy, tx, ty int) bool {
	var along, across int
	switch {
	case dx != 0: // 東西向
		along = (tx - ox) * dx
		across = ty - oy
	case dy != 0: // 南北向
		along = (ty - oy) * dy
		across = tx - ox
	default:
		return false
	}
	if along < 0 || along >= BreathDepth {
		return false
	}
	if across < 0 {
		across = -across
	}
	return across <= along
}

// BreathCell 是噴吐掃過的一格。
type BreathCell struct{ X, Y int }

// BreathCone 回傳噴吐會掃過的每一格，順序照原版的第二趟掃描
// （`138d:1a59`，見 `docs/re/23` §8）。
//
// **每一格都會畫**，地形只影響「要不要在這一格找單位」（`0x8f43` 的
// 地形檢查在畫完之後才做）—— 所以動畫要蓋滿整個錐形，不是只蓋打得到的地方。
//
// 掃描順序是「先沿噴吐方向一層一層推進，每一層再從側偏 −along 掃到 +along」。
// 這個順序決定動畫看起來是**從嘴邊往外擴**，不是整片同時亮。
func (b *Battle) BreathCone(caster *Unit) []BreathCell {
	if caster == nil {
		return nil
	}
	dx, dy := Facing(caster.Facing).Delta()
	if dx == 0 && dy == 0 {
		return nil
	}
	var out []BreathCell
	for along := 0; along < BreathDepth; along++ {
		for across := -along; across <= along; across++ {
			x := caster.X + dx*along
			y := caster.Y + dy*along
			if dx != 0 {
				y += across
			} else {
				x += across
			}
			if !InField(x, y) {
				continue
			}
			out = append(out, BreathCell{X: x, Y: y})
		}
	}
	return out
}
