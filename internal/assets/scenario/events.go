// Package scenario 解析 Demon's Winter 的劇情資料：事件表（DATA1~5.TXT）與
// 隊伍存檔（PARTY.DAT）。
//
// # 事件表格式修正（重要）
//
// 早期以純資料分析建立的模型（docs/formats/event-script.md 初版）認為每筆記錄
// 是「TEXT + count-prefixed 怪物清單 + 0~3 個數字的 trailer」，trailer 語意未解。
// 這個模型已被反組譯 DEMON.INT 的讀取函式 FUN_25be_0e77（docs/re/02 記錄）推翻：
// 所謂「trailer」根本不是變長欄位，而是兩個固定單值 slot（戰鬥隊形碼、續接碼）
// 加上「下一筆記錄自己的插圖 ID」被線性 tokenize 誤黏在一起造成的假象。
//
// 本檔案依 docs/re/02-data-loading-functions.md §2.2 記載的欄位消耗順序實作，
// 對應原版 FUN_25be_0e77 每次迭代固定讀取的六個欄位槽：
//
//	[A] 插圖 ID     -- local_26 = FUN_2f16_000f(...)
//	[B] TEXT        -- local_1e = 指標（房間敘述文字，不解析數值）
//	[C] count       -- local_2a = FUN_2f16_000f(...)，可帶負號，讀 abs(count) 個怪物 ID
//	[ids...]        -- 依序存進 *(GLOBAL+0x16+i)，MONSTER.DAT 的 0-based 索引
//	[E] 戰鬥設定     -- local_28 = FUN_2f16_000f(...)（doc 舊稱「255 終止符」，其實
//	                   不搜尋 255，是無條件讀取的固定 slot，控制戰鬥隊形走程序生成
//	                   還是預存隊形）
//	[F] 續接碼       -- local_22 = 指標，同時餵給 FUN_2f16_000f 取整數值。值為 3
//	                   會觸發「重繪本欄位文字（跳過開頭那個 '3' 字元）」；開頭是
//	                   '%' 則觸發符文字型繪製（FUN_25be_18fa，句點 '.' 是空白 glyph）
//
// 用獨立 Python 模擬器對 5 個 DATA*.TXT 做過欄位消耗驗證，全部零殘留、零錯位：
// DATA1 221 欄位 35 次迭代、DATA2 82/12、DATA3 152/26、DATA4 118/21、DATA5 59/11。
//
// 對外只露 Event 這個乾淨型別；NUL 分隔、ASCII 數字、欄位消耗順序等格式細節
// 不外洩到 API。
package scenario

import (
	"bytes"
	"fmt"
	"os"
)

// noPictureID 是插圖 ID 欄位的預設值（255 = 0xFF），代表這筆事件沒有插圖。
// 已驗證：docs/re/02-data-loading-functions.md §2.4 [A]，FUN_1d9f_281a 只在
// local_26 != 0xff 時才觸發繪圖／換片音效。
const noPictureID = 255

// chainRedrawValue 是續接碼欄位解析出的整數值等於 3 時觸發的「重繪」語意。
// 已驗證：docs/re/02 §2.4 [F]，對應原版 struct+0xa5 == 3 的檢查。
const chainRedrawValue = 3

// Event 是 DATA1~5.TXT 一筆事件記錄解析後的乾淨表示，對應 FUN_25be_0e77 的
// 一次迴圈迭代。
type Event struct {
	// PictureID 是這筆事件要顯示的插圖 ID（field A）。noPictureID(255) = 無插圖。
	// 已驗證：見 docs/re/02 §2.4 [A]。這同時解開了「每個 DATA*.TXT 檔頭固定 255」
	// 的舊謎題——那不是檔案層級 header，只是第 0 筆記錄的 PictureID 剛好是預設值。
	PictureID int

	// Text 是進入該房間／觸發該事件時顯示的敘述文字（field B）。已驗證：
	// FUN_25be_158e 是 40 字元寬、逐詞斷行、每 5 行暫停翻頁的文字顯示函式，
	// 消耗的正是這個欄位。
	Text string

	// Count 是這筆事件的怪物遭遇組數（field C），已驗證允許帶負號（原版用
	// FUN_206a_000a 做 abs()，迴圈次數取絕對值，但 Count 本身保留原始正負號）。
	// 0 = 無戰鬥的純敘述房間。
	Count int

	// MonsterIDs 依序是 MONSTER.DAT 的怪物索引（0-based），長度 = abs(Count)，
	// 可重複。已驗證：與 MONSTER.DAT 名稱表、docs/walkthrough/part-4.md 劇情場景
	// 交叉核對過（見 docs/formats/event-script.md §2.4 的多筆錨點）。
	MonsterIDs []int

	// CombatSetting 是戰鬥隊形選擇碼（field E，doc 舊稱「255 終止符」）。
	// 已驗證分支存在（FUN_17c5_000d 依此值 < 0x80 走「程序生成隊形」、
	// >= 0x80（即 CombatSetting < 0xff）走「預存隊形」），具體隊形資料內容為
	// 推測、未做動態驗證。noPictureID(255) 常見預設值，代表使用程序生成隊形。
	CombatSetting int

	// Continuation 是續接碼欄位的原始文字內容（field F）。多數記錄是純數字
	// 文字（如 "1"、"255"，此時無特殊語意，只是欄位值本身）；少數記錄是
	// 控制字元開頭的整段文字：
	//   - 開頭 '3'：戰鬥後反應文字（例如 "3With Remondadin dead..."）。
	//     ContinuationValue() 會解析出 3，觸發 IsChainRedraw()。
	//   - 開頭 '%'：符文/密語提示文字（例如 "%YMROS.IS...MINE"）。
	//     IsRuneGlyph() 為 true，句點 '.' 是符文字型的空白 glyph。
	// 已驗證機制：見 docs/re/02 §2.4 [F]。
	Continuation string
}

