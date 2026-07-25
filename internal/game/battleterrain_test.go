package game

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

func loadWorldMap(t *testing.T) *world.Map {
	t.Helper()
	m, err := world.LoadMap(filepath.Join(repoRoot(t), "workplace", "orig", "demwin", "DEM_DATA", "MAP1.MAP"))
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}
	return m
}

func loadSightTable(t *testing.T) *gamedata.SightShadow {
	t.Helper()
	tb, err := gamedata.LoadTables(filepath.Join(repoRoot(t), "workplace", "orig", "demwin", "DEM_DATA", "FILES.DAT"))
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	return tb.Sight()
}

// inset=0 時，整個 9×9 都要與世界地圖上以中心點為準的那一塊逐格相同。
func TestBattleTerrain_MirrorsWorldWindow(t *testing.T) {
	m := loadWorldMap(t)
	const cx, cy = 32, 32

	bt, err := NewBattleTerrain(m, cx, cy, 0)
	if err != nil {
		t.Fatal(err)
	}
	half := BattleTerrainSize / 2
	for gy := 0; gy < BattleTerrainSize; gy++ {
		for gx := 0; gx < BattleTerrainSize; gx++ {
			want, err := m.TileAt(cx-half+gx, cy-half+gy)
			if err != nil {
				t.Fatal(err)
			}
			if got := bt.TileAt(gx, gy); got != want {
				t.Errorf("(%d,%d) 戰場地形 0x%02x，世界地圖是 0x%02x", gx, gy, got, want)
			}
		}
	}
}

// 中心那一格永遠是遭遇發生的那一格 —— 內縮多少都一樣。
//
// 這條顧的是「視窗有沒有對準」：偏一格的話，玩家會在錯誤的地形上開打，
// 而且從畫面上完全看不出來。
func TestBattleTerrain_CentreIsEncounterTile(t *testing.T) {
	m := loadWorldMap(t)
	half := BattleTerrainSize / 2

	for _, c := range [][2]int{{32, 32}, {10, 50}, {1, 3}, {63, 63}} {
		want, err := m.TileAt(c[0], c[1])
		if err != nil {
			t.Fatal(err)
		}
		for inset := 0; inset <= gamedata.MaxLightInset; inset++ {
			bt, err := NewBattleTerrain(m, c[0], c[1], inset)
			if err != nil {
				t.Fatal(err)
			}
			if got := bt.TileAt(half, half); got != want {
				t.Errorf("中心 (%d,%d) inset=%d：戰場 0x%02x，世界地圖 0x%02x",
					c[0], c[1], inset, got, want)
			}
		}
	}
}

// 內縮之後，外圈那一環一定是空白，裡面那一塊不受影響。
func TestBattleTerrain_InsetBlanksBorder(t *testing.T) {
	m := loadWorldMap(t)
	const cx, cy = 32, 32

	full, err := NewBattleTerrain(m, cx, cy, 0)
	if err != nil {
		t.Fatal(err)
	}
	for inset := 1; inset <= gamedata.MaxLightInset; inset++ {
		bt, err := NewBattleTerrain(m, cx, cy, inset)
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < BattleTerrainSize; y++ {
			for x := 0; x < BattleTerrainSize; x++ {
				inside := x >= inset && x < BattleTerrainSize-inset &&
					y >= inset && y < BattleTerrainSize-inset
				got := bt.TileAt(x, y)
				if inside {
					if got != full.TileAt(x, y) {
						t.Errorf("inset=%d (%d,%d) 在視窗內卻與滿版不同", inset, x, y)
					}
					continue
				}
				if got != 0 {
					t.Errorf("inset=%d (%d,%d) 在視窗外，應為空白卻是 0x%02x", inset, x, y, got)
				}
			}
		}
	}
}

