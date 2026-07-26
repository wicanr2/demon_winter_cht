package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/cjk"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/i18n"
)

// check 是翻譯批次的最後防線。
//
// 多個 subagent 分頭翻譯時，壞掉的方式很固定：改到原文（查表 key 失效）、
// 吞掉 %s、寫進 Big5 打不出來的字、譯名各自發揮。
// 這幾件事肉眼很難逐條盯，全部在這裡自動擋下來。
func check(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	dataDir := fs.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	langDir := fs.String("lang", "assets/lang/zh-Hant", "翻譯目錄")
	glossPath := fs.String("glossary", "translations/glossary.md", "統一譯名表")
	_ = fs.Parse(args)

	terms, err := loadGlossary(*glossPath)
	if err != nil {
		fatal(fmt.Errorf("載入譯名表：%w", err))
	}

	var problems int
	report := func(format string, a ...any) {
		problems++
		fmt.Printf("  ✗ "+format+"\n", a...)
	}

	totalDone, totalAll := 0, 0

	for _, name := range eventFiles {
		path := filepath.Join(*langDir, i18n.CatalogFileName(name))
		cat, err := i18n.LoadCatalog(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fatal(err)
		}

		tbl, err := scenario.LoadEventTable(filepath.Join(*dataDir, name))
		if err != nil {
			fatal(err)
		}
		events := tbl.All()

		done, review, todo := cat.Stats()
		totalDone += done
		totalAll += len(cat.Entries)
		fmt.Printf("%s：%d 已翻 / %d 待複查 / %d 未翻（共 %d）\n",
			name, done, review, todo, len(cat.Entries))

		for _, e := range cat.Entries {
			// 1. 原文必須與現在的資料一致 —— 這是查表 key。
			switch {
			case e.Index < 0 || e.Index >= len(events):
				report("%s #%d：索引超出事件表範圍（共 %d 筆）", name, e.Index, len(events))
				continue
			case normalise(e.Source) != normalise(events[e.Index].Text):
				report("%s #%d：原文與 %s 對不上，譯文會整條退回英文", name, e.Index, name)
				continue
			}
			if e.Target == "" {
				continue
			}

			// 2. 控制序列的數量與順序。
			if a, b := verbs(e.Source), verbs(e.Target); !equalSlices(a, b) {
				report("%s #%d：控制序列不符，原文 %v、譯文 %v", name, e.Index, a, b)
			}

			// 3. Big5 以外的字沒有字模，畫面上會變成空白。
			if bad := nonBig5(e.Target); len(bad) > 0 {
				report("%s #%d：以下字元不在 Big5，畫面上會缺字：%s",
					name, e.Index, string(bad))
			}

			// 4. 長度失控。一頁 5 行 × 38 格，超太多就得多翻一頁。
			if w, ok := tooLong(e.Source, e.Target); !ok {
				report("%s #%d：譯文 %d 格、原文 %d 格，超過 +40%%",
					name, e.Index, w, len(e.Source))
			}

			// 5. 譯名漂移。原文出現的專有名詞，譯文裡必須是譯名表指定的那個譯法。
			for _, term := range terms {
				if !wordBoundary(term.en).MatchString(e.Source) {
					continue
				}
				if strings.Contains(e.Target, term.zh) {
					continue
				}
				// 譯名表也收複合詞（`White Knights` 與 `The Vault of the
				// White Knights` 各有一條）。複合詞命中時，短詞就別再要求 ——
				// 那不是漂移，是更精確的譯法。
				if coveredByCompound(term, terms, e.Source, e.Target) {
					continue
				}
				report("%s #%d：原文有 %q，譯文卻找不到指定譯名 %q",
					name, e.Index, term.en, term.zh)
			}
		}
	}

	// 法術名稱走同一套（來源檔, 索引）機制，一併驗。
	if names, err := spellNames(*dataDir); err == nil {
		path := filepath.Join(*langDir, i18n.CatalogFileName(spellSource))
		if cat, err := i18n.LoadCatalog(path); err == nil {
			done, review, todo := cat.Stats()
			totalDone += done
			totalAll += len(cat.Entries)
			fmt.Printf("%s：%d 已翻 / %d 待複查 / %d 未翻（共 %d）\n",
				spellSource, done, review, todo, len(cat.Entries))

			for _, e := range cat.Entries {
				switch {
				case e.Index < 0 || e.Index >= len(names):
					report("%s #%d：索引超出法術表範圍", spellSource, e.Index)
				case e.Source != names[e.Index]:
					report("%s #%d：原文與 %s 對不上", spellSource, e.Index, spellSource)
				case e.Target != "":
					if bad := nonBig5(e.Target); len(bad) > 0 {
						report("%s #%d：以下字元不在 Big5：%s",
							spellSource, e.Index, string(bad))
					}
				}
			}
		}
	}

	// 其餘的名稱目錄（道具／怪物／城鎮／月份／技能）走同一套機制。
	//
	// 這幾個原本不在檢查範圍內 —— 於是 `check` 一直回報「148/148 100%」，
	// 而實際上多了兩百多條名稱沒被算進去。**進度數字漏算比沒有數字更糟**，
	// 它會讓人以為翻完了。這些目錄沒有對應的原始資料可以逐條比對原文
	// （名稱來源各自不同），所以只驗 Big5 與完成度。
	for _, src := range nameCatalogs {
		path := filepath.Join(*langDir, i18n.CatalogFileName(src))
		cat, err := i18n.LoadCatalog(path)
		if err != nil {
			continue
		}
		done, review, todo := cat.Stats()
		totalDone += done
		totalAll += len(cat.Entries)
		fmt.Printf("%-11s：%d 已翻 / %d 待複查 / %d 未翻（共 %d）\n",
			src, done, review, todo, len(cat.Entries))
		for _, e := range cat.Entries {
			if e.Target == "" {
				continue
			}
			if bad := nonBig5(e.Target); len(bad) > 0 {
				report("%s #%d：以下字元不在 Big5，畫面上會缺字：%s",
					src, e.Index, string(bad))
			}
		}
	}

	if totalAll > 0 {
		fmt.Printf("\n進度：%d/%d（%.0f%%）\n",
			totalDone, totalAll, 100*float64(totalDone)/float64(totalAll))
	}
	if problems > 0 {
		fmt.Printf("\n%d 個問題。\n", problems)
		os.Exit(1)
	}
	fmt.Println("\n檢查通過。")
}

