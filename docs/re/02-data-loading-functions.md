# DEMON.INT 資料讀取函式反組譯分析

> 分析對象：`workplace/ghidra/export/`（Ghidra 12.1.2 headless 分析結果，見 `docs/re/00-ghidra-setup.md`）。
> 目標：解 `docs/formats/event-script.md` 的「trailer 語意」未解問題（問題 A），以及
> `docs/formats/town-and-map.md` 的「type_code → 設施」未解問題（問題 B）。
> 方法：`rulebook/62` 靜態溯源 SOP —— 從字串常數（`DATA%d.TXT`、`town%d.dat`）反查引用它的函式，
> 往下追 parse 迴圈，欄位讀進哪個變數、後續怎麼用。
> 位址換算公式：`file_offset = segment*16 + offset - 0xC400`（見 `00-ghidra-setup.md`）。

---

## 0. 結論先講

- **問題 A（trailer 語意）：已解，且用 Python 逐欄位模擬對 5 個 `DATA*.TXT` 檔案做了 100% 消耗驗證
  （221/221、82/82、152/152、118/118、59/59 個欄位，零錯位）。** 結論：doc 說的「trailer 0–3 個數字」
  根本不是一個真實存在的可變長度欄位，是**兩個互不相干、各自固定 1 個欄位的東西被線性掃描工具誤黏在一起**
  的假象。見 §2。
- **問題 B（type_code → 設施）：部分解開，且推翻原提問的前提。** 城鎮設施選單**不是**由每筆記錄的
  type_code 分派決定的，而是由**固定位置的第 30 筆（index 29）記錄的 payload bytes**，用**寫死的絕對
  offset** 讀出來的旗標/數值決定。原本「28 種 type_code 對不上 7 個設施字串」這個問題本身的前提不成立——
  那 28 種 type_code 根本不參與設施選單的分派邏輯。真正驅動選單的位元組已完整定位。records 0–28 的
  type_code 意義本次**仍未解開**（這個函式根本不讀它們）。見 §3。

---

## 1. 找函式的方法（字串錨定）

在 `strings.csv` 找到兩個關鍵格式字串：

```
31f0:2e86,10,DATA%d.TXT
31f0:3055,10,town%d.dat
```

在 `disassembly.asm` 反查誰把這兩個位址（換算後 offset 部分 `0x2e86` / `0x3055`）當立即數載入：

```
27037:25be:0e8b  MOV AX,0x2e86      <- 落在 FUN_25be_0e77 內
27891:278d:0135  MOV AX,0x3055      <- 落在 FUN_278d_0098 內
```

兩個都各只有一處引用，直接鎖定函式邊界（`functions.csv`）：

| 函式 | 位址 | 大小 | 對應資料檔 |
|---|---|---|---|
| `FUN_25be_0e77` | `25be:0e77` | 901 bytes | `DATA%d.TXT`（事件表） |
| `FUN_278d_0098` | `278d:0098` | 1884 bytes | `town%d.dat`（城鎮資料，這個函式同時是城鎮選單建構邏輯） |

---

## 2. 問題 A：`DATA*.TXT` 讀取函式與 trailer 語意

### 2.1 函式定位與檔名組字（已驗證）

`FUN_25be_0e77`（`decompiled/25be_0e77_FUN_25be_0e77.c`）一開頭：

```
25be:0e7e  LES BX,[0x4c76]
25be:0e82  MOV AL,byte ptr ES:[BX + 0xa3]     ; 讀全域狀態struct（*0x4c76）+0xa3 欄位 = 地城/資料檔索引
25be:0e8b  MOV AX,0x2e86                       ; "DATA%d.TXT"
25be:0e94  CALLF 0x2000:f30e                   ; = FUN_2f30_000e，sprintf 式組字串
```

`FUN_2f30_000e` 反編譯確認是格式化字串組字函式（呼叫 `FUN_2fd8_005f` 做底層組字，寫入指定緩衝區），
用法完全符合「用 `%d` 格式字串 + 索引組出 `DATA1.TXT`~`DATA5.TXT` 檔名」。**已驗證**。

組完檔名後，程式碼輪詢一個忙碌旗標（`FUN_1d9f_0a8b(0x522a)` != 0 就 loop），這是「等待檔案讀取完成」
的典型 DOS 忙等模式，然後取全域指標 `*0x5534/0x5536` 當作已載入的檔案內容緩衝區指標（`local_1a`）。

### 2.2 逐欄位 parse 迴圈的完整結構（已驗證，含消耗驗證）

外層 `for (local_24 = 0; local_24 < *(int*)0x52f4; local_24++)` 迴圈，每次迭代固定消耗以下欄位序列
（NUL 分隔、ASCII 十進位文字，與 `event-script.md` §2.1 一致）：

```
[A] 圖片ID欄位   -> local_26 = FUN_2f16_000f(...)
[B] TEXT 欄位    -> local_1e = 指標（不解析數值，直接存起點指標）
[C] count 欄位   -> local_2a = FUN_2f16_000f(...)
    若 count != 0：讀 abs(count) 個怪物ID欄位，依序存入 *(GLOBAL+0x16+i)
[E] 戰鬥設定欄位 -> local_28 = FUN_2f16_000f(...)   （doc 稱的「255 終止符」）
[F] 續接碼欄位   -> local_22 = 指標（不解析數值，直接存起點指標；之後視情況再解析）
```

