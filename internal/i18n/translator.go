package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Translator 是執行期的查表器。
//
// 查不到就回原文 —— 缺譯在畫面上是英文，看得見；
// 回空字串會變成「敘述消失」，那種缺陷玩到才發現。
type Translator struct {
	// byIndex 的 key 是「來源檔名 + 索引」。
	byIndex map[string]string

	// byName 是介面文案的查表（語意化 key，見 Entry.Name）。
	byName map[string]string

	// mismatched 記錄原文對不上的條目，代表翻譯檔與資料脫節。
	mismatched []Mismatch
}

// Mismatch 是一條原文對不上的記錄。
type Mismatch struct {
	Source string
	Index  int
}

func key(source string, index int) string {
	return fmt.Sprintf("%s#%d", strings.ToUpper(source), index)
}

// Load 讀入一個語言目錄下所有翻譯檔。
//
// 目錄不存在時回傳一個空的 Translator 而不是錯誤：沒有翻譯檔就是全英文，
// 那是合法狀態（例如原文模式），不該讓遊戲起不來。
func Load(dir string) (*Translator, error) {
	t := &Translator{byIndex: map[string]string{}, byName: map[string]string{}}

	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return nil, err
	}

	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		c, err := LoadCatalog(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if c.Source == "" {
			return nil, fmt.Errorf("i18n: %s 缺少 %s 檔頭，不知道對應哪個來源檔",
				e.Name(), strings.TrimSpace(markSource))
		}
		for _, ent := range c.Entries {
			if !ent.Translated() {
				continue
			}
			if ent.Name != "" {
				t.byName[ent.Name] = ent.Target
				continue
			}
			t.byIndex[key(c.Source, ent.Index)] = ent.Target
		}
	}
	return t, nil
}

// Event 回傳一條事件敘述的譯文；沒有譯文時回傳原文。
//
// 同時比對原文：對不上代表翻譯檔與現在的資料脫節，
// 這時**用原文**並記進 Mismatched()，不拿可能錯位的譯文上畫面。
func (t *Translator) Event(source string, index int, original string) string {
	got, ok := t.byIndex[key(source, index)]
	if !ok {
		return original
	}
	return got
}

// UI 回傳一條介面文案的譯文；沒有譯文就回 fallback。
//
// fallback 一律傳**中文**而不是英文 —— 這個專案的介面本來就是中文寫的，
// 翻譯檔的作用是「把它抽出來好維護、日後好換語言」，不是「英文轉中文」。
// 所以缺一條的後果是「還是中文，只是沒走目錄」，不是畫面變英文。
func (t *Translator) UI(name, fallback string) string {
	if got, ok := t.byName[name]; ok && got != "" {
		return got
	}
	return fallback
}

// UICount 回報載入了幾條介面文案。給啟動時的自檢用。
func (t *Translator) UICount() int { return len(t.byName) }

// Verify 逐條比對翻譯檔的原文與現在的資料，回報對不上的條目。
//
// 呼叫端應在載入資料後跑一次。**不要跳過** —— 索引錯位的譯文
// 每一句都通順、每一句都接錯地方，肉眼很難發現。
func (t *Translator) Verify(dir, source string, texts []string) error {
	c, err := LoadCatalog(filepath.Join(dir, CatalogFileName(source)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range c.Entries {
		if e.Index < 0 || e.Index >= len(texts) {
			t.mismatched = append(t.mismatched, Mismatch{Source: source, Index: e.Index})
			delete(t.byIndex, key(source, e.Index))
			continue
		}
		if normalise(e.Source) != normalise(texts[e.Index]) {
			t.mismatched = append(t.mismatched, Mismatch{Source: source, Index: e.Index})
			delete(t.byIndex, key(source, e.Index))
		}
	}
	return nil
}

// normalise 抹掉只影響排版的差異：原版事件文字的換行與多重空白
// 在斷行時本來就會被重排，不該因此判定原文變動。
func normalise(s string) string { return strings.Join(strings.Fields(s), " ") }

// Mismatched 回傳原文對不上的條目。非空代表翻譯檔需要重新抽取。
func (t *Translator) Mismatched() []Mismatch { return t.mismatched }

// Count 回傳已載入的譯文條數。
func (t *Translator) Count() int { return len(t.byIndex) }
