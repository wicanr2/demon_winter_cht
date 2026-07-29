package main

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

func validModernManifest() modernThemeManifest {
	m := modernThemeManifest{
		Schema: modernThemeSchema, ID: "modern-ega", Label: "Modern EGA",
		FrameWidth: gfx.EGATileWidth, FrameHeight: gfx.EGATileHeight,
		TerrainFrames: modernTerrainFrames, CombatFrames: modernCombatFrames,
		MonsterFrames: modernMonsterFrames, ShipFrames: modernShipFrames,
	}
	m.Sheets.Normal = "terrain-demon.png"
	m.Sheets.Winter = "terrain-winter.png"
	m.Sheets.Combat = "combat.png"
	m.Sheets.Monsters = "monster.png"
	m.Sheets.Ships = "ship.png"
	return m
}

func TestValidateModernManifest(t *testing.T) {
	if err := validateModernManifest(validModernManifest()); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		edit func(*modernThemeManifest)
	}{
		{"schema", func(m *modernThemeManifest) { m.Schema = 2 }},
		{"id", func(m *modernThemeManifest) { m.ID = "wrong" }},
		{"size", func(m *modernThemeManifest) { m.FrameWidth = 31 }},
		{"terrain count", func(m *modernThemeManifest) { m.TerrainFrames-- }},
		{"combat count", func(m *modernThemeManifest) { m.CombatFrames-- }},
		{"monster count", func(m *modernThemeManifest) { m.MonsterFrames-- }},
		{"ship count", func(m *modernThemeManifest) { m.ShipFrames-- }},
		{"nested path", func(m *modernThemeManifest) { m.Sheets.Normal = "../bad.png" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := validModernManifest()
			c.edit(&m)
			if err := validateModernManifest(m); err == nil {
				t.Fatal("預期拒絕不合規 manifest")
			}
		})
	}
}
