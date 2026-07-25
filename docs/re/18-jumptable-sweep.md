# 第三張跳表修復 + 全檔同類跳表掃描（DEMON.INT）

修復對象：`FUN_138d_3c81`（Use 道具／怪物 AI 效果套用引擎的第二層分派，18 項，
索引為效果類型 `[0x4e2e]`）目前唯一未套 override 的間接跳表，並把上一輪
（`docs/re/12`）「只修兩張已知表」的做法擴大成全檔同類 pattern 掃描。

分析方法沿用 `docs/re/00-ghidra-setup.md`（位址換算 `file_offset = segment*16 +
offset - 0xC400`）、`docs/re/12-ghidra-jumptable-fix.md`（`JumpTable.writeOverride()`
才是 decompiler 真正會採用的修復手法，光加 `COMPUTED_JUMP` reference 不夠）。
全程 docker + `tools/ghidra_headless.sh`，post-script 用 Java（Ghidra 12.x 無
內建 Jython）。

> 本文取代協調者先前因執行 agent 卡在背景輪詢迴圈而暫時寫出的簡版報告
> （當時只完成腳本與重跑，尚未逐一讀完 18 個 case body）。背景任務其實已在
> 該版本寫出前就完成，本文以完整讀完的 case body 內容重寫，四項目標全部給出
> 精確整數公式，不是候選清單。

---

### 協調者獨立複核（2026-07-25）

四項公式是強主張，抽兩條逐字重驗，**兩條都成立**：

**即死（`FUN_138d_0f0d`）** —— 反編譯全文與本文 §「即死」一節逐字吻合：

```c
iVar1 = FUN_1d9f_0e0b(100);                               // Roll(100)
if ((param_3 * *(int *)0x4e30) / *(int *)0x4e32 < iVar1)  // rate = SP*K/M
    → 印失敗訊息
else
    → FUN_138d_1c94(...)                                   // 已知的死亡/勝負結算
```

即 `Roll(100) <= K*SP/M` 觸發。`0x4e30` 別處以 `< 1` 判正負（K 的符號語意），
`0x4e32` 恆為除數，K/M 的指派無疑義。

**束縛施加（`FUN_138d_0e04`）** —— 同樣逐字吻合，並確認了一個機制細節：

```c
if (*(int *)(param_1 * 0x26 + 0x4ec4) < 2) {              // 尚未被束縛才可施加
  if (Roll(100) < 目標.力量(0x4ec0)*4 - (param_2 * 4 * K) / M)  → 抵抗
  else {
    目標.status_counter(0x4ecc) = (param_2 * K) / M;      // 持續回合
    目標.status(0x4ec4)         = *(int *)0x4e2c;         // ★ 存的是「符文系別」
  }
}
```

**束縛狀態欄位存的是施法的符文系 id，不是布林旗標** —— 這正是解除（`FUN_138d_3098`）
要求「系別相符」的原因，兩支函式互相印證。本文 §「束縛施加／解除」已記載此點，複核確認無誤。

**順帶修掉一條既有錯誤斷言**：`docs/re/16` 曾寫 `spell_school_id(0x4e2c) == 0x11`，
實際程式碼是 `*(int *)0x4e2e != 0x11`（判 `effect_type` 不是符文系）。
`0x4e2c` 全檔只跟 `1`/`4`/`6`/`8` 比較，值域到不了 `0x11`。已在該文更正。

**一處不宜過度推論**：`0x4e2c`–`0x4e32` 四個 word 確實是法術記錄的前四欄
（school / effect_type / K / M）連續攤平到全域，但**`0x4e34` 不是 `w4`** ——
它全檔 6 處用法都是傳給 `FUN_1d9f_1361`（印字串）的指標。
法術表 offset 8 的 `w4` 仍未解，別因為位址相鄰就當它被載到 `0x4e34`。

---

## 結論先講

**第三張表修復成功，且全檔掃描找到的另外 18 張同類跳表裡有 17 張也一併修好，
只有 1 張因根因較深（見 §4.3）誠實跳過、沒有硬標。**

**四項目標全部解出，其中三項（即死、束縛施加、束縛解除、枯萎）拿到
可直接寫進 spec 的精確整數公式，`Use` 道具效果套用的整條 18 項分派表已完整
解出並與另一張姊妹表（`FUN_138d_2e63`）互相印證：**

| 目標 | 解出程度 |
|---|---|
| 即死判定（`effect_type 2`） | **已驗證**，完整公式（見 §7.1） |
| 束縛施加（`effect_type 11`） | **已驗證**，完整公式（見 §7.2） |
| 束縛解除（`effect_type 10`） | **已驗證**，機制清楚，一個欄位語意仍 `[假設]`（見 §7.3） |
| 枯萎（`effect_type 14`） | **已驗證**，完整公式，且證實與其他「單屬性削弱」公式不同套（見 §7.4） |
| `Use` 道具效果套用 | **已驗證**，18 項全部映射到具體處理函式（見 §6） |

量化對比（詳見 §5）：函式總數 361 → **394**（+33）、全檔警告總數 269 → **86**
（**-68.0%**）、`FUN_138d_3c81` 從**完全無法反編譯**（decompiler 內部例外）→
**118 行，僅 1 個良性警告**（override 本身的提示註解）。三次獨立重跑
（`sweep`/`sweep2`/`sweep3`）結果穩定可重現，最後兩次逐位元組一致。

---

## 1. 第三張表：位址、內容、修復結果

### 1.1 位址（已驗證，直接讀原始 binary 核對）

| 項目 | 值 |
|---|---|
| 外層函式 | `FUN_138d_3c81`（`138d:3c81`，file offset `0xb151`） |
| 跳表本體 | `138d:3f95`（file offset `0xb465`），18 個 word |
| 邊界檢查 | `cmp ax,0x12`（**18**，索引 0..17）／`jae` 超界跳到跳表後緊接的預設處理 |
| 間接 JMP | `138d:3fc1`（file offset `0xb491`），`jmp word ptr cs:[bx+0x3f95]` |

