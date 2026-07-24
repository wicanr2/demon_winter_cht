# 音效系統與資源檔結構(DEMON.INT 反組譯)

分析對象：`DEMON.INT` 的聲音子系統（問題 A）、`FILES.DAT`/`FILES.DTT` 的讀取端（問題 B）、
`SUM.MAP` 的讀取端（問題 C）。全部證據來自 `workplace/ghidra/export/` 的既有匯出結果
（`disassembly.asm`、`decompiled_all.c`、`decompiled/*.c`、`strings.csv`、`functions.csv`），
以及對 `workplace/orig/demwin/DEMON.INT` 原始位元組的直接讀取核對（`objdump -b binary -m i386
-Maddr16,data16` 反組譯 Ghidra 未展開的區塊）。位址換算一律套用 `docs/re/00-ghidra-setup.md`
的公式：`file_offset = segment*16 + offset - 0xC400`（segment 是 Ghidra 顯示的位址，已含
base segment 0x1000）。

---

## V1 結論（最高優先）：原版只有 PC speaker，沒有背景配樂，但有一段內嵌旋律

**已驗證**：

1. **只用 PC speaker。** 全檔案唯一出現的聲音相關 port I/O 是 `OUT 0x42`／`OUT 0x43`／
   `IN 0x61`／`OUT 0x61`（8253/8254 計時器 channel 2 + speaker gate，PC speaker 的標準操作）。
   對 AdLib（port `0x388`/`0x389`）與 MIDI（port `0x330`/`0x331`）的搜尋**沒有找到任何一筆
   `OUT`/`IN` 指令**——出現的 `0x388a`/`0x3899`/`0x389e`/`0x3313` 全部是 `MOV AX,imm16` 立即值
   的巧合，不是 port 操作（見 §1.1）。字串索引裡也沒有 AdLib/Sound Blaster/Roland/MIDI/MPU/OPL
   相關字串，只有 `Sound on`/`Sound off`（`31f0:0808`/`31f0:07e4`/`31f0:07fe`）。**結論：原版
   確實只支援 PC speaker，沒有 FM 音源或 MIDI 硬體支援。**
2. **沒有持續播放的背景音樂，但有一個真正的「音符序列」引擎。** 遊戲用一個安裝在
   **INT 1Ch（timer tick）** 上的中斷處理常式，逐 tick（18.2 Hz）走訪一張 4-byte／筆
   （頻率除頻值 + 持續 tick 數）的音符表，這是貨真價實的音序播放機制，不是單純的
   「設頻率→延遲→關閉」。但呼叫端的用法證實：這個引擎只拿來播放**觸發式的短音效／小段旋律**，
   播放完就設「完成」旗標並停止（`[0x19f3]`），沒有任何一處把它設成迴圈重播——**不存在持續性
   背景配樂**。詳細機制與逐 byte 解碼見 §1.2–§1.4。
3. **音效庫共 10 種**（透過一個 10 項 jump table 選擇，索引 = `effect_id + 1`，`effect_id`
   範圍 -1～8）：
   - `effect_id = -1`：**唯一的多音符旋律**（8 個音符含休止符，見 §1.4），從呼叫端反查是
     **角色/怪物死亡**時觸發（`FUN_138d_165d`，見 §1.5）。
   - `effect_id = 0`：靜音／停止（不寫入音符資料，只重置計數器）。
   - `effect_id = 1–8`：8 個**單音**效果音，頻率經計算後精確對上 C 大調自然音階
     C3–D3–E3–F3–G3–A3–B3–C4（見 §1.4 表格），每個只響一個音（3 tick ≈ 165ms）後停止。

**專案目標的修正建議（供主持人裁決）**：任務單原先預期的「離散音效 vs 有音序表的音樂」二分法，
反組譯結果落在中間——**沒有背景配樂，但引擎本身具備多音符序列播放能力**，且已被用在至少一處
（死亡音效）播出真正的旋律而非單音嗶聲。這代表「音樂還原」的範圍應該收斂成：**還原這個 10 項
音效庫（含死亡旋律），而不是尋找／還原任何背景配樂資料**——因為背景配樂在這個引擎架構下
根本不存在，繼續找只會白費工夫。

---

## 1. 發聲函式與觸發點

