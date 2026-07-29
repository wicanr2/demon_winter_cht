package main

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
)

type videoTheme struct {
	normal, winter   *ui.Tileset
	combat, monsters *ui.SpriteSheet
	ships            *ui.SpriteSheet
	icons            *modernIconTheme
}

type themeID string

const (
	themeEGA    themeID = "ega"
	themeCGA    themeID = "cga"
	themeModern themeID = "modern"
)

func loadVideoTheme(dataDir string, mode gfx.VideoMode) (*videoTheme, error) {
	loadTiles := func(set gfx.TerrainSet) (*ui.Tileset, error) {
		src, err := gfx.LoadTilesetMode(dataDir, set, mode)
		if err != nil {
			return nil, err
		}
		return ui.NewTileset(src), nil
	}
	loadSprites := func(base string) (*ui.SpriteSheet, error) {
		src, err := gfx.LoadSpriteSheetMode(dataDir, base, mode)
		if err != nil {
			return nil, err
		}
		return ui.NewSpriteSheet(src), nil
	}
	t := &videoTheme{}
	var err error
	if t.normal, err = loadTiles(gfx.NormalTiles); err != nil {
		return nil, fmt.Errorf("DEMON: %w", err)
	}
	if t.winter, err = loadTiles(gfx.WinterTiles); err != nil {
		return nil, fmt.Errorf("WINTER: %w", err)
	}
	if t.combat, err = loadSprites("COMBAT"); err != nil {
		return nil, fmt.Errorf("COMBAT: %w", err)
	}
	if t.monsters, err = loadSprites("MONSTER"); err != nil {
		return nil, fmt.Errorf("MONSTER: %w", err)
	}
	if t.ships, err = loadSprites("SHIP"); err != nil {
		return nil, fmt.Errorf("SHIP: %w", err)
	}
	return t, nil
}

func modernVideoTheme(src *videoTheme) *videoTheme {
	return &videoTheme{
		normal:   ui.NewTileset(gfx.ModernizeTileset(src.normal.Source())),
		winter:   ui.NewTileset(gfx.ModernizeTileset(src.winter.Source())),
		combat:   ui.NewSpriteSheet(gfx.ModernizeSpriteSheet(src.combat.Source())),
		monsters: ui.NewSpriteSheet(gfx.ModernizeSpriteSheet(src.monsters.Source())),
		ships:    ui.NewSpriteSheet(gfx.ModernizeSpriteSheet(src.ships.Source())),
	}
}

// themeNameKey 只回傳穩定識別字；玩家看見的名稱一律由 ui.json 提供。
func themeNameKey(id themeID) string {
	if id == themeCGA {
		return "theme.name.cga"
	}
	if id == themeModern {
		return "theme.name.modern"
	}
	return "theme.name.ega"
}

func nextThemeID(id themeID) themeID {
	switch id {
	case themeEGA:
		return themeCGA
	case themeCGA:
		return themeModern
	default:
		return themeEGA
	}
}

// toggleVideoTheme 在原版 EGA、CGA 與 Modern Icon 素材間即時切換。
//
// 兩套 theme 已在啟動時完整預載，這裡只交換指標；因此不會出現 atlas
// 上傳中的半幀。倚天 16×15 字型刻意不切換：F8 是美術 theme，不應讓
// 中文可讀性跟著改變。
func (a *app) toggleVideoTheme() error {
	next := nextThemeID(a.themeID)

	t := a.videoThemes[next]
	if t == nil {
		return fmt.Errorf("theme %s unavailable", next)
	}

	a.normal, a.winter = t.normal, t.winter
	a.combatSprites, a.monsterSprites, a.shipSprites = t.combat, t.monsters, t.ships
	a.modernIcons = t.icons
	a.videoMode = t.normal.Mode()
	a.themeID = next
	name := a.tr.UI(themeNameKey(next))
	a.message = fmt.Sprintf(a.tr.UI("theme.changed"), name)
	a.logf(a.tr.UI("theme.log"), name)
	return nil
}
