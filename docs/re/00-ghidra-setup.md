# Ghidra Headless 反組譯環境（DEMON.INT）

本檔記錄 Demon's Winter（SSI, 1988）逆向工程專案的 Ghidra headless 環境：怎麼跑、
DEMON.INT 的載入設定、16-bit real mode 定址換算、踩過的雷，以及匯出檔案格式。
給後續分析 agent 和未來的自己看。

## 環境現況（2026-07-24 建置）

- Ghidra **12.1.2**（tag `Ghidra_12.1.2_build`，發佈 2026-06-05，GitHub Releases 當時最新穩定版）
- base image `eclipse-temurin:21-jdk-jammy`（Ghidra 12.x 系列要求 JDK 21+）
- image：`demwin-ghidra:12.1.2`，Dockerfile 在 `docker/ghidra/Dockerfile`
- 全程 docker，系統上沒有另外裝 Ghidra / JDK / Python

## 怎麼跑

### 1. 建 image（第一次或改了 Dockerfile 才需要）

```bash
cd /home/anr2/cht/daemon_winter
docker build -t demwin-ghidra:12.1.2 -f docker/ghidra/Dockerfile docker/ghidra
```

### 2. 跑分析（完整自動分析 + 匯出）

```bash
cd /home/anr2/cht/daemon_winter
./tools/ghidra_headless.sh orig/demwin/DEMON.INT demwin_int ExportAnalysis.java
```

- 第一個參數是 binary 相對於 `workplace/` 的路徑
- 第二個參數是 Ghidra 專案名稱（專案檔存到 `workplace/ghidra/<專案名稱>/`）
- 第三個參數（選用）是要跑的 post-script 檔名，去 `tools/ghidra_scripts/` 找
- 只想跑自動分析、不跑 post-script：省略第三個參數即可
- 重跑不用先清專案，腳本已經帶 `-overwrite`

實測跑一次（自動分析 + 匯出）約 **50～60 秒**（52 秒自動分析 + 匯出階段幾秒）。

### 3. 看結果

匯出檔案都在 `workplace/ghidra/export/`（不入版控，見 `.gitignore`）：

```bash
ls workplace/ghidra/export/
# functions.csv  strings.csv  disassembly.asm  decompiled_all.c  decompiled/
```

## DEMON.INT 的載入設定

| 項目 | 值 | 來源 |
|---|---|---|
| 檔案大小 | 173,380 bytes | `ls -la` |
| MZ header 大小 | 960 paragraphs = 0x3C00 bytes | header offset 0x08（`e_cparhdr`） |
| code 起始 file offset | **0x3C00** | header 大小本身 |
| pages | 339（0x0153） | header offset 0x04 |
| lastpage | 324（0x0144） | header offset 0x02 |
| relocations | 3807（0x0EDF） | header offset 0x06 |
| entry point（header 原始值） | CS:IP = **2037:0009** | header offset 0x16/0x14 |
| SS:SP（header 原始值） | **21F0:A09A** | header offset 0x0E/0x10 |
| overlay | 無（image size 恰等於 file size） | 已知事實 |

Ghidra loader 自動辨識為 **`Old-style DOS Executable (MZ)`**，Language/Compiler 自動選定
**`x86:LE:16:Real Mode:default`**，跟已知設定一致。`tools/ghidra_headless.sh` 仍然明確帶
`-processor "x86:LE:16:Real Mode"` 強制指定，不依賴自動辨識（穩定性考量：換一支別的
DOS binary 時語言辨識可能不準，明確指定比較保險）。

## 16-bit real mode 定址：segment:offset 與檔案位移換算

**這是後續 agent 對回原始檔案位移最需要的一段，務必讀完再動手。**

### 關鍵發現：Ghidra 實際載入基準是 segment 0x1000，不是 header 原始的 0x2037

