# 72 — `CYPHER.SHP` 是符文字型，`%` 是切字型的標記

`docs/formats/event-script.md` §129 對事件文字裡的 `%` 控制字元寫著：

> **推測** 符號用途：`%` 可能標記「這段文字要用特殊字型／等寬符文顯示」，
> 句點取代空白可能是因為符文顯示程式用固定格數排版

**這個推測成立。** 這一篇把證據鏈補完。

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

## 4. 三方閉合

| 證據 | 來源 |
|---|---|
| `%` 開頭才載入 `CYPHER.SHP` | 反組譯 `0x1a85c` |
| 27 個 16×16 字形 ＝ 26 字母 + 句點 | 檔案大小 |
| 內容是符文不是拉丁字母 | dump 出來的 PNG |
| 密語文字用句點取代空白 | `event-script.md` §129 的資料觀察 |

四條互相獨立，指向同一個結論。

---

## 5. 這修正了 `docs/re/65` 的猜測

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

## 6. 對中文化的影響

密語提示是**符文圖形**，不是文字 —— 中文化時有三種選擇：

1. 照原版用符文顯示（玩家看不懂，得查手冊或猜）
2. 用中文顯示（破壞「解謎」的設計）
3. 符文 + 中文對照

**這需要決定，還沒決定。** 記在這裡，`CONTEXT.md` §7 的 C 組要補一項。

原版的設計意圖顯然是 1 —— 那些提示本來就該讓玩家費工夫。
但 1990 年軟體世界代理版怎麼處理，`docs/manual-cht/` 可能有線索，還沒查。
