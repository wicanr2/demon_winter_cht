package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
)

const (
	modernThemeSchema   = 1
	modernTerrainFrames = 102
	modernCombatFrames  = 44
	modernMonsterFrames = 240
	modernShipFrames    = 32
)

type modernThemeManifest struct {
	Schema        int    `json:"schema"`
	ID            string `json:"id"`
	Label         string `json:"label"`
	FrameWidth    int    `json:"frameWidth"`
	FrameHeight   int    `json:"frameHeight"`
	TerrainFrames int    `json:"terrainFrames"`
	CombatFrames  int    `json:"combatFrames"`
	MonsterFrames int    `json:"monsterFrames"`
	ShipFrames    int    `json:"shipFrames"`
	Sheets        struct {
		Normal   string `json:"normal"`
		Winter   string `json:"winter"`
		Combat   string `json:"combat"`
		Monsters string `json:"monsters"`
		Ships    string `json:"ships"`
	} `json:"sheets"`
}

// loadModernPNGTheme 載入已核准的逐格 Modern EGA atlas。現在的完整調色預覽
// 不走這支；只有明確傳入 -modern-theme-dir 才會用外部美術覆蓋 preview。
func loadModernPNGTheme(dir string) (*videoTheme, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "theme.json"))
	if err != nil {
		return nil, fmt.Errorf("讀取 Modern EGA manifest：%w", err)
	}
	var m modernThemeManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("解析 Modern EGA manifest：%w", err)
	}
	if err := validateModernManifest(m); err != nil {
		return nil, err
	}
	path := func(name string) string { return filepath.Join(dir, name) }
	normal, err := gfx.LoadPNGTileset(
		path(m.Sheets.Normal), gfx.NormalTiles, m.FrameWidth, m.FrameHeight, m.TerrainFrames)
	if err != nil {
		return nil, err
	}
	winter, err := gfx.LoadPNGTileset(
		path(m.Sheets.Winter), gfx.WinterTiles, m.FrameWidth, m.FrameHeight, m.TerrainFrames)
	if err != nil {
		return nil, err
	}
	combat, err := gfx.LoadPNGSpriteSheet(
		path(m.Sheets.Combat), m.FrameWidth, m.FrameHeight, m.CombatFrames)
	if err != nil {
		return nil, err
	}
	monsters, err := gfx.LoadPNGSpriteSheet(
		path(m.Sheets.Monsters), m.FrameWidth, m.FrameHeight, m.MonsterFrames)
	if err != nil {
		return nil, err
	}
	ships, err := gfx.LoadPNGSpriteSheet(
		path(m.Sheets.Ships), m.FrameWidth, m.FrameHeight, m.ShipFrames)
	if err != nil {
		return nil, err
	}
	return &videoTheme{
		normal: ui.NewTileset(normal), winter: ui.NewTileset(winter),
		combat: ui.NewSpriteSheet(combat), monsters: ui.NewSpriteSheet(monsters),
		ships: ui.NewSpriteSheet(ships),
	}, nil
}

func validateModernManifest(m modernThemeManifest) error {
	switch {
	case m.Schema != modernThemeSchema:
		return fmt.Errorf("Modern EGA manifest schema = %d，預期 %d", m.Schema, modernThemeSchema)
	case m.ID != "modern-ega":
		return fmt.Errorf("Modern EGA manifest id = %q，預期 modern-ega", m.ID)
	case m.FrameWidth != gfx.EGATileWidth || m.FrameHeight != gfx.EGATileHeight:
		return fmt.Errorf("Modern EGA frame = %dx%d，預期 %dx%d",
			m.FrameWidth, m.FrameHeight, gfx.EGATileWidth, gfx.EGATileHeight)
	case m.TerrainFrames != modernTerrainFrames:
		return fmt.Errorf("Modern EGA terrainFrames = %d，預期 %d", m.TerrainFrames, modernTerrainFrames)
	case m.CombatFrames != modernCombatFrames:
		return fmt.Errorf("Modern EGA combatFrames = %d，預期 %d", m.CombatFrames, modernCombatFrames)
	case m.MonsterFrames != modernMonsterFrames:
		return fmt.Errorf("Modern EGA monsterFrames = %d，預期 %d", m.MonsterFrames, modernMonsterFrames)
	case m.ShipFrames != modernShipFrames:
		return fmt.Errorf("Modern EGA shipFrames = %d，預期 %d", m.ShipFrames, modernShipFrames)
	}
	for label, name := range map[string]string{
		"normal": m.Sheets.Normal, "winter": m.Sheets.Winter, "combat": m.Sheets.Combat,
		"monsters": m.Sheets.Monsters, "ships": m.Sheets.Ships,
	} {
		if name == "" || filepath.Base(name) != name {
			return fmt.Errorf("Modern EGA sheets.%s 必須是同目錄內的檔名，收到 %q", label, name)
		}
	}
	return nil
}
