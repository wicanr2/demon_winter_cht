# Demon's Winter 繁體中文化 — 專案計畫

> SSI《Demon's Winter》(1988, DOS)。反組譯原版引擎理解其行為，在 Go / Ebiten 上乾淨重寫可跨平台執行的引擎，
> 並完成介面與劇情文本的繁體中文化。
>
> 版本:v1 (2026-07-24)｜狀態:規劃中，尚未進入實作

---

## 1. 目標

1. **引擎還原**:在 Go + Ebiten 重寫一套可執行的 Demon's Winter 引擎，行為對齊原版。
2. **資料格式全解**:地圖、精靈圖、畫面圖、字型、道具/怪物表、事件腳本、存檔，全部可讀可寫。
3. **音樂 / 音效還原**:原版聲音輸出完整重現(先確認原版實際有什麼，見 §3 待驗證項)。
4. **繁體中文化**:UI、選單、戰鬥訊息、劇情文本、道具/法術/怪物名稱全中文，並附繁中攻略。

### 不做的事

- 不散布原版執行檔、資料檔、美術或音樂。公開產出只有引擎程式碼與翻譯文本，玩家自備合法原版。
- 不做玩法改動(平衡性、新內容)。這是保存專案，不是重製設計。

---

## 2. 已確認的事實(2026-07-24 偵查結果)

### 2.1 執行檔結構

| 檔案 | 大小 | 事實 |
|---|---|---|
| `DEMON.EXE` | 6,036 B | 純 loader。檢查 8087/80287 協同處理器(`8087/80287 is required!`)，載入 `\dem_data\open.pie`、`open.pic`，再啟動 `\demon.int` |
| `DEMON.INT` | 173,380 B | **真正的引擎**。MZ real-mode 執行檔，3,807 個 relocation entry，無 overlay，header 960 段落 → code 起於檔案 offset 0x3C00，入口 CS:IP = 2037:0009 |

**關鍵認知**:`.INT` 是 SSI 的「interpreter」命名慣例，但檔案內容是**原生 8086 機器碼**，不是 bytecode。
本作不像 SCUMM / AGI / SCI 那樣有虛擬機。真正資料驅動的那一層在 `DATA*.TXT` / `TOWN*.DAT` / `EXITS.DAT`
的事件表，那才是本專案要重建成腳本直譯器的對象。

所有 UI 字串都在 `DEMON.INT` 內部(明碼、未壓縮)，例如:

```
Cast what spell:        Turn Undead             !You have used that skill today
God Runes / Fire Runes / Metal Runes / Ice Runes / Spirit Runes / Wind Runes
Hour %d, Day %d in the Month of the %s
CONGRATULATIONS! You have won Demon's Winter.
```

### 2.2 資料檔清單(`DEM_DATA/`，99 檔)

| 類別 | 檔案 | 已知 / 推測 |
|---|---|---|
| 畫面圖 CGA | `*.PIC` ×10 | `OPEN.PIC` = 16,000 B = 320×200 CGA 2bpp 全螢幕(16000 = 320×200÷4 ✅)。其餘 5,184 B 為人像/場景框 |
| 畫面圖 EGA | `*.PIE` ×10 | 5,184 → 18,160 = **3.5× + 16 B**。`OPEN.PIE` 102,160 B(比例不符，可能是多幀或另一種佈局，待驗證) |
| 精靈圖 CGA | `*.SHP` ×6 | 1,728 / 2,048 / 2,816 / 6,528 / 15,360 B |
| 精靈圖 EGA | `*.SHE` ×6 | 6,048 / 7,168 / 9,856 / 22,848 / 53,760 B — **與 .SHP 精確 3.5× 對應** |
| 字型 | `ASC.FNT` 3,073 B｜`ASC.FNE`、`GOT.FNE` 2,688 B | `.FNT` = 1 B header + 3,072 = 256 字 × 12 列(8×12)。`GOT` = Gothic 花體 |
| 地圖 | `MAP1/3/5.MAP` 4,097 B｜`SUM.MAP` 15,743 B | 4,097 = 1 + 64×64 tile。`SUM` 是世界地圖，尺寸不整除 → 可能壓縮 |
| 城鎮 | `TOWN1..25.DAT` 512 B ×25｜`TOWN.TXT` | 25 座城鎮，名稱已解出(Seaside、Elbarat、Akistu、Alynhawk…) |
| 表格 | `MONSTER.DAT` 3,853 B｜`ITEMS.DAT` 724 B｜`ITEMLOCB/X.DAT` 256 B｜`EXITS.DAT` 330 B | 怪物 / 道具 / 道具配置 / 出入口表 |
| 索引 | `FILES.DAT` 2,254 B｜`FILES.DTT` 5,829 B | 檔案索引，可能是資源表 |
| 存檔 | `PARTY.DAT` / `PARTY.BAK` 1,494 B｜`TEMPLAT1/2/4/5.DAT` 256 B | 隊伍存檔 + 預建角色範本 |
| 開場 | `1SS..5SS.DAT` 511 B ×5｜`ALL_SS.DAT` 2,560 B | 疑似開場字幕 / 分鏡序列 |
| 文本 | `T.TXT`、`T2C.TXT`、`T2D.TXT`、`DATA1..5.TXT`、`EREGORE.TXT`、`WIN.TXT`、`TOWN.TXT` | NUL 分隔字串。`DATA*.TXT` 夾雜數字欄位(`255 · 文字 · 0 · 255 · 1 …`) → **事件表結構** |

