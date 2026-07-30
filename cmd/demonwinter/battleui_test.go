package main

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/audio/pcspeaker"
	"github.com/wicanr2/demon_winter_cht/internal/game"
)

func TestAttackEffectsMatchOriginalBranches(t *testing.T) {
	for _, tc := range []struct {
		name string
		unit game.Unit
		miss int
		hit  int
	}{
		{"怪物一般武器", game.Unit{WeaponIndex: 1}, pcspeaker.EffectC3, pcspeaker.EffectC4},
		{"玩家一般武器", game.Unit{IsPlayer: true, WeaponIndex: 1}, pcspeaker.EffectF3, pcspeaker.EffectC4},
		{"武器類型2", game.Unit{WeaponIndex: 2}, pcspeaker.EffectC3, pcspeaker.EffectG3},
		{"帶毒武器類型11", game.Unit{IsPlayer: true, WeaponIndex: -0xb}, pcspeaker.EffectF3, pcspeaker.EffectG3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := attackMissEffect(&tc.unit); got != tc.miss {
				t.Fatalf("未命中音效 = %d，預期 %d", got, tc.miss)
			}
			if got := attackHitEffect(&tc.unit); got != tc.hit {
				t.Fatalf("命中音效 = %d，預期 %d", got, tc.hit)
			}
		})
	}
}

func TestBreathEffectOnlyUsesDeathMelodyForFatalHit(t *testing.T) {
	for _, tc := range []struct {
		name string
		hits []game.BreathHit
		want int
	}{
		{"受傷", []game.BreathHit{{Damage: 3}}, pcspeaker.EffectC4},
		{"免疫", []game.BreathHit{{Damage: 0}}, pcspeaker.EffectC4},
		{"其中一人死亡", []game.BreathHit{
			{Damage: 2}, {Damage: 8, Killed: true},
		}, pcspeaker.EffectDeath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := breathEffect(tc.hits); got != tc.want {
				t.Fatalf("吐息音效 = %d，預期 %d", got, tc.want)
			}
		})
	}
}

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
