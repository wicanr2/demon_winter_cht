package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

func restParty(hp, sp int) []Character {
	return []Character{
		{Name: "甲", MaxHP: 20, CurrentHP: hp, MaxSP: 30, CurrentSP: sp},
		{Name: "乙", MaxHP: 20, CurrentHP: hp, MaxSP: 30, CurrentSP: sp},
	}
}

// 睡眠時數 =（27 − 時辰）+ rnd(6) − 1。
func TestRestDuration(t *testing.T) {
	for _, c := range []struct{ hour, roll, want int }{
		{15, 1, 12}, {15, 6, 17},
		{24, 1, 3}, {24, 6, 8},
	} {
		if got := RestDuration(c.hour, c.roll); got != c.want {
			t.Errorf("時辰 %d、擲 %d：睡 %d 個時辰，預期 %d",
				c.hour, c.roll, got, c.want)
		}
	}
}

// 旅店回復比紮營多一倍，而且不吃糧食。
func TestRest_InnVsCamp(t *testing.T) {
	food := 5

	inn := restParty(10, 10)
	res := Rest(&fixedRolls{vals: []int{3}}, RestInn, inn, NewClock(), &food)
	if inn[0].CurrentHP != 12 || inn[0].CurrentSP != 20 {
		t.Errorf("旅店後 HP %d SP %d，預期 12／20", inn[0].CurrentHP, inn[0].CurrentSP)
	}
	if food != 5 || res.AteFood {
		t.Errorf("旅店不該吃糧食，剩 %d 份", food)
	}

	camp := restParty(10, 10)
	Rest(&fixedRolls{vals: []int{3}}, RestCamp, camp, NewClock(), &food)
	if camp[0].CurrentHP != 11 || camp[0].CurrentSP != 15 {
		t.Errorf("紮營後 HP %d SP %d，預期 11／15", camp[0].CurrentHP, camp[0].CurrentSP)
	}
	if food != 4 {
		t.Errorf("紮營該吃掉一份糧食，剩 %d 份", food)
	}
}

// 回復量鉗到上限，不會超過。
func TestRest_ClampsToMax(t *testing.T) {
	p := restParty(19, 25)
	Rest(&fixedRolls{vals: []int{1}}, RestInn, p, NewClock(), nil)
	if p[0].CurrentHP != 20 || p[0].CurrentSP != 30 {
		t.Errorf("回復後 HP %d SP %d，應鉗在 20／30", p[0].CurrentHP, p[0].CurrentSP)
	}
}

// 沒糧食紮營，全隊扣 2 HP。
func TestRest_StarvingCostsHP(t *testing.T) {
	food := 0
	p := restParty(10, 10)
	res := Rest(&fixedRolls{vals: []int{1}}, RestCamp, p, NewClock(), &food)
	if !res.Starved {
		t.Fatal("沒糧食應該回報 Starved")
	}
	// −2 挨餓 +1 紮營回復 = 9
	if p[0].CurrentHP != 9 {
		t.Errorf("挨餓後 HP %d，預期 9", p[0].CurrentHP)
	}
}

// 中毒的人睡覺會依睡眠時數扣血；種族 4 免疫。
func TestRest_PoisonDrainsByHours(t *testing.T) {
	clock := NewClock()
	for clock.Hour() != 20 {
		clock.AdvanceHour()
	}
	hours := RestDuration(20, 2) // (27−20)+2−1 = 8

	p := restParty(20, 10)
	p[0].Status = scenario.StatusPoison
	p[1].Status = scenario.StatusPoison
	p[1].Race = restPoisonImmuneRace

	Rest(&fixedRolls{vals: []int{2}}, RestInn, p, clock, nil)
	if want := 20 - hours + 2; p[0].CurrentHP != want {
		t.Errorf("中毒者睡完 HP %d，預期 %d（扣 %d 個時辰）",
			p[0].CurrentHP, want, hours)
	}
	if p[1].CurrentHP != 20 {
		t.Errorf("種族 %d 應該免疫中毒扣血，卻掉到 %d",
			restPoisonImmuneRace, p[1].CurrentHP)
	}
}

// 血扣到 0 以下就死。
func TestRest_CanKill(t *testing.T) {
	clock := NewClock()
	for clock.Hour() != 15 {
		clock.AdvanceHour()
	}
	p := restParty(3, 0)
	p[0].Status = scenario.StatusPoison

	res := Rest(&fixedRolls{vals: []int{1}}, RestInn, p, clock, nil)
	if len(res.Died) != 1 || res.Died[0] != 0 {
		t.Errorf("中毒又只剩 3 血，睡一晚應該死，得到 %v", res.Died)
	}
}

// 睡醒一定在清晨 1–6 時，而且日期 +1。
func TestRest_WakesInTheMorning(t *testing.T) {
	r := rng.NewWithSeed(5)
	for i := 0; i < 100; i++ {
		clock := NewClock()
		for clock.Hour() != 20 {
			clock.AdvanceHour()
		}
		day := clock.Day()
		res := Rest(r, RestInn, restParty(10, 10), clock, nil)

		if clock.Hour() != res.WakeHour || res.WakeHour < 1 || res.WakeHour > 6 {
			t.Fatalf("睡醒在 %d 時，應落在 1–6", clock.Hour())
		}
		if clock.Day() != day+1 {
			t.Fatalf("睡完日期 %d，預期 %d", clock.Day(), day+1)
		}
	}
}

// 紮營會把道具的使用次數歸零 —— 過夜充能。
//
// 兩個例外照原版：上限 >= 100 的不充能、已用次數為 0xff 的不充能。
func TestRest_CampRechargesItems(t *testing.T) {
	p := restParty(10, 10)
	p[0].Inventory[0] = scenario.InventorySlot{Type: 16, Total: 3, Used: 3}
	p[0].Inventory[1] = scenario.InventorySlot{Type: 17, Total: 200, Used: 5}
	p[0].Inventory[2] = scenario.InventorySlot{Type: 18, Total: 3, Used: 0xff}
	p[0].Inventory[3] = scenario.InventorySlot{Type: 0xff}

	food := 3
	Rest(&fixedRolls{vals: []int{1}}, RestCamp, p, NewClock(), &food)

	if p[0].Inventory[0].Used != 0 {
		t.Errorf("一般道具該充能，Used = %d", p[0].Inventory[0].Used)
	}
	if p[0].Inventory[1].Used != 5 {
		t.Errorf("上限 >= 100 的不該充能，Used = %d", p[0].Inventory[1].Used)
	}
	if p[0].Inventory[2].Used != 0xff {
		t.Errorf("Used == 0xff 的不該充能，Used = %d", p[0].Inventory[2].Used)
	}
}

// 旅店不充能 —— 充能是紮營專屬的（2aed:0408 那一段在 param1==1 分支裡）。
func TestRest_InnDoesNotRecharge(t *testing.T) {
	p := restParty(10, 10)
	p[0].Inventory[0] = scenario.InventorySlot{Type: 16, Total: 3, Used: 3}
	Rest(&fixedRolls{vals: []int{1}}, RestInn, p, NewClock(), nil)
	if p[0].Inventory[0].Used != 3 {
		t.Errorf("旅店不該充能，Used = %d", p[0].Inventory[0].Used)
	}
}
