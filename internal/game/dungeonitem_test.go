package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// --- 測試用的小資料集 ---

func testDungeonItems() gamedata.DungeonItems {
	return gamedata.DungeonItems{
		{Name: "Iron key"},
		{Name: "Bed", Immovable: "It is too heavy"},
		{Name: "Bookcase", Immovable: "*"},
		{Name: "Mallet"},
	}
}

// testLocTable 造一張全空的位置表，再把指定的幾筆填進去。
func testLocTable(recs map[int]scenario.ItemLoc) *scenario.ItemLocTable {
	raw := make([]byte, scenario.ItemLocFileSize)
	t, err := scenario.ParseItemLoc(raw)
	if err != nil {
		panic(err)
	}
	for i, r := range recs {
		t.Records[i] = r
	}
	return t
}

// taker 造一個道具欄**全空**的角色。
//
// `Character{}` 的零值每一格型別都是 0 —— 那是 ITEMS.DAT 的第 0 件，
// 不是空格（空格是 `0xff`）。少了這一步，FreeSlot 會說「滿了」。
func taker() *Character {
	c := &Character{Name: "拿東西的人"}
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: scenario.SlotEmpty}
	}
	return c
}

// --- ItemsUnderfoot ---

// 只列出腳下這一格的，而且帶得回索引。
func TestItemsUnderfootMatchesAllThreeCoordinates(t *testing.T) {
	tab := testLocTable(map[int]scenario.ItemLoc{
		0: {X: 10, Y: 20, MapID: 3},
		1: {X: 10, Y: 20, MapID: 4}, // 子地圖不同
		2: {X: 11, Y: 20, MapID: 3}, // X 不同
		3: {X: 10, Y: 20, MapID: 3},
	})
	got := ItemsUnderfoot(tab, testDungeonItems(), 10, 20, 3)
	if len(got) != 2 {
		t.Fatalf("列出 %d 件，預期 2 件", len(got))
	}
	if got[0].Index != 0 || got[1].Index != 3 {
		t.Errorf("索引 = %d／%d，預期 0／3", got[0].Index, got[1].Index)
	}
	if got[0].Item.Name != "Iron key" {
		t.Errorf("第一件是 %q，預期 Iron key —— 位置表與內容表沒對齊",
			got[0].Item.Name)
	}
}

// 上限 9 件（原版 `cmp [bp-2],9`）。
func TestItemsUnderfootStopsAtNine(t *testing.T) {
	recs := map[int]scenario.ItemLoc{}
	for i := 0; i < 20; i++ {
		recs[i] = scenario.ItemLoc{X: 5, Y: 5, MapID: 1}
	}
	items := make(gamedata.DungeonItems, 50)
	got := ItemsUnderfoot(testLocTable(recs), items, 5, 5, 1)
	if len(got) != gamedata.DungeonItemsPerTile {
		t.Errorf("列出 %d 件，上限應該是 %d", len(got), gamedata.DungeonItemsPerTile)
	}
}

// 被拿走的（子地圖 0）不參與比對 —— 這一條沒有的話，
// 站在 (0,0) 就會看到全部被拿走的東西。
func TestItemsUnderfootIgnoresTakenRecords(t *testing.T) {
	tab := testLocTable(map[int]scenario.ItemLoc{0: {X: 0, Y: 0, MapID: 0}})
	if got := ItemsUnderfoot(tab, testDungeonItems(), 0, 0, 0); len(got) != 0 {
		t.Errorf("在 (0,0) 子地圖 0 列出 %d 件，預期 0", len(got))
	}
}

// --- TakeDungeonItem ---

func TestTakeMovesItemFromMapToSlot(t *testing.T) {
	tab := testLocTable(map[int]scenario.ItemLoc{0: {X: 10, Y: 20, MapID: 3}})
	c := taker()
	res := TakeDungeonItem(c, tab, testDungeonItems(), 0)
	if !res.OK {
		t.Fatalf("拿不起來：%v", res.Refusal)
	}
	if res.Slot != 0 {
		t.Errorf("收進第 %d 格，預期第 0 格", res.Slot)
	}
	got := c.Inventory[0]
	if !got.Dungeon() {
		t.Errorf("槽型別 = %#x，預期 %#x", got.Type, scenario.SlotDungeon)
	}
	if got.DungeonName != "Iron key" {
		t.Errorf("槽裡的名字 = %q，預期 Iron key", got.DungeonName)
	}
	if !tab.Taken(0) {
		t.Error("拿走了卻還留在地圖上")
	}
}

