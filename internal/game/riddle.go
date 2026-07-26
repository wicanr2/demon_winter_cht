package game

import "strings"

// 兩道密語謎題（`docs/re/65` case 10／11，逐指令見 `docs/re/84`）。
//
// **答案是明文存在資料段裡的** —— `VOID` 在 `ds:0x2c7b`、`JESRIC` 在 `ds:0x2d08`，
// 玩家要打的字串就是它們。那個年代的常態：沒有雜湊、沒有混淆，
// 因為當年沒有人會去反組譯一片 5.25 吋磁片來作弊，會的人也早就買了攻略。
//
// 這兩題是全遊戲僅有的自由文字輸入。其餘互動都是選單。

// Riddle 是一道密語謎題。
type Riddle struct {
	// Prompt 是問題的三行原文（原版逐行 `1361()` 印出來）。
	Prompt []string
	// Answer 是正解。**比對不分大小寫**（原版走 `stricmp`）。
	Answer string
	// MaxLen 是輸入欄的長度上限（原版傳給讀字串函式的參數）。
	MaxLen int
	// Right／Wrong 是答對／答錯時要顯示的訊息。
	Right []string
	Wrong []string
}

// 兩道謎題的 case 編號（`nSS.DAT` 類別 5 的值）。
const (
	// RiddleCaseSpectralPriest 是幽靈司祭那一題，在地圖 5 的 (11,20)。
	RiddleCaseSpectralPriest = 10
	// RiddleCaseTempleName 是馬利馮神殿的門房那一題，在地圖 4 的 (59,39)。
	RiddleCaseTempleName = 11
)

// SpectralPriestDoor 是答對之後被打開的那一格。
//
// 原版寫死線性索引 `0x48b`（`0x1a375`），換算 `y*64 + x` ＝ **(11,18)**——
// 正好在觸發格 (11,20) 正北兩格。原本那一格是 tile `0x0d`，
// 可通行表的值是 `0xff`（牆）；改成 `0` 之後表值變 4，走得過去。
//
// 三份互相獨立的產物（反組譯的常數、`MAP5.MAP` 的實際 tile、可通行表）
// 對到同一個結論，所以這條不是推測。
var SpectralPriestDoor = struct{ X, Y int }{11, 18}

// SpectralPriestDoorOpen 是打開之後寫進去的 tile 值。
const SpectralPriestDoorOpen = 0

// TempleRejectY 是答錯神殿那一題時隊伍被推回去的 Y 座標
//（原版 `party+0xa2 = 0x26`）。觸發格在 Y=39，推回 38 ——
// 就是把你趕出門一步，不是傳送。
const TempleRejectY = 0x26

// RiddleFor 回傳某個 case 的謎題，沒有就回 nil。
func RiddleFor(plotCase int) *Riddle {
	switch plotCase {
	case RiddleCaseSpectralPriest:
		return &Riddle{
			Prompt: []string{
				"A spectral priest utters a chant:",
				"'Power, Divinity, Spirit...' and",
				"awaits the final word of the spell",
			},
			Answer: "VOID",
			MaxLen: 0x11,
			// 答對只有音效與開牆，原版沒有任何文字 —— 保持沉默。
			Right: nil,
			Wrong: []string{"The priest fades away into mist"},
		}
	case RiddleCaseTempleName:
		return &Riddle{
			Prompt: []string{
				"A voice from nowhere speaks: 'Only",
				"those who worship Malifon may enter",
				"this temple. What is thy name?'",
			},
			Answer: "JESRIC",
			MaxLen: 7,
			Right:  []string{"You may enter."},
			Wrong: []string{
				"You are not known to this temple.",
				" Leave at once.",
			},
		}
	}
	return nil
}

// Correct 判斷輸入是否正確。原版用 `stricmp`，所以不分大小寫；
// 前後空白也吃掉 —— 原版的讀字串函式不會留下它們。
func (r *Riddle) Correct(input string) bool {
	return strings.EqualFold(strings.TrimSpace(input), r.Answer)
}