header 裡的 `e_cs`（0x2037）、`e_ss`（0x21F0）是**連結時期（relocation-relative）的原始值**，
DOS 的慣例是：真正執行時的 segment = 這個原始值 **加上**「載入時的 base segment」。
Ghidra 匯入這種 MZ 檔案時，內部選定的 base segment 是 **0x1000**（對應 file offset
0x3C00，也就是 header 結束、程式碼真正開始的位置）。

**驗證證據**（用 `functions.csv` 反查）：
- Ghidra 標出的 entry point 函式名稱直接叫 `entry`，位址是 **`3037:0009`**
  （= 0x2037 + 0x1000），大小 277 bytes，反組譯內容是：
  ```
  3037:0009  MOV BP,0x31f0
  3037:000c  TEST BP,BP
  3037:000e  JNZ 0x3000:0385
  ```
  第一行就是把 `0x31F0` 存進 BP —— 而 0x31F0 正好等於 header 原始 SS（0x21F0）
  加上同一個 base segment 0x1000（`0x21F0 + 0x1000 = 0x31F0`）。這是典型 small/medium
  model DOS 程式進場時初始化資料段暫存器的寫法，兩條證據互相印證，載入基準
  確實是 0x1000。
- 所有 `strings.csv` 裡的字串全部落在 segment **31F0**（例如 `31f0:046f` =
  `Cast what spell:`），跟上面推出的「runtime SS = 31F0」完全吻合 —— 這顆
  16-bit 程式是 DS=SS 的小記憶體模型，全域字串常數就放在堆疊段。

### 換算公式（已用 4 個已知字串逐一驗證，見下方驗收章節）

```
file_offset = segment * 16 + offset - 0xC400
```

推導：Ghidra 的 base segment 0x1000 對應 file offset 0x3C00，而 16-bit real mode
的物理位址公式本來就是 `segment*16 + offset`（mod 1MB）。兩者相減：

```
file_offset = (segment*16 + offset) - (0x1000*16) + 0x3C00
            = segment*16 + offset - 0x10000 + 0x3C00
            = segment*16 + offset - 0xC400
```

反向（file offset 換算 segment:offset，取 offset 落在 [0, 0xFFFF) 的正規化表示）：

```python
def seg_off_from_file_offset(file_off):
    linear = file_off + 0xC400
    seg = linear // 16
    off = linear % 16
    return seg, off
```

⚠️ **不要用 header 原始的 e_cs（0x2037）/ e_ss（0x21F0）直接當 Ghidra 位址用**——
那是連結時期的值，Ghidra 顯示的是加了 base segment 0x1000 之後的值
（`entry` 函式在 `3037:0009`，不是 `2037:0009`）。这两组数字只差一个固定的
0x1000，别搞混。

## 匯出檔案格式與怎麼用

都在 `workplace/ghidra/export/`：

| 檔案 | 格式 | 用途 |
|---|---|---|
| `functions.csv` | `address,name,size_bytes,is_thunk` | 函式清單，位址是 Ghidra 的 segment:offset |
| `strings.csv` | `address,length,content` | 字串清單，**後續「字串錨定找函式」的關鍵索引** |
| `disassembly.asm` | `address\tinstruction` 逐行 | 全域反組譯 |
| `decompiled_all.c` | 全部函式反編譯串接，`// ==== 函式名 @ 位址 ====` 分隔 | 方便整檔 grep |
| `decompiled/<seg>_<off>_<name>.c` | 每個函式各一檔 | 需要單一函式時直接開檔 |

「字串錨定找函式」的具體做法：
1. 在 `strings.csv` grep 目標字串，拿到它的 segment:offset（例如 `31f0:046f`）。
2. 用上面的換算公式轉成 file offset，回原始 `.INT` 檔案核對內容（雙重確認沒抓錯字串）。
3. 在 `decompiled_all.c` 或 `disassembly.asm` 裡搜尋這個字串位址的 **offset 值**
   （字串函式呼叫慣例是「call 印字函式，帶 offset 常數當參數」，segment 通常靠
   當時已設好的資料段暫存器隱含，不會每次都出現在呼叫端 —— 见下方驗收章節的實例）。
