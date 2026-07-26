package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 睡覺（旅店／野外紮營）。
//
// 原版是 `2aed:03e4`（= `0x2000:b2b4`，DEMON.INT 檔位移 `0x1eeb4`）。
// 兩個參數：`param1` 是「野外紮營」旗標，`param2` 是要睡幾晚。
// 紮營選單的 Sleep 傳 `(1, 1)`（`1000:0254`）；旅店那條把 `param2` 強制成 1
// 並走 `param1 == 1` 的分支（`2aed:03f1`）。
//
// # 一晚做的事
//
//	2aed:04ba  rnd(6)                       → r
//	2aed:04c7  睡眠時數 = (27 − 目前時辰) + r − 1
//	2aed:06f0  時辰 ← r                     ; 醒在清晨 1–6 時
//	2aed:0700  日 +1（wrap 0x23 = 35）
//	2aed:071c  月 +1（wrap 0x17 = 23）
//
// **睡眠時數只用來算中毒扣血**，時辰是直接設成 `rnd(6)` 的，不是加上去的。
//
// 每名隊員（`2aed:0554`–`05e3`）：
//
//	若 野外紮營:
//	    有糧食 → 糧食 −1
//	    沒糧食 → HP −2
//	若 狀態 == 1（中毒）:
//	    HP −= 睡眠時數
//	    若 種族 == 4 → HP += 睡眠時數        ; 抵銷，等於免疫
//	HP += 旅店 ? 2 : 1
//	SP += 旅店 ? 10 : 5
//	HP／SP 各自鉗到上限
//	HP <= 0 → 死亡
//
// 回復量是**每晚一次的定額**，不隨睡眠時數增加 —— 想補滿要睡好幾晚，
// 或去治療所。旅店比睡地上好一倍。
//
// # 紮營專屬的三件事（`2aed:0408`–`0471`）
//
//	party+0xa7 = 1        ; 光源重設為 1（火把）
//	party+0xaa = 7        ; 語意未解，紮營時固定設成 7
//	道具 +0x06 = 0        ; 使用次數歸零 ＝ 過夜充能
//	                      ; （限 +0x05 < 100 且 +0x06 != 0xff 的道具）
//
// # 還沒接
//
//   - 睡到一半被打斷（`1000:026d`：休息常式回 0 時整個離開紮營選單，
//     看起來是遭遇；那條路徑還沒讀）。
//   - 睡覺觸發的劇情夢境（`1000:0278`–`0330` 依 `party+0xbd`／`+0xb9`／`+0xba`
//     推進旗標並播兩張圖）。
//   - 紮營選單本身的其餘 12 個選項（Reorder／Identify／Worship／Xorcise／
//     View land／Trade／Drop／Equip／Use／Hunt／Cast／Quit）。

const (
	// RestHourWakeDie 是起床時辰的骰面（`2aed:04ba` 的 `rnd(6)`）。
	RestHourWakeDie = 6
	// restDurationBase 是睡眠時數公式的基準（`2aed:04c7` 的 `0x1b`）。
	restDurationBase = 27

	// restHPInn／restSPInn 是旅店一晚的回復量（`2aed:05b5`）。
	restHPInn, restSPInn = 2, 10
	// restHPCamp／restSPCamp 是野外紮營一晚的回復量（`2aed:05c1`）。
	restHPCamp, restSPCamp = 1, 5

	// restNoFoodHP 是紮營時沒有糧食要扣的血（`2aed:0581` 連兩個 DEC）。
	restNoFoodHP = 2

	// restPoisonImmuneRace 是中毒睡覺不扣血的種族（`2aed:05a1` 的 `== 4`）。
	restPoisonImmuneRace = 4

	// restRechargeMaxTotal 是「過夜會充能」的次數上限（`2aed:045f` 的 `0x64`）。
	// 上限 >= 100 的道具不充能。
	restRechargeMaxTotal = 100
	// restNeverRecharge 是「永不充能」的已用次數哨兵（`2aed:0466` 的 `0xff`）。
	restNeverRecharge = 0xff

	// RestCampTorch 是紮營時重設的光源值（`2aed:040c`）。
	RestCampTorch = 1
)

// RestKind 區分睡在哪裡。
type RestKind int

const (
	// RestInn 睡旅店：回復較多、不吃糧食。
	RestInn RestKind = iota
	// RestCamp 野外紮營：回復較少、吃一份糧食、重設光源、道具充能。
	RestCamp
)

// RestResult 是睡一晚的結果。
type RestResult struct {
	// Hours 是這一晚睡了幾個時辰（只影響中毒扣血）。
	Hours int
	// WakeHour 是起床的時辰（1–6）。
	WakeHour int
	// AteFood 是這一晚有沒有吃到糧食。
	AteFood bool
	// Starved 是沒糧食因而扣血。
	Starved bool
	// Died 列出這一晚死掉的隊員索引。
	Died []int
}

// RestDuration 回傳這一晚會睡幾個時辰。
//
// `(27 − 目前時辰) + rnd(6) − 1`。時辰大於 27 時會是負的 —— 原版沒有鉗制，
// 那個值只拿去扣中毒的血，負數等於中毒者反而回血。照抄，不補鉗制；
// 不過睡覺本來就只在時辰 15–24 才准（見 CanSleep），走不到負的。
func RestDuration(hour, wakeRoll int) int {
	return restDurationBase - hour + wakeRoll - 1
}

