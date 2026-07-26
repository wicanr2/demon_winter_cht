package manual

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const realManual = "../../assets/manual/zh-Hant/manual.txt"

func TestParse_Sections(t *testing.T) {
	m := Parse([]byte(`前言會被忽略

== 第一章
一行
　　縮排要保留

還有一段

== 第二章
內容
`))
	if m.Len() != 2 {
		t.Fatalf("章節數 = %d，預期 2（章節前的文字要忽略）", m.Len())
	}
	if got := m.Titles(); got[0] != "第一章" || got[1] != "第二章" {
		t.Errorf("標題 = %q", got)
	}
	s := m.At(0)
	want := []string{"一行", "　　縮排要保留", "", "還有一段"}
	if len(s.Lines) != len(want) {
		t.Fatalf("第一章 %d 行，預期 %d：%q", len(s.Lines), len(want), s.Lines)
	}
	for i := range want {
		if s.Lines[i] != want[i] {
			t.Errorf("第 %d 行 = %q，預期 %q —— 空行是段落間隔、行首空白是表格對齊，兩者都要保留",
				i, s.Lines[i], want[i])
		}
	}
	if m.At(-1) != nil || m.At(9) != nil {
		t.Error("界外索引應回 nil")
	}
}

// 缺檔不算錯：手札是輔助內容，缺了不該擋住玩遊戲。
func TestLoad_MissingIsEmpty(t *testing.T) {
	m, err := Load(filepath.Join(t.TempDir(), "沒有這個檔"))
	if err != nil {
		t.Fatalf("缺檔不該是錯誤：%v", err)
	}
	if m.Len() != 0 {
		t.Errorf("預期空手札，得到 %d 章", m.Len())
	}
}

// TestRealManual 對實際的手札做健檢。
//
// 這一項擋的是「編輯內容時把格式弄壞」——手札是純資料，沒有編譯期檢查，
// 少一個 `==` 就整章併到上一章去，而畫面上只會看起來「有點長」。
func TestRealManual(t *testing.T) {
	if _, err := os.Stat(realManual); os.IsNotExist(err) {
		t.Skipf("找不到 %s", realManual)
	}
	m, err := Load(realManual)
	if err != nil {
		t.Fatal(err)
	}
	if m.Len() < 5 {
		t.Fatalf("只有 %d 章，手札應該不只這樣", m.Len())
	}
	seen := map[string]bool{}
	for i, s := range m.Sections {
		if strings.TrimSpace(s.Title) == "" {
			t.Errorf("第 %d 章沒有標題", i)
		}
		if seen[s.Title] {
			t.Errorf("章節標題重複：%q", s.Title)
		}
		seen[s.Title] = true
		if len(s.Lines) == 0 {
			t.Errorf("章節 %q 沒有內容", s.Title)
		}
		// 畫面寬度有限：一行超過 30 個全形字會被裁掉。
		for j, l := range s.Lines {
			if n := len([]rune(l)); n > 34 {
				t.Errorf("章節 %q 第 %d 行有 %d 個字，畫面放不下（上限 34）：%q",
					s.Title, j, n, l)
			}
		}
	}
}
