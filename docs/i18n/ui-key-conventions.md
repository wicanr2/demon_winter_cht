# 介面文案的 key 命名規約

`assets/lang/zh-Hant/ui.txt` 的 key 是**語意化名稱不是數字索引**：介面文案會
增刪，數字一插入就要重編號，那種痛苦會讓人乾脆不維護翻譯檔。

規約寫在這裡是因為 D1 有六百多條要遷，而 key 是**一次取好就不該再改**的東西
（改 key 等於把譯文孤兒化）。

## 形式

```
<區塊>.<群組>.<內容>
```

全小寫、ASCII、用 `.` 分層、用 `_` 連接複合詞（少用）。層數 2 或 3，不要更深。

- **區塊**跟著畫面走，通常等於檔名去掉 `ui`／`.go`：
  `battle`／`town`／`camp`／`create`／`merchant`／`dungeon`／`trap`／`pool`／
  `proving`／`workshop`／`plotgift`／`manual`／`save`／`title`／`dream`／
  `ending`／`death`／`riddle`／`rune`／`bell`／`tombstone`／`status`。
- **群組**是同一畫面裡的功能分區。`battle` 用了：
  `cmd`（指令列）、`msg`（戰鬥訊息）、`cast`（施法選單）、`sp`（投入法力）、
  `item`（道具選單）、`aim`（選目標）、`field`（戰場標記）、`header`（抬頭）、
  `unit`（單位表）、`drop`（戰利品）、`hit`（命中／重擊）、`log`（紀錄）。
  只有幾條的畫面可以省略群組（`pool.title`、`title.press`）。
- **內容**用英文小寫描述那句話在做什麼，不是翻譯它。
  `nosp`（法力不夠）、`noap`（行動點不夠）、`keys`（操作提示）、
  `note`（※ 開頭的補充說明）、`header`（標題列）、`columns`（欄位名）、
  `who`（問誰）、`which`（問哪一件）、`cancelled`、`done`、`empty`。

## 硬規則

1. **同一句話只給一個 key。** 重複出現的文字（`%s 行動點不足` 在 `battleui.go`
   出現七次）共用一個 key。判準：**改了譯文，那幾處都該一起變**才共用；
   語境不同就分開（`%s 正前方沒有目標` 與 `%s 正前方沒有敵人` 是兩條）。
2. **格式動詞原封不動。** `%s`／`%d`／`%3d`／`%-2s` 的**個數與順序**必須與原字串
   一致 —— 這些字串會進 `fmt.Sprintf`，順序一換參數就對錯位置。
3. **開頭或結尾的空白（含全形空白 `　`）是版面的一部分**，不要修掉。
   `　%s 倒下了` 與 `%s 倒下了` 是**兩條**，因為縮排代表它在清單裡。
4. **key 裡不要放中文，也不要放行號、檔名以外的位置資訊。**
5. **只給 key，不要改譯文。** 這一輪是「把字串抽到目錄」，不是重譯。
   譯文有問題另外提，不要順手改 —— 那會讓 `uicheck` 的
   「目錄譯文＝程式碼 fallback」比對變成兩件事一起動。
6. 譯文必須是**繁體中文且 Big5 打得出來**（倚天字型只有 Big5 的字，
   打不出來的字畫面上是豆腐）。`dwstrings uicheck` 會擋。

## 兩種寫法

**一般情況**（函式裡有 `a *app`）：

```go
a.message = a.tr.UI("read.nothing", "這裡沒有值得再看一次的東西")
c.message = fmt.Sprintf(a.tr.UI("camp.hunt.gained", "打到 %d 份糧食（共 %d 份）"), n, total)
```

**套件層的表**（init 時還沒有 translator）：key 與中文並列，翻譯發生在畫的時候。

```go
var playerCommands = []struct {
	key    ebiten.Key
	uikey  string
	label  string
	action game.Action
}{
	{ebiten.KeyA, "battle.cmd.attack", "A 攻擊", game.ActionAttack},
	...
}
// 畫的時候：
label := a.tr.UI(c.uikey, c.label)
```

`dwstrings uicheck` 認得這個形式（同一行有 key 形狀的字面值就當成已配好）。

## 不算介面文案的

這些**不要**遷，`uicheck` 也已經排除：

- `log.Printf`／`log.Fatalf`／`fmt.Errorf`／`errors.New`／`panic` 裡的字 —— 不上畫面。
- `flag.String`／`flag.Int` 的說明 —— 命令列說明，開發用。
- `a.trace.note`／`a.trace.state` 與整個 `trace.go` —— 那是**軌跡檔**不是畫面，
  而且驗收會拿前後兩份 trace 比對，跟著介面語言漂就毀了那個訊號。
- 查表用的 key、座標參數、比較用的字面值。

## 檢查

```
tools/go.sh run ./cmd/dwstrings uicheck        # 四項檢查 ＋ 進度計數
tools/go.sh run ./cmd/dwstrings uicheck -list  # 還沒遷的逐條列出（行號 + 文字）
tools/go.sh run ./cmd/dwstrings uicheck -fix   # 缺的 key 照程式碼 fallback 補進目錄
```

檢查的四件事：key 都在目錄裡、目錄沒有孤兒、目錄譯文與程式碼 fallback 一致、
還剩幾條硬編。**打錯一個 key 的後果是靜默退回 fallback** —— 畫面一模一樣，
只是那一條永遠不走目錄，所以這個檢查不是可選的。