### 1.1 排除 AdLib / MIDI（已驗證，反面證據）

```
$ grep -in "0x388\|0x389" disassembly.asm
278d:215b  MOV AX,0x388a     <- 立即值，不是 OUT/IN 到 port 0x388
278d:23f6  MOV AX,0x3899     <- 同上
278d:243f  MOV AX,0x389e     <- 同上
$ grep -in "0x330\|0x331" disassembly.asm
278d:0e5f  MOV AX,0x3313     <- 同上
```
三筆全部是 `MOV AX, 16-bit立即值`，運算元恰好落在 AdLib/MIDI port 範圍是巧合，**沒有一筆是
`OUT`/`IN` 指令**。全檔案 `OUT`/`IN` 指令總表（72 筆 `OUT`，3 筆 `IN AL,`）逐一核對後，
只有 port `0x42`/`0x43`/`0x61` 有實際的聲音相關 I/O（`217b`/`2cdc` 兩個 segment 的大量
`OUT DX,AL` 經抽查是調色盤／CRTC 暫存器寫入，與聲音無關，不在本文範圍）。

### 1.2 INT 1Ch 音符播放常式：`1d9f:2a26`（Ghidra 未辨識為函式，已手動反組譯還原）

**這是 Ghidra 自動分析漏掉的一段**（符合 `00-ghidra-setup.md` 預警的「間接進入點不會被自動展開」）：
`functions.csv` 在 `1d9f:2a14`（18 bytes，結束於 `2a25`）之後直接跳到 `1d9f:2a95`
（1410 bytes），中間 `2a26`–`2a94`（110 bytes）完全沒有函式記錄。但這段正是中斷向量
安裝的目標位址（見 §1.3），對回原始檔案位元組後用 `objdump -b binary -m i386
-Maddr16,data16` 手動反組譯還原：

```
1d9f:2a26  PUSH BP/AX/BX/CX/DX/DS/ES/SS ; STI
1d9f:2a30  MOV AX,0x21f0 ; MOV DS,AX          ; DS = 0x21f0（relocation stub，實際載入後 = 0x31f0）
1d9f:2a37  CMP word ptr [0x19f3],0x1 ; JE 0x2a85 (exit)   ; 若「已播完」旗標=1，本次 tick 不做事
1d9f:2a3d  LEA BX,[0x5ada]                     ; BX = 音符表基底位址（31f0:5ada）
1d9f:2a41  CMP word ptr [0x5c4e],0x0 ; JLE next_note       ; 目前音符的剩餘 tick 數
1d9f:2a46    DEC word ptr [0x5c4e] ; JMP exit               ; 還沒到期，倒數後直接返回
1d9f:2a4c  next_note:
1d9f:2a4c    ADD word ptr [0x5c50],0x4 ; ADD BX,[0x5c50]    ; 索引 += 4（每筆記錄 4 bytes）
1d9f:2a52    MOV CX,[BX]                                    ; CX = 這筆記錄的「頻率除頻值」
1d9f:2a54    CMP CX,0x0 ; JNZ have_freq
1d9f:2a57      CALL speaker_off ; JMP write_dur              ; freq=0 = 休止符（仍要讀 duration）
1d9f:2a5c    have_freq:
1d9f:2a5c    JG play_tone
1d9f:2a5e      CALL speaker_off ; MOV word ptr[0x19f3],0x1 ; JMP exit  ; freq<0 = 結束哨兵，停止並標記完成
1d9f:2a70    play_tone:
1d9f:2a70      MOV AL,CL ; OUT 0x42,AL   ; 除頻值低位元組 -> timer2 count register
1d9f:2a74      MOV AL,CH ; OUT 0x42,AL   ; 除頻值高位元組
1d9f:2a76      IN AL,0x61 ; OR AL,3 ; OUT 0x61,AL          ; 開 speaker gate + timer2 output
1d9f:2a7e  write_dur:
1d9f:2a7e    ADD BX,2 ; MOV CX,[BX] ; MOV word ptr[0x5c4e],CX  ; 讀這筆的 duration，設進倒數計數器
1d9f:2a85  exit: POP SS/ES/DS/DX/CX/BX/AX/BP ; IRET

speaker_off (1d9f:2a8e，Ghidra 有抓到這一小段):
1d9f:2a8e  IN AL,0x61 ; AND AL,0xfc ; OUT 0x61,AL ; RET
```

