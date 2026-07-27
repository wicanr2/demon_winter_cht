package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func classMember(c gamedata.Class, st scenario.CombatStatus) Character {
	return Character{Name: "某人", Class: c, Status: st, CurrentHP: 10}
}

// 十間試煉室的索引 ＝ 職業 id。這一條把三張原版表的共同索引釘住：
// 座標表、職業名表、參數表都靠它對齊，一錯就全錯。
func TestProvingRoomIndexIsTheClassID(t *testing.T) {
	for i, r := range ProvingRooms {
		if int(r.Class) != i {
			t.Errorf("第 %d 間的職業是 %d —— 索引必須等於職業 id", i, r.Class)
		}
	}
	// `2SS.DAT` 類別 5 值 9 的十筆座標（`docs/re/101` §2.1）。
	for _, c := range []struct {
		x, y int
		want gamedata.Class
	}{
		{49, 12, gamedata.Ranger}, {35, 37, gamedata.Paladin},
		{49, 5, gamedata.Barbarian}, {56, 31, gamedata.Monk},
		{35, 12, gamedata.Cleric}, {42, 13, gamedata.Thief},
		{49, 37, gamedata.Wizard}, {42, 5, gamedata.Sorcerer},
		{28, 31, gamedata.Visionary}, {35, 5, gamedata.Scholar},
	} {
		idx := ProvingRoomAt(c.x, c.y)
		if idx < 0 || ProvingRooms[idx].Class != c.want {
			t.Errorf("(%d,%d) → 第 %d 間，預期職業 %d", c.x, c.y, idx, c.want)
		}
	}
	if ProvingRoomAt(0, 0) != -1 {
		t.Error("不在那十格的座標應該回 −1")
	}
}

// 打的七間與只給一句話的三間。**盜賊／靈視者／學者不用打。**
func TestProvingTrialKinds(t *testing.T) {
	fights := 0
	for i, r := range ProvingRooms {
		if r.Trial == ProvingFight {
			fights++
			if len(r.Monsters) == 0 {
				t.Errorf("第 %d 間要打卻沒有怪", i)
			}
			continue
		}
		if len(r.Monsters) != 0 {
			t.Errorf("第 %d 間不打卻掛了怪 %v", i, r.Monsters)
		}
	}
	if fights != 7 {
		t.Errorf("要打的有 %d 間，預期 7", fights)
	}
	// 牧師那間是全十間唯一的多打一：5 隻僵屍 ＋ 2 隻鬼魂。
	cleric := ProvingRooms[gamedata.Cleric].Monsters
	if len(cleric) != 7 {
		t.Fatalf("牧師那間有 %d 隻怪，預期 7", len(cleric))
	}
	for i, id := range cleric {
		want := 17
		if i >= 5 {
			want = 22
		}
		if id != want {
			t.Errorf("牧師那間第 %d 隻是 %d，預期 %d", i, id, want)
		}
	}
	// 武僧對上 Karate master（95）—— 這一格就足以證明索引沒有偏移。
	if got := ProvingRooms[gamedata.Monk].Monsters; len(got) != 1 || got[0] != 95 {
		t.Errorf("武僧那間的怪是 %v，預期 [95]（Karate master）", got)
	}
}

// 三條入場路徑。門檻是「戰鬥狀態 > 中毒」才算倒下。
func TestEnterProvingRoom(t *testing.T) {
	monk := int(gamedata.Monk)

	// 隊伍裡沒有武僧 → 直接放行。
	got, fighters := EnterProvingRoom(monk, []Character{
		classMember(gamedata.Ranger, scenario.StatusNormal),
	})
	if got != ProvingFreePass || len(fighters) != 0 {
		t.Errorf("沒有那個職業：得到 %v／%v", got, fighters)
	}

	// 有武僧但死了 → 趕出去。
	got, _ = EnterProvingRoom(monk, []Character{
		classMember(gamedata.Monk, scenario.StatusDead),
	})
	if got != ProvingComeBackWhenWell {
		t.Errorf("人倒了：得到 %v", got)
	}

	// 中毒還算能上場（門檻 `> StatusPoison`）。
	got, fighters = EnterProvingRoom(monk, []Character{
		classMember(gamedata.Ranger, scenario.StatusNormal),
		classMember(gamedata.Monk, scenario.StatusPoison),
		classMember(gamedata.Monk, scenario.StatusDead),
	})
	if got != ProvingRunTrial {
		t.Fatalf("有能上場的：得到 %v", got)
	}
	if len(fighters) != 1 || fighters[0] != 1 {
		t.Errorf("上場的是 %v，預期只有第 1 個（中毒的武僧）", fighters)
	}
}