任務單原文寫邊界檢查是 `cmpw $0x11,0x4e2e`（0x11=17）；本輪直接對原始位元組
`objdump` 核對後，實際指令是 `cmp ax,0x12`（0x12=18），`ax` 是**已經從
`[0x4e2e]` 讀進暫存器之後**的索引值（跳表前段有 `mov ax,0x4e2e; jmp <switch>`
的轉接，`cmp` 比對的是 `ax` 不是記憶體）。兩者描述的其實是同一件事：**18 項，
索引 0..17 合法，索引 == 18 才觸發超界**——數字表述方式不同（比對「17 這個
邊界值本身」vs「18 這個項數」），內容一致，18 項與跳表實際 word 數完全吻合。

驗證片段：

```bash
cd /home/anr2/cht/daemon_winter
objdump -D -b binary -m i386 -Maddr16,data16 \
  --start-address=$((0xb489)) --stop-address=$((0xb4a0)) \
  workplace/orig/demwin/DEMON.INT 2>/dev/null
# b489: cmp $0x12,%ax   (18)
# b48c: jae 0xb496       (default, = table entry 0 target)
# b48e: xchg %ax,%bx
# b48f: shl $1,%bx
# b491: jmp *%cs:0x3f95(%bx)
```

### 1.2 跳表內容（已驗證，Python 直接讀 binary，與 `docs/re/16` 先前用同一方法
讀出的清單完全吻合，本輪再次獨立核對一次）

```
effect_type  0 -> 138d:3fc6   （default，超界與 case 0 共用同一目標）
effect_type  1 -> 138d:3c8d   （5×5 AOE，docs/re/09 §2.5／docs/re/16 §5 已知）
effect_type  2 -> 138d:3cc5   （即死，本輪新解，見 §7.1）
effect_type  3-7,0xd -> 138d:3cff  （單體 HP/屬性效果，docs/re/09 §4.3 已知）
effect_type  8,0xc,0x11 -> 138d:3d98  （SP-like 欄位加值／"沒有效果"提示）
effect_type  9 -> 138d:3dcf   （二元狀態切換，本輪新解，見 §8.3）
effect_type 10 -> 138d:3e25   （束縛解除，本輪新解，見 §7.3）
effect_type 11 -> 138d:3e7f   （束縛施加，本輪新解，見 §7.2）
effect_type 14 -> 138d:3f0f   （枯萎，本輪新解，見 §7.4）
effect_type 15 -> 138d:3f48   （召喚生物選單，本輪新解，見 §8.1）
effect_type 16 -> 138d:3f5c   （職業/陣營轉換，本輪新解，見 §8.2）
effect_type 17 -> 138d:3d98   （與 8/12 共用）
```

18 個索引只對應 11 個相異目標位址（多個 case 共用 handler 是正常現象，`docs/re/12`
已有先例）。全部目標位址都落在 `FUN_138d_3c81` 附近的合理範圍內，不與跳表本身
重疊。

### 1.3 修復結果（已驗證）

`FUN_138d_3c81` 原本**完全無法反編譯**——`ExportAnalysis.java` 呼叫
`decompileFunction()` 直接丟 `AddressOutOfBoundsException` 類例外
（`docs/re/12`「未影響範圍」一節已記錄這是既有噪音，跟 `FUN_1990_3da0` 並列
兩個修復前完全匯不出來的函式）。修復後：

```c
void __cdecl16far FUN_138d_3c81(int param_1,int param_2,undefined2 param_3,int param_4)
{
  ...
  if (*(uint *)0x4e2e < 0x12) {
                    /* WARNING: Switch is manually overridden */
    switch((undefined4)*(undefined2 *)(*(uint *)0x4e2e * 2 + 0x3f95)) {
    case 0x1755d: ...   // effect_type 1: AOE
    case 0x17595: ...   // effect_type 2: 即死
    case 0x175cf: ...   // effect_type 3-7,13: 單體效果 + 抵抗判定
    case 0x17668: ...   // effect_type 8/12: SP 欄位加值 或 "無效果"，並 fallthrough
    case 0x1769f: ...   // effect_type 9/17: 二元狀態切換
    case 0x176f5: ...   // effect_type 10: 束縛解除
    case 0x1774f: ...   // effect_type 11: 束縛施加（含抵抗判定）
    case 0x177df: ...   // effect_type 14: 枯萎
    case 0x17818: ...   // effect_type 15: 召喚生物選單
    case 0x1782c: ...   // effect_type 16: 職業轉換
    case 0x17896: goto override_jmp_138d_3fc1_case_0;   // 對應 default
    }
  }
  else { override_jmp_138d_3fc1_case_0: }
  return;
}
```

**118 行，只有 1 個警告**（就是 override 本身的提示註解，不是錯誤）。11 個
相異 case body 全部可讀、全部指向具體、已存在或本輪一併展開的函式，沒有任何
`Removing unreachable block`、`Control flow encountered bad instruction data`
類警告——跟 `docs/re/12` 表一/表二修復後的乾淨程度一致。

完整反編譯內容見 `workplace/ghidra/export/decompiled/138d_3c81_FUN_138d_3c81.c`
（不入版控，重跑 `AnnotateJumpTablesSweep.java` 即可再生）。

---

## 2. 全檔同類跳表掃描：方法

### 2.1 這個編譯器對「間接跳轉 switch」只用一種固定慣用法

比對 `docs/re/12` 已修復的兩張表（`138d:258f`、`222f:12ce`）與本輪目標表的
原始位元組後發現：**這個編譯器產生的每一個間接跳轉 switch，組譯結果都是同一組
固定的 13 bytes 慣用法**：

```
3D NN 00         CMP AX, NN        ; NN = case 數量，AX 此時已是 0-based 索引
73 08            JAE +8            ; 超界跳到「緊接在 JMP 指令後」的 default
93               XCHG AX,BX
D1 E3            SHL BX,1
2E FF A7 xx xx   JMP word ptr CS:[BX+disp16]   ; disp16 = 跳表在同一 segment 內的 offset
```