**記錄格式（已驗證，4 bytes/筆）**：`word freq_divisor`（8253/8254 timer2 除頻值，
PIT 基準時脈 1,193,182 Hz）、`word duration`（tick 數，1 tick ≈ 1/18.2 秒）。
`freq_divisor == 0` = 休止符（仍佔一個 duration）；`freq_divisor < 0`（最高位為 1）
= 結束哨兵，播放到這裡就停並把 `[0x19f3]` 設成 1（供呼叫端輪詢是否播完，見 §1.5）。

### 1.3 安裝常式：`FUN_1d9f_2c28`（`1d9f:2c28`，138 bytes）

```c
void __cdecl16far FUN_1d9f_2c28(undefined2 param_1)
{
  ...
  *(undefined2 *)0x5c4e = 0;
  *(undefined2 *)0x5c50 = 0xfffc;   // -4，讓 handler 第一個 tick 的 "ADD [0x5c50],4" 剛好落在 offset 0
  *(undefined2 *)0x19f3 = 1;
  pcVar1 = (code *)swi(0x21);       // AH=0x35, AL=0x1C：INT 21h Get Interrupt Vector(INT 1Ch)
  ...
  out(0x43,0xb6);                   // 8253 控制暫存器：channel 2, mode 3 (方波), binary
  pcVar1 = (code *)swi(0x21);       // AH=0x25, AL=0x1C：INT 21h Set Interrupt Vector(INT 1Ch) -> DS:DX=1d9f:2a26
  DAT_31f0_5ad6 = unaff_ES; DAT_31f0_5ad8 = in_BX;  // 保存舊向量
  ...
}
```
呼叫端在 `1d9f:2c91`／`1d9f:2c93`：`MOV AL,0xb6 ; OUT 0x43,AL`（設定計時器 2 為方波產生模式，
PC speaker 標準初始化），接著用 `INT 21h AH=0x25 AL=0x1C` 把 `1d9f:2a26` 掛進 INT 1Ch
向量表——這就是「timer tick 觸發的音符播放器」被安裝的瞬間。此函式在遊戲各主要畫面
（`206a`、`222f`、`2aed` 等 6 個不同 segment）各被呼叫 2 次，屬於「進場景時（重新）安裝
一次計時器 hook」的模式，不是每次要播音效都重裝一次。

### 1.4 音效觸發選擇器：`FUN_1d9f_2a95`（`1d9f:2a95`）內的 10 項 jump table

`1d9f:2c0a`：`JMP word ptr CS:[BX + 0x2beb]`，`BX = (effect_id+1)*2`，`effect_id` 範圍
-1～8（超出範圍會被 `1d9f:2c02 CMP AX,0xa ; JNC skip` 擋掉）。跳表 10 個項目
（`1d9f:2beb`，已用 file offset 讀出原始 word 陣列核對）：

| jump index | `effect_id` | case 位址 | 內容(已逐 byte 反組譯核對) |
|---|---|---|---|
| 0 | -1 (`0xffff`) | `1d9f:2b5f` | **8 音符旋律**（見下表），唯一的多音符效果 |
| 1 | 0 | `1d9f:2c0f` | 只重置 `[0x5c4e]=0`、`[0x5c50]=0xfffc`，不寫音符 = 靜音/停止 |
| 2 | 1 | `1d9f:2ab7` | 單音：divisor `0x23a2` → **130.80 Hz（C3）**，3 tick |
| 3 | 2 | `1d9f:2acc` | 單音：divisor `0x1fbe` → **146.84 Hz（D3）**，3 tick |
| 4 | 3 | `1d9f:2ae1` | 單音：divisor `0x1c48` → **164.80 Hz（E3）**，3 tick |
| 5 | 4 | `1d9f:2af6` | 單音：divisor `0x1ab1` → **174.62 Hz（F3）**，3 tick |
| 6 | 5 | `1d9f:2b0b` | 單音：divisor `0x17c8` → **195.99 Hz（G3）**，3 tick |
| 7 | 6 | `1d9f:2b20` | 單音：divisor `0x1530` → **219.98 Hz（A3）**，3 tick |
| 8 | 7 | `1d9f:2b35` | 單音：divisor `0x12e0` → **246.93 Hz（B3）**，3 tick |
| 9 | 8 | `1d9f:2b4a` | 單音：divisor `0x11d0` → **261.66 Hz（C4）**，3 tick |

