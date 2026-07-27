package game

import (
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 觀室 View Room（技能 27，原版動作 `0x0f` ＝ `222f:30dc` ＝ `0x18fcc`）
//
// 手冊：「行走時按 `V` 可使用這項靈視者技能，讓具備此技能的角色**透過房門
// 看到隔壁房間的情況**。」以及「用觀室技巧也能看到陷阱，**但不會標記為
// 『已注意』**。」
//
// 兩句話在機器碼裡都對得上：往正前方最多看三格，命中的特殊格用
// **peek 模式**交給事件消費者（`25be:0263` 帶 `param != 0`）——
// 陷阱那一支在 peek 模式只印 `A trap!` 就返回，不擲、不改寫、不扣血。

// SkillViewRoom 是觀室（`docs/re/21` §1 的 id 27，角色記錄 `+0xe3`）。
const SkillViewRoom gamedata.SkillID = 27

const (
	// ViewRoomRange 是往前看幾格（`0x190f3` 的 `cmp [bp-2],3`）。
	ViewRoomRange = 3
	// PsychicUsesPerDay 是靈視技能的每日次數（`0x19027` 的 `cmp …,3`）。
	//
	// ⚠ **手冊寫「每天只能使用一次」，機器碼是 3。** 手冊講的是 Apple II 版，
	// 而這裡移植的是 DOS 版的執行檔。差異記在 `docs/re/93` §4，
	// 待 DOSBox 實跑複核。
	PsychicUsesPerDay = 3
)

// ViewRoomResult 是一次觀室的結果。
type ViewRoomResult struct {
	// NoSkill 代表隊伍裡沒有人會觀室。原版**什麼訊息都不印**就返回。
	NoSkill bool
	// Exhausted 代表今天的三次用完了（原版印 `Your psychic powers are weak`）。
	Exhausted bool
	// Hit 是看到的那一格；nil 代表三格內什麼都沒有（原版印 `You see nothing`）。
	Hit *scenario.SpecialHit
	// X, Y 是命中的座標。
	X, Y int
}

// ViewRoom 往正前方窺看。
//
// uses 指向 trailer `+0xad` 的每日計數（`scenario.ViewRoomUses`），
// **不管有沒有看到東西都會 +1** —— 原版的 `inc` 排在掃描之前（`0x19059`）。
func ViewRoom(party []Character, st *scenario.SpecialTiles, tiles TileSource,
	x, y int, f Facing, uses *byte) ViewRoomResult {

	if !partyHasViewRoom(party) {
		return ViewRoomResult{NoSkill: true}
	}
	if uses == nil || *uses >= PsychicUsesPerDay {
		return ViewRoomResult{Exhausted: true}
	}
	*uses++

	if st == nil || tiles == nil {
		return ViewRoomResult{}
	}
	dx, dy := f.Delta()
	for step := 0; step < ViewRoomRange; step++ {
		x, y = x+dx, y+dy
		// 原版明寫四個邊界值（`-1` 與 `0x40`）各比一次，不是範圍檢查。
		if x < 0 || y < 0 || x >= MapWidth || y >= MapHeight {
			break
		}
		t, err := tiles.TileAt(x, y)
		if err != nil {
			break
		}
		// 只有 tile `0x11` 才查表 —— 與走路觸發同一道閘門。
		//
		// ⚠ 原版這裡比的是**沒遮罩的**原始 byte（`cmp es:[bx+si],0x11`，
		// `0x190b3`），別處都先 `& 0x7f`。差別只在最高位被設起來的格子，
		// 那種格子在走路時會觸發、在觀室時看不到。**這裡照別處遮罩** ——
		// 不模擬那個不一致，因為它更像疏漏而不是設計。記在 `docs/re/93` §3。
		if t&0x7f != tileEventGateA {
			continue
		}
		hit := st.Lookup(byte(x), byte(y))
		if hit == nil {
			continue
		}
		// 類別 0（預設文字）與類別 4（傳送）都跳過 ——
		// 原版 `cmp ds:0x5c62,0 / je` 與 `cmp ax,4 / je` 兩道。
		if cls := hit.Tile.Class(); cls == 0 || cls == scenario.SpecialClassTeleport {
			continue
		}
		return ViewRoomResult{Hit: hit, X: x, Y: y}
	}
	return ViewRoomResult{}
}

// partyHasViewRoom 走 partyHasSkill（`viewitem.go`）—— 兩個靈視技能
// 共用同一份掃隊迴圈，不要各自抄一份「死人不算」的判斷。
func partyHasViewRoom(party []Character) bool {
	return partyHasSkill(party, SkillViewRoom)
}

// ResetPsychicUses 是睡覺把兩個靈視計數清 0（原版 `0x1ef68`–`0x1ef7c`
// 那三行，與觀地的 `+0xac` 同一段）。
func ResetPsychicUses(s *scenario.SaveGame) {
	if s == nil {
		return
	}
	s.ViewRoomUses = 0
	s.ViewItemUses = 0
}
