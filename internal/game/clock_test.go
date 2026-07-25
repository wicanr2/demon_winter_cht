package game

import "testing"

func TestNewClock_StartValues(t *testing.T) {
	c := NewClock()
	if c.Hour() != 5 || c.Day() != 8 || c.Month() != 1 {
		t.Errorf("起始時間應為 Hour 5 / Day 8 / Month 1，得到 %d/%d/%d",
			c.Hour(), c.Day(), c.Month())
	}
	if c.Light() != LightFull {
		t.Errorf("Hour 5 應是全日照，得到 %d", c.Light())
	}
}

// 驗收 2：計數器歸 1（不是歸 0）造成的節奏 ——
// 第一小時要 11 步，之後每小時只要 10 步。
func TestStep_ElevenThenTen(t *testing.T) {
	c := NewClock()
	startHour := c.Hour()

	for i := 1; i <= 10; i++ {
		if c.Step() {
			t.Fatalf("第 %d 步就推進小時，太早", i)
		}
	}
	if !c.Step() {
		t.Fatal("第 11 步應推進小時")
	}
	if c.Hour() != startHour+1 {
		t.Errorf("小時應為 %d，得到 %d", startHour+1, c.Hour())
	}
	if c.Steps() != 1 {
		t.Errorf("計數器應歸 1（不是 0），得到 %d", c.Steps())
	}

	// 第二輪只要 10 步。
	for i := 1; i <= 9; i++ {
		if c.Step() {
			t.Fatalf("第二輪第 %d 步就推進小時，太早", i)
		}
	}
	if !c.Step() {
		t.Fatal("第二輪第 10 步應推進小時")
	}
	if c.Hour() != startHour+2 {
		t.Errorf("小時應為 %d，得到 %d", startHour+2, c.Hour())
	}
}

func TestAdvanceHour_DayAndMonthCarry(t *testing.T) {
	c := &Clock{hour: hourWrap - 1, day: dayWrap - 1, month: monthWrap - 1}

	c.AdvanceHour()

	if c.Hour() != 1 {
		t.Errorf("小時應歸 1，得到 %d", c.Hour())
	}
	if c.Day() != 1 {
		t.Errorf("日應歸 1，得到 %d", c.Day())
	}
	if c.Month() != 1 {
		t.Errorf("月應歸 1，得到 %d", c.Month())
	}
}

// 一個月 34 天：從 Day 1 推進 34 次「日」應該回到 Day 1 並讓月 +1。
func TestDayWrap_ThirtyFourDaysPerMonth(t *testing.T) {
	c := &Clock{hour: 1, day: 1, month: 1}

	days := 0
	for i := 0; i < 34*hourWrap; i++ {
		prev := c.Day()
		c.AdvanceHour()
		if c.Day() != prev {
			days++
		}
		if c.Month() != 1 {
			break
		}
	}

	if days != 34 {
		t.Errorf("一個月應為 34 天，實際推進了 %d 天才換月", days)
	}
	if c.Month() != 2 {
		t.Errorf("應已進入 Month 2，得到 %d", c.Month())
	}
}

func TestLight_Curve(t *testing.T) {
	cases := []struct {
		hour int
		want LightLevel
	}{
		{1, LightDim2}, {2, LightDim2},
		{3, LightDim1}, {4, LightDim1},
		{5, LightFull}, {9, LightFull}, {13, LightFull},
		{14, LightDim1}, {15, LightDim1},
		{16, LightDim2}, {17, LightDim2},
		{18, LightDark}, {30, LightDark}, {37, LightDark},
	}
	for _, tc := range cases {
		c := &Clock{hour: tc.hour, day: 1, month: 1}
		if got := c.Light(); got != tc.want {
			t.Errorf("Hour %d 光照等級：得到 %d，預期 %d", tc.hour, got, tc.want)
		}
	}
}

func TestForcedCampAndSleepWindow(t *testing.T) {
	for h := 1; h < hourWrap; h++ {
		c := &Clock{hour: h, day: 1, month: 1}

		if got, want := c.ForcedCamp(), h == 24; got != want {
			t.Errorf("Hour %d ForcedCamp：得到 %v，預期 %v", h, got, want)
		}
		if got, want := c.CanSleep(), h >= 15 && h <= 24; got != want {
			t.Errorf("Hour %d CanSleep：得到 %v，預期 %v", h, got, want)
		}
	}
}

func TestSleepUntil_CrossDay(t *testing.T) {
	c := &Clock{hour: 20, day: 34, month: 1}
	c.SleepUntil(4, true)

	if c.Hour() != 4 {
		t.Errorf("小時應為 4，得到 %d", c.Hour())
	}
	if c.Day() != 1 || c.Month() != 2 {
		t.Errorf("跨日應讓 Day 34 → Day 1 / Month 2，得到 Day %d / Month %d", c.Day(), c.Month())
	}
}

func TestSleepUntil_SameDay(t *testing.T) {
	c := &Clock{hour: 20, day: 8, month: 1}
	c.SleepUntil(3, false)

	if c.Hour() != 3 || c.Day() != 8 {
		t.Errorf("不換日睡眠：預期 Hour 3 / Day 8，得到 Hour %d / Day %d", c.Hour(), c.Day())
	}
}