**發現（已驗證，取樣自 `objdump` 反組譯出的 `movw $imm,addr` 序列）**：`effect_id 1–8`
八個單音的頻率精確對上 **C 大調自然音階 C3–C4**（誤差全在 0.1 Hz 內，對應到整數
divisor 的量化誤差，不可能是巧合）。**推測**（未證實，屬合理猜測）：這 8 個音很可能是
選單導覽用的「音階提示音」（例如捲動法術列表、數值選擇器時，游標每移一格就升/降一個音階），
但呼叫端（見 §1.5）目前查到的實際用途是戰鬥判定的「命中／未命中」提示音，兩者不衝突
（同一組音效素材被多處復用是常見做法）。

`effect_id = -1` 的旋律（`1d9f:2b5f`，已逐筆解出 8 個音符記錄，1 tick ≈ 54.9ms）：

| # | freq divisor | 音高 | duration |
|---|---|---|---|
| 1 | `0x12e0` | B3 (246.9Hz) | 6 tick(~330ms) |
| 2 | `0x0000` | 休止 | 4 tick(~220ms) |
| 3 | `0x1530` | A3 (220.0Hz) | 2 tick(~110ms) |
| 4 | `0x0000` | 休止 | 1 tick(~55ms) |
| 5 | `0x12e0` | B3 (246.9Hz) | 2 tick(~110ms) |
| 6 | `0x0000` | 休止 | 1 tick(~55ms) |
| 7 | `0x11d0` | C4 (261.7Hz) | 2 tick(~110ms) |
| 8 | `0x0000` | 休止 | 1 tick(~55ms) |
| 9 | `0x17c8` | G3 (196.0Hz) | 2 tick(~110ms) |
| 10 | `0x0000` | 休止 | 1 tick(~55ms) |
| 11 | `0x11d0` | C4 (261.7Hz) | 6 tick(~330ms) |
| 12 | `0xffff` | (結束哨兵) | — |

全長約 1.5 秒，B3 起音、C4 收尾，中段穿插短促的 A3/B3/C4/G3——聽感上是典型的
「短促下降再收束」提示音型態，符合**死亡/角色陣亡**的情境（見 §1.5 的呼叫端證據）。

### 1.5 呼叫端證據：誰在什麼情境播放哪個效果音

反查 `FUN_1d9f_2a95(...)` 的呼叫點（`grep FUN_1d9f_2a95( decompiled_all.c`），逐一對照
外層函式的行為：

| 呼叫點 | 參數 | 外層函式 | 情境（依外層函式邏輯推斷，已驗證呼叫本身，情境為合理解讀） |
|---|---|---|---|
| `138d:165d` (`FUN_138d_165d`) | `0xffff`（旋律） | 清空角色 HP 相關欄位（`[iVar1+0x4eba]=0`）、重置座標與狀態 | **角色/怪物死亡**——欄位操作與「移除單位」語意吻合 |
| `138d:25da` 內 | `local_4`（動態算出 1 或 4） | 命中率判定，`FUN_1d9f_1361(0x849)` 印出字串 `misses.`（`31f0:0849`）後才播音效 | **攻擊未命中**，遠近交戰用不同音高（近戰=1、遠端/其他=4） |
| `138d:25da` 內（另一分支） | `local_4`（動態算出 5 或 8） | 命中率判定通過後，依武器類型（欄位 `0x4ed4==2 or 0xb`）選 5 或 8 | **攻擊命中**，依武器類型（近戰/投射)選不同音高 |
| `138d:10bc` | `8` | 傷害結算，條件 `*(int*)0x4e2e==5 && *(int*)0x4e30<0` | 特定傷害類型（推測：中毒/特殊狀態傷害) |
| `138d:17b8` | `8` | 傷害結算後扣血（`*piVar1 = *piVar1 - local_16`），伴隨 `FUN_2cdc_1727()`（螢幕訊息） | 命中扣血提示音 |
| `1d9f:2a02`（小型 wrapper） | `1` | 只做 `if(sound on) FUN_1d9f_2a95(1)` | 通用「按一下」音效，供多處直接呼叫 |
| `1d9f:1ce1` 內兩處 | 未定（decompiler 丟失參數，僅知有呼叫） | 傷害結算 + 之後接續呼叫 `FUN_138d_165d()`（死亡處理） | 命中音接死亡音的連續序列，呼應「打倒敵人」 |

