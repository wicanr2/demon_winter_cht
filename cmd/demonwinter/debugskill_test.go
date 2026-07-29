package main

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/game"
)

func TestDebugRemoveSkillClearsWholeParty(t *testing.T) {
	a := &app{members: []game.Character{{}, {}}}
	a.members[0].Skills[10] = true
	a.members[1].Skills[10] = true
	a.members[1].Skills[11] = true

	if err := a.debugRemoveSkill("10, 11"); err != nil {
		t.Fatal(err)
	}
	for i := range a.members {
		if a.members[i].Skills[10] || a.members[i].Skills[11] {
			t.Fatalf("member %d still has removed skills", i)
		}
	}
}

func TestDebugRemoveSkillRejectsInvalidID(t *testing.T) {
	a := &app{members: []game.Character{{}}}
	if err := a.debugRemoveSkill("999"); err == nil {
		t.Fatal("invalid skill ID was accepted")
	}
}
