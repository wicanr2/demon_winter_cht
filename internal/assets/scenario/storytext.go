package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 分頁劇情文字（`T.TXT`／`WIN.TXT`／`EREGORE.TXT`）。
//
// 這三個檔案一度被判定「在這份執行檔裡沒有載入路徑」（`docs/re/70` §4）——
// 那是錯的（`docs/re/82`）。它們由 `25be:19d1(page, mode)` 載入並顯示：
// `mode` 索引兩張 3 筆表（`ds:0x2f59` 檔名、`ds:0x2f65` 大小），
// `page == 0xffff` 是載入、`page >= 0` 是顯示第 page 頁。
//
// 格式極簡：
//
//	'*'  分頁
//	'\0' 分行（原版逐行印，行首的空白就是縮排）
//
// 頁數與檔案大小都對得上原版的表（見 storytext_test.go）。
const (
	storyPageSep = '*'
	storyLineSep = '\x00'
)

// StoryMode 是 `25be:19d1` 的第二個參數 —— 它選的是**檔案**。
type StoryMode int

const (
	// StoryEregore 是艾瑞戈爾與黑鏡那場戲（11 頁）。
	// 城鎮全成廢墟的事件（`25be:1ae2`）播它的第 9 頁。
	StoryEregore StoryMode = 0
	// StoryWin 是結局（7 頁）。
	StoryWin StoryMode = 1
	// StoryDream 是三場夢（3 頁）。`FUN_1000_0339(n)` 播第 n−1 頁。
	StoryDream StoryMode = 2
)

// storyFiles 是 `ds:0x2f59` 那張 3 筆遠指標表的內容，順序就是 mode。
var storyFiles = map[StoryMode]string{
	StoryEregore: "EREGORE.TXT",
	StoryWin:     "WIN.TXT",
	StoryDream:   "T.TXT",
}

// StoryFileName 回傳 mode 對應的檔名。
func StoryFileName(m StoryMode) (string, bool) {
	n, ok := storyFiles[m]
	return n, ok
}

// StoryText 是一個劇情文字檔解出來的內容。
type StoryText struct {
	// Pages 是每一頁的行。空頁（檔尾 `*` 之後那一段）已經濾掉。
	Pages [][]string
}

// ParseStoryText 解一份分頁劇情文字。
//
// 原版逐行印，所以**行內的前導空白要保留** —— 馬利馮那段預言的縮排
// 是它的排版，trim 掉會變成一坨。只有「整行都是空白」的行會被丟掉。
//
// **只砍檔尾的空頁，中間的空頁保留。** 原版用 `page` 當索引
//（城鎮變廢墟播 `EREGORE.TXT` 第 9 頁），中間少一頁後面全部錯位。
// 檔尾那個空頁是最後一個 `*` 造成的，不是一頁。
func ParseStoryText(data []byte) *StoryText {
	st := &StoryText{}
	for _, raw := range strings.Split(string(data), string(storyPageSep)) {
		var lines []string
		for _, l := range strings.Split(raw, string(storyLineSep)) {
			l = strings.TrimRight(l, " \t\r\n")
			if strings.TrimSpace(l) == "" {
				continue
			}
			lines = append(lines, l)
		}
		st.Pages = append(st.Pages, lines)
	}
	for len(st.Pages) > 0 && len(st.Pages[len(st.Pages)-1]) == 0 {
		st.Pages = st.Pages[:len(st.Pages)-1]
	}
	return st
}

// LoadStoryText 讀某個 mode 對應的檔案。
func LoadStoryText(dir string, m StoryMode) (*StoryText, error) {
	name, ok := StoryFileName(m)
	if !ok {
		return nil, fmt.Errorf("未知的劇情文字 mode %d", m)
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	return ParseStoryText(data), nil
}

// Page 取第 n 頁，超出範圍回 nil。
func (s *StoryText) Page(n int) []string {
	if s == nil || n < 0 || n >= len(s.Pages) {
		return nil
	}
	return s.Pages[n]
}

// WinFatePage 是結局名單那一頁的索引。
//
// 它不是敘述而是**一張表**：十段句子，每段接在角色名後面當他的結局
//（`docs/re/61` §2 那個 `"%s %s"` 的第二個 `%s`）。
const WinFatePage = 6

// WinFateCount 是結局句子的數量。**正好等於職業數（0–9）**，
// 所以索引就是職業編號（`docs/re/19` 的 `classOffset`）。
const WinFateCount = 10

// Fate 取職業 class 的結局句子。超出範圍回空字串。
//
// ⚠ 「索引 ＝ 職業編號」是**由數量相符推出來的判讀**（10 段對 10 個職業，
// 而且內容明顯分別對應騎士／野蠻人／遊俠／司祭／盜賊／學者…）。
// 原版取這一段的程式碼還沒讀到，所以**順序可能整體偏移或另有對應**。
func (s *StoryText) Fate(class int) string {
	page := s.Page(WinFatePage)
	if class < 0 || class >= WinFateCount || len(page) < WinFateCount {
		return ""
	}
	return page[class]
}
