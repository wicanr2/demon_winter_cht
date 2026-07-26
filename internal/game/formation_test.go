package game

import "testing"

// 原版起始存檔的陣型：五個人散在 A C E G I。
func startFormation() Formation {
	return Formation{0, 0xff, 1, 0xff, 2, 0xff, 3, 0xff, 4}
}

func TestFormation_CellLabelRoundTrip(t *testing.T) {
	want := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	for i, w := range want {
		if got := CellLabel(i); got != w {
			t.Errorf("格 %d 的字母是 %q，預期 %q", i, got, w)
		}
		if got := ParseCellLabel(rune(w[0])); got != i {
			t.Errorf("%q 解回 %d，預期 %d", w, got, i)
		}
	}
	if CellLabel(9) != "" || CellLabel(-1) != "" {
		t.Error("越界的格號應該回空字串")
	}
	for _, r := range []rune{'J', '1', 'z', '@'} {
		if got := ParseCellLabel(r); got != -1 {
			t.Errorf("%q 不是合法格號，卻解成 %d", r, got)
		}
	}
	if ParseCellLabel('c') != 2 {
		t.Error("小寫也要認")
	}
}

func TestFormation_CellOf(t *testing.T) {
	f := startFormation()
	for member, cell := range map[int]int{0: 0, 1: 2, 2: 4, 3: 6, 4: 8} {
		if got := f.CellOf(member); got != cell {
			t.Errorf("成員 %d 在第 %d 格，預期第 %d 格", member, got, cell)
		}
	}
	if f.CellOf(5) != -1 {
		t.Error("不在陣型裡的成員應該回 −1")
	}
}

func TestFormation_PlaceRefusesOccupied(t *testing.T) {
	var f Formation
	f.Clear()
	if !f.Place(4, 0) {
		t.Fatal("空格應該放得進去")
	}
	if f.Place(4, 1) {
		t.Error("已有人的格子不該被覆蓋 —— 原版是重問一次")
	}
	if f[4] != 0 {
		t.Errorf("第 4 格變成 %d，應該還是成員 0", f[4])
	}
	if f.Place(9, 1) || f.Place(-1, 1) {
		t.Error("越界的格號應該被拒絕")
	}
}

func TestFormation_ClearEmptiesEveryCell(t *testing.T) {
	f := startFormation()
	f.Clear()
	for i, v := range f {
		if v != FormationEmpty {
			t.Errorf("第 %d 格清完是 %#x，預期 0xff", i, v)
		}
	}
}

// 離隊要同時做兩件事：清掉那一格，以及把編號比他大的往前挪。
func TestFormation_RemoveMemberRenumbers(t *testing.T) {
	f := startFormation()
	f.RemoveMember(1) // 原本站 C

	if f[2] != FormationEmpty {
		t.Errorf("C 格是 %#x，預期清空", f[2])
	}
	want := map[int]int{0: 0, 1: 4, 2: 6, 3: 8} // 原 2/3/4 各往前一號
	for member, cell := range want {
		if got := f.CellOf(member); got != cell {
			t.Errorf("成員 %d 在第 %d 格，預期第 %d 格", member, got, cell)
		}
	}
}

func TestFormation_AddMemberTakesFirstGap(t *testing.T) {
	f := startFormation()
	if cell := f.AddMember(5); cell != 1 {
		t.Errorf("新成員放進第 %d 格，預期第 1 格（第一個空格 B）", cell)
	}

	var full Formation
	for i := range full {
		full[i] = byte(i)
	}
	if cell := full.AddMember(9); cell != -1 {
		t.Errorf("九格全滿應該回 −1，卻回 %d", cell)
	}
}

func TestFormationOffset(t *testing.T) {
	want := [FormationCells][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {0, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}
	for i, w := range want {
		dx, dy := FormationOffset(i)
		if dx != w[0] || dy != w[1] {
			t.Errorf("格 %s 的位移是 (%d,%d)，預期 (%d,%d)",
				CellLabel(i), dx, dy, w[0], w[1])
		}
	}
}
