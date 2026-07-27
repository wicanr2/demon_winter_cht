package game

import "testing"

func TestGlyphIndexFor(t *testing.T) {
	// 55／56 連號，第三個跳到 66（原版 0x6b3c 的特判）
	for _, c := range []struct {
		subMap, want int
	}{{55, 0}, {56, 1}, {66, 2}, {57, -1}, {65, -1}, {5, -1}, {0, -1}} {
		if got := GlyphIndexFor(c.subMap); got != c.want {
			t.Errorf("子地圖 %d → %d，預期 %d", c.subMap, got, c.want)
		}
	}
}

func TestUncurse(t *testing.T) {
	newCaster := func(sp int) *Character { return &Character{CurrentSP: sp} }

	// 腳下不是符印圖塊
	var flags [3]byte
	if r := Uncurse(newCaster(99), 0x11, 55, &flags); r != GlyphNoGlyph {
		t.Errorf("非符印圖塊得到 %v", r)
	}
	// 在符印圖塊上但子地圖不對（其他地圖也可能有 0x63）
	if r := Uncurse(newCaster(99), GlyphTile, 12, &flags); r != GlyphNoGlyph {
		t.Errorf("子地圖不對得到 %v", r)
	}

	// 法力不足 —— 而且**不能扣費**（原版 UNCURSE 的檢查在扣費之前）
	c := newCaster(49)
	if r := Uncurse(c, GlyphTile, 55, &flags); r != GlyphNotEnoughSP {
		t.Errorf("法力不足得到 %v", r)
	}
	if c.CurrentSP != 49 {
		t.Errorf("法力不足卻扣了費，剩 %d", c.CurrentSP)
	}
	if flags[0] != 0 {
		t.Error("法力不足卻把符印解掉了")
	}

	// 成功
	c = newCaster(50)
	if r := Uncurse(c, GlyphTile, 55, &flags); r != GlyphDestroyed {
		t.Fatalf("預期成功，得到 %v", r)
	}
	if c.CurrentSP != 0 {
		t.Errorf("扣費後剩 %d，預期 0", c.CurrentSP)
	}
	if flags[0] != GlyphDone {
		t.Errorf("旗標 0x%02x，預期 0x80", flags[0])
	}

	// 已解過
	if r := Uncurse(newCaster(99), GlyphTile, 55, &flags); r != GlyphAlreadyDone {
		t.Errorf("重複解得到 %v", r)
	}

	// 三個符印各自獨立
	if flags[1] != 0 || flags[2] != 0 {
		t.Error("解一個卻動到別的旗標")
	}
	Uncurse(newCaster(99), GlyphTile, 66, &flags)
	if flags[2] != GlyphDone {
		t.Error("子地圖 66 應該解到索引 2")
	}
}

func TestCircleOfLightOpen(t *testing.T) {
	if CircleOfLightOpen([3]byte{}) {
		t.Error("一個都沒解就放行了")
	}
	if CircleOfLightOpen([3]byte{GlyphDone, GlyphDone, 0}) {
		t.Error("只解兩個就放行了")
	}
	if !CircleOfLightOpen([3]byte{GlyphDone, GlyphDone, GlyphDone}) {
		t.Error("三個都解了卻擋著")
	}
	// 擋門比的是 != 0，不是 >= 0x80
	if !CircleOfLightOpen([3]byte{1, 1, 1}) {
		t.Error("擋門的門檻應該是 != 0")
	}
}

// TestCircleOfLightPushBack 釘住四格十字的推回結果 ＝ 剛才那一格。
//
// 這四組是原版那兩條式子唯一「算得漂亮」的輸入（`docs/re/98` §1）。
// 誰要是把它改寫成通用的「退一步」，第五個 case 會證明那不等價。
func TestCircleOfLightPushBack(t *testing.T) {
	for _, c := range []struct {
		x, y         int
		f            Facing
		wantX, wantY int
		label        string
	}{
		{11, 49, South, 11, 48, "從北邊走進來"},
		{11, 51, North, 11, 52, "從南邊走進來"},
		{10, 50, East, 9, 50, "從西邊走進來"},
		{12, 50, West, 13, 50, "從東邊走進來"},
	} {
		x, y := CircleOfLightPushBack(c.x, c.y, c.f)
		if x != c.wantX || y != c.wantY {
			t.Errorf("%s（%d,%d 面向 %d）：推到 (%d,%d)，預期 (%d,%d)",
				c.label, c.x, c.y, c.f, x, y, c.wantX, c.wantY)
		}
	}

	// **不是通用的「退一步」。** 斜著進 (11,49) 的話兩條式子接連生效，
	// 原版會把隊伍丟到 (9,50) 而不是退回 (10,49)。照抄式子才會有這個行為。
	if x, y := CircleOfLightPushBack(11, 49, East); x != 9 || y != 50 {
		t.Errorf("斜向進入：推到 (%d,%d)，預期 (9,50) —— 這一格證明它不是退一步", x, y)
	}
}

// TestGlyphActiveUsesDifferentThreshold 釘住「兩處門檻不同」這件事。
// 原版擋門比 != 0、傷害判定比 < 0x80；中間值在原版資料裡不存在，
// 但不要為了看起來一致而把兩者合併。
func TestGlyphActiveUsesDifferentThreshold(t *testing.T) {
	mid := [3]byte{1, 1, 1}
	if !CircleOfLightOpen(mid) {
		t.Error("擋門：非 0 就該放行")
	}
	if !GlyphActive(mid, 0) {
		t.Error("傷害：小於 0x80 就該還在傷人")
	}
	done := [3]byte{GlyphDone, GlyphDone, GlyphDone}
	if GlyphActive(done, 0) {
		t.Error("解完了不該還在傷人")
	}
	if GlyphActive(done, 5) {
		t.Error("越界索引不該回 true")
	}
}

func TestImprison(t *testing.T) {
	newCaster := func(sp int) *Character { return &Character{CurrentSP: sp} }

	// 法力不足：不扣費
	c := newCaster(99)
	if r := Imprison(c, 5, 0); r != ImprisonNotEnoughSP {
		t.Errorf("法力不足得到 %v", r)
	}
	if c.CurrentSP != 99 {
		t.Errorf("法力不足卻扣了費，剩 %d", c.CurrentSP)
	}

	// 地點不對：**照樣扣 100**（原版先扣再檢查，刻意保留）
	c = newCaster(100)
	if r := Imprison(c, 12, 40); r != ImprisonFizzles {
		t.Errorf("地點不對得到 %v", r)
	}
	if c.CurrentSP != 0 {
		t.Errorf("地點不對應該照扣，剩 %d", c.CurrentSP)
	}

	// 子地圖 5 → 成功
	c = newCaster(100)
	if r := Imprison(c, 5, 40); r != ImprisonWon {
		t.Errorf("子地圖 5 得到 %v", r)
	}
	// Y <= 6 → 成功（或的關係，不是且）
	c = newCaster(100)
	if r := Imprison(c, 12, 6); r != ImprisonWon {
		t.Errorf("Y=6 得到 %v", r)
	}
	c = newCaster(100)
	if r := Imprison(c, 12, 7); r != ImprisonFizzles {
		t.Errorf("Y=7 得到 %v，預期失敗", r)
	}
}
