# Ghidra 跳表反組譯修復（DEMON.INT）

修復對象：Ghidra 對 `DEMON.INT` 兩個已知間接跳表 switch 的反組譯「間隙」問題
（`docs/re/00-ghidra-setup.md` 第 5 條、`docs/re/06` §7、`docs/re/08` §5.4、
`docs/re/09` §0 已描述的三次錯誤斷言的根因）。全程 docker，用
`tools/ghidra_headless.sh` + Java post-script（Ghidra 12.x 沒有內建 Jython，
不能用 `.py`）。

## 結論先講

**修復成功，兩個已知的間接跳表都正確標註，decompiler 輸出品質大幅改善，
三個已知錯誤全部確認修正。** 過程中踩到一個關鍵教訓：只加 listing 層級的
`COMPUTED_JUMP` reference **不足以**讓 decompiler 產生正確的 C 碼——一定要另外
呼叫 `JumpTable.writeOverride()` 寫入 decompiler 真正會讀的跳表覆寫（見「踩雷」
一節）。另外發現 `docs/re/08` §5.4 的三個位址標籤（表三 case 7–17 共用 handler、
case 18/Quit、default）有轉譯誤差，本文已用原始 binary 逐位元組核對確認，
提報給協調者裁決（見「與既有手工分析的落差」一節），不擅自去改該文件。

## 做法

### 1. 兩張跳表（沿用協調者提供、已驗證的位址與內容）

| 位址 | 項數 | 間接 JMP 位址 | 用途 | 外層函式 |
|---|---|---|---|---|
| `138d:258f` | 15 | `138d:25b5` | 戰鬥動作分派 | `FUN_138d_1ef8` |
| `222f:12ce` | 19 | `222f:12fc` | 主指令迴圈分派 | `FUN_222f_0b0e` |

腳本執行時額外直接讀 Ghidra 記憶體裡的原始 bytes 跟寫死的清單交叉核對
（不是盲目相信清單），兩張表**全部項目 100% 吻合**：

```
[jumptable] 記憶體核對(vs 協調者提供清單): 全部 15 項吻合   ← 138d:258f
[jumptable] 記憶體核對(vs 協調者提供清單): 全部 19 項吻合   ← 222f:12ce
```

### 2. `AnnotateJumpTables.java` 的修復步驟

產出：`tools/ghidra_scripts/AnnotateJumpTables.java`。對每張表依序：

1. 把跳表本身的記憶體區域清成未定義、再標成 `word` 陣列資料
   （`ArrayDataType(WordDataType, n, 2)`），避免被誤判成程式碼。
2. 對每個 case 目標位址：若尚未反組譯就呼叫 `disassemble(Address)` 展開
   （回傳 `true`／`false`，不丟例外）。
3. 在間接 JMP 指令上，依 case 順序（**保留重複目標，不去重**）呼叫
   `Instruction.addOperandReference(0, target, RefType.COMPUTED_JUMP,
   SourceType.USER_DEFINED)`。
4. 呼叫 `CreateFunctionCmd.fixupFunctionBody(Program, Function, TaskMonitor)`
   （官方 API）重算外層函式本體——這時 CFG 已經包含新加的 computed-jump 邊，
   函式本體會正確納入 case body，且在遇到下一個已存在的函式進入點
   （如 `FUN_222f_1321`）時仍正確停止，不會過度吞併。
5. **關鍵步驟**：用 `new JumpTable(branchAddr, destList, true, 0)` +
   `jumpTab.writeOverride(function)` 寫入 decompiler 真正會採用的跳表覆寫
   （見下方「踩雷」一節，這步是第一版失敗、第二版才補上的）。
6. 兩張表都處理完後呼叫 `analyzeChanges(currentProgram)`，讓 stack、data
   reference 等其餘分析器跟進新展開的程式碼。
7. 呼叫 `runScript("ExportAnalysis.java")` 重用既有匯出腳本重新匯出五種產出
   檔案，不重複實作匯出邏輯。

跑法（沒有修改 `tools/ghidra_headless.sh`，仍照既有用法）：

```bash
cd /home/anr2/cht/daemon_winter
./tools/ghidra_headless.sh orig/demwin/DEMON.INT <專案名稱> AnnotateJumpTables.java
```

**限制**：`tools/ghidra_headless.sh` 每次都 `-import -overwrite`（全新匯入），
所以標註與匯出必須在**同一次** headless 呼叫裡完成——這也是為什麼
`AnnotateJumpTables.java` 最後用 `runScript()` 直接呼叫 `ExportAnalysis.java`，
而不是拆成兩次個別的 `ghidra_headless.sh` 呼叫（拆開的話第二次的 `-overwrite`
會把第一次標註的結果整個沖掉）。

## 踩到的雷（最重要的一條）

**只加 `COMPUTED_JUMP` reference + 重建函式本體，不足以讓 decompiler 產生正確
C 碼。** 第一版腳本只做到「加 reference + `analyzeChanges()`」，結果：

