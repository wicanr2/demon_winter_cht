package game

import "testing"

// campChar 造一名帶指定道具型別的角色，未裝備任何東西。
func campChar(name string, types ...byte) *Character {
	c := &Character{Name: name, EquippedWeapon: -1, EquippedArmor: -1}
	for i := range c.Inventory {
		c.Inventory[i].Type = 0xff
	}
	for i, t := range types {
		c.Inventory[i].Type = t
		c.Inventory[i].Enchant = int(t) // 隨手放個可辨識的值，驗證整格搬過去
	}
	return c
}

func TestDropItem_ClearsSlot(t *testing.T) {
	c := campChar("Wopple", 3)
	res := DropItem(c, 0, 0)
	if !res.OK {
		t.Fatalf("丟棄失敗：%s", res.Reason)
	}
	if !c.Inventory[0].Empty() {
		t.Errorf("丟完之後型別 %#x，預期 0xff", c.Inventory[0].Type)
	}
}

func TestDropItem_Refusals(t *testing.T) {
	cases := []struct {
		name   string
		setup  func() (*Character, int, byte)
		reason string
	}{
		{"空格", func() (*Character, int, byte) { return campChar("A"), 0, 0 },
			"reason.slot.empty"},
		{"地城道具", func() (*Character, int, byte) { return campChar("A", ItemTypeDungeon), 0, 0 },
			"reason.item.dungeon_drop"},
		{"裝備中的武器", func() (*Character, int, byte) {
			c := campChar("A", 3)
			c.EquippedWeapon = 0
			return c, 0, 0
		}, "reason.item.equipped"},
		{"裝備中的護甲", func() (*Character, int, byte) {
			c := campChar("A", 9)
			c.EquippedArmor = 0
			return c, 0, 0
		}, "reason.item.equipped"},
		{"型別 0x1c 一律不可丟", func() (*Character, int, byte) {
			return campChar("A", itemTypeNoDrop), 0, 0xff
		}, "reason.item.drop_refused"},
		{"型別 0x1d 且旗標為 0", func() (*Character, int, byte) {
			return campChar("A", itemTypeGatedDrop), 0, 0
		}, "reason.item.drop_refused"},
		{"沒有這一格", func() (*Character, int, byte) { return campChar("A", 3), 10, 0 },
			"reason.slot.invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, slot, flag := tc.setup()
			before := c.Inventory
			res := DropItem(c, slot, flag)
			if res.OK {
				t.Fatal("預期拒絕，卻成功了")
			}
			if res.Reason != tc.reason {
				t.Errorf("原因 %q，預期 %q", res.Reason, tc.reason)
			}
			if c.Inventory != before {
				t.Error("拒絕之後道具欄不應該變動")
			}
		})
	}
}

// 型別 0x1d 的門檻旗標不為 0 就丟得掉 —— 這一條是 0x1c 與 0x1d 的唯一差別。
func TestDropItem_GatedTypeAllowedWhenFlagSet(t *testing.T) {
	c := campChar("A", itemTypeGatedDrop)
	if res := DropItem(c, 0, 1); !res.OK {
		t.Fatalf("旗標為 1 時應該丟得掉，卻被擋：%s", res.Reason)
	}
	c = campChar("A", itemTypeNoDrop)
	if res := DropItem(c, 0, 1); res.OK {
		t.Fatal("型別 0x1c 不受旗標影響，應該永遠丟不掉")
	}
}

func TestGiveItem_MovesWholeSlot(t *testing.T) {
	from := campChar("Wopple", 3, 7)
	to := campChar("Stumpy")
	want := from.Inventory[1]

	res := GiveItem(from, to, 1)
	if !res.OK {
		t.Fatalf("轉手失敗：%s", res.Reason)
	}
	if res.Slot != 0 {
		t.Errorf("收在第 %d 格，預期第 0 格（第一個空格）", res.Slot)
	}
	if to.Inventory[0] != want {
		t.Errorf("收到 %+v，預期 %+v", to.Inventory[0], want)
	}
	if !from.Inventory[1].Empty() {
		t.Error("來源那一格應該清空")
	}
}

// 地城道具丟不掉，但**給得出去** —— 原版的 Trade 完全沒有型別檢查。
func TestGiveItem_DungeonItemIsTradeable(t *testing.T) {
	from := campChar("A", ItemTypeDungeon)
	to := campChar("B")
	if res := GiveItem(from, to, 0); !res.OK {
		t.Fatalf("地城道具應該給得出去，卻被擋：%s", res.Reason)
	}
}

func TestGiveItem_Refusals(t *testing.T) {
	full := campChar("B", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	equipped := campChar("A", 3)
	equipped.EquippedWeapon = 0

	cases := []struct {
		name     string
		from, to *Character
		slot     int
		reason   string
	}{
		{"收方滿了", campChar("A", 3), full, 0, "reason.inventory.member_full"},
		{"空格", campChar("A"), campChar("B"), 0, "reason.slot.empty"},
		{"裝備中", equipped, campChar("B"), 0, "reason.item.equipped"},
		{"沒有這一格", campChar("A", 3), campChar("B"), -1, "reason.slot.invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.from.Inventory
			res := GiveItem(tc.from, tc.to, tc.slot)
			if res.OK {
				t.Fatal("預期拒絕，卻成功了")
			}
			if res.Reason != tc.reason {
				t.Errorf("原因 %q，預期 %q", res.Reason, tc.reason)
			}
			if tc.from.Inventory != before {
				t.Error("拒絕之後來源道具欄不應該變動")
			}
		})
	}
}

func TestGiveItem_SelfIsRefused(t *testing.T) {
	c := campChar("A", 3)
	if res := GiveItem(c, c, 0); res.OK {
		t.Fatal("不應該可以給自己")
	}
}

func TestGiveItem_FullCarriesNameAsData(t *testing.T) {
	full := campChar("收件人", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	res := GiveItem(campChar("寄件人", 3), full, 0)
	if res.Reason != "reason.inventory.member_full" ||
		len(res.ReasonArgs) != 1 || res.ReasonArgs[0] != "收件人" {
		t.Fatalf("滿載原因應拆成 key＋名稱參數，得到 %#v／%#v",
			res.Reason, res.ReasonArgs)
	}
}
