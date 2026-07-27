package game

import "testing"

// 兩個門檻都含等於 —— 剛好 24 時的話「既能睡也能敲」。
func TestNightHourThresholdIncludesTwentyFour(t *testing.T) {
	for _, c := range []struct {
		hour int
		want BellRung
	}{
		{1, BellNothing}, {13, BellNothing}, {23, BellNothing},
		{24, BellOpens}, {25, BellOpens}, {37, BellOpens},
	} {
		if got := RingBell(c.hour); got != c.want {
			t.Errorf("%d 時敲鐘得到 %v，預期 %v", c.hour, got, c.want)
		}
	}
	// 床那一邊是 `<= 24`（`bellui.go` 用 `hour > NightHour` 才拒絕），
	// 所以 24 時是唯一「既能睡也能敲」的那一刻。
	if RingBell(NightHour) != BellOpens {
		t.Error("24 時應該敲得響")
	}
}

// 睡醒的時辰是寫死的 25，而且它剛好在鐘的門檻之後 ——
// 那條設計（先睡覺再敲鐘）靠的就是這個關係。
func TestSleepHourLandsAfterTheBellThreshold(t *testing.T) {
	if BellSleepHour <= NightHour-1 {
		t.Errorf("睡醒 %d 時，比鐘的門檻 %d 還早，那就敲不響了",
			BellSleepHour, NightHour)
	}
	if RingBell(BellSleepHour) != BellOpens {
		t.Errorf("睡醒（%d 時）之後應該敲得響", BellSleepHour)
	}
	// 睡覺只撥時鐘：不換日。原版就只有 `party[+0x9f] = 25` 一行。
	c := &Clock{hour: 10, day: 5, month: 3}
	c.SleepUntil(BellSleepHour, false)
	if c.Hour() != BellSleepHour || c.Day() != 5 || c.Month() != 3 {
		t.Errorf("睡完是 %d 月 %d 日 %d 時，預期只有時辰變成 %d",
			c.Month(), c.Day(), c.Hour(), BellSleepHour)
	}
}

// 那道門只是換 tile，而且 (21,59) 那一格從 0x3a（擋路）換成 0x39（可通行）。
func TestOpenBellDoor(t *testing.T) {
	m := &fakeMap{}
	if err := m.SetTileAt(BellDoor.X, BellDoor.Y, BellDoorClosedTile); err != nil {
		t.Fatal(err)
	}
	if !OpenBellDoor(m) {
		t.Error("關著的門應該回報開了")
	}
	got, _ := m.TileAt(BellDoor.X, BellDoor.Y)
	if got != BellDoorOpenTile {
		t.Errorf("門是 %#02x，預期 %#02x", got, BellDoorOpenTile)
	}
	// 再敲一次：原版無條件寫同一個值，這裡回報「本來就開著」。
	if OpenBellDoor(m) {
		t.Error("已經開了的門不該回報「開了」")
	}

	// 線性索引 0xed5 的換算釘住：index = y×64 + x，緩衝區第 0 格是 tile (0,0)。
	if BellDoor.Y*64+BellDoor.X != 0xed5 {
		t.Errorf("(%d,%d) → 索引 %#x，預期 0xed5",
			BellDoor.X, BellDoor.Y, BellDoor.Y*64+BellDoor.X)
	}
}
