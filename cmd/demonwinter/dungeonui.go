package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
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
	// dungeonUse 是 `U`：三段（選手上那件 → 用在哪 → 選目標）。
	dungeonUse
	// dungeonUseRoom／dungeonUseChar 是 `U` 的第三段，兩條路分開記
	// 因為**後果不同**（`docs/re/95` §3.1 的對照表）。
	dungeonUseRoom
	dungeonUseChar
	// dungeonViewItem 是 `X` 鑑物：選一件腳下的，看它的 `+4`（技能 28）。
	dungeonViewItem
)

// fromInventory 回報這個模式選的是「身上的道具」而不是「腳下的東西」。
//
// **原版就是這樣分的**：`Use`／`Drop`／`Examine` 共用 `122f:1d23`
// 選「哪個角色的第幾格」，`Take:`／`Move:` 才掃腳下（`docs/re/95` §3.8）。
func (m dungeonMode) fromInventory() bool {
	return m == dungeonDrop || m == dungeonExamine ||
		m == dungeonUse || m == dungeonUseChar
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

	// source 是 `U` 第一段選中的那件（手上的）。
	source game.CarriedDungeonItem
	// onMenu 為真時停在「用在：角色／房間／取消」那一頁。
	onMenu bool
}

// useTargets 是「用在哪」那三個選項（原版 ds:0x2334 的 `Character`／
// `Room`／`Quit`，熱鍵 `CRQ`）。
// ui:dynamic dungeon.use.target. —— label 由 `"dungeon.use.target."+t.key` 查表。
var useTargets = []struct {
	key   string
	label string
	mode  dungeonMode
}{
	{"C", "身上的另一件", dungeonUseChar},
	{"R", "這個房間裡的", dungeonUseRoom},
	{"Q", "取消", dungeonUse},
}

// openDungeonTake 是 `T` 拾取，openDungeonMove 是 `M` 移動。
// 兩者選的都是**腳下這一格**（原版共用 `222f:2da5`）。
func (a *app) openDungeonTake() { a.openUnderfootPicker(dungeonTake, "拾取") }
func (a *app) openDungeonMove() { a.openUnderfootPicker(dungeonMove, "移動") }

// openViewItem 是 `X` 鑑物（手冊「靈視 → 鑑物」，原版動作 `0x10`）。
//
// 三道前置（有沒有人會／額度／擲點）在選東西**之前**就跑完，照原版順序 ——
// 所以失敗時連選單都不會開，而且**額度照樣扣掉**。
func (a *app) openViewItem() {
	if a.itemloc == nil {
		return
	}
	switch res := game.BeginViewItem(a.rng, a.members, &a.save.ViewItemUses); {
	case res.NoSkill:
		// 原版什麼都不印。這裡說一句，不然按了沒反應像壞掉。
		a.message = a.tr.UI("viewitem.noskill")
		a.trace.note("鑑物：沒有人會")
		return
	case res.Exhausted:
		a.message = a.tr.UI("viewroom.weak")
		a.trace.note("鑑物：額度用完")
		return
	case res.Failed:
		a.message = a.tr.UI("viewitem.fails")
		a.trace.note("鑑物：擲點失敗（剩 %d 次）", game.PsychicUsesPerDay-a.save.ViewItemUses)
		return
	}
	a.openUnderfootPicker(dungeonViewItem, "鑑物")
}

