# CONTEXT — 專案脈絡與文件索引

> **這份是全專案的單一入口。** 對話被壓縮、或換一個新 session 接手時，先讀這份，
> 就能重建完整全局，再依索引跳到需要的文件。
>
> 最後更新：2026-07-25

---

## 1. 這個專案在做什麼

把 SSI《Demon's Winter》(1988/1989, DOS) 完整逆向、在 Go / Ebiten 上重寫引擎、
再做繁體中文化。定位是**文化資產保存**，替華人遊戲圈留下這款經典。

### 兩條硬性原則

1. **完整性 > 投報**。不得以「成本高、效益低」為由跳過任何素材版本、任何格式。
   卡關就換方法，記錄「卡在哪、試過什麼」，不寫「暫緩／低投報」當結論。
2. **SDD：spec 齊了才實作**。反組譯 → 收攏成規格（`docs/spec/`）→ 才動手寫程式。
   九份規格全部 READY 後才開工，目前引擎進度見 §2。

### 不做的事

不散布原版執行檔、資料檔、美術或音樂。公開產出只有引擎程式碼與翻譯文本，
玩家自備合法原版。原版資料一律 gitignore。

---

## 2. 現況一覽

### 已完成

| 領域 | 狀態 |
|---|---|
| 官方手冊繁中版 | 全 28 頁 + 附錄，`docs/manual/`（附錄 A 已於 2026-07-25 訂正三處抄錄錯誤）|
| **軟體世界中文說明書** | 1990 年台灣代理版全 51 頁，`docs/manual-cht/` |
| 社群攻略繁中版 | 全 1,914 行，`docs/walkthrough/` |
| 統一譯名表 | 426 條，`translations/glossary.md` |
| 資料格式 | 事件表／地圖／怪物／道具／存檔／CGA 與 EGA 素材，全解 |
| 反組譯 | 主迴圈、事件觸發、戰鬥、音效、移動、建角升級、**字型渲染** |
| Go 解碼器 | `internal/assets/{gamedata,world,scenario,gfx}` + `internal/rng` |
| Ghidra 環境 | docker 化，含跳表 override 修復 |
| DOSBox 環境 | docker 化，可自動送鍵與截圖 |

### 進行中／受阻

| 項目 | 狀態 |
|---|---|
| `OPEN.PIE` | 未解（102,160 B，不符 3.5× 規律）。新線索：EGA 素材一律「檔案存半寬」，先前湊候選尺寸時用的是顯示寬度，要照半寬重算 |
| Go 引擎本體 | **M1–M4 完成**（見下）。城鎮、建角 UI、存檔寫回、音效、原版版面尚未實作 |

### 引擎進度

| 里程碑 | 狀態 |
|---|---|
| M1 可走動的世界 | ✅ 載入地圖與地形圖塊、可通行性判定、子地圖進出、三層時間與步數推進。端到端驗證：從 (34,36) 連走 11 步 → X=45、Hour 5→6、Steps=1 |
| M2 事件與文字 | ✅ 觸發閘門（tile 0x11/0x53 查表、五個 tile 直接當索引、0x35 硬阻擋）、`EXITS.DAT` 掃描與事件索引累計、`DATA*.TXT` 文字顯示（40 欄逐詞斷行、每 5 行翻頁）。端到端：走進 (54,16) 跳出「The Hall of Bones…」並可翻頁。**戰鬥與傳送尚未接** |
| M3 角色 | ✅ 規則層：存檔載入、剩餘智慧點數、建角擲點（種族修正＋下限鉗制＋人類 +2）、升級（HP／SP 成長、三點屬性分配、封頂）。viewer 按 `P` 看隊伍名冊。**建角 UI 與三次重擲未做**（重擲機制尚未在程式碼中定位）|
| M4 戰鬥 | ✅ 規則＋流程：武器骰表、戰鬥風格、行動順序、普通攻擊（命中／爆擊／傷害／毒／功夫暈眩）、四個動作、法術系統（參數表 43 筆／通式／即死／束縛／枯萎）、回合狀態機、勝負判定、召喚三槽、怪物進場擲點。端到端：踏到事件格 → 讀敘述 → 6 隻怪物開打 → 跨回合結算。**玩家指令選單與 AOE 未做，目前雙方都由簡單 AI 代打** |
| M5 城鎮與經濟 | 未開始（`08-town-economy` 已 READY）|
| 中文化 | 未開始。前置：16×16 CJK 點陣字型、UI 版面重排（畫布拉到 640×400）、字串抽取與翻譯表 |

