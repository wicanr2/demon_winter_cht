# 16-bit Real Mode 反編譯工具調查與自製工具評估

本檔調查市面上「號稱支援 DOS 16-bit real mode」的反編譯/反組譯工具，逐一查證
實際支援程度，並評估自製一支針對 `DEMON.INT` 的專用分析器的可行性。

**背景**：Ghidra 12.1.2 對這顆 173 KB、3,807 個 relocation 的 16-bit real mode
binary 自動分析覆蓋不完整——某些函式只展開前段，跳表後的 case body 落在
「間隙」裡；decompiler 在間隙上不報錯，反而編造控制流，已造成三次錯誤斷言
（見 `docs/re/00` 第 5 節）。手動用 Ghidra post-script 標註跳表也只部分成功
（函式數 353→362，但反編輸出從 353 個掉到 50 個，關鍵函式整個消失）。

**查證方式**：GitHub API/`gh` CLI 讀原始碼與 repo metadata（stars、license、
最後 push 時間）、WebFetch 官方文件、以及**實際下載 Reko 0.12.3 CLI release，
在 docker（`mcr.microsoft.com/dotnet/runtime:8.0`）裡對真正的 `DEMON.INT`
跑一次完整 `reko decompile`**——這是本次調查裡唯一一項不只讀文件、而是
拿專案自己的檔案實測過的工具，結果詳見第一部分 Reko 小節與附錄。

---

## 第一部分：現有工具查證結果

### 總覽表

| 工具 | 真支援 16-bit real mode segmented addressing？ | MZ relocation | 跳表/switch 還原 | headless/批次 | 授權 | 維護狀態 |
|---|---|---|---|---|---|---|
| **Reko** | ✅ 有專門 arch `x86-real-16` + env `ms-dos` | ✅ 有實作，`MsdosImageLoader.Relocate()` 真的套用 segment fixup | ⚠️ 有通用機制（`BackwardSlicer`/`VectorBuilder`），**實測對本檔案部分函式會拋例外中止**，但拋例外≠編造假控制流 | ✅ `reko decompile --arch ... --env ...` CLI | GPL-2.0 | 非常活躍（最後 commit 2026-07-24，查證前一天） |
| **dcc**（nemerle fork）| ✅ 原始設計目標就是 DOS 16-bit | ✅（原始設計目標） | ⚠️ 有，但作者自承多處仍有 bug | ✅ CLI | GPL-2.0 | 幾乎停滯（最後 push 2025-01-17，維護者明說沒空） |
| **RetDec** | ❌ 官方 README 明列只支援 32/64-bit（x86、ARM、MIPS、PowerPC / x86-64、ARM64），沒有 16-bit | N/A | N/A | ✅ CLI | MIT | 有限維護（仍在 push，但功能面幾乎不動） |
| **radare2 / rizin** | ⚠️ 反組譯層可以（`asm.bits=16`），但**沒有**做 16-bit segmented 定址的專門建模 | ⚠️ 能辨識 MZ 格式，relocation 細節未查證到位 | ❌ 內建 `pdc` 偽反編譯器明確**不做**「switch、if/else、for/while」控制流重建；`r2dec`/`rz-ghidra` 這類外掛也沒查到對 16-bit real mode 的專門支援 | ✅ | LGPL2.1（radare2）/ LGPL3（rizin） | 活躍 |
| **Snowman** | ❌ | N/A | N/A | 原有 GUI+CLI | GPL | **已封存（archived, 2023-03-09）**；GitHub issue #131「Allow decompilation of 16-Bit files」明確是「從未實作」，作者的回覆說明 16-bit 支援需要重寫 disassembler/calling convention 假設（flat memory model 是架構性前提） |
| **dis86**（xorvoid）| ✅ 專為 16-bit real-mode DOS **遊戲**逆向設計，作者的開發動機與本專案完全同構（「分析、重新實作 1990 年代早期 DOS 遊戲」） | ❌ **明確不支援**——README 寫明「decompiler 目前只接受 text segment 的 flat binary 區域，不處理常見二進位格式（例如 MZ）」，relocation 要使用者自己先剝離 | ⚠️ 部分：while/if/switch 有做，**if-else 未實作**，作者部落格展示過把 `jmp WORD PTR cs:[bx+0x1c]` 這類跳表轉成 switch，但坦承這是他們重寫過一次架構才做到的 | ✅ CLI（`--emit-code`/`--emit-dis`/`--emit-graph` 等） | ⚠️ **repo 沒有 LICENSE 檔**（GitHub API `license: null`），未標示授權，法律上不確定能否直接用/改 | 活躍但屬個人專案內部工具（458 commits，最後 push 2026-03-11，57 stars）；作者明言「不保證對自己專案以外的用途能直接運作」 |