### 2.3 3.5× 關係的解讀(格式破解起點)

CGA→EGA 檔案大小精確 3.5 倍，可分解為:

```
3.5 = 2 (2bpp → 4bpp 色深)  ×  1.75 (200 → 350 掃描線)
```

當初據此推測「EGA 版素材為 320×350、4 bitplane」。**2026-07-25 實作解碼器並肉眼比對
DOSBox 截圖後,這個推測要分開講**:

- **成立的部分**:`.PIE` 全螢幕圖/人像框確為 4-plane sequential + MSB-first,
  檔內高度確實 ×1.75(人像 144→252);多出的 16 bytes 確實是內嵌調色盤(用標準 EGA
  調色盤會失真,反證內嵌表為真)。`.SHE` 精靈圖也是 4-plane(列內 plane 分塊)。
- **要補的部分**:3.5× 只描述**檔案層**。EGA 素材在**顯示層**還會再水平加倍一次,
  通則是「檔案存半寬、顯示時寬 ×2、高 ×1.75」——`.SHE` 在載入時加倍
  (`FUN_217b_0adf`)、`.PIE` 在 blit 時加倍(`FUN_217b_0884`)。
- **不成立的部分**:`OPEN.PIE` 不符 3.5 倍規律,仍未解。
- **「320×350」這個具體尺寸並未被證實** —— 畫面模式已確認是 EGA 640×350,
  但素材檔各有自己的尺寸(人像框檔內 144×252 等),不是全域畫布尺寸。

> **教訓**:檔案大小的算術關係是很好的**起點**,但它只約束「總位元數」,
> 不約束佈局方向,也不約束切分單位。sprite frame 尺寸這一項本專案連續猜錯兩輪
> (CGA:32×16 → 16×32 → 實際 **16×16**;EGA:16×56 → 32×28 → 實際 **16×28**),
> 每一輪的錯誤版本都整除、都通過全部自動測試。
> 定案靠的是遊戲初始化時自己宣告的 frame 大小常數 `[0x5226]`
> (EGA `0xe0`=224、CGA `0x40`=64),以及「每個 frame 必須自成一體」的肉眼判準。
> 算術自洽只是必要條件,對照原版才是充分條件。

詳見 `docs/formats/graphics.md`。

---

## 3. 待驗證項(不得憑推測寫進實作)

