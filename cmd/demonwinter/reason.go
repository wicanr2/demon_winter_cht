package main

import "fmt"

// reasonText 把規則層的穩定原因 key 與參數轉成玩家文字。
//
// internal/game 不得依賴語系資料，也不得回傳中文；所有顯示文案集中在
// assets/lang/zh-Hant/ui.json。空 key 保持空字串，方便既有成功判斷。
func (a *app) reasonText(key string, args ...any) string {
	if key == "" {
		return ""
	}
	text := a.tr.UI(key)
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}
