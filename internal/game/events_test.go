package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

func loadExits(t *testing.T) *world.ExitTable {
	t.Helper()
	p := filepath.Join(repoRoot(t), "workplace", "orig", "demwin", "DEM_DATA", "EXITS.DAT")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("game: 找不到 %s，略過需要真實資料的測試", p)
	}
	tb, err := world.LoadExits(p)
	if err != nil {
		t.Fatalf("LoadExits: %v", err)
	}
	return tb
}

// 驗收 2：類別分布必須是 0 佔 94 筆、1 佔 14 筆、2 佔 2 筆，無其他類別。
func TestExitTable_CategoryDistribution(t *testing.T) {
	tb := loadExits(t)

	got := map[int]int{}
	for _, r := range tb.All() {
		got[r.Category()]++
	}

	want := map[int]int{0: 94, 1: 14, 2: 2}
	if len(got) != len(want) {
		t.Errorf("出現的類別數：得到 %d 種 %v，預期 %d 種 %v", len(got), got, len(want), want)
	}
	for cat, n := range want {
		if got[cat] != n {
			t.Errorf("類別 %d：得到 %d 筆，預期 %d 筆", cat, got[cat], n)
		}
	}
}

// 驗收 1：真實 EXITS.DAT 的 110 筆記錄裡沒有 record[0]==0 的終止標記，
// 掃描要跑完全部 110 筆。終止標記落在 511-byte 緩衝區的檔案之後。
func TestLookupEvent_NoTerminatorWithinFile(t *testing.T) {
	tb := loadExits(t)

	for i, r := range tb.All() {
		if r.X == 0 {
			t.Fatalf("第 %d 筆的 X 為 0，檔案內不該出現終止標記", i)
		}
	}

	// 查一個不存在的座標：掃完 110 筆都沒命中，回報 Found=false。
	if q := LookupEvent(tb, 0, 0, nil); q.Found {
		t.Error("座標 (0,0) 不應命中任何事件")
	}
}

// 每一筆記錄用自己的座標查回來，類別與子值必須一致。
// 同座標有重複記錄時，原版是線性掃描命中第一筆，這裡照同一語意驗。
func TestLookupEvent_RoundTrip(t *testing.T) {
	tb := loadExits(t)
	all := tb.All()

	seen := map[[2]byte]bool{}
	for i, r := range all {
		key := [2]byte{r.X, r.Y}
		if seen[key] {
			continue // 重複座標只驗第一筆
		}
		seen[key] = true

		q := LookupEvent(tb, r.X, r.Y, nil)
		if !q.Found {
			t.Fatalf("第 %d 筆 (%d,%d) 查不到", i, r.X, r.Y)
		}
		if q.RecordIndex != i {
			t.Errorf("(%d,%d)：命中第 %d 筆，預期第 %d 筆", r.X, r.Y, q.RecordIndex, i)
		}
		if int(q.Category) != r.Category() {
			t.Errorf("(%d,%d)：類別 %d，預期 %d", r.X, r.Y, q.Category, r.Category())
		}
		if q.SubValue != r.SubValue() {
			t.Errorf("(%d,%d)：子值 %d，預期 %d", r.X, r.Y, q.SubValue, r.SubValue())
		}
	}
}

// 事件索引的累計規則：命中之前掃過的類別 1／2 記錄才計數，
// 且命中的那筆若本身是類別 1／2，索引再 +1。
func TestLookupEvent_IndexAccumulation(t *testing.T) {
	tb := loadExits(t)
	all := tb.All()

	seen := map[[2]byte]bool{}
	for i, r := range all {
		key := [2]byte{r.X, r.Y}
		if seen[key] {
			continue
		}
		seen[key] = true

		// 獨立算一次期望值，不重用被測程式的邏輯。
		want := 0
		for j := 0; j < i; j++ {
			switch all[j].Type & 0xe0 {
			case 0x20, 0x40:
				want++
			}
		}
		if c := r.Category(); c == 1 || c == 2 {
			want++
		}

		if q := LookupEvent(tb, r.X, r.Y, nil); q.Index != want {
			t.Errorf("(%d,%d) 第 %d 筆：事件索引 %d，預期 %d", r.X, r.Y, i, q.Index, want)
		}
	}
}