`FUN_2f16_000f`（`decompiled/2f16_000f_FUN_2f16_000f.c`）確認是通用 ASCII→int 轉換函式：跳過空白、
處理正負號、逐位數累加，遇到第一個非數字字元就停止——這正是**為什麼一段控制字元開頭的文字
（如 `"3With Remondadin dead..."`）也能被當成數字欄位解析出 `3`**：解析器碰到 `'3'` 讀出 3，碰到
下一個字元 `'W'`（非數字）就停手，不管後面接的是不是一大段文字。這是理解整個機制的關鍵。

### 2.3 用 Python 逐欄位模擬驗證（本次分析新增的驗證手段）

為了確認上面推出的欄位消耗順序真的跟檔案位元組對得上（而不只是「反組譯看起來像」），寫了一個
獨立的 Python 模擬器（`tokenize` NUL 分隔欄位 + 依上述順序消耗），跑遍全部 5 個 `DATA*.TXT`：

```
DATA1.TXT: 消耗 221/221 個欄位，35 次迭代，零錯位
DATA2.TXT: 消耗  82/82  個欄位，12 次迭代，零錯位
DATA3.TXT: 消耗 152/152 個欄位，26 次迭代，零錯位
DATA4.TXT: 消耗 118/118 個欄位，21 次迭代，零錯位
DATA5.TXT: 消耗  59/59  個欄位，11 次迭代，零錯位
```

**全部 5 個檔案都被乾淨消耗完，沒有任何一筆迭代因為欄位不夠而中途中斷。** 這是很強的證據：
上面推出的欄位順序（A/B/C/ids/E/F 固定 slot，不是自由長度的「trailer」）跟檔案的實際 byte 排列
完全吻合。模擬器原始碼保留在
`/tmp/claude-1000/-home-anr2-cht-daemon-winter/91ac95e6-a548-4e9d-8aa7-8427e2cab911/scratchpad/sim_event.py`
（scratchpad，不在專案版控內；如需重跑可整段複製，邏輯很短）。

### 2.4 各欄位語意逐一解出（已驗證部分）

#### [A] 圖片 ID 欄位 —— 解出檔頭「255」之謎

```
25be:0e39(前一段殘留碼) / FUN_25be_0e77 內:
if (local_26 != 0xff) {
    FUN_307c_0005(...);                          // 存起目前文字緩衝
    FUN_1d9f_281a(*(word*)(local_26*2 + 0x15ba)); // 用 local_26 查表 0x15ba，結果傳給畫圖函式
    FUN_25be_1765();                              // 還原文字緩衝（記憶體區塊 copy 0x1000 bytes）
    FUN_307c_0005(...);
}
```

`FUN_1d9f_281a`（`decompiled/1d9f_281a_FUN_1d9f_281a.c`）反編譯出來：`param_1` 落在 1–6 或等於 8 時，
會先後呼叫 `FUN_1d9f_19fe(6)` / `FUN_1d9f_19fe(5)`（**推測**：磁碟讀取音效，1988 年 SSI 遊戲常見的
「換片/讀取」音效觸發模式），接著設一個旗標、呼叫繪圖初始化（`FUN_2cdc_20b7`）、記錄座標
（`*0x4c94`/`*0x4c96`）。**結論（已驗證存在此機制，音效/繪圖用途為推測）**：欄位 A 是「這個房間要不要
顯示一張插圖」的插圖 ID，`0xff`(255) = 無插圖（預設）。

**這同時解開 `event-script.md` 開放問題 #1（每個 `DATA*.TXT` 檔頭固定 `255` 的用途）**：模擬結果顯示，
迴圈第 0 次迭代（`local_24=0`）讀的第一個欄位就是檔案最開頭那個欄位——也就是說，**檔頭的 `255`
在程式眼中根本不是特殊的「header」，它就是第 0 筆記錄的欄位 A（插圖 ID），值剛好是預設的「無插圖」**。
不是記錄總數、不是哨兵，是欄位語意重用的自然結果。**已驗證**（有反組譯 + 模擬雙重佐證）。

#### [B] TEXT 欄位 —— 房間敘述文字

`FUN_25be_158e`（`decompiled/25be_158e_FUN_25be_158e.c`）反編譯確認是**文字換行顯示函式**：
以 40 字元（`local_36=0x28`）為寬度，逐詞往回找空白斷行，`FUN_1d9f_1361` 印出每行，每印 5 行
（`local_34==5`）就觸發翻頁暫停。這就是遊戲顯示房間敘述文字的函式。`FUN_25be_0e77` 在正常路徑
（`*local_1e != '\0'`）會呼叫 `FUN_25be_158e(local_1e, ...)` —— local_1e 正是欄位 B 的起點指標。
**已驗證**。

#### [C] count 欄位與怪物 ID 陣列 —— 與既有交叉驗證完全吻合

