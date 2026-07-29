// modernpreview 匯出目前的 Modern EGA 調色預覽為 PNG manifest theme，
// 用來端到端驗證候選 atlas loader。輸出仍由使用者自備的原版素材衍生，
// 只能放 /tmp 或 workplace/dump，不得提交或打包。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
)

const (
	terrainFrames = 102
	combatFrames  = 44
	monsterFrames = 240
	shipFrames    = 32
)

type manifest struct {
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

func main() {
	dataDir := flag.String("data", "workplace/orig/demwin/DEM_DATA", "自備原版 DEM_DATA")
	outDir := flag.String("out", "workplace/dump/modern-preview-theme", "輸出目錄（不得提交／打包）")
	overlayDir := flag.String("terrain-overlays", "", "可選的 32×28 DEMON/WINTER 試片目錄")
	overlayIndices := flag.String("overlay-indices", "", "允許覆蓋的十六進位 terrain index，以逗號分隔")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(err)
	}
	normal := tiles(mustTiles(*dataDir, gfx.NormalTiles))
	winter := tiles(mustTiles(*dataDir, gfx.WinterTiles))
	if *overlayDir != "" {
		allowed := parseIndices(*overlayIndices)
		applyTerrainOverlays(*overlayDir, normal, winter, allowed)
	}
	combat := mustSprites(*dataDir, "COMBAT", combatFrames)
	monsters := mustSprites(*dataDir, "MONSTER", monsterFrames)
	ships := mustSprites(*dataDir, "SHIP", shipFrames)

	writeSheet(filepath.Join(*outDir, "terrain-demon.png"), normal, 17)
	writeSheet(filepath.Join(*outDir, "terrain-winter.png"), winter, 17)
	writeSheet(filepath.Join(*outDir, "combat.png"), sprites(combat), 11)
	writeSheet(filepath.Join(*outDir, "monster.png"), sprites(monsters), 20)
	writeSheet(filepath.Join(*outDir, "ship.png"), sprites(ships), 8)

	m := manifest{
		Schema: 1, ID: "modern-ega", Label: "Modern EGA preview export",
		FrameWidth: gfx.EGATileWidth, FrameHeight: gfx.EGATileHeight,
		TerrainFrames: terrainFrames, CombatFrames: combatFrames,
		MonsterFrames: monsterFrames, ShipFrames: shipFrames,
	}
	m.Sheets.Normal = "terrain-demon.png"
	m.Sheets.Winter = "terrain-winter.png"
	m.Sheets.Combat = "combat.png"
	m.Sheets.Monsters = "monster.png"
	m.Sheets.Ships = "ship.png"
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(*outDir, "theme.json"), raw, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("Modern EGA preview PNG theme → %s（衍生原版素材，不得提交／打包）\n", *outDir)
}

func mustTiles(dataDir string, set gfx.TerrainSet) *gfx.Tileset {
	t, err := gfx.LoadTilesetMode(dataDir, set, gfx.ModeEGA)
	if err != nil {
		panic(err)
	}
	if t.Len() != terrainFrames {
		panic(fmt.Errorf("%s 格數 = %d，預期 %d", set, t.Len(), terrainFrames))
	}
	return gfx.ModernizeTileset(t)
}

func mustSprites(dataDir, base string, expected int) *gfx.SpriteSheet {
	s, err := gfx.LoadSpriteSheetMode(dataDir, base, gfx.ModeEGA)
	if err != nil {
		panic(err)
	}
	if s.Len() != expected {
		panic(fmt.Errorf("%s 格數 = %d，預期 %d", base, s.Len(), expected))
	}
	return gfx.ModernizeSpriteSheet(s)
}

func tiles(t *gfx.Tileset) []*image.RGBA {
	out := make([]*image.RGBA, t.Len())
	for i := range out {
		out[i] = t.Tile(byte(i))
	}
	return out
}

func sprites(s *gfx.SpriteSheet) []*image.RGBA {
	out := make([]*image.RGBA, s.Len())
	for i := range out {
		out[i] = s.Frame(i)
	}
	return out
}

func writeSheet(path string, frames []*image.RGBA, cols int) {
	if err := gfx.SavePNG(path, gfx.TileSpriteSheet(frames, cols)); err != nil {
		panic(err)
	}
}

var overlayName = regexp.MustCompile(`^(demon|winter)-([0-9a-fA-F]{2})-.*[.]png$`)

func parseIndices(raw string) map[int]bool {
	if strings.TrimSpace(raw) == "" {
		panic("-terrain-overlays 需要非空的 -overlay-indices；不得默認套用未批准試片")
	}
	out := make(map[int]bool)
	for _, field := range strings.Split(raw, ",") {
		n, err := strconv.ParseUint(strings.TrimSpace(field), 16, 8)
		if err != nil {
			panic(fmt.Errorf("無效 overlay index %q：%w", field, err))
		}
		if int(n) >= terrainFrames {
			panic(fmt.Errorf("overlay index 0x%02x 超出 terrain 0..0x%02x", n, terrainFrames-1))
		}
		out[int(n)] = true
	}
	return out
}

func applyTerrainOverlays(dir string, normal, winter []*image.RGBA, allowed map[int]bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	applied := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := overlayName.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		index64, _ := strconv.ParseUint(match[2], 16, 8)
		index := int(index64)
		if !allowed[index] {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		src, _, decodeErr := image.Decode(f)
		closeErr := f.Close()
		if decodeErr != nil {
			panic(fmt.Errorf("%s：%w", path, decodeErr))
		}
		if closeErr != nil {
			panic(closeErr)
		}
		if src.Bounds().Dx() != gfx.EGATileWidth || src.Bounds().Dy() != gfx.EGATileHeight {
			panic(fmt.Errorf("%s：尺寸 %v，預期 32x28", path, src.Bounds()))
		}
		dst := image.NewRGBA(image.Rect(0, 0, gfx.EGATileWidth, gfx.EGATileHeight))
		draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
		for y := 0; y < gfx.EGATileHeight; y++ {
			for x := 0; x < gfx.EGATileWidth; x++ {
				if dst.RGBAAt(x, y).A != 0xff {
					panic(fmt.Errorf("%s：(%d,%d) 不是完全不透明", path, x, y))
				}
			}
		}
		key := fmt.Sprintf("%s-%02x", match[1], index)
		if applied[key] {
			panic(fmt.Errorf("%s：重複 overlay %s", path, key))
		}
		applied[key] = true
		if match[1] == "demon" {
			normal[index] = dst
		} else {
			winter[index] = dst
		}
		fmt.Printf("overlay %s index 0x%02x ← %s\n", match[1], index, path)
	}
	for index := range allowed {
		for _, season := range []string{"demon", "winter"} {
			key := fmt.Sprintf("%s-%02x", season, index)
			if !applied[key] {
				panic(fmt.Errorf("缺少批准的 overlay %s", key))
			}
		}
	}
}