// 陣型從**格 3** 開始填，而且其他人一格都不佔。
func TestProvingFormationStartsAtCellThree(t *testing.T) {
	f := ProvingFormation([]int{2, 4})
	if f[provingFirstCell] != 2 || f[provingFirstCell+1] != 4 {
		t.Errorf("陣型 %v：前兩個上場的應該落在格 %d／%d",
			f, provingFirstCell, provingFirstCell+1)
	}
	for i, v := range f {
		if i == provingFirstCell || i == provingFirstCell+1 {
			continue
		}
		if v != FormationEmpty {
			t.Errorf("格 %d 應該是空的，卻是 %d", i, v)
		}
	}
	// 沒有人上場（不該發生）也不要留下垃圾。
	empty := ProvingFormation(nil)
	for i, v := range empty {
		if v != FormationEmpty {
			t.Errorf("空陣型的格 %d 是 %d", i, v)
		}
	}
}

// 十間全過才開寶珠那一格。
func TestProvingRoomsClearedGatesTheOrb(t *testing.T) {
	s := &scenario.SaveGame{}
	if ProvingRoomsCleared(s) {
		t.Error("一間都沒過就說全過了")
	}
	for i := 0; i < scenario.ProvingRoomCount-1; i++ {
		PassProvingRoom(s, i)
	}
	if ProvingRoomsCleared(s) {
		t.Error("差一間也算全過了")
	}
	if OrbAvailable(s) != OrbNotYet {
		t.Error("還沒全過，寶珠不該給")
	}
	PassProvingRoom(s, scenario.ProvingRoomCount-1)
	if !ProvingRoomsCleared(s) {
		t.Error("十間都過了卻說沒全過")
	}
	if OrbAvailable(s) != OrbTaken {
		t.Error("十間全過了卻拿不到寶珠")
	}
}

// 寶珠推的是**劇情階段**，不是第 7 格劇情道具旗標。
func TestTakeOrbAdvancesThePlotStage(t *testing.T) {
	s := &scenario.SaveGame{}
	for i := 0; i < scenario.ProvingRoomCount; i++ {
		PassProvingRoom(s, i)
	}
	c := taker()
	res, slot := TakeOrbOfEvertime(s, c)
	if res != OrbTaken || slot != 0 {
		t.Fatalf("拿寶珠：%v／槽 %d", res, slot)
	}
	if c.Inventory[0].Type != OrbItemType || !c.Inventory[0].Identified {
		t.Errorf("寶珠的欄位不對：%+v", c.Inventory[0])
	}
	if s.PlotStage != PlotArrivalDue {
		t.Errorf("劇情階段 %d，預期 %d（下次睡覺播預言）",
			s.PlotStage, PlotArrivalDue)
	}
	// 拿過了就不再給（閘門讀的就是劇情階段）。
	if r, _ := TakeOrbOfEvertime(s, taker()); r != OrbAlreadyTaken {
		t.Errorf("第二次拿得到寶珠：%v", r)
	}

	// **道具欄滿了就整件事不算** —— 階段不能先推，不然寶珠永遠拿不到。
	s2 := &scenario.SaveGame{}
	for i := 0; i < scenario.ProvingRoomCount; i++ {
		PassProvingRoom(s2, i)
	}
	full := taker()
	for i := range full.Inventory {
		full.Inventory[i] = scenario.InventorySlot{Type: 1}
	}
	if r, _ := TakeOrbOfEvertime(s2, full); r != OrbNoRoom {
		t.Errorf("欄位滿了卻回 %v", r)
	}
	if s2.PlotStage != PlotBeforeArrival {
		t.Error("放不下卻把劇情階段推掉了 —— 寶珠永遠拿不到了")
	}
}