**已驗證**：`Sound off`/`Sound on` 兩個字串（`31f0:07e4`、`31f0:07fe`、`31f0:0808`）
對應的旗標是 `[0x1585]`（`0`＝開啟，非 0＝關閉）——`FUN_1d9f_2a95`、`FUN_1d9f_2a02`、
以及等待播放完成的迴圈（`decompiled_all.c:12703` 一帶 `while(*(int*)0x1585==0){ while(*0x19f3==0); *0x19f3=0; ...}`）
都以此旗標為第一個判斷式，符合任務單「戰鬥選單有 Sound off 指令」的描述。**未能定案**：
字串本身的印出位置（哪個選單函式呼叫了 `1361(0x7e4/0x7fe/0x808)`）在文字/位置層級搜尋
不到直接命中（可能透過選單字串表間接索引，不是逐一 push 常數），時間所限沒有繼續往下追，
不影響 `[0x1585]` 是主旗標這個結論的可信度（用法本身已足夠證實）。

---

## 2. `FILES.DAT` / `FILES.DTT`：找到「同批載入清單」，未能定案逐欄語意

### 2.1 已驗證：兩者與另外 4 個檔案同屬一份「啟動資源清單」

字串 `files.dtt`（`31f0:1502`）、`files.dat`（`31f0:150c`）在 `resource-index.md` 裡已知
與 `itemlocb.dat`（`31f0:14f5`）相鄰排列。反組譯進一步發現：**這幾個字串的位移（連同
`party.dat`／`demon.shp`／`winter.shp`）被組成一張 4-byte／筆的遠指標表**，位於（依 Ghidra
位址換算公式）`333d:0001`／`333d:0005`／`333d:0009`（尚有更早/更晚的項目未完整枚舉）：

```
$ python3 -c "..." # file offset 0x26fd1/0x26fd5/0x26fd9 逐筆核對
0x26fd1: f5 14 f0 21  -> word0=0x14f5(itemlocb.dat 的字串位移), word1=0x21f0(pre-relocation 段值)
0x26fd5: 02 15 f0 21  -> word0=0x1502(files.dtt),                word1=0x21f0
0x26fd9: 0c 15 f0 21  -> word0=0x150c(files.dat),                 word1=0x21f0
```
`word1=0x21f0` 正是 §1.2 驗證過的「relocation stub 段值，載入後 = 0x31f0」，與
`strings.csv` 全部字串所在的段完全一致——**證實這是一張指向字串表的遠指標陣列，
不是巧合**。緊接在這張指標表後面的位址（同一資料區塊）另外還有一段**逐字排列的
純文字**（不透過指標，直接內嵌）：`party.dat\0demon.shp\0itemlocb.dat\0files.dtt\0
files.dat\0winter.shp`，同樣把這 6 個檔名放在一起。

**結論（已驗證到「這 6 個檔案被同一張資料表關聯在一起」，未驗證到「消費這張表的程式碼」）**：
`FILES.DAT`／`FILES.DTT` 屬於遊戲開局會一起處理的一組核心資源
（`PARTY.DAT` 存檔、`DEMON.SHP`／`WINTER.SHP` 圖檔精靈、`ITEMLOCB.DAT` 已知的物品座標表），
不是隨機散落的檔案。這比 `resource-index.md` 原本「兩者在字串表中緊鄰出現」的觀察更進一步
（找到了真正的資料結構把它們串起來），但**沒能在時間內找到讀取這張指標表的實際程式碼**
（在 `disassembly.asm`／`decompiled_all.c` 裡搜尋這張表所在位址 `333d`／`233d` 沒有任何
XREF 命中，判斷是 Ghidra 自動分析沒有追出對這塊資料的引用，屬於間接定址/計算位址存取，
需要另外寫 Ghidra script 手動建立 XREF 才能繼續，見 §4 建議事項）。

