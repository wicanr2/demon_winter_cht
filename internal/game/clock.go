// Package game 是規則層：時間、移動、角色、戰鬥。
//
// 這一層不知道畫面與輸入裝置的存在，也不知道語言。所有數值行為對齊原版，
// 依據是 docs/spec/ 底下標 READY 的規格。
package game

// 三層時間計數器的進位常數。全部 1-based：重設值是 1，不是 0。
// 依據 docs/spec/06-time.md（原始位元組已核對）。
const (
	hourWrap  = 38 // 小時到 38 → 歸 1、日 +1，故實際單位數 37
	dayWrap   = 35 // 日到 35 → 歸 1、月 +1，故一個月 34 天
	monthWrap = 23 // 月到 23 → 歸 1，故一年 22 個月

	// 步數計數器達這個值才推進一小時。是 11 不是 10。
	stepsPerHour = 11

	// 新遊戲的起始時間。Hour 5 正好是全日照的第一個小時。
	startHour = 5
	startDay  = 8
)

// LightLevel 是視野／光照等級，0 最亮、3 最暗。
type LightLevel int

const (
	LightFull LightLevel = iota // 全日照
	LightDim1
	LightDim2
	LightDark
)

// lightByHour 以小時為索引查光照等級，對應原版 31f0:0x209c 的表。
// 索引 0 不會被用到（小時是 1-based），填 LightDim2 只是佔位。
var lightByHour = func() [hourWrap]LightLevel {
	var t [hourWrap]LightLevel
	set := func(lo, hi int, l LightLevel) {
		for h := lo; h <= hi; h++ {
			t[h] = l
		}
	}
	t[0] = LightDim2
	set(1, 2, LightDim2)
	set(3, 4, LightDim1)
	set(5, 13, LightFull)
	set(14, 15, LightDim1)
	set(16, 17, LightDim2)
	set(18, hourWrap-1, LightDark)
	return t
}()

// forcedCampHour 是天黑強制紮營的小時。原版時間推進函式在此回傳 0x0e。
const forcedCampHour = 24

// 可以睡覺的小時區間（含端點）。落在區間外會得到 "You are restless"。
// 證據：FUN_1000_0206 要求 struct+0x9f ∈ [0x0f, 0x18]，見 docs/re/08 §8.2。
const (
	sleepEarliestHour = 15
	sleepLatestHour   = 24
)

// Clock 是遊戲世界的時間，含驅動它的步數計數器。
//
// 時間不會自己走 —— 只有玩家移動才推進。這點與多數 CRPG 不同，要照做。
type Clock struct {
	hour  int
	day   int
	month int
	steps int
}

// NewClock 回傳新遊戲的起始時間（Hour 5、Day 8、Month 1）。
func NewClock() *Clock {
	return &Clock{hour: startHour, day: startDay, month: 1}
}

func (c *Clock) Hour() int  { return c.hour }
func (c *Clock) Day() int   { return c.day }
func (c *Clock) Month() int { return c.month }

// Steps 回傳目前的步數計數器值（1..10；剛好推進完是 1）。
func (c *Clock) Steps() int { return c.steps }

// Light 回傳目前小時的光照等級。
func (c *Clock) Light() LightLevel {
	if c.hour < 0 || c.hour >= hourWrap {
		return LightDark
	}
	return lightByHour[c.hour]
}

// ForcedCamp 回報是否已到天黑強制紮營的時刻。
func (c *Clock) ForcedCamp() bool { return c.hour == forcedCampHour }

// CanSleep 回報目前時段是否允許睡覺。不在區間內時原版印 "You are restless"。
func (c *Clock) CanSleep() bool {
	return c.hour >= sleepEarliestHour && c.hour <= sleepLatestHour
}

// AdvanceHour 推進一小時，並執行時／日／月三層進位。
func (c *Clock) AdvanceHour() {
	c.hour++
	if c.hour == hourWrap {
		c.hour = 1
		c.day++
		if c.day == dayWrap {
			c.day = 1
			c.month++
			if c.month == monthWrap {
				c.month = 1
			}
		}
	}
}

// Step 記一步，必要時推進小時。對應原版的 FUN_222f_05c6。
//
// 回傳 true 表示這一步跨過了 11 的門檻、小時已經 +1。
//
// 計數器達 11 時重設為 **1 而不是 0** —— 所以第一次推進要走 11 步，
// 之後每次只要 10 步。這是原版行為，不是 off-by-one。
//
// 原版在推進前還會插一次飢餓／中毒判定，判定回報隊伍全滅時就不推進時間。
// 那一層屬於隊伍狀態，不在 Clock 的職責內，由呼叫端在 Step 之前處理。
func (c *Clock) Step() bool {
	c.steps++
	if c.steps < stepsPerHour {
		return false
	}
	c.steps = 1
	c.AdvanceHour()
	return true
}

// SleepUntil 把時間設到指定小時。過夜（跨日）時 crossDay 傳 true。
//
// 一般睡眠不換日、醒來落在 1–6 的黎明時段（由呼叫端擲 Roll(6) 決定 hour）。
func (c *Clock) SleepUntil(hour int, crossDay bool) {
	c.hour = hour
	if !crossDay {
		return
	}
	c.day++
	if c.day == dayWrap {
		c.day = 1
		c.month++
		if c.month == monthWrap {
			c.month = 1
		}
	}
}