4. 反查是哪個函式呼叫了它，那就是字串的使用點；再往外層追呼叫鏈找到觸發它的遊戲邏輯。

## 踩到的雷

### 1. Ghidra 12.x 拿掉了內建 Jython，`.py` post-script 直接失敗

第一次嘗試用 Jython 風格的 `.py` script 當 post-script（`ExportAnalysis.py`），
`analyzeHeadless` 直接報錯：

```
ghidra.app.script.GhidraScriptLoadException: Ghidra was not started with PyGhidra. Python is not available
```

Ghidra 11.3 起把 Python 整合換成 **PyGhidra**（CPython + jpype，需要另外裝
pip 套件、對版本、走 `pyghidraRun` 啟動），到 12.x 已經完全不含 Jython
script provider。這代表要嘛額外背一層 Python venv + jpype 版本相依，要嘛
換語言寫 script。

**解法**：改用 **Java** 寫 post-script（`ExportAnalysis.java`）。`analyzeHeadless`
內建 `javac` 編譯執行 GhidraScript，JDK image 本身就夠，零額外相依，也不用煩惱
PyGhidra 版本要跟 Ghidra 版本對齊。這個專案的匯出腳本因此是 Java 不是 Python，
之後要加新的 headless 匯出邏輯建議照同一套路。

### 2. `DefinedDataIterator` 沒有 `definedStrings()` 這個 static method（API 記憶過期）

一開始想抄 `references/01-decompile-oracle.md` 裡 kb 記憶的寫法找字串，
用了不存在的 `DefinedDataIterator.definedStrings(Program)`，編譯期就報
`cannot find symbol`。用 `javap` 對 `SoftwareModeling.jar` 反查這個類別
實際有的 method，正確用法是：

```java
DefinedDataIterator.byDataInstance(currentProgram, Data::hasStringValue)
```

教訓：Ghidra API 隨版本會變，跨版本前先用 `javap -p` 或翻 `docs/ghidra_stubs/`
確認方法簽章還在，不要照舊筆記的方法名直接抄。

### 3. 反編譯偶發 `AddressOutOfBoundsException`（1 個函式反編失敗，不影響整體）

匯出反編譯時有 1 個函式（`Offset must be between 0x0 and 0x10ffef, got 0x7451990`）
反編失敗，是 decompiler 內部處理某個間接跳轉/switch 分析時算出一個超出
segmented address space 範圍的常數。Script 裡有包 try/catch，這個函式被略過，
其餘 347/348 正常反編完成，不影響整體匯出。屬於 Ghidra decompiler 對這種
16-bit 混合定址程式的已知邊界案例，暫不深究，之後真的要分析到那個函式時
再手動處理。

### 4. 別把 header 原始 CS/SS 當成 Ghidra 位址用

見上面「換算公式」章節——header 原始值（`2037:0009` / `21F0:A09A`）跟
Ghidra 實際顯示的 runtime 位址（`3037:0009`）差了一個固定的 base segment
0x1000。已用 `entry` 函式反查證實。這點在對照「已知事實」文件跟 Ghidra
實際輸出時特別容易搞混，混用會讓後續換算全部錯位 0x10000（實測踩過，見
「驗收證據」章節第一次算出來的位移跟實際差了正好 0x10000）。

### 5.（2026-07-25 新增，兩組人各自踩到）自動分析有「間隙」，反編譯會在間隙上編造控制流

**這是本專案迄今最貴的一類錯誤，已造成三次錯誤斷言，務必先讀。**

Ghidra 對某些函式只展開了**前段**，函式中後段（尤其跳表後面的 case body）
落在它完全沒處理的「間隙」裡——那些位元組不屬於任何已知函式，
`disassembly.asm` 也不會列出。而 decompiler 遇到這種情況**不會報錯，
反而會憑殘缺資訊編造出一套看似合理的控制流**。

