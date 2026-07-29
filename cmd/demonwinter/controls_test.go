package main

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/game"
)

func TestParseControlMode(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want controlMode
	}{
		{"modern", controlsModern},
		{"MODERN", controlsModern},
		{"retro", controlsRetro},
	} {
		got, err := parseControlMode(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("parseControlMode(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
	}
	if _, err := parseControlMode("unknown"); err == nil {
		t.Fatal("parseControlMode(unknown) should fail")
	}
}

func TestControlModeCycle(t *testing.T) {
	if got := controlsModern.next(); got != controlsRetro {
		t.Fatalf("modern.next = %v; want retro", got)
	}
	if got := controlsRetro.next(); got != controlsModern {
		t.Fatalf("retro.next = %v; want modern", got)
	}
}

func TestTurnFacingWraps(t *testing.T) {
	if got := turnFacing(game.North, -1); got != game.West {
		t.Fatalf("north left = %v; want west", got)
	}
	if got := turnFacing(game.West, 1); got != game.North {
		t.Fatalf("west right = %v; want north", got)
	}
	if got := turnFacing(game.East, 2); got != game.West {
		t.Fatalf("east around = %v; want west", got)
	}
}
