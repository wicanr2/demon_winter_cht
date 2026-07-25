# 法術 K/M 常數表（DEMON.INT / FILES.DAT 反組譯）

> 任務目標：找出 35 個法術（+12 種幻術/召喚生物）的 K/M 常數表，解除法術系統的
> 唯一硬阻塞。**結論：找到了。表不在 `DEMON.INT` 裡，而在 `DEM_DATA/FILES.DAT`。**
> 證據來源：`workplace/ghidra/export/`（`disassembly.asm`、`decompiled/*.c`）、
> 對 `workplace/orig/demwin/DEMON.INT` 與 `workplace/orig/demwin/DEM_DATA/FILES.DAT`
> 原始位元組的直接讀取核對、`docs/manual/part-3.md`／`part-4.md` 手冊原文交叉驗證。
> 位址換算沿用 `docs/re/00-ghidra-setup.md`：`file_offset = segment*16 + offset - 0xC400`。
> 本檔與 `docs/re/16-combat-details.md`（同日、另一輪任務找到的效果分派跳表、
> 5×5 AOE、Use 道具流程）互補，兩份文件的 `effect_type`/`school_id` 編號完全一致，
> 已交叉核對。

---

## 0. 找表的過程摘要（供下一輪參考取捨）

任務單建議的三條路都試過，最終命中的是**路徑 2 的變體**：

1. **順著 `FUN_1000_2a53` 的索引找**：追過，但 `[0x4e28]`（`FUN_1000_114f` 用來
   讀 K/M 記錄的 base pointer）被追到一個**執行期才動態配置記憶體**的 arena
   （`CALLF` 到疑似 DOS `malloc`-like 函式 `1d9f:0361 -> 3000:fa58`，回傳值存進
   `[0x5484]/[0x5486]`，經過一連串以 `[0x1535]` 表為大小、`[0x53b2]`（EGA/CGA
   模式相關的螢幕緩衝大小）為起點的「切割 arena」算術，才算出 `[0x4e28]`）。
   **這條路徑本身無法純靜態算出最終 segment:offset**（依賴 DOS 執行期記憶體
   配置結果），一度考慮改用 DOSBox `dosbox-debug` 動態附掛除錯（環境已具備，
   見 `docs/re/01-dosbox-reference.md`），但 ncurses 除錯主控台在 `tmux` 裡的
   螢幕重繪不穩定（`capture-pane` 常常回傳空白），排查成本開始超過改用路徑 2
   的成本，故**放棄這條動態附掛路線**，改採路徑 2。
2. **順著已知數值（35 個法術最低 SP）反查，在 binary 裡搜位元組序列**：
   直接對 `DEMON.INT` 搜尋 5 個符文系的最低 SP 序列（連續 byte／連續 word／
   任意 1–20 bytes 跨距）**完全沒有命中**——證實這串數字不是以連續陣列的
   形式存在 `DEMON.INT` 裡。**關鍵轉折**：在追 `[0x4e28]` 的 arena 配置過程中，
   順手發現遊戲會在開機時讀取 `\dem_data\files.dtt` 與 `\dem_data\files.dat`
   兩個檔案（DOSBox `dosbox-debug` 的 `FILES:file open` 記錄檔證實，見 §0.1）。
   `FILES.DTT` 是純文字資源（法術名稱＋效果訊息字串，43 組），`FILES.DAT`
   是與它同順序的二進位資料。**對 `FILES.DAT` 重新套用「跨距搜尋」，在
   record size=10 bytes（與 `FUN_1000_114f` 的 5-word 記錄結構完全吻合）、
   欄位為單一 byte 的假設下，命中 34/35 個已知最低 SP 值，一次到位。**
3. **順著幻術/召喚成本反查**：沒有走到這一步就已經在路徑 2 命中，故未執行；
   12 種生物的成本表本輪**沒有**在 `FILES.DAT` 找到對應區塊（見 §4）。

### 0.1 DOSBox 動態記錄佐證檔案載入順序（已驗證，僅供佐證，非本檔核心證據）