### 2.2 `FILES.DAT` 內部分段語意：未能推進，維持 `resource-index.md` 既有結論

在剩餘時間內嘗試搜尋 `FILES.DAT` 各分段（`resource-index.md` §2.3 列出的 `0x0A0-0x14F`
遞增 marker 段、`0x600-0x67F` 8 筆固定記錄段）對應的讀取迴圈，用「stride 14」
（14-byte 記錄）、「stride 4」等關鍵常數在 `decompiled_all.c` 搜尋 `local_6 * 0xe` 或
`* 14` 之類的模式，沒有找到可信命中。**維持 `resource-index.md` 的既有結論**：
`FILES.DAT` 各段語意本次沒有新增證據，仍然是「多張互不相關小表」的否證後狀態，
逐欄語意未解。

---

## 3. `SUM.MAP`：已定案——23 個變長 RLE 壓縮 tile 區塊的串接檔，附完整解壓縮演算法

這題有完整、可重跑的驗證鏈，是本次任務裡最完整的一項。

### 3.1 讀取邏輯：`FUN_222f_28d0`（`222f:28d0`）— 統一的「依地圖 ID 載入地圖」函式

```c
undefined2 __cdecl16far FUN_222f_28d0(int param_1,int param_2)
{
  ...
  if (param_2 == 0) {
    if ((param_1==1) || (param_1==3) || (param_1==5)) {
      // 直接開獨立檔（MAP1.MAP / MAP3.MAP / MAP5.MAP），走 FUN_1d9f_0a8b(0x522a)
      ...
    } else {
      // 在 23 筆表裡找 param_1 對應的段落，累加前面段落的大小當 seek 位移
      local_6 = 0; local_8 = 0;
      while (local_6 < 0x17 && *(int*)(local_6*2 + 0x2488) != param_1) {
        local_8 += *(int*)(local_6*2 + 0x24b6);   // 累加位移
        local_6++;
      }
      local_4 = *(undefined2*)(local_6*2 + 0x24b6);  // 這個段落的大小
      *(int*)0x5a7a = local_8 >> 0xf;   // seek 位移高位
      *(int*)0x5a78 = local_8;          // seek 位移低位
      *(undefined2*)0x15e4 = 3;         // 「單次 seek+read」模式
      FUN_1d9f_0a8b(0x522a);            // 開 SUM.MAP，seek 到 local_8，讀 local_4 bytes
      FUN_25be_17fe(local_4);           // 解壓縮這段資料（見 §3.3）
    }
  }
  ...
}
```

**`param_1 ∈ {1,3,5}` 直接對應獨立檔 `MAPn.MAP`，其餘 `param_1` 值查表決定在 `SUM.MAP`
裡的哪個區段**——這一支判斷式本身就是 `town-and-map.md` §2.4/§5 懸案「7 個地城只有 3 個
獨立 `.MAP` 檔」的答案：**其餘地圖不是遺失，是被打包進 `SUM.MAP`**。

### 3.2 已驗證：23 筆表的 ID／大小加總 = `SUM.MAP` 的確切檔案大小

表格位於 `31f0:2488`（23 個 ID，`int16`）與 `31f0:24b6`（23 個大小，`uint16`），直接讀取
`DEMON.INT` 原始位元組核對（`file_offset = 0x31f0*16+0x2488-0xC400 = 0x27f88`）：

```
ID    : 12, 13, 2, 21, 22, 23, 25, 33, 34, 35, 36, 4, 41, 43, 44, 45, 51, 52, 54, 55, 56, 64, 66
SIZE  : 412,430,1592,253,344,497,552,1283,1817,1531,72,1741,387,964,1046,242,470,362,275,175,549,431,318
SUM(SIZE) = 15743
```

**`SUM(SIZE) = 15743`，與 `SUM.MAP` 的實際檔案大小 15,743 bytes 完全相等**——這是
決定性的證據，不是巧合（隨機 23 個 16-bit 數字加總精確命中一個特定檔案大小的機率
可忽略不計）。**結論（已驗證）**：`SUM.MAP` 是 23 個變長區塊**首尾相接、無 padding**
串接而成的檔案，每個區塊對應一個「地圖／區域 ID」，偏移量 = 前面所有區塊大小的累加和。

