# 107 — 最後稽核：arena、曆法與三個過期問題

這一篇不是再猜一次，而是把散落在後期文件、IDA 9.4 listing 與已上線程式裡的
證據集中起來。目的有兩個：關掉其實早已解出的 worklist 項目；保留真正尚未
裁決的位元組，不用「完成度」壓力把猜測寫成原作規則。

## 1. 18 段 arena：E1 六格全部有用途

啟動配置器把 18 段指標放在 `ds:548c` 起的 far-pointer table。IDA
`sub_1DDD9`（listing 約 26080–26210）依序取出各段，建立後續 alias。

| arena 索引 | 長度 | 定案用途 | 證據 |
|---|---:|---|---|
| 2 | 256 | `ITEMLOCB.DAT` 位置表 | 檔長完全相等；`docs/re/94` 已追到讀寫 consumer |
| 3 | 5,829 | `FILES.DTT` 字串池 | 檔長完全相等；字串切表與地城道具敘述 consumer |
| 6 | — | 戰場視線遮罩 | `docs/re/22` 的 9×9 shadow table consumer |
| 8 | — | 遭遇表 | `docs/re/22`、`30` 的遭遇與掉寶 consumer |
| 14 | 256 | 戰場／觀室格網 scratch | alias `ds:514e`；清 0x80 bytes、填視野，再複製 0x51＝81 格 |
| 15 | 200 | 怪物名稱字串 scratch | alias `ds:52ca`；建立戰鬥單位時複製名稱並保存各槽指標 |

索引 14/15 最容易混淆：`514e`、`52ca` 是配置器複製後的 alias，不是
`548c + index*4` 的 pointer-table 槽位。用 alias 倒推 arena index 會差一層。

結論：E1 應結案；這些都是執行期 scratch/載入緩衝，remake 不必仿造同一塊
連續 DOS arena，只需保存相同資料邊界與索引拓樸。

## 2. DOS 版時間是 `26h/23h/17h`

IDA 9.4 `sub_228B6` 每當 `trailer[+A0h]` 累積到 11，重設為 1、增加
`trailer[+9Fh]`，再呼叫 `sub_22A53`。後者的原指令是：

```asm
cmp byte ptr es:[bx+9Fh], 26h
mov byte ptr es:[bx+9Fh], 1
inc byte ptr es:[bx+9Eh]
cmp al, 23h
mov byte ptr es:[bx+9Eh], 1
inc byte ptr es:[bx+9Dh]
cmp al, 17h
mov byte ptr es:[bx+9Dh], 1
```

因此常數不是十進位字串：

- `26h` = 38：時辰碰到 38 時回 1；
- `23h` = 35：日期碰到 35 時回 1，即有效日 1–34；
- `17h` = 23：月份碰到 23 時回 1。

劇情亦獨立要求時辰至少能到 25：夜鐘比較 `hour >= 24`，旅人的床直接寫 25
（`docs/re/100`）。所以不能為了迎合 Apple II 手冊的「26 小時」而把 DOS
程式的 `26h` 改讀成十進位 26。remake 的 `38/35/23` 是 DOS 行為重現。

## 3. `[4C90]` 不是商隊規模

舊 C9 把「戶外商隊 size base `[4C90]`」列成未知。這句混了兩層：

1. `sub_1DDD9` 明確把一段 arena 指標複製到 far pointer `ds:4C90`；
2. 換圖常式把它當 6-byte `EXITS.DAT` 記錄掃描，取 `+0..+5` 的來源座標、
   目的 map/座標/商隊參數；
3. 真正進入商隊時，基準已被複製到 `ds:5C60`；預設來源是存檔 `+AFh`，
   戶外則是命中的地圖／出口記錄參數（`docs/re/50`）。

所以 `[4C90]` 是出口記錄表基址，不是一個「尚未找到的 size 變數」。
remake 已用 `EXITS.DAT` 的第六 byte 更新 `MerchantBase`，這一格應結案。

## 4. `ITEMS.DAT f1–f6` 只剩一欄缺名字

`docs/re/25`、`30` 與 `internal/assets/gamedata/items.go` 已經共同定案：

- f1：價格；
- f3：charge kind 0–3；
- f4–f7：四個效果類別；首值為 0 時是排除清單，否則是候選清單。

舊 C7 的「f1–f6 逐欄未定」已不成立。唯一未完成的是第二數值欄：
目前程式保守叫 `WeaponSlot`，它對武器與部分可持用飾品為真，但藥膏／藥水等
邊界不夠乾淨。它目前不參與 remake 規則，故保留原值、不得據名稱新增行為。

## 5. 真正仍開著、但不該阻擋發行的證據缺口

- C1：價格量級與戰鬥金幣的原版動態 oracle；remake 公式已有靜態強證據，
  但仍值得做相鄰邊界注入。
- C5：怪物 level/EXP 的「搬入戰鬥單位」那兩條確切 MOV 尚未標名；讀取端、
  record 同構與 runtime 行為已有交叉證據。
- trailer `+09h`：位於九格 formation 與 32-bit gold 之間，兩份原版存檔皆 0，
  沒有 consumer；remake 原樣 round-trip，不賦予玩法。
- ~~C13~~：**已由 `sub_11CBF` 結案**。`11h`–`14h` 分別修改
  MaxSP／速度／力量／技巧，`15h` 是獨立的武器戰鬥效果；兵器庫釘頭鎚
  `12h/0Ch` 因而確定是速度 +2（`docs/re/109`）。

其餘三項是 provenance/命名或 oracle 強化，不是已知的破關缺口。若往後取得新存檔
或第二份同引擎遊戲，應優先用資料差異交叉驗證，而不是讓 remake 猜出新規則。
