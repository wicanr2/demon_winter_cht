package main

import (
	"strings"
	"testing"
)

func validModernIconManifest() modernIconManifest {
	m := modernIconManifest{
		Schema: modernIconSchema, ID: "modern-icon",
		FrameWidth: modernIconWidth, FrameHeight: modernIconHeight,
	}
	m.Tiles.Normal = map[string]string{"0x01": "normal-01.png"}
	m.Tiles.Winter = map[string]string{"0x01": "winter-01.png"}
	m.TileVariants.Normal = map[string][]string{
		"0x0e": {"normal-0e-a.png", "normal-0e-b.png"},
	}
	m.TileVariants.Winter = map[string][]string{
		"0x0e": {"winter-0e-a.png", "winter-0e-b.png"},
	}
	m.Sprites = map[string]string{"0x1e": "party-north-a.png"}
	m.BattleSprites.Combat = map[string]string{"0x14": "combat-14.png"}
	m.BattleSprites.Monsters = map[string]string{"0xef": "monster-ef.png"}
	m.BattleSprites.MonsterSets = map[string]modernIconDirections{
		"0x03": {
			South: "monster-03-south.png", West: "monster-03-west.png",
			East: "monster-03-east.png", North: "monster-03-north.png",
		},
	}
	m.BattleSprites.Ships = map[string]string{"0x1f": "ship-1f.png"}
	return m
}

func TestValidateModernIconManifest(t *testing.T) {
	if err := validateModernIconManifest(validModernIconManifest()); err != nil {
		t.Fatalf("有效 manifest 被拒絕：%v", err)
	}
	tests := []struct {
		name string
		edit func(*modernIconManifest)
	}{
		{"schema", func(m *modernIconManifest) { m.Schema++ }},
		{"id", func(m *modernIconManifest) { m.ID = "modern-ega" }},
		{"width", func(m *modernIconManifest) { m.FrameWidth = 32 }},
		{"height", func(m *modernIconManifest) { m.FrameHeight = 28 }},
		{"empty", func(m *modernIconManifest) {
			m.Tiles.Normal, m.Tiles.Winter = nil, nil
			m.TileVariants.Normal, m.TileVariants.Winter = nil, nil
			m.Sprites = nil
			m.BattleSprites.Combat = nil
			m.BattleSprites.Monsters = nil
			m.BattleSprites.MonsterSets = nil
			m.BattleSprites.Ships = nil
		}},
		{"index", func(m *modernIconManifest) {
			m.Tiles.Normal = map[string]string{"0xff": "bad.png"}
		}},
		{"path", func(m *modernIconManifest) {
			m.Tiles.Normal = map[string]string{"0x01": "../normal.png"}
		}},
		{"sprite path", func(m *modernIconManifest) {
			m.Sprites = map[string]string{"0x1e": "../party.png"}
		}},
		{"combat index", func(m *modernIconManifest) {
			m.BattleSprites.Combat = map[string]string{"44": "bad.png"}
		}},
		{"monster index", func(m *modernIconManifest) {
			m.BattleSprites.Monsters = map[string]string{"240": "bad.png"}
		}},
		{"ship index", func(m *modernIconManifest) {
			m.BattleSprites.Ships = map[string]string{"32": "bad.png"}
		}},
		{"battle sprite path", func(m *modernIconManifest) {
			m.BattleSprites.Monsters = map[string]string{"0": "../monster.png"}
		}},
		{"variant count", func(m *modernIconManifest) {
			m.TileVariants.Normal = map[string][]string{"0x0e": {"only-one.png"}}
		}},
		{"variant path", func(m *modernIconManifest) {
			m.TileVariants.Normal = map[string][]string{
				"0x0e": {"good.png", "../bad.png"},
			}
		}},
		{"monster set index", func(m *modernIconManifest) {
			m.BattleSprites.MonsterSets = map[string]modernIconDirections{
				"0x1e": {
					South: "s.png", West: "w.png", East: "e.png", North: "n.png",
				},
			}
		}},
		{"monster set missing direction", func(m *modernIconManifest) {
			m.BattleSprites.MonsterSets = map[string]modernIconDirections{
				"0x03": {South: "s.png", West: "w.png", East: "e.png"},
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := validModernIconManifest()
			tc.edit(&m)
			if err := validateModernIconManifest(m); err == nil {
				t.Fatal("無效 manifest 未被拒絕")
			}
		})
	}
}

func TestModernIconRejectsLegacyFrameSize(t *testing.T) {
	m := validModernIconManifest()
	m.FrameWidth, m.FrameHeight = 32, 28
	err := validateModernIconManifest(m)
	if err == nil || !strings.Contains(err.Error(), "64x56") {
		t.Fatalf("32×28 拒絕訊息 = %v", err)
	}
}