| # | 問題 | 驗證方法 |
|---|---|---|
| ~~V1~~ | ~~原版有沒有配樂?~~ | **已解 2026-07-24**:只有 PC speaker。反組譯確認 AdLib(0x388/9)、MIDI(0x330/1)零命中,只有 8253+speaker(0x42/43/61)。有 note-sequencer(INT 1Ch)播離散音效,無背景音樂。見 `docs/re/03`。**影響「音樂還原」目標,見 §3.2** |
| ~~V2~~ | ~~EGA 素材是否真為 320×350 4-plane?~~ | **已解 2026-07-25**:`.PIE` 已驗證(16-byte 內嵌調色盤 + 4-plane sequential,檔內 144×252、顯示 288×252);`.SHE` 已驗證(檔內 frame 16×28、224 B,4-plane 列內分塊,載入時水平加倍成 32×28);CGA `.PIC` 全解、`.SHP` frame 16×16。畫面模式是 EGA 640×350。剩 `OPEN.PIE` 未解(見 V3)。見 `docs/formats/graphics.md` |
| V3 | `OPEN.PIE` 102,160 B 為何不符 3.5× 比例? | **確認確實不符**:`(102160-16)/4` 無因數 5,排除任何 320/640 寬佈局。六種候選皆雜訊,未解 |
| ~~V4~~ | ~~`SUM.MAP` 15,743 B 是否壓縮?~~ | **已解 2026-07-24**:是 RLE 壓縮的 23 個 sub-map 串接,size 表加總=15,743(已複核)。含 map_id 2/4(無獨立 .MAP 檔者)。見 `docs/re/03` |
| ~~V5~~ | ~~`DATA*.TXT` 的數字欄位語意~~ | **已解 2026-07-24**,見 §3.1、`docs/re/02` |
| V6 | `FILES.DAT` / `FILES.DTT` 的角色 | 部分解:同屬一份資源清單,但消費該表的程式碼未定位。見 `docs/re/03` |
| ~~V7~~ | ~~8087 用在哪?~~ | **已解 2026-07-24**:命中/傷害是**純整數**運算。全檔平坦掃描只有 6 筆真 x87 指令,全在啟動期 runtime 樣板;唯一浮點使用點(RNG 換算)是**軟體模擬**非硬體 x87。**Go 引擎不需處理 x87 精度對齊**。見 `docs/re/06` |

### 3.1 V5 已解:事件表結構(2026-07-24)

`DATA1..5.TXT` 是**地城事件表**,每筆記錄的結構為:

```
敘述文字 \0 [遭遇數量 \0 怪物ID×N \0 (255 終止符) \0 trailer(0~3 個數字)]
```

- 數字全部是 ASCII 十進位文字,NUL 分隔
- **怪物 ID 是 `MONSTER.DAT` 的 0-based 索引**(該檔共 99 隻怪物)
- 五個檔各自對應一個地城區域

**驗證方式(獨立複核通過)**:把怪物 ID 對回 `MONSTER.DAT` 的名稱,與同一筆記錄的敘述文字比對:

| 事件文字 | 解出的怪物 ID | `MONSTER.DAT` 名稱 |
|---|---|---|
| 「四隻狗頭人的帳篷」 | `[26, 26, 26, 26]` | Kobold ×4 |
| 「Uffuspgot 狗頭人隊長的帳篷」 | `[26, 26, 85, 2, 2]` | Kobold ×2 + **Uffuspgot** + Orc ×2 |
| 「cave bear 的巢穴」 | `[67]` | **Cave bear** ×1 |

文字敘述、怪物種類、隻數三者完全吻合,且 Xeres(91)、Remondadin(93)、Jesric(96)、
Eregore(97)、Guardian(98)等劇情角色都對得上攻略描述的場景。這是實證,不是推測。

**trailer 之謎已於反組譯後解開**(2026-07-24):`FUN_25be_0e77` 的實際 parse 流程證實
所謂「trailer」不是變長欄位,而是兩個固定單值槽 + 下一筆記錄的 leading picture ID
被線性 tokenize 誤歸給前一筆。value 3 觸發自動接續重繪、`%` 觸發 rune-glyph 繪製。
完整結論見 `docs/re/02-data-loading-functions.md`。

詳見 `docs/formats/event-script.md`(已加修正 callout)、`docs/re/02`,解析器 `tools/parse_datatxt.py`。

### 3.3 RNG 已解:引擎對拍的基石(2026-07-25)

```
state = (state × 125) mod 2796203
```

- 狀態:32 位元,存在 real mode `0x481c`(低字)/`0x481e`(高字)
- 乘數 125(`0x7D`);模數 `0x2AAAAB` = 2,796,203,是 Wagstaff 質數 (2²³+1)/3
- 種子來源:DOS 系統時鐘 `INT 21h, AH=2Ch`(重現時改用固定種子以便對拍)
- 擲骰 `Roll(n)`:負數取絕對值;n=0 或 1 直接回 1 且**不推進狀態**;否則 `floor(uniform × n) + 1`

