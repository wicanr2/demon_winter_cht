package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	worlddata "github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

// oceanEverywhere 是「四周都是海」的地形，用來測放船的位置選擇。
func oceanEverywhere(int, int) byte { return tileOceanA }

// landEverywhere 是「四周都是陸地」。
func landEverywhere(int, int) byte { return 0 }

func emptyShips() *[ShipSlots]scenario.Ship { return &[ShipSlots]scenario.Ship{} }

// 空格的判定是**船體值為 0**，不是座標為 0 —— (0,0) 是合法位置。
func TestShipExists_UsesHullNotCoords(t *testing.T) {
	at00 := scenario.Ship{X: 0, Y: 0, Hull: 40}
	if !at00.Exists() {
		t.Error("停在 (0,0) 的船被當成空格了")
	}
	blank := scenario.Ship{X: 12, Y: 34}
	if blank.Exists() {
		t.Error("船體 0 的格子應該是空的")
	}
}

// 買船：放進第一個可行的方向，扣錢，船體滿值。
func TestBuyShip(t *testing.T) {
	ships := emptyShips()
	res := BuyShip(ships, oceanEverywhere, 20, 30, 44, 1000, 570)
	if !res.OK || res.Gold != 430 || res.Slot != 0 {
		t.Fatalf("買船：%+v", res)
	}
	got := ships[0]
	// 四個方向的順序是西、東、北、南 —— 第一個就是西邊。
	want := scenario.Ship{X: 19, Y: 30, MapID: 44, Hull: scenario.ShipMaxHull, Unknown4: 2}
	if got != want {
		t.Errorf("放下的船 %+v，預期 %+v", got, want)
	}
}

// 陸地上放不了船。
func TestBuyShip_NeedsWater(t *testing.T) {
	ships := emptyShips()
	res := BuyShip(ships, landEverywhere, 20, 30, 44, 1000, 570)
	if res.OK {
		t.Error("四周都是陸地卻放下了船")
	}
	if res.Gold != 1000 {
		t.Errorf("沒放成卻扣了錢，剩 %d", res.Gold)
	}
}

// 只有一格是海就放那一格。
func TestBuyShip_PicksTheOnlyWater(t *testing.T) {
	ships := emptyShips()
	// 只有南邊 (20,31) 是海。
	tileAt := func(x, y int) byte {
		if x == 20 && y == 31 {
			return tileOceanB
		}
		return 0
	}
	res := BuyShip(ships, tileAt, 20, 30, 44, 1000, 100)
	if !res.OK {
		t.Fatalf("南邊是海卻沒放成：%+v", res)
	}
	if ships[0].X != 20 || ships[0].Y != 31 {
		t.Errorf("船放在 (%d,%d)，預期 (20,31)", ships[0].X, ships[0].Y)
	}
}

// 同一張子地圖的同一格不能疊兩艘；換一張圖就可以。
func TestBuyShip_NoStacking(t *testing.T) {
	ships := emptyShips()
	ships[3] = scenario.Ship{X: 19, Y: 30, MapID: 44, Hull: 10}

	// 西邊被佔了 → 退到東邊。
	res := BuyShip(ships, oceanEverywhere, 20, 30, 44, 1000, 100)
	if !res.OK || ships[res.Slot].X != 21 {
		t.Fatalf("西邊有船就該改放東邊，得到 %+v／%+v", res, ships[res.Slot])
	}

	// 同座標但不同子地圖 —— 不算佔用。
	ships2 := emptyShips()
	ships2[3] = scenario.Ship{X: 19, Y: 30, MapID: 45, Hull: 10}
	res2 := BuyShip(ships2, oceanEverywhere, 20, 30, 44, 1000, 100)
	if !res2.OK || ships2[res2.Slot].X != 19 {
		t.Errorf("別張地圖的船不該擋住這裡，得到 %+v", res2)
	}
}

func TestBuyShip_AllSlotsUsed(t *testing.T) {
	ships := emptyShips()
	for i := range ships {
		ships[i] = scenario.Ship{X: byte(i), Y: 99, MapID: 44, Hull: 10}
	}
	res := BuyShip(ships, oceanEverywhere, 20, 30, 44, 1000, 100)
	if res.OK {
		t.Error("10 格全滿卻還買得到船")
	}
	if res.Gold != 1000 {
		t.Error("沒買成卻扣了錢")
	}
}

func TestBuyShip_NotEnoughGold(t *testing.T) {
	ships := emptyShips()
	if res := BuyShip(ships, oceanEverywhere, 20, 30, 44, 100, 570); res.OK {
		t.Error("錢不夠卻買到船")
	}
	if ships[0].Exists() {
		t.Error("沒付錢卻放下了船")
	}
}

