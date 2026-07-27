package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// fakeMap／errOutOfMap 在 crushingwalls_test.go，這裡直接用同一份。

// graveyard 造一片與 MAP3.MAP 同形狀的墓碑（x 20..30、y 57..62 全是 0x56）。
func graveyard() *fakeMap {
	m := &fakeMap{}
	for y := 57; y <= 62; y++ {
		for x := 20; x <= 30; x++ {
			_ = m.SetTileAt(x, y, TombstoneOpenTile)
		}
	}
	return m
}

func TestTombstoneShiftBlocksThirty(t *testing.T) {
	m := graveyard()
	stones := TombstoneShift(rng.NewWithSeed(1234), m)

	if len(stones) != TombstoneBlockCount {
		t.Fatalf("長出 %d 塊，預期 %d", len(stones), TombstoneBlockCount)
	}
	// 全部落在那片 11×6 的外框內，而且不重複。
	seen := map[[2]int]bool{}
	for _, p := range stones {
		if p.X < 20 || p.X > 30 || p.Y < 57 || p.Y > 62 {
			t.Errorf("(%d,%d) 掉到墓園外面了", p.X, p.Y)
		}
		if seen[[2]int{p.X, p.Y}] {
			t.Errorf("(%d,%d) 長了兩次 —— 原版靠「不是墓碑就重擲」排除重複", p.X, p.Y)
		}
		seen[[2]int{p.X, p.Y}] = true
	}

	// 三格強制留通，**排在長石頭之後**，所以就算被挑中也會被寫回去。
	for _, p := range tombstoneKeepOpen {
		got, _ := m.TileAt(p.X, p.Y)
		if got != TombstoneOpenTile {
			t.Errorf("(%d,%d) 應該強制留通，卻是 %#02x", p.X, p.Y, got)
		}
	}

	// 剩下的墓碑數量：66 − 30 ＋（三格被挑中又寫回去的）。
	open := 0
	for y := 57; y <= 62; y++ {
		for x := 20; x <= 30; x++ {
			if v, _ := m.TileAt(x, y); v == TombstoneOpenTile {
				open++
			}
		}
	}
	if open < 66-TombstoneBlockCount {
		t.Errorf("還能走的格子只剩 %d，比 66−30 還少", open)
	}
}

// 每一次都不一樣 —— 這是「墓碑在你面前挪動」的意義。
func TestTombstoneShiftIsRandomEachTime(t *testing.T) {
	a := TombstoneShift(rng.NewWithSeed(1), graveyard())
	b := TombstoneShift(rng.NewWithSeed(2), graveyard())
	same := 0
	for i := range a {
		if a[i] == b[i] {
			same++
		}
	}
	if same == len(a) {
		t.Error("兩個不同種子擲出完全相同的迷宮")
	}
}

// 三格留通的作用是「踩進入口之後至少有一步可以走」。
// 它們各對一個入口的相鄰格，而且都在隨機範圍內 —— 兩件事都釘住。
func TestTombstoneKeepOpenNeighboursTheEntrances(t *testing.T) {
	entrances := [3]struct{ X, Y int }{{28, 60}, {24, 58}, {20, 60}}
	for i, p := range tombstoneKeepOpen {
		e := entrances[i]
		dx, dy := p.X-e.X, p.Y-e.Y
		if abs(dx)+abs(dy) != 1 { // abs 在 spell.go
			t.Errorf("留通格 (%d,%d) 不在入口 (%d,%d) 旁邊", p.X, p.Y, e.X, e.Y)
		}
		if p.X < 20 || p.X > 30 || p.Y < 57 || p.Y > 62 {
			t.Errorf("留通格 (%d,%d) 在隨機範圍外，那樣就沒有意義了", p.X, p.Y)
		}
	}
}
