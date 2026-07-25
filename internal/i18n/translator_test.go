package i18n

import (
	"path/filepath"
	"testing"
)

// writeTestCatalog 建一個暫時的語言目錄。
func writeTestCatalog(t *testing.T, dir string, c *Catalog) {
	t.Helper()
	if err := WriteCatalog(filepath.Join(dir, CatalogFileName(c.Source)), c); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_MissingDirIsOriginalTextMode(t *testing.T) {
	tr, err := Load(filepath.Join(t.TempDir(), "沒有這個目錄"))
	if err != nil {
		t.Fatalf("翻譯目錄不存在應視為原文模式，不該是錯誤：%v", err)
	}
	if got := tr.Event("DATA1.TXT", 0, "original"); got != "original" {
		t.Errorf("原文模式應回原文，得到 %q", got)
	}
}

func TestTranslator_LookupAndFallback(t *testing.T) {
	dir := t.TempDir()
	writeTestCatalog(t, dir, &Catalog{Source: "DATA1.TXT", Entries: []Entry{
		{Index: 0, Source: "Two guards are in the room", Target: "房裡有兩名衛兵。"},
		{Index: 1, Source: "untranslated line", Target: ""},
		{Index: 2, Source: "needs review", Target: "舊譯文", NeedsReview: true},
	}})

	tr, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if got := tr.Event("DATA1.TXT", 0, "Two guards are in the room"); got != "房裡有兩名衛兵。" {
		t.Errorf("已翻譯的條目應回中文，得到 %q", got)
	}
	// 未翻譯與待複查都退回原文 —— 缺譯在畫面上是英文，看得見；
	// 回空字串會變成敘述憑空消失。
	if got := tr.Event("DATA1.TXT", 1, "untranslated line"); got != "untranslated line" {
		t.Errorf("未翻譯應回原文，得到 %q", got)
	}
	if got := tr.Event("DATA1.TXT", 2, "needs review"); got != "needs review" {
		t.Errorf("待複查應回原文，得到 %q", got)
	}
	if got := tr.Event("DATA1.TXT", 99, "no entry"); got != "no entry" {
		t.Errorf("沒有這一條時應回原文，得到 %q", got)
	}
	if got := tr.Event("DATA9.TXT", 0, "other file"); got != "other file" {
		t.Errorf("別的來源檔不應命中，得到 %q", got)
	}
}

// 索引錯位是這一層要擋的主要事故：每一句都通順、每一句都接錯地方。
func TestVerify_DropsDriftedEntries(t *testing.T) {
	dir := t.TempDir()
	writeTestCatalog(t, dir, &Catalog{Source: "DATA1.TXT", Entries: []Entry{
		{Index: 0, Source: "Two guards are in the room", Target: "房裡有兩名衛兵。"},
		{Index: 1, Source: "A bookcase adorns the far wall", Target: "遠處牆邊有一座書架。"},
	}})

	tr, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Count() != 2 {
		t.Fatalf("應載入 2 條譯文，得到 %d", tr.Count())
	}

	// 第 1 條的原文換掉了 —— 譯文可能整段對到別的房間。
	texts := []string{"Two guards are in the room", "A completely different room"}
	if err := tr.Verify(dir, "DATA1.TXT", texts); err != nil {
		t.Fatal(err)
	}

	if n := len(tr.Mismatched()); n != 1 {
		t.Fatalf("應回報 1 條對不上，得到 %d", n)
	}
	if got := tr.Event("DATA1.TXT", 1, texts[1]); got != texts[1] {
		t.Errorf("原文對不上的條目必須退回英文，得到 %q", got)
	}
	if got := tr.Event("DATA1.TXT", 0, texts[0]); got != "房裡有兩名衛兵。" {
		t.Errorf("沒問題的條目不該被牽連，得到 %q", got)
	}
}

// 索引超出事件表範圍也算脫節。
func TestVerify_DropsOutOfRangeIndex(t *testing.T) {
	dir := t.TempDir()
	writeTestCatalog(t, dir, &Catalog{Source: "DATA1.TXT", Entries: []Entry{
		{Index: 5, Source: "gone", Target: "沒了"},
	}})
	tr, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Verify(dir, "DATA1.TXT", []string{"only one record"}); err != nil {
		t.Fatal(err)
	}
	if n := len(tr.Mismatched()); n != 1 {
		t.Errorf("索引超範圍應回報，得到 %d 條", n)
	}
}

// 只差在換行與空白的原文不算變動 —— 那些差異在斷行時本來就會被重排。
func TestVerify_IgnoresWhitespaceOnlyDifferences(t *testing.T) {
	dir := t.TempDir()
	writeTestCatalog(t, dir, &Catalog{Source: "DATA1.TXT", Entries: []Entry{
		{Index: 0, Source: "Two guards\nare in  the room", Target: "房裡有兩名衛兵。"},
	}})
	tr, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Verify(dir, "DATA1.TXT", []string{"Two guards are in the room"}); err != nil {
		t.Fatal(err)
	}
	if n := len(tr.Mismatched()); n != 0 {
		t.Errorf("只差空白不該判定為變動，卻回報了 %d 條", n)
	}
	if got := tr.Event("DATA1.TXT", 0, "x"); got != "房裡有兩名衛兵。" {
		t.Errorf("譯文應保留，得到 %q", got)
	}
}

// 缺 @@ 檔頭的翻譯檔不知道要對應哪個來源，必須報錯而不是靜默忽略。
func TestLoad_RejectsCatalogWithoutSourceHeader(t *testing.T) {
	dir := t.TempDir()
	writeTestCatalog(t, dir, &Catalog{Source: "", Entries: []Entry{
		{Index: 0, Source: "x", Target: "中"},
	}})
	if _, err := Load(dir); err == nil {
		t.Error("缺來源檔頭時應回傳錯誤")
	}
}
