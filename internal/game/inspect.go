package game

import "github.com/wicanr2/demon_winter_cht/internal/assets/scenario"

// 探查周圍（`I`）——「物品」那六個指令的最後一個（`docs/re/96`）
//
// **它不是一個動作。** 主選單的其他指令都會從 `FUN_222f_0b0e` 回傳一個動作碼，
// 交給那張 21 格的分派表；`I` 走的是 switch case 5，就地把 `ds:0x5c66` 設成
// `0x80` 就結束了：
//
//	222f:1274  MOV word ptr [0x5c66],0x80
//	222f:127a  MOV word ptr [BP-0x12],0xfffe   ; 「重畫一次」
//	222f:127f  JMP 迴圈尾
//
// 旗標由**地圖繪製常式**（`FUN_222f_1404`，`222f:14c7`）讀。成立時它在畫每一格
// 之前掃 `ITEMLOCB.DAT`，比對 `(X, Y, 子地圖)`，命中就把該格改畫成 tile `0x38`
// ——「地城道具圖示」。畫完主迴圈在 `222f:0bf4` 把旗標清回 0，所以它只維持一次重畫。
//
// 手冊 part-3 §55 獨立佐證：「按 I 會自動掃描視野範圍內所有可見空間，找出地下城
// 物品……在有物品的位置顯示地下城物品圖示」。沒有技能門檻、沒有每日次數。
//
// **與原版的一處差異**：原版那個掃描迴圈以「X 欄 == 0xff」當終止符，而出貨的
// `ITEMLOCB.DAT` 第 50 筆之後是沒清乾淨的殘留（第 50 筆 X ＝ `0xef`），所以原版
// 會掃出 50 筆有效記錄之外去。這裡照 `ItemLocRecordCount` 掃固定 50 筆 ——
// 出貨資料的前 50 筆沒有任何一筆 X 是 `0xff`，兩者在真實資料上結果相同，
// 差別只在原版那段越界讀取（那是原版的瑕疵，不是可觀察的遊戲行為）。

// InspectSpot 是探查標出來的一格。
type InspectSpot struct {
	X, Y byte
}

// InspectSurroundings 列出目前子地圖上所有放著地城道具的格子。
//
// 回傳的是**整張子地圖**的命中格，不是視野內的 —— 原版是「畫每一格時順便查」，
// 等價於「查完再由呈現層裁到視野」，而後者不必把版面常數帶進規則層。
//
// 不去重：出貨資料裡有兩對座標相同的記錄（`(0x0d,0x1a)` 與 `(0x33,0x19)`），
// 原版就是逐筆命中逐筆蓋同一個 tile，重複蓋沒有可觀察的差別。
func InspectSurroundings(t *scenario.ItemLocTable, mapID byte) []InspectSpot {
	if t == nil || mapID == scenario.ItemLocTaken {
		return nil
	}
	var out []InspectSpot
	for _, r := range t.Records {
		// 拿走的記錄子地圖被清成 0，自然不會命中任何真實子地圖。
		if r.MapID != mapID {
			continue
		}
		out = append(out, InspectSpot{X: r.X, Y: r.Y})
	}
	return out
}