用 `dosbox-debug`（`apt install dosbox-debug`，docker 內臨時安裝，未動
`workplace/orig/`）搭配 Xvfb 起遊戲，`FILES:file open` log 顯示開機早期
（載入 `demon.int` 之後、進主選單之前）依序開啟：

```
\demon.int
\dem_data\got.fnE
\dem_data\party.dat
\dem_data\demon.shE
\dem_data\itemlocb.dat
\dem_data\files.dtt
\dem_data\files.dat
```

`files.dtt`／`files.dat` 緊鄰載入，佐證兩者是配對資源。本檔最終結論**不依賴**
這次動態紀錄（純靜態位元組核對即可重現，見 §6 可重跑片段），這裡列出只是
如實記錄找表過程、供下一輪參考「動態附掛在本專案的即時成本」。

---

## 1. 表的位置與結構（已驗證）

**檔案**：`workplace/orig/demwin/DEM_DATA/FILES.DAT`（2254 bytes）
**表位置**：檔案位移 `0x45e`（1118）到 `0x60c`（1548），共 430 bytes
**記錄數**：43 筆，每筆 10 bytes（5 個有號 16-bit word，little-endian）
**索引順序**：與同目錄 `FILES.DTT` 的前 43 組「名稱＋訊息」字串**完全同序**
（`FILES.DTT` 開頭依序是 `COLUMN OF FIRE / burnt for`、`FLAME STRIKE / struck
down by fire`……直到 `THE END`），這個順序正是 `FUN_1000_114f(spell_index)`
（`docs/re/09` §4.1 已驗證的記錄載入函式）與 `FUN_1000_293d`（施法選單，
`docs/re/00` 已驗證）共用的 `spell_index`（0–42），與 `docs/spec/02-combat.md`
提到「法術一定有個 ID」的那個 ID **是同一個空間**。

### 記錄結構（已驗證）

```
offset 0: school_id   (word, 1..6)   對應 [0x4e2c]（docs/re/16 稱 spell_school_id）
offset 2: effect_type (word, 0..17)  對應 [0x4e2e]（docs/re/09/16 稱 effect_type）
offset 4: K           (word, 有號)   對應 [0x4e30]
offset 6: M           (word, 有號)   對應 [0x4e32]，= 最低 SP 投入（見 §2）
offset 8: w4          (word)         用途未解，觀察值 0/1/2（見 §5）
```

`school_id` 對照（已驗證，由資料本身的分組規律讀出，逐一與符文系清單核對）：

| school_id | 符文系 |
|---|---|
| 1 | 火焰符文 Fire Runes |
| 2 | 金屬符文 Metal Runes |
| 3 | 風之符文 Wind Runes |
| 4 | 寒冰符文 Ice Runes |
| 5 | 靈魂符文 Spirit Runes |
| 6 | 幻術/召喚類別（僅 2 筆佔位記錄，見 §4） |

**找表方法（可重跑）**：用「已知 34/35 個最低 SP 數值 + 已知 record 位置（0,1,2,…
略過幻術/召喚/句尾佔位記錄）」當 oracle，對 `FILES.DAT` 做**跨距＋欄位偏移**
暴力掃描（record_size 2–20 bytes、field_offset 0–19 bytes 全組合），只有
`record_size=10, field_offset=6, table_start=0x45e` 這一組合給出 34/35 的
高分命中（其餘偏移只是同一個表的等價表示，`table_start+field_offset` 恆為
常數 `0x464`）。腳本見 §6。

---

## 2. `M` = 最低 SP 投入（已驗證，來自程式碼本身，非猜測）

`FUN_1000_11e5`（`docs/re/09` §4.2 已知的效果套用函式之一）在套用效果前，
對玩家輸入的 SP 投入量做這一步檢查（反編譯逐字節錄，`workplace/ghidra/export/
decompiled/1000_11e5_FUN_1000_11e5.c`）：