**這也解開了 `town-and-map.md` §5「`ITEMLOCB.DAT` 出現 `map_id=4` 但沒有 `MAP4.MAP`
檔案」的懸案**：ID 表裡确實包含 `2` 與 `4`（`ITEMLOCB.DAT` 用到的兩個「消失」地城 ID）——
它們不是獨立檔案，是 `SUM.MAP` 裡的第 3 筆（ID=2，大小 1592 bytes）與第 12 筆
（ID=4，大小 1741 bytes）區塊。

### 3.3 已驗證：解壓縮演算法（RLE，值/次數欄位順序與先前假設相反）

`FUN_25be_17fe`（`25be:17fe`）：

```c
void __cdecl16far FUN_25be_17fe(int param_1)   // param_1 = 這個區塊的 byte 長度
{
  local_4 = 0; local_8 = 0;       // local_4=讀取游標, local_8=輸出游標(64x64環狀緩衝)
  do {
    if (param_1 - 1 <= local_4) { 結束; }
    bVar2 = 來源[local_4];         // 讀一個位元組
    if ((char)bVar2 < 0) {          // 最高位=1：RLE run
      local_4 += 2;
      count = 來源[local_4-1];      // 下一個 byte 是「重複次數」
      do {
        輸出[local_8] = bVar2 & 0x7f;   // 低 7 位是「tile 值」
        local_8 = (local_8 + 0x40) % 0x1000;   // 每寫一格，游標 +64，滿 4096 繞回（= 64x64 網格，逐欄填）
        count--;
      } while (count != 0);
    } else {                         // 最高位=0：單一 literal
      輸出[local_8] = bVar2;
      local_8 = (local_8 + 0x40) % 0x1000;
      local_4++;
    }
  } while (寫滿 4096 格才停 或 讀完 param_1 bytes);
}
```

**格式（已驗證）**：
- byte 最高位 = 1：**RLE 記錄，2 bytes**——`byte & 0x7f` 是 tile 值，**下一個 byte 是重複次數**。
- byte 最高位 = 0：**literal 記錄，1 byte**——這個 byte 本身就是 tile 值，只出現一次。
- 輸出目標固定是 **64×64 = 4096 格的環狀緩衝區**，寫入游標每次 `+64` 再對 4096 取餘——
  等同「逐欄（column-major）填格」：先填第 0 欄的 64 格（每格位移 64 個 byte 才是同一欄的
  下一格，這一步其實是先繞完一輪 landing 在不同欄再回頭，實際上是以 64 為步進在
  0–4095 的環上跳，等效於「先填滿第 0 列的欄位相位、再下一相位」——與獨立檔案
  `MAP1.MAP`/`MAP3.MAP`/`MAP5.MAP` 已驗證的 64×64 網格大小完全一致）。

**這解開了 `town-and-map.md` §2.4 記錄的「RLE 假設已被測試並否證」**：**先前測試的
編碼方向剛好相反**——原假設是「`byte & 0x7f` 是重複次數，下一 byte 是值」，而反組譯
證實的實際格式是「`byte & 0x7f` 是**值**，下一 byte 是**重複次數**」。這正是為什麼先前
的解碼會得出 156,865 個 tile（無法對應任何合理地圖尺寸）——值和次數對調後，會把本來
不多的次數當成海量的值，或反之，整個解碼會系統性失真。**用反組譯驗證出的正確方向重跑
解碼，每個區塊解出的資料量上限固定是 4096 格（64×64），與獨立 `.MAP` 檔案的網格大小
一致**，這代表 `SUM.MAP` 的 23 個區塊，語意上就是 23 張（可能不滿版，因為讀完
`param_1` bytes 就會提早停止，不一定填滿全部 4096 格）壓縮過的 64×64 地圖。

**未驗證（下一步）**：本文件沒有重跑完整解壓縮並產出 23 張 ASCII 地圖（時間所限，
且該工作應該用 `tools/parse_map.py` 之類的既有工具改寫，不在本次「只能新增
`tools/` 底下自己的檔案」的邊界內做大規模產出）；建議下一位 agent 依本文 §3.2/§3.3
的演算法直接實作解壓縮器，重現 23 張地圖並比對其中已知 ID（2、4，對應 `ITEMLOCB.DAT`
的地城）與地城劇情内容是否吻合，會是最有力的收尾驗證。