// 「在腳邊」是含斜角的 3×3。
func TestFindShipNear(t *testing.T) {
	ships := emptyShips()
	ships[5] = scenario.Ship{X: 21, Y: 31, MapID: 44, Hull: 40}

	for _, c := range []struct {
		x, y int
		want int
	}{
		{20, 30, 5}, // 斜對角
		{21, 31, 5}, // 同格
		{22, 32, 5},
		{19, 31, -1}, // 差 2 格
		{21, 33, -1},
	} {
		if got := FindShipNear(ships, c.x, c.y); got != c.want {
			t.Errorf("在 (%d,%d) 找到 %d，預期 %d", c.x, c.y, got, c.want)
		}
	}
}

// 空格不能被找到 —— 否則走到地圖角落會憑空修出一艘船。
func TestFindShipNear_IgnoresEmptySlots(t *testing.T) {
	if got := FindShipNear(emptyShips(), 0, 0); got != -1 {
		t.Errorf("在 (0,0) 找到空格 %d，那會憑空生出一艘船", got)
	}
}

// 修船：費用 =（75 − 船體）× E ÷ 2，修完滿值。
func TestRepairShip(t *testing.T) {
	ships := emptyShips()
	ships[2] = scenario.Ship{X: 20, Y: 30, MapID: 44, Hull: 55}

	e := testEconomy() // E = 30
	res := RepairShip(e, ships, 20, 30, 1000)
	if want := (75 - 55) * 30 / 2; !res.OK || res.Cost != want {
		t.Fatalf("修船費用 %d，預期 %d（%+v）", res.Cost, want, res)
	}
	if ships[2].Hull != scenario.ShipMaxHull {
		t.Errorf("修完船體 %d，預期 %d", ships[2].Hull, scenario.ShipMaxHull)
	}
	if res.Gold != 1000-res.Cost {
		t.Errorf("金幣 %d，預期 %d", res.Gold, 1000-res.Cost)
	}
}

func TestRepairShip_NothingToRepair(t *testing.T) {
	if res := RepairShip(testEconomy(), emptyShips(), 20, 30, 1000); res.OK {
		t.Error("腳邊沒船卻修好了")
	}
}

// 滿血的船不收錢。
func TestRepairShip_AlreadyFine(t *testing.T) {
	ships := emptyShips()
	ships[0] = scenario.Ship{X: 20, Y: 30, Hull: scenario.ShipMaxHull}
	res := RepairShip(testEconomy(), ships, 20, 30, 1000)
	if res.OK || res.Gold != 1000 {
		t.Errorf("滿血的船不該收錢：%+v", res)
	}
}

func TestRepairShip_NotEnoughGold(t *testing.T) {
	ships := emptyShips()
	ships[0] = scenario.Ship{X: 20, Y: 30, Hull: 5}
	res := RepairShip(testEconomy(), ships, 20, 30, 10)
	if res.OK {
		t.Error("錢不夠卻修好了")
	}
	if ships[0].Hull != 5 {
		t.Error("沒付錢卻修了船")
	}
}

// 原版存檔裡本來就有兩艘船 —— 解碼要認得出來。
func TestRealSave_HasTwoShips(t *testing.T) {
	save := loadParty(t)
	var found []int
	for i, s := range save.Ships {
		if s.Exists() {
			found = append(found, i)
		}
	}
	if len(found) != 2 {
		t.Fatalf("原版存檔應該有 2 艘船，找到 %d 艘（格號 %v）", len(found), found)
	}
	for _, i := range found {
		s := save.Ships[i]
		if s.Hull > scenario.ShipMaxHull {
			t.Errorf("第 %d 艘船體 %d 超過滿值 %d", i, s.Hull, scenario.ShipMaxHull)
		}
		// 兩艘都是原版放的，那個未解的 byte 都是 2。
		if s.Unknown4 != 2 {
			t.Errorf("第 %d 艘的 +0x04 = %d，兩份原版存檔都是 2", i, s.Unknown4)
		}
	}
}

// 搭船的存檔值是**格號 +1**，0 留給「沒搭船」。
func TestBoatValueAndIndex(t *testing.T) {
	if BoatValue(-1) != 0 {
		t.Error("沒搭船應該存 0")
	}
	for slot := 0; slot < ShipSlots; slot++ {
		v := BoatValue(slot)
		if v != byte(slot+1) {
			t.Errorf("格號 %d 的存檔值 %d，預期 %d", slot, v, slot+1)
		}
		if got := BoatIndex(v); got != slot {
			t.Errorf("存檔值 %d 換回格號 %d，預期 %d", v, got, slot)
		}
		if !Sailing(v) {
			t.Errorf("存檔值 %d 應該代表在船上", v)
		}
	}
	if Sailing(0) || BoatIndex(0) != -1 {
		t.Error("0 應該代表沒搭船")
	}
}