```c
if (param_2 < *(int *)0x4e32) {           // 投入的 SP < M
    *(int *)0x4c7c = *(int *)0x4c7c + 1;
    FUN_1d9f_1361(0x2f9);                  // "not enough points" 一類訊息
    ...
    return 0;                              // 施法失敗
}
```

`0x4e32` 正是 `FUN_1000_114f` 載入的第 4 個 word（記錄 offset 6），也就是本檔
稱的 `M`。**這證實 `M` 就是玩家手冊/`translations/glossary.md` 記載的「最低 SP」
——不是猜測，是直接讀到的判定式**，因此可以用手冊/glossary 的已知數值當 oracle
反查整張表（見 §1 找表方法）。

---

## 3. 35 個法術完整參數表（已驗證，附交叉驗證）

依 `translations/glossary.md` 第 6 節符文系分類列出。`spell_index` 是本表在
`FILES.DAT` 裡的記錄編號（0–42 空間，見 §1）。「glossary 最低SP」欄取自
`translations/glossary.md` 既有資料；「手冊原文」欄取自 `docs/manual/part-3.md`／
`part-4.md`（如有明確數字）。

| spell_index | 法術 | school | type | K | M | glossary最低SP | 手冊/攻略交叉驗證 |
|---|---|---|---|---|---|---|---|
| 0 | 烈焰之柱 Column of Fire | Fire | 5(HP) | −3 | **1** | 1 | 一致 |
| 35 | 魔法火炬 Magic Torch | Fire | 12(光源) | 2 | **2** | 3 | **不一致**，見 §3.1 |
| 3 | 烈焰護盾 Flame Shield | Fire | 7(護甲) | 1 | **4** | 4 | 一致 |
| 2 | 火焰風暴 Fire Storm | Fire | 1(AOE) | 15 | **10** | 10 | 一致；type=1 與 `docs/re/16` 的 AOE handler（138d:3c8d）交叉印證 |
| 28 | 熔解 Melt | Fire | 10(解除束縛) | 1 | **11** | 11 | 一致；手冊「每一級束縛需投入 11 點法力值」，與 M=11 完全吻合 |
| 1 | 烈焰打擊 Flame Strike | Fire | 2(高K/M判定) | 25 | **16** | 16 | 一致；見 §3.2（即死類） |
| 7 | 力量術 Strength | Metal | 4(力量) | 1 | **1** | 1 | 一致 |
| 8 | 護甲術 Armor | Metal | 7(護甲) | 1 | **2** | 2 | 一致 |
| 4 | 劍刃術 Sword | Metal | 5(HP，傷害) | −5 | **2** | 2 | 一致；訊息 "sliced for"，實為金屬系單體傷害法術 |
| 9 | 鏽蝕護甲 Rust Armor | Metal | 7(護甲，負向) | −1 | **3** | 3 | 一致 |
| 5 | 鎖鏈束縛 Chains | Metal | 11(施加束縛) | 1 | **10** | 10 | 一致；手冊「每投入 10 點法力值可獲得一級束縛」與 M=10 完全吻合 |
| 29 | 掙脫束縛 Break Bonds | Metal | 10(解除束縛) | 1 | **11** | 11 | 一致；手冊「每一級束縛需 11 點」 |
| 6 | 死亡之刃 Death Blade | Metal | 2(高K/M判定) | 18 | **15** | 15 | 一致；見 §3.2 |
| 15 | 寒顫 Chill | Ice | 3(技巧，負向) | −1 | **1** | 1 | 一致 |
| 36 | 晶光 Crystalight | Ice | 12(光源) | 2 | **2** | 2 | 一致 |
| 18 | 寒冰護盾 Ice Shield | Ice | 7(護甲) | 1 | **3** | 3 | 一致 |
| 16 | 緩速 Slow | Ice | 6(速度，負向) | −3 | **3** | 3 | 一致 |
| 14 | 冰雹風暴 Hail Storm | Ice | 1(AOE) | 8 | **7** | 7 | 一致 |
| 17 | 冰封 Freeze | Ice | 11(施加束縛) | 1 | **9** | 9 | 一致 |
| 32 | 治療 Heal | Spirit | 5(HP) | 3 | **1** | 1 | 一致；手冊「至少能治癒相當於投入法力值數量的生命值」與 §2.4 reroll 下限機制吻合 |
| 20 | 衰弱 Weaken | Spirit | 4(力量，負向) | −1 | **1** | 1 | 一致 |
| 21 | 笨拙 Clumsiness | Spirit | 3(技巧，負向) | −3 | **2** | 2 | 一致 |
| 22 | 聖域 Sanctuary | Spirit | 7(護甲) | 1 | **3** | 3 | 一致 |
| 34 | 法力轉移 Transference | Spirit | 13(SP) | 3 | **3** | 3 | 一致；手冊「至少 3 點」逐字吻合 |
| 33 | 解毒 Cure Poison | Spirit | 9(解毒) | 60 | **9** | 9 | 一致；見 §3.3 |
| 23 | 枯萎打擊 Wither Strike | Spirit | 14(獨立判定) | 25 | **15** | 15 | 一致；見 §3.4 |
| 19 | 靈魂折磨 Spirit Wrack | Spirit | 2(高K/M判定) | 26 | **20** | 20 | 一致；見 §3.2 |
| 37 | 復活 Resurrect | Spirit | 8(獨立判定) | 25 | **25** | 25 | **完全吻合**，見 §3.5（K=M=25 對上手冊「25 點、25% 成功率」） |
| 12 | 勝利之翼 Wings of Victory | Wind | 3(技巧) | 1 | **1** | 1 | 一致 |
| 13 | 羽翼 Wings | Wind | 6(速度) | 3 | **4** | 4 | 一致 |
| 31 | 生命之息 Breath of Life | Wind | 5(HP，治療) | 10 | **5** | 5 | 一致 |
| 10 | 暴風 Tempest | Wind | 1(AOE) | 7 | **6** | 6 | 一致 |
| 27 | 御風而行 Wind Walk | Wind | 17(獨立判定) | 1 | **10** | 10 | 一致；見 §3.6（傳送術，非復活術） |
| 11 | 凝滯之氣 Still Air | Wind | 11(施加束縛) | 1 | **11** | 11 | 一致 |
| 30 | 自由之風 Freedom | Wind | 10(解除束縛) | 1 | **13** | 13 | 一致 |