// nameCatalogs 是「純名稱」的翻譯目錄，沒有控制序列也沒有長度限制，
// 只需要驗 Big5 與完成度。
var nameCatalogs = []string{itemSource, monsterSource, townSource, monthSource, skillSource}

// verbSeq 抓 printf 風格的控制序列。
var verbSeq = regexp.MustCompile(`%[-+ #0]*\d*(?:\.\d+)?[a-zA-Z%]`)

func verbs(s string) []string {
	out := verbSeq.FindAllString(s, -1)
	if out == nil {
		return []string{}
	}
	return out
}

func equalSlices(a, b []string) bool {
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

// nonBig5 回傳譯文裡 Big5 打不出來的字，去重後依序排列。
func nonBig5(s string) []rune {
	seen := map[rune]bool{}
	var out []rune
	for _, r := range s {
		if r < 0x80 || seen[r] {
			continue
		}
		if _, _, ok := cjk.Big5(r); !ok {
			seen[r] = true
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// tooLong 以顯示格數比對長度：一個中文字兩格、一個 ASCII 一格
// （原文以 8 像素等寬字估，譯文的中文字寬是它的兩倍）。
func tooLong(src, dst string) (int, bool) {
	w := 0
	for _, r := range dst {
		if r < 0x80 {
			w++
		} else {
			w += 2
		}
	}
	return w, float64(w) <= float64(len(src))*1.4
}

func normalise(s string) string { return strings.Join(strings.Fields(s), " ") }

// coveredByCompound 回報是否有一條「包含 term 的更長譯名」也在這一條命中，
// 且它的譯法確實出現在譯文裡。
func coveredByCompound(term glossaryTerm, all []glossaryTerm, src, dst string) bool {
	lower := strings.ToLower(term.en)
	for _, other := range all {
		if len(other.en) <= len(term.en) ||
			!strings.Contains(strings.ToLower(other.en), lower) {
			continue
		}
		if wordBoundary(other.en).MatchString(src) && strings.Contains(dst, other.zh) {
			return true
		}
	}
	return false
}
