package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dataDir 是原版遊戲資料目錄。CLAUDE.md 規定 workplace/orig/ 唯讀，測試只讀取，
// 不修改任何檔案；檔案不存在時整組測試 skip（不是 fail），符合任務要求
// 「對真實原始檔案測試，檔案不存在則 skip」。
const dataDir = "../../../workplace/orig/demwin/DEM_DATA"

func dataPath(name string) string {
	return filepath.Join(dataDir, name)
}

// skipIfMissing 在對應原始檔案不存在時 skip 該測試（不是 fail），符合任務要求
// 「對真實原始檔案測試，檔案不存在則 skip」。用 os.Stat 直接判斷檔案是否存在，
// 不靠解析 error 訊息字串猜測失敗原因。
func skipIfMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("原始檔案不存在，skip: %s", path)
	}
}

func TestLoadEventTable_AllFilesFullyConsumed(t *testing.T) {
	// 已驗證的欄位數／記錄數錨點，來自 docs/re/02-data-loading-functions.md §2.3
	// 獨立 Python 模擬器對 5 個 DATA*.TXT 的 100% 消耗驗證結果
	// （221/82/152/118/59 個欄位，35/12/26/21/11 次迭代，零殘留零錯位）。
	cases := []struct {
		file         string
		wantEventLen int
	}{
		{"DATA1.TXT", 35},
		{"DATA2.TXT", 12},
		{"DATA3.TXT", 26},
		{"DATA4.TXT", 21},
		{"DATA5.TXT", 11},
	}

	for _, c := range cases {
		c := c
		t.Run(c.file, func(t *testing.T) {
			path := dataPath(c.file)
			skipIfMissing(t, path)

			table, err := LoadEventTable(path)
			if err != nil {
				t.Fatalf("LoadEventTable(%s) 失敗（代表欄位消耗順序跟檔案對不上，零殘留/零錯位的要求沒達成）: %v", path, err)
			}
			if got := table.Len(); got != c.wantEventLen {
				t.Errorf("%s: 事件記錄數 = %d，預期 %d（docs/re/02 記載的獨立模擬器結果）", c.file, got, c.wantEventLen)
			}
		})
	}
}

// TestLoadEventTable_DATA2Anchors 驗證三個已知內容錨點（任務驗收重點）：
// 「四隻狗頭人的帳篷」「Uffuspgot 隊長的帳篷」「cave bear 的巢穴」的怪物 ID。
func TestLoadEventTable_DATA2Anchors(t *testing.T) {
	path := dataPath("DATA2.TXT")
	skipIfMissing(t, path)

	table, err := LoadEventTable(path)
	if err != nil {
		t.Fatalf("LoadEventTable(%s) 失敗: %v", path, err)
	}

	events := table.All()

	find := func(substr string) (Event, bool) {
		for _, e := range events {
			if strings.Contains(e.Text, substr) {
				return e, true
			}
		}
		return Event{}, false
	}

	t.Run("四隻狗頭人的帳篷", func(t *testing.T) {
		e, ok := find("tent of four kobolds")
		if !ok {
			t.Fatal("找不到「tent of four kobolds」這筆事件")
		}
		want := []int{26, 26, 26, 26}
		if !intsEqual(e.MonsterIDs, want) {
			t.Errorf("MonsterIDs = %v，預期 %v", e.MonsterIDs, want)
		}
	})

	t.Run("Uffuspgot隊長的帳篷", func(t *testing.T) {
		e, ok := find("Uffuspgot the kobold Captain")
		if !ok {
			t.Fatal("找不到「Uffuspgot the kobold Captain」這筆事件")
		}
		want := []int{26, 26, 85, 2, 2}
		if !intsEqual(e.MonsterIDs, want) {
			t.Errorf("MonsterIDs = %v，預期 %v", e.MonsterIDs, want)
		}
	})

	t.Run("cave_bear的巢穴", func(t *testing.T) {
		e, ok := find("den of a cave bear")
		if !ok {
			t.Fatal("找不到「den of a cave bear」這筆事件")
		}
		want := []int{67}
		if !intsEqual(e.MonsterIDs, want) {
			t.Errorf("MonsterIDs = %v，預期 %v", e.MonsterIDs, want)
		}
	})
}

// TestLoadEventTable_ChainRedrawAndRuneGlyph 驗證 docs/re/02 §2.4 [F] 記載的
// 兩種控制前綴機制：'3' 觸發重繪、'%' 觸發符文字型。用 DATA1.TXT 已知案例
// （Remondadin 戰後反應文字、鐘室符文密語）核對。
func TestLoadEventTable_ChainRedrawAndRuneGlyph(t *testing.T) {
	path := dataPath("DATA1.TXT")
	skipIfMissing(t, path)

	table, err := LoadEventTable(path)
	if err != nil {
		t.Fatalf("LoadEventTable(%s) 失敗: %v", path, err)
	}

	var sawChainRedraw, sawRuneGlyph bool
	for _, e := range table.All() {
		if e.IsChainRedraw() {
			sawChainRedraw = true
			t.Logf("chain redraw 觸發於: %q -> ChainRedrawText()=%q", e.Continuation, e.ChainRedrawText())
		}
		if e.IsRuneGlyph() {
			sawRuneGlyph = true
		}
	}
	if !sawChainRedraw {
		t.Error("DATA1.TXT 應該至少有一筆記錄觸發 IsChainRedraw()（如 Remondadin 戰後反應文字），卻一筆都沒有")
	}
	if !sawRuneGlyph {
		t.Error("DATA1.TXT 應該至少有一筆記錄觸發 IsRuneGlyph()（如鐘室符文密語），卻一筆都沒有")
	}
}

// TestLoadEventTable_DATA3RuneCipher 驗證 event-script.md §2.4 提到的
// 「YMROS IS MINE」符文密語錨點（庫瑞克事件表）。
func TestLoadEventTable_DATA3RuneCipher(t *testing.T) {
	path := dataPath("DATA3.TXT")
	skipIfMissing(t, path)

	table, err := LoadEventTable(path)
	if err != nil {
		t.Fatalf("LoadEventTable(%s) 失敗: %v", path, err)
	}

	found := false
	for _, e := range table.All() {
		if e.IsRuneGlyph() && strings.Contains(e.Continuation, "YMROS") {
			found = true
			if !strings.Contains(e.RuneText(), "YMROS.IS...MINE") {
				t.Errorf("RuneText() = %q，預期包含 YMROS.IS...MINE", e.RuneText())
			}
		}
	}
	if !found {
		t.Error("DATA3.TXT 應該有一筆「YMROS IS MINE」符文密語記錄")
	}
}

func TestLoadEventTable_MissingFile(t *testing.T) {
	_, err := LoadEventTable(filepath.Join(dataDir, "DOES_NOT_EXIST.TXT"))
	if err == nil {
		t.Fatal("預期檔案不存在時回傳 error，卻回傳 nil")
	}
}

func TestParseFieldInt(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"255", 255},
		{"0", 0},
		{"-3", -3},
		{"3With Remondadin dead the black room is strangely silent", 3},
		{"%RING.BELL...AT.....MIDNIGHT", 0},
		{"", 0},
		{"  42", 42},
	}
	for _, c := range cases {
		if got := parseFieldInt(c.in); got != c.want {
			t.Errorf("parseFieldInt(%q) = %d，預期 %d", c.in, got, c.want)
		}
	}
}

func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
