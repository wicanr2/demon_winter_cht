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
}

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

func videoModeName(mode gfx.VideoMode) string {
	if mode == gfx.ModeCGA {
		return "CGA"
	}
	return "EGA"
}

// toggleVideoTheme 在原版 EGA（.SHE）與 CGA（.SHP）素材間即時切換。
//
// 兩套 theme 已在啟動時完整預載，這裡只交換指標；因此不會出現 atlas
// 上傳中的半幀。倚天 16×15 字型刻意不切換：F8 是美術 theme，不應讓
// 中文可讀性跟著改變。
func (a *app) toggleVideoTheme() error {
	next := gfx.ModeCGA
	if a.videoMode == gfx.ModeCGA {
		next = gfx.ModeEGA
	}

	t := a.videoThemes[next]
	if t == nil {
		return fmt.Errorf("%s theme 尚未載入", videoModeName(next))
	}

	a.normal, a.winter = t.normal, t.winter
	a.combatSprites, a.monsterSprites, a.shipSprites = t.combat, t.monsters, t.ships
	a.videoMode = next
	name := videoModeName(next)
	a.message = a.tr.UI("theme.changed", "顯示主題：") + name
	a.logf("F8 → %s theme", name)
	return nil
}