實際踩過的三次：

| 誤判 | 真相 | 怎麼抓到 |
|---|---|---|
| `docs/re/06`：戰鬥選單 case 0xa = Attack（還標「已驗證」） | 0xa 是 Examine，Attack 是 case 5 | 讀 `138d:258f` 原始跳表 |
| `docs/re/04`：`FUN_222f_0b0e` 的 case 0x4 呼叫 `FUN_25be_0263` | 該 call 根本不在此函式 | 全檔搜 `CALLF` 位元組樣式，只 2 個命中且都在別處 |
| `docs/re/03`：SUM.MAP 游標 `(cursor+64) % 4096` | 應為 `+64`，越界 `-4095`（column-major） | 該公式數學上只覆蓋 64 格，逐字實作丟掉 98% 資料 |

**規則**：

- **含跳表的 switch，一律回原始指令讀**，不可信反編譯輸出。
  跳表本身可以直接用 Python 從 binary 解出來（`struct.unpack_from('<NH', ...)`），
  比讀反編譯快也可靠。
- 一個函式的反編譯輸出若**行數遠大於宣告的 byte 數**（例：1827 bytes 膨脹成 6557 行）、
  或帶大量 `Removing unreachable block` 警告，直接視為不可信。
- 間隙區可用 `objdump -D -b binary -m i386 -Maddr16,data16` 對原始位元組硬解。

### 6. `disassembly.asm` 裡的 `CALLF` 目標未重定位

`CALLF` 指令的立即數 segment 是**連結期的原始值**，要 `+0x1000` 才是 Ghidra 顯示的 segment。
例如檔案裡的 `CALLF 15be:0263` 實際指向 `FUN_25be_0263`。

**不要拿 `disassembly.asm` 的 `CALLF` 目標直接去比對 `functions.csv`**——會全部對不上。
同理，近跳轉（near jump/call）顯示的 segment 只是當前段的別名，沒有跨段意義。

（這條與第 4 點同源，都是 base segment 0x1000 造成的；但第 4 點講 header，
這條講指令內的立即數，兩處都會咬人。兩組獨立進行的分析各自踩到一次。）

## 自動分析覆蓋品質評估

**跟任務說明書預警的最壞情況（「stripped binary 遊戲主邏輯常在 indirect
call/jump table 之後，自動分析往往只覆蓋到 runtime」）比起來，這次實測結果
明顯比較好，如實記錄：**

- 自動分析找到 **353 個函式**，分布在 **55 個不同的 segment**。
  （註：Ghidra 的函式邊界啟發式分析對這種 real-mode 二進位帶一點非決定性，
  重跑幾次函式總數會在 348～353 之間小幅浮動，不影響結論。）
- 大型 segment（很可能是遊戲主邏輯本體，非執行環境樣板）函式數量都不低：
  `1d9f`=65、`1000`=28、`310e`=27、`222f`=25、`217b`=24、`2cdc`=22、
  `138d`=22、`1990`=19、`278d`=17、`25be`=14、`3196`=9……這些加總約佔
  353 個函式的 8 成以上；剩下十幾個 segment 大多只有 1～2 個函式，看起來
  比較像編譯器 runtime library（浮點運算模擬、記憶體管理等小工具函式）。
- **直接證據**：字串錨定驗證時，`Cast what spell:` 反查到的函式
  `FUN_1000_293d`（見下方「驗收證據」）反編出來是一段語意完整、能看懂的
  施法選單邏輯（讀角色可用法術清單、拼接法術名字串、呼叫 `FUN_1000_2a53`
  做法術學校選擇）——這已經是遊戲主邏輯層級，不是 runtime/DOS wrapper。
