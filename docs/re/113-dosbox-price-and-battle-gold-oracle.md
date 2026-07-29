# C1：DOSBox 原版價格與戰鬥金幣動態 oracle

> 狀態：**C1 結案**（2026-07-29）  
> 原版：`DEMON.INT` SHA-256
> `fc1df05513bfa0f1a38f95ce0fbe5e6ec390c8192b2f99b3b6118b3c23868ea5`  
> 環境：DOSBox 0.74-3、EGA、`fixed 8000`

## 1. 市集價格：Seaside 匕首 2 Gold

既有的存檔注入實驗已把隊伍放進 Seaside 並進入 Marketplace：

- `workplace/dosbox/shots/t4-market.png`：商品 `dagger`，畫面直接顯示
  `Gold: 2`。
- `workplace/dosbox/shots/t5-purchase.png`：選 Purchase 後仍顯示
  `dagger / Gold: 2`，並進入 `Give to character ?`。

這不是 `ItemValue` 的商隊隨機貨，而是城鎮市集的確定價格鏈：

```
dagger 底價 2 × Seaside E 10 ÷ 10 = 2
初始議價層級 0（隊伍有人會 Persuasion）→ 仍為 2
```

原版畫面、`ITEMS.DAT` 底價、`TOWN*.DAT` 經濟係數與 remake
`TownVisit.Price` 四方一致。`internal/game/town_test.go` 新增
`TestSeasideDaggerPriceMatchesDOSBoxOracle`，把這筆外部 oracle 釘進測試。

這筆樣本證明的是**市集確定價格鏈與量級**。商隊還會再乘 0.6–1.4 的 RNG
係數，不能要求兩個不同 RNG session 的單筆貨價逐值相等；其靜態 consumer
與分布測試仍見 `docs/re/44`–`52`。

## 2. 戰鬥金幣：固定遭遇得到 31 Gold

### 2.1 控制實驗

原版隨附存檔在神殿門口會固定遭遇：

```
4 × Lvl 1 mage + 1 × Lvl 2 fighter
```

手動打完會把走位、命中、傷害與 AI 的隨機性一起混進實驗。本次只在
**可寫副本**改兩處控制流：

- file `0x784F`：一名玩家完成行動後，將主迴圈檢查用的 `AX` 設為勝利 `1`；
- file `0x7854`：呼叫正常 `sub_1A3EB` 結算時 push 勝利參數 `1`。

原始 bytes 與實驗 bytes：

```
0x784F  8B 46 EE    mov ax,[bp-12h]    → B8 01 00  mov ax,1
0x7854  FF 76 EE    push [bp-12h]       → 6A 01 90  push 1; nop
```

沒有修改 `MONSTER.DAT`、怪物 level／數量、RNG、1.7／2.1 常數、金幣
迴圈、顯示函式或 party gold。pristine original 保持唯讀；命令以 trap
還原 `workplace/dosbox/game/`。這是縮短到**正常勝利結算**的控制實驗，
不是把答案寫進原版。

### 2.2 預測與輸出

remake／反組譯公式：

```
每隻怪 = trunc(1.7^level) + Roll(trunc(2.1^level)) + 3
```

固定遭遇的界線：

```
Lvl 1：每隻 5–6，四隻共 20–24
Lvl 2：一隻 6–9
總額：26–33
```

在戰鬥選單按 `Q` 讓第一名玩家完成行動後，原版正常結算畫面顯示：

```
Exp per chr: 24    Gold: 31
```

證據截圖：`workplace/dosbox/shots/c1-gold-q-02.png`（研究工作區，不進
repo）。`31 ∈ [26,33]`；EXP 也吻合 `(4×22 + 34) ÷ 5 = 24`，
是同一批五隻單位確實進入結算的獨立校驗。

單一輸出不能證明 RNG 機率分布，但足以裁決「level 欄位與量級是否接錯」：
若仍把怪物 `unit+1Ah` 當附魔或固定零，或誤把 EXP 當指數，輸出會落在完全
不同量級。再加上 `docs/re/111` 的直接寫入表，證據鏈閉合。

## 3. 結論

- 市集確定價格：原版 Seaside dagger 2，remake 2。
- 戰鬥金幣：原版固定 level 組合輸出 31，remake 預測 26–33。
- C1 的兩個動態 oracle 都已取得；金幣公式可由「強靜態判讀」升為
  **原版動態驗證**。
