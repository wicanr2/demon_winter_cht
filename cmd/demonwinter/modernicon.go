package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

const (
	modernIconSchema = 1
	modernIconWidth  = gfx.EGATileWidth * scale
	modernIconHeight = gfx.EGATileHeight * scale
)

// modernIconTheme 是 Modern Icon 的高解析呈現材質。只收已真正重畫並核准的
// 索引；缺格就保留底下的相容預覽，不拿舊圖或假圖冒充完整新素材。
type modernIconTheme struct {
	normal, winter map[byte]*ebiten.Image
	sprites        map[byte]*ebiten.Image
	combat         map[byte]*ebiten.Image
	monsters       map[byte]*ebiten.Image
	ships          map[byte]*ebiten.Image
}

type modernIconManifest struct {
	Schema      int    `json:"schema"`
	ID          string `json:"id"`
	FrameWidth  int    `json:"frameWidth"`
	FrameHeight int    `json:"frameHeight"`
	Tiles       struct {
		Normal map[string]string `json:"normal"`
		Winter map[string]string `json:"winter"`
	} `json:"tiles"`
	Sprites       map[string]string `json:"sprites"`
	BattleSprites struct {
		Combat   map[string]string `json:"combat"`
		Monsters map[string]string `json:"monsters"`
		Ships    map[string]string `json:"ships"`
	} `json:"battleSprites"`
}

func loadModernIconTheme(dir string) (*modernIconTheme, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "theme.json"))
	if err != nil {
		return nil, fmt.Errorf("讀取 Modern Icon manifest：%w", err)
	}
	var m modernIconManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("解析 Modern Icon manifest：%w", err)
	}
	if err := validateModernIconManifest(m); err != nil {
		return nil, err
	}
	loadSet := func(label string, entries map[string]string, allowAlpha bool,
		frames int) (map[byte]*ebiten.Image, error) {
		out := make(map[byte]*ebiten.Image, len(entries))
		for key, name := range entries {
			index, err := parseModernIconIndexLimit(key, frames)
			if err != nil {
				return nil, fmt.Errorf("Modern Icon %s index %q：%w", label, key, err)
			}
			src, err := loadModernIconPNG(filepath.Join(dir, name), allowAlpha)
			if err != nil {
				return nil, fmt.Errorf("Modern Icon %s[%#02x]：%w", label, index, err)
			}
			out[index] = ebiten.NewImageFromImage(src)
		}
		return out, nil
	}
	normal, err := loadSet("normal", m.Tiles.Normal, false, modernTerrainFrames)
	if err != nil {
		return nil, err
	}
	winter, err := loadSet("winter", m.Tiles.Winter, false, modernTerrainFrames)
	if err != nil {
		return nil, err
	}
	sprites, err := loadSet("sprites", m.Sprites, true, modernTerrainFrames)
	if err != nil {
		return nil, err
	}
	combat, err := loadSet("battleSprites.combat", m.BattleSprites.Combat, true,
		modernCombatFrames)
	if err != nil {
		return nil, err
	}
	monsters, err := loadSet("battleSprites.monsters", m.BattleSprites.Monsters, true,
		modernMonsterFrames)
	if err != nil {
		return nil, err
	}
	ships, err := loadSet("battleSprites.ships", m.BattleSprites.Ships, true,
		modernShipFrames)
	if err != nil {
		return nil, err
	}
	return &modernIconTheme{
		normal: normal, winter: winter, sprites: sprites,
		combat: combat, monsters: monsters, ships: ships,
	}, nil
}

func validateModernIconManifest(m modernIconManifest) error {
	switch {
	case m.Schema != modernIconSchema:
		return fmt.Errorf("Modern Icon manifest schema = %d，預期 %d", m.Schema, modernIconSchema)
	case m.ID != "modern-icon":
		return fmt.Errorf("Modern Icon manifest id = %q，預期 modern-icon", m.ID)
	case m.FrameWidth != modernIconWidth || m.FrameHeight != modernIconHeight:
		return fmt.Errorf("Modern Icon frame = %dx%d，預期最終呈現尺寸 %dx%d",
			m.FrameWidth, m.FrameHeight, modernIconWidth, modernIconHeight)
	case len(m.Tiles.Normal)+len(m.Tiles.Winter)+len(m.Sprites)+
		len(m.BattleSprites.Combat)+len(m.BattleSprites.Monsters)+
		len(m.BattleSprites.Ships) == 0:
		return fmt.Errorf("Modern Icon manifest 至少要列一張已重畫材質")
	}
	for label, set := range map[string]struct {
		entries map[string]string
		frames  int
	}{
		"normal":                 {m.Tiles.Normal, modernTerrainFrames},
		"winter":                 {m.Tiles.Winter, modernTerrainFrames},
		"sprites":                {m.Sprites, modernTerrainFrames},
		"battleSprites.combat":   {m.BattleSprites.Combat, modernCombatFrames},
		"battleSprites.monsters": {m.BattleSprites.Monsters, modernMonsterFrames},
		"battleSprites.ships":    {m.BattleSprites.Ships, modernShipFrames},
	} {
		for key, name := range set.entries {
			if _, err := parseModernIconIndexLimit(key, set.frames); err != nil {
				return fmt.Errorf("Modern Icon tiles.%s index %q：%w", label, key, err)
			}
			if name == "" || filepath.Base(name) != name {
				return fmt.Errorf("Modern Icon tiles.%s[%s] 必須是同目錄內的檔名，收到 %q",
					label, key, name)
			}
		}
	}
	return nil
}

