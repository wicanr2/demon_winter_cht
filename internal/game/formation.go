package game

// 3×3 陣型格（存檔 trailer `+0x00`–`+0x08`，出處 `docs/re/34`）。
//
// 紮營選單的 Reorder 就是在編這張表。畫面長這樣：
//
//	Positions:
//	   A B C
//	   D E F
//	   G H I
//
// 每一格存一名成員的編號，`0xFF` 代表空格。原版起始存檔是
// `00 ff 01 ff 02 ff 03 ff 04` —— 五個人散在 A C E G I，隔一格站一個。

const (
	// FormationCells 是格數，FormationCols／FormationRows 是排法。
	FormationCells = 9
	FormationCols  = 3
	FormationRows  = 3

	// FormationEmpty 是空格。
	FormationEmpty = 0xff
)

// Formation 是九個格子，索引 0–8 對應 A–I。
type Formation [FormationCells]byte

// CellLabel 回傳格號的顯示字母（A–I）。超出範圍回空字串。
func CellLabel(cell int) string {
	if cell < 0 || cell >= FormationCells {
		return ""
	}
	return string(rune('A' + cell))
}

// ParseCellLabel 把 A–I（大小寫皆可）換成格號，不是字母回 −1。
func ParseCellLabel(r rune) int {
	if r >= 'a' && r <= 'z' {
		r -= 'a' - 'A'
	}
	cell := int(r - 'A')
	if cell < 0 || cell >= FormationCells {
		return -1
	}
	return cell
}

// Clear 把九格全部清空。**Reorder 一開始就做這件事** ——
// 原版不是「換兩個人的位置」，是整張表重填（`1000:03c6`）。
func (f *Formation) Clear() {
	for i := range f {
		f[i] = FormationEmpty
	}
}

// Occupied 回報這一格有沒有人。
func (f Formation) Occupied(cell int) bool {
	return cell >= 0 && cell < FormationCells && f[cell] != FormationEmpty
}

// CellOf 回傳這名成員站在第幾格，不在陣型裡回 −1。
func (f Formation) CellOf(member int) int {
	for i, v := range f {
		if v != FormationEmpty && int(v) == member {
			return i
		}
	}
	return -1
}

// Place 把成員放進指定格。格號越界或那一格已有人就回 false ——
// 原版是「重問一次」（`1000:04b4` 的 `!= 0xff → jmp 回去`），不是覆蓋。
func (f *Formation) Place(cell, member int) bool {
	if cell < 0 || cell >= FormationCells {
		return false
	}
	if f[cell] != FormationEmpty {
		return false
	}
	f[cell] = byte(member)
	return true
}

// RemoveMember 把一名成員移出陣型，並把編號比他大的往前挪一格。
//
// 挪編號這件事不是多此一舉：陣型格存的是**成員在隊伍陣列裡的索引**，
// 有人離隊之後後面的人整個往前移，格子裡的舊索引就會指錯人。
// 原版在 `0x14af2` 的同一個迴圈裡把兩件事一起做完。
func (f *Formation) RemoveMember(member int) {
	for i, v := range f {
		if v == FormationEmpty {
			continue
		}
		switch {
		case int(v) == member:
			f[i] = FormationEmpty
		case int(v) > member:
			f[i] = v - 1
		}
	}
}

// AddMember 把新成員放進第一個空格，回傳格號；沒有空格回 −1
// （`0x15089`）。
func (f *Formation) AddMember(member int) int {
	for i, v := range f {
		if v == FormationEmpty {
			f[i] = byte(member)
			return i
		}
	}
	return -1
}

// FormationOffset 回傳格號的相對座標，以中央格（E，格號 4）為原點。
//
//	A(-1,-1) B(0,-1) C(1,-1)
//	D(-1, 0) E(0, 0) F(1, 0)
//	G(-1, 1) H(0, 1) I(1, 1)
//
// 原版佈陣（`0xc615`–`0xc6c4`）就是拿這組位移去加一個中心點。
// **本專案還沒有把它接到戰鬥擺位上** —— 原版的中心點是 64 寬緩衝裡的
// (13, 13)，與本專案 9×9 的戰場網格還沒對上（見 `docs/re/34` §4）。
func FormationOffset(cell int) (dx, dy int) {
	if cell < 0 || cell >= FormationCells {
		return 0, 0
	}
	return cell%FormationCols - 1, cell/FormationCols - 1
}