// 「睡不睡得著」的時辰區間在 Clock.CanSleep（clock.go 早就解出來了）。

// Rest 讓隊伍睡一晚，直接改動 members 與 clock。
//
// food 是糧食份數的指標；紮營會吃掉一份，沒有就每人扣血。傳 nil 代表不管糧食。
func Rest(r RollSource, kind RestKind, members []Character, clock *Clock,
	food *int) RestResult {

	res := RestResult{WakeHour: 1}
	if r != nil {
		res.WakeHour = r.Roll(RestHourWakeDie)
	}
	hour := 0
	if clock != nil {
		hour = clock.Hour()
	}
	res.Hours = RestDuration(hour, res.WakeHour)

	hpGain, spGain := restHPInn, restSPInn
	if kind == RestCamp {
		hpGain, spGain = restHPCamp, restSPCamp
		if food != nil {
			if *food > 0 {
				*food--
				res.AteFood = true
			} else {
				res.Starved = true
			}
		}
	}

	for i := range members {
		c := &members[i]
		hp, sp := c.CurrentHP, c.CurrentSP

		if res.Starved {
			hp -= restNoFoodHP
		}
		// 中毒（狀態 1）睡覺會掉血，掉的是睡眠時數 —— 睡越久掉越多。
		// 種族 4 把它加回去，等於免疫。
		if c.Status == scenario.StatusPoison && int(c.Race) != restPoisonImmuneRace {
			hp -= res.Hours
		}
		hp += hpGain
		sp += spGain

		if hp > c.MaxHP {
			hp = c.MaxHP
		}
		if sp > c.MaxSP {
			sp = c.MaxSP
		}
		c.CurrentHP, c.CurrentSP = hp, sp
		if hp <= 0 {
			res.Died = append(res.Died, i)
		}
		// 每日一次的旗標由睡覺清掉（`2aed:0513` 那個迴圈）：
		// 鑑定 `+0xed`、敬拜 `+0xf1`、驅邪 `+0xf2`
		// （打獵的 `+0xef` 尚未在規則層建模）。
		c.IdentifiedToday = false
		c.WorshipedToday = false
		c.ExorcisedToday = false

		if kind == RestCamp {
			rechargeItems(c)
		}
	}

	if clock != nil {
		clock.WakeAt(res.WakeHour)
	}
	return res
}

// rechargeItems 把可充能道具的已用次數歸零（`2aed:0455`–`0471`）。
func rechargeItems(c *Character) {
	for i := range c.Inventory {
		it := &c.Inventory[i]
		if it.Empty() || it.Total >= restRechargeMaxTotal ||
			it.Used == restNeverRecharge {
			continue
		}
		it.Used = 0
	}
}

// 打獵（紮營選單的 Hunt）。
//
// 原版在 `1000:08af`–`0945`。手冊：「在野外或在船上，每日一次出去找尋食物，
// 能否找到須靠運氣。」
//
//	1000:0894  if 狀態(char+0x102) > 1 → 印訊息，不能打獵
//	1000:08bb  if 技能旗標(char+0xd0) == 0 → 印訊息，不會打獵
//	1000:08e2  char+0xef = 1                 ; 標記本日已打獵
//	1000:08e8  rnd(16)
//	1000:08f2  收穫 = rnd(16) − 6            ; 負數鉗成 0
//	1000:0933  糧食 += 收穫，上限 255
//
// `char+0xd0` 是技能旗標陣列（`+0xc8` 起）的第 8 格 —— **技能 8 就是「狩獵」**
// （`docs/re/21`）。`char+0xef` 那個旗標由睡覺清掉（`2aed:0513`），
// 兩邊合起來就是手冊講的「每日一次」。
const (
	// HuntDie 是打獵的骰面（`1000:08e8` 的 `rnd(16)`）。
	HuntDie = 16
	// huntPenalty 是骰完要扣的固定值（`1000:08f2` 的 `+0xfffa` ＝ −6）。
	// 所以 16 面裡有 6 面是空手而回。
	huntPenalty = 6
	// SkillHunting 是狩獵技能的 id（記錄內位移 0xd0 − 0xc8）。
	SkillHunting = 8
	// maxRationsHeld 是糧食份數的上限（`1000:0936` 的 `0xff`）。
	maxRationsHeld = 255
)

// HuntResult 是一次打獵的結果。
type HuntResult struct {
	// Gained 是拿到幾份糧食，0 代表空手而回。
	Gained int
	// Reason 非空代表這個人根本打不了獵。
	Reason string
}

// Hunt 讓一名角色打獵，回傳收穫並把糧食加進 rations。
//
// 狀態超過中毒（束縛以上）或沒學狩獵技能的人打不了。
func Hunt(r RollSource, c *Character, rations *int) HuntResult {
	switch {
	case c == nil || r == nil:
		return HuntResult{Reason: "沒有人可以去打獵"}
	case c.Status > scenario.StatusPoison:
		return HuntResult{Reason: c.Name + " 的狀態沒辦法出去打獵"}
	case !c.Skills[SkillHunting]:
		return HuntResult{Reason: c.Name + " 不會狩獵"}
	}

	got := r.Roll(HuntDie) - huntPenalty
	if got < 0 {
		got = 0
	}
	if rations != nil {
		*rations += got
		if *rations > maxRationsHeld {
			*rations = maxRationsHeld
		}
	}
	return HuntResult{Gained: got}
}
