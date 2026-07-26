package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/rng"
)

// flatTerrain 造一塊可站範圍全部同值的戰場地形（外圍留 0，等於不可站）。
func flatTerrain(tile byte) *BattleTerrain {
	var t BattleTerrain
	for y := BattleFieldMin; y <= BattleFieldMax; y++ {
		for x := BattleFieldMin; x <= BattleFieldMax; x++ {
			t[y*BattleTerrainSize+x] = tile
		}
	}
	return &t
}

func TestDeployPartyAt_FollowsFormation(t *testing.T) {
	f := Formation{0, 0xff, 1, 0xff, 2, 0xff, 3, 0xff, 4}
	want := map[int][2]int{
		0: {BattleCentreX - 1, BattleCentreY - 1}, // A
		1: {BattleCentreX + 1, BattleCentreY - 1}, // C
		2: {BattleCentreX, BattleCentreY},         // E
		3: {BattleCentreX - 1, BattleCentreY + 1}, // G
		4: {BattleCentreX + 1, BattleCentreY + 1}, // I
	}
	for member, w := range want {
		x, y, ok := DeployPartyAt(f, member)
		if !ok {
			t.Fatalf("成員 %d 應該有位置", member)
		}
		if x != w[0] || y != w[1] {
			t.Errorf("成員 %d 站 (%d,%d)，預期 (%d,%d)", member, x, y, w[0], w[1])
		}
	}
	if _, _, ok := DeployPartyAt(f, 7); ok {
		t.Error("不在陣型裡的成員不該上場")
	}
}

// 全員擠在前五格時，佈陣就是前兩列 —— 與 PARTY.BAK 的排法對應。
func TestDeployPartyAt_CompactFormation(t *testing.T) {
	f := Formation{0, 1, 2, 3, 4, 0xff, 0xff, 0xff, 0xff}
	seen := map[[2]int]bool{}
	for m := 0; m < 5; m++ {
		x, y, ok := DeployPartyAt(f, m)
		if !ok {
			t.Fatalf("成員 %d 應該有位置", m)
		}
		if seen[[2]int{x, y}] {
			t.Errorf("成員 %d 與別人站同一格 (%d,%d)", m, x, y)
		}
		seen[[2]int{x, y}] = true
	}
	if len(seen) != 5 {
		t.Errorf("五個人只佔了 %d 格", len(seen))
	}
}

func TestScatterMonster_StaysWithinTwoOfCentre(t *testing.T) {
	terrain := flatTerrain(7)
	r := rng.NewWithSeed(1)
	for i := 0; i < 500; i++ {
		x, y, ok := ScatterMonster(r, terrain, nil)
		if !ok {
			t.Fatal("空曠戰場應該一定擲得出位置")
		}
		if absInt(x-BattleCentreX) > 2 || absInt(y-BattleCentreY) > 2 {
			t.Fatalf("落點 (%d,%d) 超出中心 ±2", x, y)
		}
	}
}

// 兩個獨立的證據：怪物不會疊在一起，也不會站到不是空地的格子上。
func TestScatterMonster_AvoidsOccupiedAndNonGround(t *testing.T) {
	terrain := flatTerrain(7)
	// 把中心那一列以外全部改成別的地形，只留中心那一列是空地。
	for y := 0; y < BattleGridHeight; y++ {
		for x := 0; x < BattleGridWidth; x++ {
			if y != BattleCentreY {
				terrain[y*BattleTerrainSize+x] = 99
			}
		}
	}
	taken := map[[2]int]bool{{BattleCentreX, BattleCentreY}: true}
	occupied := func(x, y int) bool { return taken[[2]int{x, y}] }

	r := rng.NewWithSeed(3)
	for i := 0; i < 4; i++ {
		x, y, ok := ScatterMonster(r, terrain, occupied)
		if !ok {
			t.Fatalf("第 %d 隻應該還有位置", i)
		}
		if y != BattleCentreY {
			t.Errorf("落點 (%d,%d) 不在唯一那一列空地上", x, y)
		}
		taken[[2]int{x, y}] = true
	}
	// 中心那一列 ±2 共五格，一格被隊伍佔著、四格給了怪物 → 沒位置了。
	if _, _, ok := ScatterMonster(r, terrain, occupied); ok {
		t.Error("位置用光了還擲得出來")
	}
}

// 牆與界外都不可站。
func TestOpenGround_RejectsOutsideTheField(t *testing.T) {
	terrain := flatTerrain(7)
	for _, p := range [][2]int{
		{0, 0}, {BattleWallLow, BattleCentreY}, {BattleWallHigh, BattleCentreY},
		{BattleCentreX, BattleFieldMax + 1},
	} {
		if OpenGround(terrain, nil, p[0], p[1]) {
			t.Errorf("(%d,%d) 在可站範圍外，不該放得下", p[0], p[1])
		}
	}
}
