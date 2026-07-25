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
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
)

// 版面。原版 CGA 是 320×200、視野在畫面左側的方框裡。
// 這裡的尺寸是暫定值，尚未對齊原版版面。
const (
	viewTilesX  = 11
	viewTilesY  = 11
	statusWidth = 232
	scale       = 2
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

	exits  *world.ExitTable
	events *scenario.EventTable
	tables *gamedata.Tables

	members    []game.Character
	showRoster bool

	monsters *gamedata.MonsterTable
	rng      *rng.RNG

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
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			return ebiten.Termination
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
			a.message = "blocked"
		case game.MoveExitedSubmap:
			a.message = "left submap"
		default:
			a.message = ""
		}
		if advanced {
			a.message = fmt.Sprintf("hour -> %d", a.clock.Hour())
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
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	a.canvas.Clear()
	a.drawWorld(a.canvas)
	switch {
	case a.battle != nil:
		a.drawBattle(a.canvas)
	case a.showRoster:
		a.drawRoster(a.canvas)
	default:
		a.drawStatus(a.canvas)
	}
	ui.DrawTextBox(a.canvas, a.box, a.font, 0, textBoxTop, markerColor)

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
			a.message = fmt.Sprintf("teleport (%d,%d) TODO", q.TeleportX, q.TeleportY)
			return
		}
		idx = q.Index
	}

	ev, err := a.events.ByIndex(idx)
	if err != nil {
		a.message = fmt.Sprintf("event %d out of range", idx)
		return
	}

	a.box = ui.NewTextBox(ev.Text)
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
			X: 2, Y: i + 1,
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
			X: 8, Y: i + 1,
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
	a.log = []string{fmt.Sprintf("Round %d", a.battle.Round())}
}

// logf 把一行訊息推進戰鬥紀錄，只留最後幾行。
func (a *app) logf(format string, args ...any) {
	a.log = append(a.log, fmt.Sprintf(format, args...))
	if len(a.log) > battleLogLines {
		a.log = a.log[len(a.log)-battleLogLines:]
	}
}

const battleLogLines = 8

// updateBattle 推進戰鬥。目前雙方都由簡單 AI 代打，
// 按 Space 逐步執行，方便肉眼核對每一步。
func (a *app) updateBattle() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if !inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return nil
	}

	if out := a.battle.Outcome(); out != game.Ongoing {
		if out == game.Victory {
			a.logf("All monsters dead")
		} else {
			a.logf("Party defeated")
		}
		a.battle = nil
		return nil
	}

	cur := a.battle.Current()
	if cur == nil {
		a.battle.BeginRound()
		a.logf("-- Round %d --", a.battle.Round())
		return nil
	}

	enemies := a.battle.Enemies(cur)
	if len(enemies) == 0 {
		a.battle.EndTurn()
		return nil
	}
	target := a.battle.Unit(enemies[a.rng.Roll(len(enemies))-1])

	res := a.battle.ResolveAttack(cur, target, 0)
	switch {
	case !res.Hit:
		a.logf("%s misses.", cur.Name)
	case res.NoEffect:
		a.logf("%s: no effect on %s", cur.Name, target.Name)
	case res.Killed:
		a.logf("%s kills %s (%d)", cur.Name, target.Name, res.Damage)
	default:
		verb := "hits"
		if res.Critical {
			verb = "CRITS"
		}
		a.logf("%s %s %s for %d", cur.Name, verb, target.Name, res.Damage)
	}
	a.battle.EndTurn()
	return nil
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
		"P:      party",
		"Space:  page text",
		"Esc:    quit",
	}
	for i, s := range lines {
		a.font.Draw(dst, s, x, 6+i*(a.font.Height()+2))
	}
}

// 文字視窗放在地圖與狀態列下方，不疊在上面。
// 高度是 (5 行 + 上下各一行邊距) × 行高。
const textBoxTop = viewTilesY * gfx.TileHeight

