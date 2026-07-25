package game

import "testing"

// 投入量不能低於法術最低法力，也不能超過施法者現有的法力。
//
// 上界破了會憑空生出法力，下界破了會用不足的點數施法 —— 兩種都是規則錯，
// 而且畫面上看起來一切正常。
func TestSPInput_ClampsToRange(t *testing.T) {
	for _, c := range []struct{ adjust, want int }{
		{-100, 3}, {-17, 3}, {-10, 10}, {0, 20}, {5, 20}, {100, 20},
	} {
		s := NewSPInput(3, 20)
		s.Adjust(c.adjust)
		if got := s.Amount(); got != c.want {
			t.Errorf("從 20 調整 %+d 後 = %d，預期 %d", c.adjust, got, c.want)
		}
	}
}

// 預設值是目前法力 —— 多數人就是要全投，省一次輸入。
func TestSPInput_DefaultsToMax(t *testing.T) {
	s := NewSPInput(1, 19)
	if got := s.Amount(); got != 19 {
		t.Errorf("預設 = %d，預期 19", got)
	}
}

// 第一次按數字要清掉預設值重打。
//
// 少了這一步，預設 19 的情況下想打 5 會接成 195，然後被上限擋掉、
// 看起來像按鍵沒反應。
func TestSPInput_FirstDigitReplacesDefault(t *testing.T) {
	s := NewSPInput(1, 19)
	s.AppendDigit(5)
	if got := s.Amount(); got != 5 {
		t.Errorf("按 5 之後 = %d，預期 5", got)
	}
}

func TestSPInput_DigitsAccumulate(t *testing.T) {
	s := NewSPInput(1, 150)
	s.AppendDigit(1)
	s.AppendDigit(2)
	s.AppendDigit(3)
	if got := s.Amount(); got != 123 {
		t.Errorf("輸入 1、2、3 之後 = %d，預期 123", got)
	}
}

// 接出來會超過上限的那一鍵直接忽略，不截斷也不歸零。
func TestSPInput_IgnoresOverflowDigit(t *testing.T) {
	s := NewSPInput(1, 19)
	s.AppendDigit(1)
	s.AppendDigit(9)
	if got := s.Amount(); got != 19 {
		t.Fatalf("輸入 1、9 之後 = %d，預期 19", got)
	}
	s.AppendDigit(5)
	if got := s.Amount(); got != 19 {
		t.Errorf("超過上限的那一鍵應被忽略，卻變成 %d", got)
	}
}

// 0 是合法的一個位數（要打 10 就得按得到）。
func TestSPInput_ZeroDigit(t *testing.T) {
	s := NewSPInput(1, 20)
	s.AppendDigit(1)
	s.AppendDigit(0)
	if got := s.Amount(); got != 10 {
		t.Errorf("輸入 1、0 之後 = %d，預期 10", got)
	}
}

// 退格之後再打數字要接在後面，不能又被當成「第一次輸入」清掉。
func TestSPInput_BackspaceThenDigit(t *testing.T) {
	s := NewSPInput(1, 200)
	s.AppendDigit(1)
	s.AppendDigit(2)
	s.AppendDigit(3) // 123
	s.Backspace()    // 12
	s.AppendDigit(5) // 125
	if got := s.Amount(); got != 125 {
		t.Errorf("退格後再按 5 = %d，預期 125", got)
	}
}

// 退到見底再夾回下限，不能變成 0 或負數。
func TestSPInput_BackspaceToEmpty(t *testing.T) {
	s := NewSPInput(3, 20)
	for i := 0; i < 5; i++ {
		s.Backspace()
	}
	if got := s.Amount(); got != 3 {
		t.Errorf("一直退格之後 = %d，預期夾回下限 3", got)
	}
}

// 法力剛好等於最低需求時，範圍退化成一個值，仍然要能施放。
func TestSPInput_ExactlyAffordable(t *testing.T) {
	s := NewSPInput(7, 7)
	if s.Min() != 7 || s.Max() != 7 {
		t.Fatalf("範圍 = %d–%d，預期 7–7", s.Min(), s.Max())
	}
	s.Adjust(-5)
	if got := s.Amount(); got != 7 {
		t.Errorf("調整後 = %d，預期 7", got)
	}
}

// 上界小於下界（法力不足）時不能生出反向區間。
//
// 呼叫端本來就該先擋掉法力不足，但真的傳進來時，Min > Max 會讓 clamp
// 的兩個分支互相打架，Amount() 的結果取決於判斷順序 —— 那種 bug 很難查。
func TestSPInput_HandlesUnaffordable(t *testing.T) {
	s := NewSPInput(10, 4)
	if s.Min() > s.Max() {
		t.Errorf("範圍反了：%d–%d", s.Min(), s.Max())
	}
	if got := s.Amount(); got < s.Min() || got > s.Max() {
		t.Errorf("Amount() = %d 落在範圍 %d–%d 之外", got, s.Min(), s.Max())
	}
}
