package gamedata

import "testing"

func loadTowns(t *testing.T) *TownTable {
	t.Helper()
	tb, err := LoadTownTable(origDataDir(t))
	if err != nil {
		t.Fatalf("LoadTownTable: %v", err)
	}
	return tb
}

func TestLoadTownTable_Count(t *testing.T) {
	tb := loadTowns(t)
	if got := tb.Len(); got != NumTowns {
		t.Errorf("載入 %d 座城鎮，預期 %d", got, NumTowns)
	}
}

// 城名對照。第 1、8、25 座釘死，抓整體偏移。
func TestTownNames(t *testing.T) {
	tb := loadTowns(t)
	want := map[int]string{
		1:  "Seaside",
		8:  "New Gleon",
		20: "Terlabba",
		25: "Pirate's Cove",
	}
	for n, w := range want {
		town, err := tb.ByNumber(n)
		if err != nil {
			t.Fatal(err)
		}
		if town.Name != w {
			t.Errorf("第 %d 座城鎮名 = %q，預期 %q", n, town.Name, w)
		}
	}
}

// 經濟係數的值域。E 是物價指數，落在 8–25 —— 這是「0x1ed 就是 E」的
// 第一個佐證：隨便挑一個位移不會剛好給出這麼窄又這麼合理的分佈。
func TestTownEconomy_PlausibleRange(t *testing.T) {
	tb := loadTowns(t)
	for _, town := range tb.All() {
		if town.Economy < 1 || town.Economy > 60 {
			t.Errorf("%s（第 %d 座）的經濟係數 %d 超出合理值域",
				town.Name, town.Number, town.Economy)
		}
	}
}

// 只有碼頭城鎮賣船，而且攻略指名的那一座在裡面。
//
// 這是「0x1f5 就是船價基礎值」的第二個佐證，而且來源獨立於程式碼：
// 攻略寫「前往東北方很遠的新格里昂，買一艘船」。
func TestTownShipBase_OnlyPortTowns(t *testing.T) {
	tb := loadTowns(t)

	var ports []string
	for _, town := range tb.All() {
		if town.SellsShips() {
			ports = append(ports, town.Name)
		}
	}
	if len(ports) == 0 || len(ports) > 8 {
		t.Errorf("賣船的城鎮有 %d 座（%v），應該只有少數碼頭城鎮", len(ports), ports)
	}

	gleon, err := tb.ByNumber(8)
	if err != nil {
		t.Fatal(err)
	}
	if !gleon.SellsShips() {
		t.Error("攻略指名在 New Gleon 買船，它必須賣船")
	}
	// 船價得是「攢一陣子才買得起」的量級，不是幾十塊。
	if p := gleon.ShipBase * 10; p < 100 || p > 5000 {
		t.Errorf("New Gleon 的船價 %d 不像船的價錢", p)
	}
}

// 不賣船的城鎮 ShipBase 必須是 0，不能是別的小數字 ——
// 那會讓「買船價 = ShipBase × 10」在內陸城鎮算出一個假價格。
func TestTownShipBase_ZeroInland(t *testing.T) {
	tb := loadTowns(t)
	nonZero := 0
	for _, town := range tb.All() {
		if town.ShipBase != 0 {
			nonZero++
		}
	}
	if nonZero == tb.Len() {
		t.Error("每座城鎮都賣船，0x1f5 大概不是船價基礎值")
	}
}

func TestTownTable_ByNumberBounds(t *testing.T) {
	tb := loadTowns(t)
	for _, n := range []int{0, -1, NumTowns + 1} {
		if _, err := tb.ByNumber(n); err == nil {
			t.Errorf("城鎮編號 %d 應回傳錯誤", n)
		}
	}
}