// 事件索引不得超出對應 DATA*.TXT 的記錄數，否則顯示文字時會越界。
func TestLookupEvent_IndexWithinDataFile(t *testing.T) {
	tb := loadExits(t)

	max := 0
	seen := map[[2]byte]bool{}
	for _, r := range tb.All() {
		key := [2]byte{r.X, r.Y}
		if seen[key] {
			continue
		}
		seen[key] = true
		if q := LookupEvent(tb, r.X, r.Y, nil); q.Index > max {
			max = q.Index
		}
	}

	// EXITS.DAT 是隨 dataset 整份覆寫的，手上這份對應哪個 DATA*.TXT 未定，
	// 所以這裡只檢查「不超過任何一個 DATA 檔的筆數」這個寬鬆上界。
	t.Logf("事件索引最大值 = %d", max)
	if max < 0 || max > 16 {
		t.Errorf("事件索引最大值 %d 落在不合理的範圍（類別 1/2 共 16 筆，索引不該超過 16）", max)
	}
}

func TestTriggerFor(t *testing.T) {
	cases := []struct {
		tile byte
		want TriggerKind
	}{
		{0x11, TriggerLookup},
		{0x53, TriggerLookup},
		{0x35, TriggerHardBlock},
		{0x25, TriggerDirectIndex},
		{0x26, TriggerDirectIndex},
		{0x2e, TriggerDirectIndex},
		{0x5b, TriggerDirectIndex},
		{0x64, TriggerDirectIndex}, // Ghidra 漏掉的那一個
		{0x00, TriggerNone},
		{0x01, TriggerNone},
		{0x7f, TriggerNone},
	}
	for _, c := range cases {
		if got := TriggerFor(c.tile); got != c.want {
			t.Errorf("tile 0x%02x：得到 %d，預期 %d", c.tile, got, c.want)
		}
	}
}

// 傳送目標從 0x1ff 反向成長：第 n 個傳送點的目標在 0x1ff − (2n+2)。
// 真實 EXITS.DAT 沒有類別 4 的記錄，所以這裡用合成資料驗公式本身。
func TestLookupEvent_TeleportTailIndexing(t *testing.T) {
	// 三筆記錄：兩個傳送點在前，第三筆是要命中的傳送點（第 3 個，n=2）。
	recs := []world.ExitRecord{
		{X: 5, Y: 5, Type: 0x80}, // 類別 4，teleportCount 0 -> 1
		{X: 6, Y: 6, Type: 0x80}, // 類別 4，teleportCount 1 -> 2
		{X: 7, Y: 7, Type: 0x81}, // 類別 4 子值 1，命中時 teleportCount = 2
	}

	tail := make([]byte, exitBufferSize)
	idx := exitBufferSize - (2*2 + 2) // 0x1ff - 6 = 505
	tail[idx] = 33
	tail[idx+1] = 44

	q := LookupEvent(fakeExits(recs), 7, 7, tail)

	if !q.Found || q.Category != CatTeleport {
		t.Fatalf("應命中類別 4，得到 Found=%v Category=%d", q.Found, q.Category)
	}
	if q.TeleportX != 33 || q.TeleportY != 44 {
		t.Errorf("傳送目標：得到 (%d,%d)，預期 (33,44)", q.TeleportX, q.TeleportY)
	}
	if q.SubValue != 1 {
		t.Errorf("子值：得到 %d，預期 1", q.SubValue)
	}
}

// 掃到 X==0 就停，後面的記錄不再看。
func TestLookupEvent_StopsAtTerminator(t *testing.T) {
	recs := []world.ExitRecord{
		{X: 5, Y: 5, Type: 0x20},
		{X: 0, Y: 0, Type: 0}, // 終止標記
		{X: 9, Y: 9, Type: 0x20},
	}
	if q := LookupEvent(fakeExits(recs), 9, 9, nil); q.Found {
		t.Error("終止標記之後的記錄不應被查到")
	}
}

type fakeExits []world.ExitRecord

func (f fakeExits) All() []world.ExitRecord { return f }
