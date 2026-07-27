package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 地城道具的操作層：`T` 拾取、`D` 丟棄。
//
// 規則在 `internal/game/dungeonitem.go`（原版 `25be:0077` ＝ 動作 `0x09`
// 與 `222f:2088(1)` → `122f:2845`）。這裡只做選單、訊息與存檔寫回。
//
// 手冊那六個指令（`I`／`E`／`T`／`D`／`M`／`U`）**整套解謎鏈都在上面** ——
// 主線的 Qoorik、Asaht、魔晶、冥河擺渡人、光之環都是從地城道具的敘述
// 給出來的。這一輪先接會改變狀態的兩個（拾取與丟棄），
// `E`／`M`／`U`／`I` 隨後。

// dungeonMode 是這個畫面在做哪一件事。
type dungeonMode int

const (
	dungeonTake dungeonMode = iota
	dungeonDrop
	dungeonExamine
	dungeonMove
)

// fromInventory 回報這個模式選的是「身上的道具」而不是「腳下的東西」。
//
// **原版就是這樣分的**：`Use`／`Drop`／`Examine` 共用 `122f:1d23`
// 選「哪個角色的第幾格」，`Take:`／`Move:` 才掃腳下（`docs/re/95` §3.8）。
func (m dungeonMode) fromInventory() bool {
	return m == dungeonDrop || m == dungeonExamine
}

// dungeonScreen 是拾取／丟棄的選單。
//
// 拾取是兩段式的，順序照原版：**先選東西、再選人**
// （`0x19891` 的拿不走檢查排在 `0x198ed` 的「Character to take」之前）。
type dungeonScreen struct {
	mode  dungeonMode
	stage int // 0 ＝ 選道具，1 ＝ 選角色（只有拾取用得到）

	spots   []game.DungeonSpot        // 拾取：腳下這一格有什麼
	carried []game.CarriedDungeonItem // 丟棄：全隊身上有什麼

	cursor int
	// pick 是第一段選中的那一件，第二段才有意義。
	pick int
}

// openDungeonTake 是 `T` 拾取，openDungeonMove 是 `M` 移動。
// 兩者選的都是**腳下這一格**（原版共用 `222f:2da5`）。
func (a *app) openDungeonTake() { a.openUnderfootPicker(dungeonTake, "拾取") }
func (a *app) openDungeonMove() { a.openUnderfootPicker(dungeonMove, "移動") }

func (a *app) openUnderfootPicker(mode dungeonMode, what string) {
	if a.itemloc == nil {
		return
	}
	spots := game.ItemsUnderfoot(a.itemloc, a.dungeonItems,
		byte(a.party.X()), byte(a.party.Y()), byte(a.mapID))
	if len(spots) == 0 {
		a.message = a.tr.UI("dungeon.nothing", "這裡沒有東西")
		return
	}
	a.dungeon = &dungeonScreen{mode: mode, spots: spots}
	a.trace.note("%s：腳下 %d 件", what, len(spots))
}

// openDungeonDrop 是 `D` 丟棄，openDungeonExamine 是 `E` 檢視。
//
// 兩者的選單都列**全隊**身上的地城道具 —— 原版的選取常式
// （`122f:1d23`）回傳的就是「哪個角色的第幾格」。
func (a *app) openDungeonDrop()    { a.openCarriedPicker(dungeonDrop, "丟棄") }
func (a *app) openDungeonExamine() { a.openCarriedPicker(dungeonExamine, "檢視") }

func (a *app) openCarriedPicker(mode dungeonMode, what string) {
	if a.itemloc == nil {
		return
	}
	carried := game.CarriedDungeonItems(a.members, a.dungeonItems)
	if len(carried) == 0 {
		a.message = a.tr.UI("dungeon.carry.none", "身上沒有地城道具")
		return
	}
	a.dungeon = &dungeonScreen{mode: mode, carried: carried}
	a.trace.note("%s：身上 %d 件", what, len(carried))
}

func (a *app) updateDungeon() error {
	d := a.dungeon
	n := a.dungeonChoices()
	if n == 0 {
		a.dungeon = nil
		return nil
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		// 第二段按 ESC 退回選道具，不是整個關掉。
		if d.stage == 1 {
			d.stage, d.cursor = 0, 0
			return nil
		}
		a.dungeon = nil
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		d.cursor = (d.cursor + 1) % n
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		d.cursor = (d.cursor - 1 + n) % n
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.confirmDungeon()
	}
	return nil
}

// count 是目前這一段有幾個可選項。第二段選的是人，所以要問隊伍人數。
func (a *app) dungeonChoices() int {
	d := a.dungeon
	switch {
	case d.mode.fromInventory():
		return len(d.carried)
	case d.stage == 1:
		return len(a.members)
	default:
		return len(d.spots)
	}
}