```
*(byte*)(GLOBAL_4c76 + 0xa6) = (byte)local_2a;              // count 存進 struct+0xa6
for (local_28=0; local_28 < FUN_206a_000a(local_2a); local_28++) {
    uVar1 = FUN_2f16_000f(local_1a, ...);
    *(byte*)(GLOBAL_4c76 + local_28 + 0x16) = uVar1;         // 每個怪物ID存進 struct+0x16+i
    ...
}
```

`FUN_206a_000a`（`decompiled/206a_000a_FUN_206a_000a.c`，反編譯顯示成 `void` 是 decompiler 對
far call + 暫存器回傳值型別判斷失敗的已知瑕疵，**看 disassembly 才準**）：

```
206a:000a  MOV AX,[BP+6]
206a:0011  CMP AX,0
206a:0015  JG (跳過negate)
206a:0017  NEG AX
206a:001c  ...RETF
```

這就是 `abs(x)`。**已驗證**：count 欄位允許帶負號（`FUN_2f16_000f` 支援 `-`），迴圈次數取絕對值。

`struct+0xa6`（count）與 `struct+0x16`（怪物 ID 陣列）這組 offset，在**完全獨立的另一個函式**
`FUN_17c5_000d`（`17c5:000d`，戰鬥初始化，1515 bytes）裡也出現同一組 offset 的讀寫：

```c
*(byte*)(GLOBAL_4c76 + 0xa6) = *(byte*)(*0x514e + iVar6);
for (local_6 = 0; local_6 < *(byte*)(GLOBAL_4c76+0xa6); local_6++) {
    *(byte*)(GLOBAL_4c76 + local_6 + 0x16) = *(byte*)(*0x514e + local_4);
    local_4 += 4;
}
```

兩個完全不同的函式，同一組 struct offset 扮演一致的角色（count / 怪物ID陣列），**交叉驗證了
`event-script.md` 已確認的「count = 怪物遭遇組數、id_1..id_count = MONSTER.DAT 索引」模型完全正確**，
且進一步確認這是遊戲內部固定的「目前戰鬥編組」資料結構，不是本分析臆測。

同時，`FUN_25be_0e77` 在 `count != 0` 分支會呼叫 `FUN_1d9f_0f80(0x15, 10, 0x2e91)`，
`0x2e91` 正是字串 `"Combat!"`（`strings.csv`: `31f0:2e91,7,Combat!`）——**已驗證**：進入有怪物的房間
會先印出「Combat!」橫幅，這也是 count 欄位確實是「遭遇組數」的直接旁證。

#### [E] 戰鬥設定欄位（doc 稱「255 終止符」）—— 推翻「搜尋終止符」假說

模擬結果直接推翻 `event-script.md` §4.3 的「是不是每筆都有 255 終止符」疑問：**根本沒有「搜尋 255」
這回事**。程式碼是**無條件讀取 ID 清單後緊接著的下一個欄位**，不管它的值是什麼：

```
DATA2.TXT it=0 (四隻狗頭人帳篷): C=4, ids=[26,26,26,26], E=1     <- 不是255！
DATA2.TXT it=3 (Uffuspgot首領戰): C=5, ids=[...],        E=2     <- 不是255！
DATA1.TXT it=30 (火焰房間):       C=1, ids=[46],          E=4     <- 不是255！
DATA5.TXT it=10 (惡魔橋頭戰):     C=3, ids=[53,53,46],     E=5     <- 不是255！
```

這些「非 255」的案例模擬器全部順利消耗、後續迭代完全不受影響，證明 E 欄位就是**固定 1 個 slot**，
值是什麼都合法，`255` 只是最常見的**預設值**，不是需要搜尋比對的哨兵。這也直接解釋
`event-script.md` §4.3 提出的「DATA1 #34、DATA2 多筆 4-kobold 記錄沒接 255」異常——**這些記錄根本沒
異常，它們只是 E 欄位剛好不是 255**。

`FUN_25be_0e77` 對 E 欄位的用法（`count != 0` 且進入戰鬥時）：

```c
if (local_28 < 0xff) {
    FUN_2f30_000e(local_16);   // 重建/重讀某個資料
    ...
    local_34 = 0x80;
} else {
    local_34 = 0;
    FUN_222f_2cf2(0x1d9f, *(int*)0x50f0 - 1, *(int*)0x50ee - 1);  // 見下
}
uVar3 = FUN_17c5_000d(local_34);   // 呼叫戰鬥初始化，帶 local_34 當 flag
```

`FUN_222f_2cf2`（`decompiled/222f_2cf2_FUN_222f_2cf2.c`）拿兩個座標參數（`param_2`, `param_3`，這裡
分別傳入「目前 X 座標 -1」「目前 Y 座標 -1」，`0x50f0`/`0x50ee` **推測**是隊伍目前地圖座標），
內部邏輯是在一個字元對照表（`0x2508`/`0x257e`）裡查值、寫進 `0x5210` 起的緩衝——具體用途受限於
時間未完全展開，但「傳入座標各減 1」這個呼叫模式，**在 1980 年代 CRPG 裡是典型的「戰鬥觸發後把隊伍
推回上一格」機制**（假設，中信心；沒有進一步反編譯 `FUN_222f_2cf2` 內部全部邏輯來 100% confirm
「推回」這個具體遊戲效果，只確認了它會被呼叫、傳入座標-1、且只在 `E==255` 時發生）。