// logicalSize 是邏輯畫布尺寸；視窗是它的整數倍。
func logicalSize() (int, int) {
	w := viewTilesX*gfx.TileWidth + statusWidth
	if min := (ui.TextBoxColumns + 2) * ui.CGAGlyphWidth; w < min {
		w = min
	}
	h := textBoxTop + (ui.TextBoxPageLines+2)*(ui.CGAGlyphHeight+2)
	// 名冊一次列五個人、每人三行，比地圖高，畫布要容得下。
	if rosterH := (3*5 + 4) * (ui.CGAGlyphHeight + 2); h < rosterH {
		h = rosterH
	}
	return w, h
}

// drawBattle 畫戰鬥狀態：雙方單位的 HP 與最近幾行紀錄。
func (a *app) drawBattle(dst *ebiten.Image) {
	x := viewTilesX*gfx.TileWidth + 8
	y := 6
	line := func(s string) {
		a.font.Draw(dst, s, x, y)
		y += a.font.Height() + 2
	}

	line(fmt.Sprintf("COMBAT  round %d", a.battle.Round()))
	for _, u := range a.battle.Units() {
		tag := " "
		if u == a.battle.Current() {
			tag = ">"
		}
		state := ""
		if !u.Alive() {
			state = " dead"
		}
		name := u.Name
		if len(name) > 12 {
			name = name[:12]
		}
		line(fmt.Sprintf("%s%-12s%3d/%-3d%s", tag, name, u.HP, u.MaxHP, state))
	}
	line("")
	for _, s := range a.log {
		line(s)
	}
	line("")
	line("Space: step  Esc: quit")
}

var raceName = []string{"Human", "Elf", "Dwarf", "DarkElf", "Troll"}

var className = []string{
	"Ranger", "Paladin", "Barbarian", "Monk", "Cleric",
	"Thief", "Wizard", "Sorcerer", "Visionary", "Scholar",
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
	x := viewTilesX*gfx.TileWidth + 8
	y := 6
	line := func(s string) {
		a.font.Draw(dst, s, x, y)
		y += a.font.Height() + 2
	}

	line("PARTY")
	for _, c := range a.members {
		pts, err := c.RemainingSkillPoints(a.tables)
		if err != nil {
			pts = -1
		}
		line(fmt.Sprintf("%-7s L%-2d %-8s", c.Name, c.Level, nameOf(className, int(c.Class))))
		line(fmt.Sprintf("  %-7s HP %3d/%3d", nameOf(raceName, int(c.Race)), c.CurrentHP, c.MaxHP))
		line(fmt.Sprintf("  SP %3d/%3d  free %d", c.CurrentSP, c.MaxSP, pts))
	}
	line("")
	line("P: back")
}

func (a *app) Layout(int, int) (int, int) {
	w, h := logicalSize()
	return w * scale, h * scale
}

func main() {
	dataDir := flag.String("data", "workplace/orig/demwin/DEM_DATA",
		"原版資料目錄（玩家自備合法副本）")
	mapFile := flag.String("map", "MAP1.MAP", "要載入的地圖檔")
	dataFile := flag.String("events", "DATA1.TXT", "要載入的事件表")
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
	save, err := scenario.LoadSaveGame(filepath.Join(*dataDir, "PARTY.DAT"))
	if err != nil {
		log.Fatalf("載入隊伍存檔：%v", err)
	}
	members := make([]game.Character, 0, len(save.Characters))
	for _, sc := range save.Characters {
		members = append(members, game.FromSave(sc))
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
		world:    game.NewWorld(m, tables),
		party:    game.NewParty(*startX, *startY, game.South, 0),
		clock:    game.NewClock(),
		tiles:    m,
		exits:    exits,
		events:   events,
		tables:   tables,
		members:  members,
		monsters: monsters,
		rng:      rng.New(),
		normal:   loadSet(gfx.NormalTiles),
		winter:   loadSet(gfx.WinterTiles),
		font:     font,
	}

	lw, lh := logicalSize()
	a.canvas = ebiten.NewImage(lw, lh)

	ebiten.SetWindowSize(lw*scale, lh*scale)
	ebiten.SetWindowTitle("冬之魔 Demon's Winter")

	if err := ebiten.RunGame(a); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
