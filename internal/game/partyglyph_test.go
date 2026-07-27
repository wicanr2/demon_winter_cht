package game

import "testing"

// 四個面向各自的基底，直接對 `ds:0x210f` 的原始位元組 `1e 1b 18 21`。
func TestPartyGlyphBasePerFacing(t *testing.T) {
	// 座標取偶數，讓 (axis & 1) == 0，量到的就是基底本身。
	for _, c := range []struct {
		f    Facing
		want byte
	}{{North, 0x1e}, {East, 0x1b}, {South, 0x18}, {West, 0x21}} {
		if got := PartyGlyph(c.f, 10, 10, false); got != c.want {
			t.Errorf("面向 %d：glyph %#x，預期 %#x", c.f, got, c.want)
		}
	}
}

// 走路動畫看的是**行進軸**：東西向看 X、南北向看 Y。
// 這一條是原版 `222f:0b8c` 那個 `facing & 1` 分支的直接翻譯 ——
// 若寫成「一律看 X」，南北向走動時 glyph 不會換，畫面上是「滑行」不是走路。
func TestPartyGlyphAnimatesAlongTravelAxis(t *testing.T) {
	// 南北向：Y 變才換 glyph，X 變不換。
	if PartyGlyph(North, 10, 10, false) == PartyGlyph(North, 10, 11, false) {
		t.Error("面向北：Y 換了奇偶，glyph 卻沒換")
	}
	if PartyGlyph(North, 10, 10, false) != PartyGlyph(North, 11, 10, false) {
		t.Error("面向北：X 換了奇偶不該影響 glyph")
	}
	// 東西向：反過來。
	if PartyGlyph(East, 10, 10, false) == PartyGlyph(East, 11, 10, false) {
		t.Error("面向東：X 換了奇偶，glyph 卻沒換")
	}
	if PartyGlyph(East, 10, 10, false) != PartyGlyph(East, 10, 11, false) {
		t.Error("面向東：Y 換了奇偶不該影響 glyph")
	}
}

// 走路動畫只有兩格，而且與相鄰面向的基底不重疊
// （基底相差 3，+1 之後不會撞到下一個面向）。
func TestPartyGlyphStaysInItsOwnPair(t *testing.T) {
	for _, f := range []Facing{North, East, South, West} {
		a := PartyGlyph(f, 10, 10, false)
		b := PartyGlyph(f, 11, 11, false)
		if b != a+1 && b != a {
			t.Errorf("面向 %d：兩個奇偶得到 %#x／%#x，不是相鄰的一對", f, a, b)
		}
	}
}

// 搭船時走另一組（原版 facing + 0x3f），而且與走路那組完全不重疊。
func TestPartyGlyphShip(t *testing.T) {
	walk := map[byte]bool{}
	for _, f := range []Facing{North, East, South, West} {
		walk[PartyGlyph(f, 10, 10, false)] = true
		walk[PartyGlyph(f, 11, 11, false)] = true
	}
	for _, f := range []Facing{North, East, South, West} {
		got := PartyGlyph(f, 10, 10, true)
		if want := byte(0x3f + int(f)); got != want {
			t.Errorf("搭船面向 %d：glyph %#x，預期 %#x", f, got, want)
		}
		if walk[got] {
			t.Errorf("搭船面向 %d 的 glyph %#x 與走路那組重疊", f, got)
		}
	}
	// 搭船時座標奇偶不影響（原版那條路徑沒有 +1）。
	if PartyGlyph(East, 10, 10, true) != PartyGlyph(East, 11, 11, true) {
		t.Error("搭船時 glyph 不該隨座標變")
	}
}
