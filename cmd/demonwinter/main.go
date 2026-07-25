// Command demonwinter 是《冬之魔》重製引擎的執行檔。
//
// 目前完成度：載入原版地圖與地形圖塊，用原版的可通行性表與時間規則走動。
// 戰鬥、事件、角色、原版 UI 版面都還沒實作 ——
// 這是給規格層做可視化驗證的外殼，不是遊戲本體。
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
)

// 版面。原版 CGA 是 320×200、視野在畫面左側的方框裡。
// 這裡的尺寸是暫定值，尚未對齊原版版面。
const (
	viewTilesX  = 11
	viewTilesY  = 11
	statusWidth = 216
	scale       = 3
)

var markerColor = color.RGBA{0xff, 0xff, 0x55, 0xff}

type app struct {
	world *game.World
	party *game.Party
	clock *game.Clock
	tiles *world.Map

	// canvas 是邏輯解析度的離屏畫布。所有東西先畫在這裡，
	// 最後用 nearest 整數放大貼到視窗 —— 點陣字與 16×16 圖塊
	// 一旦被線性濾波就糊掉，這是唯一能保住像素邊緣的做法。
	canvas *ebiten.Image

	normal, winter *ui.Tileset
	useWinter      bool
	font           *ui.Font

	lastTile byte
	message  string
}

func (a *app) tileset() *ui.Tileset {
	if a.useWinter {
		return a.winter
	}
	return a.normal
}

// 方向鍵：原版是先轉向再前進，同一次按鍵完成兩件事。
var keyFacing = []struct {
	key ebiten.Key
	f   game.Facing
}{
	{ebiten.KeyUp, game.North},
	{ebiten.KeyRight, game.East},
	{ebiten.KeyDown, game.South},
	{ebiten.KeyLeft, game.West},
}

func (a *app) Update() error {
	for _, kf := range keyFacing {
		if !inpututil.IsKeyJustPressed(kf.key) {
			continue
		}
		a.party.Turn(kf.f)
		res, tile, advanced := a.world.Walk(a.party, a.clock)
		a.lastTile = tile

		switch res {
		case game.MoveBlocked:
			a.message = "blocked"
		case game.MoveExitedSubmap:
			a.message = "left submap"
		default:
			a.message = ""
		}
		if advanced {
			a.message = fmt.Sprintf("hour -> %d", a.clock.Hour())
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		a.useWinter = !a.useWinter
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	a.canvas.Clear()
	a.drawWorld(a.canvas)
	a.drawStatus(a.canvas)

	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(scale, scale)
	screen.DrawImage(a.canvas, op)
}

func (a *app) drawWorld(dst *ebiten.Image) {
	ts := a.tileset()
	halfX, halfY := viewTilesX/2, viewTilesY/2

	for dy := 0; dy < viewTilesY; dy++ {
		for dx := 0; dx < viewTilesX; dx++ {
			mx, my := a.party.X()-halfX+dx, a.party.Y()-halfY+dy
			if mx < 0 || mx >= game.MapWidth || my < 0 || my >= game.MapHeight {
				continue
			}
			v, err := a.tiles.TileAt(mx, my)
			if err != nil {
				continue
			}
			img := ts.Tile(v & 0x7f)
			if img == nil {
				continue
			}
			ui.DrawImageAt(dst, img, dx*gfx.TileWidth, dy*gfx.TileHeight)
		}
	}

	// 隊伍位置暫時用方框標示。原版的隊伍 glyph 動畫表（0x210f）尚未解讀，
	// 見 docs/spec/04-movement.md 未解表。
	ui.StrokeRect(dst,
		halfX*gfx.TileWidth, halfY*gfx.TileHeight,
		gfx.TileWidth, gfx.TileHeight, markerColor)
}

var facingName = []string{"N", "E", "S", "W"}

func (a *app) drawStatus(dst *ebiten.Image) {
	x := viewTilesX*gfx.TileWidth + 8
	lines := []string{
		fmt.Sprintf("Hour %2d Day %2d Mon %2d", a.clock.Hour(), a.clock.Day(), a.clock.Month()),
		fmt.Sprintf("Steps %2d  Light %d", a.clock.Steps(), a.clock.Light()),
		fmt.Sprintf("X %2d Y %2d Facing %s", a.party.X(), a.party.Y(), facingName[a.party.Facing()]),
		fmt.Sprintf("Tile %3d  Depth %d", a.lastTile, a.party.Depth()),
		fmt.Sprintf("Set %s", a.tileset().Name()),
		"",
		a.message,
		"",
		"Arrows: move",
		"Tab:    season",
		"Esc:    quit",
	}
	for i, s := range lines {
		a.font.Draw(dst, s, x, 6+i*(a.font.Height()+2))
	}
}

// logicalSize 是邏輯畫布尺寸；視窗是它的整數倍。
func logicalSize() (int, int) {
	return viewTilesX*gfx.TileWidth + statusWidth, viewTilesY * gfx.TileHeight
}

func (a *app) Layout(int, int) (int, int) {
	w, h := logicalSize()
	return w * scale, h * scale
}

func main() {
	dataDir := flag.String("data", "workplace/orig/demwin/DEM_DATA",
		"原版資料目錄（玩家自備合法副本）")
	mapFile := flag.String("map", "MAP1.MAP", "要載入的地圖檔")
	startX := flag.Int("x", 32, "起始 X")
	startY := flag.Int("y", 32, "起始 Y")
	flag.Parse()

	if _, err := os.Stat(*dataDir); err != nil {
		log.Fatalf("找不到原版資料目錄 %s：%v\n"+
			"本專案不散布原版資料，請用 -data 指向你自己的合法副本。", *dataDir, err)
	}

	tables, err := gamedata.LoadTables(filepath.Join(*dataDir, "FILES.DAT"))
	if err != nil {
		log.Fatalf("載入 FILES.DAT：%v", err)
	}
	m, err := world.LoadMap(filepath.Join(*dataDir, *mapFile))
	if err != nil {
		log.Fatalf("載入地圖：%v", err)
	}

	loadSet := func(s gfx.TerrainSet) *ui.Tileset {
		ts, err := gfx.LoadTileset(filepath.Join(*dataDir, string(s)), s)
		if err != nil {
			log.Fatalf("載入圖塊集 %s：%v", s, err)
		}
		return ui.NewTileset(ts)
	}
	// 目前整個 viewer 走 CGA 素材，字型也用 CGA 版保持一致。
	// EGA 兩套素材都已可解碼，之後要做成可切換。
	font, err := ui.LoadCGAFont(filepath.Join(*dataDir, "ASC.FNT"))
	if err != nil {
		log.Fatalf("載入字型：%v", err)
	}

	a := &app{
		world:  game.NewWorld(m, tables),
		party:  game.NewParty(*startX, *startY, game.South, 0),
		clock:  game.NewClock(),
		tiles:  m,
		normal: loadSet(gfx.NormalTiles),
		winter: loadSet(gfx.WinterTiles),
		font:   font,
	}

	lw, lh := logicalSize()
	a.canvas = ebiten.NewImage(lw, lh)

	ebiten.SetWindowSize(lw*scale, lh*scale)
	ebiten.SetWindowTitle("冬之魔 Demon's Winter")

	if err := ebiten.RunGame(a); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