**指令級證據**(段 `30c2`):

```asm
30c2:000f  MOV BX,0x7d      ; 乘數 125
30c2:0012  MUL BX           ; 兩次 16-bit MUL 湊 32-bit 乘法
30c2:0017  MUL BX
30c2:0029  MOV CX,0xaaab    ; 模數低位
30c2:002c  MOV BX,0x2a      ; 模數高位 → 0x2AAAAB
30c2:004a  SUB word ptr [0x481c],AX
30c2:004e  SBB word ptr [0x481e],DX
```

原版把狀態轉浮點再乘 n,用的是**自製軟體浮點函式庫**(段 `310e`,模擬 80 位元擴充精度),
不是硬體 x87。`internal/rng` 改用整數運算取代:遊戲中最大只到 `RNG(100)`,遠小於模數,
兩者結果一致,且免除浮點捨入的平台差異。

> 先前 `docs/re/06` 把除法輔助 `FUN_3016_0068` 列為卡點。實地稽核發現該函式反編譯乾淨
> (59 行、零警告),是上一輪預算用盡而非真的解不開 —— 這也是「未解」標記要定期複查的理由。

實作:`internal/rng`,7 項測試全綠(含與原版 32 位元借位減法算術的逐步對拍 10 萬步)。

### 3.2 V1 已解:原版無配樂,只有 PC speaker 音效(2026-07-24)

反組譯確認(已獨立複核 port I/O):

- **無任何 FM/MIDI 硬體支援**。AdLib(port `0x388`/`0x389`)、MIDI(`0x330`/`0x331`)在整個
  `DEMON.INT` 反組譯中**零命中**;唯一的聲音 I/O 是 8253 timer channel 2 + speaker gate
  (`0x42`/`0x43`/`0x61`),全部落在音效引擎 segment `1d9f`。
- **有 note-sequencer,但只播離散音效,不是背景音樂**。INT 1Ch(timer tick)handler 走一張
  4-byte/record 的表(除頻值 + 持續時間)。effect `-1` 是 8 音符死亡旋律,`1`~`8` 是精確的
  C 大調音階單音,`0` 是靜音。觸發點是命中/落空/傷害/死亡等戰鬥事件。

**對專案目標的影響**:CLAUDE.md 原列「音樂/音效都要還原」。**原版根本沒有音樂可還原**——
所以這條目標的正確範圍是:**忠實重現 PC speaker 音效序列**(那張 4-byte 表 + C 大調音階 + 死亡旋律),
而**不是**自製配樂填補。自製配樂會違反保存的本意(見 rulebook/93 素材鐵則、83 完整性)。
`audio/` 模組據此定位為 PC speaker 合成,不是 music player。

詳見 `docs/re/03-audio-and-resources.md`。

---

> **已有線索**:`PARTY.DAT` 的欄位位移在社群攻略第 6 節(HEX EDITING)已公開一部分
> (角色 1–5 的力量/智力/耐力/技巧/HP/SP/經驗值、隊伍金錢在 `0x51E`)。
> 整理版見 `docs/walkthrough/part-6.md`。這是階段 2 存檔格式的起點,但只涵蓋部分欄位,
> 其餘仍需反組譯補齊 —— 別把它當完整規格。

---

## 4. 架構決策(2026-07-24 定案)

| 決策 | 選擇 | 理由 |
|---|---|---|
| **引擎路線** | 反組譯當 oracle + Go/Ebiten 乾淨重寫 | 直接照抄反編出的 `FUN_xxx` 會纏繞 8086 runtime、無型別、不可維護，也無法中文化。反編只用來確認「原版怎麼算」，程式碼手寫 |
| **素材範圍** | EGA(`.PIE/.SHE/.FNE`)優先，CGA(`.PIC/.SHP/.FNT`)隨後**必做** | 完整性原則:兩套素材都是要保存的數位文物，CGA 不是砍掉的選項，只是排序在後 |
| **中文排版** | 內部渲染畫布拉到 640×400 | 中文需 16×16 點陣才可讀。英文 8×8 字型放大兩倍後與中文同高，版面比例一致。代價是所有 UI 座標要重排 |
| **版控** | 現有 repo `wicanr2/demon_winter_cht`(目前為空)，`git init` + 接遠端 | 原版 zip / 解壓資料 / PDF / Ghidra 專案全部 gitignore |

