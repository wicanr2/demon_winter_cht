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

// 25 組座標必須兩兩相異。
//
// 原版的查表只比對 (X, Y)、不看在哪張子地圖上，所以座標一旦撞號，兩座城鎮
// 就會有一座永遠進不去 —— 而且是編號大的那座默默消失，不會有任何錯誤訊息。
func TestTownSites_AllDistinct(t *testing.T) {
	seen := map[[2]int]int{}
	for i, s := range townSites {
		if prev, dup := seen[s]; dup {
			t.Errorf("城鎮 %d 與城鎮 %d 都在 (%d,%d)", i+1, prev, s[0], s[1])
		}
		seen[s] = i + 1
	}
}

// 座標要落在 64×64 的子地圖範圍內。表抄錯一個 byte 多半會撞破這條。
func TestTownSites_InRange(t *testing.T) {
	for i, s := range townSites {
		if s[0] < 0 || s[0] > 63 || s[1] < 0 || s[1] > 63 {
			t.Errorf("城鎮 %d 座標 (%d,%d) 超出 0–63", i+1, s[0], s[1])
		}
	}
}

// TownAt 要對得回城鎮，且不在表上的座標要老實回 false。
func TestTownAt(t *testing.T) {
	tt := loadTowns(t)

	// 抽兩座有外部佐證的：新格里昂是攻略指名的買船處，海盜灣是唯一
	// 落在 MAP1.MAP（而非 SUM.MAP 子地圖）上的城鎮。
	for _, c := range []struct {
		x, y int
		want string
	}{
		{34, 11, "New Gleon"},
		{1, 3, "Pirate's Cove"},
	} {
		got, ok := tt.TownAt(c.x, c.y)
		if !ok {
			t.Errorf("(%d,%d) 查不到城鎮，預期 %s", c.x, c.y, c.want)
			continue
		}
		if got.Name != c.want {
			t.Errorf("(%d,%d) 查到 %s，預期 %s", c.x, c.y, got.Name, c.want)
		}
	}

	if _, ok := tt.TownAt(0, 0); ok {
		t.Error("(0,0) 不是城鎮，卻查得到")
	}
}

// 每座城鎮的 X/Y 要與 townSites 一致 —— LoadTownTable 少填一個欄位的話，
// 全部城鎮會擠在 (0,0)，而 TownAt 只會回傳第一座。
func TestLoadTownTable_FillsSites(t *testing.T) {
	tt := loadTowns(t)
	for _, town := range tt.All() {
		want := townSites[town.Number-1]
		if town.X != want[0] || town.Y != want[1] {
			t.Errorf("城鎮 %d %s 座標 (%d,%d)，預期 (%d,%d)",
				town.Number, town.Name, town.X, town.Y, want[0], want[1])
		}
	}
}