### 2.2 全檔掃描與交叉驗證

用 Python 對整份 `DEMON.INT`（173,380 bytes）搜尋固定的 3-byte 錨點
`2E FF A7`（`jmp word ptr cs:[bx+disp16]` 的 prefix+opcode+modrm），共命中
**21 處**。對每一處都往回讀 8 bytes，核對是否精確吻合上述 `CMP/JAE/XCHG/SHL`
序列——**21 處全部吻合，無一例外**，包括：

- `docs/re/12` 已修復的兩張已知表（`138d:258f` n=15、`222f:12ce` n=19）——
  掃描器獨立算出的項數與已驗證的 15、19 完全一致，**這是掃描器本身正確性的
  交叉驗證**。
- 本輪任務指定的第三張表（`138d:3f95` n=18）——同上，獨立驗證吻合。
- 另外 **18 張此前完全沒人碰過的表**。

```bash
cd /home/anr2/cht/daemon_winter
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
pattern = bytes([0x2e, 0xff, 0xa7])
idx, hits = 0, []
while True:
    idx = data.find(pattern, idx)
    if idx == -1: break
    hits.append(idx); idx += 1
print('總命中數:', len(hits))
for h in hits:
    pre = data[h-8:h]
    ok = (pre[0]==0x3d and pre[3]==0x73 and pre[4]==0x08
          and pre[5]==0x93 and pre[6]==0xd1 and pre[7]==0xe3)
    n = pre[1] | (pre[2]<<8)
    disp = int.from_bytes(data[h+3:h+5],'little')
    print(hex(h), 'n=', n, 'table_disp=', hex(disp), 'pattern_ok=', ok)
"
```

### 2.3 每張表標註前的驗證條件（不通過就跳過，不硬標）

對每一處候選，標註前都要求同時滿足：

1. **項數與邊界檢查吻合**——`CMP AX,NN` 讀出的 `NN` 與跳表實際 word 數必須
   相等（掃描器結構上保證一致，因為兩者都從同一組 bytes 讀出，但仍另外在
   Java 端加了一次記憶體核對，見 `AnnotateJumpTablesSweep.processOne()` 步驟 0）。
2. **所有目標位址落在合理範圍**——用 `functions.csv` 算出每個 segment 的
   file-offset 覆蓋範圍，21 張表的全部目標無一逸出所屬 segment 的既有函式
   分佈範圍。
3. **目標不與跳表本身重疊**。

21 張全部通過這三項靜態驗證。實際標註時另外會遇到**動態面的失敗**（`fixupFunctionBody`
/ `JumpTable.writeOverride()` 丟例外）——這類失敗記錄在 §4「踩雷」，跟上面
三項靜態驗證是兩回事：靜態驗證確保「這張表本身合理」，動態失敗反映的是
「Ghidra 這一輪自動分析對周邊函式邊界的既有非決定性」，見下節。

---

## 3. 全檔掃描結果總表

| 表 | 位址 | 項數 | 外層函式 | 結果 |
|---|---|---|---|---|
| 已知#1 | `138d:258f` | 15 | `FUN_138d_1ef8` | 成功（`docs/re/12` 已修，本輪重跑再次確認） |
| 已知#2 | `222f:12ce` | 19 | `FUN_222f_0b0e` | 成功（同上） |
| **目標#3** | **`138d:3f95`** | **18** | **`FUN_138d_3c81`** | **成功**（本輪任務目標，見 §1） |
| 新#1 | `1000:01c7` | 14 | `FUN_1000_01e3` | 成功 |
| 新#2 | `1000:1820` | 18 | `FUN_1000_11e5` | 成功 |
| 新#3 | `1000:33c3` | 9 | `FUN_1000_2a53` | 成功 |
| 新#4 | `138d:0c65` | 17 | `FUN_138d_065e` | 成功 |
| 新#5 | `138d:2f4c` | 17 | `FUN_138d_2e63` | 成功（見 §6.3，與目標#3 姊妹表） |
| 新#6 | `1990:40af` | 15 | `FUN_1990_3da0` | 成功 |
| 新#7 | `1d9f:1dd0` | 5 | `FUN_1d9f_1ce1` | 成功 |
| 新#8 | `1d9f:2beb` | 10 | `FUN_1d9f_2a95` | 成功 |
| 新#9 | `206a:0ea0` | 6 | `FUN_206a_02c7` | 成功 |
| 新#10 | `222f:0563` | 21 | `FUN_222f_0003` | 成功 |
| 新#11 | `25be:06dd` | 9 | `FUN_25be_0263` | 成功 |
| 新#12 | `25be:0deb` | 16 | `FUN_25be_0263`（同函式第二張表） | 成功 |
| 新#13 | `25be:139a` | 7 | `FUN_25be_0e77` | **跳過**（見 §4.3） |
| 新#14 | `278d:08ec` | 12 | `FUN_278d_0098` | 成功 |
| 新#15 | `278d:0beb` | 9 | `FUN_278d_0932` | 成功 |
| 新#16 | `278d:2b83` | 14 | `FUN_278d_22bc` | 成功 |
| 新#17 | `2aed:0b7d` | 5 | `FUN_2aed_07be` | 成功 |
| 新#18 | `2aed:1d63` | 31 | `FUN_2aed_14c2` | 成功 |

**21 張找到，20 張標註成功（含目標#3），1 張誠實跳過。**

---

## 4. 踩到的雷

### 4.1 處理順序會互相影響——同一 segment 內必須照位址由小到大處理

第一版腳本照「已知表 → 目標#3 → 掃描新表」的邏輯順序寫，**結果目標#3 處理時
失敗**：`getFunctionAt(138d:3c81)` 回傳 `null`（「外層函式不存在」）。

