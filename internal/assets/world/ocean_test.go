package world

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

// 逐指令對照原版的擲點器：state = state*125（低 16 位），最高位是 1 就回 true。
func TestOceanDither_MatchesOriginalLCG(t *testing.T) {
	d := NewOceanDither()

	state := uint16(oceanLCGSeed)
	for i := 0; i < 1000; i++ {
		state *= oceanLCGMultiplier
		want := state >= 0x8000
		if got := d.Next(); got != want {
			t.Fatalf("第 %d 次擲點 = %v，預期 %v（state=0x%04x）", i, got, want, state)
		}
	}
}

// 擲點結果要接近公平硬幣。
//
// 乘法同餘產生器很容易退化（種子是偶數就會一路把低位乘成 0），所以這條
// 不是形式主義：偏掉的話海面會整片變成同一種浪花，等於這個功能沒做。
func TestOceanDither_RoughlyFair(t *testing.T) {
	d := NewOceanDither()
	const n = 20000
	ones := 0
	for i := 0; i < n; i++ {
		if d.Next() {
			ones++
		}
	}
	if ones < n*45/100 || ones > n*55/100 {
		t.Errorf("20000 次擲出 %d 次 alt（%.1f%%），偏離公平硬幣太多", ones, 100*float64(ones)/n)
	}
}

// 只能動海面格，其他 tile 一律不准碰。
func TestOceanDither_OnlyTouchesOcean(t *testing.T) {
	tiles := make([]byte, 4096)
	for i := range tiles {
		tiles[i] = byte(i % 102)
	}
	before := append([]byte(nil), tiles...)

	NewOceanDither().Apply(tiles)

	for i := range tiles {
		if before[i] == OceanTile {
			if tiles[i] != OceanTile && tiles[i] != OceanTileAlt {
				t.Fatalf("第 %d 格從 0x%02x 變成 0x%02x", i, before[i], tiles[i])
			}
			continue
		}
		if tiles[i] != before[i] {
			t.Errorf("第 %d 格不是海面（0x%02x）卻被改成 0x%02x", i, before[i], tiles[i])
		}
	}
}

// 這個替換必須是純外觀的：兩個 tile 的可通行性要完全相同。
//
// 這條是 ocean.go「放在算繪端而不是寫進 Map」那個決定的前提。哪天原版
// 資料被換成可通行性不同的版本，這裡會紅，提醒要改成在載入時就地套用。
func TestOceanDither_PurelyCosmetic(t *testing.T) {
	dir := origDataDir(t)
	tb, err := gamedata.LoadTables(filepath.Join(dir, "FILES.DAT"))
	if err != nil {
		t.Fatalf("LoadTables: %v", err)
	}
	a := tb.Passability(OceanTile)
	b := tb.Passability(OceanTileAlt)
	if a != b {
		t.Errorf("0x%02x 可通行性 0x%02x，0x%02x 是 0x%02x —— 兩者不同就不是純外觀替換了",
			OceanTile, byte(a), OceanTileAlt, byte(b))
	}
	if !a.Blocked() {
		t.Errorf("海面 tile 0x%02x 應該不可步行", OceanTile)
	}
}

// 同一個種子要跑出同一張圖 —— 截圖比對靠這個。
func TestOceanDither_Reproducible(t *testing.T) {
	mk := func() []byte {
		tiles := make([]byte, 512)
		for i := range tiles {
			tiles[i] = OceanTile
		}
		NewOceanDitherSeed(1234).Apply(tiles)
		return tiles
	}
	a, b := mk(), mk()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("同種子跑出不同結果，第 %d 格 0x%02x vs 0x%02x", i, a[i], b[i])
		}
	}
}