`FUN_17c5_000d` 依 `local_34`（來自 E 是否 <0xff）分兩條完全不同路徑：`local_34 < 0x80` 時走一段
「用固定演算法程序化擺放怪物隊形」的路徑；`local_34 >= 0x80` 時走另一段「直接從緩衝區 `0x514e`
複製怪物編組資料到 `GLOBAL+0x16`」的路徑。**結論（已驗證分支存在，具體遊戲效果為推測，中信心）**：
E 欄位控制「這場戰鬥用自動生成的隊形，還是用預先準備好的特殊隊形（例如：狗頭人帳篷內、橋上的
Demon lord 戰、火焰房間的戰鬥，這些場景明顯需要跟一般走廊遭遇不同的排陣）」。這解釋了為什麼
「特殊場景」的戰鬥記錄（帳篷、橋、火焰房）E 值都不是預設的 255。

#### [F] 續接碼欄位（doc 稱「trailer 第一個數字」）—— 完整解開

```c
*(byte*)(GLOBAL_4c76 + 0xa5) = uVar1;   // uVar1 = FUN_2f16_000f(local_22), 存進 struct+0xa5
...
if (*(char*)(iVar2 + 0xa5) != 3) break;   // 若 != 3，跳出外層 while(true)，正常結束
param_2 = 1;                              // 若 == 3，帶 param_2=1 重新從迴圈頂端跑一次
...
// param_2==1 時的分支（在函式最開頭）：
FUN_25be_158e((char*)local_22 + 1, ...);  // 顯示 field F 指標+1（跳過開頭那個控制字元）的內容
*(byte*)(GLOBAL_4c76 + 0xa5) = 0;
return 2;
```

且函式緊鄰的前一段殘留碼（`25be:0e39`~`0e76`，疑似呼叫端／再入包裝，位址上緊接在
`FUN_25be_0e77` 之前但 `functions.csv` 沒有把它算進任何函式邊界——這段區域符合 `00-ghidra-setup.md`
提到的「間接跳轉/跳表沒被自動展開」情形）同樣檢查 `struct+0xa5 == 3`，是獨立的第二個佐證，
確認 `0xa5` 是一個會被明確檢查、有控制流意義的欄位，不是巧合。

**用模擬結果直接對照三種欄位 F 的實際內容，機制完全吻合**：

| DATA 檔 #迭代 | Field F 內容 | `FUN_2f16_000f` 解析結果 | 對應行為 |
|---|---|---|---|
| DATA1 it=0（一般房間） | `"1"` | 1 | 正常結束，`0xa5=1`，不觸發連鎖 |
| DATA1 it=11（Remondadin戰後） | `"3With Remondadin dead the black room is strangely silent..."` | **3**（解析在第一個非數字字元 `W` 停止） | 觸發連鎖：外層迴圈重跑一次，顯示 `local_22+1` = `"With Remondadin dead..."`（跳過開頭 `'3'`） |
| DATA1 it=24（鐘室） | `"%RING.BELL...AT.....MIDNIGHT"` | 0（`%` 非數字，直接回傳0，不觸發 `==3` 連鎖） | 另一條分支 `if (*local_22=='%')` 呼叫 `FUN_25be_18fa` 顯示符文 |

**這就是 `event-script.md` §2.5 說的「控制字元開頭的文字（`3`/`%` 前綴）」的真正機制**：這些「特殊
子型別記錄」根本不是獨立的一筆記錄，它們是**前一筆記錄的 field F 整段吞下去的內容**——因為 field F
本來就是「讀到下一個 NUL 為止的一整段文字」，而 `FUN_2f16_000f` 剛好會把它開頭的數字字元部分解析
成一個小整數（`3` 或 0），藉此讓同一個 generic parser 兼職當「續接旗標讀取器」用。這也解釋了
`event-script.md` §4 提到的 `DATA4 #4` 離群值 `[5]`：field F 內容以 `'5'` 開頭，解析出 5，
只是這個函式沒有為「5」寫特別分支（只認 3 和 `%`），所以看起來像沒被處理，但機制上完全一致，
不是異常。

`FUN_25be_18fa`（`decompiled/25be_18fa_FUN_25be_18fa.c`）反編譯確認是**符文字型繪製函式**：從
`param_1+1`（跳過開頭的 `%`）開始，逐字元轉換（`'.' → 0`，其他字元 `→ char-0x40`），
以 9 欄一列的網格座標（`local_6%9+1, local_6/9+1`）呼叫 `FUN_1d9f_1b3a` 畫出。**這解開
`event-script.md` §2.5 提出的「為什麼句子裡的空白被句點取代」疑問**：不是排版巧合，是**符文字型
的字碼表設計**——`'.'` 被選來當「空白 glyph」的專用替代字元，因為 ASCII 空白 (0x20) 若直接拿去算
`char-0x40` 會撞到別的合法 glyph 編號，用 `.` 明確標記「這格畫空白 glyph 0」。**已驗證**（有完整
反組譯佐證，逐字元轉換公式清楚）。

### 2.5 對 `event-script.md` 的修正總表

