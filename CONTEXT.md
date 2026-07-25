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
   目前**刻意還沒開始寫引擎**，先把機制全部搞清楚。

### 不做的事

不散布原版執行檔、資料檔、美術或音樂。公開產出只有引擎程式碼與翻譯文本，
玩家自備合法原版。原版資料一律 gitignore。

---

## 2. 現況一覽

### 已完成

| 領域 | 狀態 |
|---|---|
| 官方手冊繁中版 | 全 28 頁 + 附錄，`docs/manual/` |
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
| 可通行性對照表 `[0x5500]` | **擋住移動實作** — 機制已驗證，表的來源檔未追出、內容未 dump |
| 事件類別 0 的下游消費者 | `EXITS.DAT` 裡佔 94/110。索引**有**被算出（`docs/re/05` 舊述不準），但這條路徑上沒被消費，真正的消費者未定位 |
| 一天幾小時（26 vs 38）| DOSBox 實測未完成（隊伍被困小中庭）|
| `OPEN.PIE` | 未解（102,160 B，不符 3.5× 規律）|
| Go 引擎本體 | **尚未開始**（依 SDD，多數規格已 READY，可以開工了）|

### 這一輪（2026-07-25）解掉的長期未解項

召喚／幻術、即死／束縛／枯萎、怪物進場擲點、兩個戰鬥邊界條件、爆擊門檻條件、
城鎮六大設施全部公式、市集議價與說服技能、治療所費率來源、
**種族欄位位置**、遊戲內部技能 id 表、`FILES.DAT` 表布局、道具槽陣列起點。

## 3. 文件索引

### 規格層 `docs/spec/` — 實作的唯一依據

**只有標 READY 的才可以實作。**

| 檔案 | 狀態 |
|---|---|
| [`README.md`](docs/spec/README.md) | SDD 工作方式、規格分級定義 |
| [`01-rng.md`](docs/spec/01-rng.md) | **READY** — 亂數產生器 |
| [`02-combat.md`](docs/spec/02-combat.md) | **READY** — 戰鬥系統（全部子系統）|
| [`03-events.md`](docs/spec/03-events.md) | **DRAFT** — 事件觸發（類別 0 的下游消費者未定位）|
| [`04-movement.md`](docs/spec/04-movement.md) | **READY** — 移動與模式切換（可通行性表待 dump）|
| [`05-character.md`](docs/spec/05-character.md) | **READY** — 角色建立與升級 |
| [`08-town-economy.md`](docs/spec/08-town-economy.md) | **READY** — 城鎮與經濟 |
| 06 時間、07 素材渲染、09 字型 | 待寫 |

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
| [`17-font-format.md`](docs/re/17-font-format.md) | **字型格式**（CGA 8×8 bit-plane、EGA 16×14）與繪字函式鏈 |
| [`18-jumptable-sweep.md`](docs/re/18-jumptable-sweep.md) | 跳表全檔清掃（21 張）+ **即死／束縛／枯萎的精確判定公式** |
| [`19-town-economy.md`](docs/re/19-town-economy.md) | **城鎮經濟六大設施全部公式**；市集議價與說服技能；角色記錄 6 個新欄位 |
| [`20-summon-and-combat-units.md`](docs/re/20-summon-and-combat-units.md) | **召喚／幻術**、戰鬥單位 19 欄結構、怪物進場擲點、傷害與排序邊界 |
| [`21-skills-races-and-files-dat.md`](docs/re/21-skills-races-and-files-dat.md) | **遊戲內部技能 id 表**、種族系統與修正公式、`FILES.DAT` 完整布局 |

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
| `docs/manual/part-0..4.md` | 官方手冊繁中版 |
| `docs/walkthrough/part-1..6.md` | 社群攻略繁中版 |

### 工具與 skill

| 路徑 | 內容 |
|---|---|
| `.claude/skills/ghidra-dos16-re/` | **16 位元 DOS Ghidra 反組譯 skill** |
| `tools/ghidra_headless.sh`、`tools/ghidra_scripts/` | Ghidra 封裝與 post-script |
| `tools/dosbox_run.sh` | DOSBox 封裝 |
| `tools/parse_*.py` | 各資料格式的 Python 解析器 |
| `internal/` | Go 引擎（目前只有解碼器與 RNG） |

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
| **frame（sprite）** | `.SHP`/`.SHE` 的單張圖。CGA 16×32、EGA 32×28 |
| **glyph** | 字型單字。CGA `.FNT` 8×8 2bpp（16 B）、EGA `.FNE` 16×14 1bpp（28 B），皆 96 字（0x20–0x7F） |
| **plane** | EGA 的位元平面。`.SHE` 是「列內 4 個 plane 各佔連續 4 bytes」 |

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

### 三個反覆咬人的陷阱

**① decompiler 會靜默捏造控制流。**
Ghidra 對某些函式只展開前段，跳表後的 case body 落在間隙裡，
decompiler 不報錯反而編造。**已造成三次錯誤斷言，每次都被標成「已驗證」**。
判別法：反編譯行數遠大於宣告 byte 數、或大量 unreachable 警告 → 不可信。
含跳表的 switch 一律回原始指令讀。

**② 算術對不代表方向對。**
檔案大小的算術關係只約束「總位元數」，不約束佈局方向。本專案踩過四次：
CGA sprite 16×32 vs 32×16、`OPEN.PIC` linear vs 交錯、`SUM.MAP` column-major vs
row-major、`.SHE` 尺寸 32×28 vs 16×56。**四次裡三次的錯誤版本都能通過全部測試。**
→ 視覺產物一律 dump 出來肉眼比對原版。

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
| EGA 素材都是 320×350 four-plane | 只有部分成立，`.SHE` 是 32×28 |
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

規格層已大致齊備（6 份裡 5 份 READY），**引擎本體可以開工了**。
若要先把剩下的缺口補完，優先序是：

1. **dump 可通行性對照表 `[0x5500]`** —— 唯一擋住移動實作的東西。
   機制已驗證，缺的是表的來源檔與內容
2. **事件類別 0 的下游消費者** —— 唯一擋住事件規格升 READY 的東西。
   已知索引有被算出，缺的是誰消費它。建議 DOSBox 走進純文字房間單步觀察 `0x52f4`
3. **一天幾小時（26 vs 38）** —— 擋住時間系統規格（06）
4. **補寫規格 06 時間、07 素材渲染、09 字型**（證據等級已夠，尚未收攏）
5. `OPEN.PIE`、`FILES.DAT` 三段未解區、`ITEMS.DAT` 的 f1–f6

### 可以直接開工的部分

`01-rng`、`02-combat`、`05-character`、`08-town-economy` 四份 READY，
涵蓋亂數、完整戰鬥、建角升級、全部城鎮設施。這些不依賴上面任何一項缺口。
