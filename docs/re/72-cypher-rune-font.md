# 72 — `CYPHER.SHP` 是符文字型，`%` 是切字型的標記

> ## ⚠ 這一篇的前提是錯的
>
> 我寫這篇時以為「`%` → 符文字型」是未解的（依據是
> `docs/formats/event-script.md` §129 標著「**推測**」）。
>
> **那個機制早在 `docs/re/02` §2.4 就解出來並標「已驗證」了**，
> 而且比本篇詳細：`FUN_25be_18fa` 從 `param_1+1` 起逐字元轉換
> （`'.' → 0`、其他 `→ char − 0x40`），以**9 欄一列**的網格畫出。
> `internal/assets/scenario/events.go` 也早有 `IsRuneGlyph()`／`RuneText()`，
> 註解裡連函式位址都寫著（commit `84f683e`，2026-07-25）。
>
> **本篇真正新增的只有三件事**：
> 1. `CYPHER.SHP` 這個檔案就是那個字型（大小 1728 ＝ 27×64 佐證）
> 2. dump 出來肉眼確認是符文而非拉丁字母
> 3. §6 的中文化決定（符文不翻）
>
> 根因見 §7。

`docs/formats/event-script.md` §129 對事件文字裡的 `%` 控制字元寫著
「**推測** 符號用途……」—— **那一行是過期的**，見 §7。

---

## 1. `%` 開頭 → 載入 `CYPHER.SHP`

`docs/re/71` §3 找到 `CYPHER.SHP` 的載入點在 `25be:18fa`（`0x1b0da`）。
用 offset 反查法（`docs/re/61` §1）掃它的呼叫端，兩處：

```
0f578  push ds:0xcec / ds:0xcea      ; 字串 far ptr
       call 15be:18fa

1a85c  cmp BYTE PTR es:[bx], 0x25    ; 字串的第一個字元是 '%'？
1a860  jne 跳過
1a862  push es / bx                  ; 把那個字串傳進去
1a864  call 15be:18fa
```

**`0x25` 就是 `'%'`。** 判斷字串首字元是 `%` 才呼叫那支 ——
而那支的第一件事就是載入 `CYPHER.SHP`。

---

## 2. 檔案大小自己說話

```
CYPHER.SHP = 1728 bytes
1728 ÷ 64 = 27 frames        (16×16、64 B/frame，與 docs/re/07 的 .SHP 一致)
```

**27 個字形 ＝ 26 個字母 + 1。** 那個「+1」就是句點 ——
`event-script.md` 觀察到「句中空格被替換成句點 `.`」，
因為符文字型裡**有句點的字形，沒有空白的字形**。

## 3. 肉眼確認

`tools/cypherdump` → `workplace/dump/persist/82-cypher-font.png`

27 個 frame 排成 9×3，內容是**神祕符號**（不是拉丁字母）。
與檔名 `CYPHER`（密碼／暗號）對得上。

依本專案硬規則，視覺產物一律 dump 出來肉眼比對 ——
「1728 ÷ 64 = 27」再漂亮也只是算術，圖畫出來才算驗過。

---

## 4. 與 `docs/re/02` 的關係

`docs/re/02` §2.4 已經有的（**比本篇早，也比本篇精確**）：

```
FUN_25be_18fa：從 param_1+1 起，'.' → 0、其他 → char − 0x40，
               以 9 欄一列的網格座標呼叫 FUN_1d9f_1b3a 畫出
```

`char − 0x40` 讓 `'A'`(0x41) → 1、`'Z'`(0x5a) → 26、`'.'` → 0 ——
**glyph 0 是空白、1–26 是 A–Z，剛好 27 個**。

這與本篇的「1728 ÷ 64 ＝ 27」互相印證：檔案裡的 frame 數
正好等於字碼表需要的 glyph 數。本篇的貢獻是把**檔案**接上那個機制，
不是解開機制本身。

（順帶：我 dump 時用 9 列排版，碰巧與原版的 9 欄一致 —— 那是巧合。）

---

## 5. 三方閉合

| 證據 | 來源 |
|---|---|
| `%` 開頭才載入 `CYPHER.SHP` | 反組譯 `0x1a85c` |
| 27 個 16×16 字形 ＝ 26 字母 + 句點 | 檔案大小 |
| 內容是符文不是拉丁字母 | dump 出來的 PNG |
| 密語文字用句點取代空白 | `event-script.md` §129 的資料觀察 |

四條互相獨立，指向同一個結論。

---

## 6. 這修正了 `docs/re/65` 的猜測

`docs/re/65` §3 猜 `CYPHER.SHP`「與兩道密碼（`VOID`／`JESRIC`）有關」。

**方向對，但更精確**：它不是密碼輸入盤，是**顯示密語提示用的符文字型**。
玩家看到的 `%YMROS.IS...MINE`、`%RING.BELL...AT.....MIDNIGHT`、`%DIVINITY`
這些提示，在原版畫面上是用這 27 個符文畫出來的 —— 所以看起來像天書，
玩家得自己解讀。