| doc 原敘述 | 本次分析結論 | 證據等級 |
|---|---|---|
| 「每個 `DATA*.TXT` 檔頭固定 `255`，用途未解」 | 不是檔案層級的 header，是第 0 筆記錄的「插圖 ID」欄位重用了同一套 parser，值恰好是預設值 255（無插圖） | 已驗證（反組譯+模擬） |
| 「trailer 是 0–3 個數字，語意未解」 | 不存在可變長度 trailer。固定是 2 個欄位：field E（戰鬥隊形選擇碼，doc 誤認的「255終止符」，其實不搜尋255）+ field F（續接碼／符文文字／反應文字共用同一 slot）。doc 觀察到的「trailer 第二個數字」其實是**下一筆記錄自己的插圖 ID 欄位**，跟目前這筆記錄無關 | 已驗證（反組譯+跨5檔案模擬100%消耗） |
| 「trailer 第二值可能是強制出口/傳送索引（DATA1 #10/#3、DATA3 #21、DATA4 #13，樣本太少）」 | 推翻。這幾筆的「第二個數字」在新模型下是**下一筆記錄的插圖ID**，跟出口/傳送完全無關；真正跟「移動」語意最接近的其實是 field E 控制的戰鬥隊形選擇分支（`E==255` 時呼叫的 `FUN_222f_2cf2` 疑似把隊伍座標退回一格），但那是「戰鬥後效果」不是「出口索引」 | 已驗證（field 對應關係）+ 推測（`FUN_222f_2cf2` 的確切遊戲效果） |
| 「`count`+ids 之後不是每筆都接 255，可能是資料不規則或輸入誤植」 | 推翻「不規則/誤植」的猜測。是正常設計：field E 本來就沒有「必須是255」的限制，非255值（1,2,4,5等）觸發「使用預先準備的戰鬥隊形」而非程序生成隊形 | 已驗證 |
| 「`DATA4 #4` 離群值 `[5]` 語意未解」 | 機制上與 `3`/`%` 前綴完全一致（field F 開頭數字被 `FUN_2f16_000f` 解析），只是這個函式沒有為值 5 寫特殊分支，不算離群，只是没處理 | 已驗證（機制）+ 推測（值5本身沒被此函式用到，可能給了別的呼叫端） |

### 2.6 仍未解的部分（誠實列出）

- `FUN_222f_2cf2` 與 `FUN_17c5_000d` 的完整內部邏輯沒有全部反編譯到底，field E 觸發的兩條分支
  （程序生成隊形 vs. 預存隊形）**確認分支存在**，但「预存隊形資料實際長什麼樣子、退回一格的具體
  視覺效果」未做動態驗證。
- Field E 除了 255（預設）和特殊場景值（1/2/4/5）以外，數值本身除了「決定走哪個分支」還有沒有其他
  用途（例如指向哪一種隊形模板），本次只確認 `< 0xff` 就走同一條分支，沒有逐值細分。
- `struct 0x4c76 + 0xa3`（檔案索引，1–5）、`+0xa5`（續接碼）、`+0xa6`（count）、`+0x16`（怪物ID陣列）
  這個全域狀態 struct 的其他欄位語意本次沒有系統性盤點，只確認了跟本次兩個問題直接相關的幾個。
- `*(int*)0x52f4`（迴圈上限，「要跳到第幾筆記錄」）跟地圖座標的對應關係——即 `event-script.md` §4
  第 6 點「記錄與地圖座標的對應」——本次沒有花時間往回追這個變數怎麼被設定，這是後續如果要接上
  `MAP*.MAP` 座標系統的建議切入點。

---

## 3. 問題 B：`town%d.dat` 讀取／城鎮選單函式

### 3.1 函式定位（已驗證，但 decompiler 輸出品質差）

`FUN_278d_0098`（`278d:0098`，1884 bytes）。**這個函式的反編譯結果品質很差**，開頭就有 Ghidra
自己的警告：

```
/* WARNING: Instruction at (ram,0x0002bd9b) overlaps instruction at (ram,0x0002bd9a) */
/* WARNING: Control flow encountered bad instruction data */
/* WARNING (jumptable): Unable to track spacebase fully for stack */
/* WARNING: Removing unreachable block (ram,0x0002bc23) */
```

反編譯出來甚至出現 `out(0x8b, ...)`、`in(0xa3)` 這種直接存取硬體 I/O port 的假指令、以及一個
12-case 的 `switch(local_5a)`——這些明顯是 **decompiler 把跳表資料誤判成程式碼**（`00-ghidra-setup.md`
與任務背景都預先警告過這個雷）。**本節分析完全繞開這段壞掉的反編譯，直接讀 `disassembly.asm`
原始指令**，這也是任務背景建議的作法。

檔名組字邏輯（`278d:0135` 附近，跟 `DATA%d.TXT` 那段幾乎同構）：

```
278d:0135  MOV AX,0x3055     ; "town%d.dat"
```

（往前幾行有 `278d:0021/0030 CALLF 0x3000:1884` 等前置呼叫，屬於同一段初始化，沒有再深入拆解，
不影響本節結論。）

### 3.2 城鎮選單建構邏輯（已驗證，逐行對照 disassembly）