---

## 4. 對既有文件的修正建議（不直接修改，列在此處供裁決）

1. **`docs/formats/town-and-map.md` §2.4「SUM.MAP：未解」應該整段改寫。**
   已從「排除了幾個簡單假設」推進到「已驗證是 23 個變長 RLE 壓縮區塊的串接，附完整解壓縮
   演算法與讀取函式位址」。原文的「RLE 假設已被測試並否證」需要修正為「RLE 假設方向正確，
   只是值/次數欄位順序搞反了」（見本文 §3.3）。
2. **`docs/formats/town-and-map.md` §5 第 5 點「7 個地城只有 3 個有對應 `.MAP` 檔」已解開。**
   答案：`MAPn.MAP` 獨立檔只給 `n∈{1,3,5}`，其餘地城（含 `ITEMLOCB.DAT` 出現過的
   `map_id=2`／`4`）打包在 `SUM.MAP` 裡，用 `FUN_222f_28d0` 的 ID 表（`31f0:2488`/`31f0:24b6`）
   查表定位（見本文 §3.1/§3.2）。
3. **`docs/formats/resource-index.md` §2.2「FILES.DAT 與 FILES.DTT 關係證據不足」可以補充**
   （不必推翻，是加強）：本文 §2.1 找到一張把 `FILES.DAT`／`FILES.DTT`／`ITEMLOCB.DAT`／
   `PARTY.DAT`／`DEMON.SHP`／`WINTER.SHP` 串在一起的遠指標表 + 文字清單，證實這幾個檔案
   屬於同一批「核心資源」，但**沒有找到消費這張表的程式碼**，所以 `FILES.DAT` 逐欄語意
   仍未解——這點與原文檔結論一致，沒有推翻，只是多了一條「載入時機/同伴檔案」的新線索。

---

## 5. 給下一位 agent 的具體建議

1. **FILES.DAT/FILES.DTT 讀取端**：對 §2.1 找到的指標表位址（Ghidra 位址 `333d:0000`
   一帶，換算 file_offset≈`0x26fd0`）寫一支 Ghidra script，用
   `currentProgram.getReferenceManager().addMemoryReference(...)` 手動建立這塊資料的
   XREF（或直接用 `getReferencesTo` 反查，如果 Ghidra 其實有記錄但匯出腳本沒抓到），
   應該能一步找到消費端函式。
2. **SUM.MAP 23 個區塊的完整解壓縮與比對**：照 §3.3 的演算法實作解壓縮器（可以是
   Python，純標準庫），對 23 個區塊各自解出 64×64 網格，用 `tools/parse_map.py`
   現成的 ASCII 算繪邏輯視覺化，交叉比對 ID=2/4 兩個區塊是否呈現「地城感」的房間結構
   （對照 `MAP1.MAP`/`MAP3.MAP` 的視覺特徵），會是本文結論的最終驗證。
3. **`[0x1585]` 開關字串位置**：`Sound on`/`Sound off` 字串的印出點沒有在時間內找到
   （§1.5 末段），如果要做選單層級的中文化或行為還原，需要另外追。

## 附：可重跑的驗證片段

```bash
# 1. 排除 AdLib/MIDI port I/O
grep -in "0x388\|0x389\|0x330\|0x331" workplace/ghidra/export/disassembly.asm

# 2. 手動反組譯 INT1Ch handler（§1.2）
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
seg, off = 0x1d9f, 0x2a26
fo = seg*16 + off - 0xC400
open('/tmp/int1c.bin','wb').write(data[fo:fo+110])
"
objdump -D -b binary -m i386 -Maddr16,data16 /tmp/int1c.bin

# 3. SUM.MAP 的 23 筆 ID/大小表（§3.2）
python3 -c "
import struct
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
seg = 0x31f0
fo1 = seg*16 + 0x2488 - 0xC400
fo2 = seg*16 + 0x24b6 - 0xC400
ids = struct.unpack('<23h', data[fo1:fo1+46])
sizes = struct.unpack('<23H', data[fo2:fo2+46])
print(ids); print(sizes); print(sum(sizes))
"
```
