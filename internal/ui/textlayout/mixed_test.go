package textlayout

import (
	"strings"
	"testing"
)

func TestCellWidth(t *testing.T) {
	cases := []struct {
		ch   rune
		want int
	}{
		{'A', CellWidthASC},
		{' ', CellWidthASC},
		{'~', CellWidthASC},
		{'冬', CellWidthCJK},
		{'，', CellWidthCJK},
		{'Ω', CellWidthCJK},
	}
	for _, c := range cases {
		if got := CellWidth(c.ch); got != c.want {
			t.Errorf("CellWidth(%q) = %d，預期 %d", c.ch, got, c.want)
		}
	}
}

// 格寬必須等於字模畫出來的寬度，否則會疊字或留縫。
//
// 這是實際踩過的坑：ASCII 格寬 8 配上放大兩倍的 8×8 字模，
// 畫面上每個英文字母都疊掉前一個的一半。
func TestCellWidth_MatchesRenderedGlyphWidth(t *testing.T) {
	const cgaGlyphWidth, asciiScale = 8, 2

	if CellWidthASC != cgaGlyphWidth*asciiScale {
		t.Errorf("ASCII 格寬 %d，但字模放大後寬 %d —— 會疊字",
			CellWidthASC, cgaGlyphWidth*asciiScale)
	}
	const etenGlyphWidth = 16
	if CellWidthCJK != etenGlyphWidth {
		t.Errorf("中文格寬 %d，但倚天字模寬 %d", CellWidthCJK, etenGlyphWidth)
	}
}

func TestTextWidth_MixesHalfAndFullWidth(t *testing.T) {
	const s = "HP 100/冬"
	want := 7*CellWidthASC + 1*CellWidthCJK
	if got := TextWidth(s); got != want {
		t.Errorf("TextWidth(%q) = %d，預期 %d", s, got, want)
	}
}

// 中文必須逐字可斷。沿用英文的逐詞斷行會讓整段中文變成一個「詞」擠成一行 ——
// 這是中文排版最容易被寫錯的地方，釘死它。
func TestWrapMixed_BreaksChineseBetweenCharacters(t *testing.T) {
	s := strings.Repeat("冬", 10)
	// 一行只放得下 4 個中文字。
	lines := WrapMixed(s, 4*CellWidthCJK)

	if len(lines) != 3 {
		t.Fatalf("10 個中文字、每行 4 個應斷成 3 行，得到 %d 行：%q", len(lines), lines)
	}
	want := []string{"冬冬冬冬", "冬冬冬冬", "冬冬"}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("第 %d 行 = %q，預期 %q", i, lines[i], w)
		}
	}
}

// 沒有一行可以超過指定寬度。這是「不破版」的底線。
func TestWrapMixed_NoLineExceedsWidth(t *testing.T) {
	const width = 20 * CellWidthASC
	inputs := []string{
		"你在雪地裡看見一隻猴子，牠正盯著你手上的春之石。",
		"Demon's Winter 冬之魔 是 SSI 在 1988 年發行的 CRPG。",
		"HP 45/60  SP 12/20  等級 7  經驗 12345",
		strings.Repeat("測試", 50),
	}
	for _, in := range inputs {
		for i, ln := range WrapMixed(in, width) {
			if w := TextWidth(ln); w > width {
				t.Errorf("輸入 %q 的第 %d 行寬 %d 超過 %d：%q", in, i, w, width, ln)
			}
		}
	}
}

// 斷行不能吃字也不能加字。
func TestWrapMixed_PreservesAllNonSpaceRunes(t *testing.T) {
	const in = "你在雪地裡看見 Malifon 的信徒，他們正在唱誦。"
	joined := strings.Join(WrapMixed(in, 10*CellWidthCJK), "")

	strip := func(s string) string { return strings.ReplaceAll(s, " ", "") }
	if strip(joined) != strip(in) {
		t.Errorf("斷行後內容改變：\n得到 %q\n預期 %q", strip(joined), strip(in))
	}
}

