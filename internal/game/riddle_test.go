package game

import "testing"

// 答案不分大小寫（原版走 `stricmp`）。
//
// 這條容易被實作成 `==` —— 攻略上寫的是大寫 `VOID`，測試也照抄大寫，
// 於是小寫過不了關這件事永遠不會被發現，直到玩家打了 `void`。
func TestRiddleCorrectIsCaseInsensitive(t *testing.T) {
	r := RiddleFor(RiddleCaseSpectralPriest)
	if r == nil {
		t.Fatal("找不到幽靈司祭那一題")
	}
	for _, in := range []string{"VOID", "void", "Void", " void ", "vOiD"} {
		if !r.Correct(in) {
			t.Errorf("%q 應該算對", in)
		}
	}
	for _, in := range []string{"", "voi", "voids", "JESRIC"} {
		if r.Correct(in) {
			t.Errorf("%q 應該算錯", in)
		}
	}
}

// 兩題的輸入上限照原版：司祭那題 0x11、神殿那題 7。
//
// 神殿那題的 7 剛好是 `JESRIC` 的長度 + 1（結尾 NUL）——
// **上限比答案短的話這一題會永遠答不出來**，所以要釘住。
func TestRiddleMaxLenFitsAnswer(t *testing.T) {
	for _, c := range []int{RiddleCaseSpectralPriest, RiddleCaseTempleName} {
		r := RiddleFor(c)
		if r == nil {
			t.Fatalf("case %d 沒有題目", c)
		}
		if len(r.Answer) > r.MaxLen {
			t.Errorf("case %d：答案 %q 有 %d 字，輸入上限只有 %d —— 打不完",
				c, r.Answer, len(r.Answer), r.MaxLen)
		}
	}
}

// 非謎題的 case 不該回傳題目。
func TestRiddleForOtherCases(t *testing.T) {
	for _, c := range []int{0, 1, 9, 12, 14, 15, 99} {
		if RiddleFor(c) != nil {
			t.Errorf("case %d 不是密語謎題，卻拿到題目", c)
		}
	}
}
