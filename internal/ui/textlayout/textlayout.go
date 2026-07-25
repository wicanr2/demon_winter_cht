// Package textlayout 是文字排版的純邏輯：斷行、分頁、翻頁狀態。
//
// 刻意不依賴 Ebiten —— Ebiten 在 init 期就要求顯示器，混在一起會讓
// 這些純函式在無頭環境下連測都測不了。繪製由 ui 套件負責。
package textlayout

import "strings"

// 原版的事件文字顯示：40 字元寬、逐詞斷行、每 5 行暫停翻頁。
// 對應 FUN_25be_158e，見 docs/re/02-data-loading-functions.md。
const (
	Columns   = 40
	PageLines = 5
)

// WrapText 把一段文字依 columns 逐詞斷行。
//
// 超過一行寬度的單字不切開，讓它自己佔一行 —— 原版也是逐詞處理，
// 沒有連字符斷字邏輯。換行字元視為斷詞空白。
func WrapText(s string, columns int) []string {
	if columns <= 0 {
		return nil
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) <= columns {
			cur += " " + w
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	return append(lines, cur)
}

// Paginate 把已斷行的文字切成每頁 perPage 行。
func Paginate(lines []string, perPage int) [][]string {
	if perPage <= 0 || len(lines) == 0 {
		return nil
	}
	var pages [][]string
	for i := 0; i < len(lines); i += perPage {
		end := i + perPage
		if end > len(lines) {
			end = len(lines)
		}
		pages = append(pages, lines[i:end])
	}
	return pages
}

// TextBox 是一個可翻頁的文字視窗。
//
// 只管「顯示哪一頁、還有沒有下一頁」，不處理輸入 —— 由呼叫端決定用哪個鍵翻頁。
type TextBox struct {
	pages [][]string
	page  int
}

// NewTextBox 建立一個顯示 s 的文字視窗，依原版規格斷行分頁。
func NewTextBox(s string) *TextBox {
	return &TextBox{pages: Paginate(WrapText(s, Columns), PageLines)}
}

// Active 回報是否還有內容要顯示。
func (b *TextBox) Active() bool { return b != nil && b.page < len(b.pages) }

// HasMore 回報翻頁後是否還有下一頁。
func (b *TextBox) HasMore() bool { return b != nil && b.page+1 < len(b.pages) }

// Lines 回傳目前這一頁的行。非 Active 時回傳 nil。
func (b *TextBox) Lines() []string {
	if !b.Active() {
		return nil
	}
	return b.pages[b.page]
}

// Advance 翻到下一頁。已經是最後一頁時，視窗變成非 Active。
func (b *TextBox) Advance() {
	if b != nil {
		b.page++
	}
}
