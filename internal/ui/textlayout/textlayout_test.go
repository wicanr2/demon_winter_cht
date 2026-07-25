package textlayout

import (
	"strings"
	"testing"
)

func TestWrapText_RespectsColumnWidth(t *testing.T) {
	s := "The Hall of Bones. The floor of his room is littered with hundreds of skulls and bones."
	lines := WrapText(s, Columns)

	if len(lines) == 0 {
		t.Fatal("不應回傳空結果")
	}
	for i, ln := range lines {
		if len(ln) > Columns {
			t.Errorf("第 %d 行長度 %d 超過 %d 欄：%q", i, len(ln), Columns, ln)
		}
	}

	// 斷行不得改動內容：把所有行接回來要等於原文的詞序列。
	if got, want := strings.Join(lines, " "), strings.Join(strings.Fields(s), " "); got != want {
		t.Errorf("斷行改動了內容\n得到 %q\n預期 %q", got, want)
	}
}

// 逐詞斷行：不能把單字切開。
func TestWrapText_DoesNotSplitWords(t *testing.T) {
	lines := WrapText("aaa bbb ccc ddd", 7)
	want := []string{"aaa bbb", "ccc ddd"}

	if len(lines) != len(want) {
		t.Fatalf("行數：得到 %d %v，預期 %d %v", len(lines), lines, len(want), want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("第 %d 行：得到 %q，預期 %q", i, lines[i], want[i])
		}
	}
}

// 超過整行寬度的單字不切開，讓它自己佔一行。
func TestWrapText_OverlongWordGetsOwnLine(t *testing.T) {
	lines := WrapText("ab abcdefghij cd", 5)
	want := []string{"ab", "abcdefghij", "cd"}

	if len(lines) != len(want) {
		t.Fatalf("行數：得到 %d %v，預期 %d %v", len(lines), lines, len(want), want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("第 %d 行：得到 %q，預期 %q", i, lines[i], want[i])
		}
	}
}

func TestWrapText_EmptyAndDegenerate(t *testing.T) {
	if got := WrapText("", Columns); got != nil {
		t.Errorf("空字串應回傳 nil，得到 %v", got)
	}
	if got := WrapText("   \n\t ", Columns); got != nil {
		t.Errorf("純空白應回傳 nil，得到 %v", got)
	}
	if got := WrapText("abc", 0); got != nil {
		t.Errorf("欄寬 0 應回傳 nil，得到 %v", got)
	}
}

// 換行字元視為斷詞空白，不保留原始換行。
func TestWrapText_NewlineTreatedAsSpace(t *testing.T) {
	lines := WrapText("aaa\nbbb", 20)
	if len(lines) != 1 || lines[0] != "aaa bbb" {
		t.Errorf("得到 %v，預期 [\"aaa bbb\"]", lines)
	}
}

func TestPaginate(t *testing.T) {
	lines := []string{"1", "2", "3", "4", "5", "6", "7"}
	pages := Paginate(lines, PageLines)

	if len(pages) != 2 {
		t.Fatalf("頁數：得到 %d，預期 2", len(pages))
	}
	if len(pages[0]) != 5 || len(pages[1]) != 2 {
		t.Errorf("每頁行數：得到 %d/%d，預期 5/2", len(pages[0]), len(pages[1]))
	}
	if pages[1][1] != "7" {
		t.Errorf("最後一行：得到 %q，預期 \"7\"", pages[1][1])
	}
}

func TestPaginate_Degenerate(t *testing.T) {
	if got := Paginate(nil, 5); got != nil {
		t.Errorf("空輸入應回傳 nil，得到 %v", got)
	}
	if got := Paginate([]string{"a"}, 0); got != nil {
		t.Errorf("每頁 0 行應回傳 nil，得到 %v", got)
	}
}

func TestTextBox_Paging(t *testing.T) {
	// "word " 每個佔 5 欄，40 欄一行放 8 個 → 每頁 5 行 = 40 個詞。
	// 用 100 個確保跨到第 3 頁。
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("word ")
	}
	b := NewTextBox(sb.String())

	if !b.Active() {
		t.Fatal("剛建立時應為 Active")
	}
	if !b.HasMore() {
		t.Fatal("這段文字應該不只一頁")
	}

	pages := 0
	for b.Active() {
		pages++
		b.Advance()
		if pages > 100 {
			t.Fatal("翻頁沒有終止")
		}
	}
	if pages != 3 {
		t.Errorf("翻了 %d 頁，預期 3 頁（100 詞 ÷ 每行 8 詞 ÷ 每頁 5 行）", pages)
	}
	if b.Active() || b.HasMore() {
		t.Error("翻完後應為非 Active 且沒有下一頁")
	}
}

// nil 的 TextBox 要能安全查詢，讓呼叫端不必到處判空。
func TestTextBox_NilSafe(t *testing.T) {
	var b *TextBox
	if b.Active() {
		t.Error("nil TextBox 不應為 Active")
	}
	if b.HasMore() {
		t.Error("nil TextBox 不應有下一頁")
	}
	b.Advance() // 不應 panic
}
