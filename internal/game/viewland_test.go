package game

import "testing"

func viewChar(skill bool) *Character {
	c := &Character{Name: "斥候", EquippedWeapon: -1, EquippedArmor: -1}
	c.Skills[SkillViewLand] = skill
	return c
}

func TestCanViewLand_AllowsTheScoutOnTheWorldMap(t *testing.T) {
	if ok, why := CanViewLand(viewChar(true), 44, false); !ok {
		t.Fatalf("會觀地、在大地圖上、今天沒用過，應該可以：%s", why)
	}
}

func TestCanViewLand_Refusals(t *testing.T) {
	poisoned := viewChar(true)
	poisoned.Status = 2

	cases := []struct {
		name      string
		c         *Character
		mapID     int
		usedToday bool
		reason    string
	}{
		{"沒有這個人", nil, 44, false, "沒有這個人"},
		{"狀態太差", poisoned, 44, false, "現在爬不上高處"},
		{"不會觀地", viewChar(false), 44, false, "不會觀地"},
		{"今天用過了", viewChar(true), 44, true, "今天已經看過了"},
		{"在地城裡", viewChar(true), 10, false, "這裡看不到地形"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, why := CanViewLand(tc.c, tc.mapID, tc.usedToday)
			if ok {
				t.Fatal("預期擋下來")
			}
			if why != tc.reason {
				t.Errorf("理由 %q，預期 %q", why, tc.reason)
			}
		})
	}
}

// 分界正好落在 11：10 是地城、11 是世界格 (1,1)。
func TestCanViewLand_MapIDBoundary(t *testing.T) {
	if ok, _ := CanViewLand(viewChar(true), 11, false); !ok {
		t.Error("子地圖 11 是世界格，應該看得到")
	}
	if ok, _ := CanViewLand(viewChar(true), 10, false); ok {
		t.Error("子地圖 10 不是世界格，不該看得到")
	}
}

func TestViewLandStep_ClampsToTheMap(t *testing.T) {
	if x, y := ViewLandStep(0, 0, West); x != 0 || y != 0 {
		t.Errorf("左上角往西走到 (%d,%d)，預期原地", x, y)
	}
	if x, y := ViewLandStep(MapWidth-1, MapHeight-1, South); x != MapWidth-1 || y != MapHeight-1 {
		t.Errorf("右下角往南走到 (%d,%d)，預期原地", x, y)
	}
	if x, y := ViewLandStep(5, 5, East); x != 6 || y != 5 {
		t.Errorf("往東走到 (%d,%d)，預期 (6,5)", x, y)
	}
}