// 貼在地圖角落時，越界的格子維持空白而不是繞回另一邊。
//
// 64×64 的地圖用 y*64+x 定址，少一個邊界檢查就會 wrap —— 玩家會在
// 地圖左上角遭遇時看到右下角的地形，而且不會當掉。
func TestBattleTerrain_ClipsAtMapEdge(t *testing.T) {
	m := loadWorldMap(t)

	bt, err := NewBattleTerrain(m, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	half := BattleTerrainSize / 2
	for y := 0; y < half; y++ {
		for x := 0; x < BattleTerrainSize; x++ {
			if got := bt.TileAt(x, y); got != 0 {
				t.Errorf("(%d,%d) 對應世界座標 y<0，應為空白卻是 0x%02x", x, y, got)
			}
		}
	}
}

func TestBattleTerrain_RejectsBadInset(t *testing.T) {
	m := loadWorldMap(t)
	for _, k := range []int{-1, gamedata.MaxLightInset + 1, 99} {
		if _, err := NewBattleTerrain(m, 32, 32, k); err == nil {
			t.Errorf("內縮量 %d 應回傳錯誤", k)
		}
	}
}

// 日夜曲線：正午看得最遠、深夜看得最近，而且整條曲線都在合法範圍內。
func TestLightInset_DayNightCurve(t *testing.T) {
	noon := gamedata.LightInsetAt(10)
	if noon != 0 {
		t.Errorf("白天（10 時）內縮量 = %d，預期 0（整個戰場都看得到）", noon)
	}
	night := gamedata.LightInsetAt(22)
	if night <= noon {
		t.Errorf("深夜（22 時）內縮量 %d 應該大於白天的 %d", night, noon)
	}
	for h := 0; h < gamedata.HoursPerDay; h++ {
		if k := gamedata.LightInsetAt(h); k < 0 || k > gamedata.MaxLightInset {
			t.Errorf("%d 時的內縮量 %d 超出 0–%d", h, k, gamedata.MaxLightInset)
		}
	}
}

// 超出範圍的時辰要取模，不能越界。
func TestLightInset_WrapsHour(t *testing.T) {
	for _, h := range []int{-1, gamedata.HoursPerDay, gamedata.HoursPerDay + 10, 1000} {
		k := gamedata.LightInsetAt(h)
		if k < 0 || k > gamedata.MaxLightInset {
			t.Errorf("時辰 %d 得到內縮量 %d，超出合法範圍", h, k)
		}
	}
	if a, b := gamedata.LightInsetAt(3), gamedata.LightInsetAt(3+gamedata.HoursPerDay); a != b {
		t.Errorf("時辰 3 與 3+%d 應該一樣，得到 %d vs %d", gamedata.HoursPerDay, a, b)
	}
}

// 視線遮蔽接上戰場地形：被遮住的格子在 Visible() 之後要變成空白，
// 其餘一格都不能動。
func TestBattleTerrain_VisibleBlanksShadowedCells(t *testing.T) {
	s := loadSightTable(t)

	var bt BattleTerrain
	for i := range bt {
		bt[i] = 0x01 // 不擋視線的地形
	}
	const bx, by = 2, 2
	bt[by*BattleTerrainSize+bx] = gamedata.SightBlockerTiles[0]

	vis, err := bt.Visible(s)
	if err != nil {
		t.Fatal(err)
	}
	shadow := map[int]bool{}
	for _, c := range s.ShadowAt(bx, by) {
		shadow[c] = true
	}
	if len(shadow) == 0 {
		t.Fatal("(2,2) 的陰影是空的，測試前提不成立")
	}
	for i := range bt {
		want := bt[i]
		if shadow[i] {
			want = 0
		}
		if vis[i] != want {
			t.Errorf("第 %d 格 = 0x%02x，預期 0x%02x（遮蔽 %v）", i, vis[i], want, shadow[i])
		}
	}
}

// 用真實地城資料跑一次視線遮蔽，確認它真的有作用。
//
// 前面那些測試都是自己造的地形，遮蔽表就算被解成「每組都是空的」也照樣
// 會綠 —— 那種壞法最像「功能沒接上」。這條拿 MAP1 的一段走廊來跑：
// 中心看得到走廊、看不穿牆，被塗掉的格子應該佔可見地形的一大半。
func TestBattleTerrain_ShadowHasEffectOnRealMap(t *testing.T) {
	m := loadWorldMap(t)
	s := loadSightTable(t)

	bt, err := NewBattleTerrain(m, 20, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	vis, err := bt.Visible(s)
	if err != nil {
		t.Fatal(err)
	}

	withTerrain, blanked := 0, 0
	for i := range bt {
		if bt[i] == 0 {
			continue
		}
		withTerrain++
		if vis[i] == 0 {
			blanked++
		}
	}
	if withTerrain == 0 {
		t.Fatal("MAP1 (20,20) 附近沒有任何地形，測試前提不成立")
	}
	if blanked == 0 {
		t.Errorf("%d 格地形一格都沒被遮蔽 —— 遮蔽表可能解成空的了", withTerrain)
	}
	// 中心那一格永遠看得到：它就是視點。
	half := BattleTerrainSize / 2
	if bt.TileAt(half, half) != 0 && vis.TileAt(half, half) == 0 {
		t.Error("中心格被自己遮住了")
	}
	t.Logf("MAP1 (20,20)：%d 格有地形，其中 %d 格被視線擋住", withTerrain, blanked)
}