// **三個 byte 全清** —— 原版 `Take:` 連寫三個 0（`0x199f9`–`0x19a10`），
// 不是只清子地圖那一個（那是 `N` 動作的作法）。
func TestTakeZeroesTheWholeRecord(t *testing.T) {
	tab := testLocTable(map[int]scenario.ItemLoc{0: {X: 10, Y: 20, MapID: 3}})
	TakeDungeonItem(taker(), tab, testDungeonItems(), 0)
	if r := tab.Records[0]; r.X != 0 || r.Y != 0 || r.MapID != 0 {
		t.Errorf("拿走之後 = (%d,%d,%d)，預期 (0,0,0)", r.X, r.Y, r.MapID)
	}
}

// `+1` 欄有話 → 印那句話，而且**東西留在原地**。
func TestTakeRefusesImmovableAndLeavesItAlone(t *testing.T) {
	tab := testLocTable(map[int]scenario.ItemLoc{1: {X: 1, Y: 1, MapID: 1}})
	c := taker()
	res := TakeDungeonItem(c, tab, testDungeonItems(), 1)
	if res.OK {
		t.Fatal("It is too heavy 的東西被拿走了")
	}
	if res.Refusal != TakeFromData || res.Message != "It is too heavy" {
		t.Errorf("理由 = %v／%q，預期 TakeFromData ＋ 資料裡那一句",
			res.Refusal, res.Message)
	}
	if tab.Taken(1) {
		t.Error("拒絕了卻把地圖上那一筆清掉 —— 東西憑空消失")
	}
	if !c.Inventory[0].Empty() {
		t.Error("拒絕了卻還是塞進道具欄")
	}
}

// `*` 是佔位不是台詞 —— 印通用句，不要把星號丟到畫面上。
func TestTakeStarPrintsGenericRefusal(t *testing.T) {
	tab := testLocTable(map[int]scenario.ItemLoc{2: {X: 1, Y: 1, MapID: 1}})
	res := TakeDungeonItem(taker(), tab, testDungeonItems(), 2)
	if res.OK {
		t.Fatal("`*` 的東西被拿走了")
	}
	if res.Refusal != TakeSilent {
		t.Errorf("理由 = %v，預期 TakeSilent —— 星號是佔位不是台詞", res.Refusal)
	}
	if res.Message != "" {
		t.Errorf("星號被當成台詞帶出來了：%q", res.Message)
	}
}

// 道具欄滿了 → 拿不起來，**而且東西留在地上**。
func TestTakeWithNoRoomLeavesItOnTheGround(t *testing.T) {
	tab := testLocTable(map[int]scenario.ItemLoc{0: {X: 1, Y: 1, MapID: 1}})
	c := taker()
	for i := range c.Inventory {
		c.Inventory[i] = scenario.InventorySlot{Type: 1}
	}
	res := TakeDungeonItem(c, tab, testDungeonItems(), 0)
	if res.OK {
		t.Fatal("道具欄滿了還拿得起來")
	}
	if res.Refusal != TakeNoRoom {
		t.Errorf("理由 = %v，預期 TakeNoRoom", res.Refusal)
	}
	if tab.Taken(0) {
		t.Error("拿不起來卻把地圖上那一筆清掉 —— 道具永久消失")
	}
}

// 已經被拿走的不能再拿一次（重複觸發、或介面沒刷新時的保險）。
func TestTakeRefusesAlreadyTaken(t *testing.T) {
	tab := testLocTable(nil) // 全部 (0,0,0)
	if res := TakeDungeonItem(taker(), tab, testDungeonItems(), 0); res.OK {
		t.Error("已經被拿走的又拿了一次 —— 可以無限複製")
	}
}

// --- DropDungeonItem ---

func TestDropPutsItBackAtCurrentPosition(t *testing.T) {
	items := testDungeonItems()
	tab := testLocTable(map[int]scenario.ItemLoc{0: {X: 10, Y: 20, MapID: 3}})
	c := taker()
	TakeDungeonItem(c, tab, items, 0)

	res := DropDungeonItem(c, tab, items, 0, 44, 55, 2)
	if !res.OK {
		t.Fatal("丟不掉")
	}
	if res.Index != 0 {
		t.Errorf("寫回第 %d 筆，預期第 0 筆（靠名字回推）", res.Index)
	}
	if r := tab.Records[0]; r.X != 44 || r.Y != 55 || r.MapID != 2 {
		t.Errorf("位置 = (%d,%d,%d)，預期 (44,55,2)", r.X, r.Y, r.MapID)
	}
	if !c.Inventory[0].Empty() {
		t.Error("丟掉了槽卻沒清空 —— 道具被複製了一份")
	}
}

