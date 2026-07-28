package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/wicanr2/demon_winter_cht/internal/i18n"
)

// `dwstrings uicheck` —— 介面文案的遷移狀態檢查
//
// **為什麼需要這一支。** `ui.txt` 是唯一沒有自動檢查的翻譯檔：事件目錄
// 有 `dwstrings check` 拿原文當 key 比對，介面文案的 key 是自己取的名字，
// **打錯一個字的後果是靜默退回 fallback** —— 畫面看起來一模一樣，
// 只是那一條永遠不走目錄。六百多條要遷的時候，這種錯會累積到沒人查得完。
//
// 檢查四件事：
//
//	1. 程式碼裡每一個 `tr.UI(key, fallback)` 的 key 都在 `ui.txt` 裡
//	2. `ui.txt` 裡沒有孤兒 key（程式碼已經不用了）
//	3. 目錄裡的譯文與程式碼裡的 fallback 一致（不一致代表有人只改了一邊）
//	4. `cmd/demonwinter/` 還剩幾條硬編的中文字面值（＝ D1 的進度）
//
// 第 3 項是刻意的：本專案的 fallback 一律是中文，兩邊本來就該一樣。
// 允許它們漂開等於允許「畫面上的字」有兩個真相來源。

// uiSkipCallees 是「這一層呼叫裡的中文字串不算介面文案」的函式。
// 偵錯輸出、旗標說明、錯誤訊息都不上畫面。
var uiSkipCallees = map[string]bool{
	"log.Printf": true, "log.Println": true, "log.Fatalf": true, "log.Fatal": true,
	"fmt.Errorf": true, "errors.New": true, "panic": true,
	"flag.String": true, "flag.Int": true, "flag.Bool": true, "flag.Float64": true,
}

type uiCall struct {
	key, fallback string
	pos           token.Position
}

type uiHardcoded struct {
	text string
	pos  token.Position
}

