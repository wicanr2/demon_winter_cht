package textlayout

// 中英混排的排版格。
//
// **字模尺寸與排版格解耦**：斷行與游標推進一律用格寬，
// 換字模不破版、不疊字；字模在格內置中由繪製端處理。
//
// 兩種字都是 16×16 格。中文用倚天 16×15 原生點陣直繪；
// 英文是原版 CGA 8×8 均勻放大兩倍 —— 與圖塊同一個放大倍率，
// 一個 ASCII 像素是 2×2、一個中文像素是 1×1，英文和美術對齊、中文銳利。
//
// **英文不是半形。** 8 像素格配 16 像素寬的字模會疊字；
// 要半形就得把字模橫向不放大、縱向放大兩倍，那會讓原版字形變形。
// 依 rulebook/81「底圖整數倍放大、CJK 原生直繪」，這裡選不變形。
const (
	CellHeight   = 16
	CellWidthCJK = 16
	CellWidthASC = 16
)

// MixedColumns 是中文化後文字區的像素寬。
//
// 畫布 640 扣掉左右各 16 的邊界 = 608，一行 38 格。
// 原版事件文字是 40 欄，中文化後重新斷行，不沿用原欄數。
const MixedColumns = 608

// CellWidth 回傳一個字元佔的排版格寬。
func CellWidth(ch rune) int {
	if ch < 0x80 {
		return CellWidthASC
	}
	return CellWidthCJK
}

// TextWidth 回傳一段字串排出來的像素寬。
func TextWidth(s string) int {
	w := 0
	for _, ch := range s {
		w += CellWidth(ch)
	}
	return w
}

// WrapMixed 依像素寬度斷行，中英混排通用。
//
// 中文逐字可斷、英文以空白斷詞 —— 這是中文排版與英文的關鍵差異。
// 直接沿用 WrapText 的逐詞斷行會讓整段中文變成一個「詞」而擠成一行。
func WrapMixed(s string, pixelWidth int) []string {
	if pixelWidth <= 0 || s == "" {
		return nil
	}

	var lines []string
	var cur []rune
	curW := 0

	flush := func() {
		lines = append(lines, string(cur))
		cur, curW = nil, 0
	}

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\n' {
			flush()
			continue
		}

		// ASCII 單字整組處理，不從中間切開。
		if ch < 0x80 && ch != ' ' {
			j := i
			wordW := 0
			for j < len(runes) && runes[j] < 0x80 && runes[j] != ' ' && runes[j] != '\n' {
				wordW += CellWidthASC
				j++
			}
			// 單字本身就超過一行寬時不再退讓，讓它自己佔一行（同 WrapText）。
			if curW+wordW > pixelWidth && curW > 0 {
				flush()
			}
			cur = append(cur, runes[i:j]...)
			curW += wordW
			i = j - 1
			continue
		}

		w := CellWidth(ch)
		if curW+w > pixelWidth {
			flush()
			// 行首不留空白。
			if ch == ' ' {
				continue
			}
		}
		cur = append(cur, ch)
		curW += w
	}
	if len(cur) > 0 {
		flush()
	}
	return lines
}

// NewMixedTextBox 建立一個中英混排的可翻頁文字視窗。
func NewMixedTextBox(s string, pixelWidth int) *TextBox {
	return &TextBox{pages: Paginate(WrapMixed(s, pixelWidth), PageLines)}
}