追查後發現：這一輪全新 `-overwrite` 匯入的自動分析（`docs/re/00` 記錄過
函式邊界判定有 ±5 左右的非決定性）這次沒有在 `138d:3c81` 建立獨立函式，
且緊鄰在前的 `FUN_138d_2e63`（本身也帶一張未修的跳表，新#5）原本的殘破邊界
干擾了自動分析對 `3c81` 的判斷。**改成「同一 segment 內嚴格照位址由小到大處理」
（`138d`: `065e` → `1ef8` → `2e63` → `3c81`）後問題自然消失**——先把 `2e63`
的跳表修好，`3c81` 的自我修復（見 4.2）才穩定生效。

同理，`25be` segment 內 `FUN_25be_0263` 帶兩張表，`fixupFunctionBody` 展開後
會一路長到緊鄰的 `FUN_25be_0e77`（新#13 的外層函式）進入點附近。把 `25be:0e77`
的表排到 `25be:0263` 兩張表**之前**處理，讓 `0e77` 先被鎖定成「已存在」的函式，
可以避免 `0263` 展開時併吞 `0e77` 的進入點——但這張表最終仍因另一個更深的原因
沒有標成功，見 4.3。

### 4.2 自我修復：`getFunctionAt` 回傳 `null` 時改用 `createFunction()`

針對 4.1 的現象，在 `processOne()` 加了一段自我修復：`getFunctionAt` 找不到
函式時，先確認目標位址已反組譯，再呼叫 `createFunction(funcAddr, null)`
主動切出一個函式。這對目標#3 有效：

```
[sweep] 外層函式 @ 138d:3c81 不存在（getFunctionAt 為 null），
        getFunctionContaining -> null，嘗試 createFunction() 自我修復
[sweep] createFunction(138d:3c81) 成功: FUN_138d_3c81
[sweep] fixupFunctionBody(138d:3c81) -> false，本體大小 699 bytes -> 699 bytes
[sweep] JumpTable.writeOverride() 完成，已寫入 18 項目的地
```

**一個值得記錄的怪現象**：`createFunction()` 切出的函式本體只有 699 bytes
（`3c81` 到 `3f3c`），比跳表本身（`3f95`）與間接 JMP（`3fc1`）的位址都短，
`fixupFunctionBody` 也沒有把它撐大。照理這樣 `writeOverride()` 應該要像
4.3 的情況一樣丟 `InvalidInputException: Switch is not in function body`，
但**這次沒有丟例外，且反編譯結果完全正確**（§1.3 的 118 行乾淨輸出）。
推測原因：decompiler 的 p-code CFG 走的是「指令是否已展開」而不是嚴格的
「函式 body AddressSetView 是否涵蓋」，只要目標位址的指令存在（本輪 §2 已對
全部 18 個目標呼叫過 `disassemble()`），配合 `JumpTable` override 提供的
明確目的地清單，decompiler 仍能正確重建流程——`functions.csv` 裡
`FUN_138d_3c81` 的宣告大小（699 bytes）因此是**已知但無傷大雅的
metadata 誤差**，不影響反編譯內容本身的正確性，如實記錄供後續 agent 參考，
避免誤判「函式大小這麼小，跳表一定沒修好」。

### 4.3 唯一跳過的一張：`25be:139a`（`FUN_25be_0e77`）

第一版（處理順序：先修 `25be:0263` 的兩張表，再修 `25be:0e77`）失敗於
`writeOverride()`：

```
ghidra.util.exception.InvalidInputException: Switch is not in function body
	at ghidra.program.model.pcode.JumpTable.getSwitchNamespace(JumpTable.java:256)
```

原因：`FUN_25be_0263` 光是修好**自己的第一張**表，`fixupFunctionBody` 就把
它的本體從 721 bytes 撐到 4072 bytes（`0263+4072 ≈ 0x124b`），已經遠遠超過
`25be:0e77`（`FUN_25be_0e77` 的進入點）；修好第二張表後更撐到 5720 bytes。
`25be:0e77` 的間接 JMP 在 `25be:13b0`，落在 `0263` 展開後的範圍內，導致
`writeOverride` 判定這個「switch」不屬於它自己名義上的外層函式。

改成「先修 `25be:0e77`，再修 `25be:0263` 的兩張表」（見 4.1）後，失敗點變了：

```
[sweep] !!! 找不到 JMP 指令 @ 25be:13b0，無法加 reference，此表跳過
```

因為這次換成 `25be:13b0` 這段位元組**在自動分析階段完全沒有被展開**
（不像其餘 20 張表的 JMP 指令都已經被原始自動分析線性掃描到）。加了
「JMP 指令本身若未展開就主動 `disassemble()`」的補強後（因為位元組樣式已經
逐一核對過，這裡主動展開是安全的），問題又變回 4.1 一開始的那個例外：

```
ghidra.util.exception.InvalidInputException: Switch is not in function body
```

**結論：`25be:0263` 與 `25be:0e77` 之間存在真正的控制流糾纏，不是單純的
「處理順序」或「函式未展開」問題能解開的**。合理的假設（`[假設，未解]`）是：
`FUN_25be_0e77` 從一開始就不是真正獨立的頂層函式，而是 `FUN_25be_0263`
內部某條分支路徑延伸出去的程式碼——因為 `0263` 本身也帶跳表（且是兩張），
在**這兩張表都還沒修好之前**，原始自動分析的殘缺 CFG 剛好在 `0e77` 這個
位置切出一個看似獨立的函式（跟 `docs/re/12` 記錄的
「`FUN_222f_0b0e` case 0x4 誤呼叫 `FUN_25be_0263`」是同一種病根：跳表沒修好時
decompiler／函式邊界分析都會在殘缺 CFG 上編造出看似合理但錯誤的結構）。

**這張表誠實跳過，沒有標註 override**——`FUN_25be_0e77` 現在的狀態跟修復前
完全一樣（133 行、0 警告，函式本體 901 bytes 不變），**沒有造成任何退步**，
只是它內部的 7 個 case body 仍然讀不到，維持原狀。這個位置**不在** `138d`
段（跟即死／束縛／枯萎／`Use` 效果四項目標無關），故未進一步深挖，留給
下一輪如果要處理 `25be` 段（命中率/傷害相關的另一個模組）時再一併處理。

