package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func TestSeaCostsAndEscape(t *testing.T) {
	b := NewSeaBattle(rng.NewWithSeed(1), 75, []*SeaUnit{{
		Name: "pirate", Kind: SeaPirate, X: 1, Y: 1, Hull: 20,
	}})
	if !b.Move(false) || b.Points != 5 {
		t.Fatalf("forward: points=%d", b.Points)
	}
	if !b.Turn(-1) || b.Points != 3 {
		t.Fatalf("turn: points=%d", b.Points)
	}
	if !b.Move(true) || b.Points != 0 {
		t.Fatalf("reverse: points=%d", b.Points)
	}
	if b.Fire(East).Fired {
		t.Fatalf("0 點時不應能開砲")
	}

	p := b.PlayerShip()
	p.X, p.Y, p.Facing = 0, 10, West
	b.Points = SeaTurnPoints
	b.Move(false)
	if b.Outcome != SeaEscaped {
		t.Fatalf("edge exit = %v", b.Outcome)
	}
}

func TestSeaCannonDamageAndVictory(t *testing.T) {
	b := NewSeaBattle(rng.NewWithSeed(1), 75, []*SeaUnit{{
		Name: "pirate", Kind: SeaPirate, X: SeaCentre, Y: SeaCentre - 1,
		Hull: 1, MaxHull: 1, Experience: 25,
	}})
	res := b.Fire(North)
	if !res.Hit || !res.Sunk || res.Damage < 1 || res.Damage > 10 {
		t.Fatalf("shot = %+v", res)
	}
	if b.Outcome != SeaVictory || b.Experience() != 25 {
		t.Fatalf("outcome=%v exp=%d", b.Outcome, b.Experience())
	}
}

func TestSeaMonsterClosesAndSinks(t *testing.T) {
	b := NewSeaBattle(rng.NewWithSeed(1), 1, []*SeaUnit{{
		Name: "serpent", Kind: SeaMonster, X: SeaCentre, Y: SeaCentre - 1, Hull: 20,
	}})
	results := b.EnemyTurn()
	if len(results) != 1 || !results[0].Hit || b.Outcome != SeaSunk {
		t.Fatalf("results=%+v outcome=%v hull=%d", results, b.Outcome, b.PlayerShip().Hull)
	}
}