程式碼結構：`internal/assets`（純解碼，回傳 `image.RGBA`，不認識 Ebiten）→
`internal/game`（規則層，不認識畫面）→ `internal/ui`（Ebiten 呈現層）→ `cmd/demonwinter`。

建置與驗證一律走 docker：`tools/go.sh <go 子指令>`、
`tools/screenshot.sh <out.png> [KEYS=Up,Right,…] [程式參數]`（headless Xvfb + xdotool 送鍵）。

### 這一輪（2026-07-25）解掉的長期未解項

召喚／幻術、即死／束縛／枯萎、怪物進場擲點、兩個戰鬥邊界條件、爆擊門檻條件、
城鎮六大設施全部公式、市集議價與說服技能、治療所費率來源、
**種族欄位位置**、遊戲內部技能 id 表、`FILES.DAT` 表布局、道具槽陣列起點。

同日整理軟體世界 1990 年中文說明書（`docs/manual-cht/`）後另外確認：

- 拿它的附錄 A 對 `FILES.DAT 0x158`，**抓出 `docs/manual/part-4.md` 三處抄錄錯誤**（已訂正）
- 山丘「通過時間兩倍」＝ `04-movement` 的「某些 tile 步數 +2」，機制對上、待補 tile 清單
- 建角**三次重擲機制**（低於 6 可重擲，共三次）——手冊有、程式碼未定位，記入 `05-character` 未解
- **MGA 單色版執行檔 `PLAY` 不在手上的 dump 裡**，該版本無從逆向
- ⚠ 「一天 26 小時」是英文手冊的中譯，**不構成第二個佐證**，26 vs wrap 38 的落差依舊未解

同日稍晚**推翻並訂正 sprite frame 尺寸**（見 §5 已被推翻的斷言）：
CGA `.SHP` 是 16×16（64 B）、EGA `.SHE` 是 16×28（224 B），
frame 數 COMBAT 44／CYPHER 27／DEMON 102／MONSTER 240／SHIP 32／WINTER 102。
連帶確認：EGA 素材一律「檔案存半寬、顯示時寬 ×2」（`.SHE` 載入時加倍、
`.PIE` blit 時加倍，`.PIE` 顯示尺寸是 288×252）；
`DEMON`/`WINTER` 的 102 格是地形圖塊集，`WINTER` 是雪地版。

追查 sprite 尺寸時，發現 **CGA 字型 `ASC.FNT` 一直解不出可讀字形**（atlas 全是雜訊，
但文件標著「已驗證」——那句其實驗的是 EGA）。一併解掉：
`.FNT` 是 **packed 2bpp**（byte0 = 左 4px、byte1 = 右 4px）不是 planar、
**無檔頭**（3,073 B = 3,072 資料 + 結尾 `0x1a` DOS EOF，原解碼器誤跳開頭一個 byte、
整份位移一格）、**96 字 × 2 bank**（bank0 一般、bank1 反白，bank 間距 1,536）。
`.FNE`（EGA 16×14）不受影響，原結論成立。

也認出**山丘 = tile `0x0e`／`0x2b`**（移動多算一步的那兩個值）：
圖塊畫出來是起伏等高線，與手冊「山丘通過時間為一般地形兩倍」對得上。

## 3. 文件索引