**35/35 個法術全部在表中找到記錄，34/35 個 `M` 值與 `translations/glossary.md`
逐字吻合，1 個（魔法火炬）有 1 點落差，見 §3.1。**

### 3.1 唯一的落差：魔法火炬（Magic Torch）

`FILES.DAT` 讀出 `M=2`，`glossary.md` 與手冊原文都記載「最低投入 3 點法力值」
（`docs/manual/part-3.md:266`：「**魔法火炬（Magic Torch）**：紮營用法術，用來
提供光源。最低投入 3 點法力值，可提供相當於一支火把一天份的光亮」）。

**判讀**：這是 35 個法術裡唯一的落差，其餘 34 個（含另一個同 `effect_type`
的光源法術「晶光」M=2，glossary 也記 2，完全一致）都精確吻合，統計上不像
讀表位置錯誤（若位置錯，理應大量報錯而非單點）。手冊原文緊接著寫「可提供
**相當於一支火把一天份**的光亮」——這句話描述的更像是「3 點法力值換算成
的具體效果（一天份亮度）」而非「施法的最低門檻」，程式碼裡真正的門檻檢查
（`if (param_2 < M) fail`）用的是 `M=2`，手冊的「3」很可能是「達到某個
效果基準（如燈光持續時間對齊遊戲內一天）所需的投入」而非硬性下限。
**標記：`M=2` 為已驗證的程式碼行為；手冊「3」為描述性建議值，兩者不算真衝突，
但需要下一輪如果有機會用 DOSBox 動態驗證「投入 2 點能否成功施放」來坐實**。

### 3.2 高 K/M 判定類（type=2）：可能就是「即死類」法術

