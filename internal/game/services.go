package game

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

// 城鎮設施的「做下去會怎樣」那一半。
//
// 定價公式在 economy.go（`docs/re/19` 全部解完了），但那些只回傳數字 ——
// 錢沒扣、傷沒治、糧沒進袋。這個檔案補的是套用結果的那一步。
//
// 每一項都回傳 ServiceResult 而不是直接改全域狀態，理由是呼叫端要能
// 「先問價、玩家確認、再執行」—— 原版每一項也都是先印價錢再問 Yes/No。

// ServiceResult 是一次設施服務的結果。
type ServiceResult struct {
	// OK 為 true 代表服務完成、Gold 已經是扣款後的餘額。
	OK bool
	// Reason 是沒成的原因。
	Reason string
	// Gold 是結束後的金幣（沒成交時等於原值）。
	Gold int
	// Cost 是這次服務的價錢（沒成交時仍然填，方便顯示「你差多少」）。
	Cost int
}

// notEnoughGold 是四個設施共用的同一句話。原版在五處都是同構的 32-bit 比較。
func notEnoughGold(gold, cost int) ServiceResult {
	return ServiceResult{Reason: "金幣不夠", Gold: gold, Cost: cost}
}

// --- 治療所 ---

// Heal 在治療所替一名角色做「他現在需要的那一項」。
//
// 服務項目不是玩家選的，是**由角色狀態決定的**（`docs/re/19` §5.2）：
// 死亡 → 復活、束縛 → 解除、中毒 → 解毒、否則看有沒有外傷。
//
// 三個套用規則照原版：
//   - 四種情況都把狀態清成正常
//   - 復活後生命值是 **1**，不是滿血，且束縛等級一併清零
//   - 一般治療才補滿生命值
func Heal(e Economy, c *Character, gold int) (HealerService, ServiceResult) {
	svc, cost := e.HealerQuote(UnitStatus(c.Status), c.Level, c.BindLevel,
		c.MaxHP-c.CurrentHP)
	switch {
	case svc == HealerNone:
		return svc, ServiceResult{Reason: c.Name + " 不需要治療", Gold: gold}
	case cost > gold:
		return svc, notEnoughGold(gold, cost)
	}

	wasDead := svc == HealerResurrect
	c.Status = 0
	switch {
	case wasDead:
		c.CurrentHP = 1
		c.BindLevel = 0
	case svc == HealerHeal:
		c.CurrentHP = c.MaxHP
	}
	return svc, ServiceResult{OK: true, Gold: gold - cost, Cost: cost}
}

// --- 酒館 ---

// BuyRations 在酒館買糧。
//
// count 必須落在 [MinRations, MaxRations]；總價 = 單價 × 份數。
// 糧食是隊伍共用的一個 byte，所以上限 255（與打獵同一條）。
func BuyRations(e Economy, gold, rations, count int) ServiceResult {
	if count < MinRations || count > MaxRations {
		return ServiceResult{Gold: gold,
			Reason: "一次只能買 1–200 份"}
	}
	cost := e.RationUnitPrice() * count
	if cost > gold {
		return notEnoughGold(gold, cost)
	}
	if rations+count > maxRationsHeld {
		return ServiceResult{Gold: gold, Cost: cost,
			Reason: "隊伍帶不了那麼多糧食"}
	}
	return ServiceResult{OK: true, Gold: gold - cost, Cost: cost}
}

// --- 神殿 ---

// Donate 向神殿捐獻，1 金幣換 1 點經驗值。
//
// 沒有倍率，也不看等級或神祇 —— 就是 1:1（`docs/re/19` §3.2 指令級證據）。
func Donate(c *Character, gold, amount int) ServiceResult {
	if amount < 1 {
		return ServiceResult{Gold: gold, Reason: "要捐多少？"}
	}
	if amount > gold {
		return notEnoughGold(gold, amount)
	}
	c.Experience = TempleDonation(c.Experience, amount)
	return ServiceResult{OK: true, Gold: gold - amount, Cost: amount}
}