### 4.1 引擎分層

```
┌─────────────────────────────────────────────────────┐
│  cmd/demonwinter    ← Ebiten 主迴圈、視窗、輸入      │
├─────────────────────────────────────────────────────┤
│  ui/     選單、對話框、戰鬥 HUD、640×400 版面        │
│  i18n/   UTF-8 翻譯覆蓋層(英文 key → 繁中)         │
│  font/   8×8 英文點陣 + 16×16 CJK 點陣渲染          │
├─────────────────────────────────────────────────────┤
│  game/   規則層:角色、法術、戰鬥、時間、事件流程    │
│  script/ 事件腳本直譯器(DATA*.TXT / TOWN*.DAT)     │
├─────────────────────────────────────────────────────┤
│  assets/ 格式解碼:PIC/PIE/SHP/SHE/FNT/FNE/MAP/DAT  │
│  audio/  PC speaker 音效合成(依 V1 結果調整)       │
└─────────────────────────────────────────────────────┘
```

每層對上只露窄介面(deep modules):`assets` 只回傳解好的 `image.Image` 與結構體，
不讓 EGA plane 佈局洩漏到 `game`;`i18n` 只在 `ui` 邊界作用，`game` 不知道語言存在。

---

## 5. 階段規劃

> **⚠ 狀態的單一真相在 [`CONTEXT.md` §7 Worklist](./CONTEXT.md#7-worklist狀態的單一真相來源)。**
> 本節的 checkbox 於 2026-07-26 對程式碼核實同步過一次，之後可能再度落後 ——
> 判斷「某項做了沒」以 §7 與實際程式碼為準，不要只看這裡的方框。

### 階段 0 — 基礎建設

- [x] `git init` + 接遠端 + `.gitignore`(原版資料不入版控)
- [x] Ghidra 反組譯環境(docker),見 `docs/re/00-ghidra-setup.md`
- [x] DOSBox reference 環境(docker + Xvfb + xdotool),見 `docs/re/01-dosbox-reference.md`。
      **EGA 全鏈路跑通**(開機→選單→進遊戲→移動→存檔),可自動化送鍵 + 截圖。
      交叉驗證:進遊戲畫面的指令選單與我們從 DEMON.INT 抽出的字串一字不差。
- [x] Go + Ebiten 建置環境(`tools/go.sh`,全程 docker)
- [x] `CONTEXT.md` 建立詞彙表(見 `CONTEXT.md` §4 術語表)

**驗收:已達成**。DOSBox 跑出開場畫面(`workplace/dosbox/shots/01-ega-opening.png`,
可見 Novotrade 移植署名)並完整進入遊戲。

> **⚠ 完整性待辦(CGA 缺檔)**:這份 `Demons Winter (1988).zip` **缺少 `GOT.FNT`**
> (CGA 版裝飾字型),只有 EGA 版的 `GOT.FNE`。CGA 模式讀到 `GOT.FNT` 會卡死,
> 開場美術(`OPEN.PIC`)本身渲染正常。依完整性原則(rulebook/83),這不是「CGA 不做」的
> 理由,而是**素材缺檔待補**——需另尋完整版遊戲檔補 `GOT.FNT`。在此之前 CGA 標記為
> 「美術可用、互動不可用」,邏輯層驗證一律走 EGA(兩者共用同一份 `DEMON.INT`)。

### 階段 1 — 反組譯當 oracle

- [x] Ghidra 載入 `DEMON.INT` → 353 個函式,位址換算公式已驗證
- [x] 字串錨定路徑驗證可行(`Cast what spell:` → `FUN_1000_293d` 施法選單邏輯)
- [x] 定位檔案載入函式(DATA/MAP/TOWN/SUM.MAP,見 `docs/re/02`、`03`、`05`)
- [x] 定位聲音輸出點 → V1 結案(見 `docs/re/03`)
- [x] 流程驅動層三大塊反組譯(見 `docs/re/04`~`06`):
  - 主迴圈指令分派表(`FUN_222f_0b0e` + 三張平行表,已驗證按鍵表 `WPSCLTDMEUIVXRQ`)
  - 事件觸發(座標→EXITS→事件索引,`FUN_222f_1321` 寫回共用全域 `0x52f4`)
  - 戰鬥框架 + 命中/傷害公式(整數)、隨機遭遇 1/64、勝利條件鏈
- [x] **RNG 精確公式已解**:`state = (state × 125) mod 2796203`(見 §3.3)
- [x] 頂層狀態機的 overworld 移動 / town 進出迴圈(逐行讀 asm 解掉,見 `docs/re/04`、`08`)
- [x] `PARTY.DAT +0xa0` 已確認為每小時步數計數器；`+0x9c` 是獨立的遭遇倒數。
- [x] 法術效果、7 個戰鬥選單動作(Attack 以外)的細節(`docs/re/09`、`15`、`16`、`23`)

> **位址換算公式**(已實測驗證,後續反組譯一律沿用):
> `file_offset = (segment − 0x1000) × 16 + offset + 0x3C00`
> 陷阱:MZ header 的 entry `2037:0009` 是連結期相對值,Ghidra 載入基準為 segment `0x1000`,
> 真正 entry 在 `3037:0009`。別把 header 原始 CS/SS 當 Ghidra 位址。

**驗收**:每個抽出的演算法寫成 `docs/re/*.md`，含反組譯片段 + 重述的規則 + 對應原始 offset。
斷言任何機制前先在 DOSBox 實跑確認(rulebook/62 靜態溯源、rulebook/65 對 reference 驗收)。

### 階段 2 — 資料格式全解

- [x] `.FNE` / `.FNT` 字型 → **已解 2026-07-25**:
      EGA 16×14 1bpp(28 B);CGA 8×8 **packed** 2bpp(16 B)、**無檔頭**
      (3,073 B = 3,072 資料 + 結尾 `0x1a` DOS EOF)、96 字 × 2 bank(一般/反白)。
      兩套 atlas 皆逐字可讀,見 `docs/re/17`
- [x] `.PIC` 畫面圖 → PNG(CGA 全解,與 DOSBox 截圖肉眼比對一致)
- [x] `.PIE` 畫面圖/人像框 → PNG(4-plane + 內嵌調色盤,已驗證)
- [x] `.SHP` 精靈圖 → PNG sprite sheet(CGA 全解,frame **16×16**、64 B)
- [x] `.SHE` 精靈圖 → **已解 2026-07-25**:檔內 frame **16×28**、224 B,
      4-plane 列內分塊(`row×8 + plane×2 + col`),無壓縮。
      載入時水平加倍成 32×28、448 B(`FUN_217b_0adf`),blit 的 `0x1c0` stride
      指的是加倍後的緩衝區。證據:`1d9f:00bf [0x5226]=0xe0` + 檔案大小乘法 +
      六個檔逐格肉眼比對(六個檔一律 224 B,`CYPHER` 不是特例)
- [x] `OPEN.PIE` → **影像已解**(608×336,`docs/formats/graphics.md` §5),開場已接上。
      ⚠ 遺留謎題:原版找的是 `TITLE.PIC`,那個檔不在這份 dump 裡(不擋任何事)
- [x] `.MAP` 地圖 → 64×64 tile 陣列(ASCII 算繪確認),tileset 對應待解
- [x] `MONSTER.DAT`(99 隻)/ `ITEMS.DAT`(30 件)→ 解析器完成,部分欄位語意待驗
- [x] `TOWN*.DAT` 17-byte 記錄結構;設施代碼映射待反組譯
- [x] `DATA*.TXT` 事件表結構(V5 已結案,見 §3.1);trailer 語意待反組譯
- [x] `FILES.DTT` 主字串池(501 字串)
- [ ] `PARTY.DAT` 存檔格式 → 可讀可寫(已解姓名/種族/屬性/道具欄邊界;
      DOSBox 動態 diff 已解 X/Y 座標、朝向與時間欄位;
      職業、已學技能仍未解 → 待下一輪 DOSBox 實驗)
- [x] `SUM.MAP` 結構(RLE,`docs/re/02`)
- [ ] `FILES.DAT` 資源 arena 索引 2/3/6/8/14/15 的內容
- [x] `EXITS.DAT` 座標對應(`internal/assets/world/mapfile.go`,已接進事件查找)

**驗收**:每種格式的解碼輸出**肉眼比對 DOSBox 截圖**，不接受「程式沒報錯」當通過。
解碼器同時實作 encode，`decode(encode(x)) == x` 對全部原檔成立。

### 階段 3 — Go / Ebiten 引擎

- [x] `assets` 套件:所有格式的 Go 解碼器
- [x] 渲染層:640×400 畫布，EGA 16 色調色盤，sprite blit
- [x] 世界地圖移動、城鎮進出、地城探索(M1／M5)
- [x] 戰鬥系統(回合、移動點數、命中/傷害、法術)(M4)
- [x] 角色系統(建角、屬性、技能、裝備、升級)(M3)
- [x] 事件腳本**文字**直譯器 + 對話顯示(M2)
- [ ] 事件**動作**分派(21 格跳表只解出約 5 格語意,引擎只支援「顯示文字 + 開戰」)
- [x] 存檔 / 讀檔(與原版 `PARTY.DAT` 雙向相容,byte-for-byte 驗過)
- [x] 音效還原(依 V1,九個 PC speaker 效果)

**驗收**:每個子系統對 DOSBox 原版做同輸入對拍。
**不接受**用 debug hook(發道具/瞬移/強制進城)串起來的「能跑完」。

### 階段 4 — 繁體中文化

- [ ] 從 `DEMON.INT` 抽出全部 UI 字串 → 翻譯表
- [x] 從 `*.TXT` 抽出劇情文本 → 翻譯表(七個目錄 356 條 100%)
- [x] 16×15 倚天點陣字型(Big5 分區索引已驗證)
- [ ] `GOT.FNE` 花體風格的中文標題處理
- [x] 統一譯名表 + 漂移掃描(`translations/glossary.md`,`dwstrings check` 自動閘)
- [x] UI 版面重排以容納中文(640×400 畫布 + 排版格 + 標點禁則)
- [x] 遊戲內標題畫面接上(`OPEN.PIE` + 中文提示字)
- [ ] 標題**美術本身**仍是英文(花體 logo 沒重繪)

**驗收**:每個畫面截圖比對 — 無破框、無截字、無亂碼。譯名一致性掃描全過。

### 階段 5 — 驗證與打包

- [ ] headless 確定性回歸測試
- [ ] **正常玩家路徑實測**:新開檔 → 建角 → 走完主線 → 破關，全程不用任何 debug 捷徑
- [ ] 世界可達性檢查(flood-fill:落點、城鎮、船必須連通)
- [ ] 移除所有 debug hook 後重跑
- [ ] Linux / Windows / macOS 打包(Docker build)
- [ ] 玩家向 README(繁中)+ `ENGINEERING.md`(技術文件)分離

### 階段 6 — 攻略與文件

- [x] 統一譯名表 `translations/glossary.md`(426 條,涵蓋 21 節)
- [x] 官方遊戲手冊繁中版 `docs/manual/`(16 張掃描跨頁全數轉錄翻譯)
- [x] 社群攻略繁中版 `docs/walkthrough/`(1,914 行,含附魔表與存檔欄位表)
- [x] SSI 引擎架構研究報告 `docs/research/ssi-engine-architecture.md`
- [ ] 整理既有《遊戲攻略:冬之魔》PDF 為繁中電子攻略
- [ ] 資料格式規格書 `docs/formats/*.md`(4 份已完成,尚有未解項待補)

> 手冊與攻略的繁中化不只是「附加資料」:手冊定義了所有規則與數值,攻略提供可破關鏈與
> 部分存檔欄位位移,兩者是階段 1 反組譯的**文件層 oracle**(優先序低於 DOSBox 實跑,
> 高於反組譯推論)。同時譯名表一次定案,後續遊戲內文字中文化直接沿用,不會漂移。

---

## 6. 工作紀律(貫穿全程)

1. **完整性 > 投報**。不得以「成本高、效益低」為由跳過任何素材版本、任何一首曲子、任何一種格式。
   卡關就換方法(靜態反追溯 ↔ DOSBox 動態 ↔ 截圖 oracle)，記錄「卡在哪、試過什麼」，不寫「暫緩」。
2. **驗收看 reference，不看內部訊號**。編譯過 / 測試綠 / subagent 回報完成 都只代表內部自洽。
   宣稱任何「X 完成」前先問:一個真實玩家不碰任何後門，走得到 X 嗎?
3. **oracle 優先序**:人親自實測 > DOSBox 原版實跑 > 攻略/文件 > 反組譯推論。
4. **視覺產物一律 dump 出來肉眼比對**。「資料對但顯示錯」這種 bug 測試照樣綠。
5. **全程 Docker**;Python 一律 docker uv.venv，不污染系統。
6. **subagent 產出協調者獨立核實**才收(重跑測試 + 看 diff + 視覺 dump)。
7. **進度真相在 code + 本檔的 checklist**，不在某輪對話說過的「完成了」。

---

## 7. 主要風險

| 風險 | 影響 | 對策 |
|---|---|---|
| ~~`DATA*.TXT` 事件表語意解不出~~ | ~~劇情流程無法重建~~ | **已緩解**:事件表結構(V5)、座標→事件映射(`docs/re/05`)、觸發時機均已反組譯。剩「類別0房間為何被閘門排除」的矛盾待解(見 `docs/re/05` §4) |
| ~~反組譯進不了主邏輯~~ | ~~演算法只能靠黑箱觀察~~ | **已緩解**:字串錨定路徑驗證可行,已定位讀檔/事件/戰鬥/音效多個核心函式。jump table 誤判如預期出現,靠 `disassembly.asm` 原始指令繞過 |
| ~~RNG 精確公式未解~~ | ~~戰鬥無法對拍原版~~ | **已解 2026-07-25**:`state = (state × 125) mod 2796203`,模數 `0x2AAAAB` 是 Wagstaff 質數 (2²³+1)/3,狀態在 `0x481c`。指令級確認(`MUL BX(0x7d)`×2 + `0xaaab`/`0x2a` + `SUB`/`SBB`)。已實作 `internal/rng`,7 項測試全綠 |
| ~~8087 浮點精度差異~~ | — | **已排除**:命中/傷害純整數,x87 只在啟動樣板。Go 不需對齊浮點(V7 結案) |
| 中文塞不進原版版面 | UI 破框 | 640×400 畫布已預留空間;仍需逐畫面截圖驗收 |
| ~~原版無配樂~~ | ~~「音樂還原」目標需重新定義~~ | **已處理**:V1 確認只有 PC speaker,目標重定義為忠實重現音效序列(§3.2),不自製配樂 |

---

## 8. 目錄結構(規劃)

```
daemon_winter/
├── PLAN.md                    ← 本檔
├── CONTEXT.md                 ← 術語表 / 專案脈絡
├── ENGINEERING.md             ← 技術文件
├── README.md                  ← 玩家向(繁中)
├── cmd/demonwinter/           ← Ebiten 主程式
├── internal/
│   ├── assets/                ← 格式解碼器
│   ├── game/                  ← 規則層
│   ├── script/                ← 事件腳本直譯器
│   ├── ui/                    ← 版面 / 選單 / HUD
│   ├── font/                  ← 點陣字渲染
│   ├── i18n/                  ← 翻譯覆蓋層
│   └── audio/                 ← 音效
├── translations/              ← 翻譯表(TSV/JSON)+ 統一譯名表
├── docs/
│   ├── re/                    ← 反組譯筆記
│   └── formats/               ← 資料格式規格書
├── tools/                     ← 抽取 / 轉檔 / 對拍腳本(Python, docker uv)
├── docker/                    ← Dockerfile / compose(build + dosbox)
└── workplace/                 ← 原版資料與中間產物(gitignore)
```

---

## 9. 下一步

1. 階段 0 剩餘項:Docker 建置環境 + DOSBox reference 容器 + `CONTEXT.md`
2. 同時啟動階段 1 的 Ghidra 環境準備與階段 2 的字型格式(最小、最容易先拿到肉眼可驗結果的目標)
3. 先解 V1(有沒有配樂)— 它決定「音樂還原」這條目標的實際範圍
