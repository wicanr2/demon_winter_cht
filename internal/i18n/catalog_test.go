package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogFileName(t *testing.T) {
	cases := map[string]string{
		"DATA1.TXT": "data1.txt",
		"TOWN.TXT":  "town.txt",
	}
	for in, want := range cases {
		if got := CatalogFileName(in); got != want {
			t.Errorf("CatalogFileName(%q) = %q，預期 %q", in, got, want)
		}
	}
}

// 寫出去再讀回來要一模一樣。譯文含換行、引號、全形標點都不能走樣。
func TestCatalog_RoundTrip(t *testing.T) {
	src := &Catalog{
		Source: "DATA1.TXT",
		Entries: []Entry{
			{Index: 0, Source: "Two guards are in the room", Target: "房裡有兩名衛兵。"},
			{Index: 3, Source: "line one\nline two", Target: "第一行\n第二行"},
			{Index: 7, Source: "They yell 'Glory to Xeres'", Target: "他們高喊「澤瑞斯萬歲」"},
			{Index: 9, Source: "untranslated", Target: ""},
			{Index: 12, Source: "changed", Target: "舊譯文", NeedsReview: true},
		},
	}

	path := filepath.Join(t.TempDir(), "data1.txt")
	if err := WriteCatalog(path, src); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCatalog(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.Source != src.Source {
		t.Errorf("來源檔名 = %q，預期 %q", got.Source, src.Source)
	}
	if len(got.Entries) != len(src.Entries) {
		t.Fatalf("條數 = %d，預期 %d", len(got.Entries), len(src.Entries))
	}
	for i, w := range src.Entries {
		g := got.Entries[i]
		if g.Index != w.Index || g.Source != w.Source ||
			g.Target != w.Target || g.NeedsReview != w.NeedsReview {
			t.Errorf("第 %d 條 = %+v，預期 %+v", i, g, w)
		}
	}
}

// 條目要依索引排序，不隨寫入順序漂移 —— 不然每次重抽都產生假 diff。
func TestWriteCatalog_SortsByIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data1.txt")
	err := WriteCatalog(path, &Catalog{Source: "DATA1.TXT", Entries: []Entry{
		{Index: 9, Source: "c"}, {Index: 1, Source: "a"}, {Index: 4, Source: "b"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{1, 4, 9}
	for i, w := range want {
		if got.Entries[i].Index != w {
			t.Errorf("第 %d 條索引 = %d，預期 %d", i, got.Entries[i].Index, w)
		}
	}
}

func TestCatalog_Stats(t *testing.T) {
	c := &Catalog{Entries: []Entry{
		{Target: "已翻"},
		{Target: "已翻二"},
		{Target: "待複查", NeedsReview: true},
		{Target: ""},
	}}
	done, review, todo := c.Stats()
	if done != 2 || review != 1 || todo != 1 {
		t.Errorf("Stats = (%d, %d, %d)，預期 (2, 1, 1)", done, review, todo)
	}
}

func TestEntry_Translated(t *testing.T) {
	cases := []struct {
		e    Entry
		want bool
	}{
		{Entry{Target: "有譯文"}, true},
		{Entry{Target: ""}, false},
		{Entry{Target: "有譯文但待複查", NeedsReview: true}, false},
	}
	for _, c := range cases {
		if got := c.e.Translated(); got != c.want {
			t.Errorf("%+v.Translated() = %v，預期 %v", c.e, got, c.want)
		}
	}
}

// 譯文裡出現 `## `／`:: en` 這類標記時不能把檔案結構打壞。
//
// 中文譯文不會這樣開頭，但把邊界寫死比祈禱它不發生可靠。
func TestCatalog_MarkerLikeText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data1.txt")
	// 原文含 # 開頭的行：這是註解行的形狀，載入時必須當內容而不是註解。
	src := &Catalog{Source: "X.TXT", Entries: []Entry{
		{Index: 0, Source: "plain", Target: "正常譯文"},
	}}
	if err := WriteCatalog(path, src); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\n## 0\n") {
		t.Error("索引標記應獨佔一行")
	}
}
