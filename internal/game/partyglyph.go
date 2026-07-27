package game

// 地圖上的隊伍圖示（`docs/re/97`）
//
// 原版不是用方框標隊伍位置，而是把**一個地形圖塊的索引**寫進繪製緩衝區
// 的隊伍那一格，跟地形一起畫出來（`222f:0bbc`）。所以 `DEMON.SHE`／`.SHP`
// 那 102 格裡有一部分不是地形，是角色與船的 glyph。
//
//	222f:0b85  CMP word ptr [0x52d4],0   ; 在船上？
//	222f:0b8c  facing & 1 == 1 → [bp-2] = X（0x50f0）
//	222f:0b9f                  → [bp-2] = Y（0x50ee）
//	222f:0ba5  MOV BX,[0x4e32]           ; 面向
//	222f:0ba9  MOV AL,[BX+0x210f]        ; 走路 glyph 基底
//	222f:0bb6  ADD AX, [bp-2] & 1        ; 兩格動畫
//	222f:0bc1  （在船上）AX = 面向 + 0x3f
//
// 動畫用的是**行進軸上的座標奇偶**：東西向看 X、南北向看 Y。
// 走一步座標變一格，glyph 就換一次 —— 不需要計時器，走路自然有兩格動畫。

// partyGlyphBase 是 `ds:0x210f` 那四個 byte（依面向：北／東／南／西）。
//
// 原始位元組：`1e 1b 18 21`（`DEMON.INT` 檔位移 0x26c0f）。
var partyGlyphBase = [4]byte{30, 27, 24, 33}

// shipGlyphBase 是搭船時的 glyph 基底（原版 `facing + 0x3f`）。
const shipGlyphBase = 0x3f

// PartyGlyph 回傳這一刻該畫在隊伍那一格的圖塊索引。
//
// facing 用 game.Facing 的 0–3；x／y 是隊伍座標；sailing 是搭船狀態。
func PartyGlyph(facing Facing, x, y int, sailing bool) byte {
	f := int(facing) & 3
	if sailing {
		return byte(shipGlyphBase + f)
	}
	// 面向東西（奇數）看 X，南北（偶數）看 Y。
	axis := y
	if f&1 == 1 {
		axis = x
	}
	return partyGlyphBase[f] + byte(axis&1)
}