`%RING.BELL...AT.....MIDNIGHT` 對得上 `docs/re/65` case 12 的鐘
（`Do you wish to ring the bell?` / `Nothing happens.` /
`The sound of angels crying opens up from the heavens…`）——
**提示與機關在兩張不同的表裡，這是它們第一次接上。**

---

## 7. 對中文化的決定：符文照原樣，不翻

查完攻略之後這一項不必猶豫了。`docs/walkthrough/part-4.md`：

> §55 走到房間西北角，移動那個老舊書櫃。裡面會出現**一連串符文**，
> 對應「YMROS IS MINE」（依姆羅斯是我的）。
> **如果你是不看攻略自己玩，這一點非常重要 —— 這些符文之後會再次出現，
> 用來解開其他謎題。**
>
> §141 在這裡到處走走，會看到好幾則**符文訊息**，分別對應
> 虛無（VOID）、力量（POWER）、靈魂（SPIRIT）、神性（DIVINITY），
> 其中「虛無」就是北邊幽靈祭司那道謎題的答案。

所以這不是「有些文字用花體字顯示」，是**一整套要玩家自己建立對照表的解謎機制**：

1. 圖書室給一組**已知答案**的符文（`YMROS IS MINE`）
2. 玩家由此推出符文 ↔ 字母的對照
3. 之後各處的符文訊息（`VOID`／`POWER`／`SPIRIT`／`DIVINITY`）就解讀得出來
4. 答案**要用英文輸入**（`VOID`、`JESRIC` 是明文存在資料段的字串）

**第 4 點決定了一切**：對照的目標語言必然是英文。
把符文換成中文，玩家就無從得知該輸入 `JESRIC` ——**整條鏈會斷**。

### 定案

| 項目 | 做法 |
|---|---|
| 符文訊息本身 | **照原版用 `CYPHER.SHP` 顯示，不翻** |
| 玩家要輸入的答案 | 保留英文（`VOID`／`JESRIC`）|
| 中文化的施力點 | **手冊與提示文字**用中文解釋「這是符文密語，需要自己對照」|

這與 `docs/walkthrough/part-4.md` 的翻譯策略一致 ——
它把 `YMROS IS MINE` 處理成「原文 + 中文註」，沒有把符文本身換掉。

> 這是「中文化不等於把每個字元換成中文」的一個具體案例。
> 符文是**謎題的載體**，翻掉它等於把謎題刪掉。

### 已實作

在這之前引擎**完全沒有顯示符文** —— `scenario.Event` 早有
`IsRuneGlyph()`／`RuneText()`（commit `84f683e`），但**沒有任何呼叫端**，
`checkEvent` 只顯示 `ev.Text`，`%` 開頭那四筆事件等於不存在。

補上的部分：

| 層 | 內容 |
|---|---|
| `internal/assets/scenario/events.go` | `RuneGlyphs(text)`：`'.' → 0`、`'A'–'Z' → char − 0x40`，非法字元回 **-1**（不靜默當空白，免得把資料異常偽裝成正常）|
| `cmd/demonwinter/runebox.go` | 載入 `CYPHER.SHP`、9 欄網格繪製、中文說明 |
| `cmd/demonwinter/main.go` | `checkEvent` 在 `IsRuneGlyph()` 時走符文畫面 |
| `assets/lang/zh-Hant/ui.json` | 五條說明文字（`rune.*`）|

原版共四筆符文事件：`%RING.BELL...AT.....MIDNIGHT`（DATA1）、
`%YMROS.IS...MINE`（DATA3）、`%.SECRET..ENTRANCE..TO.ICE..CATHEDRAL…`（DATA4）、
`%...VOID`（DATA5）。

實機：`workplace/dump/persist/84-rune-glyphs.png`
（`-rune=YMROS.IS...MINE`，9 欄排版正確、中文說明在下方）。
`-rune` 這個偵錯旗標是為驗收加的 —— 那四筆要走到特定事件格才看得到。


---

## 8. 根因：修正寫在「總表」裡，原文沒改

`docs/re/02` §2.5 有一張「**對 `event-script.md` 的修正總表**」，
把該文件的過期敘述逐條列出來。但 **`event-script.md` 的原文那幾行沒有改**。

於是：

- 我查 `event-script.md` §129 → 看到「推測」→ 以為未解
- 沒查 `docs/re/02` → 那裡標著「已驗證」

**同一件事在兩份文件裡的信心等級不同步。**

這比「沒查手上有的」更難防 —— 我**確實查了**一份文件，只是查到過期的那份。
可行的做法有兩條：

1. **寫「對 X 的修正總表」時，同時改 X 的原文**（加 `⚠ 已於 … 更正` 的行內註記）。
   總表適合當歷史紀錄，不適合當唯一的更正管道。
2. 看到「推測／未解」時，**grep 那個關鍵字（這裡是 `18fa`）確認沒有別處標「已驗證」**。
   信心等級低的敘述特別值得交叉查 —— 因為它會誘發「去解一遍」。

本輪已照 1 修好 `event-script.md` §129。