從 `278d:0273` 開始是一段清楚、非跳表汙染的**線性選單建構程式碼**（不是 switch/jump table，是
連續的「印字串 + 存選單條目」區塊接龍），逐段對照如下：

```asm
278d:0273  MOV [BP-0x5c], 0             ; 選單索引 i = 0
278d:0278  PUSH DS
278d:0279  MOV AX,0x306e                ; "Go to marketplace"（無條件，沒有任何 CMP 判斷）
278d:027d  CALLF 0x1000:ee6e            ; 印字串
...                                      ; 存選單條目 choice[i]=0, 按鍵='M'(0x4d)
278d:02bc  LES BX,[0x5534]              ; 目前城鎮資料緩衝區指標
278d:02c0  CMP byte ptr ES:[BX+0x1ee],0 ; 讀 buffer+0x1ee
278d:02c6  JZ (跳過)                     ; 若為0，不加 Healers
278d:02c9  MOV AX,0x3080                ; "Healers"
...                                      ; choice[i]=1, 按鍵='H'(0x48)
278d:0310  CMP byte ptr ES:[BX+0x1ef],0 ; 讀 buffer+0x1ef
278d:0316  JZ (跳過)
278d:0319  MOV AX,0x3088                ; "Rest in the Inn"
...                                      ; choice[i]=2, 按鍵='R'(0x52)
278d:0360  CMP byte ptr ES:[BX+0x1f0],0 ; 讀 buffer+0x1f0
278d:0366  JZ (跳過)
278d:0369  MOV AX,0x3098                ; "Town guild"
...                                      ; choice[i]=3, 按鍵='T'(0x54)
278d:03ac  MOV AL, byte ptr ES:[BX+0x1f1]  ; 讀 buffer+0x1f1（不是布林，是數值！）
278d:03ba  TEST AX,AX ; JNZ 繼續 / JMP 跳過
278d:03c1  CMP AX,0xb                   ; 值==11 走特殊分支（table @0x2fe8）
278d:03c4  JNZ (一般分支：table[(value)*4 + 0x4c94])
278d:03f1  MOV AX,0x30a3                ; "Church of "（後面接組字串把城鎮名接上去）
...                                      ; choice[i]=4, 按鍵='C'(0x43)
278d:0586  MOV [BP-0x58],0              ; college 迴圈索引 j=0
278d:058f  MOV SI,[BP-0x58]
278d:0592  MOV AL, byte ptr ES:[BX+SI+0x1f2]  ; 讀 buffer+0x1f2+j（迴圈，3個槽）
278d:059c  CMP AX,0xff ; JNZ 繼續 / JMP 跳過該槽
278d:05a9  MOV AX,0x30cf                ; "%d) College of"
...                                      ; 迴圈到 j<3（對應找到的 "1) College"/"2) College"/"3) College" 字串）
278d:04a2  CMP byte ptr ES:[BX+0x1f5],0 ; 讀 buffer+0x1f5
278d:04a8  JZ (跳過)
278d:04ab  MOV AX,0x30b3                ; "Docks"
...                                      ; choice[i]=5, 按鍵='D'(0x44)
278d:04f2  CMP byte ptr ES:[BX+0x1f6],0 ; 讀 buffer+0x1f6
278d:04f8  JZ (跳過)
278d:04fb  MOV AX,0x30b9                ; "Pub"
...                                      ; choice[i]=6, 按鍵='P'(0x50)
278d:053f  MOV AX,0x30bd                ; "Inspect character"（無條件，選單固定最後一項）
...                                      ; choice[i]=7
```

**已驗證**（逐行對照反組譯位址、CMP/JZ 判斷邏輯、字串位址全部核對過）：城鎮選單由「目前城鎮
資料緩衝區」裡**固定絕對 offset** `0x1ee`、`0x1ef`、`0x1f0`、`0x1f1`、`0x1f2~0x1f4`（迴圈3槽）、
`0x1f5`、`0x1f6` 這幾個 byte 決定要不要顯示對應設施，**Marketplace 永遠顯示、Inspect character
永遠是選單最後一項**，兩者都不查任何旗標。

### 3.3 對回 `town-and-map.md` 已驗證的 17-byte/30-record 結構——關鍵發現

`town-and-map.md` §1.1 已驗證 `TOWN*.DAT` 是「30 筆 17-byte 記錄 + 檔尾 2 bytes」，即
record `i` 落在檔案 offset `[i*17, i*17+17)`。用這個已驗證的切法反查上面幾個絕對 offset：

```
0x1ee = 494 = 29*17 + 1   -> record 29（最後一筆，第30筆）的 payload byte 0
0x1ef = 495 = 29*17 + 2   -> record 29 的 payload byte 1
0x1f0 = 496 = 29*17 + 3   -> record 29 的 payload byte 2
0x1f1 = 497 = 29*17 + 4   -> record 29 的 payload byte 3
0x1f2~0x1f4 = 498~500     -> record 29 的 payload byte 4~6
0x1f5 = 501                -> record 29 的 payload byte 7
0x1f6 = 502                -> record 29 的 payload byte 8
```