- `disassembly.asm`（逐指令反組譯）**完全正確**——函式本體正確從 1827 bytes
  擴大到約 4200+ bytes，逐行核對跟 `docs/re/08` 手工用 `objdump` 解出的內容
  幾乎一致（面向增量表 `0x2107`/`0x15da`/`0x15d2`、可通行性表 `0x5500`、入口
  對照表 `struct+0x22..+0x26` 全部對得上）。
- 但 `decompiled_all.c` 裡 `FUN_222f_0b0e` 的反編譯輸出**幾乎沒變**
  （6548 行 / 59 個警告，跟修復前 6557 行 / 59 警告幾乎一樣），而且**仍然**
  反編出對 `FUN_25be_0263` 的呼叫——第一版沒有真正修好已知錯誤。

原因：decompiler 對 `BRANCHIND`（間接跳轉）的解析，走的是自己的 p-code 層級
jump table 分析，**不會單看 listing 上的 reference**。要讓 decompiler 真的
採用「已知答案」，要用 Ghidra 內建的 jump table override 機制。這不是憑記憶
猜的——是直接讀這個 image 裡隨附的官方範例腳本
`Ghidra/Features/Decompiler/ghidra_scripts/SwitchOverride.java`
（"Override indirect jump destinations" 這個 GUI 功能背後的實作）找到的正確
用法：除了加 `addOperandReference`，還要另外
`new JumpTable(branchAddr, destList, true, 0).writeOverride(function)`。
補上這步之後，`decompiled_all.c` 明確標出
`/* WARNING: Switch is manually overridden */`，反編譯品質才真正改善
（見下方量化對比）。

**教訓給後續遇到同類問題的 agent**：Ghidra scripting 裡「listing 有 reference」
跟「decompiler 真的採用這個答案」是兩件不同的事，遇到「加了 reference 但
decompiler 輸出沒變」時，先去找 image 隨附的官方範例腳本（
`find /opt/ghidra -path "*ghidra_scripts*" -iname "*.java" | xargs grep -l
"switch\|jumptable"`），不要自己猜 API 組合。

其餘照舊踩雷（跟 `docs/re/00`、`docs/re/08` 一致，不重複列）：`javap` 反查
API 簽章（`ReferenceManager.addMemoryReference`、`CodeUnit.addOperandReference`、
`JumpTable` 建構子、`CreateFunctionCmd.fixupFunctionBody` 都是先用 `javap -p`
對這個 image 裡的 `Base.jar`/`SoftwareModeling.jar` 反查過才用，過程中發現
`JumpTable`/`JumpTable.BasicOverride` 這兩個類別本身就是為了這個場景設計的）。

## 量化對比（改善前 vs 後）

| 指標 | 修復前（基準） | 修復後 | 變化 |
|---|---|---|---|
| 函式總數 | 353 | 361 | **+8**（`docs/re/00` 已記錄函式邊界判定有 ±5 左右的非決定性，+8 明顯超出雜訊範圍） |
| `FUN_222f_0b0e` 宣告大小 | 1827 bytes（已知是錯的，實際控制流延伸到 `222f:1321`） | 4581 bytes | 正確反映真實函式範圍 |
| `FUN_222f_0b0e` 反編譯行數 | 6557 行 | **336 行** | **-94.9%** |
| `FUN_222f_0b0e` unreachable 類警告 | 59 個 | **1 個** | **-98.3%** |
| `FUN_138d_1ef8` 宣告大小 | 565 bytes | 2753 bytes | 正確反映真實函式範圍 |
| `FUN_138d_1ef8` 反編譯行數 | （未於任務單列出基準，但 `docs/re/09` §0 記錄原本帶 `Control flow encountered bad instruction data`／`Type propagation algorithm not settling` 等明確警告） | 224 行，1 個警告 | 乾淨、無 unreachable |
| `decompiled_all.c` 警告總數 | 332 | **269** | **-19.0%** |

## 三個已知錯誤的抽驗結果

### 1. 戰鬥動作 case 5 = Attack、case 10(0xa) = 純顯示 —— **確認修正**

新反編譯出的 `FUN_138d_1ef8`（`decompiled/1ef8_..._FUN_138d_1ef8.c`）明確標出
`/* WARNING: Switch is manually overridden */`，逐一核對 15 個 case：

- **`case 0x15c83`（= `138d:23b3`，case 5）**：`FUN_1990_050d(...)` 找目標後
  直接 `FUN_138d_25da(param_1,iVar7,iVar8,iVar6)`——就是命中/傷害核心函式，
  **確認是 Attack**。
- **`case 0x15db0`（= `138d:24e0`，case 10/0xa）**：`FUN_17c5_1056(0x1000,
  param_1); bVar5=true;`——呼叫 Examine 專用函式、不碰 HP/傷害欄位，
  **確認是純顯示（Examine），不是 Attack**。

`docs/re/06` 原先「case 0xa = 已驗證 Attack」的錯誤斷言，方向與 `docs/re/09`
的修正結論完全一致（Attack=5、Examine=10）。

### 2. `FUN_222f_0b0e` case 0x4 不應呼叫 `FUN_25be_0263` —— **確認修正**

新反編譯出的 `FUN_222f_0b0e` 全文（336 行）**完全沒有出現 `FUN_25be_0263`**：