### 規格層 `docs/spec/` — 實作的唯一依據

**只有標 READY 的才可以實作。**

| 檔案 | 狀態 |
|---|---|
| [`README.md`](docs/spec/README.md) | SDD 工作方式、規格分級定義 |
| [`01-rng.md`](docs/spec/01-rng.md) | **READY** — 亂數產生器 |
| [`02-combat.md`](docs/spec/02-combat.md) | **READY** — 戰鬥系統（全部子系統）|
| [`03-events.md`](docs/spec/03-events.md) | **READY** — 事件觸發 |
| [`04-movement.md`](docs/spec/04-movement.md) | **READY** — 移動與模式切換 |
| [`05-character.md`](docs/spec/05-character.md) | **READY** — 角色建立與升級 |
| [`06-time.md`](docs/spec/06-time.md) | **READY** — 時間系統 |
| [`07-graphics.md`](docs/spec/07-graphics.md) | **READY** — 素材格式與渲染 |
| [`08-town-economy.md`](docs/spec/08-town-economy.md) | **READY** — 城鎮與經濟 |
| [`09-fonts.md`](docs/spec/09-fonts.md) | **READY** — 字型與中文化 |

### 逆向筆記 `docs/re/` — 過程與證據

| 檔案 | 內容 |
|---|---|
| [`00-ghidra-setup.md`](docs/re/00-ghidra-setup.md) | **必讀** — Ghidra 環境、位址換算、六條踩雷 |
| [`01-dosbox-reference.md`](docs/re/01-dosbox-reference.md) | DOSBox 環境、timeline 語法、自動化踩雷 |
| [`02-data-loading-functions.md`](docs/re/02-data-loading-functions.md) | 事件表與城鎮的讀取函式 |
| [`03-audio-and-resources.md`](docs/re/03-audio-and-resources.md) | 音效（V1 結案）、SUM.MAP RLE |
| [`04-main-loop-state-machine.md`](docs/re/04-main-loop-state-machine.md) | 主迴圈、指令分派、時間、勝利條件 |
| [`05-event-triggering.md`](docs/re/05-event-triggering.md) | 座標 → EXITS → 事件索引 |
| [`06-combat-system.md`](docs/re/06-combat-system.md) | 戰鬥框架、命中傷害（⚠ 選單 case 表已作廢，見 09） |
| [`07-sprite-blit.md`](docs/re/07-sprite-blit.md) | EGA sprite blit、`.SHE` 格式 |
| [`08-movement-and-modes.md`](docs/re/08-movement-and-modes.md) | 移動、面向、可通行判定、模式切換 |
| [`09-spells-and-actions.md`](docs/re/09-spells-and-actions.md) | 8 個戰鬥動作的正確 case 編號與公式 |
| [`10-character-and-economy.md`](docs/re/10-character-and-economy.md) | 建角、升級公式 |
| [`11-dosbox-verification.md`](docs/re/11-dosbox-verification.md) | DOSBox 三項實測 |
| [`12-ghidra-jumptable-fix.md`](docs/re/12-ghidra-jumptable-fix.md) | 跳表 override 修復 |
| [`13-realmode-tooling-survey.md`](docs/re/13-realmode-tooling-survey.md) | real mode 工具鏈調查 |
| [`14-rng-float-equivalence.md`](docs/re/14-rng-float-equivalence.md) | RNG 浮點等價性證明 |
| [`15-spell-constants.md`](docs/re/15-spell-constants.md) | **法術 K/M 參數表**（在 `FILES.DAT`，不在 `DEMON.INT`） |
| [`16-combat-details.md`](docs/re/16-combat-details.md) | 命中修正、爆擊、戰敗判定、AOE |
| [`17-font-format.md`](docs/re/17-font-format.md) | **字型格式**（CGA 8×8 packed 2bpp 兩 bank、EGA 16×14 1bpp）與繪字函式鏈 |
| [`18-jumptable-sweep.md`](docs/re/18-jumptable-sweep.md) | 跳表全檔清掃（21 張）+ **即死／束縛／枯萎的精確判定公式** |
| [`19-town-economy.md`](docs/re/19-town-economy.md) | **城鎮經濟六大設施全部公式**；市集議價與說服技能；角色記錄 6 個新欄位 |
| [`20-summon-and-combat-units.md`](docs/re/20-summon-and-combat-units.md) | **召喚／幻術**、戰鬥單位 19 欄結構、怪物進場擲點、傷害與排序邊界 |
| [`21-skills-races-and-files-dat.md`](docs/re/21-skills-races-and-files-dat.md) | **遊戲內部技能 id 表**、種族系統與修正公式、`FILES.DAT` 完整布局 |
| [`22-resource-arena-and-passability.md`](docs/re/22-resource-arena-and-passability.md) | **18 段資源記憶體區**、**可通行性表**、子地圖退出、**事件消費者** |

