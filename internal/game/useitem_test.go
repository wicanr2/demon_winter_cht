package game

import "testing"

func TestLightSourceLevel_TorchAndLantern(t *testing.T) {
	cases := map[byte]int{ItemTypeTorch: TorchLight, ItemTypeLantern: LanternLight, 3: 0, 25: 0, 28: 0}
	for typ, want := range cases {
		if got := LightSourceLevel(typ); got != want {
			t.Errorf("型別 %d 的光源等級 %d，預期 %d", typ, got, want)
		}
	}
	if LanternLight <= TorchLight {
		t.Error("提燈應該比火把亮")
	}
}

func TestUseInCamp_TorchIsConsumed(t *testing.T) {
	c := campChar("A", ItemTypeTorch)
	res := UseInCamp(c, 0)
	if !res.OK {
		t.Fatalf("火把應該用得掉：%s", res.Reason)
	}
	if res.Light != TorchLight || !res.Consumed {
		t.Errorf("結果 %+v，預期光源 %d、道具消失", res, TorchLight)
	}
	if !c.Inventory[0].Empty() {
		t.Error("用完之後那一格應該清空")
	}
}

func TestUseInCamp_LanternIsBrighter(t *testing.T) {
	c := campChar("A", ItemTypeLantern)
	if res := UseInCamp(c, 0); res.Light != LanternLight {
		t.Errorf("提燈點出光源 %d，預期 %d", res.Light, LanternLight)
	}
}

// 裝備中的光源用掉之後，裝備索引要跟著失效。
func TestUseInCamp_ClearsEquipIndex(t *testing.T) {
	c := campChar("A", ItemTypeTorch)
	c.EquippedWeapon = 0
	UseInCamp(c, 0)
	if c.EquippedWeapon == 0 {
		t.Error("用掉的那一格還被當成裝備中")
	}
}

func TestUseInCamp_Refusals(t *testing.T) {
	stunned := campChar("A", ItemTypeTorch)
	stunned.Status = 2

	// 有效果但施法還離不開戰鬥。
	effect := campChar("A", 25)
	effect.Inventory[0].Power = 5
	effect.Inventory[0].Total = 3

	// 平凡裝備：強度 0，原版的過濾就擋掉了。
	plain := campChar("A", 3)

	// 次數用完。
	spent := campChar("A", 25)
	spent.Inventory[0].Power = 5
	spent.Inventory[0].Total = 3
	spent.Inventory[0].Used = 3

	cases := []struct {
		name   string
		c      *Character
		slot   int
		reason string
	}{
		{"沒有這個人", nil, 0, "沒有這個人"},
		{"狀態太差", stunned, 0, "現在沒辦法用東西"},
		{"空格", campChar("A"), 0, "這一格是空的"},
		{"沒有這一格", campChar("A", ItemTypeTorch), 10, "沒有這一格"},
		{"平凡裝備", plain, 0, "這件現在用不了"},
		{"次數用完", spent, 0, "這件現在用不了"},
		{"有效果的道具", effect, 0, "這件要在戰鬥中才用得出來"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if res := UseInCamp(tc.c, tc.slot); res.OK {
				t.Fatal("預期擋下來")
			} else if res.Reason != tc.reason {
				t.Errorf("理由 %q，預期 %q", res.Reason, tc.reason)
			}
		})
	}
}
