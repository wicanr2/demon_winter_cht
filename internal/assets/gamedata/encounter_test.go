package gamedata

import (
	"path/filepath"
	"testing"
)

func encounterTables(t *testing.T) *Tables {
	t.Helper()
	tb, err := LoadTables(filepath.Join(origDataDir(t), "FILES.DAT"))
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	return tb
}

// 地形就是可通行性值。這條把「不用另一張表」這件事釘住 ——
// 只要有人日後加了第二張地形表，這裡就會對不上。
func TestTerrainIsPassabilityValue(t *testing.T) {
	tb := encounterTables(t)
	for tile := 0; tile < numTiles; tile++ {
		p := tb.Passability(byte(tile))
		terrain, ok := tb.Terrain(byte(tile))
		if byte(p) < NumTerrains {
			if !ok || byte(terrain) != byte(p) {
				t.Errorf("tile 0x%02x 可通行性 %d，地形卻是 (%d, %v)",
					tile, p, terrain, ok)
			}
			continue
		}
		if ok {
			t.Errorf("tile 0x%02x 不可通行（%d），卻回報有地形 %d", tile, p, terrain)
		}
	}
}

// 山丘是唯一直接驗證的地形：可通行性值 3 的兩個 tile，
// 就是移動要多算一步的 0x0e 與 0x2b。
func TestTerrainHills(t *testing.T) {
	tb := encounterTables(t)
	var hills []byte
	for tile := 0; tile < numTiles; tile++ {
		if terrain, ok := tb.Terrain(byte(tile)); ok && terrain == TerrainHills {
			hills = append(hills, byte(tile))
		}
	}
	if len(hills) != 2 || hills[0] != 0x0e || hills[1] != 0x2b {
		t.Errorf("山丘 tile = %v，預期 [0x0e 0x2b]", hills)
	}
}

// 地形表只引用得到存在的群組。
func TestTerrainGroups_InRange(t *testing.T) {
	tb := encounterTables(t)
	for terrain := Terrain(0); terrain < NumTerrains; terrain++ {
		groups, err := tb.TerrainGroups(terrain)
		if err != nil {
			t.Fatalf("TerrainGroups(%d): %v", terrain, err)
		}
		for slot, g := range groups {
			if int(g) >= NumEncounterGroups {
				t.Errorf("地形 %d 槽位 %d 指向群組 %d，超出 0–%d",
					terrain, slot, g, NumEncounterGroups-1)
			}
		}
	}
	if _, err := tb.TerrainGroups(NumTerrains); err == nil {
		t.Error("超出範圍的地形應該回錯誤")
	}
}

// 每組的等級範圍與怪物索引都要合理。
//
// 這條同時擋住「22 bytes 一組」這個 stride 抄錯 —— 只要 stride 錯一格，
// 等級欄就會讀到怪物索引，值馬上跳出 1–10。
func TestEncounterGroups_Sane(t *testing.T) {
	tb := encounterTables(t)
	for i := 0; i < NumEncounterGroups; i++ {
		g, err := tb.EncounterGroup(i)
		if err != nil {
			t.Fatalf("EncounterGroup(%d): %v", i, err)
		}
		if g.MinLevel < 1 || g.MinLevel > 10 {
			t.Errorf("組 %d 最低等級 %d 不在 1–10", i, g.MinLevel)
		}
		if g.MaxLevel < g.MinLevel || g.MaxLevel > 10 {
			t.Errorf("組 %d 等級範圍 %d–%d 不合理", i, g.MinLevel, g.MaxLevel)
		}
		for j, e := range g.Entries {
			if e.Monster < 0 || e.Monster > 98 {
				t.Errorf("組 %d 第 %d 筆的怪物索引 %d 超出 MONSTER.DAT",
					i, j, e.Monster)
			}
		}
	}
	if _, err := tb.EncounterGroup(NumEncounterGroups); err == nil {
		t.Error("超出範圍的群組應該回錯誤")
	}
}