### 資料格式 `docs/formats/`

| 檔案 | 內容 |
|---|---|
| [`event-script.md`](docs/formats/event-script.md) | `DATA*.TXT` 事件表 |
| [`game-data-tables.md`](docs/formats/game-data-tables.md) | `MONSTER.DAT`、`ITEMS.DAT`、`PARTY.DAT` |
| [`town-and-map.md`](docs/formats/town-and-map.md) | 城鎮、地圖、`EXITS.DAT`、`SUM.MAP` |
| [`graphics.md`](docs/formats/graphics.md) | CGA / EGA 素材格式 |
| [`resource-index.md`](docs/formats/resource-index.md) | `FILES.DAT` / `FILES.DTT` |

### 研究與計畫

| 檔案 | 內容 |
|---|---|
| [`PLAN.md`](PLAN.md) | 專案計畫、待驗證項、階段分解、風險表 |
| [`README.md`](README.md) | 對外門面、文件索引、授權聲明 |
| [`docs/research/ssi-engine-architecture.md`](docs/research/ssi-engine-architecture.md) | **SSI 引擎架構研究** — 對 Gold Box 系列可能有用 |

### 翻譯

| 檔案 | 內容 |
|---|---|
| [`translations/glossary.md`](translations/glossary.md) | **統一譯名表 426 條** — 所有翻譯的唯一依據 |
| `docs/manual/part-0..4.md` | 官方手冊（SSI 英文版）繁中翻譯 |
| [`docs/manual-cht/`](docs/manual-cht/) | **軟體世界 1990 年中文說明書**全 51 頁轉錄＋考據。`README` 出處與驗證價值、`01` 正文、`02` 附錄 A–E 全表、`03` 當年譯名與誤譯清單 |
| `docs/walkthrough/part-1..6.md` | 社群攻略繁中版 |

### 工具與 skill

| 路徑 | 內容 |
|---|---|
| `.claude/skills/ghidra-dos16-re/` | **16 位元 DOS Ghidra 反組譯 skill** |
| `tools/ghidra_headless.sh`、`tools/ghidra_scripts/` | Ghidra 封裝與 post-script |
| `tools/dosbox_run.sh` | DOSBox 封裝 |
| `tools/parse_*.py` | 各資料格式的 Python 解析器 |
| `tools/go.sh`、`tools/screenshot.sh` | docker 內編譯／測試；headless 跑起來送鍵截圖 |
| `internal/`、`cmd/` | Go 引擎（assets 解碼、game 規則層、ui 呈現層、demonwinter 執行檔） |

---

## 4. 術語表（Ubiquitous Language）

專案內外一致使用這些詞，寫程式碼、文件、commit 都照這個來。

### 遊戲資料