// uiKeyStem 認得「看起來像 key 或 key 前綴」的字面值。
//
// **有些 key 是算出來的**，例如 `fmt.Sprintf("trap.name%d", c)` 與
// `riddle.priest` ＋ `".0"`。靜態掃描看不到組出來的完整字串，
// 所以改成收「字面值裡的 key 前綴」，再讓目錄裡以該前綴開頭的 key
// 算作有呼叫端。寧可漏報孤兒，也不要對著一個正常用著的 key 亮紅燈。
var uiKeyStem = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z0-9_%+-]*)+$`)

func uiCheck(args []string) {
	fs := flag.NewFlagSet("uicheck", flag.ExitOnError)
	srcDir := fs.String("src", "cmd/demonwinter", "要掃的原始碼目錄")
	langDir := fs.String("lang", "assets/lang/zh-Hant", "翻譯目錄")
	list := fs.Bool("list", false, "把還沒遷的字串逐條列出來（給遷移用）")
	fix := fs.Bool("fix", false, "把缺的 key 照程式碼的 fallback 補進 ui.txt")
	_ = fs.Parse(args)

	calls, hard, stems, err := scanUIStrings(*srcDir)
	if err != nil {
		fatal(err)
	}

	cat, err := i18n.LoadCatalog(filepath.Join(*langDir, "ui.txt"))
	if err != nil {
		fatal(fmt.Errorf("載入 ui.txt：%w", err))
	}
	inCatalog := map[string]string{}
	for _, e := range cat.Entries {
		if e.Name == "" {
			continue
		}
		inCatalog[e.Name] = e.Target
	}

	var problems int
	report := func(format string, a ...any) {
		problems++
		fmt.Printf("  ✗ "+format+"\n", a...)
	}

	if *fix {
		if err := uiAppendMissing(filepath.Join(*langDir, "ui.txt"), calls, inCatalog); err != nil {
			fatal(err)
		}
		// 補完重讀一次，後面的檢查才看得到新條目。
		cat, err = i18n.LoadCatalog(filepath.Join(*langDir, "ui.txt"))
		if err != nil {
			fatal(err)
		}
		inCatalog = map[string]string{}
		for _, e := range cat.Entries {
			if e.Name != "" {
				inCatalog[e.Name] = e.Target
			}
		}
	}

	used := map[string]bool{}
	for _, c := range calls {
		used[c.key] = true
		target, ok := inCatalog[c.key]
		switch {
		case !ok:
			report("%s:%d 的 key %q 不在 ui.txt 裡 —— 會靜默退回 fallback",
				filepath.Base(c.pos.Filename), c.pos.Line, c.key)
		case target != c.fallback:
			report("%s:%d 的 key %q：目錄譯文與程式碼 fallback 不一致\n"+
				"      目錄 %q\n      程式 %q",
				filepath.Base(c.pos.Filename), c.pos.Line, c.key, target, c.fallback)
		}
	}
	dynamic := 0
	var orphans []string
	for name := range inCatalog {
		if used[name] {
			continue
		}
		// 算出來的 key：目錄裡以某個字面值前綴開頭就算有呼叫端。
		covered := false
		for _, st := range stems {
			if st != "" && strings.HasPrefix(name, st) {
				covered = true
				break
			}
		}
		if covered {
			dynamic++
			continue
		}
		orphans = append(orphans, name)
	}
	sort.Strings(orphans)
	for _, o := range orphans {
		report("ui.txt 的 key %q 沒有任何呼叫端 —— 孤兒條目", o)
	}

	fmt.Printf("介面文案：已遷 %d 條（其中 %d 條 key 是算出來的）／目錄 %d 條／還硬編 %d 條\n",
		len(used)+dynamic, dynamic, len(inCatalog), len(hard))

	if *list {
		byFile := map[string][]uiHardcoded{}
		for _, h := range hard {
			byFile[h.pos.Filename] = append(byFile[h.pos.Filename], h)
		}
		names := make([]string, 0, len(byFile))
		for n := range byFile {
			names = append(names, n)
		}
		sort.Slice(names, func(i, j int) bool {
			return len(byFile[names[i]]) > len(byFile[names[j]])
		})
		for _, n := range names {
			fmt.Printf("\n--- %s（%d）\n", n, len(byFile[n]))
			for _, h := range byFile[n] {
				fmt.Printf("%d\t%s\n", h.pos.Line, h.text)
			}
		}
	}

	if problems > 0 {
		fmt.Printf("\n%d 個問題\n", problems)
		os.Exit(1)
	}
	fmt.Println("介面文案檢查通過")
}

// scanUIStrings 掃一個目錄下所有 .go，回傳 `tr.UI` 呼叫與仍硬編的中文字串。
func scanUIStrings(dir string) ([]uiCall, []uiHardcoded, []string, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return nil, nil, nil, err
	}

	var calls []uiCall
	var hard []uiHardcoded
	stems := map[string]bool{}

	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			// 先收 UI 呼叫，並記下屬於它的字面值節點，之後掃硬編時跳過。
			consumed := map[ast.Node]bool{}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "UI" || len(call.Args) != 2 {
					return true
				}
				k, ok1 := stringLit(call.Args[0])
				f, ok2 := stringLit(call.Args[1])
				if !ok1 || !ok2 {
					return true
				}
				consumed[call.Args[0]] = true
				consumed[call.Args[1]] = true
				calls = append(calls, uiCall{k, f, fset.Position(call.Pos())})
				return true
			})

			// 再掃硬編。`skip` 是「目前在某個不上畫面的呼叫裡」的深度計數。
			var skip int
			var walk func(ast.Node) bool
			walk = func(n ast.Node) bool {
				if n == nil {
					return false
				}
				if call, ok := n.(*ast.CallExpr); ok && uiSkipCallees[calleeName(call.Fun)] {
					skip++
					for _, a := range call.Args {
						ast.Inspect(a, walk)
					}
					skip--
					return false
				}
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING || consumed[n] || skip > 0 {
					return true
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				if uiKeyStem.MatchString(s) {
					stems[strings.SplitN(s, "%", 2)[0]] = true
				}
				if !hasCJK(s) {
					return true
				}
				hard = append(hard, uiHardcoded{s, fset.Position(lit.Pos())})
				return true
			}
			ast.Inspect(file, walk)
		}
	}
	sort.Slice(calls, func(i, j int) bool { return lessPos(calls[i].pos, calls[j].pos) })
	sort.Slice(hard, func(i, j int) bool { return lessPos(hard[i].pos, hard[j].pos) })
	out := make([]string, 0, len(stems))
	for st := range stems {
		out = append(out, st)
	}
	sort.Strings(out)
	return calls, hard, out, nil
}

func lessPos(a, b token.Position) bool {
	if a.Filename != b.Filename {
		return a.Filename < b.Filename
	}
	return a.Line < b.Line
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// calleeName 把 `log.Printf` 這種呼叫還原成點記法，給 uiSkipCallees 查表。
func calleeName(e ast.Expr) string {
	switch f := e.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name + "." + f.Sel.Name
		}
		return f.Sel.Name
	}
	return ""
}

func hasCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}


// uiAppendMissing 把程式碼裡有、目錄裡沒有的 key 追加到 ui.txt。
//
// 譯文直接用程式碼裡的 fallback —— 本專案的 fallback 一律是中文，
// 它就是現在畫面上的字。`:: en` 留空：原版英文有對應的話由人補，
// 自動填一個猜的英文比留空更糟。
func uiAppendMissing(path string, calls []uiCall, inCatalog map[string]string) error {
	seen := map[string]bool{}
	var add []uiCall
	for _, c := range calls {
		if _, ok := inCatalog[c.key]; ok || seen[c.key] {
			continue
		}
		seen[c.key] = true
		add = append(add, c)
	}
	if len(add) == 0 {
		fmt.Println("沒有缺的 key")
		return nil
	}
	sort.Slice(add, func(i, j int) bool { return add[i].key < add[j].key })

	var b strings.Builder
	for _, c := range add {
		b.WriteString("\n## " + c.key + "\n:: zh\n" + c.fallback + "\n")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(b.String()); err != nil {
		return err
	}
	fmt.Printf("補了 %d 條進 %s\n", len(add), path)
	return nil
}