// 拿起來再丟下去，**撿得回來**。手冊「一定能在原地撿回」那一條。
func TestTakeDropTakeRoundTrip(t *testing.T) {
	items := testDungeonItems()
	tab := testLocTable(map[int]scenario.ItemLoc{0: {X: 10, Y: 20, MapID: 3}})
	c := taker()

	TakeDungeonItem(c, tab, items, 0)
	DropDungeonItem(c, tab, items, 0, 7, 8, 1)

	under := ItemsUnderfoot(tab, items, 7, 8, 1)
	if len(under) != 1 || under[0].Index != 0 {
		t.Fatalf("丟下去之後腳下有 %d 件，預期 1 件（索引 0）", len(under))
	}
	if res := TakeDungeonItem(c, tab, items, 0); !res.OK {
		t.Fatalf("撿不回來：%v", res.Refusal)
	}
	if c.Inventory[0].DungeonName != "Iron key" {
		t.Errorf("撿回來的是 %q", c.Inventory[0].DungeonName)
	}
}

// 一般道具不能走這條路 —— 那是營地的 DropItem 管的。
func TestDropRejectsOrdinaryItems(t *testing.T) {
	c := taker()
	c.Inventory[0] = scenario.InventorySlot{Type: 3}
	tab := testLocTable(nil)
	if res := DropDungeonItem(c, tab, testDungeonItems(), 0, 1, 1, 1); res.OK {
		t.Error("一般道具被當成地城道具丟到地上了")
	}
}

// 名字對不上就拒絕。原版沒有這道檢查，會寫到有效區之外。
func TestDropRejectsUnknownName(t *testing.T) {
	c := taker()
	c.Inventory[0] = scenario.NewDungeonSlot("不存在的東西")
	tab := testLocTable(nil)
	res := DropDungeonItem(c, tab, testDungeonItems(), 0, 1, 1, 1)
	if res.OK {
		t.Fatal("名字認不出來卻還是丟出去了")
	}
	if !c.Inventory[0].Dungeon() {
		t.Error("丟失敗卻把槽清掉了 —— 道具憑空消失")
	}
}

// --- CarriedDungeonItems ---

func TestCarriedDungeonItemsListsWholeParty(t *testing.T) {
	items := testDungeonItems()
	party := []Character{*taker(), *taker()}
	party[0].Name, party[1].Name = "甲", "乙"
	party[0].Inventory[2] = scenario.NewDungeonSlot("Iron key")
	party[0].Inventory[3] = scenario.InventorySlot{Type: 1} // 一般道具，不算
	party[1].Inventory[0] = scenario.NewDungeonSlot("Mallet")

	got := CarriedDungeonItems(party, items)
	if len(got) != 2 {
		t.Fatalf("列出 %d 件，預期 2 件", len(got))
	}
	if got[0].Member != 0 || got[0].Slot != 2 || got[0].Index != 0 {
		t.Errorf("第一件 = %+v，預期 甲 的第 2 格、索引 0", got[0])
	}
	if got[1].Member != 1 || got[1].Index != 3 {
		t.Errorf("第二件 = %+v，預期 乙 的 Mallet（索引 3）", got[1])
	}
}

// 名字對不上時索引是 −1，而不是靜靜指到第 0 件。
func TestCarriedDungeonItemsMarksUnknownName(t *testing.T) {
	party := []Character{*taker()}
	party[0].Inventory[0] = scenario.NewDungeonSlot("殘值")
	got := CarriedDungeonItems(party, testDungeonItems())
	if len(got) != 1 || got[0].Index != -1 {
		t.Errorf("認不出的名字 index = %v，預期 −1", got)
	}
}

// --- ExamineDungeonItem ---

// `+2` 欄有話就回那句。
func TestExamineReturnsTheLookText(t *testing.T) {
	items := gamedata.DungeonItems{
		{Name: "Old bookcase", Look: "It has been moved to reveal a passage"},
		{Name: "Mallet"}, // `+2` 空著
	}
	if got, ok := ExamineDungeonItem(items, "Old bookcase"); !ok ||
		got != "It has been moved to reveal a passage" {
		t.Errorf("檢視得到 %q／%v", got, ok)
	}
	// `+2` 空著 → 沒話可說，訊息由介面組（要塞譯名進去）。
	if got, ok := ExamineDungeonItem(items, "Mallet"); ok || got != "" {
		t.Errorf("空的 +2 欄回了 %q／%v", got, ok)
	}
	// 名字認不出來也是「沒話可說」，不要 panic 也不要回第 0 件的敘述。
	if _, ok := ExamineDungeonItem(items, "殘值"); ok {
		t.Error("認不出的名字卻回了敘述")
	}
}
