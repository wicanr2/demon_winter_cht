package scenario

import "testing"

// TestStoryText_PageCounts 釘住三個檔案的頁數。
//
// 頁數是可以被資料打死的預測：`docs/re/04` §5.1 的結局序列用到頁 0–5，
// 夢境只播 0–2（`FUN_1000_0339(n)` → 頁 n−1，n ∈ 1..3），
// 而城鎮變廢墟那段播 `EREGORE.TXT` 第 9 頁。
// 頁數少一頁，那些索引就全部落空。
func TestStoryText_PageCounts(t *testing.T) {
	cases := []struct {
		mode  StoryMode
		pages int
	}{
		{StoryEregore, 10}, // 0–9；檔尾那個空段是最後一個 '*'，不是一頁
		{StoryWin, 7},
		{StoryDream, 3},
	}
	for _, c := range cases {
		name, _ := StoryFileName(c.mode)
		skipIfMissing(t, dataPath(name))
		st, err := LoadStoryText(dataDir, c.mode)
		if err != nil {
			t.Fatalf("%s：%v", name, err)
		}
		if got := len(st.Pages); got != c.pages {
			t.Errorf("%s：%d 頁，預期 %d", name, got, c.pages)
		}
	}
}

// TestStoryText_DreamPagesMatchCode 釘住「第 2 頁把程式碼逐句敘述了一遍」
//（`docs/re/82` §3）。這三句是 `docs/re/79`／`80` 讀出來的三件事的敘事版本，
// 拿它們當錨點：資料換了或分頁解錯，這裡會紅。
func TestStoryText_DreamPagesMatchCode(t *testing.T) {
	name, _ := StoryFileName(StoryDream)
	skipIfMissing(t, dataPath(name))
	st, err := LoadStoryText(dataDir, StoryDream)
	if err != nil {
		t.Fatal(err)
	}
	joined := func(n int) string {
		s := ""
		for _, l := range st.Page(n) {
			s += " " + l
		}
		return s
	}
	if got := joined(1); !contains(got, "Orb of Evertime") {
		t.Errorf("頁 1 應該是馬利馮的預言（第二場夢），得到 %.60q", got)
	}
	page2 := joined(2)
	for _, want := range []string{
		"torn from their souls", // → 清薩滿與司祭技能
		"temples to ruins",      // → 神殿 tile 換成廢墟
		"The gods are dead",     // → 清神祇
	} {
		if !contains(page2, want) {
			t.Errorf("頁 2 應該提到 %q —— 那是 docs/re/79／80 讀出來的三件事之一", want)
		}
	}
}

// TestStoryText_WinFates 釘住結局名單那一頁。
//
// **十段句子對十個職業**是本專案的判讀（數量相符 + 內容明顯分別對應
// 騎士／野蠻人／遊俠／司祭／盜賊／學者…）。數量一旦不是 10，那個判讀就垮了，
// 所以這裡把數量釘住，而不是把「哪一段對哪個職業」釘住。
func TestStoryText_WinFates(t *testing.T) {
	name, _ := StoryFileName(StoryWin)
	skipIfMissing(t, dataPath(name))
	st, err := LoadStoryText(dataDir, StoryWin)
	if err != nil {
		t.Fatal(err)
	}
	page := st.Page(WinFatePage)
	if len(page) != WinFateCount {
		t.Fatalf("結局名單頁有 %d 段，預期 %d（＝職業數）", len(page), WinFateCount)
	}
	for i := 0; i < WinFateCount; i++ {
		if st.Fate(i) == "" {
			t.Errorf("職業 %d 的結局句子是空的", i)
		}
	}
	if st.Fate(-1) != "" || st.Fate(WinFateCount) != "" {
		t.Error("界外的職業編號應回空字串")
	}
}

// TestParseStoryText_KeepsIndent 釘住「行首空白要保留」。
// 馬利馮那段預言的縮排是它的排版，trim 掉會變成一坨。
func TestParseStoryText_KeepsIndent(t *testing.T) {
	st := ParseStoryText([]byte("first\x00    indented\x00\x00   \x00last*"))
	if len(st.Pages) != 1 {
		t.Fatalf("預期 1 頁（檔尾空段不算），得到 %d", len(st.Pages))
	}
	got := st.Pages[0]
	want := []string{"first", "    indented", "last"}
	if len(got) != len(want) {
		t.Fatalf("行數 %d，預期 %d：%q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 行 = %q，預期 %q", i, got[i], want[i])
		}
	}
}

// TestParseStoryText_KeepsInteriorEmptyPages 釘住「中間的空頁不能砍」。
// 原版拿 page 當索引，中間少一頁後面全部錯位 —— 而錯位的結局是
// 「播出來的是別段劇情」，畫面上看起來很正常。
func TestParseStoryText_KeepsInteriorEmptyPages(t *testing.T) {
	st := ParseStoryText([]byte("a**c*"))
	if len(st.Pages) != 3 {
		t.Fatalf("預期 3 頁（a／空／c），得到 %d", len(st.Pages))
	}
	if len(st.Pages[1]) != 0 {
		t.Errorf("中間那頁應該是空的，得到 %q", st.Pages[1])
	}
	if len(st.Pages[2]) != 1 || st.Pages[2][0] != "c" {
		t.Errorf("第 2 頁 = %q，預期 [\"c\"] —— 索引不能被中間的空頁往前擠", st.Pages[2])
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