func (a *app) confirmDungeon() {
	d := a.dungeon
	switch d.mode {
	case dungeonDrop:
		a.dropDungeonItem(d.carried[d.cursor])
		return
	case dungeonExamine:
		a.examineDungeonItem(d.carried[d.cursor])
		return
	case dungeonMove:
		a.moveDungeonItem(d.spots[d.cursor])
		return
	}
	if d.stage == 0 {
		spot := d.spots[d.cursor]
		// 拿不走的東西**在問人之前就攔下來**，照原版的順序。
		if r, msg := game.DungeonTakeRefusal(spot.Item); r != game.TakeAllowed {
			a.message = a.refusalText(spot.Index, r, msg)
			a.dungeon = nil
			a.trace.note("拾取 %s：%s", spot.Item.Name, a.message)
			return
		}
		d.pick, d.stage, d.cursor = d.cursor, 1, 0
		return
	}
	a.takeDungeonItem(d.spots[d.pick], d.cursor)
}

func (a *app) takeDungeonItem(spot game.DungeonSpot, member int) {
	if member < 0 || member >= len(a.members) {
		return
	}
	res := game.TakeDungeonItem(&a.members[member], a.itemloc, a.dungeonItems, spot.Index)
	a.dungeon = nil
	if !res.OK {
		a.message = a.refusalText(spot.Index, res.Refusal, res.Message)
		a.trace.note("拾取 %s 失敗：%s", spot.Item.Name, a.message)
		return
	}
	a.message = fmt.Sprintf(a.tr.UI("dungeon.taken", "%s 拿走了 %s"),
		a.members[member].Name, a.dungeonName(spot.Item.Name))
	a.trace.note("拾取 %s → %s 第 %d 格",
		spot.Item.Name, a.members[member].Name, res.Slot)
}

func (a *app) dropDungeonItem(c game.CarriedDungeonItem) {
	res := game.DropDungeonItem(&a.members[c.Member], a.itemloc, a.dungeonItems,
		c.Slot, byte(a.party.X()), byte(a.party.Y()), byte(a.mapID))
	a.dungeon = nil
	if !res.OK {
		// 走到這裡代表選單給了一個不該出現的項目 —— 那是本專案的 bug，
		// 不是玩家做錯什麼，所以訊息說得直白一點。
		a.message = a.tr.UI("dungeon.drop.failed", "放不下去")
		a.trace.note("丟棄 %s 失敗（選單給了無效項目）", c.Name)
		return
	}
	a.message = fmt.Sprintf(a.tr.UI("dungeon.dropped", "放下了 %s"),
		a.dungeonName(c.Name))
	a.trace.note("丟棄 %s 於 (%d,%d) 地圖%d",
		c.Name, a.party.X(), a.party.Y(), a.mapID)
}

// moveDungeonItem 是 `M`：推開一件家具（原版 `222f:3621`）。
//
// 改的那一格**不一定是腳下那一格** —— 三件推得動的家具開的都是自己
// 旁邊那一格（`Old bookcase` 在 (46,12)，開的是 (46,11)）。
func (a *app) moveDungeonItem(spot game.DungeonSpot) {
	a.dungeon = nil
	res := game.MoveDungeonItem(a.dungeonItems, spot.Index,
		func(x, y int) (byte, bool) { return a.tileValue(x, y) })

	switch res.Kind {
	case game.MoveCant:
		a.message = a.tr.UI("dungeon.cant", "你辦不到")
		a.trace.note("移動 %s：推不動", spot.Item.Name)
	case game.MoveNothingHappens:
		a.message = a.tr.UI("dungeon.nothing.happens", "什麼也沒發生")
		a.trace.note("移動 %s：什麼也沒發生", spot.Item.Name)
	case game.MoveChanged:
		if err := a.writeTile(res.X, res.Y, res.Tile); err != nil {
			a.message = fmt.Sprintf("改寫地圖失敗：%v", err)
			a.trace.note("移動 %s 失敗：%v", spot.Item.Name, err)
			return
		}
		a.message = a.tr.UI("dungeon.something", "發生了什麼事……")
		a.trace.note("移動 %s：(%d,%d) → tile %d",
			spot.Item.Name, res.X, res.Y, res.Tile)
	}
}

// tileValue 讀目前這張地圖的一格**原始值**（不遮罩）。
//
// 原版比對「那一格是不是已經改過了」用的是原始 byte（`0x19611` 直接
// `mov al, es:[bx+si]`），所以這裡也不能套 `& 0x7f` —— 遮罩過的話，
// 最高位被設起來的格子會被誤判成「已經是目標 tile」而不動。
func (a *app) tileValue(x, y int) (byte, bool) {
	t, err := a.tiles.TileAt(x, y)
	if err != nil {
		return 0, false
	}
	return t, true
}

