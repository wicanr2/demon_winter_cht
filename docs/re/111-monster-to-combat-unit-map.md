# `MONSTER.DAT` → 戰鬥單位的 11 欄間接搬移表

> 狀態：**C5 結案**（2026-07-29）
>
> Oracle：IDA 9.4 `sub_17C5D`，`loc_17FB5`–`loc_180B7`；
> `DEMON.INT` DS:`0A44`，檔案 offset `0x26544`。

## 1. 為何以前掃不到

舊研究搜尋 `unit+1Ah` 與 `unit+24h` 的直接寫入，只找到玩家／召喚路徑，
怪物建立段看似只有讀取。原因是怪物資料不是十一條固定目的 MOV：

1. `sub_17C5D` 從 MONSTER ASCII 記錄逐欄呼叫 `sub_2F16F` 轉成整數。
2. 欄位序號查 DS:`0A44` 的 word。
3. 該 word 乘 2，成為 38-byte 戰鬥單位中的 byte offset。
4. 以 `unitBase + monsterSlot×38 + mappedWord×2` 間接寫入。

因此靜態搜尋目的絕對位址不會命中；真正的 schema 在資料段映射表。

## 2. 原始映射表

依專案位址公式：

```text
file = (31F0h − 1000h) × 10h + 0A44h + 3C00h = 26544h
```

原始 11 個 little-endian word：

```text
02 00  06 00  09 00  03 00  04 00  0B 00
05 00  12 00  0D 00  07 00  11 00
```

即：

```text
2, 6, 9, 3, 4, 11, 5, 18, 13, 7, 17
```

## 3. 完整欄位對照

| MONSTER 數值欄 | 映射 word | unit byte offset | 語意 |
|---:|---:|---:|---|
| 0 | 2 | `+04h` | 速度 |
| 1 | 6 | `+0Ch` | 力量 |
| 2 | 9 | `+12h` | 技巧 |
| 3 | 3 | `+06h` | HP |
| 4 | 4 | `+08h` | 攻擊／武器骰索引 |
| 5 | 11 | `+16h` | sprite／圖塊欄 |
| 6 | 5 | `+0Ah` | 護甲點數 |
| 7 | 18 | `+24h` | 擊殺經驗值 |
| 8 | 13 | `+1Ah` | level／戰利金幣指數 |
| 9 | 7 | `+0Eh` | SP |
| 10 | 17 | `+22h` | 種族／元素類型 |

兩個 C5 目標因而有直接搬移證據：

```text
MONSTER field 7 -> map[7] = 18 -> unit + 18×2 = +24h
MONSTER field 8 -> map[8] = 13 -> unit + 13×2 = +1Ah
```

## 4. 後續消費端交叉驗證

- `unit+24h` 在怪物建立迴圈立即累加進 `ds:52D0/52D2` 的本場經驗池。
- `unit+1Ah` 在勝利結算作為
  `1.7^level + Roll(2.1^level) + 3` 的金幣指數。
- `unit+0Ah` 被命中公式扣除，檢視畫面以 `Armor: %3d pts.` 顯示。
- `unit+22h` 驅動龍吐息與元素免疫。

這些 consumer 與映射表、召喚表同構三條證據彼此獨立；C5 不再依賴
「值域看起來合理」的推論。

## 5. Remake 狀態

`gamedata.Monster` 與 `game.NewMonsterUnit` 已使用上述語意，包含
`ArmorPoints`、`Experience`、`Level` 與 `Special`。本次不需要改玩法程式；
修正的是證據等級、欄位註解與 worklist。既有 monster parser／combat tests
繼續守住真實檔案錨點與單位搬移。