### 4.4 三次獨立驗證，結果穩定可重現

同一支腳本前後跑了 3 次（`demwin_int_sweep`／`sweep2`／`sweep3`，逐步修補
自我修復與處理順序的問題），**最後兩次（處理順序已修正版）結果完全一致**：
20 張成功、1 張跳過（`25be:139a`），函式總數穩定在 394、警告數穩定在 86，
`FUN_138d_3c81` 的反編譯內容 byte-for-byte 相同（`md5sum` 核對過）。
**這代表這套修復不是巧合，而是可重現的穩定結果**。

---

## 5. 量化前後對比

| 指標 | 修復前（基準，`docs/re/12` 完成後的狀態） | 本輪修復後 | 變化 |
|---|---|---|---|
| 函式總數 | 361 | **394** | **+33**（+9.1%） |
| 全檔警告總數（`decompiled_all.c` 內 `WARNING` 行數） | 269 | **86** | **-183（-68.0%）** |
| `FUN_138d_3c81` 反編譯 | **完全失敗**（decompiler 內部例外，無輸出檔） | **118 行，1 個良性警告** | 從無到有 |
| `FUN_138d_2e63`（新#5，姊妹表） | 1727 行，含 `Control flow encountered bad instruction data`／`Removing unreachable block` 等明確警告 | **54 行，1 個良性警告** | **-96.9% 行數** |
| `FUN_1990_3da0`（新#6） | 宣告 257 bytes（明顯過小，另一個原本完全反編失敗的函式） | 反編成功，930 bytes，118 行，1 警告 | 從無到有 |
| `FUN_222f_0003`（新#10） | 宣告 1039 bytes | 1435 bytes，293 行，1 警告 | 正確反映真實範圍 |
| `FUN_25be_0263`（新#11+#12，兩張表同函式） | 宣告 721 bytes | 3370 bytes，450 行，2 警告（各表一個 override 提示） | 正確反映真實範圍 |
| `FUN_138d_1ef8`／`FUN_222f_0b0e`（`docs/re/12` 已修，本輪重跑） | 223 行 / 1 警告；335 行 / 1 警告 | 相同（無退步，確認穩定） | 持平 |
| `FUN_25be_0e77`（唯一跳過） | 901 bytes，133 行，0 警告 | 相同（無退步） | 持平 |

剩餘 86 個警告全部來自本輪掃描 pattern**沒有涵蓋**的其他位置（不同的間接跳轉
寫法，或跟跳表無關的既有噪音，如 `docs/re/00` 記錄過的 `2000:0133`／
`1000:04b3` 一類 pcode 探測噪音），不是本輪修復範圍內的迴歸。

---

## 6. `Use` 道具效果套用：完整解法

### 6.1 兩張姊妹跳表互相印證

`docs/re/16` §4.4 已知 `Use` 道具的效果套用會依道具類型分派到兩個引擎之一：

```
道具 type ∈ [8,0xc] 或 {0x18,0x19,0xe}（護甲類/特殊道具）→ FUN_138d_2e63（17 項，新#5）
其他（一般消耗品/武器類）                                → FUN_138d_3c81（18 項，目標#3）
```

**本輪兩張都修好了**，而且兩張表的 case handler **高度重疊**（同一組函式被
兩個獨立的跳表各自引用）：

| effect_type | `FUN_138d_3c81`（目標#3） | `FUN_138d_2e63`（新#5） |
|---|---|---|
| 1（AOE） | `FUN_138d_134d` | `FUN_138d_134d` |
| 2（即死） | `FUN_138d_0f0d` | `FUN_138d_0f0d` |
| 3-7,13（單體效果） | `FUN_138d_10bc` | `FUN_138d_10bc` |
| 8,12（SP欄位/無效果） | `FUN_138d_2f7e` | `FUN_138d_2f7e` |
| 9（二元狀態切換） | `FUN_138d_2fa0` | `FUN_138d_2fa0` |
| 10（束縛解除） | `FUN_138d_3098` | `FUN_138d_3098` |
| 11（束縛施加） | `FUN_138d_0e04` | `FUN_138d_0e04` |
| 14（枯萎） | `FUN_138d_0fa5` | `FUN_138d_0fa5` |
| 15（召喚選單） | `FUN_138d_3161` | `FUN_138d_3161` |
| 16（附身：陣營對翻） | `FUN_138d_0cb6` | `FUN_138d_0cb6` |

**這是很強的交叉驗證**——兩個獨立的跳表（不同的呼叫路徑、不同的參數傳遞
方式）各自指向完全相同的一組處理函式，代表這組函式的語意判讀不是巧合，
而是遊戲引擎本身「效果類型 → 處理函式」的統一映射表（`docs/re/09` §4.3
已知的 Cast 路徑效果類型欄位語意，在 Use 道具路徑也適用同一套）。

### 6.2 已驗證：Use 道具效果套用的完整流程

延續 `docs/re/16` §4 已驗證的前置檢查（行動點 ≥3、限玩家、已裝備武器/護甲
或任意消耗品）與效果記錄載入（`FUN_1000_114f`，5-word 記錄：`effect_index`/
`effect_type`/`K`/`M`），效果類型分派到 §6.1 的其中一個函式後：

```
1. 效果類型分派到 §6.1 對應的具體函式（本輪完整解出，11 個相異處理路徑）
2. 大多數路徑先呼叫 FUN_138d_3fc9 挑一個合法目標（0-14 戰鬥槽位，-5=取消）
3. 依效果類型各自的判定式（見 §7、§8）套用效果
4. 統一收尾：FUN_138d_1e19(action_result)，並把回傳碼 3 remap 成 4
   （跟 docs/re/16 §3.3 主迴圈分派碼 3=戰鬥繼續、4=重置鏡頭繼續 一致）
```