- 自動分析階段跑出的 pcode 錯誤（`Unable to resolve constructor at
  2000:0133` 之類）逐一核對後，只有 `0000:4b80` 這一個真的落在檔案範圍外
  （換算成 file offset 是負值，對應到 real mode 中斷向量表那類「合法但不在
  這個檔案裡」的位址）；其餘（`2000:0133`、`1000:5000`、`1000:04b3`、
  `1000:fc05`、`2000:d230`、`2000:6ef0`）換算後其實都落在檔案範圍**內**，
  是 decompiler 的 switch/間接跳轉分析把資料位元組誤當成程式碼位址去試探
  導致的失敗，**不是**「這塊區域沒被分析到」的訊號，只是分析嘗試撞到非
  程式碼位元組時的正常噪音。

**結論**：自動分析對這顆 16-bit real mode binary 的覆蓋品質**足夠拿來直接動手做
後續反組譯分析**，字串錨定找函式這條路完全可行且已用四個已知字串逐一驗證通過。
沒有觀察到「只覆蓋 runtime、進不了遊戲主碼」的現象——但這不代表所有分支都被找到，
間接呼叫/跳表後面沒被自動辨識成 code 的區塊仍可能存在（尤其那十幾個只有 1～2
個函式的小 segment，值得之後針對性補查），後續 agent 遇到「這段邏輯應該有函式
但 functions.csv 查不到」時，可以合理懷疑是間接跳轉沒被自動展開，這時候照
`~/.claude/knowledge-base/retro-cht/retro-game-remake/references/01-decompile-oracle.md`
的「線索常數」破法（decompile 全部函式後 grep 已知常數）繼續往下挖。

## 驗收證據

### 1. `analyzeHeadless` 實際跑完，無 fatal error

自動分析總耗時 44～52 秒（兩次跑的數字略有差異），`REPORT: Analysis
succeeded for file` / `REPORT: Import succeeded` 都有出現，過程中的
`WARN`/`ERROR` 都是 decompiler 對個別函式的 pcode 分析噪音（見上一節），
不影響整體匯入與匯出。

### 2. `functions.csv` 函式數量

**353** 個函式（不含 header 列；重跑會在 348～353 間小幅浮動，見上一節說明）。

### 3. 四個已知字串，`strings.csv` 逐一 grep 確認

```
$ grep -F "Cast what spell:" strings.csv
31f0:046f,16,Cast what spell:

$ grep -F "Turn Undead" strings.csv
31f0:07ca,11,Turn Undead

$ grep -F "God Runes" strings.csv
31f0:04c6,9,God Runes

$ grep -F "CONGRATULATIONS! You have won Demon's Winter." strings.csv
31f0:066a,86,CONGRATULATIONS! You have won Demon's Winter. I hope you have enjoyed your  adventure.
```

四個全部找到。且逐一用「換算公式」轉成 file offset 後直接讀原始 `DEMON.INT`
位元組核對，內容完全吻合（見下表），確認公式正確、Ghidra 位址沒有跑掉：

| 字串 | Ghidra 位址 | 換算 file offset | 直接讀檔驗證 |
|---|---|---|---|
| `Cast what spell:` | 31f0:046f | 0x25f6f | 吻合 |
| `Turn Undead` | 31f0:07ca | 0x262ca | 吻合 |
| `God Runes` | 31f0:04c6 | 0x25fc6 | 吻合 |
| `CONGRATULATIONS! ...` | 31f0:066a | 0x2616a | 吻合 |

### 4. 挑一個字串反查函式：`Cast what spell:` → `FUN_1000_293d`

`Cast what spell:`（`31f0:046f`，offset 部分 `0x46f`）在
`decompiled_all.c` 裡被 `FUN_1000_293d` 呼叫（`FUN_1d9f_1361(0x46f)`——
segment 靠當時已設好的資料段暫存器隱含傳遞，呼叫端只帶 offset 常數，
這也是字串錨定要用「offset 值」去 grep、不能整個 `segment:offset` 一起
搜的原因）。完整反編譯結果：