`type=2` 只出現在 4 個法術裡的其中 3 個：**烈焰打擊 Flame Strike**（K=25,M=16）、
**死亡之刃 Death Blade**（K=18,M=15）、**靈魂折磨 Spirit Wrack**（K=26,M=20）。
K、M 數值都明顯高於其他類型（多數法術 K∈[1,10]、M∈[1,13]），且與手冊描述
高度吻合：

> `docs/manual/part-3.md:278`「**死亡之刃（Death Blade）**：一把魔法之劍憑空
> 出現，試圖將受害者斬殺。**若未能成功，則不會造成任何傷害**。除非投入大量
> 法力值施放出威力強大的死亡之刃，否則成功機率不高。」
>
> `docs/manual/part-3.md:264`「**烈焰打擊（Flame Strike）**：試圖召喚一道火焰
> 箭，直接將目標擊殺。這個法術耗費極高，若沒有投入大量法力值很難成功施放。
> **在所有致死系法術中，這是威力最強的一個**。」

「若未能成功，則不會造成任何傷害」證實這**不是** `docs/re/09` §4.2 的
「reroll-biased magnitude」傷害公式（那個公式的結果永遠 ≥ 上限的 1/3，
不會有「完全不造成傷害」的結果）——這應該是一個**獨立的成功/失敗判定**
（很可能是 `RNG(100) <= SP*K/M`，類似 §3.5 復活術的判定式），成功才即死，
失敗則毫無效果。**[假設，未驗證]**：本輪沒有在 `FUN_1000_11e5`／
`FUN_138d_10bc`／`FUN_138d_3c81`（`docs/re/16` 已知的三個效果分派函式）裡
找到明確標成 `case 2:` 的獨立 handler（`FUN_1000_11e5` 的 switch 直接把
`type==2` 歸進「不足點數/拒絕」分支——這條路徑應該只服務非戰鬥情境；
`FUN_138d_3c81` 的 18 項跳表裡 `effect_type 2 -> 138d:3cc5` **本輪未展開**，
留給下一輪，見 §8）。K/M 數值本身**已驗證**（來自表），公式細節**未驗證**。

### 3.3 解毒（Cure Poison，type=9）：K=60 的合理解讀

手冊原文：「這個法術**通常都能成功**，但若要確保成功，可以多投入一些法力值。」
若採用與 §3.2 類似的「`RNG(100) <= SP*K/M`」判定式，代入 `M`（最低 SP=9）：
`成功率 = 9*60/9 = 60%`——在最低投入下就有 60% 成功率，符合「通常都能成功」
的描述；投入更多 SP 會讓比例超過 100%（穩定成功），符合「多投入法力值確保
成功」。**[假設，公式套用未逐位元組核對，但與手冊定性描述吻合度高]**。
`Poison`（trap 用，spell_index 39，同樣 type=9，K=5,M=5）可能是相反方向的
「施加中毒」判定，同一公式在 `M` 投入下給出 `5*5/5=100%`（陷阱通常必定成功）。

### 3.4 枯萎打擊（Wither Strike，type=14）：獨立於 type=2 的另一個判定

手冊原文：「**枯萎打擊（Wither Strike）**：危險的法術，使受害者急速衰老，
**力量、技巧、速度都會下降**，效果持續整場戰鬥。**投入的法力值越多，成功機率
越高**。」——這與 §3.2 的「即死類」在型別上被程式碼明確**區分開來**
（type=14，不是 type=2），符合「同時降三個屬性」這種不同於單純 HP 傷害/
即死的複合效果，猜測需要連續套用 3 次類似 §4.3（`docs/re/09`）的
技巧/力量/速度欄位變更，但**判定/套用的細節函式本輪未定位**（`FUN_138d_3c81`
跳表 `effect_type 14 -> 138d:3f0f`，未展開）。`Youth`（spell_index 40，
NPC/陷阱用，同為 type=14，K=25,M=18）很可能是相反方向的「返老還童」判定，
兩者共用同一機制。

### 3.5 復活（Resurrect，type=8）：K=M=25 精確對應手冊「25 點、25%」