// PrayAtTemple 付費祈禱，把呼喚神祇的成功率補回 20。
//
// 三道前置檢查照原版（`docs/re/19` §3.3）：
//
//   - 信的不是這座神殿的神 → 拒絕。**沒有信仰（Deity == 0）的不擋** ——
//     原版是先看「已建立信仰關係」旗標，旗標為零時整段比對跳過。
//   - 成功率已經是滿值 20 → 「你與神的關係已經很好」
//   - 費用 = 角色等級 × 50
func PrayAtTemple(c *Character, gold, townDeity int) ServiceResult {
	if c.Deity != 0 && c.Deity != townDeity {
		return ServiceResult{Gold: gold, Reason: c.Name + " 信的不是這位神"}
	}
	if c.PrayChance >= FavorMax {
		return ServiceResult{Gold: gold,
			Reason: c.Name + " 與神的關係已經很好"}
	}
	cost := PrayCost(c.Level)
	if cost > gold {
		return notEnoughGold(gold, cost)
	}
	c.PrayChance = FavorMax
	// 沒有信仰的人祈禱完就算是這座神殿的信徒了 —— 原版在改宗時才寫 +0xf0，
	// 這裡補上是為了讓「下次還能不能在別座神殿祈禱」有一致的答案。
	// **這一步原版沒有**，是本作的補完；改宗（Convert）還沒實作。
	if c.Deity == 0 {
		c.Deity = townDeity
	}
	return ServiceResult{OK: true, Gold: gold - cost, Cost: cost}
}

// --- 公會 ---

// levelUpExp 是升級所需的累計經驗值，索引是**目前等級**（1 級要 300 點升 2 級）。
//
// **來源是手冊第 6 頁，不是反組譯。** 原版的門檻表用浮點查表存取
// （`FUN_2aed_0770` → 310e 段的軟浮點函式庫），本專案還沒逐一解碼出那 20 個值。
// 攻略 `docs/walkthrough/part-1.md` 的表與手冊一致，兩份獨立來源互相印證，
// 但**都不是程式碼**，所以這裡標明出處而不是宣稱已驗證。
var levelUpExp = [...]int{
	300, 700, 1100, 1800, 2800,
	4600, 7500, 12600, 21600, 37700,
	66400, 118000, 210800, 377600, 677600,
	1217500, 2189300, 3938200, 7086100, 12752200,
}

// MaxLevel 是門檻表涵蓋的最高等級。
const MaxLevel = len(levelUpExp) + 1

// ExpForNextLevel 回傳從目前等級升到下一級所需的累計經驗值。
// 已達最高等級時回傳 0。
func ExpForNextLevel(level int) int {
	if level < 1 || level > len(levelUpExp) {
		return 0
	}
	return levelUpExp[level-1]
}

// CanLevelUp 回報經驗值夠不夠升級，以及還差多少。
//
// 公會**不收錢**（`docs/re/10` §2：整個升級流程沒有任何扣款指令）。
func (c *Character) CanLevelUp() (bool, int) {
	need := ExpForNextLevel(c.Level)
	if need == 0 {
		return false, 0
	}
	if c.Experience >= need {
		return true, 0
	}
	return false, need - c.Experience
}

// --- 學院 ---

// LearnSkill 在學院學一項技能。
//
// 兩種成本要同時付得起（`docs/spec/08` §學院）：
//
//	points   = 技能學費表[技能][職業]           （1–10）
//	金幣費用 = points × (5 × points + 25)
//	還要 角色剩餘可用智力點數 >= points
//
// 智力點數不是另一個欄位，是「智力 − 已學技能各自的成本」算出來的
// （見 Character.RemainingSkillPoints）—— 所以學了就永久佔著。
func LearnSkill(t *gamedata.Tables, c *Character, gold int, skill gamedata.SkillID) (ServiceResult, error) {
	if c.HasSkill(skill) {
		return ServiceResult{Gold: gold, Reason: c.Name + " 已經會這項技能了"}, nil
	}
	points, err := t.SkillCost(skill, c.Class)
	if err != nil {
		return ServiceResult{Gold: gold}, err
	}
	remaining, err := c.RemainingSkillPoints(t)
	if err != nil {
		return ServiceResult{Gold: gold}, err
	}
	cost := CollegeGoldCost(points)
	if remaining < points {
		return ServiceResult{Gold: gold, Cost: cost,
			Reason: fmt.Sprintf("%s 的智力點數不夠（要 %d，剩 %d）",
				c.Name, points, remaining)}, nil
	}
	if cost > gold {
		return notEnoughGold(gold, cost), nil
	}
	c.Skills[skill] = true
	return ServiceResult{OK: true, Gold: gold - cost, Cost: cost}, nil
}
