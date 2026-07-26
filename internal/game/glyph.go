package game

// 主線：三個緋紅符印、光之環、禁錮（`docs/re/59`–`61`）
//
// 這是原版唯一的破關路徑：
//
//	① 走到三塊狹長陸地盡頭的符印上（tile 0x63），施 UNCURSE 解除
//	② 三個都解完，crimson forcefield 才會放行進入光之環
//	③ 在光之環裡施 IMPRISON，禁錮惡魔 → 結局
//
// UNCURSE 與 IMPRISON **不在 `FILES.DTT` 的 43 筆法術表裡**，
// 它們是通用施法選單在特定條件下換上的另一組選項（熱鍵 U／I）。

const (
	// GlyphTile 是符印所在的地圖圖塊。站在它上面 UNCURSE 才有效
	// （`0x6b10  cmp es:[bx+si],0x63`）。
	GlyphTile = 0x63

	// GlyphSubMapBase 是第一個符印所在的子地圖編號。
	// 索引 ＝ 子地圖 − 55，其中子地圖 66 特判成 2（`0x6b3c`）。
	GlyphSubMapBase = 55
	// glyphSubMapThird 是第三個符印的子地圖（66，即 `0x42`）。
	glyphSubMapThird = 66
	// glyphThirdIndex 是它特判後的索引。
	glyphThirdIndex = 2

	// GlyphDone 是解除後寫進旗標的值（`0x6bb8  mov ...,0x80`）。
	// 用 0x80 而不是 1 —— 作者把低 7 bit 留著沒用。
	GlyphDone = 0x80

	// UncurseCost 是解除一個符印的法力（`0x6b8e  ax -= 50`）。
	UncurseCost = 50
	// ImprisonCost 是禁錮的法力（`0x6bfb  ax -= 100`）。
	ImprisonCost = 100

	// ImprisonSubMap／ImprisonMaxY 是禁錮成功的地點條件（`0x6c07`、`0x6c0f`）。
	// **或的關係，不是且。** 子地圖 5 小於 10，依 `docs/re/24` 那是地城。
	ImprisonSubMap = 5
	ImprisonMaxY   = 6
)

// GlyphIndexFor 把子地圖編號換成符印索引，不是符印所在的子地圖回 -1。
//
// 原版是 `索引 = 子地圖 − 55`，再對 11 做一次特判 ——
// 也就是說 55／56 連號，第三個跳到 66。
func GlyphIndexFor(subMap int) int {
	idx := subMap - GlyphSubMapBase
	if subMap == glyphSubMapThird {
		return glyphThirdIndex
	}
	if idx < 0 || idx > 1 {
		return -1
	}
	return idx
}

// GlyphResult 是一次 UNCURSE 的結果。
type GlyphResult int

const (
	// GlyphNoGlyph：腳下不是符印圖塊（"I see no glyph!"）。
	GlyphNoGlyph GlyphResult = iota
	// GlyphAlreadyDone：這個符印已經解過（"It is already inactive"）。
	GlyphAlreadyDone
	// GlyphNotEnoughSP：法力不足（"That requires 50 SP!"）。
	GlyphNotEnoughSP
	// GlyphDestroyed：解除成功。
	GlyphDestroyed
)

// Uncurse 解除腳下的符印，回傳結果並在成功時改寫 flags 與施法者法力。
//
// 判定順序照原版：圖塊 → 旗標 → 法力 → 扣費 → 寫旗標。
// **法力檢查在扣費之前**，所以 UNCURSE 不會像 IMPRISON 那樣白扣
// （原版的 "That requires 50 SP!" 在 `0x6aef`，早於 `0x6b8e` 的扣費）。
func Uncurse(caster *Character, tile byte, subMap int, flags *[3]byte) GlyphResult {
	if tile != GlyphTile {
		return GlyphNoGlyph
	}
	idx := GlyphIndexFor(subMap)
	if idx < 0 {
		return GlyphNoGlyph
	}
	if flags[idx] != 0 {
		return GlyphAlreadyDone
	}
	if caster == nil || caster.CurrentSP < UncurseCost {
		return GlyphNotEnoughSP
	}
	caster.CurrentSP -= UncurseCost
	flags[idx] = GlyphDone
	return GlyphDestroyed
}

// CircleOfLightOpen 回報光之環的力場是否已經散去。
//
// 原版比的是 `!= 0`（`0x1a569`–`0x1a57f`），不是 `>= 0x80` ——
// 而 `222f:0b0e` 的符印傷害判定比的卻是 `< 0x80`。兩處門檻不同，
// 但中間值在原版資料裡不存在，實務上等價（`docs/re/59` §3）。
// 這裡照擋門那一處寫。
func CircleOfLightOpen(flags [3]byte) bool {
	for _, f := range flags {
		if f == 0 {
			return false
		}
	}
	return true
}

// GlyphActive 回報這個符印是否還在傷害隊伍。
//
// 用的是傷害判定那一處的門檻（`< 0x80`），與 CircleOfLightOpen 刻意不同 ——
// 兩個原版判斷本來就不一樣，不要為了「看起來一致」把它們合併。
func GlyphActive(flags [3]byte, idx int) bool {
	if idx < 0 || idx >= len(flags) {
		return false
	}
	return flags[idx] < GlyphDone
}

// ImprisonResult 是一次 IMPRISON 的結果。
type ImprisonResult int

const (
	// ImprisonNotEnoughSP：法力不足（"That requires 100 SP!"）。
	ImprisonNotEnoughSP ImprisonResult = iota
	// ImprisonFizzles：地點不對（"The spell fizzles..."）。**法力照樣扣掉。**
	ImprisonFizzles
	// ImprisonWon：禁錮成功 —— 破關。
	ImprisonWon
)

// Imprison 施放禁錮。回傳結果並改寫施法者法力。
//
// ⚠ **地點不對也會扣掉 100 點法力。** 原版的順序是先扣再檢查地點
// （`0x6bfb` 扣費、`0x6c07` 才檢查），在錯的地方施放就是白白損失 ——
// 攻略叮嚀「先備份 PARTY.DAT」正是為此。這個順序刻意照抄，不要「修好」它。
func Imprison(caster *Character, subMap, y int) ImprisonResult {
	if caster == nil || caster.CurrentSP < ImprisonCost {
		return ImprisonNotEnoughSP
	}
	caster.CurrentSP -= ImprisonCost
	if subMap == ImprisonSubMap || y <= ImprisonMaxY {
		return ImprisonWon
	}
	return ImprisonFizzles
}
