package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/i18n"
)

// storyModes 是三個分頁劇情文字檔（`docs/re/82`）。
var storyModes = []struct {
	mode scenario.StoryMode
	key  string
}{
	{scenario.StoryDream, "T"},
	{scenario.StoryWin, "WIN"},
	{scenario.StoryEregore, "EREGORE"},
}

// dumpStory 把三場夢、結局、艾瑞戈爾那三個檔抽成翻譯目錄。
//
// **一頁一條，不是一行一條。** 原版的行是照它 40 欄畫面斷的，
// 中文的斷行位置本來就不一樣（而且中文一行放得下的字數也不同），
// 逐行對譯會逼譯者遷就英文的斷點，譯出來會很怪。
// 抽成整頁之後，畫面那邊自己重新斷行。
func dumpStory(args []string) {
	fs := flag.NewFlagSet("story", flag.ExitOnError)
	dataDir := fs.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	outDir := fs.String("out", "assets/lang/zh-Hant", "翻譯目錄輸出位置")
	_ = fs.Parse(args)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	cat := &i18n.Catalog{Source: storyCatalogSource}
	for _, m := range storyModes {
		st, err := scenario.LoadStoryText(*dataDir, m.mode)
		if err != nil {
			fatal(fmt.Errorf("載入 %s：%w", m.key, err))
		}
		for i := range st.Pages {
			text := storyPageText(st.Pages[i])
			if strings.TrimSpace(text) == "" {
				continue
			}
			cat.Entries = append(cat.Entries, i18n.Entry{
				Index:  -1,
				Name:   storyKey(m.key, i),
				Source: text,
			})
		}
	}

	out := filepath.Join(*outDir, i18n.CatalogFileName(storyCatalogSource))
	merged, added, drifted := mergeInto(out, cat)
	if err := i18n.WriteCatalog(out, merged); err != nil {
		fatal(err)
	}
	fmt.Printf("%-10s %3d 頁（新增 %d、原文變動 %d）→ %s\n",
		storyCatalogSource, len(merged.Entries), added, drifted, out)
}

// storyCatalogSource 是劇情文字翻譯目錄的來源名。
const storyCatalogSource = "STORY"

// storyKey 是一頁的翻譯 key。
func storyKey(file string, page int) string {
	return fmt.Sprintf("story.%s.%d", file, page)
}

// storyPageText 把一頁的行併成一段，行間用換行。
//
// 原版的縮排是它的排版（那首預言靠縮排分句），所以**保留** ——
// 譯者看得到原文的層次，中文也可以照著縮。
func storyPageText(lines []string) string {
	return strings.Join(lines, "\n")
}