**全部落在同一筆記錄：record 29（30 筆裡的最後一筆）的 payload 區段（byte 1–13）內。**
程式碼從頭到尾**沒有讀過 record 29 自己的 type_code（byte 0，file offset 493）**，也**完全沒有
迭代/比對其他 29 筆記錄的 type_code**——選單建構邏輯是直接用寫死的絕對 offset 存取緩衝區，
跟「type_code 分派」完全無關。

### 3.4 用實際 25 個 `TOWN*.DAT` 檔案交叉驗證（本次分析新增）

把 25 個城鎮檔案的 record 29 payload 全部 dump 出來對照：

```
 1 Seaside          healers=1 rest=1 guild=0 church=10 college=[255,255,255] docks= 0 pub= 1
 2 Elbarat          healers=1 rest=1 guild=0 church= 8 college=[23,255,255]  docks= 0 pub=12
 3 Akistu           healers=1 rest=1 guild=0 church= 5 college=[2,3,7]       docks= 0 pub= 6
 4 Alynhawk         healers=0 rest=1 guild=0 church= 0 college=[10,1,255]    docks= 0 pub= 0
 5 Paladine         healers=0 rest=1 guild=0 church= 0 college=[255,255,255] docks= 0 pub= 7
 6 Erguard          healers=0 rest=0 guild=1 church= 8 college=[255,255,255] docks= 0 pub= 0
 7 Janthrin         healers=1 rest=1 guild=0 church= 0 college=[255,255,255] docks=57 pub= 3
 8 New Gleon        healers=0 rest=0 guild=1 church= 0 college=[255,255,255] docks=61 pub= 0
 9 Myrquacid        healers=0 rest=1 guild=1 church= 4 college=[27,255,255]  docks= 0 pub= 0
10 Urlock           healers=0 rest=1 guild=0 church= 5 college=[255,255,255] docks= 0 pub= 0
11 Dragontooth      healers=1 rest=1 guild=1 church= 0 college=[2,3,255]     docks=67 pub= 8
12 Chandris         healers=1 rest=0 guild=0 church= 1 college=[9,255,255]   docks= 0 pub= 7
13 Iris             healers=1 rest=1 guild=1 church= 2 college=[255,255,255] docks= 0 pub= 0
14 Asaht            healers=1 rest=1 guild=1 church=11 college=[23,255,255]  docks=68 pub= 2
15 Irondome         healers=0 rest=0 guild=0 church= 0 college=[255,255,255] docks= 0 pub= 0
16 Ynoth            healers=0 rest=1 guild=0 church= 9 college=[23,26,255]   docks= 0 pub= 9
17 Aurora           healers=1 rest=1 guild=1 church= 0 college=[25,24,255]   docks= 0 pub= 0
18 Woodhaven        healers=1 rest=1 guild=1 church= 8 college=[1,23,7]      docks= 0 pub= 0
19 Idlewood         healers=0 rest=1 guild=0 church= 1 college=[11,255,255]  docks= 0 pub= 4
20 Terlabba         healers=1 rest=1 guild=1 church= 5 college=[10,255,255]  docks= 0 pub= 0
21 Loven            healers=0 rest=1 guild=0 church= 0 college=[3,2,255]     docks= 0 pub= 0
22 Mojured          healers=0 rest=1 guild=0 church= 0 college=[22,255,255]  docks= 0 pub= 5
23 Lumisle Island   healers=0 rest=1 guild=1 church= 4 college=[9,255,255]   docks= 0 pub= 0
24 Land's Edge      healers=1 rest=0 guild=1 church=10 college=[24,255,255]  docks=60 pub=10
25 Pirate's Cove    healers=0 rest=1 guild=1 church= 0 college=[255,255,255] docks= 0 pub=11
```

重跑指令（純標準庫，讀 `workplace/orig/demwin/DEM_DATA/TOWN*.DAT` + `TOWN.TXT`，邏輯就是
「取 record 29 = byte[493:510]，payload = byte[1:14]」，可直接照著 §3.3 的 offset 換算複製貼上重寫）。

**吻合度**：
- `healers`、`rest`、`guild` 三個欄位在全部 25 個城鎮裡**嚴格只出現 0 或 1**，跟反組譯看到的
  `CMP byte,0 / JZ` 布林判斷完全對應。**已驗證**。
- `church` 欄位範圍落在 **0–11**，跟反組譯 `CMP AX,0xb`（11）的特殊分支邊界完全對應
  （有城鎮出現剛好 11，如 Asaht；也有城鎮出現 0–10 之間各種值）。**已驗證**。
- college 三槽多數是 `255`（無此槽），非 255 時是 1–27 左右的小整數，跟反組譯迴圈裡
  `CMP AX,0xff / JNZ` 的判斷完全對應。**已驗證**存在此欄位且觸發條件吻合；欄位值本身指向哪個
  具體學院資料（`table @0x553a` 的內容）本次沒有展開。
- `docks`、`pub` 在有些城鎮是 0（不顯示），有些是十位數到大約 68 的正整數——跟反組譯的
  `CMP byte,0 / JZ` 存在性判斷吻合（非0就顯示），但**數值本身除了「非0=顯示」之外還代表什麼
  （地點座標？特殊事件 ID？）本次未解**。

