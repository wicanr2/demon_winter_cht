package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

func testTown() gamedata.Town {
	return gamedata.Town{Number: 8, Name: "New Gleon", Economy: 13, ShipBase: 61}
}

// 進城後的價格體系直接由城鎮的 E 推出來。
func TestEnterTown_EconomyFromTownData(t *testing.T) {
	v := EnterTown(testTown(), nil)

	if v.Economy.E != 13 {
		t.Errorf("經濟係數 = %d，預期 13", v.Economy.E)
	}
	if got, want := v.Economy.ShipPrice(), 610; got != want {
		t.Errorf("船價 = %d，預期 %d", got, want)
	}
	if !v.HasDocks() {
		t.Error("New Gleon 應該有碼頭")
	}
}

func TestEnterTown_InlandHasNoDocks(t *testing.T) {
	v := EnterTown(gamedata.Town{Number: 1, Name: "Seaside", Economy: 10}, nil)
	if v.HasDocks() {
		t.Error("船價基礎值為 0 的城鎮不該有碼頭")
	}
}

// 議價初值取決於隊伍有沒有人會說服，所以必須在進城當下決定。
func TestEnterTown_HaggleInitialState(t *testing.T) {
	var plain Character
	if got := EnterTown(testTown(), []Character{plain}).HaggleState(0); got != 1 {
		t.Errorf("沒人會說服時議價初值 = %d，預期 1", got)
	}

	var skilled Character
	skilled.Skills[gamedata.SkillPersuasion] = true
	if got := EnterTown(testTown(), []Character{skilled}).HaggleState(0); got != 0 {
		t.Errorf("有人會說服時議價初值 = %d，預期 0", got)
	}
}

// 售價 = 標價套 E 之後再套議價折扣，兩層都要生效。
func TestTownVisit_Price(t *testing.T) {
	v := EnterTown(testTown(), nil) // E = 13，議價初值 1

	const base = 100
	listed := v.Economy.BuyPrice(base) // 100 × 13 / 10 = 130
	if listed != 130 {
		t.Fatalf("標價 = %d，預期 130", listed)
	}
	// 議價狀態 1 → 打掉標價的 6%
	if got, want := v.Price(0, base), HagglePrice(listed, 1); got != want {
		t.Errorf("售價 = %d，預期 %d", got, want)
	}

	// 議價成功後價格要真的往下走。
	before := v.Price(0, base)
	v.SetHaggleState(0, 5)
	if after := v.Price(0, base); after >= before {
		t.Errorf("議價五次後 %d 應低於 %d", after, before)
	}
}

// 議價狀態是每件商品各自獨立的。
func TestTownVisit_HaggleIsPerItem(t *testing.T) {
	v := EnterTown(testTown(), nil)
	v.SetHaggleState(3, 7)

	if got := v.HaggleState(3); got != 7 {
		t.Errorf("商品 3 的議價狀態 = %d，預期 7", got)
	}
	if got := v.HaggleState(4); got != 1 {
		t.Errorf("商品 4 不該被商品 3 影響，得到 %d", got)
	}
}

// 超出範圍的商品索引不能 panic。
func TestTownVisit_HaggleOutOfRange(t *testing.T) {
	v := EnterTown(testTown(), nil)
	for _, i := range []int{-1, 999} {
		if got := v.HaggleState(i); got != 0 {
			t.Errorf("索引 %d 應回傳 0，得到 %d", i, got)
		}
		v.SetHaggleState(i, 5) // 不能 panic
	}
}

func TestFacilityName(t *testing.T) {
	if n := len(AllFacilities); n != 7 {
		t.Errorf("設施有 %d 種，原版 DEMON.INT 是 7 個字串", n)
	}
	seen := map[string]bool{}
	for _, f := range AllFacilities {
		name := FacilityName(f)
		if name == "?" {
			t.Errorf("設施 %d 沒有中文名", f)
		}
		if seen[name] {
			t.Errorf("設施名 %q 重複", name)
		}
		seen[name] = true
	}
}