```c
undefined2 __cdecl16far FUN_1000_293d(void)

{
  char *pcVar1;
  char cVar2;
  undefined2 uVar3;
  undefined2 unaff_SS;
  undefined2 unaff_DS;
  char acStack_1c [20];
  int iStack_8;
  int local_6;
  uint local_4;

  while( true ) {
    local_4 = FUN_1000_053c(0);
    if (*(byte *)((int)*(undefined4 *)0x4c76 + 0x9a) == local_4) {
      return 0;
    }
    if (*(char *)((int)*(undefined4 *)0x4c7e + local_4 * 0x104 + 0x102) == '\0') break;
    FUN_1d9f_1361(*(undefined2 *)0x25,*(undefined2 *)0x27);
    FUN_1d9f_0d4e();
    FUN_1d9f_0d64();
    FUN_1d9f_28f9(*(undefined2 *)0x4c84);
  }
  FUN_1d9f_28f9(*(undefined2 *)0x4c84);
  local_6 = 0;
  while ((local_6 < 10 &&
         (cVar2 = *(char *)((int)*(undefined4 *)0x4c7e + local_4 * 0x104 + local_6),
         acStack_1c[local_6] = cVar2, cVar2 != '\0'))) {
    local_6 = local_6 + 1;
  }
  pcVar1 = acStack_1c + local_6;
  local_6 = local_6 + 1;
  *pcVar1 = ':';
  acStack_1c[local_6] = '\0';
  FUN_2cdc_1727();
  FUN_1d9f_1361(acStack_1c);
  FUN_1d9f_1361(0x46f);              // <-- "Cast what spell:" (31f0:046f)
  iStack_8 = FUN_1000_2a53(0,local_4);
  if (iStack_8 != 0) {
    uVar3 = FUN_1000_11e5(local_4,0,iStack_8 + -1,0);
    return uVar3;
  }
  return 0;
}
```

反編出來的邏輯完全能對上遊戲行為（[推測] 標記，尚未逐句對照原始碼確認每個
細節，但整體結構清楚）：迴圈找出可以施法的角色（`*(char*)(角色資料+0x102)`
非零 = 有法術可施展）、把角色名字組成 `"<角色名>:"` 印出來、接著印
`"Cast what spell:"`、再呼叫 `FUN_1000_2a53(0, local_4)` 做法術學校選單
（這個函式反編出來確實是九宮格式選單邏輯，對應施法介面的法術學校分類，
`FUN_1000_2a53` 完整反編譯內容見 `workplace/ghidra/export/decompiled/1000_2a53_FUN_1000_2a53.c`）。
這條字串→函式→呼叫鏈的證據，說明環境不只是「跑得起來」，反編譯品質也
足以支撐後續逐函式分析遊戲邏輯。

## 給後續 agent 的建議下一步

1. **先用字串錨定，不要從 entry point 硬啃。** `strings.csv` 裡的字串（UI
   提示、法術名、道具名、戰鬥訊息……）幾乎都是找到對應遊戲邏輯最快的路，
   已驗證可行（見上）。
2. **换算公式先記住，位址要對回檔案位移前一定套公式**：
   `file_offset = segment*16 + offset - 0xC400`。
3. **不要混用 header 原始 CS/SS 跟 Ghidra 顯示的位址**——差一個 base segment
   0x1000，已在「踩到的雷」章節記錄。
4. 遇到「這段邏輯應該有函式但 `functions.csv` 查不到」，先假設是間接
   跳轉/switch table 沒被自動展開，照 `62-static-provenance-trace` 的
   反向溯源 SOP 或 `01-decompile-oracle.md` 的「線索常數」破法往下挖，
   不要直接下「封死、要動態」的結論。
5. 需要新的匯出邏輯，照 `tools/ghidra_scripts/ExportAnalysis.java` 的
   模式寫 Java GhidraScript（不要碰 `.py`，Ghidra 12.x 這個環境沒配
   PyGhidra）。
