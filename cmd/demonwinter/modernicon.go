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
	loadSet := func(label string, entries map[string]string) (map[byte]*ebiten.Image, error) {
		out := make(map[byte]*ebiten.Image, len(entries))
		for key, name := range entries {
			index, err := parseModernIconIndex(key)
			if err != nil {
				return nil, fmt.Errorf("Modern Icon %s index %q：%w", label, key, err)
			}
			src, err := loadModernIconPNG(filepath.Join(dir, name))
			if err != nil {
				return nil, fmt.Errorf("Modern Icon %s[%#02x]：%w", label, index, err)
			}
			out[index] = ebiten.NewImageFromImage(src)
		}
		return out, nil
	}
	normal, err := loadSet("normal", m.Tiles.Normal)
	if err != nil {
		return nil, err
	}
	winter, err := loadSet("winter", m.Tiles.Winter)
	if err != nil {
		return nil, err
	}
	return &modernIconTheme{normal: normal, winter: winter}, nil
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
	case len(m.Tiles.Normal)+len(m.Tiles.Winter) == 0:
		return fmt.Errorf("Modern Icon manifest 至少要列一張已重畫材質")
	}
	for label, entries := range map[string]map[string]string{
		"normal": m.Tiles.Normal, "winter": m.Tiles.Winter,
	} {
		for key, name := range entries {
			if _, err := parseModernIconIndex(key); err != nil {
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
	n, err := strconv.ParseUint(s, 0, 8)
	if err != nil || n >= modernTerrainFrames {
		return 0, fmt.Errorf("必須是 0–%d 的十進位或 0x 十六進位索引", modernTerrainFrames-1)
	}
	return byte(n), nil
}

func loadModernIconPNG(path string) (image.Image, error) {
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
			if a != 0xffff {
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
