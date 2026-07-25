package game

// 怪物 AI 的目標選擇。
//
// 原版的怪物回合是 `FUN_138d_03f5`（Ghidra 138d:03f5 = DEMON.INT 檔位移
// 0x78c5）。決策樹是這樣（戰鬥單位記錄 38 bytes 一筆，基底 DS:0x4eb4）：
//
//	7939  cmp [bx+0x4ed6],7  / jle  ┐ 種族／元素類型（unit+0x22）
//	7940  cmp [bx+0x4ed6],13 / jge  ┤ 落在 8–12 才有噴吐
//	7947  rnd(10) < 4               ┘ 30% 機率
//	7959  call 038d:17b8            ; 噴吐攻擊（該函式的錨點字串是 "breathes!"）
//
//	799a  cmp [bx+0x4ec2],0 / jle   ; 法力（unit+0x0e）用完就不施法
//	79a1  rnd(10) > 4               ; 50% 機率
//	7972  call 038d:65e             ; AI 選法術並施放
//
//	79c9  mov ax,[bx+0x4ed2]        ; **記在單位記錄裡的目標槽位**（unit+0x1e）
//	79e8  cmp [bx+0x4eb4],0  / jle  ; 目標的 X（unit+0x00）<= 0 → 換一個
//	79ef  cmp [bx+0x4ed4],10 / jle  ; 目標的陣營（unit+0x20）必須 >= 11 = 玩家
//	79f9  call 0990:0002            ; 打它（與玩家共用的普通攻擊核心）
//	7a07  rnd(15) − 1               ; 換目標：隨機挑一個槽位，回頭重驗
//
// `unit+0x00` 是 **X 座標**，不是 HP —— 由噴吐函式反推出來的：它拿
// `unit+0x00`／`unit+0x02` 加上方向增量表求「面前那一格」（見下）。
// 而 **X == 0 正是「空槽或已死」的哨兵值**（行動順序也用同一條，見
// battlefield.go），所以這個檢查等價於「目標還在場上」。
//
// 這裡實作的是**目標記憶**那一段：怪物會咬著同一個目標打，直到它倒下才
// 隨機換人。這與「每回合都挑第一個敵人」差很多 —— 後者會讓整群怪物擠在
// 同一個隊員身上，前者會自然散開。
//
// 噴吐那一支的入口已經讀了一段（038d:17b8 = 檔位移 0x8c88）：
//
//	8ca2  mov ax,[bx+0x4ec8]   ; 施放者的朝向（unit+0x14）
//	8cad  add ax,[si+0x15da]   ; ΔX 表 = {0, 1, 0, -1}
//	8cb1  add ax,[bx+0x4eb4]   ;   + 施放者 X   → 面前那一格
//	8cb8  mov ax,[bx+0x4eb6]   ; 施放者 Y（unit+0x02）
//	8cbc  add ax,[si+0x15d2]   ; ΔY 表 = {-1, 0, 1, 0}
//	8cd5  迴圈掃全部 15 個槽位，跳過 X == 0 的
//
// `DS:0x15d2` 起的 8 個 word 是 ΔY／ΔX 兩張連續的方向表（`move.go` 早就
// 記著這兩個位址），程式用相差 4 個 word 的兩個基底去讀。詳見 breath.go。
//
// `unit+0x20` 原本標「語意未解」，現在解了：**它是陣營欄位，值 2（怪物）
// 與 11（玩家）** —— 與 `Unit.IsPlayer` 註解裡早就寫著的「原版陣營值 2／11」
// 對上。兩個獨立呼叫點互相印證：AI 用 `> 10` 篩「目標必須是玩家」，
// 噴吐用同一條把命中者分成玩家／怪物兩堆計數（0x8dcf）。
// 所以本實作的 `IsPlayer != of.IsPlayer` 就是原版那條，不是近似。
//
// **還沒實作**：AI 選法術（038d:65e，1263 bytes，Ghidra 反編譯本身有警告）。
// 噴吐的範圍已解，見 breath.go；傷害計算還沒讀完。

// noAITarget 是「還沒有目標」的哨兵值。
const noAITarget = -1

// AITarget 決定這隻怪物這回合要打誰。
//
// 沿用記在單位上的目標；那個目標死了或不再是敵人就隨機換一個，
// 換法照原版：隨機挑一個槽位、驗證、不合格就再挑（rnd(15)−1）。
//
// 找不到任何敵人時回 nil。
func (b *Battle) AITarget(u *Unit) *Unit {
	if u == nil {
		return nil
	}
	if t := b.Unit(u.AITargetSlot); b.validAITarget(u, t) {
		return t
	}

	// 原版是「隨機挑、驗證、不合格再挑」的迴圈。這裡先確認真的還有敵人，
	// 免得在全滅的情況下無限重挑 —— 原版沒有這個保護，因為它只在
	// 「還有敵人活著」的前提下被呼叫。
	alive := b.Enemies(u)
	if len(alive) == 0 {
		u.AITargetSlot = noAITarget
		return nil
	}
	for {
		slot := b.rng.Roll(CombatSlots) - 1
		if t := b.Unit(slot); b.validAITarget(u, t) {
			u.AITargetSlot = slot
			return t
		}
	}
}

// validAITarget 回報這個單位能不能當 of 的攻擊目標。
func (b *Battle) validAITarget(of, t *Unit) bool {
	return t != nil && t.Alive() && t.IsPlayer != of.IsPlayer
}
