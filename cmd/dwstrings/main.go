// Command dwstrings 從原版資料抽出所有可翻譯字串，產生翻譯目錄。
//
// 兩個來源、兩種用途：
//
//   - `DATA*.TXT` 的事件敘述 —— 引擎執行時真的會讀，是**執行期資源**。
//     抽出來的目錄就是翻譯檔本身，引擎照它替換。
//   - `DEMON.INT` 的 UI 字串池 —— 本專案是重製而不是改機碼，
//     引擎不讀它。抽出來只當**清單參考**，用途是確認 UI 該有哪些文案，
//     不是執行期資源。兩者不要混。
//
// 用法：
//
//	dwstrings events -data <資料目錄> -out <輸出目錄>
//	dwstrings ui     -int  <DEMON.INT> -out <輸出檔>
//	dwstrings spells -data <資料目錄> -out <輸出目錄>
//	dwstrings check  -data <資料目錄> -lang <翻譯目錄>
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/i18n"
)

// eventFiles 是引擎會載入的五個事件表。
var eventFiles = []string{"DATA1.TXT", "DATA2.TXT", "DATA3.TXT", "DATA4.TXT", "DATA5.TXT"}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "events":
		dumpEvents(os.Args[2:])
	case "ui":
		dumpUI(os.Args[2:])
	case "spells":
		dumpSpells(os.Args[2:])
	case "items":
		dumpItems(os.Args[2:])
	case "monsters":
		dumpMonsters(os.Args[2:])
	case "towns":
		dumpTowns(os.Args[2:])
	case "check":
		check(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "用法：dwstrings events|spells|items|monsters|towns|ui|check [選項]")
	os.Exit(2)
}

func dumpEvents(args []string) {
	fs := flag.NewFlagSet("events", flag.ExitOnError)
	dataDir := fs.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	outDir := fs.String("out", "assets/lang/zh-Hant", "翻譯目錄輸出位置")
	_ = fs.Parse(args)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	total := 0
	for _, name := range eventFiles {
		tbl, err := scenario.LoadEventTable(filepath.Join(*dataDir, name))
		if err != nil {
			fatal(fmt.Errorf("載入 %s：%w", name, err))
		}

		cat := &i18n.Catalog{Source: name}
		for i, ev := range tbl.All() {
			if strings.TrimSpace(ev.Text) == "" {
				continue
			}
			cat.Entries = append(cat.Entries, i18n.Entry{Index: i, Source: ev.Text})
		}

		out := filepath.Join(*outDir, i18n.CatalogFileName(name))
		// 已有翻譯就保留，只補新條目、標出原文變動的條目。
		merged, added, drifted := mergeInto(out, cat)
		if err := i18n.WriteCatalog(out, merged); err != nil {
			fatal(err)
		}
		fmt.Printf("%-10s %3d 筆（新增 %d、原文變動 %d）→ %s\n",
			name, len(merged.Entries), added, drifted, out)
		total += len(merged.Entries)
	}
	fmt.Printf("合計 %d 筆事件敘述\n", total)
}

// mergeInto 把新抽出的條目併進既有翻譯檔。
//
// **既有譯文不覆蓋。** 原文變動時保留譯文並標記為待複查 ——
// 直接丟掉譯文會讓人白翻一次，直接沿用又會讓譯文與原文悄悄脫節。
func mergeInto(path string, fresh *i18n.Catalog) (merged *i18n.Catalog, added, drifted int) {
	old, err := i18n.LoadCatalog(path)
	if err != nil {
		// 檔案還不存在是正常的第一次抽取。
		return fresh, len(fresh.Entries), 0
	}

	prev := map[int]i18n.Entry{}
	for _, e := range old.Entries {
		prev[e.Index] = e
	}

	out := &i18n.Catalog{Source: fresh.Source}
	for _, e := range fresh.Entries {
		p, ok := prev[e.Index]
		switch {
		case !ok:
			added++
		case p.Source != e.Source:
			e.Target = p.Target
			e.NeedsReview = true
			drifted++
		default:
			e.Target = p.Target
			e.NeedsReview = p.NeedsReview
		}
		out.Entries = append(out.Entries, e)
	}
	return out, added, drifted
}

