package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func TestFirstStepTowardDetoursAroundOccupiedCell(t *testing.T) {
	target := &Unit{Slot: 0, X: 12, Y: 10, HP: 10}
	player := &Unit{Slot: PlayerSlotStart, X: 10, Y: 10, HP: 10, IsPlayer: true}
	blocker := &Unit{Slot: PlayerSlotStart + 1, X: 11, Y: 10, HP: 10, IsPlayer: true}
	b := NewBattle(rng.NewWithSeed(1), []*Unit{target, player, blocker})

	got, ok := b.FirstStepToward(player, target, East)
	if !ok {
		t.Fatal("正東被隊友擋住時應找到繞路")
	}
	if got != South {
		t.Fatalf("第一步 = %v，預期往南繞路", got)
	}
}

func TestFirstStepTowardNoRoute(t *testing.T) {
	target := &Unit{Slot: 0, X: 12, Y: 10, HP: 10}
	player := &Unit{Slot: PlayerSlotStart, X: 10, Y: 10, HP: 10, IsPlayer: true}
	units := []*Unit{target, player}
	for i, p := range [][2]int{{9, 10}, {11, 10}, {10, 9}, {10, 11}} {
		units = append(units, &Unit{Slot: PlayerSlotStart + 1 + i, X: p[0], Y: p[1], HP: 10})
	}
	b := NewBattle(rng.NewWithSeed(1), units)

	if _, ok := b.FirstStepToward(player, target, East); ok {
		t.Fatal("四面被占滿時不應聲稱有路")
	}
}