`FUN_1000_11e5` 的 `case 8: case 9:`（`docs/re/09` §5 已列出程式碼，本輪
重新解讀）：

```c
case 8: case 9:
    do {
        local_8 = FUN_1000_1877();               // 挑目標
        if (取消) return 0;
        local_4 = target.status(+0x102);
    } while ((local_4 != 5) && (type == 8));      // type==8 時，必須挑到 status==5 的目標
    ...
    roll_a = RNG(100);
    roll_b = RNG(100);
    if (type==8 && roll_b<roll_a) roll_a = roll_b;   // type==8 額外取兩次骰的較小值
    if ((SP*K/M) < roll_a) { print fail; }
    else {
        if (type==8) { target.status(+0x102)=0; target.+0xfd=1; }   // 清除狀態、設定旗標
    }
```

**已驗證**：`type==8` 要求目標的 `+0x102` 狀態欄位為 `5`——對照
`docs/re/06`／`09` 已知「`+0x102`＝0 正常/1 中毒」的部分編碼，`5` 高度可能
代表「死亡」，使 `type==8` 的目標篩選變成「只能對死者施法」，完全符合
「復活術」的語意。**已驗證**：`spell_index 37`（復活術）的 `K=25, M=25`，
代入這條判定式在最低投入（`SP=25`）時：`成功率 = 25*25/25 = 25%`——與手冊
「**最低投入 25 點法力值，成功機率只有 25%**」（`docs/manual/part-3.md:324`）
**逐字精確吻合**，是本輪信心度最高的交叉驗證。成功後 `target.status=0`
（清除死亡狀態）、`target.+0xfd=1`（可能是「甦醒/待恢復」旗標），與手冊
「若角色復活成功，中毒、束縛等所有異常狀態都會一併解除」的描述方向一致
（`+0x102` 這個欄位很可能同時編碼多種異常狀態，清零即等於解除全部異常）。

**修正 `docs/re/09` §4.4 的猜測**：`docs/re/09` 原本猜測「`type=0x11`（17）
可能是復活術」，本輪確認**是錯的**——`0x11` 實際是御風而行（見 §3.6），
復活術是 `type=8`。`docs/re/09` §5 把 `type∈{8,9}` 一併描述成「幻術/召喚
共用骨架」，本輪確認**這個描述只對「Cast 施法」路徑裡的 `9`（真正的召喚，
若目標存在合法對象）成立，`8` 在復活術這個實例上其實是「檢查目標是否已死」
的專用判定，兩者共用程式碼骨架但語意分岔**（`type==8` 分支多了一個「目標
必須是 status==5」的前置迴圈，`type==9` 沒有這個限制）。

### 3.6 御風而行（Wind Walk，type=17）：傳送術，不是復活術

`FUN_1000_11e5` 的 `case 0x11:`：

```c
case 0x11:
    print(name, msg);
    local_4 = FUN_1000_1902(...);       // 扣 SP / 檢查
    if (local_4 != 0) { print(cast_ok_msg); play_sound(); }
    party.+0xa9 = 0;
    if (party.+0xbe == 0) {
        party.+0xa1 = 0x1c; party.+0xa2 = 0x32; party.+0xa3 = 0x22;   // (28,50,34)
    } else {
        party.+0xa1 = 0x27; party.+0xa2 = 0x23; party.+0xa3 = 0x2b;   // (39,35,43)
    }
    return 1;
```

寫入 3 個固定數值到 `+0xa1/+0xa2/+0xa3`（依旗標 `+0xbe` 二選一），最合理的
解讀是「目的地座標（X/Y/地圖編號）」的預設傳送點，與手冊「**御風而行
（Wind Walk）**：可將施法者與整支隊伍以魔法傳送到安全地點的法術，在紮營中
任何時候都能使用」完全吻合（"the party is displaced" 是這個法術在
`FILES.DTT` 的訊息文字，同樣對應「傳送」語意）。`+0xa1/+0xa2/+0xa3` 這三個
欄位的具體語意（是否真的是座標、`+0xbe` 旗標代表什麼）**未逐一核對**，
標記 `[假設]`，但「這是傳送術而非復活術」的判定本身**已驗證**（K=1, M=10，
與手冊/glossary「最低 10 點」一致，且程式碼行為與手冊描述的傳送語意吻合，
不涉及 magnitude/傷害/HP 欄位）。