### 6.3 `FUN_138d_2e63` 本身內容（新#5，已驗證）

```c
void FUN_138d_2e63(param_1..param_7) {
  if (*(uint*)0x4e2e < 0x11) {  // 17 項，索引 0..0x10
    switch(...) {
      case ...: FUN_138d_134d(0x1000,param_7,param_3,param_4,param_5); break;  // AOE
      case ...: FUN_138d_0f0d(param_1,param_2,param_5,param_7); break;         // 即死
      case ...: FUN_138d_10bc(param_1,param_2,param_5,param_7); break;         // 單體效果
      case ...: *(int*)(param_2*0x26+0x4ec2) += param_5; FUN_138d_2f7e(); break;
      case ...: FUN_138d_2fa0(param_1,param_5,param_6,param_7); break;
      case ...: FUN_138d_3098(param_1,param_5,param_7); break;                 // 束縛解除
      case ...: FUN_138d_0e04(param_1,param_5,param_7); break;                 // 束縛施加
      case ...: FUN_138d_0fa5(param_1,param_5,param_7); break;                 // 枯萎
      case ...: FUN_138d_3161(param_2,param_6,param_7); break;                 // 召喚選單
      case ...: FUN_138d_0cb6(param_1,param_5,param_7); break;                 // 附身：陣營對翻（見 docs/re/23 §4）
      default: goto override_jmp_138d_2f76_case_0;
    }
  } else { override_jmp_138d_2f76_case_0: }
}
```

`docs/re/16` §4.4 原本判定這個函式「反編譯不可信、未展開，這是本輪最大的
未解缺口」——**本輪已完整解決**，且驗證出它跟目標#3 是同一套 handler 集合，
不是獨立的另一套邏輯。

---

## 7. 四項目標的精確公式（可直接寫入 spec）

以下全部標「已驗證」（直接讀本輪修復後的乾淨反編譯得出，非猜測），使用的
RNG 原語 `FUN_1d9f_0e0b(N)` 是 `docs/re/01`／`docs/re/09` 已知的 `RNG(N)`（回傳
`[0,N)` 或 `[1,N]`，精確邊界未在本輪重新核對，沿用既有結論）。所有除法皆為
整數除法（`idiv`，向零捨去）。`K`/`M` 是 `docs/re/09` §4.1 已驗證的效果記錄
欄位（`[0x4e30]`=K、`[0x4e32]`=M），`SP_invested` 是呼叫鏈傳入的「本次投入
點數」參數（`docs/re/16` §4.4 已知，`FUN_138d_3c81(caster, magnitude_param,
effect_idx, 1)` 的 `magnitude_param`，一路傳到各 handler 的 SP 相關參數位）。

### 7.1 即死判定（`effect_type 2`，`FUN_138d_0f0d`）

```
成功率 = K * SP_invested / M         （百分比，0-100，整數除法）
roll = RNG(100)
若 roll <= 成功率:
    印出目標名 + 成功訊息
    呼叫 FUN_138d_1c94(0, target_slot, SP_invested, action_result)
        → 死亡結算（docs/re/16 §3 已驗證的死亡bookkeeping+勝負判定鏈）
    若死亡導致戰鬥結束（回傳碼 != 3），直接把該回傳碼往上傳
否則:
    印出 "Spell fails"
```

**關鍵差異**：即死判定**不走** `docs/re/09` §4.2 那套「`RNG(K*SP/M)` 重擲偏向
上限 2/3 區間」的 magnitude 公式，而是**直接把 `K*SP/M` 當成百分比機率**做
單次擲骰——這是「SP 投入 → 效果強度」公式家族裡專屬於「二元成功/失敗」類
效果的簡化版本，跟 §7.2/§7.4 的判定式同一套風格（差異只在成功後做什麼）。

**與手冊/攻略的對照**：即死類法術（烈焰打擊/死亡之刃/靈魂折磨/枯萎打擊）
若 `K/M` 設計得夠大，SP 投入越多、死亡機率越高，符合直覺；具體到每個法術
各自的 `K`/`M` 數值仍未取出（`docs/re/09` §7.1 已知的未解項，本輪解出的是
**公式結構**，不是逐一法術的係數）。

**也是本輪唯一發現「單體效果會正確觸發勝負判定」的路徑**——對照 `docs/re/16`
§3.4 已驗證的「AOE 擊殺不會立即觸發 `FUN_138d_1d70`」缺口，即死法術**沒有**
這個缺口，殺死最後一個單位會正常結束戰鬥（見程式碼裡 `FUN_138d_1c94` 直接
被呼叫，跟 §5 AOE 呼叫的 `FUN_138d_165d`（純 bookkeeping，不含勝負判定）不同）。

### 7.2 束縛施加（`effect_type 11`，`FUN_138d_0e04`）

```
若 目標.status(0x4ec4) < 2（尚未處於特殊狀態）:
    resist_score = 目標.力量(0x4ec0) * 4 - SP_invested * 4 * K / M
    roll = RNG(100)
    若 roll < resist_score:
        FAIL：印出目標名 + 抵抗訊息（字串 0x6dc）
    否則 SUCCESS：
        目標.status_counter(0x4ecc) = K * SP_invested / M
        目標.status(0x4ec4) = 本次效果的 spell_school_id([0x4e2c])
        印出目標名 + 成功訊息
否則:
    印出 "Spell fails"（已經處於狀態中，無法疊加）
```

**已驗證**：抵抗判定用的是目標「力量」欄位（`0x4ec0`，`docs/re/09` §4.3 已確認
的欄位語意）乘以 4 當作基礎抗性分數，法術投入的 SP 越多、`K*SP/M*4` 扣得越多，
越容易命中——這是經典的「屬性 save vs 施法強度」機制，跟 §1（命中率）、§2
（爆擊）用的「`skill*4`」量綱一致（`docs/re/16` §1 已知命中率系統整體以 4
為縮放單位）。**這正是 `docs/re/09` §7.2「束縛/解縛法術 SP 投入倍數提升成功率」
未解項的答案**：不是「倍數」而是「線性扣減對方的固定抗性分數」。

