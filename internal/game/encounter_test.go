package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// 1/64：原版是對 RNG 原始輸出做 `& 0x3f == 0x34`，不是 rnd(64)。
func TestEncounterTriggered(t *testing.T) {
	if !EncounterTriggered(0x34) {
		t.Error("0x34 應該命中")
	}
	// 遮罩只看低 6 bit，高位不影響。
	for _, v := range []int{0x74, 0xb4, 0x1234 &^ 0x3f | 0x34} {
		if !EncounterTriggered(v) {
			t.Errorf("0x%x 的低 6 bit 是 0x34，應該命中", v)
		}
	}
	if EncounterTriggered(0x33) || EncounterTriggered(0x35) {
		t.Error("相鄰值不該命中")
	}
	// 命中率必須是 1/64。
	hits := 0
	for v := 0; v < 64; v++ {
		if EncounterTriggered(v) {
			hits++
		}
	}
	if hits != 1 {
		t.Errorf("64 個值裡命中 %d 次，預期 1", hits)
	}
}

// 隻數 = 8 − rnd(7)，值域 1–7。
func TestEncounterCount(t *testing.T) {
	for roll := 1; roll <= 7; roll++ {
		want := 8 - roll
		if got := EncounterCount(&fixedRolls{vals: []int{roll}}); got != want {
			t.Errorf("rnd(7) = %d：隻數 %d，預期 %d", roll, got, want)
		}
	}
	r := rng.NewWithSeed(3)
	for i := 0; i < 500; i++ {
		if n := EncounterCount(r); n < 1 || n > 7 {
			t.Fatalf("隻數 %d 落在 1–7 之外", n)
		}
	}
}

// 擲出來的怪物一定來自該地形掛的群組，而且數量落在 1–7。
func TestRollEncounter_StaysInTerrain(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(17)

	for terrain := gamedata.Terrain(0); terrain < gamedata.NumTerrains; terrain++ {
		allowed := map[int]bool{}
		slots, err := tb.TerrainGroups(terrain)
		if err != nil {
			t.Fatal(err)
		}
		for _, gi := range slots {
			g, err := tb.EncounterGroup(int(gi))
			if err != nil {
				continue
			}
			for _, e := range g.Entries {
				allowed[e.Monster] = true
			}
		}

		got := 0
		for level := 1; level <= 10; level++ {
			for i := 0; i < 60; i++ {
				mons := RollEncounter(r, tb, terrain, level)
				if mons == nil {
					continue
				}
				got++
				if len(mons) < 1 || len(mons) > 7 {
					t.Fatalf("地形 %d 擲出 %d 隻，超出 1–7", terrain, len(mons))
				}
				for _, m := range mons {
					if !allowed[m] {
						t.Fatalf("地形 %d（%s）擲出怪物 %d，不在它的群組裡",
							terrain, terrain.Name(), m)
					}
				}
			}
		}
		if got == 0 {
			t.Errorf("地形 %d（%s）在所有難度下都擲不出遭遇", terrain, terrain.Name())
		}
	}
}

// 沼澤那一組十筆全是「清一色群」，所以出來的一定同種。
//
// 這條驗的是 0x2c9e 的鎖定邏輯 —— 少了它，玩家會在沼澤看到
// 鬼火跟鬼墳族混在一起，而原版不會。
func TestRollEncounter_SwampPacksAreUniform(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(23)

	seen := 0
	for level := 1; level <= 10; level++ {
		for i := 0; i < 200; i++ {
			mons := RollEncounter(r, tb, gamedata.TerrainSwamp, level)
			if len(mons) < 2 {
				continue
			}
			// 沼澤組（第 13 組）出來的必須同種；其他組（昆蟲、毒蛇等）不限。
			if !swampOnly(tb, mons) {
				continue
			}
			seen++
			for _, m := range mons[1:] {
				if m != mons[0] {
					t.Fatalf("沼澤群出現混雜：%v", mons)
				}
			}
		}
	}
	if seen == 0 {
		t.Error("一次都沒擲到沼澤專屬群，測試沒驗到東西")
	}
}

// swampOnly 回報這批怪是不是全來自沼澤專屬那一組。
func swampOnly(tb *gamedata.Tables, mons []int) bool {
	g, err := tb.EncounterGroup(13)
	if err != nil {
		return false
	}
	in := map[int]bool{}
	for _, e := range g.Entries {
		in[e.Monster] = true
	}
	for _, m := range mons {
		if !in[m] {
			return false
		}
	}
	return true
}

// 難度會改變遇得到的東西 —— 低難度不該冒出高等級的怪。
func TestRollEncounter_ScalesWithLevel(t *testing.T) {
	tb := loadTables(t)
	r := rng.NewWithSeed(31)

	lowest := func(level int) map[int]bool {
		out := map[int]bool{}
		for i := 0; i < 400; i++ {
			for _, m := range RollEncounter(r, tb, gamedata.TerrainPlains, level) {
				out[m] = true
			}
		}
		return out
	}
	low, high := lowest(1), lowest(9)
	if len(low) == 0 || len(high) == 0 {
		t.Fatal("兩個難度都要擲得出東西")
	}
	// 完全一樣就代表等級檢查沒生效。
	same := 0
	for m := range low {
		if high[m] {
			same++
		}
	}
	if same == len(low) && len(low) == len(high) {
		t.Error("難度 1 與 9 遇到的怪完全相同，等級檢查沒生效")
	}
}

func TestRollEncounter_NilArgs(t *testing.T) {
	tb := loadTables(t)
	if RollEncounter(nil, tb, gamedata.TerrainPlains, 1) != nil {
		t.Error("沒有擲點來源卻擲得出遭遇")
	}
	if RollEncounter(rng.NewWithSeed(1), nil, gamedata.TerrainPlains, 1) != nil {
		t.Error("沒有資料表卻擲得出遭遇")
	}
	if RollEncounter(rng.NewWithSeed(1), tb, gamedata.NumTerrains, 1) != nil {
		t.Error("地形超出範圍卻擲得出遭遇")
	}
}