// 走到船那一格就上船，走回陸地就下船，中間船跟著隊伍走。
func TestStepBoat_BoardSailDisembark(t *testing.T) {
	ships := emptyShips()
	ships[4] = scenario.Ship{X: 10, Y: 10, MapID: 44, Hull: 50}

	// 走到 (10,10) → 上船。
	boat, res := StepBoat(ships, 0, tileOceanA, 10, 10, 44)
	if res != BoardOn || BoatIndex(boat) != 4 {
		t.Fatalf("上船失敗：res=%d boat=%d", res, boat)
	}

	// 在海上走一步 → 船跟著。
	boat, res = StepBoat(ships, boat, tileOceanB, 11, 10, 44)
	if res != BoardNone || !Sailing(boat) {
		t.Fatalf("海上航行不該下船：res=%d boat=%d", res, boat)
	}
	if ships[4].X != 11 || ships[4].Y != 10 {
		t.Errorf("船沒跟著走，停在 (%d,%d)", ships[4].X, ships[4].Y)
	}

	// 走上陸地 → 下船，船留在剛才那一格。
	boat, res = StepBoat(ships, boat, 0x01, 12, 10, 44)
	if res != BoardOff || Sailing(boat) {
		t.Fatalf("上岸失敗：res=%d boat=%d", res, boat)
	}
	if ships[4].X != 12 || ships[4].Y != 10 {
		t.Errorf("下船後船的座標 (%d,%d)，預期跟著寫回 (12,10)",
			ships[4].X, ships[4].Y)
	}
}

// 不在船上、腳下也沒船 —— 什麼都不該發生。
func TestStepBoat_NothingHappens(t *testing.T) {
	ships := emptyShips()
	boat, res := StepBoat(ships, 0, 0x01, 5, 5, 44)
	if res != BoardNone || boat != 0 {
		t.Errorf("不該有動作：res=%d boat=%d", res, boat)
	}
}

// 別張子地圖的船不能上。
func TestStepBoat_MapIDMatters(t *testing.T) {
	ships := emptyShips()
	ships[0] = scenario.Ship{X: 10, Y: 10, MapID: 45, Hull: 50}
	if _, res := StepBoat(ships, 0, tileOceanA, 10, 10, 44); res != BoardNone {
		t.Error("別張地圖的船不該上得了")
	}
}

func TestReachableBoatAt_CrossMapCoordinates(t *testing.T) {
	var ships [ShipSlots]scenario.Ship
	ships[1] = scenario.Ship{X: 4, Y: 20, MapID: 44, Hull: 50}
	ships[2] = scenario.Ship{X: 30, Y: 4, MapID: 45, Hull: 50}

	tests := []struct {
		name        string
		x, y, mapID int
		want        int
	}{
		{"同圖", 4, 20, 44, 1},
		{"地圖差10且X加56", 60, 20, 34, 1},
		{"地圖差10且X減56", 60, 20, 54, 1},
		{"地圖差1且Y加56", 30, 60, 44, 2},
		{"地圖差1且Y減56", 30, 60, 46, 2},
		{"錯軸不算", 60, 20, 43, -1},
		{"只差地圖不算", 4, 20, 43, -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ReachableBoatAt(&ships, tc.x, tc.y, tc.mapID); got != tc.want {
				t.Fatalf("ReachableBoatAt(%d,%d,map%d) = %d，預期 %d",
					tc.x, tc.y, tc.mapID, got, tc.want)
			}
		})
	}
}

func TestReachableBoatAt_ComposesWithCrossEdge(t *testing.T) {
	var ships [ShipSlots]scenario.Ship
	ships[3] = scenario.Ship{X: 4, Y: 20, MapID: 44, Hull: 50}

	// 隊伍在 map34 走到東界 x=60 時，原版會先以跨圖座標命中船，
	// 再把位置換成 map44 x=4；兩個判定必須落在同一艘。
	if got := ReachableBoatAt(&ships, 60, 20, 34); got != 3 {
		t.Fatalf("換圖前登船槽 = %d，預期 3", got)
	}
	cross := worlddata.CrossEdge(34, 60, 20)
	if !cross.Crossed || cross.MapID != 44 || cross.X != 4 || cross.Y != 20 {
		t.Fatalf("CrossEdge = %+v，預期 map44 (4,20)", cross)
	}
	if got := BoatAt(&ships, cross.X, cross.Y, cross.MapID); got != 3 {
		t.Fatalf("換圖後精確船槽 = %d，預期 3", got)
	}
}

// 兩個海面 tile 都算海 —— 只認一個的話，摻浪花那一格會被當成陸地下船。
func TestIsOcean_BothTiles(t *testing.T) {
	if !IsOcean(tileOceanA) || !IsOcean(tileOceanB) {
		t.Error("0x14 與 0x62 都是海面")
	}
	if IsOcean(0x01) {
		t.Error("陸地被當成海了")
	}
}