### 7.3 束縛解除（`effect_type 10`，`FUN_138d_3098`）

```
status = 目標.status(0x4ec4)
若 status != 5（非死亡）且 (status < 2 或 目標.status_counter(0x4ecc) <= K*SP_invested/M):
    school = 本次效果的 spell_school_id([0x4e2c])
    若 school == 1: school = 4     // 別名重映射
    若 school == status:            // 只有「解除法術鎖定的狀態種類」跟目標目前的狀態相符才生效
        目標.status_counter = 0
        目標.status = 0             // 完全清除
        印出成功訊息
    否則：（不做任何事，落入下方 fail 分支）
否則:
    印出 "Spell fails"
```

**已驗證的機制**：解除類法術有**兩層門檻**——(1) 目標當前狀態的殘餘
`status_counter` 要小於等於本次投入的 `K*SP/M`（**弱束縛可以用少量 SP 解除，
強束縛需要更多 SP**，這與 §7.2 施加時 `status_counter = K*SP/M` 直接對應：
解除所需的力度跟當初施加的力度掛鉤）；(2) 解除法術鎖定的「種類」
（`spell_school_id`）要跟目標身上的狀態值相符（不能用錯誤系別的解除術清除
狀態）。**`[假設，未解]`**：`spell_school_id` 每個數值具體對應哪個狀態種類
（束縛/冰封/沉睡/…）本輪未逐一核對，只確認了「必須相符」這條規則本身，以及
`1→4` 這個特殊重映射的存在（可能是某個系別的別名）。

### 7.4 枯萎（`effect_type 14`，`FUN_138d_0fa5`）

```
roll = RNG(100)
threshold = K * SP_invested / M
若 roll > threshold:
    FAIL：印出 "Spell fails"
否則 SUCCESS：
    印出目標名
    目標.速度(0x4eb8)  = max(3, RNG(目標當前速度))
    目標.力量(0x4ec0)  = max(3, RNG(目標當前力量))
    目標.技巧(0x4ec6)  = max(3, RNG(目標當前技巧))
```

**已驗證，且解答了 `docs/re/09` §4.4「枯萎打擊理論上可能是連續呼叫三次
type 3/4/6 而非獨立公式，但未驗證」的懸念**——**答案是否定的**：枯萎打擊
**是獨立的 `effect_type`（14），不是三次 type 3/4/6 呼叫**，而且套用方式也
不同：type 3/4/6（`docs/re/09` §4.3）是「用 `RNG(K*SP/M)` 算出一個扣減量再
從現有值扣」，枯萎則是「把現有值直接**重擲成一個更小的隨機值**」（`RNG(current)`
本身就以目前的值當上限，值越高、重擲後掉得可能越多），兩者的數學行為不同
（枯萎沒有 `K`/`M` 參與屬性本身的計算，`K`/`M` 只決定**是否觸發**）。下限都是
`clamp(3, ...)`，跟 `docs/re/09` §4.3 已驗證的「屬性削弱不能壓到 3 以下」規則
一致。

---

## 8. 額外解出的三項（非任務原定四項，但直接受益於同一次修復）

### 8.1 召喚生物選單（`effect_type 15`，`FUN_138d_3161`）

`docs/re/16` §6 記錄「召喚生物的屬性來源仍未解出」——本輪找到的不是屬性來源
本身，而是**召喚流程的入口**：這是一個完整的**生物選擇選單**（12 個生物欄位
`0x51e0` 陣列 + 熱鍵 A-L/X、`FUN_2cdc_033d` 游標選單元件、逐一算出每隻生物的
名字字串與「花費」欄位 `0x51d4+idx*0x16+0x10`），玩家選定生物後檢查
`caster.SP-like欄位(0x4ec2) >= 生物花費`，通過後呼叫 `FUN_138d_34d6(param_1,
選定索引,param_2,param_3,10)` 真正執行召喚。**`[未解]`**：`FUN_138d_34d6`
本身（第 5 參數固定常數 `10`）沒有在本輪展開，是下一輪要追的目標——這是繼
`docs/re/16` §6 提到的 `FUN_138d_2f7e`／`FUN_138d_2fa0` 之後，**更接近答案**
的新入口點。

### 8.2 附身術（`effect_type 16`，`FUN_138d_0cb6`）—— 已解

```
成功率 = SP_invested * 300 / (2 * 目標.HP(0x4eba) + 目標.法力(0x4ec2))
roll = RNG(100)
若 roll > 成功率: 判定失敗（local_6 = 0）
否則依目標當前 field_4ed4 值做配對轉換：
    1↔0xc, 2↔0xb, 4↔0xe（雙向映射表）
若無法映射（值不在表內）：失敗
成功：目標.field_4ed4 = 映射後的值
```

本輪寫下這一段時，除數還掛著「`FUN_3016_000a` 未展開的縮放函式」；
`docs/re/23` §4 把它讀出來了 —— 那是 32-bit 除法常式，被除數 `SP × 300`，
除數 `2×HP + 法力`。**目標愈健康、法力愈滿愈難附身。**

`field_4ed4` 就是陣營欄位（`docs/re/20` §9.5 的完整值域）。這個效果是
**附身術（POSSESSION，法術 id 26）**：把單位搶到另一邊。配對表裡沒有
3／13（幻化生物），所以幻影不能被附身。原文標的「職業轉換」是錯的標籤。

### 8.3 二元狀態切換（`effect_type 9`，`FUN_138d_2fa0`）——與 Cast 路徑行為確認不同

```c
local_6 = 2;  // 預設「無變化/抵抗」
status = 目標.status(0x4ec4)
若 spell_id == 0x27:
    若 status < 2 且 RNG(100) < K*SP/M: local_6 = 1   // 施加
否則:
    若 status == 1 且 RNG(100) <= K*SP/M: local_6 = 0  // 清除
若 local_6 == 2: 印 "Spell fails"
否則: 目標.status(0x4ec4) = local_6; 印成功訊息
```

