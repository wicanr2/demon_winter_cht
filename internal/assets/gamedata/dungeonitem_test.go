package gamedata

import (
	"os"
	"path/filepath"
	"testing"
)

func loadItems(t *testing.T) DungeonItems {
	t.Helper()
	p := filepath.Join(origDataDir(t), "FILES.DTT")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Skipf("原始檔案不存在，skip: %s", p)
	}
	pool, err := LoadStringPool(p)
	if err != nil {
		t.Fatal(err)
	}
	items, err := LoadDungeonItems(pool)
	if err != nil {
		t.Fatal(err)
	}
	return items
}

// 50 件、每件 6 條、正好用完前 300 條 —— 兩個數字都是原版明寫的
// （`0x18d64` 的 `cmp …,0x32`、`0x18cd6` 的 `add …,6`）。
func TestDungeonItemLayout(t *testing.T) {
	items := loadItems(t)
	if len(items) != DungeonItemCount {
		t.Fatalf("解出 %d 件，預期 %d", len(items), DungeonItemCount)
	}
	if DungeonItemCount*DungeonItemFields != 300 {
		t.Fatalf("50 × 6 應該是 300")
	}
}

// **每一件都有名字。** 這是切割對不對最直接的訊號：
// 只要 6 的步長錯位一格，就會有一堆空名字。
func TestEveryDungeonItemHasAName(t *testing.T) {
	for i, it := range loadItems(t) {
		if it.Name == "" {
			t.Errorf("第 %d 件沒有名字 —— 6 條一件的切法可能錯位", i)
		}
	}
}

// `UseWith` 存的是**另一件道具的名字**，而且一定找得到同名的一件。
//
// 這一條是整個切割方式最強的交叉驗證：18 個字串全部命中同一份清單裡的
// 其他成員，這在切錯的情況下不可能發生。
func TestUseWithAlwaysNamesAnotherItem(t *testing.T) {
	items := loadItems(t)
	n := 0
	for i, it := range items {
		if it.UseWith == "" {
			continue
		}
		n++
		if _, ok := items.ByName(it.UseWith); !ok {
			t.Errorf("第 %d 件（%s）要用 %q，但清單裡沒有這件東西",
				i, it.Name, it.UseWith)
		}
	}
	if n == 0 {
		t.Fatal("一件需要搭配道具的都沒有 —— 那 +4 欄大概不是這個意思")
	}
	t.Logf("%d 件需要搭配另一件道具", n)
}

// `ActionBecome`（`N`）的參數也是道具名，同樣要找得到。
func TestBecomeActionNamesAnotherItem(t *testing.T) {
	items := loadItems(t)
	n := 0
	for i, it := range items {
		if it.Action() != ActionBecome {
			continue
		}
		n++
		if _, ok := items.ByName(it.ActionParam()); !ok {
			t.Errorf("第 %d 件（%s）用完會變成 %q，但清單裡沒有這件東西",
				i, it.Name, it.ActionParam())
		}
	}
	if n == 0 {
		t.Fatal("沒有任何 N 動作 —— 動作碼的判讀要重看")
	}
}

// 動作碼只有 D／N／T／P／S 五種，沒有別的。
func TestActionCodesAreClosed(t *testing.T) {
	ok := map[DungeonItemAction]bool{
		ActionNone: true, ActionDescribe: true, ActionBecome: true,
		ActionTeleport: true, ActionPassage: true, ActionStory: true,
	}
	for i, it := range loadItems(t) {
		if !ok[it.Action()] {
			t.Errorf("第 %d 件（%s）的動作碼是 %q，不在已知的五種裡",
				i, it.Name, string(it.UseResult[0]))
		}
	}
}

// 拿得走的與拿不走的都要有 —— 全有或全無代表判讀錯了。
func TestSomeItemsAreImmovable(t *testing.T) {
	takeable, fixed := 0, 0
	for _, it := range loadItems(t) {
		if it.CanTake() {
			takeable++
		} else {
			fixed++
		}
	}
	if takeable == 0 || fixed == 0 {
		t.Fatalf("拿得走 %d 件、拿不走 %d 件 —— 應該兩者都有", takeable, fixed)
	}
	t.Logf("拿得走 %d 件、拿不走 %d 件", takeable, fixed)
}

// 字串池太短要報錯，不要切出一堆空的。
func TestLoadDungeonItemsRejectsShortPool(t *testing.T) {
	if _, err := LoadDungeonItems(&StringPool{}); err == nil {
		t.Error("空字串池應該被拒絕")
	}
}
