---
name: ghidra-dos16-re
description: 用 docker 化的 Ghidra headless 反組譯 16 位元 DOS(MZ real mode)執行檔的完整流程、位址換算、跳表 override 修復,以及「decompiler 何時不可信」的判別法。觸發:反組譯 DOS 老遊戲/老 exe、Ghidra headless、16-bit real mode、segment:offset 對不回檔案位移、analyzeHeadless 跑不起來、Ghidra 12.x post-script 失敗、switch 被反編譯成亂碼、函式反編譯行數暴增、程式碼區域沒被發現。
---

# Ghidra 反組譯 16 位元 DOS 執行檔

> 來源:Demon's Winter(SSI 1989)引擎逆向,2026-07 實測。
> 適用於任何 MS-DOS MZ real mode 執行檔(1980s–90s 遊戲的主流格式)。

這份記的是**血淚換來的東西**,不是官方文件抄寫。核心結論:
Ghidra 對 16 位元 real mode 的自動分析**會靜默失敗並編造控制流**,
但只要知道怎麼修,它仍是目前最實用的選擇。

---

## 1. 環境(全程 docker)

`eclipse-temurin:21-jdk-jammy` + 下載 Ghidra release。**版本號寫死在 Dockerfile** 才可重現。
實測 Ghidra 12.1.2 可用。

跑分析務必加 `--user $(id -u):$(id -g)`,否則容器產出的檔案屬 root,host 讀寫受限。

### Ghidra 12.x 的兩個 API 陷阱

| 陷阱 | 症狀 | 解法 |
|---|---|---|
| **內建 Jython 被移除** | `.py` post-script 直接失敗,報 `PyGhidraScriptProvider` | post-script 改用 **Java** 寫,零額外相依 |
| **API 方法消失** | `DefinedDataIterator.definedStrings()` 不存在 | `javap` 反查 jar 找當前簽章;正確用法 `byDataInstance(program, Data::hasStringValue)` |

> **不要憑記憶寫 Ghidra script。** 先 `javap` 或翻該版本的 jar 確認簽章,比反覆試錯快得多。

---

## 2. [核心] segment:offset ↔ 檔案位移換算

**最容易錯、錯了會全盤皆錯的一步。**

### 陷阱:MZ header 的 CS:IP 不是 Ghidra 的位址

MZ header 裡的 entry `CS:IP` 是**連結時期的相對值**。Ghidra 載入時套自己的基準段
(實測為 `0x1000`),所以 header 寫 `2037:0009` 時,Ghidra 裡實際在 `3037:0009`。

判斷方法:反組譯 `entry` 函式,看它的立即值與 SS relocation 落點,與 header 值互相印證。

### 通用公式

```
file_offset = (segment − load_base_seg) × 16 + offset + hdrpara × 16
```

- `load_base_seg`:Ghidra 的載入基準段(實測 `0x1000`,**要自己驗證**,別假設)
- `hdrpara`:MZ header offset 8 的 uint16,乘 16 得 header 大小 = code 在檔案中的起點

```python
import struct
d = open(BIN, 'rb').read()
lastpage, pages, relocs, hdrpara = struct.unpack('<HHHH', d[2:10])
ip, cs = struct.unpack('<HH', d[20:24])
image_size = (pages - 1) * 512 + (lastpage or 512)
# image_size == len(d) → 無 overlay
```

### 驗證公式(必做,不可略過)

挑 3–4 個**已知的明碼字串**,從 `strings.csv` 取其 `segment:offset`,套公式算出檔案位移,
回原始 binary 讀那個位置比對內容。全部吻合才算數。

```python
off = (seg - 0x1000) * 16 + ofs + hdrpara * 16
assert d[off:off+len(s)] == s
```

### 另一個位址陷阱:`CALLF` 立即數未重定位

`disassembly.asm` 裡 `CALLF` 的立即數 segment 是**連結期原始值**,要 `+0x1000`
才是 Ghidra 顯示的 segment。檔案裡的 `CALLF 15be:0263` 實際指向 `FUN_25be_0263`。

**不要拿 `CALLF` 目標直接比對 `functions.csv`** —— 會全部對不上。
近跳轉顯示的 segment 只是當前段的別名,無跨段意義。

