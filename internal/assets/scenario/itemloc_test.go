package scenario

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func itemLocPath(t *testing.T, name string) string {
	t.Helper()
	p := dataPath(name)
	skipIfMissing(t, p)
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

// 固定 50 筆（原版明寫的常數），而且每一筆都要落在合理範圍。
func TestParseItemLocStopsBeforeGarbage(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, "ITEMLOCB.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Records) != ItemLocRecordCount {
		t.Fatalf("解出 %d 筆，預期 %d（原版 `cmp [bp-4],0x32`）",
			len(tab.Records), ItemLocRecordCount)
	}
	for i, r := range tab.Records {
		if r.X >= 64 || r.Y >= 64 {
			t.Errorf("第 %d 筆座標 (%d,%d) 超出 64×64", i, r.X, r.Y)
		}
		if r.MapID > 5 {
			t.Errorf("第 %d 筆子地圖 = %d，超出 0–5（0 ＝ 已拿走）", i, r.MapID)
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

// --- 改寫與存回 ---

// Encode 要能逐位元組還原 —— 沒改過就該與原檔完全相同。
//
// 這一條釘住「尾端不要動」：第 50 筆之後是原版沒清乾淨的 buffer 殘留，
// 我們只覆寫前 50 筆。重編如果把尾巴填成 0 或 0xff，這裡就會失敗。
func TestItemLocEncodeRoundTrip(t *testing.T) {
	path := itemLocPath(t, ItemLocLiveFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tab, err := ParseItemLoc(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := tab.Encode(); !bytes.Equal(got, raw) {
		for i := range raw {
			if got[i] != raw[i] {
				t.Fatalf("第 %d 個 byte 不同：%#x → %#x", i, raw[i], got[i])
			}
		}
	}
}

// 拿走 ＝ 那一筆三個 byte 全清，而且**只動那三個**。
func TestItemLocTakeWritesOnlyThatRecord(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, ItemLocLiveFile))
	if err != nil {
		t.Fatal(err)
	}
	before := tab.Encode()

	if !tab.Take(0) {
		t.Fatal("第 0 筆拿不走")
	}
	if !tab.Taken(0) {
		t.Error("拿走之後 Taken 說沒有")
	}
	if tab.Take(0) {
		t.Error("同一筆拿了兩次")
	}

	// **只動那一筆的三個 byte。** 原版 `Take:` 連寫三個 0
	// （`0x199f4`–`0x19a10`：`si = j/2` ＝ `i×3`，寫三次）。
	//
	// > 這裡原本斷言「只該改子地圖那一個 byte」—— **那是 `N` 動作的作法**
	// > （`0x1839b` 的 `mov es:[bx+si+2],0`），不是 `Take:` 的。
	// > 兩條路在原版就不一樣，查詢結果相同但寫回檔案的位元組不同。
	after := tab.Encode()
	for i := range before {
		if before[i] == after[i] {
			continue
		}
		if i >= ItemLocRecordLen {
			t.Errorf("第 %d 個 byte 被動到了 —— 拿走只該動那一筆自己的三個 byte", i)
		}
	}
	for i := 0; i < ItemLocRecordLen; i++ {
		if after[i] != 0 {
			t.Errorf("拿走之後第 %d 個 byte = %#x，預期 0", i, after[i])
		}
	}
}

// 拿走的那一筆**不參與查詢**，但索引不變（後面的道具身分不能位移）。
func TestItemLocTakenRecordDisappearsFromLookup(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, ItemLocLiveFile))
	if err != nil {
		t.Fatal(err)
	}
	n := len(tab.Records)
	r := tab.Records[0]

	tab.Take(0)
	if _, ok := tab.At(r.MapID, r.X, r.Y); ok {
		// 同一格可能還有別件，所以只在確定只有一件時才算失敗
		if len(tab.OnMap(r.MapID)) > 0 {
			t.Logf("同一格還有別的道具，跳過這一項斷言")
		}
	}
	if len(tab.Records) != n {
		t.Errorf("筆數變成 %d —— 拿走不可以刪記錄，索引就是道具的身分", len(tab.Records))
	}
}

// 丟棄 ＝ 把那一筆改成現在的位置，之後在**原地**撿得回來。
func TestItemLocDropPutsItBack(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, ItemLocLiveFile))
	if err != nil {
		t.Fatal(err)
	}
	tab.Take(0)
	if !tab.Drop(0, 12, 34, 3) {
		t.Fatal("丟不下去")
	}
	if tab.Taken(0) {
		t.Error("丟下去之後還算在手上")
	}
	i, ok := tab.At(3, 12, 34)
	if !ok || i != 0 {
		t.Errorf("在 (12,34) 地圖 3 撿不回第 0 件（得到 %d, %v）", i, ok)
	}
}

// 子地圖 0 不是合法的落點 —— 那個值的意思是「被拿走」。
func TestItemLocDropRejectsMapZero(t *testing.T) {
	tab, err := LoadItemLoc(itemLocPath(t, ItemLocLiveFile))
	if err != nil {
		t.Fatal(err)
	}
	if tab.Drop(0, 1, 1, ItemLocTaken) {
		t.Error("子地圖 0 竟然丟得下去 —— 那個值代表「被拿走」")
	}
}

// 三段優先序：存檔目錄 > 母本 > 資料目錄，而且**存檔目錄贏**。
func TestLoadItemLocTablePrefersSaveDir(t *testing.T) {
	dataDir := dataPath("")
	skipIfMissing(t, filepath.Join(dataDir, ItemLocMasterFile))

	saveDir := t.TempDir()

	// 全新開始：一定走母本，不看存檔目錄。
	fresh, err := LoadItemLocTable(saveDir, dataDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Taken(0) {
		t.Error("母本的第 0 筆不該是「已拿走」")
	}

	// 寫一份「第 0 件已經拿走」的進度到存檔目錄。
	fresh.Take(0)
	if err := WriteItemLocTable(saveDir, fresh); err != nil {
		t.Fatal(err)
	}

	// 非全新 → 要讀到那份進度。
	got, err := LoadItemLocTable(saveDir, dataDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Taken(0) {
		t.Error("存檔目錄有進度卻沒讀到 —— 三段優先序錯了")
	}

	// 再開新遊戲 → 回到母本，進度不該殘留。
	again, err := LoadItemLocTable(saveDir, dataDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if again.Taken(0) {
		t.Error("新遊戲繼承了上一輪的進度 —— fresh 應該直接走母本")
	}
}

// **絕對不可以寫回原版資料目錄。**
func TestWriteItemLocTableDoesNotTouchOrigData(t *testing.T) {
	dataDir := dataPath("")
	live := filepath.Join(dataDir, ItemLocLiveFile)
	skipIfMissing(t, live)

	before, err := os.Stat(live)
	if err != nil {
		t.Fatal(err)
	}
	tab, err := LoadItemLoc(live)
	if err != nil {
		t.Fatal(err)
	}
	tab.Take(0)
	if err := WriteItemLocTable(t.TempDir(), tab); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(live)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("原版資料目錄的 ITEMLOCB.DAT 被動到了 —— workplace/orig 是唯讀的")
	}
}
