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

	"github.com/wicanr2/demon_winter_cht/internal/assets/cjk"
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/i18n"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// scale 是視窗相對於邏輯畫布的整數倍率。版面本身在 internal/ui/layout。
const scale = 2

var (
	markerColor = color.RGBA{0xff, 0xff, 0x55, 0xff}
	// textColor 是中文的前景色。倚天字模是 1bpp，顏色在渲染時決定。
	textColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

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
	font           *ui.MixedFont

	exits  *world.ExitTable
	events *scenario.EventTable
	tables *gamedata.Tables

	// tr 把事件敘述換成中文。查不到就回原文 —— 缺譯在畫面上是英文，看得見。
	tr         *i18n.Translator
	eventsFile string

	members    []game.Character
	showRoster bool

	monsters *gamedata.MonsterTable
	towns    *gamedata.TownTable
	items    *gamedata.ItemTable

	// town 非 nil 時遊戲在城鎮畫面，地圖與戰鬥輸入都停止。
	town *townScreen
	rng  *rng.RNG

	// prayChance 是祈禱的成功率，會隨每次祈禱變動，跨戰鬥保留。
	prayChance int

	// save／savePath 是進度存檔。savePath 預設不是原版資料目錄 ——
	// 玩家的原版 PARTY.DAT 是他自己的合法副本，不該被遊玩進度蓋掉。
	save     *scenario.SaveGame
	savePath string
	// quitting 為真時畫面停在離開確認框。
	quitting bool

	// battle 非 nil 時遊戲進入戰鬥模式，地圖輸入停止。
	battle     *game.Battle
	log        []string
	pendingIDs []int

	box *ui.TextBox

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
	if handled, err := a.updateQuitDialog(); handled || err != nil {
		return err
	}
	if a.town != nil {
		return a.updateTown()
	}
	if a.battle != nil {
		return a.updateBattle()
	}

	// 文字視窗開著時吃掉所有輸入，只認翻頁鍵 —— 與原版一樣，
	// 讀完敘述才能繼續走。
	if a.box.Active() {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
			inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			a.box.Advance()
			// 敘述讀完才開打。
			if !a.box.Active() && a.pendingIDs != nil {
				a.startBattle(a.pendingIDs)
				a.pendingIDs = nil
			}
		}
		// ESC 只關掉文字視窗，不結束遊戲。
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			for a.box.Active() {
				a.box.Advance()
			}
			a.pendingIDs = nil
		}
		return nil
	}

	for _, kf := range keyFacing {
		if !inpututil.IsKeyJustPressed(kf.key) {
			continue
		}
		a.party.Turn(kf.f)
		res, tile, advanced := a.world.Walk(a.party, a.clock)
		a.lastTile = tile

		switch res {
		case game.MoveBlocked:
			a.message = "前方無法通行"
		case game.MoveExitedSubmap:
			a.message = "離開子地圖"
		default:
			a.message = ""
		}
		if advanced {
			a.message = fmt.Sprintf("時間來到 %d 時", a.clock.Hour())
		}
		if res == game.MoveOK {
			a.checkEvent(tile)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		a.useWinter = !a.useWinter
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		a.showRoster = !a.showRoster
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		a.openTownPicker()
	}
	// ESC 只收起名冊。離開遊戲一律走 F10（見 save.go）。
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.showRoster = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		a.saveNow()
	}
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	a.canvas.Clear()
	if a.town != nil {
		a.drawTown(a.canvas)
		if a.quitting {
			a.drawQuitDialog(a.canvas)
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
	// 戰鬥有自己的戰場，不畫世界地圖 —— 把大地圖留在底下會讓
	// 「單位站在哪一格」完全看不出來。
	if a.battle == nil {
		a.drawWorld(a.canvas)
	}
	switch {
	case a.battle != nil:
		a.drawBattlefield(a.canvas)
		a.drawBattle(a.canvas)
	case a.showRoster:
		a.drawRoster(a.canvas)
	default:
		a.drawStatus(a.canvas)
	}
	if a.battle != nil && !a.box.Active() {
		a.drawBattleCommands(a.canvas)
	}
	ui.DrawMixedTextBox(a.canvas, a.box, a.font, 0, layout.TextBoxTop, markerColor)

	if a.quitting {
		a.drawQuitDialog(a.canvas)
	}

	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(scale, scale)
	screen.DrawImage(a.canvas, op)
}

func (a *app) drawWorld(dst *ebiten.Image) {
	ts := a.tileset()
	halfX, halfY := layout.ViewTilesX/2, layout.ViewTilesY/2

	for dy := 0; dy < layout.ViewTilesY; dy++ {
		for dx := 0; dx < layout.ViewTilesX; dx++ {
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
			ui.DrawImageScaled(dst, img,
				dx*gfx.TileWidth*layout.TileScale, dy*gfx.TileHeight*layout.TileScale, layout.TileScale)
		}
	}

	// 隊伍位置暫時用方框標示。原版的隊伍 glyph 動畫表（0x210f）尚未解讀，
	// 見 docs/spec/04-movement.md 未解表。
	ui.StrokeRect(dst,
		halfX*gfx.TileWidth*layout.TileScale, halfY*gfx.TileHeight*layout.TileScale,
		gfx.TileWidth*layout.TileScale, gfx.TileHeight*layout.TileScale, markerColor)
}

// checkEvent 走 docs/spec/03-events.md 的觸發鏈：
// 落點 tile 決定要不要查表 → 查 EXITS.DAT 取事件索引 → 讀 DATA*.TXT 顯示文字。
//
// 戰鬥（記錄的 Count != 0）與傳送尚未實作，目前只把它們寫進狀態列。
func (a *app) checkEvent(tile byte) {
	idx := -1

	switch game.TriggerFor(tile) {
	case game.TriggerHardBlock, game.TriggerNone:
		return

	case game.TriggerDirectIndex:
		// tile 值本身就是 DATA*.TXT 的記錄索引，不查 EXITS.DAT。
		idx = int(tile)

	case game.TriggerLookup:
		q := game.LookupEvent(a.exits, byte(a.party.X()), byte(a.party.Y()), nil)
		if !q.Found {
			return
		}
		if q.Category == game.CatTeleport {
			a.message = fmt.Sprintf("傳送至 (%d,%d)（未實作）", q.TeleportX, q.TeleportY)
			return
		}
		idx = q.Index
	}

	ev, err := a.events.ByIndex(idx)
	if err != nil {
		a.message = fmt.Sprintf("事件 %d 超出範圍", idx)
		return
	}

	a.box = ui.NewMixedTextBox(a.tr.Event(a.eventsFile, idx, ev.Text))
	// Count != 0 代表這一格帶遭遇；文字讀完才開打。
	a.pendingIDs = nil
	if ev.Count != 0 {
		a.pendingIDs = append([]int(nil), ev.MonsterIDs...)
	}
}

// startBattle 依事件記錄的怪物清單布置戰場。
//
// 怪物的速度與生命在 MONSTER.DAT 的基礎值上做進場擾動。
func (a *app) startBattle(ids []int) {
	var units []*game.Unit

	for i, id := range ids {
		if i >= game.MonsterSlotEnd {
			break
		}
		m, err := a.monsters.ByIndex(id)
		if err != nil {
			continue
		}
		speed, hp := game.RollMonsterStats(a.rng, m.Speed, m.HP)
		units = append(units, &game.Unit{
			Slot: i, Name: m.Name,
			X: 2, Y: i + 1, Facing: int(game.East),
			Speed: speed, Strength: m.Strength, Skill: m.Skill,
			Level: m.Level, Intellect: m.Level,
			HP: hp, MaxHP: hp,
			WeaponIndex:   m.AttackType,
			RaceOrElement: m.Special,
			MaxSP:         m.SP, CurrentSP: m.SP,
		})
	}

	for i, c := range a.members {
		slot := game.PlayerSlotStart + i
		if slot >= game.PlayerSlotEnd {
			break
		}
		units = append(units, &game.Unit{
			Slot: slot, Name: c.Name,
			X: 8, Y: i + 1, Facing: int(game.West),
			Speed:      c.Traits[gamedata.Speed],
			Strength:   c.Traits[gamedata.Strength],
			Skill:      c.Traits[gamedata.Skill],
			Intellect:  c.Traits[gamedata.Intellect],
			Level:      c.Level,
			HP:         c.CurrentHP,
			MaxHP:      c.MaxHP,
			MaxSP:      c.MaxSP,
			CurrentSP:  c.CurrentSP,
			IsPlayer:   true,
			Berserking: c.HasSkill(gamedata.SkillBerserking),
		})
	}

	a.battle = game.NewBattle(a.rng, units)
	a.battle.BeginRound()
	// 祈禱成功率跨戰鬥保留、每次祈禱永久遞減；初值 20% 來自手冊
	// （反組譯只確認了遞減量 −5，初始化位置未逐指令追出）。
	if a.prayChance == 0 {
		a.prayChance = game.PrayInitialChance
	}
	a.log = []string{fmt.Sprintf("第 %d 回合", a.battle.Round())}
}

// logf 把一行訊息推進戰鬥紀錄，只留最後幾行。
func (a *app) logf(format string, args ...any) {
	a.log = append(a.log, fmt.Sprintf(format, args...))
	if len(a.log) > battleLogLines {
		a.log = a.log[len(a.log)-battleLogLines:]
	}
}

const battleLogLines = 8

var facingName = []string{"北", "東", "南", "西"}

func (a *app) drawStatus(dst *ebiten.Image) {
	lines := []string{
		fmt.Sprintf("%2d時 %2d日 %2d月", a.clock.Hour(), a.clock.Day(), a.clock.Month()),
		fmt.Sprintf("步數 %2d  光照 %d", a.clock.Steps(), a.clock.Light()),
		fmt.Sprintf("座標 %2d,%-2d 面向%s", a.party.X(), a.party.Y(), facingName[a.party.Facing()]),
		fmt.Sprintf("地形 %3d  深度 %d", a.lastTile, a.party.Depth()),
		fmt.Sprintf("圖塊 %s", a.tileset().Name()),
		"",
		a.message,
		"",
		"方向鍵：移動",
		"Tab：切換季節",
		"P：隊伍名冊",
		"T：進入城鎮",
		"S：存檔",
		"空白鍵：翻頁",
		"F10：離開遊戲",
	}
	a.font.DrawLines(dst, lines, layout.StatusX, layout.StatusY)
}

var raceName = []string{"人類", "精靈", "矮人", "黑暗精靈", "巨魔"}

var className = []string{
	"遊俠", "聖騎士", "蠻族", "武僧", "牧師",
	"盜賊", "巫師", "術士", "靈視者", "學者",
}

func nameOf(list []string, i int) string {
	if i < 0 || i >= len(list) {
		return "?"
	}
	return list[i]
}

// drawRoster 顯示隊伍五人的基本資料。
//
// 屬性顯示的是天生值（不含裝備加成），與存檔欄位一致。
func (a *app) drawRoster(dst *ebiten.Image) {
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	line("隊伍名冊")
	for _, c := range a.members {
		pts, err := c.RemainingSkillPoints(a.tables)
		if err != nil {
			pts = -1
		}
		line(fmt.Sprintf("%-8s %d級 %s", c.Name, c.Level, nameOf(className, int(c.Class))))
		line(fmt.Sprintf(" %s 生命 %3d/%3d", nameOf(raceName, int(c.Race)), c.CurrentHP, c.MaxHP))
		line(fmt.Sprintf(" 法力 %3d/%3d 未用點數 %d", c.CurrentSP, c.MaxSP, pts))
	}
	line("")
	line("P：返回")
}

func (a *app) Layout(int, int) (int, int) {
	return layout.CanvasWidth * scale, layout.CanvasHeight * scale
}

func main() {
	dataDir := flag.String("data", "workplace/orig/demwin/DEM_DATA",
		"原版資料目錄（玩家自備合法副本）")
	etenDir := flag.String("eten", "workplace/eten",
		"倚天中文字型目錄，需含 STDFONT.15 與 SPCFONT.15（自備）")
	mapFile := flag.String("map", "MAP1.MAP", "要載入的地圖檔")
	dataFile := flag.String("events", "DATA1.TXT", "要載入的事件表")
	savePath := flag.String("save", "workplace/save/PARTY.DAT",
		"進度存檔路徑。刻意不預設在原版資料目錄，免得蓋掉玩家的原版存檔")
	langDir := flag.String("lang", "assets/lang/zh-Hant",
		"翻譯目錄。指向不存在的路徑即為原文模式")
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

	exits, err := world.LoadExits(filepath.Join(*dataDir, "EXITS.DAT"))
	if err != nil {
		log.Fatalf("載入 EXITS.DAT：%v", err)
	}
	events, err := scenario.LoadEventTable(filepath.Join(*dataDir, *dataFile))
	if err != nil {
		log.Fatalf("載入事件表：%v", err)
	}
	monsters, err := gamedata.LoadMonsterTable(filepath.Join(*dataDir, "MONSTER.DAT"))
	if err != nil {
		log.Fatalf("載入怪物表：%v", err)
	}
	towns, err := gamedata.LoadTownTable(*dataDir)
	if err != nil {
		log.Fatalf("載入城鎮表：%v", err)
	}
	items, err := gamedata.LoadItemTable(filepath.Join(*dataDir, "ITEMS.DAT"))
	if err != nil {
		log.Fatalf("載入道具表：%v", err)
	}
	save, fresh, err := loadSave(*savePath, *dataDir)
	if err != nil {
		log.Fatalf("載入隊伍存檔：%v", err)
	}
	if fresh {
		log.Printf("沒有進度存檔，用原版 PARTY.DAT 當起始狀態。存檔會寫到 %s", *savePath)
	}
	members := make([]game.Character, 0, len(save.Characters))
	for _, sc := range save.Characters {
		members = append(members, game.FromSave(sc))
	}

	tr, err := i18n.Load(*langDir)
	if err != nil {
		log.Fatalf("載入翻譯：%v", err)
	}
	// 逐條比對譯文的原文與現在的資料。索引錯位的譯文每一句都通順、
	// 每一句都接錯地方，不自動比對根本看不出來。
	texts := make([]string, 0, events.Len())
	for _, ev := range events.All() {
		texts = append(texts, ev.Text)
	}
	if err := tr.Verify(*langDir, *dataFile, texts); err != nil {
		log.Fatalf("核對翻譯：%v", err)
	}
	if bad := tr.Mismatched(); len(bad) > 0 {
		log.Printf("警告：%d 條譯文的原文與 %s 對不上，那幾條退回英文。"+
			"重跑 `dwstrings events` 更新翻譯檔。", len(bad), *dataFile)
	}

	loadSet := func(s gfx.TerrainSet) *ui.Tileset {
		ts, err := gfx.LoadTileset(filepath.Join(*dataDir, string(s)), s)
		if err != nil {
			log.Fatalf("載入圖塊集 %s：%v", s, err)
		}
		return ui.NewTileset(ts)
	}
	// ASCII 走原版 CGA 字型（8×8），中文走倚天點陣（16×15）。
	// 兩者放進同一套排版格後同高，可以混排。
	ascii, err := ui.LoadCGAFont(filepath.Join(*dataDir, "ASC.FNT"))
	if err != nil {
		log.Fatalf("載入原版字型：%v", err)
	}
	cjkFont, err := cjk.Load(
		filepath.Join(*etenDir, "STDFONT.15"),
		filepath.Join(*etenDir, "SPCFONT.15"))
	if err != nil {
		log.Fatalf("載入倚天中文字型：%v\n"+
			"本專案不散布倚天字型，請用 -eten 指向含 STDFONT.15／SPCFONT.15 的目錄。", err)
	}
	font, err := ui.NewMixedFont(cjkFont, ascii, textColor)
	if err != nil {
		log.Fatalf("建立混排字型：%v", err)
	}

	a := &app{
		world:      game.NewWorld(m, tables),
		party:      game.NewParty(*startX, *startY, game.South, 0),
		clock:      game.NewClock(),
		tiles:      m,
		exits:      exits,
		events:     events,
		tr:         tr,
		eventsFile: *dataFile,
		tables:     tables,
		members:    members,
		monsters:   monsters,
		towns:      towns,
		items:      items,
		rng:        rng.New(),
		normal:     loadSet(gfx.NormalTiles),
		winter:     loadSet(gfx.WinterTiles),
		font:       font,
		save:       save,
		savePath:   *savePath,
	}

	a.canvas = ebiten.NewImage(layout.CanvasWidth, layout.CanvasHeight)

	ebiten.SetWindowSize(layout.CanvasWidth*scale, layout.CanvasHeight*scale)
	ebiten.SetWindowTitle("冬之魔 Demon's Winter")

	if err := ebiten.RunGame(a); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