| 術語 | 意義 |
|---|---|
| **事件表** | `DATA1..5.TXT`，地城的「房間描述 + 遭遇怪物」宣告式表格。**不是 bytecode** |
| **dataset** | 一個地城／區域，切換時 `EXITS.DAT` 會被整份覆寫 |
| **sub-map** | `SUM.MAP` 內的 23 個 RLE 壓縮地圖之一，依 map ID 取用 |
| **slot（戰鬥）** | 戰鬥場上的 15 個單位槽位，0–6 怪物、7–14 玩家 |
| **slot（道具）** | 角色道具欄的 10 格，每格 17 bytes，從記錄內 **`0x0c`** 起 |
| **trailer** | `PARTY.DAT` 尾端 194 bytes 的隊伍共用資料區，從 `0x514` 起 |
| **frame（sprite）** | `.SHP`/`.SHE` 的單張圖。**CGA 16×16（64 B）、EGA 16×28（224 B）**，兩套 frame 數一一對應 |
| **glyph** | 字型單字。CGA `.FNT` 8×8 packed 2bpp（16 B，96 字 ×2 bank：一般／反白）、EGA `.FNE` 16×14 1bpp（28 B，96 字 0x20–0x7F）|
| **plane** | EGA 的位元平面。`.SHE` 是「列內 4 個 plane 各佔連續 2 bytes」（檔案）／「各佔連續 4 bytes」（載入加倍後的記憶體） |
| **載入時加倍** | EGA `.SHE` 讀進記憶體後被 `FUN_217b_0adf` 就地水平加倍（224 B → 448 B/frame）。`.PIE` 則在 blit 時加倍。EGA 素材一律「檔案存半寬、顯示時寬 ×2」 |

### 逆向工程

| 術語 | 意義 |
|---|---|
| **位址換算公式** | `file_offset = (segment − 0x1000) × 16 + offset + 0x3C00` |
| **間隙（gap）** | Ghidra 自動分析沒展開的程式碼區域，decompiler 會在上面編造控制流 |
| **跳表 override** | 用 `JumpTable.writeOverride()` 告訴 decompiler 正確的 switch 目標 |
| **oracle** | 判定正確性的依據來源，有優先序（見 §5） |
| **字串錨定** | 用語意明確的字串反查引用它的函式，找主邏輯的主要手段 |

### 規格分級

| 級別 | 意義 |
|---|---|
| **READY** | 行為、公式、邊界條件、驗收方式都明確，**可以實作** |
| **DRAFT** | 主要行為清楚，但有缺口，**需先補齊** |
| **BLOCKED** | 有已知未解問題擋著，**不可實作** |

---

## 5. 新 session 必須知道的關鍵事實

### 引擎本質

`DEMON.INT` 是**原生 8086 機器碼，不是 bytecode**。`.INT` 只是 SSI 的
「interpreter」命名慣例。這遊戲**沒有虛擬機** —— 控制流全寫死在機器碼裡，
資料檔只提供內容素材。所以重寫時要做的是「事件表直譯器」，不是 VM。

`DEMON.EXE`（6 KB）只是 loader，用 DOS EXEC（`INT 21h, AH=4Bh`）啟動 `DEMON.INT`。

### oracle 優先序（由高到低）

1. **人親自實測原版**
2. **DOSBox 原版實跑**（`tools/dosbox_run.sh`）
3. 官方手冊 / 社群攻略
4. 反組譯推論

**低層級不可推翻高層級。** 反組譯看起來再合理，被原版實跑打臉就是反組譯錯了。

> ⚠ **翻譯本不是獨立來源。** `docs/manual-cht/`（軟體世界中文說明書）是英文手冊的中譯，
> 它寫的數字與英文版相同時，只代表翻譯忠實，**不能當第二個佐證**——
> 「一天 26 小時」就是這樣一個誤判陷阱。
> 它的價值在轉錄層：同一份原始資料的兩份中文抄本同時對程式碼，抄錯的一方會現形。

### 三個反覆咬人的陷阱