### 3.5 對 `town-and-map.md` 的修正 —— 推翻原問題前提

`town-and-map.md` §1.2、§1.5、§5-1 提出的問題是：「28 種相異 type_code（跨 30 筆記錄 × 25 城鎮）
對不上 7 個設施字串，需要反組譯找 dispatch table」。

**本次結論：這個問題的前提不成立，不存在「type_code → 設施」的 dispatch。**

- 7 個設施選單完全由 **record 29（固定位置，不是靠 type_code 找出來的）** 的 payload bytes
  （offset 1–13 內的特定子欄位）決定，跟其餘 29 筆記錄的 type_code 完全無關。
- doc 觀察到的「28 種 type_code」，本次**完全沒有被這個選單函式讀取或使用**——這代表那 28 種
  type_code（分布在 records 0–28）是驅動**別的東西**（`town-and-map.md` 自己推測的「NPC/招募名單」
  或其他事件觸發，見該文 §1.4），不是設施選單。**這部分本次分析沒有解開，仍是未解問題**，
  但至少排除了「跟這 7 個設施字串有關」這個假說方向。
- `town-and-map.md` 提出的「Iris（13）、Land's Edge（24）type-A 記錄數為 0，可能是小型據點」這類
  基於 type_code 統計的推論，跟本次發現的設施選單機制**無直接關係**（那些統計是關於 records 0–28，
  本次分析的機制只涉及 record 29），不受本次結論影響，也未被推翻或證實，維持該文原本的假設狀態。

### 3.6 仍未解的部分（誠實列出）

- **records 0–28（每城鎮 29 筆）的 type_code 意義完全未解**——本次分析找到的函式根本不讀它們。
  下一步應該用類似的字串錨定法（例如找「NPC 對話」、「招募」相關的字串或 `4c76`/`4c7e` 這類反覆
  出現的全域 struct 存取模式）去找真正處理這 29 筆記錄的函式。
- Church 欄位值（0–11）對應的座標表 `0x4c94`（一般分支）與 `0x2fe8`（值11特殊分支）的實際內容
  沒有 dump 出來核對，只確認了分支存在、邊界值（11）吻合。
- College 三槽（payload byte 4–6）的數值指向的 `table @0x553a` 內容沒有展開，只確認「非255觸發
  College選單項目」這個存在性層面的機制。
- Docks／Pub 欄位除了「非0=顯示」之外，數值本身（如 Elbarat 的 pub=12、Dragontooth 的 docks=67）
  在後續程式碼裡怎麼被用（是否跟 Church 一樣是座標表索引），本次沒有追完整——已知這兩個判斷式
  之後的程式碼在函式更後段（`278d:04fb` 之後），受時間限制沒有逐行展開，但看得出結構跟前面幾個
  一致（先印字串、再存 choice 條目），**推測**（低-中信心）也是類似的「附加數值」欄位而非純布林，
  因為原始資料裡數值明顯不只 0/1。
- Record 29 自己的 type_code（byte 0，file offset 493）本次分析發現這個選單函式完全不讀它，
  它的值（本次 dump 顯示落在 8–25 之間，跟其他記錄的 type_code 值域重疊）用途仍未知——可能是
  Town Maker 編輯工具留下的無意義殘留（呼應 `town-and-map.md` §1.3 的 ELRIC 樣板殘留現象），
  也可能被別的函式用到，未驗證。

---

## 4. 給後續 agent 的建議

1. **問題 A 已經解到可以直接更新 `event-script.md` 的程度**（此文件內容需要 repo 負責人裁決是否
   採納 §2.5 的修正表）。剩下值得深挖的是 `FUN_222f_2cf2`／`FUN_17c5_000d` 的隊形資料內容，如果
   要做「事件 → 戰鬥」的忠實重現會需要。
2. **問題 B 需要換方向**：不要再找「type_code dispatch table」，改找「誰在處理 records 0–28」。
   建議：先看 `town-and-map.md` §1.3 提到的 ELRIC 樣板殘留、NPC 陣列假說相關的字串（例如查
   `strings.csv` 有沒有 "recruit"、"join"、角色職業相關字串），用同樣的字串錨定法找函式；或者
   直接在 `FUN_278d_0098` 的其餘 1884 bytes 裡（本次只完整解讀了 `0x273`–`0x582` 這段選單建構，
   函式其餘部分含被 decompiler 誤判的跳表區段）用 disassembly 逐段核對，可能其他子邏輯有處理到
   records 0–28。
3. `FUN_278d_0098` 反編譯品質差是真實限制，不是分析疏漏——後續要展開它剩下的部分，一律用
   `disassembly.asm` 對照，不要相信 `decompiled/278d_0098_*.c` 的內容（尤其那個 12-case
   switch 和硬體 I/O port 呼叫，已確認是誤判）。
4. 本文 §2.3 的 Python 模擬器驗證手法（把反組譯推出的欄位消耗順序，拿真實資料檔跑一遍看能不能
   100% 消耗完）是本次最有效的驗證工具，比單純讀反編譯 C 碼可靠，因為它會在「理解錯欄位順序」時
   立刻暴露（消耗不完或提前結束）。建議後續分析其他資料檔的 parse 邏輯時比照辦理。
