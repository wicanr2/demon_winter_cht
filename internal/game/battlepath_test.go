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

func TestFirstStepTowardStraightAndLShape(t *testing.T) {
	for _, tc := range []struct {
		name      string
		targetX   int
		targetY   int
		preferred Facing
		want      Facing
	}{
		{"直線", 13, 10, East, East},
		{"L 形同長路徑依偏好", 12, 12, South, South},
	} {
		t.Run(tc.name, func(t *testing.T) {
			monster := &Unit{Slot: 0, X: 10, Y: 10, HP: 10}
			target := &Unit{
				Slot: PlayerSlotStart, X: tc.targetX, Y: tc.targetY,
				HP: 10, IsPlayer: true,
			}
			b := NewBattle(rng.NewWithSeed(1), []*Unit{monster, target})
			got, ok := b.FirstStepToward(monster, target, tc.preferred)
			if !ok || got != tc.want {
				t.Fatalf("第一步 = %v, %v，預期 %v, true", got, ok, tc.want)
			}
		})
	}
}

func TestFirstStepTowardDetoursAroundTerrain(t *testing.T) {
	monster := &Unit{Slot: 0, X: 10, Y: 10, HP: 10}
	target := &Unit{Slot: PlayerSlotStart, X: 12, Y: 10, HP: 10, IsPlayer: true}
	terrain := flatTerrain(7)
	terrain[10*BattleTerrainSize+11] = 99
	b := NewBattle(rng.NewWithSeed(1), []*Unit{monster, target})
	b.Terrain = terrain

	got, ok := b.FirstStepToward(monster, target, East)
	if !ok || got != South {
		t.Fatalf("正東有牆時第一步 = %v, %v，預期 South, true", got, ok)
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