func (a *app) openUnderfootPicker(mode dungeonMode, what string) {
	if a.itemloc == nil {
		return
	}
	spots := game.ItemsUnderfoot(a.itemloc, a.dungeonItems,
		byte(a.party.X()), byte(a.party.Y()), byte(a.mapID))
	if len(spots) == 0 {
		a.message = a.tr.UI("dungeon.nothing")
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
func (a *app) openDungeonUse()     { a.openCarriedPicker(dungeonUse, "使用") }

func (a *app) openCarriedPicker(mode dungeonMode, what string) {
	if a.itemloc == nil {
		return
	}
	carried := game.CarriedDungeonItems(a.members, a.dungeonItems)
	if len(carried) == 0 {
		a.message = a.tr.UI("dungeon.carry.none")
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
		// 後面的段落按 ESC 退回上一段，不是整個關掉。
		switch {
		case d.onMenu:
			d.onMenu, d.cursor = false, 0
		case d.stage == 1:
			d.stage, d.cursor = 0, 0
		default:
			a.dungeon = nil
		}
		return nil
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
	case d.onMenu:
		return len(useTargets)
	case d.mode == dungeonUseRoom:
		return len(d.spots)
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
	if d.onMenu {
		a.pickUseTarget(useTargets[d.cursor].mode)
		return
	}
	switch d.mode {
	case dungeonUse:
		// 第一段選完 → 進「用在哪」那一頁。
		d.source = d.carried[d.cursor]
		d.onMenu, d.cursor = true, 0
		return
	case dungeonUseChar:
		a.useDungeonItem(d.carried[d.cursor].Index, d.carried[d.cursor].Name, true)
		return
	case dungeonUseRoom:
		a.useDungeonItem(d.spots[d.cursor].Index, d.spots[d.cursor].Item.Name, false)
		return
	case dungeonViewItem:
		a.viewItemHint(d.spots[d.cursor])
		return
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
	a.message = fmt.Sprintf(a.tr.UI("dungeon.taken"),
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
		a.message = a.tr.UI("dungeon.drop.failed")
		a.trace.note("丟棄 %s 失敗（選單給了無效項目）", c.Name)
		return
	}
	a.message = fmt.Sprintf(a.tr.UI("dungeon.dropped"),
		a.dungeonName(c.Name))
	a.trace.note("丟棄 %s 於 (%d,%d) 地圖%d",
		c.Name, a.party.X(), a.party.Y(), a.mapID)
}

// pickUseTarget 處理「用在：角色／房間／取消」那一頁的選擇。
func (a *app) pickUseTarget(mode dungeonMode) {
	d := a.dungeon
	switch mode {
	case dungeonUseChar:
		// 排掉自己 —— 拿一件東西對它自己用沒有意義，而原版的清單
		// 會把它列出來（同一個選取常式），選了就是 `Nothing happens`。
		var others []game.CarriedDungeonItem
		for _, c := range d.carried {
			if c.Member != d.source.Member || c.Slot != d.source.Slot {
				others = append(others, c)
			}
		}
		if len(others) == 0 {
			a.message = a.tr.UI("dungeon.use.noother")
			a.dungeon = nil
			return
		}
		d.mode, d.carried, d.onMenu, d.cursor = dungeonUseChar, others, false, 0
	case dungeonUseRoom:
		spots := game.ItemsUnderfoot(a.itemloc, a.dungeonItems,
			byte(a.party.X()), byte(a.party.Y()), byte(a.mapID))
		if len(spots) == 0 {
			a.message = a.tr.UI("dungeon.nothing")
			a.dungeon = nil
			return
		}
		d.mode, d.spots, d.onMenu, d.cursor = dungeonUseRoom, spots, false, 0
	default:
		a.dungeon = nil
	}
}

// useDungeonItem 是 `U` 的第三段：比對 `+4` 並執行 `+5`。
//
// onCharacter 分開兩條路的後果（`docs/re/95` §3.1）：
//
//	Room       來源那一格清空、目標從地圖上消失、**新道具放在腳下**
//	Character  來源那一格清空、目標那一格變成新道具（型別 0xfe）
func (a *app) useDungeonItem(target int, targetName string, onCharacter bool) {
	d := a.dungeon
	src := d.source
	a.dungeon = nil

	res := game.UseDungeonItem(a.dungeonItems, src.Name, target)
	if res.Outcome == game.DungeonUseNothing {
		// **什麼都不消耗** —— 用錯東西不該懲罰玩家。
		a.message = a.tr.UI("dungeon.nothing.happens")
		a.trace.note("使用 %s → %s：什麼也沒發生", src.Name, targetName)
		return
	}
	a.trace.note("使用 %s → %s：%v", src.Name, targetName, res.Outcome)

	switch res.Outcome {
	case game.DungeonUseDescribe:
		// 敘述在 `FILES.DTT` 的第 `164 + i×6 + 5` 條，去掉開頭那個動作碼。
		a.box = ui.NewMixedTextBox(a.useResultText(target, res.Text))
	case game.DungeonUseBecome:
		a.becomeDungeonItem(src, target, res.NewName, onCharacter)
	case game.DungeonUseTeleport:
		a.teleportTo(res.X, res.Y, int(res.MapID))
	case game.DungeonUsePassage:
		if err := a.writeTile(res.X, res.Y, res.Tile); err != nil {
			a.message = fmt.Sprintf(a.tr.UI("dungeon.map.writefailed"), err)
			return
		}
		a.message = a.tr.UI("dungeon.something")
	case game.DungeonUseStory:
		a.runDungeonStory(res.Story)
	}
}

// useResultText 是 `+5` 的敘述。
//
// **翻譯檔存的是去掉動作碼 `D` 之後的內文**（`cmd/dwstrings dungeonitems`
// 抽的時候就切掉了）—— 動作碼是控制字元不是台詞，讓譯者看到它只會多一種
// 翻壞的方式。所以這裡直接查、直接用，不再切第一個字元。
func (a *app) useResultText(index int, param string) string {
	i := gamedata.DungeonItemFirstString + index*gamedata.DungeonItemFields + 5
	return a.tr.Event(dungeonSourceFile, i, param)
}

// becomeDungeonItem 是動作碼 `N`。
func (a *app) becomeDungeonItem(src game.CarriedDungeonItem, target int,
	newName string, onCharacter bool) {

	// 來源那一件被用掉了（原版寫型別 0xff，名字留在後面當殘值）。
	a.members[src.Member].Inventory[src.Slot] =
		scenario.InventorySlot{Type: scenario.SlotEmpty}

	if onCharacter {
		// 目標那一格直接變成新道具（型別 0xfe），不碰位置表。
		if t, ok := a.dungeonSlotOf(target); ok {
			a.members[t.Member].Inventory[t.Slot] = scenario.NewDungeonSlot(newName)
		}
	} else {
		// 目標從地圖上消失（只清子地圖那一個 byte，`0x1839b`），
		// 新道具用 Drop 那一支放在腳下（`0x183e0`）。
		if target >= 0 && target < len(a.itemloc.Records) {
			a.itemloc.Records[target].MapID = scenario.ItemLocTaken
		}
		if i, ok := a.dungeonItems.ByName(newName); ok {
			a.itemloc.Drop(i, byte(a.party.X()), byte(a.party.Y()), byte(a.mapID))
		}
	}
	a.message = fmt.Sprintf(a.tr.UI("dungeon.yousee"),
		a.dungeonName(newName))
}

// dungeonSlotOf 找出第 index 件現在在誰的哪一格。
func (a *app) dungeonSlotOf(index int) (game.CarriedDungeonItem, bool) {
	for _, c := range game.CarriedDungeonItems(a.members, a.dungeonItems) {
		if c.Index == index {
			return c, true
		}
	}
	return game.CarriedDungeonItem{}, false
}

// runDungeonStory 是動作碼 `S`。
//
// **只做讀得出來的那一半。** `S1` 的門檻與一次性閂鎖、`S2` 的傳送座標
// 都是明碼（`docs/re/95` §3.5）；`15be:1694(n)` 那個「播第 n 段劇情」
// 還沒讀出來（那支是資源載入常式，不是劇情播放器），所以劇情文字先缺。
func (a *app) runDungeonStory(n int) {
	switch n {
	case 1:
		// 冰之祭壇 ＋ 祈禱卷軸 ＝ 結局的不朽／凡人抉擇。
		if a.save.PlotStage == 0 {
			a.message = a.tr.UI("dungeon.story.notyet")
			return
		}
		if a.save.EndingOffered != 0 {
			a.message = a.tr.UI("dungeon.story.done")
			return
		}
		a.save.EndingOffered = 1
		a.won = true
	case 2:
		// 光之環：傳送到 (11,27)。
		a.teleportTo(circleLightX, circleLightY, a.mapID)
	}
}

// circleLightX／circleLightY 是 `S2` 的落點（原版 `0x186e7`：
// `+0xa1 = 0x0b`、`+0xa2 = 0x1b`）。**不改子地圖** —— 那一段沒有寫 `+0xa3`。
const (
	circleLightX = 11
	circleLightY = 27
)

// teleportTo 把隊伍搬到指定座標；子地圖不同就換圖。
func (a *app) teleportTo(x, y, mapID int) {
	if mapID != a.mapID {
		// changeMap 自己會在失敗時留在原地並說清楚。
		a.changeMap(mapID, x, y)
	} else {
		a.party.TeleportTo(x, y)
	}
	a.message = a.tr.UI("dungeon.something")
	a.trace.note("傳送到 (%d,%d) 地圖%d", x, y, mapID)
}

// viewItemHint 是 `X` 鑑物的結果：印出這件東西要搭配哪一件。
//
// **這是解謎提示系統**，不是鑑定價值 —— 原版讀的是 `+4` 欄
// （`0x1949b`），與 `U` 的比對用同一欄。
func (a *app) viewItemHint(spot game.DungeonSpot) {
	a.dungeon = nil
	with, ok := game.ViewItemHint(a.dungeonItems, spot.Index)
	if !ok {
		a.message = a.tr.UI("viewitem.nothing")
		a.trace.note("鑑物 %s：沒有搭配對象", spot.Item.Name)
		return
	}
	// 原版是 `An image of %s` ＋ `comes to you` 兩行。
	a.message = fmt.Sprintf(a.tr.UI("viewitem.image"),
		a.dungeonName(with))
	a.trace.note("鑑物 %s → %s", spot.Item.Name, with)
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
		a.message = a.tr.UI("dungeon.cant")
		a.trace.note("移動 %s：推不動", spot.Item.Name)
	case game.MoveNothingHappens:
		a.message = a.tr.UI("dungeon.nothing.happens")
		a.trace.note("移動 %s：什麼也沒發生", spot.Item.Name)
	case game.MoveChanged:
		if err := a.writeTile(res.X, res.Y, res.Tile); err != nil {
			a.message = fmt.Sprintf(a.tr.UI("dungeon.map.writefailed"), err)
			a.trace.note("移動 %s 失敗：%v", spot.Item.Name, err)
			return
		}
		a.message = a.tr.UI("dungeon.something")
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
			a.tr.UI("dungeon.nothing.special"),
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
		return a.tr.UI("dungeon.cant")
	case game.TakeNoRoom:
		return a.tr.UI("dungeon.noroom")
	case game.TakeGone:
		return a.tr.UI("dungeon.gone")
	case game.TakeFromData:
		return a.tr.Event(dungeonSourceFile,
			gamedata.DungeonItemFirstString+index*gamedata.DungeonItemFields+1, msg)
	}
	return ""
}

// dungeonSourceFile 是地城道具字串在翻譯目錄裡的 key。
//
// **它不是檔名。** 那 300 條躺在 `FILES.DTT` 裡（索引 164–463），
// 但 `FILES.DTT` 這個 key 已經被**法術名**用掉了（`cmd/dwstrings` 的
// `spellSource`）—— 與 `SKILLS`／`MONTHS` 同一個情況：字串同源、
// 索引語意不同，擠在同一個目錄裡只會讓人對錯條目，
// 而且 `dwstrings spells` 重生那個目錄時會把這些條目沖掉。
//
// 索引沿用 `FILES.DTT` 的**絕對條目編號**（第 i 件的 `+0` 欄 ＝ `164 + i×6`），
// 與 `cmd/dwstrings dungeonitems` 抽出來的目錄對齊。
//
// **300 條裡只有 112 條抽出來翻。** `+3`（座標）與 `+4`（另一件道具的名字，
// 是查表鍵）一條都不能翻，`*` 是佔位、`T`／`P`／`S` 的參數是數字 ——
// 判準與理由寫在 `cmd/dwstrings/dungeonitems.go`。
const dungeonSourceFile = "DUNGEONITEM"

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

	// 「用在哪」那一頁自成一格，先擋在前面。
	if d.onMenu {
		line(fmt.Sprintf(a.tr.UI("dungeon.use.on"),
			a.dungeonName(d.source.Name)))
		line("")
		for i, t := range useTargets {
			line(memberMark(d.cursor, i) + t.key + "：" +
				a.tr.UI("dungeon.use.target."+t.key))
		}
		line("")
		line(a.tr.UI("dungeon.keys"))
		return
	}

	switch {
	case d.mode.fromInventory():
		title := a.tr.UI("dungeon.drop.title")
		switch d.mode {
		case dungeonExamine:
			title = a.tr.UI("dungeon.examine.title")
		case dungeonUse:
			title = a.tr.UI("dungeon.use.title")
		case dungeonUseChar:
			title = fmt.Sprintf(a.tr.UI("dungeon.use.onwhat"),
				a.dungeonName(d.source.Name))
		}
		line(title)
		line("")
		for i, c := range d.carried {
			line(fmt.Sprintf("%s%s　%s",
				memberMark(d.cursor, i), a.dungeonName(c.Name),
				a.members[c.Member].Name))
		}
	case d.stage == 1:
		line(fmt.Sprintf(a.tr.UI("dungeon.take.who"),
			a.dungeonName(d.spots[d.pick].Item.Name)))
		line("")
		for i := range a.members {
			c := &a.members[i]
			line(fmt.Sprintf(a.tr.UI("dungeon.member.freeslots"),
				memberMark(d.cursor, i), c.Name, freeSlots(c)))
		}
	default:
		title := a.tr.UI("dungeon.take.title")
		switch d.mode {
		case dungeonMove:
			title = a.tr.UI("dungeon.move.title")
		case dungeonUseRoom:
			title = fmt.Sprintf(a.tr.UI("dungeon.use.onwhat"),
				a.dungeonName(d.source.Name))
		case dungeonViewItem:
			title = a.tr.UI("viewitem.title")
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
	line(a.tr.UI("dungeon.keys"))
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