**① decompiler 會靜默捏造控制流。**
Ghidra 對某些函式只展開前段，跳表後的 case body 落在間隙裡，
decompiler 不報錯反而編造。**已造成三次錯誤斷言，每次都被標成「已驗證」**。
判別法：反編譯行數遠大於宣告 byte 數、或大量 unreachable 警告 → 不可信。
含跳表的 switch 一律回原始指令讀。

**② 算術對不代表方向對，也不代表切分單位對。**
檔案大小的算術關係只約束「總位元數」。本專案踩過五次：
`OPEN.PIC` linear vs 交錯、`SUM.MAP` column-major vs row-major、
CGA sprite 32×16 vs 16×32、`.SHE` 16×56 vs 32×28，
以及最後一次 —— **sprite frame 尺寸整批錯**（CGA 實際 16×16、EGA 實際 16×28）。
**錯誤版本幾乎都能通過全部測試**，有兩次還畫得出「看起來像那麼回事」的圖。
→ 視覺產物一律 dump 出來肉眼比對原版；一整批佈局假設全數失敗時，
回頭懷疑的應該是**切分單位**，不是佈局。

**②-b sprite 的專屬判準：每個 frame 必須自成一體。**
解出來若是「一格裡有 2 個或 4 個並排／堆疊的小圖」，
那是尺寸解錯把相鄰 frame 疊起來了，不是美術打包慣例。
這個徵狀出現過兩次，兩次都被合理化成「美術資源本來就這樣打包」，
於是錯誤結論頂著「已驗證」標籤活了下來。

**③ 「已驗證」標籤本身要存疑。**
特別是任何以反編譯 C 為唯一依據的結論。

### 已被推翻的斷言（不要重蹈）

| 曾經寫過 | 真相 |
|---|---|
| `EXITS.DAT` 是 165 筆 2-byte 座標對 | 110 筆 3-byte `[X,Y,type]` |
| `FILES.DAT` 是資源索引 | 不是；`FILES.DTT` 才是字串池 |
| `TEMPLAT*.DAT` 是角色範本 | 是特定地點的特殊道具資料 |
| `PARTY.DAT` 的 `0x0c` 是種族 | **種族在 `0xf5`**。`0x0c` 是**道具槽陣列的起點**（一度誤判為「裝備武器槽索引」，那個在 `0x100`）|
| 戰鬥選單 case 0xa 是 Attack | 是 Examine；Attack 是 case 5 |
| EGA 素材都是 320×350 four-plane | 只有部分成立。EGA 素材的規則是「檔案存半寬、顯示時寬 ×2、高 ×1.75」|
| sprite frame 是 CGA 16×32、EGA 32×28（448 B）| **CGA 16×16（64 B）、EGA 16×28（224 B）**。錯在兩點：(1) 只讀 blit 端 `FUN_217b_07cf`，沒發現它讀的緩衝區在載入時已被 `FUN_217b_0adf` 水平加倍過，把記憶體 stride `0x1c0`(448) 當成磁碟格式；(2) 算術分不出來 —— 16×32 與 16×16 的位元組數都整除全部檔案，「3.5× 規律」兩種讀法都成立。真憑據是遊戲初始化時自己宣告的 `[0x5226]`（EGA `0xe0`=224、CGA `0x40`=64）並用它乘出各檔大小。詳見 `docs/formats/graphics.md` §4 |
| `CYPHER.SHE`/`CYPHER.SHP` 是 frame 尺寸的特例 | **不是特例，224/64 才是常態**。它只是 frame 數 27 為奇數，才無法被（錯誤的）448/128 整除，於是被當成特例硬湊過去 |
| `0a95`/`0b19` 位元展開家族與 `.SHE` 路徑無關 | 它們不在 **blit** 路徑上，但在 **載入** 路徑上（`FUN_1d9f_0a8b` → `0adf` → `0b19`），做的是水平像素加倍 |
| 一個 sprite frame 內裝 2（CGA）／4（EGA）個小圖是美術打包慣例 | 是解碼錯誤造成的假象 —— 相鄰 frame 被疊進同一格 |
| RNG 用 80 位元擴充精度 | IEEE double（53 bit） |
| 城鎮經濟的程式碼「落在 Ghidra 未發現區域」 | 只對一半。Ghidra 有線性掃描出指令，只是**沒建函式邊界、decompiler 沒被觸發**。改讀 `disassembly.asm` 的函式間隙（含 segment 開頭第一個命名函式**之前**的區域）就能讀（`docs/re/19` §1） |
| 角色記錄 `+0x102` 是戰鬥狀態位元旗標 | 是**單一列舉值**：`0`正常／`1`中毒／`2`,`3`,`4`束縛三級／`5`死亡 |
| 角色記錄裡「職業（Class）完全未找到」 | 在 `+0xf6`（0–9），兩個獨立呼叫點都拿它當技能學費表的欄索引 |
| EXP 是 3-byte 欄位 | **儲存是 4 bytes**（`0xc4`–`0xc7`），只是數值封頂在 `0x00FFFFFF`。兩種說法各對一半 |
| 傳送法術判的是 `spell_school_id(0x4e2c) == 0x11` | 判的是 **`effect_type`（`0x4e2e`）**。`0x4e2c` 全檔只跟 `1`/`4`/`6`/`8` 比較，值域到不了 `0x11` |
| 枯萎打擊可能是連續呼叫三次 `type 3/4/6` | **不是**。它是獨立的 `effect_type 14`，且套用方式不同 —— 三屬性各自「以現值為上限重擲」，不是扣減 |
| 束縛狀態是布林旗標 | 是**符文系 id**。這是「解除術系別要相符」規則能運作的前提 |