// 英文單字不從中間切開。
//
// 只檢查 ASCII：中文相鄰字被斷開是正確行為（「春之石」可以斷成「春之」+「石」），
// 拿它當「不可切開的詞」會測錯方向。
func TestWrapMixed_KeepsAsciiWordsIntact(t *testing.T) {
	words := map[string]bool{"the": true, "Shard": true, "of": true, "Spring": true}

	for _, ln := range WrapMixed("the Shard of Spring 春之石", 12*CellWidthASC) {
		for _, w := range strings.Fields(ln) {
			if w[0] >= 0x80 {
				continue // 中文片段，逐字可斷
			}
			if !words[w] {
				t.Errorf("英文單字被切開：%q（行 %q）", w, ln)
			}
		}
	}
}

// 超過整行寬度的長單字自己佔一行，不會陷入無窮迴圈。
func TestWrapMixed_OverlongWordGetsItsOwnLine(t *testing.T) {
	long := strings.Repeat("A", 30)
	lines := WrapMixed("hi "+long+" bye", 10*CellWidthASC)

	found := false
	for _, ln := range lines {
		if strings.TrimSpace(ln) == long {
			found = true
		}
	}
	if !found {
		t.Errorf("過長的單字應獨佔一行，得到 %q", lines)
	}
}

// 換行字元強制斷行，且保留空行（原版事件文字用空行分段）。
func TestWrapMixed_HonoursExplicitNewlines(t *testing.T) {
	lines := WrapMixed("第一段\n\n第二段", 20*CellWidthCJK)
	want := []string{"第一段", "", "第二段"}
	if len(lines) != len(want) {
		t.Fatalf("得到 %d 行 %q，預期 %d 行 %q", len(lines), lines, len(want), want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("第 %d 行 = %q，預期 %q", i, lines[i], want[i])
		}
	}
}

// 行首不留下被擠過來的空白。
func TestWrapMixed_NoLeadingSpaceOnWrappedLine(t *testing.T) {
	for _, ln := range WrapMixed("冬之魔 是 SSI 的 遊戲 一款 經典 角色 扮演", 8*CellWidthCJK) {
		if strings.HasPrefix(ln, " ") {
			t.Errorf("行首不應有空白：%q", ln)
		}
	}
}

func TestWrapMixed_Degenerate(t *testing.T) {
	if got := WrapMixed("", 100); got != nil {
		t.Errorf("空字串應回傳 nil，得到 %q", got)
	}
	if got := WrapMixed("冬", 0); got != nil {
		t.Errorf("寬度 0 應回傳 nil，得到 %q", got)
	}
	// 連一個中文字都塞不下時仍要終止，不能無窮迴圈。
	if got := WrapMixed("冬之魔", CellWidthASC); len(got) == 0 {
		t.Error("寬度不足一個中文字時仍應回傳內容，不可吃掉文字")
	}
}

// 中文化後的文字視窗仍是每頁 5 行。
func TestNewMixedTextBox_PaginatesByFiveLines(t *testing.T) {
	b := NewMixedTextBox(strings.Repeat("冬", 38*7), MixedColumns)
	if !b.Active() {
		t.Fatal("視窗應為 active")
	}
	if n := len(b.Lines()); n != PageLines {
		t.Errorf("第一頁 %d 行，預期 %d 行", n, PageLines)
	}
	if !b.HasMore() {
		t.Error("7 行文字應有第二頁")
	}
	b.Advance()
	if n := len(b.Lines()); n != 2 {
		t.Errorf("第二頁 %d 行，預期 2 行", n)
	}
	if b.HasMore() {
		t.Error("第二頁後不應還有下一頁")
	}
}

// MixedColumns 一行放得下 38 個中文字：畫布 640 扣掉左右各 16 的邊界。
func TestMixedColumns_FitsThirtyEightCJK(t *testing.T) {
	if n := MixedColumns / CellWidthCJK; n != 38 {
		t.Errorf("一行可放 %d 個中文字，預期 38", n)
	}
	if MixedColumns+2*16 != 640 {
		t.Errorf("MixedColumns + 左右邊界 = %d，預期畫布寬 640", MixedColumns+2*16)
	}
}
