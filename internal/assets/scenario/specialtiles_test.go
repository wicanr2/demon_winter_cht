package scenario

import (
	"bytes"
	"os"
	"testing"
)

func loadSpecial(t *testing.T, mapID int) (*SpecialTiles, []byte) {
	t.Helper()
	path := dataPath(SpecialTileFileName(mapID))
	skipIfMissing(t, path)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	st, err := ParseSpecialTiles(raw)
	if err != nil {
		t.Fatalf("解 %s 失敗：%v", path, err)
	}
	return st, raw
}

// TestSpecialTiles_TeleportCountMatchesDests 釘住 `docs/re/77` §4 那條
// **會被資料打死的預測**：類別 4 的筆數必須等於檔尾非零座標對的個數
//（因為第 k 筆用第 k 對）。五個檔案全對就不是巧合。
//
// 這條測試值錢的地方在於它同時擋住兩種錯：前段的表尾判斷、
// 以及檔尾反向表的偏移公式。任一邊算錯，數字就對不上。
func TestSpecialTiles_TeleportCountMatchesDests(t *testing.T) {
	want := map[int]int{1: 2, 2: 0, 3: 7, 4: 3, 5: 2}
	for mapID := 1; mapID <= SpecialTileMapCount; mapID++ {
		st, _ := loadSpecial(t, mapID)
		n := 0
		for _, tile := range st.Tiles {
			if tile.Class() == SpecialClassTeleport {
				n++
			}
		}
		if n != len(st.Dests) {
			t.Errorf("%dSS.DAT：類別 4 有 %d 筆，檔尾座標對 %d 組，必須相等",
				mapID, n, len(st.Dests))
		}
		if n != want[mapID] {
			t.Errorf("%dSS.DAT：類別 4 筆數 = %d，預期 %d（docs/re/77 §4）",
				mapID, n, want[mapID])
		}
	}
}

// TestSpecialTiles_RoundTrip 釘住「解出來再寫回去必須逐位元組相同」。
//
// 這是存檔正確性的前提：原版自己會讀這個檔，佈局差一個 byte 就整份錯位。
// 兩張表往中間長，所以 round-trip 同時驗前段長度與檔尾偏移。
func TestSpecialTiles_RoundTrip(t *testing.T) {
	for mapID := 1; mapID <= SpecialTileMapCount; mapID++ {
		st, raw := loadSpecial(t, mapID)
		got := st.Encode()
		if !bytes.Equal(got, raw) {
			for i := range raw {
				if got[i] != raw[i] {
					t.Errorf("%dSS.DAT：偏移 %d 不同（原 0x%02x，寫回 0x%02x）",
						mapID, i, raw[i], got[i])
					break
				}
			}
		}
	}
}

// TestSpecialTiles_LookupMutualTeleport 用 `5SS.DAT` 那組互為往返的傳送
//（`docs/re/77` §4）驗查表與配對：(28,59) → (11,5)、(11,4) → (28,58)。
//
// 目的地都差一格，這樣踩上去不會立刻被彈回來 —— 原版的設計，不是本專案的。
func TestSpecialTiles_LookupMutualTeleport(t *testing.T) {
	st, _ := loadSpecial(t, 5)
	cases := []struct{ x, y, dx, dy byte }{
		{28, 59, 11, 5},
		{11, 4, 28, 58},
	}
	for _, c := range cases {
		hit := st.Lookup(c.x, c.y)
		if hit == nil {
			t.Fatalf("(%d,%d) 應該命中", c.x, c.y)
		}
		if !hit.Teleport {
			t.Fatalf("(%d,%d) 應該是傳送，類別是 %d", c.x, c.y, hit.Tile.Class())
		}
		if hit.Dest.X != c.dx || hit.Dest.Y != c.dy {
			t.Errorf("(%d,%d) → (%d,%d)，預期 (%d,%d)",
				c.x, c.y, hit.Dest.X, hit.Dest.Y, c.dx, c.dy)
		}
	}
	if st.Lookup(0, 0) != nil {
		t.Error("(0,0) 不該命中 —— X == 0 是表尾標記")
	}
}

// TestSpecialTiles_EventIndexCountsOnlyEventClasses 釘住「兩個計數器互不相干」。
//
// 原版掃描時類別 1／2 走一個計數器、類別 4 走另一個（`0x172df`／`0x172e7`）。
// 共用一個的話，一張同時有事件與傳送的地圖會兩邊都算錯 ——
// 而 `3SS.DAT` 正好同時有 26 筆事件類與 7 筆傳送。
func TestSpecialTiles_EventIndexCountsOnlyEventClasses(t *testing.T) {
	st, _ := loadSpecial(t, 3)
	seq := 0
	for i, tile := range st.Tiles {
		cls := tile.Class()
		if cls != SpecialClassEventA && cls != SpecialClassEventB {
			continue
		}
		seq++
		hit := st.Lookup(tile.X, tile.Y)
		if hit == nil {
			t.Fatalf("記錄 %d (%d,%d) 查不到", i, tile.X, tile.Y)
		}
		// 座標可能重複，命中的是第一筆；只在確實是同一筆時比對序號。
		if hit.Index != i {
			continue
		}
		if hit.EventIndex != seq {
			t.Errorf("記錄 %d (%d,%d)：序號 = %d，預期 %d",
				i, tile.X, tile.Y, hit.EventIndex, seq)
		}
	}
	if seq == 0 {
		t.Fatal("3SS.DAT 應該有事件類記錄")
	}
}