```bash
$ awk '/^\/\/ ==== FUN_222f_0b0e @/{f=1;next} /^\/\/ ==== /{f=0} f' \
    workplace/ghidra/export/decompiled_all.c | grep FUN_25be_0263
# 無輸出
```

取而代之，case 0-3/4（移動/轉向）的 handler 正確反編成完整的移動邏輯：
面向查表 `0x2107`/`0x15da`/`0x15d2`、可通行性表 `0x5500`、入口對照表
`struct+0x22..+0x26`、目的地 tile 直接觸發 `∈{0x2e,0x25,0x26,0x64,0x5b}`、
地城樓層邊界判定（`dataset%10`）、`FUN_222f_05c6` 步數計數——**跟 `docs/re/08`
§6 手工用 objdump 解出的完整移動流程逐段對得上**，不是巧合。

### 3. 新反組譯的 case body 內容 vs `docs/re/08`、`docs/re/09` 手工解出的結果

**`docs/re/09`（戰鬥動作分派，15 項）**：**完全一致**，15 個 case 逐一核對
（0-3 共用移動/轉向、4=前進、5=Attack、6=Cast→`FUN_138d_4188`、
7=Use→`FUN_17c5_18ab`、8=Turn Undead→`FUN_17c5_1a65`、
9=Dodge（`0x4ecc=[0x5190]/3`，字串 `0x827`，逐位元組公式完全吻合）、
10=Examine→`FUN_17c5_1056`、11=Sound→切換 `[0x1585]`+`FUN_1d9f_29a4`、
12=Pray→`FUN_138d_3ad2`、13=Leech→`FUN_17c5_0f2d`、14=`return 7`），
無一衝突。

**`docs/re/08`（主指令迴圈分派，19 項）**：內容（行為）**完全一致**，但發現
**位址標籤有落差**，如實回報如下（不擅自判定、不修改該文件）：

| case | `docs/re/08` §5.4 標的位址 | 本次直接讀 binary + Ghidra 驗證的位址 | 差值 |
|---|---|---|---|
| 7–17 共用 handler（`MOV AX,[BP-4]; RETURN`） | `222f:12b8` | `222f:12c8` | `+0x10` |
| 18／Quit | `222f:1298` | `222f:12a8` | `+0x10` |
| ≥19／default | `222f:11a8` | 與 7–17 相同（`222f:12c8`），不是獨立位址 | — |

驗證方式（可重跑）：直接對 `DEMON.INT` 原始 bytes 在兩組候選位址各自
`objdump` 反組譯，只有 `222f:12c8`／`222f:12a8` 落在乾淨的指令邊界上且內容
與 `docs/re/08` **描述的行為**完全吻合（`0x12c8` 處確實是
`MOV AX,[BP-4]` 接 `JMP`；`0x12a8` 處確實是 `MOV AX,0x3 / PUSH AX / CALLF
1d9f:19d3`confirm 對話框）；`0x12b8`／`0x1298` 落在鄰近指令中間，不是合法的
指令起點。三個位址都差固定的 `0x10`（16 bytes），研判是 `docs/re/08`
手工換算時的單一運算疏失，**不是**行為理解錯誤——該文件對這幾個 case
「做什麼」的描述本身是對的，只有數字標籤誤差。已用 Ghidra 記憶體讀值
（跟協調者提供清單「全部 19 項吻合」）與新反編譯出的 `case 0x235b8: goto
LAB_222f_0b55;`（`LAB_222f_0b55: return local_4 - 1;`）雙重確認本次修復用的
位址正確。**提報供協調者裁決是否要更新 `docs/re/08`**，本次未動該文件。

## 產出物

- `tools/ghidra_scripts/AnnotateJumpTables.java`（新增）
- `workplace/ghidra/export/`（重新匯出，不入版控）：`functions.csv`
  （361 筆）、`strings.csv`、`disassembly.asm`（42856 行）、
  `decompiled_all.c`、`decompiled/*.c`（360 個函式，2 個因既有的 decompiler
  內部例外略過——跟修復前的既有噪音同類，非本次新增，見下方「未影響範圍」）
- 本文件 `docs/re/12-ghidra-jumptable-fix.md`

## 未影響範圍（誠實記錄）

- 匯出時仍有 2 個函式（`FUN_138d_3c81`、`FUN_1990_3da0`）decompile 失敗被
  `ExportAnalysis.java` 的既有 try/catch 略過——跟 `docs/re/00` 記錄的既有
  噪音（`AddressOutOfBoundsException` 類）同一種問題，不是本次修復引入的
  新問題，也不影響本次驗收的兩個目標函式。
- 本次只處理任務指定的這兩張已知跳表，沒有嘗試通用的「自動偵測同類 pattern」
  版本——按任務指示「不要為了通用性犧牲這兩張已知表的正確性」，優先把已知
  基準做對做透。專案裡若還有其他同類未展開的跳表（`docs/re/00` 提過的十幾個
  小 segment），可以照本文的 `AnnotateJumpTables.java` 模式（複製
  `JumpTableSpec` 條目）逐一補上，流程已驗證可行。
