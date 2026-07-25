// Package i18n 是執行期的翻譯層：把原版事件敘述換成中文。
//
// 設計上只做一件事 —— 依（來源檔, 記錄索引）查譯文，查不到就回原文。
// 缺譯是可見的降級（畫面上出現英文），不是崩潰，也不是靜默的空字串。
//
// **原文一起存進翻譯檔並在載入時比對。** 原版資料換版或抽取邏輯改動時，
// 譯文會悄悄對到別的記錄上；把原文釘在翻譯檔裡，這種漂移才擋得住。
package i18n

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Entry 是一條可翻譯字串。
type Entry struct {
	// Index 是這條字串在來源事件表裡的記錄索引。
	Index int
	// Source 是原版英文。載入時用來比對，防止譯文對錯記錄。
	Source string
	// Target 是譯文。空字串代表尚未翻譯。
	Target string
	// NeedsReview 代表原文變動過、譯文需要複查。
	NeedsReview bool
}

// Translated 回報這一條是否已翻譯且不需複查。
func (e Entry) Translated() bool { return e.Target != "" && !e.NeedsReview }

// Catalog 是一個來源檔的翻譯目錄。
type Catalog struct {
	// Source 是來源檔名，例如 DATA1.TXT。
	Source  string
	Entries []Entry
}

// CatalogFileName 把來源檔名換成翻譯檔名：DATA1.TXT → data1.txt。
//
// 不用 .json：譯文是散文，JSON 會把換行變成 \n，review diff 幾乎沒法看。
func CatalogFileName(source string) string {
	base := strings.TrimSuffix(source, filepath.Ext(source))
	return strings.ToLower(base) + ".txt"
}

// 翻譯檔的行首標記。刻意選不會出現在遊戲散文開頭的形式。
const (
	markSource = "@@ "   // 檔頭：來源檔名
	markIndex  = "## "   // 一條記錄的開始，後接索引
	markReview = "!! "   // 待複查旗標
	markEN     = ":: en" // 原文區塊開始
	markZH     = ":: zh" // 譯文區塊開始
)

// WriteCatalog 把翻譯目錄寫成純文字檔。
func WriteCatalog(path string, c *Catalog) error {
	var b strings.Builder
	b.WriteString("# 由 cmd/dwstrings 產生。en 區塊不要手改，zh 區塊填譯文。\n")
	b.WriteString("# 原文變動時會標上 !! ，代表譯文需要複查。\n")
	b.WriteString(markSource + c.Source + "\n")

	entries := append([]Entry(nil), c.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Index < entries[j].Index })

	for _, e := range entries {
		b.WriteString("\n" + markIndex + strconv.Itoa(e.Index) + "\n")
		if e.NeedsReview {
			b.WriteString(markReview + "原文已變動，請複查譯文\n")
		}
		b.WriteString(markEN + "\n" + e.Source + "\n")
		b.WriteString(markZH + "\n" + e.Target + "\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// LoadCatalog 讀入一個翻譯目錄。
func LoadCatalog(path string) (*Catalog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c := &Catalog{}
	var cur *Entry
	var block *[]string
	var en, zh []string

	flush := func() {
		if cur == nil {
			return
		}
		cur.Source = strings.Join(en, "\n")
		cur.Target = strings.Join(zh, "\n")
		c.Entries = append(c.Entries, *cur)
		cur, en, zh, block = nil, nil, nil, nil
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	line := 0
	for sc.Scan() {
		line++
		s := sc.Text()
		switch {
		case strings.HasPrefix(s, "#") && !strings.HasPrefix(s, markIndex):
			continue
		case strings.HasPrefix(s, markSource):
			c.Source = strings.TrimSpace(strings.TrimPrefix(s, markSource))
		case strings.HasPrefix(s, markIndex):
			flush()
			idx, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(s, markIndex)))
			if err != nil {
				return nil, fmt.Errorf("i18n: %s:%d 索引解析失敗：%w", path, line, err)
			}
			cur = &Entry{Index: idx}
		case strings.HasPrefix(s, markReview):
			if cur != nil {
				cur.NeedsReview = true
			}
		case s == markEN:
			block = &en
		case s == markZH:
			block = &zh
		default:
			if block != nil {
				*block = append(*block, s)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	flush()

	// 尾端多出來的空行是寫檔格式造成的，不是譯文的一部分。
	for i := range c.Entries {
		c.Entries[i].Source = strings.TrimRight(c.Entries[i].Source, "\n")
		c.Entries[i].Target = strings.TrimRight(c.Entries[i].Target, "\n")
	}
	return c, nil
}

// Stats 回傳已翻譯、待複查、未翻譯的條數。
func (c *Catalog) Stats() (done, review, todo int) {
	for _, e := range c.Entries {
		switch {
		case e.NeedsReview:
			review++
		case e.Target != "":
			done++
		default:
			todo++
		}
	}
	return
}
