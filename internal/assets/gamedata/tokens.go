// Package gamedata 解析 Demon's Winter 的規則資料表：
// 怪物屬性表（MONSTER.DAT）、道具屬性表（ITEMS.DAT）、主字串池（FILES.DTT）。
//
// 這三個檔案共用同一種格式假設：純 ASCII 文字，用 0x00（NUL）取代空白/逗號
// 當分隔符，不是原本任務指派時假設的「固定長度二進位記錄」。這個格式假設
// 已用實際檔案驗證過（token 數剛好整除記錄數，見各檔案對應的
// docs/formats/*.md），細節與逐欄位驗證狀態見：
//
//   - docs/formats/game-data-tables.md — MONSTER.DAT／ITEMS.DAT
//   - docs/formats/resource-index.md  — FILES.DTT
//
// Python 參考實作（已驗證可跑）：tools/parse_monster.py、tools/parse_items.py、
// tools/parse_files_index.py。本套件是這三支腳本的邏輯移植，對外只露乾淨的
// Go 結構體與查詢方法，NUL 分隔、ASCII 數字這些格式細節不外洩到 API。
package gamedata

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readNULDelimitedTokens 讀取檔案並依 NUL（0x00）分隔成 token 陣列。
//
// 若檔案本身以 NUL 結尾（split 後最後一個 token 會是空字串），去掉這個尾端
// 空 token——這對應原始檔案「每筆記錄結尾都有 NUL」的寫法，不是真正的資料，
// 三支 Python 參考腳本都有這個處理（`if tokens and tokens[-1] == b"": tokens = tokens[:-1]`）。
// 檔案中段的空字串（例如 FILES.DTT 裡代表「留空欄位」的 122 個空字串）則保留，
// 不在這裡過濾。
func readNULDelimitedTokens(path string) ([][]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gamedata: 讀取 %s 失敗: %w", path, err)
	}
	tokens := bytes.Split(data, []byte{0x00})
	if len(tokens) > 0 && len(tokens[len(tokens)-1]) == 0 {
		tokens = tokens[:len(tokens)-1]
	}
	return tokens, nil
}

// parseASCIIInt 把一個 token 解析成十進位整數（可能帶負號，如 MONSTER.DAT
// 的 attack_type 欄位用負值表示「帶毒」）。
//
// 解析前會先 TrimSpace：真實的 MONSTER.DAT 裡發現一筆資料異常
// （索引 49「Swarm」的 special 欄位，原始內容是 "0 "，多一個尾端空白，
// 已用 hex dump 核對確認是原始檔案本身的內容，不是本套件的 tokenize 邏輯
// 切錯位置）。Python 參考實作用 `int()`，對前後空白本來就寬容，所以
// tools/parse_monster.py 從沒報過這個異常；這裡刻意行為對齊參考實作
// （視為單一資料裡的無害空白，不是格式錯誤），否則這隻怪物會在 Go 版本
// 解析失敗，但 Python 版本正常——兩份「同一份輸入」的解析器行為理應一致。
func parseASCIIInt(field string, token []byte) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(string(token)))
	if err != nil {
		return 0, fmt.Errorf("gamedata: 欄位 %s 的值 %q 不是合法整數: %w", field, token, err)
	}
	return v, nil
}