// TestSpecialTiles_Mutations 釘住三種改寫，尤其 Advance 的「只推進一次」。
//
// 守衛 `attr < 0xc1` 的數字不是隨手挑的：原版出廠的 `1SS.DAT` 有一筆
// `0x66 → 0xc6`，而 `0xc6 >= 0xc1`，所以第二次來就不會再加（`docs/re/78` §2）。
func TestSpecialTiles_Mutations(t *testing.T) {
	st := &SpecialTiles{Tiles: []SpecialTile{
		{X: 5, Y: 6, Attr: 0x66}, // 類別 3
		{X: 7, Y: 8, Attr: 0x40}, // 類別 2，值 0
		{X: 9, Y: 10, Attr: 0x61},
	}}

	if !st.Advance(0) {
		t.Fatal("0x66 應該可以推進")
	}
	if got := st.Tiles[0].Attr; got != 0xc6 {
		t.Errorf("推進後 attr = 0x%02x，預期 0xc6（類別 3 → 6）", got)
	}
	if st.Advance(0) {
		t.Error("0xc6 >= 0xc1，第二次不該再推進 —— 這就是那個守衛的意義")
	}

	st.MarkVisited(1)
	if got := st.Tiles[1].Attr; got != 0x41 {
		t.Errorf("標記已造訪後 attr = 0x%02x，預期 0x41（類別留著、值變 1）", got)
	}
	if st.Tiles[1].Class() != SpecialClassEventB {
		t.Error("標記已造訪不該改掉類別 —— 記錄還要繼續命中")
	}

	st.Consume(2)
	if got := st.Tiles[2].Attr; got != 0 {
		t.Errorf("清零後 attr = 0x%02x，預期 0", got)
	}
	if st.Lookup(9, 10) == nil {
		t.Error("清零只讓它不再分派，記錄本身還在（座標仍查得到）")
	}
	if cls := st.Tiles[2].Class(); cls != 0 {
		t.Errorf("清零後類別應為 0，得到 %d", cls)
	}

	// 界外索引不該 panic —— 改寫端的 index 來自查表結果，但呼叫方可能傳舊值。
	st.Consume(-1)
	st.Consume(99)
	st.MarkVisited(99)
	if st.Advance(99) {
		t.Error("界外索引不該回報有改動")
	}
}

// TestSplitAllSS 釘住新遊戲的重建，並且用原版資料驗證
// 「母本 vs 工作副本」這個判讀（`docs/re/78` §2）：
//
//   - 3／4／5 號沒被玩過 → 切出來的區塊與磁碟上的 nSS.DAT 逐位元組相同
//   - 1／2 號被玩過 → 只在**屬性欄**不同（每 3 bytes 的第 3 欄）
//
// 後者是關鍵：如果差異出現在座標欄，那就不是「玩過的痕跡」而是切法錯了。
func TestSplitAllSS(t *testing.T) {
	path := dataPath("ALL_SS.DAT")
	skipIfMissing(t, path)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := SplitAllSS(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != SpecialTileMapCount {
		t.Fatalf("切出 %d 塊，預期 %d", len(blocks), SpecialTileMapCount)
	}

	clean := map[int]bool{3: true, 4: true, 5: true}
	for mapID := 1; mapID <= SpecialTileMapCount; mapID++ {
		_, disk := loadSpecial(t, mapID)
		block := blocks[mapID-1]
		if len(block) != SpecialTileFileSize {
			t.Errorf("區塊 %d 長度 %d，預期 %d", mapID-1, len(block), SpecialTileFileSize)
			continue
		}
		if clean[mapID] {
			if !bytes.Equal(block, disk) {
				t.Errorf("%dSS.DAT 應與 ALL_SS 區塊完全相同（沒被玩過）", mapID)
			}
			continue
		}
		diffs := 0
		for i := range block {
			if block[i] == disk[i] {
				continue
			}
			diffs++
			if col := i % specialRecordLen; col != specialOffAttr {
				t.Errorf("%dSS.DAT 偏移 %d（第 %d 欄）不同 —— 差異只該出現在屬性欄，"+
					"出現在座標欄表示切法錯了", mapID, i, col+1)
			}
		}
		if diffs == 0 {
			t.Errorf("%dSS.DAT 預期與 ALL_SS 有差異（出廠是玩過的狀態）", mapID)
		}
	}

	if _, err := SplitAllSS(raw[:100]); err == nil {
		t.Error("長度不符應回傳錯誤")
	}
}

// TestParseSpecialTiles_RejectsWrongSize 釘住長度檢查。
// 511 不是 512 —— 原版載入與存回都用 0x1ff，多寫一個 byte 就蓋掉別的東西。
func TestParseSpecialTiles_RejectsWrongSize(t *testing.T) {
	for _, n := range []int{0, 510, 512} {
		if _, err := ParseSpecialTiles(make([]byte, n)); err == nil {
			t.Errorf("長度 %d 應被拒絕", n)
		}
	}
	st, err := ParseSpecialTiles(make([]byte, SpecialTileFileSize))
	if err != nil {
		t.Fatalf("全零的檔案是合法的空清單：%v", err)
	}
	if len(st.Tiles) != 0 || len(st.Dests) != 0 {
		t.Errorf("全零應解出空清單，得到 %d 筆記錄／%d 組座標",
			len(st.Tiles), len(st.Dests))
	}
}
