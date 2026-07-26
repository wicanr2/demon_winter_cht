package world

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExits_RealFile(t *testing.T) {
	dir := origDataDir(t)
	et, err := LoadExits(filepath.Join(dir, "EXITS.DAT"))
	if err != nil {
		t.Fatalf("LoadExits 失敗: %v", err)
	}

	if got := et.Len(); got != exitRecordCount {
		t.Fatalf("Len() = %d, want %d", got, exitRecordCount)
	}
	all := et.All()
	if len(all) != exitRecordCount {
		t.Fatalf("All() 長度 = %d, want %d", len(all), exitRecordCount)
	}

	// 兩邊的座標都必須落在 64×64 地圖的可用範圍（1–62）。
	// 只驗來源座標的話，切分單位錯了也可能剛好過 —— 目的座標一起驗才擋得住。
	for i, r := range all {
		if r.FromX < 1 || r.FromX > 62 || r.FromY < 1 || r.FromY > 62 {
			t.Errorf("記錄 #%d 來源座標 (%d,%d) 超出 1–62", i, r.FromX, r.FromY)
		}
		if r.ToX < 1 || r.ToX > 62 || r.ToY < 1 || r.ToY > 62 {
			t.Errorf("記錄 #%d 目的座標 (%d,%d) 超出 1–62", i, r.ToX, r.ToY)
		}
	}

	first := all[0]
	got, ok := et.Lookup(first.FromX, first.FromY)
	if !ok {
		t.Fatalf("Lookup(%d,%d) 應該命中第一筆", first.FromX, first.FromY)
	}
	if got != first {
		t.Errorf("Lookup 應回傳第一筆（比照原版線性掃描），得到 %+v", got)
	}
	if _, ok := et.Lookup(0, 0); ok {
		t.Error("Lookup(0,0) 預期 ok=false")
	}
}

// TestLoadExits_ExitsArePaired 是「6-byte 才是正確切分單位」的關鍵證據。
//
// 出口成對：從 A 走到 B，B 的落點附近也有一筆走回 A。目的地一律差一格
// （不然踩過去會立刻被彈回來），所以「走回來那一筆」的來源座標與
// 本筆目的座標相鄰而不相等。
//
// 3-byte 的舊切法把每一筆的欄位都切錯位，湊不出這種關係 ——
// 這條測試就是拿資料本身當 oracle，不靠「反組譯讀起來合理」。
func TestLoadExits_ExitsArePaired(t *testing.T) {
	dir := origDataDir(t)
	et, err := LoadExits(filepath.Join(dir, "EXITS.DAT"))
	if err != nil {
		t.Fatalf("LoadExits 失敗: %v", err)
	}
	all := et.All()

	adjacent := func(ax, ay, bx, by byte) bool {
		dx, dy := int(ax)-int(bx), int(ay)-int(by)
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		return dx+dy <= 1
	}

	paired := 0
	for _, r := range all {
		for _, back := range all {
			// 回程：來源在本筆目的地的相鄰格，且它指回本筆的來源格。
			if adjacent(back.FromX, back.FromY, r.ToX, r.ToY) &&
				adjacent(back.ToX, back.ToY, r.FromX, r.FromY) {
				paired++
				break
			}
		}
	}

	// 40/55 是實測值，不是門檻猜的。**沒有回程的 15 筆不是雜訊** ——
	// 它們依 Unknown 欄位分群：
	//
	//	Unknown 10 → 六筆全部通往圖 5（禁錮惡魔那張圖，game.ImprisonSubMap）
	//	Unknown  8 → 四筆通往圖 1／圖 4
	//	Unknown  1 → 三筆通往圖 34 的同一格 (55,8)
	//
	// 單向通往終局區域是合理的設計（走進去就不能走回來）。
	// 拿實測值當錨點而不是放寬門檻 —— 這個數字變了就該回頭看為什麼。
	const wantPaired = 40
	if paired != wantPaired {
		t.Errorf("有回程的出口 %d/%d，預期 %d。數字變了就要查為什麼，"+
			"不要直接改這個期望值", paired, len(all), wantPaired)
	}

	// 這才是「6-byte 才對」的判別式：同一份位元組用舊的 3-byte 切法讀，
	// 成對數會掉到 **0**。門檻式的斷言（「大多數要成對」）沒有這個力道 ——
	// 門檻可以被放寬到讓錯的切法也過。
	if n := pairedAtStride3(t, dir); n != 0 {
		t.Errorf("3-byte 切法竟然有 %d 筆成對 —— "+
			"那這條測試就不能用來判別切分單位了", n)
	}
}

// pairedAtStride3 用被推翻的 3-byte 切法重算成對數，當作對照組。
func pairedAtStride3(t *testing.T, dir string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "EXITS.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	type rec struct{ a, b, c byte }
	var recs []rec
	for i := 0; i+2 < len(data); i += 3 {
		recs = append(recs, rec{data[i], data[i+1], data[i+2]})
	}
	adj := func(ax, ay, bx, by byte) bool {
		dx, dy := int(ax)-int(bx), int(ay)-int(by)
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		return dx+dy <= 1
	}
	n := 0
	for _, r := range recs {
		for _, back := range recs {
			if adj(back.a, back.b, r.b, r.c) && adj(back.b, back.c, r.a, r.b) {
				n++
				break
			}
		}
	}
	return n
}

func TestLoadExits_WrongSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "BAD.DAT")
	if err := os.WriteFile(path, make([]byte, 10), 0o644); err != nil {
		t.Fatalf("寫測試檔失敗: %v", err)
	}
	if _, err := LoadExits(path); err == nil {
		t.Error("LoadExits 對長度不符的檔案預期回傳 error，卻沒有")
	}
}

func TestLoadExits_MissingFile(t *testing.T) {
	if _, err := LoadExits("/nonexistent/path/EXITS.DAT"); err == nil {
		t.Error("LoadExits 對不存在的檔案預期回傳 error，卻沒有")
	}
}