// ContinuationValue 依原版 FUN_2f16_000f 的語意解析 Continuation 開頭的整數：
// 跳過空白、可選正負號，讀到第一個非數字字元就停止；找不到任何數字時回傳 0。
// 這正是為什麼 "3With Remondadin dead..." 會被解析成 3、"%YMROS..." 會被解析
// 成 0（'%' 不是數字，立刻停手）。
func (e Event) ContinuationValue() int {
	return parseFieldInt(e.Continuation)
}

// IsChainRedraw 回報 Continuation 是否觸發「重繪」語意（ContinuationValue()==3）。
// 已驗證：docs/re/02 §2.4 [F]，對應原版 struct+0xa5==3 的分支，會用
// ChainRedrawText() 的內容重新顯示一次。
func (e Event) IsChainRedraw() bool {
	return e.ContinuationValue() == chainRedrawValue
}

// ChainRedrawText 回傳觸發重繪時應顯示的文字：Continuation 跳過開頭那個控制
// 字元（'3'）之後的內容，對應原版 FUN_25be_158e((char*)local_22 + 1, ...)。
// Continuation 為空字串時回傳空字串。
func (e Event) ChainRedrawText() string {
	if len(e.Continuation) == 0 {
		return ""
	}
	return e.Continuation[1:]
}

// IsRuneGlyph 回報 Continuation 是否是符文/密語提示文字（開頭為 '%'）。
// 已驗證：docs/re/02 §2.4 [F]，對應原版 `if (*local_22=='%')` 呼叫
// FUN_25be_18fa 符文字型繪製函式的分支。
func (e Event) IsRuneGlyph() bool {
	return len(e.Continuation) > 0 && e.Continuation[0] == '%'
}

// RuneText 回傳符文文字跳過開頭 '%' 之後的內容，對應原版 FUN_25be_18fa 的
// param_1+1。已驗證的字碼規則（未在此套件實作字型繪製，只回傳文字本身）：
// 逐字元 '.' 代表空白 glyph（0），其餘字元 glyph 編號 = char - 0x40。
func (e Event) RuneText() string {
	if !e.IsRuneGlyph() {
		return ""
	}
	return e.Continuation[1:]
}

// HasPicture 回報這筆事件是否要顯示插圖（PictureID != noPictureID）。
func (e Event) HasPicture() bool {
	return e.PictureID != noPictureID
}

// parseFieldInt 重現原版 FUN_2f16_000f 的 ASCII→int 語意：跳過前導空白，
// 可選一個正負號，累加十進位數字直到遇到第一個非數字字元為止；完全沒有數字
// 時回傳 0。這是本套件唯一的整數解析入口，欄位 A/C/ids/E/F 全部共用同一套
// 語意（原版也是同一個函式），刻意不用 strconv.Atoi（那需要整段字串都是數字，
// 會在 field F 是 "3With Remondadin dead..." 這類文字時直接報錯，而原版恰恰
// 是靠「解析到第一個非數字就停」才能讓同一個 generic parser 兼職當續接碼
// 讀取器用，見 docs/re/02 §2.2）。
func parseFieldInt(s string) int {
	i := 0
	n := len(s)
	for i < n && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	neg := false
	if i < n && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	start := i
	val := 0
	for i < n && s[i] >= '0' && s[i] <= '9' {
		val = val*10 + int(s[i]-'0')
		i++
	}
	if i == start {
		return 0
	}
	if neg {
		val = -val
	}
	return val
}

// EventTable 是單一 DATA*.TXT 解析後的查詢介面。建立方式一律透過
// LoadEventTable，零值不可用。
type EventTable struct {
	events []Event
}

