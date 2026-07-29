package main

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/game"
)

func TestUnitToCharacterUsesSlotNotDuplicateName(t *testing.T) {
	members := []game.Character{{Name: "同名"}, {Name: "同名"}}
	u := &game.Unit{Name: "同名", Slot: game.PlayerSlotStart + 1}
	if got := u2c(members, u); got != &members[1] {
		t.Fatal("同名角色必須依戰鬥槽位配回第二人")
	}
}

func TestMonsterSpellAreaCentreUsesAITarget(t *testing.T) {
	target := &game.Unit{X: 17, Y: 9}
	x, y := monsterSpellAreaCentre(game.EffectAOE, target)
	if x != target.X || y != target.Y {
		t.Fatalf("怪物範圍法術中心 = (%d,%d)，預期 AI 目標 (%d,%d)",
			x, y, target.X, target.Y)
	}
}

func TestMonsterSpellAreaCentreIgnoresNonAreaTarget(t *testing.T) {
	target := &game.Unit{X: 17, Y: 9}
	x, y := monsterSpellAreaCentre(game.EffectInstantDeath, target)
	if x != 0 || y != 0 {
		t.Fatalf("單體法術不應帶範圍中心，得到 (%d,%d)", x, y)
	}
}
