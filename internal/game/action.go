package game

// 戰鬥行動點。
//
// 每個單位輪到時拿到一池點數，動作從池裡扣，扣光就換下一個單位。
//
// **點數初值 = 速度屬性。** 反組譯只追到全域變數 `[0x5190]` 被各動作讀寫，
// 初始化點落在 Ghidra 的分析間隙裡沒追出來（見 `docs/re/09` §1.2）；
// 初值來自手冊「每個角色的移動點數等於其速度值」（`docs/manual/part-3.md`）。
// 消耗量兩邊完全吻合，這是手冊補上反組譯缺口的一處。
//
// 動作代號沿用原版跳表的 case 編號（見 `docs/spec/02-combat.md` 動作分派表），
// 不另外編號 —— 對照反組譯時不必再做一次心算換算。
type Action int

const (
	ActionForward    Action = 4
	ActionAttack     Action = 5
	ActionCast       Action = 6
	ActionUseItem    Action = 7
	ActionTurnUndead Action = 8
	ActionDodge      Action = 9
	ActionExamine    Action = 10
	ActionSound      Action = 11
	ActionPray       Action = 12
	ActionLeech      Action = 13

	// ActionTurnCW／ActionTurnCCW／ActionAboutFace 是三個轉向鍵。
	// 原版把轉向做在主迴圈而不是動作跳表裡，這裡給它們負數代號以示區別。
	ActionTurnCW    Action = -1
	ActionTurnCCW   Action = -2
	ActionAboutFace Action = -3

	// ActionEndTurn 是主動結束回合（ESC）。
	ActionEndTurn Action = -4
)

// actionSpec 是一個動作的成本與回合語意。
type actionSpec struct {
	cost int
	// endsTurn 為真代表這個動作做完就換下一個單位，不管還剩多少點。
	endsTurn bool
	// consumesAll 為真代表花掉全部剩餘點數（只有閃避）。
	consumesAll bool
	name        string
}

// actionSpecs 是手冊「移動點數」表。
//
// 手冊在 C／U／T／P／L／ESC 標了星號代表「會結束該角色的回合」；
// **攻擊沒有星號** —— 12 點速度的角色可以攻擊兩次再閃避，
// 這是原版刻意的攻守配置設計，不是疏漏。
var actionSpecs = map[Action]actionSpec{
	ActionForward:    {cost: 2, name: "前進"},
	ActionTurnCW:     {cost: 1, name: "右轉"},
	ActionTurnCCW:    {cost: 1, name: "左轉"},
	ActionAboutFace:  {cost: 1, name: "迴轉"},
	ActionAttack:     {cost: 3, name: "攻擊"},
	ActionCast:       {cost: 3, endsTurn: true, name: "施法"},
	ActionUseItem:    {cost: 3, endsTurn: true, name: "使用道具"},
	ActionTurnUndead: {cost: 3, endsTurn: true, name: "驅散不死"},
	ActionPray:       {cost: 3, endsTurn: true, name: "祈禱"},
	ActionLeech:      {cost: 3, endsTurn: true, name: "汲取法力"},
	ActionDodge:      {cost: 0, endsTurn: true, consumesAll: true, name: "閃避"},
	ActionExamine:    {cost: 0, name: "檢視"},
	ActionSound:      {cost: 0, name: "音效"},
	ActionEndTurn:    {cost: 0, endsTurn: true, name: "結束回合"},
}

// ActionName 回傳動作的中文名稱。
func ActionName(a Action) string { return actionSpecs[a].name }

// ActionCost 回傳動作的點數成本。閃避回傳 0（它吃掉全部剩餘點數）。
func ActionCost(a Action) int { return actionSpecs[a].cost }

// Points 回傳目前行動單位的剩餘行動點數。
//
// 先叫一次 Current() 確保這一輪已配點 —— 配點是在 Current() 裡做的，
// 直接讀欄位會在「還沒有人問過誰該行動」時拿到 0，
// 而那個 0 看起來就像「點數用完了」。
func (b *Battle) Points() int {
	b.Current()
	return b.points
}

// beginTurn 替新輪到的單位配點。
//
// 一個回合內同一個單位只配一次 —— 用 pointsFor 記住是誰的點數池，
// 而不是在 EndTurn 時配下一個人的，因為中途死亡會讓游標跳過單位。
func (b *Battle) beginTurn(u *Unit) {
	if b.pointsFor == u {
		return
	}
	b.pointsFor = u
	b.points = u.Speed
}

// CanAct 回報目前單位是否還付得起這個動作。
func (b *Battle) CanAct(a Action) bool {
	u := b.Current()
	if u == nil {
		return false
	}
	spec, ok := actionSpecs[a]
	if !ok {
		return false
	}
	// 閃避永遠可以按（就算剩 0 點也只是加成為 0），與原版一致：
	// 它是「結束回合並把剩下的點換成迴避」，不是要付費的動作。
	if spec.consumesAll {
		return true
	}
	return b.points >= spec.cost
}

// Spend 扣掉一個動作的點數並處理回合結束。
//
// 回傳 ok=false 代表點數不足，什麼都沒發生 —— 呼叫端不該執行該動作的效果。
// 回傳 spent 是實際花掉的點數（閃避會是全部剩餘）。
//
// **先確認付得起再執行效果。** 原版是進入動作前檢查 `[0x5190] >= 3`，
// 不足就播提示音並直接返回；把順序倒過來會讓角色用零點數攻擊。
func (b *Battle) Spend(a Action) (spent int, ok bool) {
	if !b.CanAct(a) {
		return 0, false
	}
	spec := actionSpecs[a]

	spent = spec.cost
	if spec.consumesAll {
		spent = b.points
	}
	b.points -= spent

	if spec.endsTurn || b.points <= 0 {
		b.EndTurn()
	}
	return spent, true
}

// DoDodge 把剩餘點數全部換成閃避加成並結束回合，回傳狀態計數增量。
//
//	增量 = floor(剩餘點數 / 3)
//
// 做成一支方法而不是「先查加成、再 Spend」兩步 —— 那兩步的順序寫反
// （先扣點再算加成）會永遠得到 0，而且看起來完全合理。
//
// 實際命中率修正見 DodgeHitModifier。手冊寫「每 3 點讓命中率 −1」，
// 命中率的尺度是「技巧 × 4」，所以手冊的 −1 等於程式碼的 −4，兩者一致。
func (b *Battle) DoDodge() int {
	u := b.Current()
	if u == nil {
		return 0
	}
	bonus := b.points / 3
	b.Spend(ActionDodge)
	u.StatusCount += bonus
	return bonus
}
