# 介面文案目錄（`assets/lang/<lang>/ui.txt`）

`CONTEXT.md` §7 的 C1：介面文案原本全部硬編在 `cmd/demonwinter/*.go` 裡。
這一份記錄機制與第一批遷移。

---

## 1. 為什麼需要語意化 key

既有的翻譯目錄（事件敘述、法術名、道具名、怪物名…）都用
**「來源檔名 + 數字索引」** 當 key，因為那些字串在原版資料裡有天然的序號 ——
用數字最不會對錯位。

介面文案不一樣：它們是**本專案自己寫的**，會增刪。
數字索引一插入就要重編號，那種痛苦會讓人乾脆不維護翻譯檔。

所以 `Entry` 加了 `Name`：`## ` 後面**不是數字時當語意化 key**。

```
## plot.uncurse
:: en
UNCURSE
:: zh
解咒
```

兩種目錄共用同一個解析器，`Atoi` 失敗不報錯而是走名稱路徑 ——
這是刻意的，不是寬鬆處理。

---

## 2. fallback 一律是中文

```go
a.tr.UI("plot.destroyed", "力量閃現，符印的魔法被摧毀了")
```

**第二個參數是中文，不是英文。** 理由：

這個專案的介面本來就是中文寫的，翻譯目錄的作用是
「把它抽出來好維護、日後好換語言」，**不是「英文轉中文」**。
所以缺一條的後果是「還是中文，只是沒走目錄」——
不像事件敘述那邊，缺譯會讓畫面變英文（那是刻意的，看得見）。

`:: en` 區塊放原版英文只為了**對照**（有對應的話），不參與查表。

---

## 3. 第一批：主線（12 條）

| key | 中文 | 原版英文 |
|---|---|---|
| `plot.uncurse` | 解咒 | `UNCURSE` |
| `plot.imprison` | 禁錮 | `IMPRISON` |
| `plot.noglyph` | 這裡沒有符印 | `I see no glyph!` |
| `plot.inactive` | 這個符印已經失效了 | `It is already inactive` |
| `plot.needsp` | 那需要 %d 點法力 | `That requires %d SP!` |
| `plot.destroyed` | 力量閃現，符印的魔法被摧毀了 | `In a flash of power…` |
| `plot.fizzles` | 法術消散了…… | `The spell fizzles...` |
| `plot.won` | 惡魔被禁錮了 | —（本專案自加）|
| `plot.forcefield` | 緋紅的力場擋住了通往光之環的路 | `A crimson forcefield…` |
| `plot.glyphdrain` | 符印的力量侵蝕著隊伍 | —（原版是逐點扣血無訊息）|
| `plot.fell` | %s 倒下了 | — |
| `plot.allfell` | 全隊都倒下了 | — |

挑主線當第一批是因為它剛做完（`docs/re/59`–`64`），
原文對照都還在手上 —— 那八條有原版英文的，是這一輪反組譯讀出來的。

實機驗過走目錄之後破關流程不變（`83-ui-catalog.png`）。

---

## 4. 這不是「翻譯那 840 條」

`cmd/dwstrings ui` 抽出的 840 條是 **`DEMON.INT` 的參考清單**：

- 引擎**不讀** `DEMON.INT`（本專案是重製不是改機碼）
- 那份清單含大量把機器碼誤判成字串的雜訊（`PHHH`、`ZXPR` 這種）

它的用途是「確認 UI 該有哪些文案」，不是執行期資源。
C1 的工作內容是**把 Go 裡的中文字面抽進目錄**，兩者常被混為一談。

---

## 5. 剩下的

`cmd/demonwinter/` 裡含中文的行數（`grep -c`）：

| 檔案 | 行數 |
|---|---|
| `main.go` | ~294 |
| `battleui.go` | ~278 |
| `townui.go` | ~114 |
| `townservices.go` | ~93 |
| 其餘 | ~200 |

不是每一行都是待遷移的字串（註解也算進去了），但量級在數百條。
機制有了之後可以分批遷移，**優先度仍然最低** ——
畫面已經是中文，這一項改善的是可維護性與日後換語言，
依 `rules/10` 的優先序（正確性 > 可落地 > 時程 > 可維護性）排在後面。

---

## 6. 原版選單的逐字原文（對照用）

`docs/re/81` 用 DOSBox 實跑把兩張主要選單截了下來。
**這是 C1 的原文對照表** —— 在這之前只能從反組譯猜項目名稱。

### 主指令選單（`94-orig-main-menu.png`）

```
Walk          Party info    Save Game     Camp          Look
Take          Drop          Move          Examine       Use
Inspect       View room     X)View item   Read descr…   Quit
```

`X)View item` 帶明示的鍵，因為 `V` 已經被 `View room` 佔走。

### 紮營選單（`95-orig-camp-menu.png`）

```
Party    Reorder   Sleep    Identify   Worship   Xorcise   View land
Trade    Drop      Equip    Use        Hunt      Cast      Quit
```

**原版把 Exorcise 拼成 `Xorcise`** —— `E` 被 `Equip` 佔了，所以連拼字都改。
手冊（`docs/manual-cht` p.28–31）列的是 `X` Exorcise，**與畫面不一致，畫面為準**。

### 兩點提醒

1. **首字母撞了就改字**是這款遊戲的慣例（`Xorcise`、`X)View item`）。
   中文化時這個約束消失了 —— 中文選項用不著首字母當熱鍵，
   熱鍵可以直接沿用原版字母，標在中文前面（本專案就是這樣做的）。
2. 本專案紮營選單的 14 項與原版一致，**熱鍵也全部對得上**，
   只有畫面上的排列順序不同（本專案按行分組，原版是一直排）。
   要不要對齊順序是呈現層的決定，還沒做。