// examineDungeonItem 是 `E`：印出 `+2` 欄那段敘述。
//
// 敘述可能很長（`Old bookcase` 那句 40 幾個字），所以走文字框不走狀態列 ——
// 原版也是 `15be:158e`（與事件敘述同一支）。
func (a *app) examineDungeonItem(c game.CarriedDungeonItem) {
	a.dungeon = nil
	look, ok := game.ExamineDungeonItem(a.dungeonItems, c.Name)
	if !ok {
		// 原版 ds:0x241a `You see nothing special about the %s`。
		a.message = fmt.Sprintf(
			a.tr.UI("dungeon.nothing.special", "%s 看不出有什麼特別"),
			a.dungeonName(c.Name))
		a.trace.note("檢視 %s：沒什麼特別", c.Name)
		return
	}
	// 敘述在 `FILES.DTT` 的第 `164 + i×6 + 2` 條。
	text := look
	if c.Index >= 0 {
		text = a.tr.Event(dungeonSourceFile,
			gamedata.DungeonItemFirstString+c.Index*gamedata.DungeonItemFields+2, look)
	}
	a.box = ui.NewMixedTextBox(text)
	a.trace.note("檢視 %s", c.Name)
}

// dungeonName 是畫面上的地城道具名。
//
// 前面那個 `/` 是原版的記號（手冊：「地城物品在清單中前面會加上 `/`」）——
// 一般道具沒有，看到斜線就知道這是解謎用的獨一無二物件。
// 記號**加在譯名外面**，不進翻譯檔：它是型別的顯示規則，不是名字的一部分。
func (a *app) dungeonName(name string) string {
	return "/" + a.tr.Event(dungeonSourceFile,
		a.dungeonStringIndex(name), name)
}

// refusalText 把拿不走的理由變成畫面上那一行。
//
// 三種理由的文字**來源不同**：資料那句在 `FILES.DTT`（第 index 件的
// `+1` 欄 ＝ `164 + index×6 + 1`，走 `tr.Event` 對索引翻譯），
// 另外兩句是引擎自己的（走 `tr.UI` 的語意化 key）。
func (a *app) refusalText(index int, r game.DungeonRefusal, msg string) string {
	switch r {
	case game.TakeSilent:
		return a.tr.UI("dungeon.cant", "你辦不到")
	case game.TakeNoRoom:
		return a.tr.UI("dungeon.noroom", "放不下了")
	case game.TakeGone:
		return a.tr.UI("dungeon.gone", "那件東西已經不在這裡了")
	case game.TakeFromData:
		return a.tr.Event(dungeonSourceFile,
			gamedata.DungeonItemFirstString+index*gamedata.DungeonItemFields+1, msg)
	}
	return ""
}

// dungeonSourceFile 是地城道具名在翻譯檔裡的來源標籤。
//
// 索引用的是 `FILES.DTT` 的**絕對條目編號**（第 i 件的 `+0` 欄
// ＝ `164 + i×6`），與 `cmd/dwstrings` 抽出來的編號同一套。
const dungeonSourceFile = "FILES.DTT"

// dungeonStringIndex 把名字換回 `FILES.DTT` 的條目編號，給翻譯查表用。
// 找不到就回 −1（`tr.Event` 會退回原文）。
func (a *app) dungeonStringIndex(name string) int {
	i, ok := a.dungeonItems.ByName(name)
	if !ok {
		return -1
	}
	return gamedata.DungeonItemFirstString + i*gamedata.DungeonItemFields
}

func (a *app) drawDungeon(dst *ebiten.Image) {
	d := a.dungeon
	y := layout.StatusY
	line := func(s string) {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}

	switch {
	case d.mode.fromInventory():
		title := a.tr.UI("dungeon.drop.title", "放下哪一件？")
		if d.mode == dungeonExamine {
			title = a.tr.UI("dungeon.examine.title", "看哪一件？")
		}
		line(title)
		line("")
		for i, c := range d.carried {
			line(fmt.Sprintf("%s%s　%s",
				memberMark(d.cursor, i), a.dungeonName(c.Name),
				a.members[c.Member].Name))
		}
	case d.stage == 1:
		line(fmt.Sprintf(a.tr.UI("dungeon.take.who", "誰來拿 %s？"),
			a.dungeonName(d.spots[d.pick].Item.Name)))
		line("")
		for i := range a.members {
			c := &a.members[i]
			line(fmt.Sprintf("%s%s　空 %d 格",
				memberMark(d.cursor, i), c.Name, freeSlots(c)))
		}
	default:
		title := a.tr.UI("dungeon.take.title", "拿哪一件？")
		if d.mode == dungeonMove {
			title = a.tr.UI("dungeon.move.title", "推哪一件？")
		}
		line(title)
		line("")
		// **只列名字。** 原版的清單也只有名字，拿不走的理由等你選了才說。
		// 這裡一度把理由接在名字後面，結果 `/Bed （It is too hea` 被欄寬
		// 裁掉半句 —— 而畫面上看起來就像資料本來就那樣寫。
		// 最長的名字（`Serpent pillar`）加任何後綴都塞不進 21 格。
		for i, sp := range d.spots {
			line(memberMark(d.cursor, i) + a.dungeonName(sp.Item.Name))
		}
	}
	line("")
	line(a.tr.UI("dungeon.keys", "↑↓：選擇　Enter：確定　Esc：返回"))
}

// freeSlots 是這名角色還有幾個空格 —— 拿之前先看得到，
// 免得選了人才被告訴「放不下了」。
func freeSlots(c *game.Character) int {
	n := 0
	for i := range c.Inventory {
		if c.Inventory[i].Empty() {
			n++
		}
	}
	return n
}