---

## 4. 幻術/召喚（12 種生物）：本輪**未找到**專屬 K/M 表

`FILES.DAT` 的 43 筆記錄裡，`spell_index 24`（Summon）與 `25`（Illusion）
是**全零的佔位記錄**（`school=6, type=15, K=0, M=0, w4=0`）——這兩筆只是
「選單分類標題」，不是實際可施放的幻術/召喚生物本身各自的參數。

真正的 12 種生物（`translations/glossary.md` 第 7 節，幻術 SP 2/4/6/8/10/
14/18/20/20/20/20/20，召喚 SP 為兩倍）應該有自己獨立的表，本輪：

- 對 `DEMON.INT`、`FILES.DAT`、`DEM_DATA/*` 全部檔案跑過與 §1 相同的跨距
  暴力掃描（record_size 2–20、field 0–19），**用 12 個幻術成本值當 oracle
  完全沒有命中**。
- 對 12 個召喚成本值（幻術的兩倍）同樣掃描，**沒有命中**。
- `docs/re/16-combat-details.md` §6 已經從程式碼路徑追過 `effect_type 8/9`
  在 `FUN_138d_3c81`（Use 道具/AI 路徑）的行為，發現與「建立新怪物」的直覺
  語意不符（限定目標必須是玩家），懷疑真正建立新戰鬥單位的邏輯藏在
  `FUN_138d_2f7e`／`FUN_138d_2fa0`（都未展開）。**本輪與 `docs/re/16` 得到
  相同的「未解」結論，沒有新突破**。

**合理推測**（`[假設，未驗證]`）：12 種生物的資料**可能**在 `MONSTER.DAT`
裡（怪物基礎屬性表，`docs/formats/game-data-tables.md` 已有格式紀錄），
幻術/召喚只是「用 SP investment 決定能否召喚成功＋召喚出哪一隻」，屬性直接
借用 `MONSTER.DAT` 既有的怪物記錄，而不是額外開一張 K/M 表。這與
`docs/re/09` §8 建議的「短期可先用 `MONSTER.DAT` 對應生物的基礎屬性頂替」
方向一致，但**這只是合理推測，不是找到的證據**，如實標記。

---

## 5. 未解欄位：`w4`（記錄第 5 個 word，offset 8）

觀察值集中在 `{0, 1, 2}`，35 個真實法術裡各值分布：0 出現最多，1 次之，
2 較少（Wings of Victory、Strength、Armor、Transference、Possession 等
5–6 筆是 2）。嘗試過的假設：

- 「符文系內由便宜到貴的檔位」——不成立（同校內 M 值排序與 w4 值沒有單調關係）。
- 「reroll 次數上限」或「特殊旗標（如需要目標存活/死亡）」——沒有找到程式碼
  讀取 `rec+8` 這個欄位的證據（`FUN_1000_114f` 的反編譯明確寫「rec+8（第 5 個
  word）在本輪追蹤的函式內未見使用」，`docs/re/09` 已有這個結論，本輪重新
  核對過仍然如此，`FUN_138d_10bc`／`FUN_1000_11e5`／`FUN_138d_3c81` 三個
  已知效果套用函式**都沒有讀取這個位移**）。

**維持未解**，標記 `[未解]`，不做無依據的猜測。

---

## 6. 可重跑的驗證片段