### 逐項細節

#### Reko（最值得深入的候選）

- 命令列語法：`reko decompile --arch x86-real-16 --env ms-dos --base <seg:off> <file>`
  ——直接支援 `x86-real-16` 架構與 `ms-dos` 環境。
  來源：[Reko FAQ](https://github.com/uxmal/reko/wiki/FAQ)。
- MZ loader 原始碼 `src/ImageLoaders/MzExe/MsdosImageLoader.cs`：實際讀取
  `ExeLoader.e_lfaRelocations`／`e_cRelocations`，逐筆 relocation entry 算出
  `offset = segOffset*0x10 + offset`，把原始 segment 值加上 `addrLoad.Selector`
  寫回，並用 `AddSegmentReference()` 建立新 segment——這是貨真價實的 MZ
  relocation fixup，不是聲稱支援而已。Entry point 用 `e_cs+addrLoad.Selector`
  與 `e_ip` 組成 `Address.SegPtr`，跟本專案在 `docs/re/00` 推出的
  「base segment + header 原始值」邏輯完全同構。
  來源：[MsdosImageLoader.cs](https://github.com/uxmal/reko/blob/master/src/ImageLoaders/MzExe/MsdosImageLoader.cs)。
- 跳表還原：`src/Decompiler/Scanning/BackwardSlicer.cs` + `VectorBuilder.cs`
  是通用（跨架構）的間接跳轉回溯分析機制，配 GUI 端的 `JumpTableDialog`／
  `JumpTableInteractor` 供人工介入。來源：GitHub code search
  （`gh api search/code`，命中路徑列於上）。
- License：**GPL-2.0**（`gh api repos/uxmal/reko` 查得）。
- 維護狀態：`pushed_at: 2026-07-24T16:40:52Z`——查證當下（2026-07-25）前一天
  才有 commit，2593 stars，165 個 open issues（有人在用、有人在報 bug，
  不是死專案）。

**實測**（見附錄完整紀錄）：下載 `CmdLine-0.12.3` release（.NET 8，
在 `mcr.microsoft.com/dotnet/runtime:8.0` docker 容器內跑，全程未在系統
Python/直接環境安裝任何東西），對真正的 `DEMON.INT` 跑
`reko decompile --arch x86-real-16 --env ms-dos`：

- **成功找到 373 個函式**（Ghidra 是 353 個，量級相近）。
- **函式分布的 segment 集合與 Ghidra 高度重合**：Ghidra 報的 11 個「大型」
  segment（`1000, 138d, 1990, 217b, 222f, 25be, 278d, 2cdc, 310e, 3196, 1d9f`，
  這些是 Ghidra 內部 `+0x1000` base 之後的值）減去 `0x1000` 之後，跟 Reko
  獨立算出的 segment 集合有 **10/11 完全對上**
  （`0000, 038d, 0990, 117b, 122f, 15be, 178d, 1cdc, 210e, 0d9f`）——兩套
  完全不同程式碼庫、不同演算法，各自從 entry point 做程式碼發現，結果高度
  一致，這是「Ghidra 找到的函式邊界大致可信」的獨立交叉驗證證據。
- **跳表確實有還原成功的案例**：輸出的 C 檔裡有 switch 陳述式，其中一個
  是 inline case body（直接操作 `ss->*`/`ds->*` 記憶體，不是單純呼叫轉發），
  另一個是 8 個 case 分別呼叫 `fn038D_BE83()`／`fn038D_FEAB()`／…的呼叫轉發表。
- **但也真的會在複雜函式上失敗**：跑一次完整分析，log 裡出現大量
  `error: An error occurred while rewriting procedure to high-level language`
  （`Index was out of range` / `Index was outside the bounds of the array`，
  出自 `CompoundConditionCoalescer`／`RegionFactory`／`StructureAnalysis`）
  以及一次 `Boundless recursion in fn0D9F_282F while finding definitions of sp`
  （SSA 定義查找無窮遞迴，出自 `SsaTransform.IdentifierTransformer`）。
  **關鍵差異**：Reko 這些失敗都是**顯式拋例外、標成 `error:`，該函式直接沒有
  C 輸出**，不會像 Ghidra 那樣在間隙上悄悄編造一段看起來合理但其實錯的
  控制流。對本專案的痛點（「不報錯的假控制流」）而言，Reko 的失敗模式
  明顯比較安全——雖然可用率一樣有限，但至少「失敗」跟「可信」是可以分開的。
- **未能解開跟 Ghidra 一樣的間隙**：手動追查 Ghidra 標註跳表後消失的
  `FUN_222f_0b0e`（對應 Reko 的原始 segment `122f` offset `0b0e`），Reko
  在 `122f` segment 找到的函式清單裡**同樣沒有這個位址**（`0832` 之後
  直接跳到 `1321`，中間 ~0x2800 bytes 完全沒有函式邊界），代表這不是
  Ghidra 特有的啟發式弱點，而是這段程式碼本身只能透過已知跳表的位元組
  內容才找得到——兩個獨立工具都需要「外部提供的跳表知識」才能补上。
  這直接支持第二部分「自製工具要顯式跟隨已知跳表」的設計方向。

#### dcc（Cristina Cifuentes, 1994 博士論文產物）

- 原始碼公開取得無問題，至少 4 個 GitHub fork/mirror：
  [8l/dcc](https://github.com/8l/dcc)（原始碼原樣保存）、
  [Nico01/dcc](https://github.com/Nico01/dcc)、
  [biappi/DCC](https://github.com/biappi/DCC)（號稱移植到現代機器，
  但 2 stars、最後 push 2021-07、`license: null`，README 沒提供編譯/測試
  細節，查無法確認真的能跑）、
  [nemerle/dcc](https://github.com/nemerle/dcc)（「heavily updated」，
  155 stars，GPL-2.0，最後 push 2025-01-17，是四個裡最活躍的）。
- nemerle/dcc 的 Readme（`qt5` 分支）自述：只支援小型 80286 DOS 程式反編到
  C（不支援 C++），**維護者原文承認「版本仍在許多情況下有缺陷」**，已知
  問題包含靜態 200 字元緩衝區溢位、記憶體重新配置導致指標失效、陣列/指標
  支援不完整，並且**明說「目前沒有時間在此專案上工作」**。沒有提供當代
  編譯說明。來源：`gh api repos/nemerle/dcc/readme`。
- 結論：歷史地位重要（本專案很多手法呼應 dcc 論文的方法論），但作為
  **可用工具**基本上是研究原型等級，不建議依賴它處理一個 173 KB、
  有跳表/間接呼叫的真實遊戲執行檔。

#### RetDec（Avast）

- 官方 README 明文列出支援架構：「32-bit: Intel x86, ARM, MIPS, PIC32, and
  PowerPC；64-bit: x86-64, ARM64」——**完全沒有 16-bit**。這不是查不到，
  是查到了明確排除。授權 MIT，`pushed_at: 2026-05-26`（有基本維護但功能面
  幾乎不動）。**直接排除，不用進一步測試。**

#### radare2 / rizin

- `asm.bits=16` 可以讓反組譯器用 16-bit 模式解碼指令，DOS 相關的實戰紀錄
  （如 `zxvf.org` 的 `r2com101` 文章）證實可以拿來看 COM 檔案，但那是
  **反組譯**（disassembly）層級。
- decompiler 層：內建 `pdc`（pseudo-decompiler）官方文件明講「不實作
  switch、if/else、for/while 的控制流改善」——等於只是排版過的組語，
  不是我們要的東西。`r2dec`／`rz-ghidra` 這類第三方 decompiler 外掛沒有
  查到任何關於 16-bit real mode segmented addressing 的專門支援說明；
  `rz-ghidra` 本質是把 Ghidra 的 P-code decompiler 包一層介面，**跟直接用
  Ghidra 撞到的限制是同一組**，沒有額外價值。

#### Snowman

- Repo 已 **archived**（`gh api` 確認 `archived: true`，最後 push
  2023-03-09）。GitHub issue #131「Allow decompilation of 16-Bit files」
  討論串明確指出 16-bit 支援從未實作，且架構上假設 flat memory model，
  要支援 real mode 需要重寫 disassembler 與 calling convention 的底層假設，
  不是加參數就能解決。**直接排除。**

#### dis86（xorvoid）—— 查到的最貼近本專案動機的工具，但幫不上忙

這是本次調查裡唯一一個「開發動機描述」跟本專案幾乎逐字重複的工具——作者
部落格明講是為了「逆向工程與重新實作一款 1990 年代早期的 DOS 遊戲」而寫，
用 Rust 開發，管線是「機器碼 → 反組譯 → 初始 IR → 最佳化 IR → AST → C」，
支援輸出反組譯／IR／控制流圖（graphviz）／C 程式碼，能跑在 headless CLI 上。

但有兩個實質性問題讓它現階段對本專案幫助有限：

1. **不處理 MZ 格式與 relocation**——README 原文：「The decompiler accepts
   only a flat binary region for the text segment. It doesn't handle common
   binary file-formats (e.g. MZ) at the moment.」也就是說使用者要自己先把
   目標函式所在的 segment 從 `DEMON.INT` 切出來、算好 relocation，跟我們
   現在手動用 Python/objdump 做的事情性質相同，dis86 只接手「切好的一段
   flat binary 之後」的分析，並不會替我們省掉最麻煩的那一步。
2. **沒有 LICENSE 檔**——`gh api repos/xorvoid/dis86` 回傳 `license: null`，
   repo 根目錄查無 `LICENSE` 檔案。在沒有明確授權聲明下直接拿來改或整合，
   法律上站不住腳，作者本人也在 README 講明「不保證對自己專案以外的用途
   能直接運作」，等於是私有內部工具開放原始碼閱覽，不是以「給別人用」
   為目標維護的專案。

**查不到的部分**：dis86 是否曾被拿來實際跑過其他遊戲（除作者自己的專案）、
它處理的目標遊戲確切是哪一款——部落格文章刻意沒有點名，多個搜尋來源
（含維基百科的 *Betrayal at Krondor* 條目）看起來相關但無法直接證實，
如實標註為不確定，沒有把它寫進表格當作結論。

---

## 第二部分：自製工具可行性評估

### 核心認知

**目標不是通用反編譯器，是「跟隨已知跳表的程式碼發現器 + CFG 建構器」。**
第一部分的查證結果剛好印證這個方向的必要性：連 Reko（有 20 年歷史的通用
反編譯框架、GPL、活躍維護、對 x86-real-16 有原生支援）都在跟 Ghidra
**同一個位址**（`FUN_222f_0b0e` / Reko 的 `122f:0b0e`）上一樣找不到函式邊界
——因為那段程式碼只能透過跳表的位元組內容才找得到，兩個工具的啟發式
（recursive descent + 間接跳轉回溯）都不足以獨立推出。我們手上剛好有這個
「外部知識」（已解出兩張跳表），這是自製工具唯一有把握贏過現成工具的點。

### 建議技術選型

**Capstone（Python binding，`CS_ARCH_X86` + `CS_MODE_16`）**。

- 已在 docker（`python:3.12-slim`，未動系統 Python）裡實測：`pip install
  capstone` 裝好後，`Cs(CS_ARCH_X86, CS_MODE_16)` 對 `DEMON.INT` 的真實
  位元組（`0x3C09` 附近）直接解出一段合理的 16-bit 指令序列，其中包含
  `lcall 0xd9f, 0x1b1e`——**遠呼叫的 segment:offset 立即數被正確拆出來**，
  這正是目前 `disassembly.asm` 裡 `CALLF` 目標需要另外重定位才能用
  （`docs/re/00` 踩雷 #6）的那個麻煩點，Capstone 原生 API 就能拿到結構化
  欄位，不用再事後 regex 猜。
- 比較基準：`objdump -D -b binary -m i386 -Maddr16,data16`——已經在用，
  能動、但只有文字輸出，程式化解析要重新 parse 文字格式，且不容易拿到
  操作數的結構化欄位（立即數、位移、暫存器分開的欄位）。Capstone 省掉
  這層 parse。
- 兩者都不做程式碼發現（code discovery）——這是我們自己要寫的部分，跟用
  哪個反組譯引擎無關。

### 分階段工作量估計

**第一階段（半天～一天，難度低）：程式碼發現 + 跳表跟隨**

- 輸入：entry point、已知 353+ 函式邊界（`functions.csv`）、兩張跳表
  （`138d:258f` 15 項、`222f:12ce` 19 項，同段內 little-endian uint16 offset）。
- 邏輯：從每個已知函式入口做 recursive descent（線性掃 + 遇到
  `jmp`/`call`/條件跳轉分岔），遇到「跳表模式」（`jmp [bx+table]` 這類）時，
  如果該跳表位址在已知清單裡，直接展開已解出的 target 清單當作額外的
  分支入口；如果不在清單裡，**明確標記成「未知跳表」輸出成清單**（不要
  猜、不要用啟發式硬解，這是本專案已經吃過虧的教訓——見 `docs/re/00`
  第 5 節三次錯誤斷言）。
- 產出：一份「已發現位元組範圍」的區間清單，跟 `functions.csv` 做差集，
  就是目前 Ghidra 沒覆蓋到的間隙清單——這本身就是一個有用的中間產物，
  可以直接拿去指名要後續哪些位址需要人工核對。
- 這階段最省力，因為公式（segment*16+offset-0xC400）、跳表格式、函式
  邊界都已經是專案裡現成的已知事實，不用重新推導。

**第二階段（一到兩天，中等難度）：基本區塊切分 + CFG**

- 用 Capstone 逐指令解碼，遇到分支/呼叫/返回類指令就切基本區塊邊界。
- 16-bit x86 的分支指令集合有限（`Jcc`/`JMP`/`CALL`/`CALLF`/`RET`/`RETF`/
  `LOOP*`），規則明確，不需要處理 x86-64 那種複雜的間接定址爆炸。
- 遠呼叫/跳的目標 segment 就是已知的固定值（同一支程式的靜態程式碼，沒有
  動態載入的模組），可以直接建邊；真正麻煩的只有間接跳轉/呼叫（`jmp
  [bx+...]` 這類），這正是第一階段要顯式標成「未知」而不是硬猜的部分。
- 輸出：文字版 CFG（每個基本區塊列出起訖位址、後繼列表）加 Mermaid
  （`graph TD` 語法直接可貼進 markdown），不需要額外畫圖工具。

**第三階段（可選，一天以上，模式識別）：**

- 迴圈（`回頭邊` back-edge 偵測）、if/else（分岔後匯流點偵測）、switch
  （已知跳表直接映射成 case 清單，不用像 Reko 那樣做通用的 backward
  slicing 去猜）都是圖論上定義清楚的問題，實作量可控。
- 這階段的價值是「幫忙讀」，不是「取代讀」——目標是把 raw CFG 轉成人類
  掃一眼能認出「這是個 for 迴圈」的排版，不追求生成可編譯的 C。

**不值得自製的部分：**

- **完整 C 反編譯（型別推斷、資料流最佳化、SSA）**。這正是 Reko 花了
  十幾年、GPL 開源、還在活躍維護，**在我們自己的檔案上實測仍然會在複雜
  函式上拋例外中止**的部分。自製版本不可能比它更完備，而且我們的真實
  需求（讀懂一支 173 KB 遊戲的邏輯）本來就不需要重新編譯回去，讀組語
  + 註解 + CFG 已經是這幾輪真正解掉難題的方法（任務背景已指出：三次
  錯誤斷言全部是靠繞過 decompiler、直接讀原始指令抓出來的）。
- **通用檔案格式/架構支援**。我們只有一支 binary，MZ header 解析、
  relocation table、base segment 換算公式全部已經在 `docs/re/00` 解完並
  驗證過，不需要做成可重用的通用 loader。
- **啟發式間接跳轉推斷**（猜測未知跳表的目標）。這是 Reko 的
  `BackwardSlicer` 想做但仍不完備的問題，而且本專案的教訓剛好是「猜的
  控制流比沒有更危險」——不猜、只標記待人工核對，是更安全也更省工的
  選擇。

---

## 第三部分：綜合建議

**現成工具 vs 自製：兩者不是互斥選項，建議搭配用，不是二選一。**

1. **Reko 值得留著當第二意見/交叉驗證工具**，不建議當主力：
   - 它已經證實跟 Ghidra 獨立算出高度重合的函式邊界（10/11 大型 segment
     對上），可以在懷疑 Ghidra 某個函式邊界是不是啟發式錯誤時，拿 Reko
     的結果對照一次，兩者一致就多一分信心，不一致就是該深入核對的訊號。
   - 它的失敗模式（顯式拋例外）比 Ghidra 的 decompiler（悄悄編造控制流）
     安全，複雜函式失敗時至少不會產生「看起來合理但是錯的」C 碼。
   - 但它一樣解不開需要跳表知識才能找到的函式（`FUN_222f_0b0e` 案例已
     實測驗證），所以**不能取代**「跟隨已知跳表」這件事，也不能取代
     `docs/re/00` 已經建立、逐位元組驗證過的 base segment 換算流程。
   - 授權 GPL-2.0，只是拿來跑分析、不重新散布/修改整合進本專案程式碼，
     沒有授權疑慮。

2. **dcc、RetDec、Snowman、radare2/rizin 的 decompiler：不建議投入時間**。
   RetDec 和 Snowman 架構上排除 16-bit real mode，dcc 是研究原型等級且
   已停滯維護，radare2/rizin 的 decompiler 對這個問題域（switch/跳表
   還原）明確不做。

3. **dis86 值得每隔一段時間關注**（活躍開發中，跟本專案目標高度同構），
   但現階段：(a) 不處理 MZ/relocation，等於幫不上最麻煩的那步；
   (b) 沒有授權聲明，不建議依賴。之後如果它加了 MZ 支援、補上授權檔，
   值得重新評估。

4. **建議自製最小可用版本（MVP）做到第一階段 + 第二階段**：
   - 第一階段輸出「已知跳表跟隨後的程式碼發現結果」+「跟 Ghidra
     `functions.csv` 的差集清單」——這個清單直接可以拿去比對 Reko 的結果
     做三方交叉驗證（Ghidra / Reko / 自製工具，三個獨立來源一致的部分
     可信度最高）。
   - 第二階段輸出文字版 CFG + Mermaid，把已知跳表的 case 分支直接映射
     成邊，不用等第三階段的模式識別。
   - 第三階段（迴圈/if-else/switch 排版）視實際需求再做，不是 MVP 的
     必要條件——目前這幾輪解題本來就是直接讀組語，排版只是加分。
   - 技術選型：Capstone（`CS_MODE_16`）取代目前手動 parse `objdump`
     文字輸出/`disassembly.asm`，其餘沿用專案已有的位址換算公式與已知
     函式/跳表清單，不用重新發明。

**查不到 / 不確定的部分，如實列出：**

- dis86 作者部落格刻意沒點名開發動機對應的具體遊戲，無法證實是否為
  *Betrayal at Krondor*，只列為可能關聯、不當結論用。
- radare2/rizin 對 MZ relocation table 的處理細節（不只是格式辨識，
  是否真的做 segment fixup）沒有查到原始碼等級的證據，表格中列為
  「未查證到位」而非「否」，避免猜測。
- biappi/DCC fork 是否真的能在現代系統編譯執行，只有 repo 描述宣稱，
  没有找到建置紀錄或 CI 結果佐證，因此在正文中沒有把它當作可信選項，
  只在 dcc 段落裡提及存在。
- Reko 的跳表還原（`BackwardSlicer`）在什麼條件下會成功、什麼條件下會
  拋例外，沒有讀完整套原始碼去精確定位邊界條件，只有本次實測結果
  （部分函式成功、部分函式拋例外）當經驗證據，不代表窮舉過所有情況。

---

## 附錄：Reko 實測完整紀錄（可重跑）

```bash
# 全程 docker，未在系統 Python/環境安裝任何東西
mkdir -p /tmp/reko_test && cd /tmp/reko_test
cp /home/anr2/cht/daemon_winter/workplace/orig/demwin/DEMON.INT .

# 下載 Reko 0.12.3 官方 release（GPL-2.0，2026-06-16 發布）
curl -sL -o cmdline.zip \
  "https://github.com/uxmal/reko/releases/download/version-0.12.3/CmdLine-0.12.3-x64-f6a16a2abd.zip"
unzip -q cmdline.zip -d cmdline

# 在 .NET 8 官方 docker image 裡跑，不動系統環境
docker run --rm -v "$(pwd)":/work -w /work mcr.microsoft.com/dotnet/runtime:8.0 \
  dotnet /work/cmdline/decompile.dll --arch x86-real-16 --env ms-dos /work/DEMON.INT

# 結果落在 ./DEMON.reko/ 底下：DEMON_code_000{0,1,2}.{asm,dis,c}、DEMON.h、
# analysis_99_crash.txt（SSA 無窮遞迴那次的完整 stack trace）

# 函式總數
grep -oE '^fn[0-9A-Fa-f_]+' DEMON.reko/DEMON_code_000*.asm | sort -u | wc -l
# => 373

# segment 集合比對（Ghidra 的 11 個大 segment，扣掉 Ghidra 的 +0x1000 base）
# Ghidra: 1000 138d 1990 217b 222f 25be 278d 2cdc 310e 3196 1d9f
# 扣 0x1000 後應為: 0000 038d 0990 117b 122f 15be 178d 1cdc 210e 2196 0d9f
grep -ohE 'fn[0-9A-Fa-f]{4}_' DEMON.reko/DEMON_code_000*.asm \
  | sed 's/fn//;s/_//' | sort -u
# => 10/11 精確對上（僅 2196 沒有對應項，Reko 該區間最近的是 21E4）

# 跳表消失位置交叉驗證：Ghidra 的 FUN_222f_0b0e 對應 Reko 原始 segment 122f
grep -oE 'fn122F_[0-9A-Fa-f]+' DEMON.reko/DEMON_code_000*.asm | sort -u
# => 0832 1321 1404 1448 1809 1813 1816 1849 184B 1851 1859 18AA 18AD
#    2CF2 66E4 B43E （没有 0B0E，跟 Ghidra 一样是空的）

# 錯誤/例外次數（同一個問題域：複雜函式的 structure analysis 會失敗）
grep -c "^error:\|^fn.*error:" DEMON.reko/*.log 2>/dev/null || true
```

capstone 16-bit 模式驗證（技術選型佐證，非完整程式碼發現流程）：

```bash
docker run --rm -v "$(pwd)":/work -w /work python:3.12-slim bash -c "
pip install -q capstone
python3 -c \"
import capstone
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_16)
with open('DEMON.INT','rb') as f: data = f.read()
code = data[0x3C09:0x3C09+64]
for insn in md.disasm(code, 0x0009):
    print('0x%x:\t%s\t%s' % (insn.address, insn.mnemonic, insn.op_str))
\"
"
# 輸出可正確解出 dec/cmp/jne/mov/push/lcall 等指令，
# 其中 "lcall 0xd9f, 0x1b1e" 證實遠呼叫的 segment:offset
# 立即數可直接從結構化欄位取得，不需要事後 regex 重定位。
```