---

## 6. 工作紀律

### 每一輪都要做

1. 更新受影響的 markdown
2. **重新檢視既有斷言，清掉被推翻的**（不是加註解了事，要讓讀者不被舊結論誤導）
3. commit + push
4. 更新這份 CONTEXT.md 的「現況一覽」

### 環境硬規則

- **編譯一律走 docker**（`golang:1.22-bookworm`），不污染系統
- Python 不得在系統 pip install，要用就 docker
- `workplace/orig/` 唯讀，不得修改
- 原版資料、截圖、Ghidra 專案全部 gitignore

### 多 agent 協調

- **agent 還在跑時量到的中間產物不能當結論**（踩過：把迭代暫態誤判為失敗）
- **`git add` 要指名檔案，不要加目錄**（踩過：掃進別的 agent 仍在編輯的檔案）
- 每個 agent 的結論**協調者要獨立複核一條證據鏈**才收

---

## 7. 下一步

**九份規格全部寫完、全部 READY。SDD 的規格階段到此結束，引擎本體可以開工。**

| 規格 | 涵蓋 |
|---|---|
| `01-rng` | 亂數（含浮點等價性證明）|
| `02-combat` | 完整戰鬥：行動順序、命中傷害、爆擊、毒、戰鬥風格、法術、召喚、怪物生成 |
| `03-events` | 座標→事件、傳送、一次性遭遇 |
| `04-movement` | 移動、可通行性、子地圖進出 |
| `05-character` | 建角、種族、技能、升級 |
| `06-time` | 時／日／月、日夜、隨機遭遇 |
| `07-graphics` | CGA/EGA 全部素材格式 |
| `08-town-economy` | 六大城鎮設施 |
| `09-fonts` | 字型 + 中文化設計 |

剩下的未解項**沒有一項擋住開工**：

1. `OPEN.PIE`（唯一未解的素材；開場可先用已驗證的 CGA 版頂著，
   但依完整性原則必須補上）
2. 資源 arena 索引 2/3/6/8/14/15 的內容、`ITEMS.DAT` 的 `f1`–`f6`
3. 各規格文末列的小項（都標明了不擋實作）
