package i18n

import (
	"os"
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

// TestUICatalog 釘住介面文案目錄（語意化 key，`docs/i18n/ui-catalog.md`）。
func TestUICatalog(t *testing.T) {
	dir := t.TempDir()
	body := `{
  "locale": "zh-Hant",
  "entries": [
    {"key":"plot.uncurse","en":"UNCURSE","text":"解咒"},
    {"key":"plot.needsp","text":"那需要 %d 點法力"}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "ui.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tr, err := Load(dir)
	if err != nil {
		t.Fatalf("Load 失敗：%v", err)
	}
	if got := tr.UI("plot.uncurse"); got != "解咒" {
		t.Errorf("UI(plot.uncurse) = %q", got)
	}
	if got := tr.UI("plot.needsp"); got != "那需要 %d 點法力" {
		t.Errorf("UI(plot.needsp) = %q", got)
	}
	// 不存在的 key 必須醒目，不能靜默退回藏在程式裡的另一份文字。
	if got := tr.UI("nope"); got != "⟦nope⟧" {
		t.Errorf("不存在的 key 應醒目顯示，得到 %q", got)
	}
	if n := tr.UICount(); n != 2 {
		t.Errorf("UICount = %d，預期 2", n)
	}
}

// TestUIAndIndexCoexist 釘住「名稱型與數字型目錄互不干擾」。
func TestUIAndIndexCoexist(t *testing.T) {
	dir := t.TempDir()
	must := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("ui.json", `{"locale":"zh-Hant","entries":[{"key":"a.b","text":"介面"}]}`)
	must("data1.txt", "@@ DATA1.TXT\n\n## 0\n:: en\nOrig\n:: zh\n事件\n")

	tr, err := Load(dir)
	if err != nil {
		t.Fatalf("Load 失敗：%v", err)
	}
	if got := tr.UI("a.b"); got != "介面" {
		t.Errorf("UI = %q", got)
	}
	if got := tr.Event("DATA1.TXT", 0, "Orig"); got != "事件" {
		t.Errorf("Event = %q", got)
	}
	// 名稱型的條目不該污染數字型的表
	if got := tr.Event("UI", -1, "Orig"); got != "Orig" {
		t.Errorf("名稱型條目跑進 byIndex 了：%q", got)
	}
}

// TestVerify_IgnoresNameTypeEntries 釘住「名稱型條目不參與索引核實」。
//
// 續接碼第二段用的是名稱型 key（`chain.DATA1.TXT.3`），沒有數字索引。
// 曾經因為沒跳過它們，每一條都被 `Index < 0` 判成「索引超出範圍」，
// 實機一啟動就印「4 條譯文的原文對不上，重跑 dwstrings events」——
// 而譯文完全正確，`dwstrings check` 那邊是 368/368 通過的。
func TestVerify_IgnoresNameTypeEntries(t *testing.T) {
	dir := t.TempDir()
	body := `@@ DATA1.TXT

## 0
:: en
Two guards are in the room
:: zh
房裡有兩名衛兵。

## chain.DATA1.TXT.3
:: zh
續接碼第二段
`
	if err := os.WriteFile(filepath.Join(dir, "data1.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tr, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Verify(dir, "DATA1.TXT", []string{"Two guards are in the room"}); err != nil {
		t.Fatal(err)
	}
	if n := len(tr.Mismatched()); n != 0 {
		t.Errorf("名稱型條目不該被判定脫節，卻回報了 %d 條", n)
	}
	if got := tr.UI("chain.DATA1.TXT.3"); got != "續接碼第二段" {
		t.Errorf("名稱型譯文應保留，得到 %q", got)
	}
}
