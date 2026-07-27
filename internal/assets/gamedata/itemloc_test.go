package gamedata

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func itemLocPath(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(origDataDir(t), name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Skipf("原始檔案不存在，skip: %s", p)
	}
	return p
}

// 兩個檔案逐 byte 相同 —— `docs/formats/town-and-map.md` §4.1 用 md5 驗過，
// 這裡用專案自己的讀檔路徑再釘一次。
func TestItemLocBAndXAreIdentical(t *testing.T) {
	b, err := os.ReadFile(itemLocPath(t, "ITEMLOCB.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	x, err := os.ReadFile(itemLocPath(t, "ITEMLOCX.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, x) {
		t.Error("ITEMLOCB.DAT 與 ITEMLOCX.DAT 不同 —— 主檔／備份的推測要重看")
	}
	if len(b) != ItemLocFileSize {
		t.Errorf("檔案長度 = %d，預期 %d", len(b), ItemLocFileSize)
	}
}

// 解析出來的每一筆都要落在合理範圍，而且**在殘留段之前就停**。
func TestParseItemLocStopsBeforeGarbage(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, "ITEMLOCB.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Records) == 0 {
		t.Fatal("一筆都沒解出來")
	}
	if len(tab.Records) >= ItemLocRecordCount {
		t.Errorf("解出 %d 筆 —— 尾端的 buffer 殘留沒有被擋掉", len(tab.Records))
	}
	for i, r := range tab.Records {
		if r.X >= 64 || r.Y >= 64 {
			t.Errorf("第 %d 筆座標 (%d,%d) 超出 64×64", i, r.X, r.Y)
		}
		if r.MapID < 1 || r.MapID > 5 {
			t.Errorf("第 %d 筆子地圖 = %d，不在 1–5", i, r.MapID)
		}
	}
}

// 子地圖 2 與 4 是**合法的**：它們沒有獨立的 `MAPn.MAP`，
// 但在 `SUM.MAP` 裡（`docs/re/03` §3.2）。早期文件把 4 當成謎題，
// 那個懸案已經結了 —— 這條測試防止有人又把它們當雜訊濾掉。
func TestItemLocAcceptsSumMapDungeons(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, "ITEMLOCB.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[byte]int{}
	for _, r := range tab.Records {
		seen[r.MapID]++
	}
	for _, id := range []byte{1, 3, 4} {
		if seen[id] == 0 {
			t.Errorf("子地圖 %d 一筆都沒有 —— 預期有（見 town-and-map §4.2 的分布）", id)
		}
	}
	if seen[2] != 0 {
		t.Logf("子地圖 2 出現 %d 筆 —— 早期文件說從不出現，值得回頭看", seen[2])
	}
}

// `(0,0)` 是佔位不是結尾：它夾在有效記錄中間，後面還有東西。
func TestItemLocEmptyIsAPlaceholderNotATerminator(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, "ITEMLOCB.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	first := -1
	for i, r := range tab.Records {
		if r.Empty() {
			first = i
			break
		}
	}
	if first < 0 {
		t.Skip("這份資料沒有 (0,0) 佔位")
	}
	if first == len(tab.Records)-1 {
		t.Error("(0,0) 出現在最後一筆 —— 那就分不出佔位與結尾了")
	}
}

// 查詢介面：表裡真的有的座標查得到，沒有的查不到。
func TestItemLocAt(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, "ITEMLOCB.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	want := tab.Records[0]
	if _, ok := tab.At(want.MapID, want.X, want.Y); !ok {
		t.Errorf("第一筆 (%d,%d) 地圖 %d 查不到", want.X, want.Y, want.MapID)
	}
	if _, ok := tab.At(99, 1, 1); ok {
		t.Error("子地圖 99 竟然查得到")
	}
	if n := len(tab.OnMap(1)) + len(tab.OnMap(3)) +
		len(tab.OnMap(4)) + len(tab.OnMap(5)) + len(tab.OnMap(2)); n != len(tab.Records) {
		t.Errorf("依地圖分組之後總數 = %d，與全表 %d 對不上", n, len(tab.Records))
	}
}

// 長度不對就報錯，不要靜默截斷。
func TestParseItemLocRejectsWrongSize(t *testing.T) {
	if _, err := ParseItemLoc(make([]byte, 100)); err == nil {
		t.Error("100 bytes 的檔案應該被拒絕")
	}
}
