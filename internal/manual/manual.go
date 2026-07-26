// Package manual 是遊戲內手札。
//
// 1988 年的遊戲把種族、職業、神祇、地形、陷阱這些查詢用的資料放在紙本手冊裡
// —— 一方面是當年的慣例，一方面也兼作防拷。三十幾年後那本手冊多半已經不在
// 玩家手上，所以本專案把它搬進遊戲：**要查的東西按一個鍵就看得到，
// 不必去翻 PDF。**
//
// 內容來源是 `docs/manual-cht/`（1990 年《軟體世界》中文說明書的轉錄），
// 重新排版成適合 640×400 畫面的寬度後放在 `assets/manual/<lang>/manual.txt`。
//
// 格式刻意簡單，讓翻譯與增修都不必動程式：
//
//	== 章節標題
//	內文一行
//	內文一行
//
//	== 下一章
//
// 空行保留（它是段落間隔），行首空白保留（表格對齊靠它）。
package manual

import (
	"bufio"
	"bytes"
	"os"
	"strings"
)

// sectionPrefix 是章節標題的前綴。
const sectionPrefix = "== "

// Section 是一個章節。
type Section struct {
	Title string
	Lines []string
}

// Manual 是整本手札。
type Manual struct {
	Sections []Section
}

// Parse 解析手札內容。
//
// 章節前的內容（如果有）會被忽略 —— 沒有標題的文字在畫面上無處可放。
func Parse(data []byte) *Manual {
	m := &Manual{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t\r")
		if title, ok := strings.CutPrefix(line, sectionPrefix); ok {
			m.Sections = append(m.Sections, Section{Title: strings.TrimSpace(title)})
			continue
		}
		if len(m.Sections) == 0 {
			continue
		}
		cur := &m.Sections[len(m.Sections)-1]
		cur.Lines = append(cur.Lines, line)
	}
	// 章末的空行在畫面上只是多出一段捲動距離。
	for i := range m.Sections {
		s := &m.Sections[i]
		for len(s.Lines) > 0 && strings.TrimSpace(s.Lines[len(s.Lines)-1]) == "" {
			s.Lines = s.Lines[:len(s.Lines)-1]
		}
	}
	return m
}

// Load 讀取指定語言的手札。
//
// **檔案不存在不算錯誤** —— 回傳空手札，遊戲照常跑，只是翻不到東西。
// 手札是輔助內容，缺了不該擋住玩遊戲。
func Load(path string) (*Manual, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manual{}, nil
		}
		return nil, err
	}
	return Parse(data), nil
}

// Len 是章節數。
func (m *Manual) Len() int {
	if m == nil {
		return 0
	}
	return len(m.Sections)
}

// Titles 回傳所有章節標題，供目錄畫面用。
func (m *Manual) Titles() []string {
	if m == nil {
		return nil
	}
	out := make([]string, 0, len(m.Sections))
	for _, s := range m.Sections {
		out = append(out, s.Title)
	}
	return out
}

// At 取第 i 章，超出範圍回 nil。
func (m *Manual) At(i int) *Section {
	if m == nil || i < 0 || i >= len(m.Sections) {
		return nil
	}
	return &m.Sections[i]
}