func parseModernIconIndex(s string) (byte, error) {
	return parseModernIconIndexLimit(s, modernTerrainFrames)
}

func parseModernIconIndexLimit(s string, frames int) (byte, error) {
	n, err := strconv.ParseUint(s, 0, 8)
	if err != nil || n >= uint64(frames) {
		return 0, fmt.Errorf("必須是 0–%d 的十進位或 0x 十六進位索引", frames-1)
	}
	return byte(n), nil
}

func loadModernIconPNG(path string, allowAlpha bool) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("開啟 %s：%w", path, err)
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("解碼 %s：%w", path, err)
	}
	b := src.Bounds()
	if b.Dx() != modernIconWidth || b.Dy() != modernIconHeight {
		return nil, fmt.Errorf("%s 尺寸 %dx%d，預期 %dx%d",
			path, b.Dx(), b.Dy(), modernIconWidth, modernIconHeight)
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if !allowAlpha && a != 0xffff {
				return nil, fmt.Errorf("%s 在 (%d,%d) 不是完全不透明", path, x-b.Min.X, y-b.Min.Y)
			}
		}
	}
	return src, nil
}

func (t *modernIconTheme) tile(winter bool, index byte) *ebiten.Image {
	if winter {
		return t.winter[index]
	}
	return t.normal[index]
}

func (t *modernIconTheme) sprite(index byte) *ebiten.Image { return t.sprites[index] }
func (t *modernIconTheme) combatSprite(index int) *ebiten.Image {
	return t.combat[byte(index)]
}
func (t *modernIconTheme) monsterSprite(index int) *ebiten.Image {
	return t.monsters[byte(index)]
}
func (t *modernIconTheme) shipSprite(index int) *ebiten.Image {
	return t.ships[byte(index)]
}

// drawModernIconWorld 在 1280×800 最終畫布直接畫 64×56 terrain。
// 這不是把概念稿縮回 32×28；正式素材從一開始就以顯示尺寸重新繪製。
func (a *app) drawModernIconWorld(screen *ebiten.Image) {
	if a.themeID != themeModern || a.modernIcons == nil || !a.modernIconWorldVisible() {
		return
	}
	halfX, halfY := layout.ViewTilesX/2, layout.ViewTilesY/2
	inset := a.viewInset()
	partyGlyph := game.PartyGlyph(a.party.Facing(),
		a.party.X(), a.party.Y(), a.party.Sailing())
	inspected := make(map[[2]byte]bool, len(a.inspectSpots))
	for _, s := range a.inspectSpots {
		inspected[[2]byte{s.X, s.Y}] = true
	}
	for dy := 0; dy < layout.ViewTilesY; dy++ {
		for dx := 0; dx < layout.ViewTilesX; dx++ {
			if !game.ViewVisible(dx, dy, inset) {
				continue
			}
			mx, my := a.party.X()-halfX+dx, a.party.Y()-halfY+dy
			if mx < 0 || mx >= game.MapWidth || my < 0 || my >= game.MapHeight {
				continue
			}
			tile := a.drawTiles[my*game.MapWidth+mx] & 0x7f
			if inspected[[2]byte{byte(mx), byte(my)}] {
				tile = dungeonItemTile
			}
			if dx == halfX && dy == halfY {
				if sprite := a.modernIcons.sprite(partyGlyph); sprite != nil {
					x := (layout.MapOriginX + dx*gfx.EGATileWidth) * scale
					y := (layout.MapOriginY + dy*gfx.EGATileHeight) * scale
					if ground := a.modernIcons.tile(a.useWinter, tile); ground != nil {
						ui.DrawImageAt(screen, ground, x, y)
					} else if ground := a.tileset().Tile(tile); ground != nil {
						ui.DrawImageScaled(screen, ground, x, y, scale)
					}
					ui.DrawImageAt(screen, sprite, x, y)
					continue
				}
				tile = partyGlyph
			}
			img := a.modernIcons.tile(a.useWinter, tile)
			if img == nil {
				continue
			}
			ui.DrawImageAt(screen, img,
				(layout.MapOriginX+dx*gfx.EGATileWidth)*scale,
				(layout.MapOriginY+dy*gfx.EGATileHeight)*scale)
		}
	}
	for _, sp := range a.trapSpots {
		dx, dy := sp.X-a.party.X()+halfX, sp.Y-a.party.Y()+halfY
		if game.ViewVisible(dx, dy, inset) {
			ui.StrokeRect(screen,
				(layout.MapOriginX+dx*gfx.EGATileWidth)*scale,
				(layout.MapOriginY+dy*gfx.EGATileHeight)*scale,
				modernIconWidth, modernIconHeight, trapMarkerColor)
		}
	}
}