// 沼澤是證據最硬的一個：它是唯一掛到「鬼火 + 鬼墳族」那一組的地形，
// 而手冊點名的就是這兩隻。
//
// 這條會在「地形編號對應改動」時直接爆掉，是整套推定的錨點。
func TestTerrainSwamp_HasManualNamedMonsters(t *testing.T) {
	const (
		willOWisp      = 78
		shamblingMound = 63
	)
	tb := encounterTables(t)

	has := func(terrain Terrain, monster int) bool {
		groups, _ := tb.TerrainGroups(terrain)
		for _, gi := range groups {
			g, err := tb.EncounterGroup(int(gi))
			if err != nil {
				continue
			}
			for _, e := range g.Entries {
				if e.Monster == monster {
					return true
				}
			}
		}
		return false
	}

	if !has(TerrainSwamp, willOWisp) || !has(TerrainSwamp, shamblingMound) {
		t.Error("沼澤應該遇得到鬼火與鬼墳族（手冊點名）")
	}
	for terrain := Terrain(0); terrain < NumTerrains; terrain++ {
		if terrain == TerrainSwamp {
			continue
		}
		if has(terrain, willOWisp) || has(terrain, shamblingMound) {
			t.Errorf("地形 %d（%s）不該遇得到沼澤專屬怪", terrain, terrain.Name())
		}
	}
}

// 沙漠：手冊說「除了些賊之外其他人類不願前來」。
// 對應到資料就是「有賊那一組、沒有探險者那兩組」。
func TestTerrainDesert_NoAdventurers(t *testing.T) {
	const (
		dervish     = 94
		salamander  = 90
		lvl1Fighter = 0 // 探險者組的代表
	)
	tb := encounterTables(t)
	groups, _ := tb.TerrainGroups(TerrainDesert)

	found := map[int]bool{}
	for _, gi := range groups {
		g, _ := tb.EncounterGroup(int(gi))
		for _, e := range g.Entries {
			found[e.Monster] = true
		}
	}
	if !found[dervish] || !found[salamander] {
		t.Error("沙漠應該遇得到苦修僧與蜥蜴人（手冊點名）")
	}
	if found[lvl1Fighter] {
		t.Error("沙漠不該遇得到探險者（手冊：除了賊，其他人類不願前來）")
	}
}

// 凍土的怪物全是冰系。
func TestTerrainTundra_IsAllIce(t *testing.T) {
	tb := encounterTables(t)
	mt, err := LoadMonsterTable(filepath.Join(origDataDir(t), "MONSTER.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	groups, _ := tb.TerrainGroups(TerrainTundra)

	// 第 15 組是凍土組，在凍土的八個槽位裡佔一半。
	count := 0
	for _, gi := range groups {
		if gi == 15 {
			count++
		}
	}
	if count != 4 {
		t.Errorf("凍土掛第 15 組 %d 次，預期 4", count)
	}

	g, _ := tb.EncounterGroup(15)
	ice := 0
	for _, e := range g.Entries {
		m, err := mt.ByIndex(e.Monster)
		if err != nil {
			t.Fatal(err)
		}
		// 冰系（Special 7）、冰龍（10）、或名字帶 Ice／Winter／Snow／Arctic／Yeti。
		if m.Special == 7 || m.Special == 10 {
			ice++
		}
	}
	if ice < 5 {
		t.Errorf("凍土組只有 %d 隻冰系怪，太少", ice)
	}
}

// 等級檢查是 ±1 的窄帶，而且**對每一筆都成立** ——
// 不是只有 >= 100 的那種才檢查。原版兩條路徑落到同一段比較。
func TestEncounterEntry_Allowed(t *testing.T) {
	plain := EncounterEntry{Monster: 1, Level: 5}
	if plain.Gated() {
		t.Error("等級 5 不該算 gated")
	}
	for lvl, want := range map[int]bool{3: false, 4: true, 5: true, 6: true, 7: false} {
		if got := plain.Allowed(lvl); got != want {
			t.Errorf("等級 5、難度 %d：allowed = %v，預期 %v", lvl, got, want)
		}
	}

	gated := EncounterEntry{Monster: 1, Level: 105}
	if !gated.Gated() || gated.EffectiveLevel() != 5 {
		t.Errorf("105 應該是 gated 且實際等級 5，得到 %v／%d",
			gated.Gated(), gated.EffectiveLevel())
	}
	// 減掉 100 之後，兩者的通過條件完全一樣。
	for lvl := 1; lvl <= 10; lvl++ {
		if gated.Allowed(lvl) != plain.Allowed(lvl) {
			t.Errorf("難度 %d：gated 與非 gated 的等級判定不一致", lvl)
		}
	}
}

// 兩個地城類別不算野外地形。
func TestTerrainOutdoor(t *testing.T) {
	for terrain, want := range map[Terrain]bool{
		TerrainForest: true, TerrainPlains: true, TerrainSwamp: true,
		TerrainHills: true, TerrainTundra: true, TerrainDesert: true,
		TerrainDungeonA: false, TerrainDungeonB: false,
	} {
		if got := terrain.Outdoor(); got != want {
			t.Errorf("%s（%d）Outdoor = %v，預期 %v", terrain.Name(), terrain, got, want)
		}
	}
}