`docs/re/16` §6 已經發現「`effect_type 8/9` 在 Use/AI 路徑 vs Cast 路徑行為
不一致」，當時沒有足夠證據判定哪個對。**本輪確認**：Use 道具路徑的
`effect_type 9`（`FUN_138d_2fa0`）是一個**簡單的二元狀態開關**（0/1，依
`spell_id==0x27` 決定要設定還是清除），**不是**召喚機制——這與 `docs/re/09`
§5 描述的 Cast 路徑「幻術/召喚共用骨架、兩次擲骰取小值」明顯是不同的邏輯。
**結論（`[假設]` 待協調者裁決）**：`effect_type` 這個索引空間在 Cast 路徑
與 Use/AI 路徑下**語意不同**（同一個數字，不同上下文代表不同效果），
不是本輪讀碼深度不夠造成的錯覺。

---

## 9. 對既有文件的修正建議（未直接修改，列出供協調者裁決）

### 對 `docs/re/16-combat-details.md`

- §4.4「`FUN_138d_2e63` ... 本輪未能在時間內修復它的跳表」——**已修復**，
  內容見本文 §6.3，且證實與 `FUN_138d_3c81` 共用同一組 handler。
- §9 未解項清單第 1 條「`FUN_138d_2e63` 的跳表未修復」——**已解決**。
- §6「召喚生物的屬性來源仍未解出」——**部分推進**：新增 §8.1 找到的召喚
  選單入口與 `FUN_138d_34d6` 這個更接近答案的候選函式，建議更新「下一步」
  指向這裡而不是原本的 `FUN_138d_2f7e`/`FUN_138d_2fa0`（這兩個本輪已展開，
  分別是「無效果提示」與「二元狀態切換」，不是召喚，見 §6.1、§8.3）。

### 對 `docs/spec/02-combat.md`

- 「即死・束縛・召喚三類與 `Use` 效果套用仍缺」——**即死、束縛（施加+解除）、
  `Use` 效果套用可以從 DRAFT 補到接近 READY**（§7.1/§7.2/§7.3 有完整整數
  公式），**枯萎同樣可以補齊**（§7.4）。召喚仍缺（§8.1 只找到入口，
  `FUN_138d_34d6` 未展開）。
- 建議把 §7 的四條公式與 §6 的 18 項分派表原樣搬進 spec 的「即死/束縛/枯萎」
  小節，並標註本文件（`docs/re/18`）為證據來源。

---

## 10. 邊界與未動範圍

- `即死/束縛/枯萎` 三項公式的 `K`/`M` 具體數值（每個法術各自的係數）**仍未
  取出**——`docs/re/09` §7.1 記錄的 `[0x4e28]` 法術學校表位址問題，本輪未涉及，
  這是公式「結構」與「係數」的差別，結構已經齊了。
- `25be:139a` 誠實跳過，`FUN_25be_0263`/`FUN_25be_0e77` 的糾纏關係只做到
  「識別出問題」，沒有深入解開（不在 `138d` 段，跟四項目標無關）。
- 未修改 `workplace/orig/`、`CONTEXT.md`、`PLAN.md`、`docs/spec/`、
  `docs/formats/`、`docs/re/00`~`17`、`internal/`、`translations/`。
- 未 `git commit`／`git push`。

---

## 附錄：可重跑驗證片段

```bash
cd /home/anr2/cht/daemon_winter

# 1. 全檔跳表掃描（§2）
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
pattern = bytes([0x2e, 0xff, 0xa7])
idx, hits = 0, []
while True:
    idx = data.find(pattern, idx)
    if idx == -1: break
    hits.append(idx); idx += 1
print('總命中數:', len(hits))
for h in hits:
    pre = data[h-8:h]
    ok = (pre[0]==0x3d and pre[3]==0x73 and pre[4]==0x08
          and pre[5]==0x93 and pre[6]==0xd1 and pre[7]==0xe3)
    n = pre[1] | (pre[2]<<8)
    disp = int.from_bytes(data[h+3:h+5],'little')
    print(hex(h), 'n=', n, 'table_disp=', hex(disp), 'pattern_ok=', ok)
"

# 2. 第三張表內容（§1.2）
python3 -c "
import struct
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
def fo(seg,off): return seg*16+off-0xC400
f = fo(0x138d,0x3f95)
for i,e in enumerate(struct.unpack_from('<18H', data, f)):
    print(f'effect_type {i:2d}: 138d:{e:04x}')
"

# 3. 邊界檢查指令核對（§1.1）
objdump -D -b binary -m i386 -Maddr16,data16 \
  --start-address=$((0xb489)) --stop-address=$((0xb4a0)) \
  workplace/orig/demwin/DEMON.INT 2>/dev/null

# 4. 重跑完整標註 + 匯出（產生 workplace/ghidra/export/，約 1-2 分鐘）
./tools/ghidra_headless.sh orig/demwin/DEMON.INT demwin_int_sweep AnnotateJumpTablesSweep.java

# 5. 驗收：函式數、警告數、目標函式反編譯內容
tail -n +2 workplace/ghidra/export/functions.csv | wc -l          # 應為 394
grep -c "WARNING" workplace/ghidra/export/decompiled_all.c        # 應為 86
cat workplace/ghidra/export/decompiled/138d_3c81_FUN_138d_3c81.c  # 118 行，1 警告

# 6. 四項目標的 handler 函式內容
cat workplace/ghidra/export/decompiled/138d_0f0d_FUN_138d_0f0d.c  # 即死
cat workplace/ghidra/export/decompiled/138d_0e04_FUN_138d_0e04.c  # 束縛施加
cat workplace/ghidra/export/decompiled/138d_3098_FUN_138d_3098.c  # 束縛解除
cat workplace/ghidra/export/decompiled/138d_0fa5_FUN_138d_0fa5.c  # 枯萎
```