func (a *app) modernIconWorldVisible() bool {
	return a.battle == nil && a.sea == nil && a.camp == nil && a.merchant == nil &&
		a.pool == nil && a.dungeon == nil && a.plotGift == nil && a.confirm == nil &&
		a.workshop == nil && !a.showRoster && !a.box.Active()
}

// drawModernIconBattleActors 在最終畫布重畫已核准的戰鬥單位。舊素材把黑底
// 一起覆寫整格，所以不能只疊透明角色：必須先還原該格地形／海面，再畫角色。
func (a *app) drawModernIconBattleActors(screen *ebiten.Image) {
	if a.themeID != themeModern || a.modernIcons == nil {
		return
	}
	if a.sea != nil {
		a.drawModernIconSeaActors(screen)
		return
	}
	if a.battle == nil || a.spInput != nil || a.spells != nil || a.useMenu != nil ||
		(a.summon != nil && !a.summon.placing) {
		return
	}

	cellW, cellH, logicalScale := a.tileMetrics()
	camX, camY := a.battleCam()
	cur := a.battle.Current()
	for _, u := range a.battle.Units() {
		if !u.Alive() {
			continue
		}
		vx, vy := u.X-camX, u.Y-camY
		if vx < 0 || vy < 0 || vx >= layout.ViewTilesX || vy >= layout.ViewTilesY {
			continue
		}
		var sprite *ebiten.Image
		if u.Slot >= game.PlayerSlotStart && u.Slot < game.PlayerSlotEnd {
			member := u.Slot - game.PlayerSlotStart
			if member >= 0 && member < len(a.members) {
				frame := 0x14 + (u.Facing&3)*2
				class := int(a.members[member].Class)
				switch {
				case class > 5:
					frame += 8
				case class > 2:
					frame += 0x10
				}
				sprite = a.modernIcons.combatSprite(frame)
			}
		} else {
			pair := [...]int{6, 4, 0, 2}
			frame := u.SpriteIndex*8 + pair[u.Facing&3] + (a.battle.Round() & 1)
			sprite = a.modernIcons.monsterSprite(frame)
		}
		if sprite == nil {
			continue
		}

		x := (layout.MapOriginX + vx*cellW) * scale
		y := (layout.MapOriginY + vy*cellH) * scale
		tx, ty := camX+vx, camY+vy
		if a.battleTerrain != nil && game.InArena(tx, ty) {
			tile := a.battleTerrain.TileAt(tx, ty) & 0x7f
			if ground := a.modernIcons.tile(a.useWinter, tile); ground != nil {
				ui.DrawImageAt(screen, ground, x, y)
			} else if ground := a.tileset().Tile(tile); ground != nil {
				ui.DrawImageScaled(screen, ground, x, y, logicalScale*scale)
			}
		}
		ui.DrawImageAt(screen, sprite, x, y)
		border := enemyColor
		if u == cur {
			border = markerColor
		} else if u.IsPlayer {
			border = partyColor
		}
		ui.StrokeRect(screen, x, y, modernIconWidth, modernIconHeight, border)
		if a.examine != nil && u.Slot == a.examine.slot() {
			ui.StrokeRect(screen, x, y, modernIconWidth, modernIconHeight, trapMarkerColor)
		}
	}
}

func (a *app) drawModernIconSeaActors(screen *ebiten.Image) {
	b := a.sea
	cellW, cellH, _ := a.tileMetrics()
	camX := b.PlayerShip().X - layout.ViewTilesX/2
	camY := b.PlayerShip().Y - layout.ViewTilesY/2
	pairs := [...]int{6, 4, 0, 2}
	for _, u := range b.Units {
		if !u.Alive() {
			continue
		}
		group := 0
		if u.Kind == game.SeaPirate {
			group = 1
		} else if u.Kind == game.SeaMonster {
			group = 2
		}
		frame := group*8 + pairs[u.Facing&3] + b.Round&1
		sprite := a.modernIcons.shipSprite(frame)
		if sprite == nil {
			continue
		}
		vx, vy := u.X-camX, u.Y-camY
		if vx < 0 || vy < 0 || vx >= layout.ViewTilesX || vy >= layout.ViewTilesY {
			continue
		}
		x := (layout.MapOriginX + vx*cellW) * scale
		y := (layout.MapOriginY + vy*cellH) * scale
		ui.FillRect(screen, x, y, modernIconWidth, modernIconHeight, seaColor)
		ui.DrawImageAt(screen, sprite, x, y)
		border := enemyColor
		if u.Kind == game.SeaPlayer {
			border = partyColor
		}
		ui.StrokeRect(screen, x, y, modernIconWidth, modernIconHeight, border)
	}
}
