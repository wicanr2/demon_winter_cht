// Command demonwinter 是《冬之魔》重製引擎的執行檔。
//
// 這段原本寫「戰鬥、事件、角色都還沒實作，是給規格層做可視化驗證的外殼」。
// 那已經過期很久了 —— 照 rulebook/63「以程式碼為準，別信過期的狀態標記」訂正。
//
// # 目前能玩到什麼
//
//   - 走動：原版地圖（含 SUM.MAP 的世界子地圖）、可通行性表、困難地形、
//     日夜與時間推進、常態／雪地圖塊切換
//   - 事件：`nSS.DAT` 查表與直接索引兩條觸發路徑、敘述文字、傳送、遭遇
//   - 戰鬥：行動點、移動與轉向、攻擊、法術（含範圍法術與選點游標）、
//     驅散不死、祈禱、汲取法力、閃避、道具、戰場地形與視線遮蔽
//   - 城鎮：走到城鎮格自動進城、七種設施、市集議價
//   - 角色：建立角色（種族／擲點／重擲／職業／姓名／隊伍位置）、存檔
//   - 中文化：倚天點陣字混排、全部介面與 148 條字串
//   - 音效：原版 PC speaker 音效
//
// # 已知還沒對齊原版的地方
//
//   - **版面不是原版版面**。這裡是自訂的 640×400（中文需要 16×16 點陣才可讀），
//     不是原版 320×200 的復刻。
//   - 怪物 AI 的決策樹、選法術、挑目標與噴吐都照原版（見 docs/re/23），
//     但走位（轉向、逼近）那一段原版還沒讀，目前是自己補的。
//
// 完整的未解清單見 CONTEXT.md 與各 docs/spec 檔案末尾。
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/cjk"
	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/gfx"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/i18n"
	"github.com/wicanr2/demon_winter_cht/internal/manual"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
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

	// drawTiles 是「拿來畫」的那份 tile 陣列：與 tiles 相同，但海面已經
	// 隨機摻過另一種浪花（見 world.OceanDither）。原版是在載入時就地改寫
	// 地圖緩衝區，這裡另存一份，讓 tiles 保持檔案解出來的原樣 —— 替換是
	// 純外觀的（兩個海面 tile 可通行性相同），規則判定一律走 tiles。
	drawTiles []byte

	// canvas 是邏輯解析度的離屏畫布。所有東西先畫在這裡，
	// 最後用 nearest 整數放大貼到視窗 —— 點陣字與 16×16 圖塊
	// 一旦被線性濾波就糊掉，這是唯一能保住像素邊緣的做法。
	canvas *ebiten.Image

	// speaker 播原版的 PC speaker 音效。原版沒有配樂，只有這幾段短音效。
	speaker *ui.Speaker

	normal, winter *ui.Tileset
	useWinter      bool
	font           *ui.MixedFont

	// exits 是 EXITS.DAT。**事件查表已經不用它了** —— 那是 `docs/re/05` §1.3
	// 誤判造成的（見 special 欄位）。留著是因為 EXITS.DAT 本身確實存在、
	// 而且是 6-byte 記錄，只是它的消費端還沒解出來（`docs/re/77` §3）。
	exits *world.ExitTable

	// special 是每張子地圖的特殊格清單（`nSS.DAT`）—— 事件與傳送的真正來源。
	// key 是子地圖編號（1–5）。沒有這張表的子地圖查不到就是沒有事件。
	special map[int]*scenario.SpecialTiles

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
	// title 非 nil 時停在開場標題畫面。
	title *ebiten.Image
	// camp 非 nil 時遊戲在紮營畫面。
	camp *campScreen

	// merchant 非 nil 時遊戲在商隊畫面（見 merchantui.go）。
	merchant *merchantScreen
	// create 非 nil 時遊戲在建角畫面。
	create *createScreen
	// spells 非 nil 時戰鬥中的施法選單開著。
	spells *spellMenu
	// spInput 非 nil 時「投入多少法力」的輸入框開著，疊在施法選單上。
	spInput *spPrompt
	// useMenu 非 nil 時戰鬥中的使用道具選單開著。
	useMenu *itemMenu
	// aoe 非 nil 時正在選範圍法術的中心點，aoeX/aoeY 是游標位置。
	aoe *aoeCursor
	// breath 是噴吐動畫（純呈現層，不擋輸入）。
	breath *breathAnim
	// debugMerchantLies 覆蓋商隊的說謊機率，負值代表照原版擲。
	debugMerchantLies int
	aoeX, aoeY        int
	// strings 是 FILES.DTT 字串池，法術名稱從這裡來。
	strings *gamedata.StringPool
	rng     *rng.RNG

	// prayChance 是祈禱的成功率，會隨每次祈禱變動，跨戰鬥保留。
	prayChance int

	// save／savePath 是進度存檔。savePath 預設不是原版資料目錄 ——
	// 玩家的原版 PARTY.DAT 是他自己的合法副本，不該被遊玩進度蓋掉。
	save     *scenario.SaveGame
	savePath string
	// quitting 為真時畫面停在離開確認框。
	quitting bool

	// battle 非 nil 時遊戲進入戰鬥模式，地圖輸入停止。
	battle *game.Battle
	// settled 代表這場戰鬥的結算（訊息、戰利品）已經做過了。
	// 結算要在「等玩家按空白鍵」之前完成，不然訊息只存在一幀。
	settled bool

	// pendingChain 是續接碼 3 的第二段文字（`docs/re/02` §2.4 [F]）。
	// 第一段讀完才顯示，顯示完才開打。
	pendingChain string

	// manual／manualUI 是遊戲內手札（`F2`）—— 原版把這些資料印在紙本手冊上，
	// 這裡搬進遊戲。manualUI 非 nil 時手札畫面開著。
	manual   *manual.Manual
	manualUI *manualScreen

	// runeBox 非 nil 代表正在顯示符文密語（`docs/re/72`）。
	runeBox *runeScreen
	// runeFont 是 CYPHER.SHP 的 27 個符文字形；讀不到時為 nil。
	runeFont []*ebiten.Image

	// ditherSeed 是海面浪花的種子。神殿變成廢墟時要用同一顆種子重建
	// drawTiles，不然海面會整片重擲、看起來像畫面壞掉。
	ditherSeed uint16
	// dreamText 是 T.TXT 的三場夢（`docs/re/82`）；讀不到時為 nil。
	dreamText *scenario.StoryText
	// dreamPage 非負時螢幕上正在播夢。
	dreamPage int

	// eregoreText 是 EREGORE.TXT（`docs/re/82`）；讀不到時為 nil。
	eregoreText *scenario.StoryText
	// eregore 非 nil 時螢幕上正在播艾瑞戈爾那一場（`docs/re/83`）。
	eregore *eregoreScreen
	// riddle 非 nil 時螢幕上正在作答密語謎題（`docs/re/84`）。
	riddle *riddleScreen

	// trace 非 nil 時把每一次狀態變化寫進軌跡檔（`-trace`）。
	// 這是 A4 全程試玩的驗收工具，見 trace.go。
	trace *tracer
	// auto 非 nil 時由自動戰鬥驅動代打（`-autofight`），見 autofight.go。
	auto *autoFighter

	// winText 是 WIN.TXT 的結局文字（`docs/re/82`）；讀不到時為 nil。
	winText *scenario.StoryText
	// ending 是結局序列的播放狀態。
	ending *endingScreen

	// won 在禁錮成功後為 true —— 遊戲通關（`docs/re/61`）。
	// 原版此時跳結局畫面（`0x07175`，"CONGRATULATIONS! You have won
	// Demon's Winter."）；本專案先顯示訊息，結局畫面另做。
	won bool

	// mapID 是目前所在的子地圖編號。11–77 是世界地圖，10 以下是地城 ——
	// 兩者的戰場視野來源不同（時辰表 vs 光源），見 terrainForBattle。
	mapID int

	// torch 是地城的光源強度，載入存檔時從隊伍欄位 +0xa7 取得。
	// 目前沒有點火把／照明術之類會改動它的機制，所以整場遊戲維持初值。
	torch byte

	// battleTerrain 是這場戰鬥腳下的地形，已套用視線遮蔽 ——
	// 看不到的格子是空白的。原版的戰場就是大地圖的局部放大，
	// 看得到多大一塊由時辰決定（見 game.NewBattleTerrain）。
	battleTerrain *game.BattleTerrain
	log           []string
	pendingIDs    []int

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
	// 軌跡在每一幀結束時取樣。放在最外層而不是各畫面裡，
	// 是為了不必在二十個 update 函式各插一行 —— 漏掉一個就會有一段黑洞。
	defer func() { a.trace.state(a.traceState()) }()
	return a.update()
}

