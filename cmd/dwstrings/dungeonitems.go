package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/i18n"
)

// dungeonItemSource 是地城道具字串的翻譯目錄 key。
//
// **它不是檔名。** 這些字躺在 `FILES.DTT` 的第 164–463 條，但 `FILES.DTT`
// 這個 key 已經被法術名用掉了（`spellSource`）—— 與 `SKILLS`／`MONTHS`
// 同一個情況：字串同源、索引語意不同。
//
// 索引用 `FILES.DTT` 的**絕對條目編號**（`164 + i×6 + 欄`），
// 與 `cmd/demonwinter/dungeonui.go` 的 `dungeonSourceFile` 對齊。
const dungeonItemSource = "DUNGEONITEM"

// 六個欄位裡只有四種是給玩家讀的散文（`docs/re/95` §2）：
//
//	+0 名稱      50 條，全部要翻
//	+1 拿不走    25 條非空，其中 18 條只有一個 `*`
//	+2 檢視      48 條非空
//	+3 推開後    16 個 `*` ＋ 3 組座標數字（`461100`）
//	+4 要配      18 條，內容是**另一件道具的名字**
//	+5 用對後    18 條，首字元是動作碼
//
// **`+3` 與 `+4` 一條都不能翻。**
// `+3` 是座標，`+4` 是查表鍵（`DungeonItems.ByName` 拿它換索引，`U` 的比對
// 與 `X` 鑑物都靠它）。翻了會讓整條解謎鏈對不上，而且畫面上看不出異狀 ——
// 只是「用對的東西也沒反應」。`X` 鑑物顯示這一欄時走的是
// `dungeonName()`，那邊會用**名稱目錄**翻，所以玩家看到的仍是中文。
//
// `*` 同理不翻：它是「拿不走但沒有台詞」的佔位，引擎看到它會改印
// `tr.UI("dungeon.cant")`，翻譯它只會讓那個判斷失效。
//
// `+5` 只有 `D`（印一段敘述）那幾條是散文；`N` 的參數是道具名，
// `T`／`P`／`S` 的參數是座標與劇情編號。**抽出來的原文已經去掉開頭的 `D`** ——
// 那是動作碼不是內文，讓譯者看到它只會多一種翻壞的方式。
const dungeonItemDescribe = byte(gamedata.ActionDescribe)

// dungeonItemEntries 是「哪些條目要翻、原文是什麼」的**單一實作**。
// 抽字（dumpDungeonItems）與品質閘（check）都走這一支，
// 兩邊各寫一份的話遲早會漂掉。
func dungeonItemEntries(dataDir string) ([]i18n.Entry, error) {
	pool, err := gamedata.LoadStringPool(filepath.Join(dataDir, spellSource))
	if err != nil {
		return nil, err
	}
	items, err := gamedata.LoadDungeonItems(pool)
	if err != nil {
		return nil, err
	}

	var out []i18n.Entry
	add := func(base, field int, s string) {
		out = append(out, i18n.Entry{Index: base + field, Source: s})
	}
	for i, it := range items {
		base := gamedata.DungeonItemFirstString + i*gamedata.DungeonItemFields
		add(base, 0, it.Name)
		if it.Immovable != "" && it.Immovable != "*" {
			add(base, 1, it.Immovable)
		}
		if it.Look != "" {
			add(base, 2, it.Look)
		}
		if it.Action() == gamedata.ActionDescribe {
			add(base, 5, it.ActionParam())
		}
	}
	return out, nil
}

// dumpDungeonItems 把地城道具的可翻譯字串抽成翻譯目錄。
func dumpDungeonItems(args []string) {
	fs := flag.NewFlagSet("dungeonitems", flag.ExitOnError)
	dataDir := fs.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	outDir := fs.String("out", "assets/lang/zh-Hant", "翻譯目錄輸出位置")
	_ = fs.Parse(args)

	entries, err := dungeonItemEntries(*dataDir)
	if err != nil {
		fatal(err)
	}
	cat := &i18n.Catalog{Source: dungeonItemSource, Entries: entries}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	out := filepath.Join(*outDir, i18n.CatalogFileName(dungeonItemSource))
	merged, added, drifted := mergeInto(out, cat)
	if err := i18n.WriteCatalog(out, merged); err != nil {
		fatal(err)
	}
	fmt.Printf("%-11s %3d 條地城道具字串（新增 %d、原文變動 %d）→ %s\n",
		dungeonItemSource, len(merged.Entries), added, drifted, out)
}