// LoadEventTable 解析指定路徑的 DATA*.TXT，回傳可查詢的事件表。
//
// 依 docs/re/02 §2.2 記載的固定欄位序列（A/TEXT/C/ids/E/F）逐筆消耗 NUL
// 分隔的欄位，直到整個 token 序列被消耗完。若某次迭代讀到一半就沒有欄位可讀
// （token 數不是這個 grammar 的整數倍），視為格式不符，回傳 error 而非 panic
// 或悄悄捨棄殘留資料。
func LoadEventTable(path string) (*EventTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("scenario: 讀取 %s 失敗: %w", path, err)
	}

	tokens := tokenizeNUL(data)

	var events []Event
	pos := 0
	n := len(tokens)
	for pos < n {
		start := pos

		// [A] 插圖 ID
		if pos >= n {
			return nil, incompleteRecordErr(path, start, "picture ID")
		}
		pictureID := parseFieldInt(tokens[pos])
		pos++

		// [B] TEXT
		if pos >= n {
			return nil, incompleteRecordErr(path, start, "text")
		}
		text := tokens[pos]
		pos++

		// [C] count（可帶負號）
		if pos >= n {
			return nil, incompleteRecordErr(path, start, "count")
		}
		count := parseFieldInt(tokens[pos])
		pos++

		// [ids...] 依 abs(count) 個怪物 ID
		var ids []int
		if count != 0 {
			want := count
			if want < 0 {
				want = -want
			}
			ids = make([]int, 0, want)
			for i := 0; i < want; i++ {
				if pos >= n {
					return nil, incompleteRecordErr(path, start, "monster id")
				}
				ids = append(ids, parseFieldInt(tokens[pos]))
				pos++
			}
		}

		// [E] 戰鬥設定
		if pos >= n {
			return nil, incompleteRecordErr(path, start, "combat setting")
		}
		combatSetting := parseFieldInt(tokens[pos])
		pos++

		// [F] 續接碼（同時是原始文字與整數值的共用 slot）
		if pos >= n {
			return nil, incompleteRecordErr(path, start, "continuation")
		}
		continuation := tokens[pos]
		pos++

		events = append(events, Event{
			PictureID:     pictureID,
			Text:          text,
			Count:         count,
			MonsterIDs:    ids,
			CombatSetting: combatSetting,
			Continuation:  continuation,
		})
	}

	return &EventTable{events: events}, nil
}

func incompleteRecordErr(path string, tokenIndex int, field string) error {
	return fmt.Errorf(
		"scenario: %s 在第 %d 個欄位之後的記錄不完整，讀到 %s 欄位時已無剩餘 token"+
			"（欄位消耗順序見 docs/re/02-data-loading-functions.md §2.2）",
		path, tokenIndex, field,
	)
}

// Len 回傳事件記錄總數。
func (t *EventTable) Len() int { return len(t.events) }

// All 回傳全部事件的唯讀複本，順序與檔案內原始順序一致。
func (t *EventTable) All() []Event {
	out := make([]Event, len(t.events))
	copy(out, t.events)
	return out
}

// ByIndex 依索引（0-based）取得事件記錄。
func (t *EventTable) ByIndex(i int) (Event, error) {
	if i < 0 || i >= len(t.events) {
		return Event{}, fmt.Errorf("scenario: 事件索引 %d 超出範圍 [0,%d)", i, len(t.events))
	}
	return t.events[i], nil
}

// tokenizeNUL 依 NUL（0x00）分隔欄位。若檔案結尾是 NUL（split 後最後一個
// token 會是空字串），去掉這個尾端空 token——這對應原始檔案「每筆記錄結尾
// 都有 NUL」的寫法，不是真正的資料（沿用 tools/parse_datatxt.py 的處理方式）。
func tokenizeNUL(data []byte) []string {
	parts := bytes.Split(data, []byte{0x00})
	if len(parts) > 0 && len(parts[len(parts)-1]) == 0 {
		parts = parts[:len(parts)-1]
	}
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = string(p)
	}
	return out
}

// 符文字型的字碼規則（`FUN_25be_18fa`，`docs/re/02` §2.4、`docs/re/72`）。
const (
	// runeGlyphBase 是字元轉 glyph 編號的減量：`'A'`(0x41) → 1、`'Z'`(0x5a) → 26。
	runeGlyphBase = 0x40
	// runeBlankChar 是「空白 glyph」的專用替代字元。
	//
	// 原版不用 ASCII 空白（0x20）—— 那拿去算 `char − 0x40` 會變成負數、
	// 撞到別的合法 glyph 編號。所以密語文字裡的空格全部寫成 `.`。
	runeBlankChar = '.'
	// RuneGridCols 是原版排版的欄數（`local_6%9+1, local_6/9+1`）。
	RuneGridCols = 9
	// RuneGlyphCount 是字型的 glyph 數：0 空白 + 1–26 字母。
	// `CYPHER.SHP` 是 1728 bytes ÷ 64 ＝ 27 個 frame，正好對上。
	RuneGlyphCount = 27
)

// RuneGlyphs 把符文文字轉成 glyph 索引序列。
//
// 超出 0–26 的字元回傳 -1（呼叫端該畫成空白或略過）——
// 原版沒有防呆，但資料裡只有大寫字母與 `.`，所以那條路走不到；
// 這裡不靜默改成 0，免得把「資料異常」偽裝成「空白」。
func RuneGlyphs(text string) []int {
	out := make([]int, 0, len(text))
	for _, ch := range text {
		switch {
		case ch == runeBlankChar:
			out = append(out, 0)
		case ch >= 'A' && ch <= 'Z':
			out = append(out, int(ch)-runeGlyphBase)
		default:
			out = append(out, -1)
		}
	}
	return out
}