---

## 3. [最重要] 跳表 override —— 修好 switch 反編譯

### 問題長什麼樣

1980 年代編譯器的 switch 慣用法:

```asm
CMP AX,0xf                     ; 邊界檢查
JNC default
XCHG AX,BX                     ; ← 打斷 Ghidra 的資料流追蹤
SHL BX,0x1
JMP word ptr CS:[BX + 0x258f]  ; 跳表在 CS 段內,與程式碼交錯
```

Ghidra 對現代目標的 switch 還原不錯,但這組(跳表在程式碼段 + `XCHG` 打斷資料流 +
real mode segment 歧義)超出自動分析能力。後果:

- 跳表後的 case body 落在**完全沒被處理的間隙**,不屬於任何函式、`disassembly.asm` 也不列出
- **decompiler 不報錯,反而憑殘缺資訊編造控制流**

### 先把跳表讀出來(用 Python,不要靠 Ghidra)

```python
import struct
base = (SEG - 0x1000) * 16 + HDRPARA * 16
tbl  = base + TABLE_OFFSET
entries = struct.unpack_from(f'<{N}H', d, tbl)   # 同段內 offset,little-endian
```

判斷表是否合理:項數與 `CMP AX,n` 的 n 吻合、所有目標都落在該段的合理程式碼範圍、
多個 case 共用同一目標(default 分支)是正常的編譯器佈局。

### ★ 關鍵:光加 reference 沒用,要 writeOverride

這是整份文件最貴的一條。**第一版只做這些會「看起來成功」但實際沒修好**:

```java
instr.addOperandReference(0, target, RefType.COMPUTED_JUMP, SourceType.USER_DEFINED);
// disassembly 層級會正確、函式邊界會擴大 —— 但 C 輸出幾乎沒變
```

原因:**decompiler 走自己的 p-code 層級分析,不看 listing 上的 reference**。
必須額外寫入 decompiler 真正會讀的覆寫:

```java
import ghidra.program.model.pcode.JumpTable;

new JumpTable(branchAddr, destList, true, 0).writeOverride(function);
```

> 這個用法可直接參考 Ghidra image 隨附的官方範例
> `Ghidra/Features/Decompiler/ghidra_scripts/SwitchOverride.java`
> (GUI 裡「Override indirect jump destinations」背後的實作)。

完整流程:
1. 把跳表區域標成 word array(`createData`),避免被當程式碼反組譯
2. 對間接 `JMP` 加 `COMPUTED_JUMP` / `USER_DEFINED` reference(依 case 順序)
3. 觸發 disassembly,讓 code discovery 走進間隙
4. 重建函式本體(`CreateFunctionCmd`)
5. **`JumpTable.writeOverride()`** ← 沒這步前面全白做

### 實測效果(單一函式)

| 指標 | 修復前 | 修復後 |
|---|---|---|
| 反編譯行數 | 6557 | **333**(−95%) |
| unreachable 警告 | 59 | **1** |
| 憑空捏造的函式呼叫 | 有 | **0** |
| switch/case 結構 | 無 | 還原出來 |

---

## 4. decompiler 何時不可信(判別規則)

**這類錯誤在本專案造成過三次錯誤斷言,而且每次都被標成「已驗證」。**

| 誤判 | 真相 | 怎麼抓到 |
|---|---|---|
| 戰鬥選單 case 0xa = Attack | 0xa 是 Examine,Attack 是 case 5 | 讀原始跳表 |
| 某 case 呼叫某函式 | 該 call 全檔只有 2 個命中,都在別處 | 搜 `CALLF` 位元組樣式 |
| RLE 游標 `(cursor+64) % 4096` | 應為 `+64`,越界 `−4095`(column-major) | 該公式數學上只覆蓋 64 格 |

### 規則

- **含跳表的 switch,一律回原始指令讀**。跳表可直接用 Python 從 binary 解出來,比讀反編譯快也可靠。
- 反編譯輸出若**行數遠大於宣告的 byte 數**(例:1827 bytes 膨脹成 6557 行)、
  或帶大量 `Removing unreachable block` 警告 → 直接視為不可信。