func (a *app) update() error {
	if handled, err := a.updateQuitDialog(); handled || err != nil {
		return err
	}
	// 結局要排在所有畫面之前 —— 破關之後不該還能回去紮營。
	if a.won {
		return a.updateEnding()
	}
	if a.title != nil {
		return a.updateTitle()
	}
	if a.dreamPage >= 0 {
		return a.updateDream()
	}
	if a.eregore != nil {
		return a.updateEregore()
	}
	if a.riddle != nil {
		return a.updateRiddle()
	}
	if a.manualUI != nil {
		return a.updateManual()
	}
	if a.runeBox != nil {
		return a.updateRuneBox()
	}
	if a.create != nil {
		return a.updateCreate()
	}
	if a.town != nil {
		return a.updateTown()
	}
	if a.battle != nil {
		return a.updateBattle()
	}
	if a.camp != nil {
		return a.updateCamp()
	}
	if a.merchant != nil {
		return a.updateMerchant()
	}

	// 文字視窗開著時吃掉所有輸入，只認翻頁鍵 —— 與原版一樣，
	// 讀完敘述才能繼續走。
	if a.box.Active() {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
			inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			a.box.Advance()
			if !a.box.Active() {
				// 續接碼 3：同一次事件處理內再顯示一段文字
				// （原版 `+0xa5 == 3` 帶 param_2=1 重跑迴圈，
				// 顯示 field F 跳過開頭 '3' 之後的內容，見 `docs/re/02` §2.4 [F]）。
				// **排在開打之前** —— 那是事件處理函式內的事，戰鬥由外層驅動。
				if a.pendingChain != "" {
					a.box = ui.NewMixedTextBox(a.pendingChain)
					a.pendingChain = ""
				} else if a.pendingIDs != nil {
					// 敘述讀完才開打。
					a.startBattle(a.pendingIDs)
					a.pendingIDs = nil
				}
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
		// 光之環的門：三個符印沒解完就擋下（`docs/re/59` §3）。
		// 原版是走進去之後才擋並把玩家推開；這裡在移動前擋，
		// 效果一樣而且不必實作推開 —— 差異記在 docs/re/62。
		if a.mapID == game.ImprisonSubMap && !game.CircleOfLightOpen(a.save.GlyphFlags) {
			a.message = a.tr.UI("plot.forcefield", "緋紅的力場擋住了通往光之環的路")
			return nil
		}
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
			a.stepBoat(tile)
			a.stepHPTick()
			a.checkEvent(tile)
			a.checkMerchantEncounter()
			a.checkRandomEncounter(tile)
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
	// B 是偵錯用的「就地開打」，和 T 鍵直接進城同一個性質：戰鬥只在
	// 特定事件格才觸發，沒有這個就沒辦法在任意地形上驗戰場畫面。
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		a.startBattle(debugBattleMonsters)
	}
	// C：紮營 —— **與原版一致**（手冊 §407：「隨時都可按 `C` 紮營休息」，
	// DOSBox 實跑也驗過，見 `docs/re/81` §4）。這裡原本是 `R`，
	// 因為舊註解寫「原版用哪個鍵沒查」。
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		a.openCamp()
	}
	// F2：手札。原版沒有這個畫面（那些資料印在紙本手冊上），見 manualui.go。
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		a.openManual()
	}
	// F1 是建立角色。原版把它放在遊戲外的 Character Utilities 選單，
	// 大地圖上根本沒有這個鍵 —— 所以挪到 F1 這種一看就是「不在原版裡」
	// 的位置，把 `C` 讓回給紮營。
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		a.openCreate()
	}
	// M 是偵錯用的「就地遇到商隊」—— 遭遇觸發還沒解（見 merchantui.go）。
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		a.openMerchant()
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
	if a.won {
		a.drawEnding(a.canvas)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
	if a.runeBox != nil {
		a.drawRuneBox(a.canvas)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
	if a.title != nil {
		a.drawTitle(a.canvas)
		if a.quitting {
			a.drawQuitDialog(a.canvas)
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
	if a.dreamPage >= 0 {
		a.canvas.Fill(color.Black)
		a.drawDream(a.canvas)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
	if a.eregore != nil {
		a.canvas.Fill(color.Black)
		a.drawEregore(a.canvas)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
	if a.riddle != nil {
		a.canvas.Fill(color.Black)
		a.drawRiddle(a.canvas)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
	if a.manualUI != nil {
		a.canvas.Fill(color.Black)
		a.drawManual(a.canvas)
		if a.quitting {
			a.drawQuitDialog(a.canvas)
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
	if a.create != nil {
		a.drawCreate(a.canvas)
		if a.quitting {
			a.drawQuitDialog(a.canvas)
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(a.canvas, op)
		return
	}
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
	// 戰鬥與紮營有自己的畫面，不畫世界地圖 —— 把大地圖留在底下，
	// 文字會疊在圖塊上完全讀不了。
	if a.battle == nil && a.camp == nil && a.merchant == nil {
		a.drawWorld(a.canvas)
	}
	switch {
	case a.battle != nil && a.spInput != nil:
		a.drawSPPrompt(a.canvas)
	case a.battle != nil && a.spells != nil:
		a.drawSpellMenu(a.canvas)
	case a.battle != nil && a.useMenu != nil:
		a.drawItemMenu(a.canvas)
	case a.battle != nil:
		a.drawBattlefield(a.canvas)
		a.drawBattle(a.canvas)
	case a.camp != nil:
		a.drawCamp(a.canvas)
	case a.merchant != nil:
		a.drawMerchant(a.canvas)
	case a.showRoster:
		a.drawRoster(a.canvas)
	default:
		a.drawStatus(a.canvas)
	}
	if a.battle != nil && a.spells == nil && a.spInput == nil && a.useMenu == nil && !a.box.Active() {
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

// loadMapArg 解讀 -map：純數字當成 SUM.MAP 的子地圖編號，其餘當檔名。
//
// 世界地圖（含 25 座城鎮裡的 24 座）全部打包在 SUM.MAP 裡，只認檔名的話
// 玩家根本走不到大陸上，自動進城也就沒地方驗。
func loadMapArg(dataDir, arg string) (*world.Map, int, error) {
	id, err := strconv.Atoi(arg)
	if err != nil {
		// 檔名形式：MAP1.MAP → 編號 1（都是地城）。
		m, err := world.LoadMap(filepath.Join(dataDir, arg))
		if err != nil {
			return nil, 0, err
		}
		n := 0
		if len(arg) > 3 && arg[3] >= '0' && arg[3] <= '9' {
			n = int(arg[3] - '0')
		}
		return m, n, nil
	}
	sm, err := world.LoadSumMap(filepath.Join(dataDir, "SUM.MAP"))
	if err != nil {
		return nil, 0, err
	}
	m, ok := sm.Segment(id)
	if !ok {
		return nil, 0, fmt.Errorf("SUM.MAP 沒有子地圖 %d，可用的是 %v", id, sm.IDs())
	}
	return m, id, nil
}

// ditheredTiles 產生「拿來畫」的那份 tile 陣列（見 app.drawTiles）。
//
// seed 為 0 時用原版的初始種子；`-seed` 有指定就拿它當浪花種子，
// 讓截圖比對可重現。
func ditheredTiles(m *world.Map, seed uint16, templeRuins byte) []byte {
	src := m.Tiles()
	out := append([]byte(nil), src[:]...)
	d := world.NewOceanDither()
	if seed != 0 {
		d = world.NewOceanDitherSeed(seed)
	}
	d.Apply(out)
	applyTempleRuins(out, templeRuins)
	return out
}

// applyTempleRuins 把神殿 tile 換成廢墟 tile —— 冬之魔降臨之後神殿全毀
// （`docs/re/79`）。
//
// 原版是在**繪製**時逐格替換（`0x1739a`：讀地圖 tile、若旗標過門檻且
// tile == `0x25` 就改寫成 `0x5b`，再寫進繪製緩衝區），規則判定用的地圖
// 陣列不動。這裡沿用同一個分工 —— 改 drawTiles、不改 a.tiles，
// 跟海面浪花是同一種「純外觀替換」。
//
// ⚠ 目前只在啟動時套一次。旗標是單向的（世界壞掉就回不去），
// 而且引擎裡還沒有任何地方會設它，所以夠用；真的接上劇情觸發時
// 這裡要改成旗標變動時重建。
func applyTempleRuins(tiles []byte, templeRuins byte) {
	if templeRuins <= scenario.TempleRuinsThreshold {
		return
	}
	for i, t := range tiles {
		if t&0x7f == templeTile {
			tiles[i] = tiles[i]&0x80 | ruinsTile
		}
	}
}

// templeTile／ruinsTile 是神殿與廢墟的 tile 值。
const (
	templeTile = 0x25
	ruinsTile  = 0x5b
)

func (a *app) drawWorld(dst *ebiten.Image) {
	ts := a.tileset()
	halfX, halfY := layout.ViewTilesX/2, layout.ViewTilesY/2

	for dy := 0; dy < layout.ViewTilesY; dy++ {
		for dx := 0; dx < layout.ViewTilesX; dx++ {
			mx, my := a.party.X()-halfX+dx, a.party.Y()-halfY+dy
			if mx < 0 || mx >= game.MapWidth || my < 0 || my >= game.MapHeight {
				continue
			}
			img := ts.Tile(a.drawTiles[my*game.MapWidth+mx] & 0x7f)
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
// checkRandomEncounter 擲野外隨機遭遇。
//
// 原版每走一步 1/64（`rnd_raw() & 0x3f == 0x34`），只在戶外、且不在船上時擲；
// 遇到什麼由腳下 tile 的地形決定（見 game.RollEncounter）。
//
// 目前的難度取隊伍最高等級，鉗在 1–10。原版存在 `ds:0x5c60`，會隨劇情推進
// 被加值（`222f:2b5f`），那條加值路徑的觸發條件還沒追出來，所以先用隊伍等級
// 近似 —— 這是**本作自己的取捨**，不是原版行為。
// checkMerchantEncounter 擲原版那道 1/64（`222f:081b`，見 `docs/re/51`）。
//
// **這一擲的回傳碼是 `0x17`，而動作分派表的 `case 0x17` 是商隊** ——
// 本專案原本把它接到隨機戰鬥上，接錯了。條件與戰鬥那條一樣：
// 戶外（子地圖編號 >= 9）、而且不在別的畫面裡。
func (a *app) checkMerchantEncounter() {
	if a.mapID < worldMapMinID || a.battle != nil || a.box.Active() ||
		a.merchant != nil || a.camp != nil {
		return
	}
	if !game.EncounterTriggered(int(a.rng.Next())) {
		return
	}
	a.openMerchant()
}

// checkRandomEncounter 走隨機戰鬥的倒數計時器（存檔 `+0x9c`）。
//
// **不是機率是倒數**（`docs/re/51` §3）：走一步減一，歸零時主迴圈
// 回傳動作碼 `0x16` 去挑怪，然後重設成 28–77 步。
// 每走一步就要減，所以這一支在地城裡也照跑，只有「要不要開打」看地形。
func (a *app) checkRandomEncounter(tile byte) {
	if a.battle != nil || a.box.Active() || a.merchant != nil {
		return
	}
	left, fight := game.StepEncounterCountdown(int(a.save.EncounterCountdown))
	a.save.EncounterCountdown = byte(left)
	if !fight {
		return
	}
	a.save.EncounterCountdown = byte(game.EncounterCountdownAfterBattle(a.rng))
	if a.mapID < worldMapMinID {
		return // 地城不開野外遭遇（挑怪表是照地形查的）
	}
	terrain, ok := a.tables.Terrain(tile)
	if !ok {
		return
	}
	mons := game.RollEncounter(a.rng, a.tables, terrain, a.encounterLevel())
	if len(mons) == 0 {
		return
	}
	a.logf("在%s遭遇了敵人", terrain.Name())
	a.startBattle(mons)
}

// worldMapMinID 是「戶外」的下界。原版拿子地圖編號跟 9 比（222f:080c），
// 小於 9 就是地城，不擲隨機遭遇。
//
// 注意這裡是 9 而不是別處用的 10 —— 原版自己在兩個地方用了不同的邊界
// （光照那條是 `< 10`）。照抄，不統一。
const worldMapMinID = 9

// debugGiveItem 把一件有效果的道具塞進第一名隊員的空格。
//
// 純偵錯用。原版的道具效果是掉寶時生成的（`FUN_1990_????` 依 ITEMS.DAT 的
// 四個候選類別擲兩層骰，見 docs/re/25），那條路徑還沒實作。
func (a *app) debugGiveItem(spec string) error {
	f := strings.Split(spec, ",")
	if len(f) != 4 {
		return fmt.Errorf("要四個欄位（type,effect,power,次數），拿到 %q", spec)
	}
	n := make([]int, 4)
	for i, part := range f {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return fmt.Errorf("第 %d 個欄位 %q 不是數字", i+1, part)
		}
		n[i] = v
	}
	if len(a.members) == 0 {
		return fmt.Errorf("隊伍是空的")
	}
	c := &a.members[0]
	for i := range c.Inventory {
		if !c.Inventory[i].Empty() {
			continue
		}
		c.Inventory[i] = scenario.InventorySlot{
			Type: byte(n[0]), Effect: n[1], Power: n[2],
			Total: n[3], Identified: true,
		}
		log.Printf("偵錯：給 %s 第 %d 格 type=%d effect=%d power=%d ×%d",
			c.Name, i, n[0], n[1], n[2], n[3])
		return nil
	}
	return fmt.Errorf("%s 的道具欄滿了", c.Name)
}

// gold／setGold 存取隊伍金幣。
//
// 存檔那一格的長度未定案（3 或 4 bytes，見 scenario 的 goldOffset 註解），
// 這裡沿用解析端的 3-byte 讀法，第 4 個 byte 原封不動留在 TrailerRaw。
func (a *app) gold() int { return a.save.Gold }

func (a *app) setGold(v int) {
	if v < 0 {
		v = 0
	}
	a.save.Gold = v
}

// encounterLevel 回傳目前的遭遇難度（1–10）。
func (a *app) encounterLevel() int {
	level := 1
	for _, c := range a.members {
		if c.Level > level {
			level = c.Level
		}
	}
	if level > 10 {
		level = 10
	}
	return level
}

func (a *app) checkEvent(tile byte) {
	idx := -1

	switch game.TriggerFor(tile) {
	case game.TriggerHardBlock, game.TriggerNone:
		return

	case game.TriggerSite:
		// 地點 tile：城鎮／神殿／學院／廢墟。**不是文字索引**
		// —— 舊版把 tile 值當 DATA*.TXT 的記錄索引，見 `docs/re/79`。
		switch game.SiteFor(tile, a.save.TempleRuins, a.save.ShardShattered) {
		case game.SiteTown:
			a.enterTownAt(a.party.X(), a.party.Y())
		case game.SiteRuins:
			a.message = a.tr.UI("site.ruins", "隊伍走過一片廢墟")
		case game.SiteTemple, game.SiteCollege:
			// 世界地圖上的神殿與學院還沒接（`docs/re/74` 的 35 所學院、
			// `docs/re/19` 的神殿三項服務目前只從城鎮進得去）。
			// **明確標記未接**，不要靜默落到別的分支假裝有反應。
			a.message = "（世界地圖上的設施還沒接）"
		}
		return

	case game.TriggerLookup:
		// 查的是這張子地圖的 `nSS.DAT`，不是 EXITS.DAT。
		// 之前餵 EXITS.DAT 是照 `docs/re/05` §1.3 的誤判做的，而那個檔
		// 其實是 6-byte 記錄 —— 用 3-byte 切開，每一筆的座標與類別都是錯的
		// （`docs/re/77` §3）。
		st := a.special[a.mapID]
		if st == nil {
			return
		}
		hit := st.Lookup(byte(a.party.X()), byte(a.party.Y()))
		if hit == nil {
			return
		}
		if hit.Teleport {
			a.party.TeleportTo(int(hit.Dest.X), int(hit.Dest.Y))
			a.message = fmt.Sprintf("被傳送到 (%d,%d)", hit.Dest.X, hit.Dest.Y)
			return
		}
		// 類別 5 走完全不同的一條：那張 16 格地點劇情表
		// （原版 `cmp ds:0x5c62,5`，`docs/re/83` §1）。
		// **不能落到下面的 DATA*.TXT 路徑** —— 它的「值」是 case 編號，
		// 不是文字索引，餵下去會顯示完全無關的房間敘述。
		if c := hit.Tile.PlotCase(); c >= 0 {
			a.locationPlot(c)
			return
		}
		// 類別 3／6 是陷阱（`0x19a4b`）。原版印完 "A trap!" 之後
		// 還是會走文字路徑，所以這裡只多一則訊息、不 return。
		if cls := hit.Tile.Class(); cls == scenario.SpecialClassTrap ||
			cls == scenario.SpecialClassTrapAlt {
			a.message = a.tr.UI("site.trap", "有陷阱！")
		}
		idx = hit.EventIndex
		// 記錄「看過了」：類別留著，記錄之後照樣命中（`docs/re/78` §3）。
		// ⚠ 這個改動目前只留在記憶體 —— 存回 `nSS.DAT` 還沒做（見 CONTEXT §7 A2）。
		st.MarkVisited(hit.Index)
	}

	a.showEvent(idx)
}

// showEvent 顯示一筆事件：文字 → 續接碼第二段 → 開打。
//
// 從 checkEvent 抽出來，因為觸發（座標／tile）與顯示是兩件事 ——
// 抽開之後 `-event=N` 偵錯旗標才驗得到顯示路徑的每個分支。
func (a *app) showEvent(idx int) {
	ev, err := a.events.ByIndex(idx)
	if err != nil {
		a.message = fmt.Sprintf("事件 %d 超出範圍", idx)
		return
	}

	// 符文密語走自己的畫面（`docs/re/72`）—— 它是圖不是文字，
	// 塞進一般文字框會變成一串看不懂的 ASCII。
	if ev.IsRuneGlyph() {
		a.openRuneBox(ev.RuneText())
		return
	}
	a.box = ui.NewMixedTextBox(a.tr.Event(a.eventsFile, idx, ev.Text))

	// 續接碼 3 的第二段（例如 "3With Remondadin dead..."）。
	// `IsChainRedraw()` 早就實作了，但一直**沒有呼叫端** ——
	// 與符文（`docs/re/72`）是同一類「解碼做完沒接上」的洞。
	a.pendingChain = ""
	if ev.IsChainRedraw() {
		orig := ev.ChainRedrawText()
		// 名稱型 key —— 第二段不是獨立記錄，沒有自己的索引
		// （`cmd/dwstrings` 產生的是 `chain.<檔名>.<索引>`）。
		a.pendingChain = a.tr.UI(
			fmt.Sprintf("chain.%s.%d", strings.ToUpper(a.eventsFile), idx), orig)
	}

	// Count != 0 代表這一格帶遭遇；文字讀完才開打。
	a.pendingIDs = nil
	if ev.Count != 0 {
		a.pendingIDs = append([]int(nil), ev.MonsterIDs...)
	}
}

// debugBattleMonsters 是 B 鍵開的測試戰鬥用的怪物（MONSTER.DAT 索引）。
//
// 三隻低階的（夠驗畫面又不會一回合把隊伍打光），外加索引 1
// 「Lvl 1 mage」—— 它有法力，AI 施法那條路徑才驗得到。
var debugBattleMonsters = []int{2, 3, 4, 1}

// terrainForBattle 切出這場戰鬥腳下的地形，並套用視線遮蔽。
//
// 看得到多大一塊由時辰決定：正午整個 9×9 都畫得出來，深夜只剩中央 3×3。
// 切不出來（不該發生）就回 nil，畫面退回沒有地形的樣子，不讓戰鬥開不起來。
func (a *app) terrainForBattle() *game.BattleTerrain {
	t, err := game.NewBattleTerrain(a.tiles, a.party.X(), a.party.Y())
	if err != nil {
		a.message = fmt.Sprintf("戰場地形切不出來：%v", err)
		return nil
	}
	return t
}

// 這裡原本有一個 battleLight()：把光照等級當成戰場視野的內縮量。
//
// **那條規則不屬於戰場。** 它來自 `0x172f4`，而那一段填的是另一塊 9×9 的
// 緩衝（`[0x514e]`）；真正的戰場地形是由 3×3 個世界 tile 各放大 5×5 拼出來的，
// 整段程式碼裡沒有任何光照內縮（`docs/re/36`）。光照要接回哪個畫面還沒查清楚，
// 在查清楚之前寧可什麼都不做，也不要把它掛在對不上的地方。

// monsterSourceFile 是怪物名稱翻譯目錄的 key，與 dwstrings 產生時一致。
const monsterSourceFile = "MONSTER.DAT"

// startBattle 依事件記錄的怪物清單布置戰場。
//
// 怪物的速度與生命在 MONSTER.DAT 的基礎值上做進場擾動。
func (a *app) startBattle(ids []int) {
	var units []*game.Unit

	// 地形要先切出來 —— 擺位與移動都要問「這一格是不是空地」。
	terrain := a.terrainForBattle()
	taken := map[[2]int]bool{}
	occupied := func(x, y int) bool { return taken[[2]int{x, y}] }

	// 隊伍先站，照紮營選單排的 3×3 陣型（docs/re/34、35）。
	for i, c := range a.members {
		slot := game.PlayerSlotStart + i
		if slot >= game.PlayerSlotEnd {
			break
		}
		x, y, ok := game.DeployPartyAt(a.save.Formation, i)
		if !ok {
			// 陣型裡沒有他 —— 原版佈陣是掃九格，沒被放進去的人不會上場。
			// 本專案不讓人憑空消失：塞進中心附近還空著的位置。
			x, y, ok = game.ScatterMonster(a.rng, terrain, occupied)
			if !ok {
				continue
			}
		}
		taken[[2]int{x, y}] = true
		units = append(units, c.CombatUnit(slot, x, y, game.West))
	}

	// 怪物散在中心 ±2，開場就貼臉 —— 原版沒有「兩軍對峙」這回事。
	for i, id := range ids {
		if i >= game.MonsterSlotEnd {
			break
		}
		m, err := a.monsters.ByIndex(id)
		if err != nil {
			continue
		}
		x, y, ok := game.ScatterMonster(a.rng, terrain, occupied)
		if !ok {
			continue // 中心附近站滿了，這一隻上不了場
		}
		taken[[2]int{x, y}] = true
		speed, hp := game.RollMonsterStats(a.rng, m.Speed, m.HP)
		units = append(units, &game.Unit{
			Slot: i, Name: a.tr.Event(monsterSourceFile, id, m.Name),
			X: x, Y: y, Facing: int(game.East),
			Speed: speed, Strength: m.Strength, Skill: m.Skill,
			Level: m.Level, Intellect: m.Level, Experience: m.Experience,
			HP: hp, MaxHP: hp,
			WeaponIndex:   m.AttackType,
			RaceOrElement: m.Special,
			MaxSP:         m.SP, CurrentSP: m.SP,
		})
	}

	a.battle = game.NewBattle(a.rng, units)
	a.battle.Terrain = terrain
	a.battle.BeginRound()
	a.battleTerrain = terrain
	// 祈禱成功率跨戰鬥保留、每次祈禱永久遞減；初值 20% 來自手冊
	// （反組譯只確認了遞減量 −5，初始化位置未逐指令追出）。
	if a.prayChance == 0 {
		a.prayChance = game.PrayInitialChance
	}
	a.log = []string{fmt.Sprintf("第 %d 回合", a.battle.Round())}
}

// logf 把一行訊息推進戰鬥紀錄，只留最後幾行。
func (a *app) logf(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	a.log = append(a.log, line)
	if len(a.log) > battleLogLines {
		a.log = a.log[len(a.log)-battleLogLines:]
	}
	// 畫面上的戰鬥紀錄只留 8 行，而且截圖只看得到最後一瞬間。
	// DW_LOG=1 時同步吐到 stderr，驗收才能看完整條時間軸。
	if logToStderr {
		log.Print(line)
	}
}

// logToStderr 由環境變數 DW_LOG 開啟，只影響輸出、不影響任何規則。
var logToStderr = os.Getenv("DW_LOG") != ""

const battleLogLines = 8

var facingName = []string{"北", "東", "南", "西"}

// monthSourceFile 是月份名稱翻譯目錄的 key，與 dwstrings 產生時一致。
const monthSourceFile = "MONTHS"

// monthName 回傳目前月份的名稱。
//
// 原版的月份不是序數而是名字：狀態列印的是
// "Hour 13, Day 17 in the Month of the Ruby"。
//
// 月編號超出名稱表時退回數字 —— 原版在這裡會讀到表外的野指標
// （進位到 23 才歸 1，但只有 22 個名字），不照抄那個 bug。
func (a *app) monthName() string {
	names := a.strings.MonthNames()
	m := a.clock.Month()
	if m < 0 || m >= len(names) {
		return fmt.Sprintf("%d", m)
	}
	return a.tr.Event(monthSourceFile, m, names[m])
}

func (a *app) drawStatus(dst *ebiten.Image) {
	lines := []string{
		fmt.Sprintf("%2d時 %2d日 %s月", a.clock.Hour(), a.clock.Day(), a.monthName()),
		fmt.Sprintf("步數 %2d  光照 %d", a.clock.Steps(), a.clock.Light()),
		fmt.Sprintf("座標 %2d,%-2d 面向%s", a.party.X(), a.party.Y(), facingName[a.party.Facing()]),
		fmt.Sprintf("地形 %3d  深度 %d", a.lastTile, a.party.Depth()),
		fmt.Sprintf("圖塊 %s", a.tileset().Name()),
		"",
	}
	// **訊息要斷行，不能截斷。** 其餘幾行是固定格式、寬度可控，
	// 截斷只會掉尾巴的空白；訊息是變動長度的句子，截斷會把後半句吃掉。
	//
	// 這是全程試玩的截圖抓到的（`docs/playtest/01`）：走到 (23,31) 時
	// 「（地點劇情 3 還沒接，見 docs/re/65）」被切成「…見 docs/re」——
	// 而畫面上完全看不出那是被裁掉的，看起來就像訊息本來就那樣寫。
	// 單點截圖驗收沒抓到，因為那些畫面的訊息都夠短。
	if a.message != "" {
		lines = append(lines, textlayout.WrapMixed(a.message, layout.StatusPixels)...)
	}
	lines = append(lines, []string{
		"",
		"方向鍵：移動",
		"Tab：切換季節",
		"P：隊伍名冊",
		"T：進入城鎮",
		"B：測試戰鬥（偵錯）",
		"C：紮營",
		"F1：建立角色（偵錯）",
		"F2：手札",
		"M：遇到商隊（偵錯）",
		"S：存檔",
		"空白鍵：翻頁",
		"F10：離開遊戲",
	}...)
	// 溢出欄寬的字會畫到畫布外被裁掉 —— 看起來像訊息被砍一半，
	// 而不是「這行太長」。存檔路徑就踩過這個。
	// 訊息已經在上面斷過行，這裡的截斷只會碰到固定格式那幾行。
	for i, s := range lines {
		lines[i] = textlayout.TruncateCells(s, layout.StatusCells)
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
		// 種族名 2–4 格（人類／黑暗精靈）、武器名 2–3 格 —— 不補到固定寬度，
		// 後面的「生命」「護甲」欄會跟著左右跳。
		line(fmt.Sprintf("%s %d級 %s",
			textlayout.PadCells(c.Name, 8), c.Level, nameOf(className, int(c.Class))))
		line(fmt.Sprintf(" %s 生命 %3d/%3d",
			textlayout.PadCells(nameOf(raceName, int(c.Race)), 4), c.CurrentHP, c.MaxHP))
		line(fmt.Sprintf(" 法力 %3d/%3d 未用點數 %d", c.CurrentSP, c.MaxSP, pts))
		line(fmt.Sprintf(" %s 護甲 %d",
			textlayout.PadCells(a.weaponLabel(c), 6), c.ArmorRating()))
	}
	line("")
	line("P：返回")
}

func (a *app) Layout(int, int) (int, int) {
	return layout.CanvasWidth * scale, layout.CanvasHeight * scale
}

// newRNG 依旗標建立亂數產生器。
//
// 固定種子是**驗收工具**：戰鬥的行動順序由怪物進場擲點決定，
// 每次跑都不一樣，用時間種子的話「走五步再施法」這種截圖驗收
// 根本重跑不出同一個畫面。
func newRNG(seed uint) *rng.RNG {
	if seed == 0 {
		return rng.New()
	}
	return rng.NewWithSeed(uint32(seed))
}

func main() {
	dataDir := flag.String("data", "workplace/orig/demwin/DEM_DATA",
		"原版資料目錄（玩家自備合法副本）")
	etenDir := flag.String("eten", "workplace/eten",
		"倚天中文字型目錄，需含 STDFONT.15 與 SPCFONT.15（自備）")
	mapFile := flag.String("map", "MAP1.MAP",
		"要載入的地圖：檔名（MAP1.MAP／MAP3.MAP／MAP5.MAP）或 SUM.MAP 的子地圖編號"+
			"（如 34 = 起始大陸，見 docs/formats/town-and-map.md §2.5）")
	dataFile := flag.String("events", "DATA1.TXT", "要載入的事件表")
	seed := flag.Uint("seed", 0,
		"亂數種子。0 = 依時間。指定固定值可讓截圖驗收重跑得到同一結果")
	volume := flag.Float64("volume", 0.25,
		"音效音量 0–1。原版沒有音量控制（喇叭只有開關），這是體貼現代耳朵")
	savePath := flag.String("save", "workplace/save/PARTY.DAT",
		"進度存檔路徑。刻意不預設在原版資料目錄，免得蓋掉玩家的原版存檔")
	manualPath := flag.String("manual", "assets/manual/zh-Hant/manual.txt",
		"遊戲內手札的內容檔（缺檔就是沒有手札，不影響遊玩）")
	langDir := flag.String("lang", "assets/lang/zh-Hant",
		"翻譯目錄。指向不存在的路徑即為原文模式")
	startX := flag.Int("x", -1, "起始 X。負值代表用存檔裡的座標")
	startY := flag.Int("y", -1, "起始 Y。負值代表用存檔裡的座標")
	// B 鍵那條偵錯路徑在 headless 截圖底下不好按（xdotool 送的鍵不一定
	// 進得了 ebiten 的輸入佇列）。開一個旗標走同一條路，讓截圖驗收可重跑。
	startBattle := flag.Bool("battle", false, "啟動後直接開一場測試戰鬥（偵錯）")
	// 用 xdotool 打完一場戰鬥不切實際（要走位、要相鄰才打得到），
	// 但「打贏之後會怎樣」得看得到 —— 這個旗標直接把怪物血量清零。
	battleWin := flag.Bool("battle-win", false,
		"偵錯：開場就把測試戰鬥的怪物全部打倒，用來驗勝利後的流程")
	battleMonsters := flag.String("battle-monsters", "",
		"測試戰鬥要放哪幾隻怪（MONSTER.DAT 索引，逗號分隔）。留空用預設那組")
	// 起始存檔裡每一件裝備的效果索引與強度都是 0，照原版規則在 Use 選單裡
	// 一件都選不到 —— 沒有這個旗標就沒辦法驗「用道具真的會生效」。
	giveItem := flag.String("give-item", "",
		"偵錯：塞一件有效果的道具給第一名隊員，格式 `type,effect,power,次數`")
	// 起始隊伍沒有人會觀地、也沒有人會學識 —— 那幾個紮營選項在
	// headless 驗收時因此一步都走不到。
	giveSkill := flag.String("give-skill", "",
		"偵錯：教第一名隊員幾個技能（技能 id，逗號分隔）")
	// 起始隊伍沒有人有信仰，敬拜那一項在 headless 驗收時走不到最後一步。
	deityFlag := flag.Int("deity", 0,
		"偵錯：給第一名隊員一個信仰（神祇 1–11）與 20% 祈禱成功率")
	// 商隊規模的基準值在原版存檔裡是 1，擲出來的規模常常是 0 ——
	// 而規模就是等級，等級 0 的價格上限只有 1，連匕首（2）都選不出來，
	// 貨單會是清一色的匕首（見 `docs/re/50` §4）。要驗大商隊得改基準。
	merchantBase := flag.Int("merchant-base", -1,
		"偵錯：商隊規模的基準值。負值代表用存檔裡的 `+0xaf`")
	// 說謊機率 = rnd(120) − 80 鉗在 0，超過一半的商隊擲到 0；
	// 擲到之後每件貨還要各自中，View mind 又只有 2/3 揭得穿 ——
	// 三層機率疊起來，headless 截圖幾乎抓不到「謊報」那一格。
	merchantLies := flag.Int("merchant-lies", -1,
		"偵錯：商隊每件貨說謊的百分比機率。負值代表照原版擲")
	startHourFlag := flag.Int("hour", 0,
		"偵錯：起始時辰（1–38）。0 代表照原版的 5 時")
	// 城鎮的貴服務（復活、修船、買船）在起始存檔的 65 金之下全都試不到。
	goldFlag := flag.Int("gold", -1, "偵錯：起始金幣。負值代表用存檔裡的")
	// 主線的 UNCURSE／IMPRISON 要 50／100 點法力，起始隊伍最高只有 29 ——
	// 沒有這個旗標就驗不到成功路徑（`docs/re/62`）。
	spFlag := flag.Int("sp", -1, "偵錯：全隊目前法力。負值代表用存檔裡的")
	// 光之環的門與 IMPRISON 要三個符印都解完才驗得到，而符印散在
	// 世界東南角三張子地圖上 —— 沒有這個旗標就得先跑完整段主線。
	glyphsFlag := flag.Bool("glyphs", false, "偵錯：三個緋紅符印都當成已解除")
	// 符文密語要走到特定事件格才看得到（原版共四筆），
	// 而字型與排版是視覺產物、必須 dump 出來肉眼比對。
	runeFlag := flag.String("rune", "",
		"偵錯：直接顯示一段符文密語（例如 YMROS.IS...MINE）")
	// 事件文字要走到特定座標才觸發，而續接碼 3 的第二段、符文、插圖
	// 這些分支都在事件顯示路徑裡 —— 沒有這個旗標就只能靠導航去碰。
	eventFlag := flag.Int("event", -1, "偵錯：直接觸發某一筆事件（DATA*.TXT 索引）")
	// -ruins 讓「冬之魔降臨之後」的世界看得見。原版把這兩個旗標接在劇情上
	// （`+0xb9 == 2` 觸發神殿全毀），而劇情觸發本身還沒接 ——
	// 沒有這個旗標就沒辦法驗收 tile 替換與廢墟訊息（`docs/re/79`）。
	plotFlag := flag.Int("plot", -1,
		"偵錯：劇情階段 +0xb9（1 = 下次睡覺冬之魔降臨）。負值代表用存檔裡的")
	endingFlag := flag.Bool("ending", false,
		"偵錯：啟動後直接播結局序列（不必真的破關）")
	// 艾瑞戈爾那一格在地圖 1 的 (60,1)，要先走完大半張地城才到得了，
	// 而它有七條分支、十頁文字 —— 全部都是視覺產物，得逐頁 dump 比對。
	eregoreFlag := flag.Int("eregore", -1,
		"偵錯：直接播艾瑞戈爾那一場。0 = 第一次見面，1 = 談崩過一次（直接播結尾）")
	// 兩道密語謎題各在一張地城深處，而且是全遊戲僅有的自由文字輸入。
	// A4 全程試玩要能事後逐行核對「走了幾步、到了哪、觸發了什麼」。
	// 只有截圖的話，漏掉一次按鍵在畫面上完全看不出來。
	// 沒有這個，按鍵重播腳本永遠過不了第一場仗（`docs/playtest/01` §4）。
	// **它不是捷徑** —— 下的是玩家下得出來的同一組指令。
	autoFightFlag := flag.Bool("autofight", false,
		"驗收：戰鬥由自動驅動代打（找最近的敵人、面向、攻擊）")
	traceFlag := flag.String("trace", "",
		"驗收：把每一次狀態變化寫進這個檔（全程試玩用，見 trace.go）")
	riddleFlag := flag.Int("riddle", -1,
		"偵錯：直接出一道密語謎題（10 = 幽靈司祭／VOID，11 = 神殿門房／JESRIC）")
	ruinsFlag := flag.Bool("ruins", false,
		"偵錯：世界已成廢墟（神殿 tile 畫成廢墟、城鎮不再進得去）")
	// 選城鎮的選單要按十幾次方向鍵才到得了後面的城鎮，headless 截圖驗收時
	// xdotool 偶爾會漏掉一兩下，跑出來的畫面就不是預期的那座城。直接指定。
	townFlag := flag.Int("town", 0, "偵錯：啟動後直接進入指定編號的城鎮（1–25）")
	// 掉寶生成解出來了（docs/re/30），但「什麼時候掉」還沒追出來，
	// 所以沒有正規入口。這個旗標是唯一能在遊戲裡看到成品的路徑。
	lootFlag := flag.String("loot", "",
		"偵錯：用掉寶生成器發道具給第一名隊員，格式 `type,等級[,件數]`")
	flag.Parse()

	if _, err := os.Stat(*dataDir); err != nil {
		log.Fatalf("找不到原版資料目錄 %s：%v\n"+
			"本專案不散布原版資料，請用 -data 指向你自己的合法副本。", *dataDir, err)
	}

	tables, err := gamedata.LoadTables(filepath.Join(*dataDir, "FILES.DAT"))
	if err != nil {
		log.Fatalf("載入 FILES.DAT：%v", err)
	}
	m, mapID, err := loadMapArg(*dataDir, *mapFile)
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
	strs, err := gamedata.LoadStringPool(filepath.Join(*dataDir, "FILES.DTT"))
	if err != nil {
		log.Fatalf("載入字串池：%v", err)
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
	if *plotFlag >= 0 {
		save.PlotStage = byte(*plotFlag)
	}
	if *ruinsFlag {
		save.TempleRuins = 0xff
		save.ShardShattered = 1
	}
	// 特殊格清單要在存檔載入之後才決定來源 —— 全新開始要從 ALL_SS.DAT
	// 重建，否則會沿用原版出廠那份「玩到一半」的狀態（docs/re/78 §2）。
	man, err := manual.Load(*manualPath)
	if err != nil {
		log.Fatalf("載入手札：%v", err)
	}
	special, err := scenario.LoadSpecialTileSet(filepath.Dir(*savePath), *dataDir, fresh)
	if err != nil {
		log.Fatalf("載入特殊格清單：%v", err)
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

	// 座標預設跟著存檔走 —— 存了位置卻從固定點開場，等於沒存。
	// 旗標給非負值時才覆蓋，那是偵錯用的傳送。
	px, py := int(save.PositionX), int(save.PositionY)
	if *startX >= 0 {
		px = *startX
	}
	if *startY >= 0 {
		py = *startY
	}

	a := &app{
		trace:      newTracer(*traceFlag),
		auto:       newAutoFighter(*autoFightFlag),
		world:      game.NewWorld(m, tables),
		party:      newPartyAt(px, py, save),
		clock:      game.ClockAt(int(save.Hour), int(save.Day), int(save.Month), int(save.TimeCounter)),
		tiles:      m,
		mapID:      mapID,
		drawTiles:  ditheredTiles(m, uint16(*seed), save.TempleRuins),
		exits:      exits,
		special:    special,
		manual:     man,
		winText:    loadWinText(*dataDir),
		dreamText:  loadStoryOrNil(*dataDir, scenario.StoryDream),
		eregoreText: loadEregoreText(*dataDir),
		dreamPage:  -1,
		ditherSeed: uint16(*seed),
		// -ending 直接跳結局序列。破關要走完整條主線，沒有這個旗標
		// 就沒辦法驗收結局畫面（同 -glyphs／-ruins 的性質）。
		won: *endingFlag,
		events:     events,
		tr:         tr,
		eventsFile: *dataFile,
		tables:     tables,
		members:    members,
		monsters:   monsters,
		towns:      towns,
		strings:    strs,
		items:      items,
		rng:        newRNG(*seed),
		normal:     loadSet(gfx.NormalTiles),
		winter:     loadSet(gfx.WinterTiles),
		font:       font,
		speaker:    ui.NewSpeaker(*volume),
		title:      loadTitle(*dataDir),
		runeFont:   loadRuneFont(*dataDir),
		save:       save,
		torch:      save.LightSource,
		savePath:   *savePath,
	}

	// 船停在海上，而海面在可通行性表裡是不可通行的 —— 沒有這一條，
	// 船看得到卻走不上去。
	a.world.Boardable = func(x, y int) bool {
		return game.BoatAt(&a.save.Ships, x, y, a.mapID) >= 0
	}

	a.canvas = ebiten.NewImage(layout.CanvasWidth, layout.CanvasHeight)

	if *startHourFlag > 0 {
		for a.clock.Hour() != *startHourFlag {
			a.clock.AdvanceHour()
		}
		log.Printf("偵錯：時辰設為 %d", a.clock.Hour())
	}
	if *merchantBase >= 0 {
		a.save.MerchantBase = byte(*merchantBase)
	}
	a.debugMerchantLies = *merchantLies
	if *goldFlag >= 0 {
		a.setGold(*goldFlag)
		log.Printf("偵錯：金幣設為 %d", a.gold())
	}
	if *riddleFlag >= 0 {
		a.openRiddle(*riddleFlag)
		log.Printf("偵錯：密語謎題 %d", *riddleFlag)
	}
	if *eregoreFlag >= 0 {
		a.openEregore(*eregoreFlag == 1)
		log.Printf("偵錯：艾瑞戈爾（met=%v）", *eregoreFlag == 1)
	}
	if *eventFlag >= 0 {
		a.showEvent(*eventFlag)
		log.Printf("偵錯：觸發事件 %d", *eventFlag)
	}
	if *runeFlag != "" {
		a.openRuneBox(*runeFlag)
		log.Printf("偵錯：顯示符文密語 %q", *runeFlag)
	}
	if *glyphsFlag {
		for i := range a.save.GlyphFlags {
			a.save.GlyphFlags[i] = game.GlyphDone
		}
		log.Printf("偵錯：三個符印都設為已解除")
	}
	if *spFlag >= 0 {
		for i := range a.members {
			a.members[i].CurrentSP = *spFlag
			if a.members[i].MaxSP < *spFlag {
				a.members[i].MaxSP = *spFlag
			}
		}
		log.Printf("偵錯：全隊法力設為 %d", *spFlag)
	}
	if *townFlag > 0 {
		town, err := towns.ByNumber(*townFlag)
		if err != nil {
			log.Fatalf("-town：%v", err)
		}
		a.town = &townScreen{visit: game.EnterTown(town, a.members)}
		a.title = nil // 直接跳過標題畫面，不然還要多送一次按鍵
		log.Printf("偵錯：直接進入 %s", town.Name)
	}
	if *lootFlag != "" {
		if err := a.debugLoot(*lootFlag); err != nil {
			log.Fatalf("-loot：%v", err)
		}
	}
	if *giveItem != "" {
		if err := a.debugGiveItem(*giveItem); err != nil {
			log.Fatalf("-give-item：%v", err)
		}
	}
	if *giveSkill != "" {
		if err := a.debugGiveSkill(*giveSkill); err != nil {
			log.Fatalf("-give-skill：%v", err)
		}
	}
	if *deityFlag != 0 {
		if len(a.members) == 0 {
			log.Fatalf("-deity：隊伍是空的")
		}
		if *deityFlag < game.DeityMin || *deityFlag > game.DeityMax {
			log.Fatalf("-deity：神祇 %d 超出 %d–%d",
				*deityFlag, game.DeityMin, game.DeityMax)
		}
		a.members[0].Deity = *deityFlag
		a.members[0].PrayChance = game.PrayInitialChance
	}
	if *startBattle {
		picks := debugBattleMonsters
		if *battleMonsters != "" {
			picks = nil
			for _, f := range strings.Split(*battleMonsters, ",") {
				n, err := strconv.Atoi(strings.TrimSpace(f))
				if err != nil {
					log.Fatalf("-battle-monsters 認不得 %q：%v", f, err)
				}
				picks = append(picks, n)
			}
		}
		a.startBattle(picks)
		if *battleWin {
			for _, u := range a.battle.Units() {
				if u != nil && !u.IsPlayer {
					u.HP = 0
				}
			}
			log.Print("偵錯：怪物全部打倒，直接進入勝利流程")
		}
	}

	ebiten.SetWindowSize(layout.CanvasWidth*scale, layout.CanvasHeight*scale)
	ebiten.SetWindowTitle("冬之魔 Demon's Winter")

	defer a.trace.close()
	a.trace.note("啟動：地圖 %d，隊伍在 (%d,%d)，%d 人",
		mapID, a.party.X(), a.party.Y(), len(a.members))
	if err := ebiten.RunGame(a); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}

// weaponLabel 回傳角色目前武器的顯示名稱（已翻譯）。
func (a *app) weaponLabel(c game.Character) string {
	w := c.Weapon()
	if w.Empty() {
		return "徒手"
	}
	return a.itemLabel(w)
}

// debugLoot 用掉寶生成器發道具給第一名隊員（格式 `type,等級[,件數]`）。
//
// 掉寶生成的規則解出來了（`docs/re/30`），但**「什麼時候掉」還沒追出來** ——
// 戰鬥勝利與寶箱的判定不在已解範圍內，所以沒有正規入口。
// 沒有這個旗標就沒辦法在遊戲裡看到生成器的成品。
func (a *app) debugLoot(spec string) error {
	f := strings.Split(spec, ",")
	if len(f) < 2 || len(f) > 3 {
		return fmt.Errorf("格式是 type,等級[,件數]，拿到 %q", spec)
	}
	n := make([]int, len(f))
	for i, part := range f {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return fmt.Errorf("第 %d 個欄位 %q 不是數字", i+1, part)
		}
		n[i] = v
	}
	count := 1
	if len(n) == 3 {
		count = n[2]
	}
	if len(a.members) == 0 {
		return fmt.Errorf("隊伍是空的")
	}
	item, err := a.items.ByIndex(n[0])
	if err != nil {
		return err
	}

	given := 0
	for mi := range a.members {
		c := &a.members[mi]
		for i := range c.Inventory {
			if given >= count {
				return nil
			}
			if !c.Inventory[i].Empty() {
				continue
			}
			slot := game.GenerateDrop(a.rng, a.tables, item, n[0], n[1])
			c.Inventory[i] = slot
			given++
			// 已用次數印原始值，不印「剩幾次」—— 充能種類 1 的 255 是
			// 「不計次」的哨兵而不是計數，減出來會是負的。
			log.Printf("偵錯：%s 第 %d 格 → %s 附魔%+d 效果%d 強度%d 次數上限%d 已用%d",
				c.Name, i, item.Name, slot.Enchant, slot.Effect, slot.Power,
				slot.Total, slot.Used)
		}
	}
	if given == 0 {
		return fmt.Errorf("全隊的道具欄都滿了")
	}
	return nil
}

// newPartyAt 建立隊伍並還原搭船狀態。
//
// 搭船狀態不還原的話，存檔存在海上的隊伍一載入就被困在水裡 ——
// 四周都是海，一步都走不了。
func newPartyAt(x, y int, save *scenario.SaveGame) *game.Party {
	p := game.NewParty(x, y, game.Facing(save.Facing), 0)
	p.SetSailing(game.Sailing(save.Boat))
	return p
}

// stepBoat 處理走完一步之後的上／下船。
//
// 上船的判定是「走到船所在的那一格」，下船是「從船上走回陸地」。
// 兩者都不用按鍵 —— 原版就是走過去就上、走上岸就下。
func (a *app) stepBoat(tile byte) {
	next, res := game.StepBoat(&a.save.Ships, a.save.Boat, tile,
		a.party.X(), a.party.Y(), a.mapID)
	a.save.Boat = next
	a.party.SetSailing(game.Sailing(next))

	switch res {
	case game.BoardOn:
		a.message = "登船"
	case game.BoardOff:
		a.message = "上岸"
	}
}

// stepHPTick 走一次每步 HP 變動（`FUN_222f_0619`，`docs/re/63`）。
//
// 同一支常式兩種模式：一般行走時**巨魔再生**，符印還在的子地圖裡
// **全隊流血**（連巨魔都不免疫）。所以走到火山那三塊狹長陸地上
// 會有「趕快解掉符印」的壓力 —— 那是原版設計的一部分。
func (a *app) stepHPTick() {
	mode := game.GlyphDrainMode(a.save.GlyphFlags, a.mapID)
	res := game.StepHPTick(a.members, mode)
	if !res.Changed && len(res.Died) == 0 {
		return
	}
	for _, i := range res.Died {
		a.message = fmt.Sprintf(a.tr.UI("plot.fell", "%s 倒下了"), a.members[i].Name)
	}
	if mode == game.StepHPDrain && len(res.Died) == 0 {
		a.message = a.tr.UI("plot.glyphdrain", "符印的力量侵蝕著隊伍")
	}
	if res.AllDead {
		a.message = a.tr.UI("plot.allfell", "全隊都倒下了")
	}
}
