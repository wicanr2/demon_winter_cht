package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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
	m.DungeonTiles = map[string]string{"0x01": "dungeon-01.png"}
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
	m.BattleSprites.ShipSets = map[string]modernIconDirections{
		"0x01": {
			South: "ship-south.png", SouthB: "ship-south-b.png",
			West: "ship-west.png", East: "ship-east.png", North: "ship-north.png",
		},
	}
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
			m.DungeonTiles = nil
			m.Sprites = nil
			m.BattleSprites.Combat = nil
			m.BattleSprites.Monsters = nil
			m.BattleSprites.MonsterSets = nil
			m.BattleSprites.Ships = nil
			m.BattleSprites.ShipSets = nil
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
		{"dungeon path", func(m *modernIconManifest) {
			m.DungeonTiles = map[string]string{"0x01": "../dungeon.png"}
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
		{"monster set bad B path", func(m *modernIconManifest) {
			set := m.BattleSprites.MonsterSets["0x03"]
			set.SouthB = "../south-b.png"
			m.BattleSprites.MonsterSets["0x03"] = set
		}},
		{"ship set index", func(m *modernIconManifest) {
			m.BattleSprites.ShipSets = map[string]modernIconDirections{
				"0x04": {South: "s.png", West: "w.png", East: "e.png", North: "n.png"},
			}
		}},
		{"ship set missing direction", func(m *modernIconManifest) {
			m.BattleSprites.ShipSets = map[string]modernIconDirections{
				"0x01": {South: "s.png", West: "w.png", East: "e.png"},
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

func TestModernIconMapEnabledOnlyForPlayableMaps(t *testing.T) {
	for _, mapID := range []int{0, 6, 10, 78, 80} {
		if modernIconMapEnabled(mapID) {
			t.Errorf("map %d 是無效地圖，不得套用 Modern Icon terrain", mapID)
		}
	}
	for _, mapID := range []int{1, 2, 3, 4, 5, 11, 34, 64} {
		if !modernIconMapEnabled(mapID) {
			t.Errorf("map %d 是可玩地圖，應允許 Modern Icon 命名空間", mapID)
		}
	}
}

func TestModernIconTerrainNamespacesDoNotCollide(t *testing.T) {
	worldTile := &ebiten.Image{}
	dungeonTile := &ebiten.Image{}
	theme := &modernIconTheme{
		normal:  map[byte]*ebiten.Image{1: worldTile},
		dungeon: map[byte]*ebiten.Image{1: dungeonTile},
	}
	if got := theme.terrainAt(34, false, 1, 0, 0); got != worldTile {
		t.Fatal("世界 map 34 沒有使用 normal namespace")
	}
	if got := theme.terrainAt(1, false, 1, 0, 0); got != dungeonTile {
		t.Fatal("地城 MAP1 沒有使用 dungeonTiles namespace")
	}
	if got := theme.terrainAt(6, false, 1, 0, 0); got != nil {
		t.Fatal("無效 map 6 不得回傳任何 terrain")
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

func TestBundledModernIconDungeonMatchesInventory(t *testing.T) {
	root := filepath.Join("..", "..")
	theme, err := loadModernIconTheme(filepath.Join(
		root, "artwork", "modern-icon", "m1", "trial"))
	if err != nil {
		t.Fatalf("載入隨附 Modern Icon：%v", err)
	}
	raw, err := os.ReadFile(filepath.Join(
		root, "artwork", "modern-icon", "m1", "dungeon-inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		Tiles []struct {
			Index string `json:"index"`
		} `json:"tiles"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	if len(theme.dungeon) != len(inventory.Tiles) {
		t.Fatalf("dungeonTiles=%d，實際使用索引=%d",
			len(theme.dungeon), len(inventory.Tiles))
	}
	for _, tile := range inventory.Tiles {
		v, err := strconv.ParseUint(strings.TrimPrefix(tile.Index, "0x"), 16, 8)
		if err != nil {
			t.Fatalf("inventory index %q：%v", tile.Index, err)
		}
		if theme.dungeon[byte(v)] == nil {
			t.Errorf("dungeonTiles 缺 %s", tile.Index)
		}
	}
}