- 間隙區用 `objdump -D -b binary -m i386 -Maddr16,data16` 對原始位元組硬解。
- **「已驗證」標籤本身要存疑** —— 特別是任何以反編譯 C 為唯一依據的結論。

---

## 5. 找到遊戲主邏輯:字串錨定

stripped binary 沒有符號,自動分析又常只覆蓋到 runtime。最有效的入口是字串:

1. 從 `strings.csv` 挑一個**語意明確、只可能出現在目標功能裡**的字串
2. 找引用它的函式
3. 反編譯 → 通常落在目標子系統中段,再往上下游擴散

實測:`Cast what spell:` 一步命中語意完整的施法選單邏輯。

**若某功能沒有專屬字串**(純數值邏輯,如 RNG、命中計算),改用「線索常數」:
找該演算法必然用到的魔術數字、查表基底位址、中斷號,從資料參照反查。

**還有一招**:字串在程式碼中的**物理位置**會洩漏用途 —— 編譯器通常把同一段程式碼
用到的字串排在一起,所以字串的鄰居就是弱型別的呼叫脈絡。
(實例:`TEMPLAT%d.DAT` 緊鄰某道具敘述文字,據此判定它不是角色範本而是道具資料。)

---

## 6. 判斷自動分析覆蓋夠不夠

不要只看函式總數,看這幾項:

- **函式在 segment 上的分布**:若大型 segment(通常是主邏輯)佔函式數多數,覆蓋算健康;
  若集中在幾個小 segment,可能只分析到 runtime。
- **decompile 錯誤位址**要區分兩種噪音:
  - 把 switch jump table 資料誤當程式碼位址 → **正常噪音**
  - 位址真的落在檔案範圍外 → 合法(如中斷向量表),但要確認不是分析走偏
- 函式邊界判定有**輕微非決定性**,同一 binary 重跑會有幾個函式差異(實測 348–353 浮動)。
  別把這個當失敗。

---

## 7. 協調多 agent 時的紀律

- **agent 還在跑時量到的中間產物不能當結論。** 實測踩過:重跑途中取樣看到
  export 反覆變動(50→360→120 個函式),誤判為「產出不穩定」,實際是迭代暫態。
- **`git add` 要指名檔案,不要加目錄** —— 多 agent 並行時會把別人仍在編輯的檔案掃進版控。
- 每個 agent 的結論**協調者要獨立複核一條證據鏈**才收(重讀指令、重算位址、
  視覺產物肉眼比對),模型越便宜、輪數越多,這條越不能省。

---

## 8. 其他 real mode 工具的現況(2026-07 調查)

| 工具 | 16-bit real mode 支援 | 備註 |
|---|---|---|
| **Ghidra** | 可用,但需本文的跳表 override 修復 | 目前最實用 |
| **Reko** | 真的支援,做 MZ relocation、有跳表還原 | 失敗時**明確拋例外**而非靜默捏造,安全性較好;GPL-2.0,活躍維護 |
| RetDec | ❌ 官方只列 32/64-bit | |
| Snowman | ❌ 已封存,平坦記憶體假設是架構性的 | |
| dcc(Cifuentes 1994) | 專為 DOS 而生,但最活躍的 fork 自承多處有 bug | |
| radare2 / rizin | 反組譯可以,但 decompiler 不重建控制流 | `rz-ghidra` 只是包 Ghidra 的 P-code |
| IDA Free | ❌ **無法自動化** —— `-B` 批次與 `-S` 腳本都被移除,無 IDAPython | 只剩 GUI |
| IDA Pro | 程式碼發現與跳表較強 | **但 Hex-Rays 對 16-bit x86 沒有 decompiler**,拿不到 C |

> 交叉驗證的價值:Reko 獨立找出 373 個函式,與 Ghidra 的 353 個在 11 個主要 segment
> 中有 10 個吻合(校正基準段後)—— 兩個獨立工具互相佐證比單一工具的自信有用。

---

## 對接

- 反編當 oracle、乾淨重寫的母方法論:`retro-game-remake`
- 查「這個值從哪來」的靜態溯源紀律:`rulebook/62-static-provenance-trace`
- 靜態追不動時的第三條路:`rulebook/64-re-screenshot-oracle`
- 有 reference 的專案怎麼驗收:`rulebook/65-verify-against-reference-not-internal-signals`