```bash
cd /home/anr2/cht/daemon_winter

# 1. 找表本身：對 FILES.DAT 用 35 個已知最低 SP 值當 oracle 做跨距暴力掃描
python3 -c "
import struct
d = open('workplace/orig/demwin/DEM_DATA/FILES.DAT','rb').read()
expected = {
 0:1, 1:16, 2:10, 3:4, 4:2, 5:10, 6:15, 7:1, 8:2, 9:3, 10:6, 11:11, 12:1, 13:4,
 14:7, 15:1, 16:3, 17:9, 18:3, 19:20, 20:1, 21:2, 22:3, 23:15,
 27:10, 28:11, 29:11, 30:13, 31:5, 32:1, 33:9, 34:3, 35:3, 36:2, 37:25,
}
for start in range(0, len(d)-430):
    ok = sum(1 for i,v in expected.items() if d[start+i*10+6]==v)
    if ok >= 30:
        print(hex(start), ok, '/', len(expected))
"

# 2. dump 完整 43 筆記錄
python3 -c "
import struct
d = open('workplace/orig/demwin/DEM_DATA/FILES.DAT','rb').read()
base = 0x45e
for i in range(43):
    w = struct.unpack('<5h', d[base+i*10:base+i*10+10])
    print(f'{i:2d} school={w[0]} type={w[1]:3d} K={w[2]:4d} M={w[3]:4d} w4={w[4]}')
"

# 3. M = 最低 SP 的判定式（FUN_1000_11e5，"not enough points" 分支）
grep -n "0x4e32" workplace/ghidra/export/decompiled/1000_11e5_FUN_1000_11e5.c

# 4. 記錄載入函式（FUN_1000_114f，5-word 結構）
cat workplace/ghidra/export/decompiled/1000_114f_FUN_1000_114f.c

# 5. 復活術 type==8 完整判定（K=M=25 對應手冊 25 點/25%）
sed -n '/case 0x11:/q;/case 8:/,/break;/p' \
  workplace/ghidra/export/decompiled/1000_11e5_FUN_1000_11e5.c
```

---

## 7.1 與既有文件的關係（供維護者裁決，本檔不直接修改對方）

`docs/formats/resource-index.md` §2.3 已經把 `FILES.DAT` 的 `0x420–0x5FF`
標成「uint16 LE 小數值，多在 0–100 範圍，未解」——這個區間**正好涵蓋**本檔
找到的法術表（`0x45e–0x60c`）。建議該文件維護者依本檔 §1–§3 的結論更新那一段
（「未解」→「法術 K/M/type/school 表，43 筆 ×10 bytes，見 `docs/re/15`」）。
`tools/parse_files_index.py` 的 `DAT_KNOWN_SECTIONS`（若有）也可以比照
`DTT_KNOWN_SECTIONS` 的寫法補上這個區段——本檔遵守任務邊界，不直接改動
`docs/formats/` 或既有 `tools/parse_files_index.py`，此段只做交接建議。

## 8. 給下一輪的具體建議

1. **`docs/spec/02-combat.md` 可以把「35 個法術的 K/M 常數表」從阻塞項移除**，
   §3 的完整參數表可以直接照抄進 spec，法術系統可從 DRAFT 升級（協調者裁決，
   本檔不動 `docs/spec/`）。
2. `effect_type` 2、10、11、14、15、16 的**套用/判定函式**仍未逐位元組展開
   （`FUN_138d_3c81` 跳表 `138d:3f95` 已知位址，見 `docs/re/16` §4.4），
   建議下一輪優先讀 `138d:3cc5`（type=2，即死類，本檔 §3.2 的公式是推論）
   與 `138d:3e25`／`138d:3e7f`（type=10/11，束縛的施加/解除機制）。
3. 幻術/召喚 12 種生物的參數表位置**仍未解**（§4），建議下一輪先確認
   `MONSTER.DAT` 是否真的被 `FUN_138d_2f7e`／`FUN_138d_2fa0`（`docs/re/16`
   §6 已定位但未展開）引用，這是目前最有機會的線索。
4. `w4` 欄位（§5）維持未解，不影響法術系統實作（無程式碼讀取它）。
5. Magic Torch 的 1 點落差（§3.1）建議有機會時用 DOSBox 動態驗證
   「投入 2 點法力值能否成功施放」坐實，不影響其餘 34 個法術的可信度。