// spellSource 是法術名稱的來源檔，同時也是翻譯目錄的 key。
const spellSource = "FILES.DTT"

// dumpSpells 把 43 個法術名稱抽成翻譯目錄。
//
// 名稱與 FILES.DAT 法術表同序，索引就是法術 id —— 與事件敘述用同一套
// （來源檔, 索引）機制，不另立格式。
func dumpSpells(args []string) {
	fs := flag.NewFlagSet("spells", flag.ExitOnError)
	dataDir := fs.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	outDir := fs.String("out", "assets/lang/zh-Hant", "翻譯目錄輸出位置")
	_ = fs.Parse(args)

	names, err := spellNames(*dataDir)
	if err != nil {
		fatal(err)
	}
	cat := &i18n.Catalog{Source: spellSource}
	for i, n := range names {
		cat.Entries = append(cat.Entries, i18n.Entry{Index: i, Source: n})
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	out := filepath.Join(*outDir, i18n.CatalogFileName(spellSource))
	merged, added, drifted := mergeInto(out, cat)
	if err := i18n.WriteCatalog(out, merged); err != nil {
		fatal(err)
	}
	fmt.Printf("%-10s %3d 個法術名（新增 %d、原文變動 %d）→ %s\n",
		spellSource, len(merged.Entries), added, drifted, out)
}

// itemSource 是道具名稱的來源檔，同時也是翻譯目錄的 key。
const itemSource = "ITEMS.DAT"

// dumpItems 把 30 個道具名稱抽成翻譯目錄。
//
// 索引就是 ITEMS.DAT 的記錄索引，也就是道具槽 `+0x00` 存的值 ——
// 與事件敘述、法術名稱共用同一套（來源檔, 索引）機制。
func dumpItems(args []string) {
	fs := flag.NewFlagSet("items", flag.ExitOnError)
	dataDir := fs.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	outDir := fs.String("out", "assets/lang/zh-Hant", "翻譯目錄輸出位置")
	_ = fs.Parse(args)

	tbl, err := gamedata.LoadItemTable(filepath.Join(*dataDir, itemSource))
	if err != nil {
		fatal(err)
	}
	cat := &i18n.Catalog{Source: itemSource}
	for i, it := range tbl.All() {
		cat.Entries = append(cat.Entries, i18n.Entry{Index: i, Source: it.Name})
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	out := filepath.Join(*outDir, i18n.CatalogFileName(itemSource))
	merged, added, drifted := mergeInto(out, cat)
	if err := i18n.WriteCatalog(out, merged); err != nil {
		fatal(err)
	}
	fmt.Printf("%-10s %3d 個道具名（新增 %d、原文變動 %d）→ %s\n",
		itemSource, len(merged.Entries), added, drifted, out)
}

// monsterSource 是怪物名稱的來源檔，同時也是翻譯目錄的 key。
const monsterSource = "MONSTER.DAT"

// dumpMonsters 把 99 個怪物名稱抽成翻譯目錄。
//
// 索引就是 MONSTER.DAT 的記錄索引，也是遭遇表與事件表引用怪物的方式。
func dumpMonsters(args []string) {
	fs := flag.NewFlagSet("monsters", flag.ExitOnError)
	dataDir := fs.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	outDir := fs.String("out", "assets/lang/zh-Hant", "翻譯目錄輸出位置")
	_ = fs.Parse(args)

	tbl, err := gamedata.LoadMonsterTable(filepath.Join(*dataDir, monsterSource))
	if err != nil {
		fatal(err)
	}
	cat := &i18n.Catalog{Source: monsterSource}
	for i, m := range tbl.All() {
		cat.Entries = append(cat.Entries, i18n.Entry{Index: i, Source: m.Name})
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	out := filepath.Join(*outDir, i18n.CatalogFileName(monsterSource))
	merged, added, drifted := mergeInto(out, cat)
	if err := i18n.WriteCatalog(out, merged); err != nil {
		fatal(err)
	}
	fmt.Printf("%-11s %3d 個怪物名（新增 %d、原文變動 %d）→ %s\n",
		monsterSource, len(merged.Entries), added, drifted, out)
}

// townSource 是城鎮名稱的來源檔，同時也是翻譯目錄的 key。
const townSource = "TOWN.TXT"

// dumpTowns 把 25 個城鎮名稱抽成翻譯目錄。
//
// 索引是**城鎮編號減 1**（`TOWN.TXT` 的第 n 個字串對應 `TOWN{n}.DAT`）。
func dumpTowns(args []string) {
	fs := flag.NewFlagSet("towns", flag.ExitOnError)
	dataDir := fs.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	outDir := fs.String("out", "assets/lang/zh-Hant", "翻譯目錄輸出位置")
	_ = fs.Parse(args)

	tbl, err := gamedata.LoadTownTable(*dataDir)
	if err != nil {
		fatal(err)
	}
	cat := &i18n.Catalog{Source: townSource}
	for i := 1; i <= gamedata.NumTowns; i++ {
		town, err := tbl.ByNumber(i)
		if err != nil {
			fatal(err)
		}
		cat.Entries = append(cat.Entries, i18n.Entry{Index: i - 1, Source: town.Name})
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	out := filepath.Join(*outDir, i18n.CatalogFileName(townSource))
	merged, added, drifted := mergeInto(out, cat)
	if err := i18n.WriteCatalog(out, merged); err != nil {
		fatal(err)
	}
	fmt.Printf("%-10s %3d 個城鎮名（新增 %d、原文變動 %d）→ %s\n",
		townSource, len(merged.Entries), added, drifted, out)
}

// spellNames 讀出 43 個法術的英文名稱。
func spellNames(dataDir string) ([]string, error) {
	pool, err := gamedata.LoadStringPool(filepath.Join(dataDir, spellSource))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, gamedata.NumSpellRecords)
	for i := 0; i < gamedata.NumSpellRecords; i++ {
		n, err := pool.SpellName(i)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func dumpUI(args []string) {
	fs := flag.NewFlagSet("ui", flag.ExitOnError)
	intPath := fs.String("int", "workplace/orig/demwin/DEMON.INT", "原版 DEMON.INT")
	out := fs.String("out", "docs/i18n/demon-int-strings.md", "清單輸出檔")
	minLen := fs.Int("min", 4, "最短字串長度")
	_ = fs.Parse(args)

	data, err := os.ReadFile(*intPath)
	if err != nil {
		fatal(err)
	}
	found := scanStrings(data, *minLen)

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fatal(err)
	}
	var b strings.Builder
	b.WriteString("# `DEMON.INT` UI 字串清單\n\n")
	b.WriteString("> 由 `cmd/dwstrings ui` 產生，**不要手改**。\n>\n")
	b.WriteString("> 本專案是重製而不是改機碼，引擎**不讀** `DEMON.INT`。\n")
	b.WriteString("> 這份清單的用途是確認 UI 該有哪些文案，不是執行期資源。\n")
	b.WriteString("> 執行期的翻譯資源只有 `assets/lang/`（事件敘述）。\n\n")
	fmt.Fprintf(&b, "字串數：%d　最短長度：%d\n\n", len(found), *minLen)
	b.WriteString("| 檔案位移 | 字串 |\n|---|---|\n")
	for _, s := range found {
		fmt.Fprintf(&b, "| `%05x` | `%s` |\n", s.off, strings.ReplaceAll(s.text, "|", "\\|"))
	}
	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("%d 條字串 → %s\n", len(found), *out)
}

type foundString struct {
	off  int
	text string
}

// scanStrings 找出 NUL 結尾的可列印字串。
//
// 只收「字母與空白佔七成以上」的候選 —— 16 位元機器碼裡到處是
// 湊巧可列印的位元組，不篩就會撈進大量雜訊。
func scanStrings(data []byte, minLen int) []foundString {
	var out []foundString
	start := -1
	for i := 0; i <= len(data); i++ {
		printable := i < len(data) && data[i] >= 0x20 && data[i] < 0x7f
		if printable {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			if s := string(data[start:i]); len(s) >= minLen && looksLikeText(s) {
				out = append(out, foundString{off: start, text: s})
			}
			start = -1
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].off < out[j].off })
	return out
}

func looksLikeText(s string) bool {
	n := 0
	for _, c := range s {
		if c == ' ' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			n++
		}
	}
	return float64(n)/float64(len(s)) > 0.75
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "dwstrings:", err)
	os.Exit(1)
}
