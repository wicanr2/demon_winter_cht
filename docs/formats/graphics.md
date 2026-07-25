# EGA / CGA 美術素材格式（2026-07-25）

驗證方法：依假設寫解碼器（`internal/assets/gfx/`）→ 輸出 PNG（`workplace/dump/gfx/`，
不入版控）→ 肉眼比對 DOSBox 原版截圖（`workplace/dosbox/shots/`）。這份文件只記錄
**已經拿截圖或跨格式一致性驗證過**的結論，卡住的部分誠實列出試過哪些假設。

---

## 1. 3.5 倍假設：部分成立，部分推翻

PLAN.md 原假設「CGA→EGA 檔案大小精確 3.5 倍 = 2bpp→4bpp(×2) × 200→350 掃描線(×1.75)」。

結論：**分解方式對，但套用對象要分兩類**：

| 素材類別 | 3.5× 規律 | 驗證狀態 |
|---|---|---|
| 全螢幕圖 `OPEN.PIC`(16,000B) | 不適用(`OPEN.PIE` 是 102,160B，見 §5) | **已驗證**只有 CGA 版適用簡單模型 |
| 場景/人像框 `PIC*.PIC`/`PRIEST.PIC` 等(5,184B → 18,160B) | **成立**，且 1.75× 正是「高度縮放」而非寬度 | **已驗證**(肉眼比對，見 §3) |
| 精靈圖 `*.SHP`/`*.SHE`(1,728~15,360B → 6,048~53,760B) | **成立**，1.75× 同樣是「高度縮放」 | **已驗證**(肉眼比對，見 §4)，但 EGA 版**內部 bit 佈局未解出**(見 §4.3) |
| `OPEN.PIE`(102,160B) | **不成立**，非 3.5× 整數關係 | **未解**，見 §5 |

`3.5 = 2bpp→4bpp(×2) × 200→350掃描線(×1.75)` 這個分解本身是對的，但 1.75×
不是套在「全螢幕」上（EGA 也有跟 CGA 一樣的 320×200 16 色模式，不是只能 350
掃描線），而是這個引擎的美術資源在 EGA 版**把每張圖／每個 sprite frame 的高度
拉高成 CGA 版的 1.75 倍**（寬度不變），推測是美術製作流程本身的選擇（EGA 版重繪
時加了更多細節列數），不是顯示卡硬體規格的直接產物。

---

## 2. CGA 格式：全部已驗證

### 2.1 全螢幕圖 `.PIC`（`OPEN.PIC` 16,000B = 320×200÷4）

**已驗證**：320×200，2bpp（4 色），**線性排列**（不是 CGA 硬體 framebuffer
的偶/奇掃描線交錯！），每列 `width/4` bytes，每 byte 4 個像素、MSB-first。
調色盤是 CGA palette 1 高亮度：`{黑, 亮青, 亮洋紅, 白}`。

**肉眼比對**：`workplace/dump/gfx/open-pic-cga-linear.png` 與
`workplace/dosbox/shots/05-cga-hang-open-pic.png` **完全吻合**——魔鬼臉孔、
"DEMON'S WINTER" 標題字樣、"NOVOTRADE"／"STRATEGIC SIMULATIONS INC" 商標、
下方製作人員名單全部清楚對上。

**踩過的雷**：一開始比照 IBM CGA 硬體 0xB8000 framebuffer 的標準佈局（偶數掃描線
存前 8000B、奇數掃描線存後 8000B，這是 CGA 硬體因頻寬限制而做的交錯），寫出
`DecodeCGAPlanar320`，結果整張圖花掉、看不出圖案（見
`workplace/dump/gfx/open-pic-cga-interleaved-control.png` 對照組）。改成單純
線性由上到下排列（`DecodeCGALinear320`）才對上——**磁碟檔案格式不等於硬體
framebuffer 格式**，遊戲載入時才會做交錯（或者根本不需要，因為 CGA mode
4/5 也可以透過 BIOS/驅動把線性資料重排進顯存，檔案不必事先交錯）。這是本專案
第一個「數學算對但畫面不對，肉眼比對抓出來」的案例。

### 2.2 場景/人像框 `.PIC`（`PIC1~6.PIC`／`PRIEST.PIC`／`SHAMEN.PIC`／`THANATOS.PIC`，5,184B）

**已驗證**：144×144，同 §2.1 的線性排列、2bpp、CGA palette 1。
`5,184B = 144/4 bytes/row × 144 rows`，整除。

**肉眼比對**：`workplace/dump/gfx/cga-portrait-PIC1.PIC.png`、
`cga-portrait-THANATOS.PIC.png` 等都解出清楚的人形圖案（THANATOS 是披風骷髏
持鐮刀的形象），且跟同一角色的 EGA 版（見 §3）姿勢、輪廓完全一致——沒有 DOSBox
截圖直接對到這幾張單獨畫面，但**兩種獨立編碼(CGA/EGA)解出同一個形狀**是很強的
交叉驗證證據。

### 2.3 精靈圖 `.SHP`

**已驗證**：每個 sprite frame 是 **16 寬 × 32 高**（`CYPHER.SHP` 是 8×32，較窄），
2bpp、線性排列（同 §2.1）。檔案裡沒有 frame 數欄位，用「檔案長度 ÷ frame
byte 數」反推 frame 數：

| 檔案 | 大小(B) | frame bytes(16/4×32=128，CYPHER 是 8/4×32=64) | frame 數 |
|---|---|---|---|
| `COMBAT.SHP` | 2,816 | 128 | 22 |
| `SHIP.SHP` | 2,048 | 128 | 16 |
| `DEMON.SHP` | 6,528 | 128 | 51 |
| `WINTER.SHP` | 6,528 | 128 | 51 |
| `MONSTER.SHP` | 15,360 | 128 | 120 |
| `CYPHER.SHP` | 1,728 | 64 | 27 |

**踩過的雷**：一開始猜寬高相反（32 寬 × 16 高），byte 數學完全對得上（因為
`32×16 = 16×32`，總 pixel 數一樣，除法當然也對），但輸出全花（見
`workplace/dump/gfx/sweep-cga-monster-32x16-frame0.png` 對照）。改成
16 寬 × 32 高後，`MONSTER.SHP` 立刻解出清楚的武士／骷髏／各種怪物剪影
（`workplace/dump/gfx/cga-sprites-MONSTER.SHP.png`），`CYPHER.SHP` 解出一排
幾何符文圖示（`workplace/dump/gfx/cga-sprites-CYPHER.SHP.png`，猜測是法術
「符文」UI 圖示，對應 `DEMON.INT` 裡的字串 "God Runes / Fire Runes / Metal
Runes..."）。**這是本專案第二個「除法對得上、方向猜反」的案例**——byte 數量
整除只能驗證「切分點」對不對，驗證不了「哪個軸是寬哪個軸是高」，這一定要肉眼看。

---

## 3. EGA 全螢幕圖／人像框格式：已驗證

### 3.1 佈局

**已驗證**：`.PIE` 檔開頭 **16 bytes 是調色盤索引表**（第 i byte = 邏輯色 i
對應到哪個實體 EGA 16 色代碼），之後是 **4-plane、sequential（plane 0 整塊
接著 plane 1 整塊...）、MSB-first** 的點陣資料。場景/人像框（`PIC1~6.PIE`／
`PRIEST.PIE`／`SHAMEN.PIE`／`THANATOS.PIE`，18,160B）解碼尺寸為
**144×252**（144×1.75，寬度不變、高度拉 1.75 倍）：

```
18,160 B = 16(調色盤) + 18,144(點陣資料)
18,144 = 144/8 bytes/row × 252 rows × 4 planes
       = 18 × 252 × 4
```

### 3.2 調色盤

**已驗證是真的調色盤，不是佔位資料**：同一張圖分別用「內嵌 16 bytes 索引表」
與「標準 16 色盤（identity mapping）」解碼，兩者色彩明顯不同——內嵌調色盤版本
色調自然（如 `PIC1.PIE` 呈現正常的紅棕色惡魔膚色），標準色盤版本則偏向不自然
的高飽和撞色。詳見 `ParsePIEPalette`。

### 3.3 肉眼比對結果

| EGA 檔 | 輸出 | 比對結果 |
|---|---|---|
| `PIC1.PIE` | `workplace/dump/gfx/ega-portrait-PIC1.PIE-withpal.png` | 惡魔持鐮刀形象，姿勢與 `cga-portrait-PIC1.PIC.png`(CGA 版)一致，**清楚無雜訊** |
| `THANATOS.PIE` | `ega-portrait-THANATOS.PIE-withpal.png` | 披風骷髏、鐮刀、背景滿月，與 `cga-portrait-THANATOS.PIC.png` 輪廓一致，有少量垂直條紋殘留雜訊但主體清楚 |
| `PRIEST.PIE` | `ega-portrait-PRIEST.PIE-withpal.png` | 兜帽僧侶持蠟燭，**乾淨無雜訊，是目前最清楚的一張** |

沒有 DOSBox 截圖直接顯示這幾張單獨畫面（這些是劇情/道具彈出圖，目前擷取的截圖
只有開場、主選單、地城畫面），但**跨格式一致性**（CGA/EGA 兩份獨立編碼解出
同一主體、同一姿勢）加上**內嵌調色盤讓色彩變得合理**兩個獨立證據，足以判定
**已驗證**，不是巧合湊出來的雜訊圖案。

---

## 4. EGA 精靈圖 `.SHE`：格式**部分未解**

### 4.1 已驗證的部分：frame 邊界

用 CGA frame 尺寸(16×32／CYPHER 8×32)乘上「高度 ×1.75、色深 ×2(bytes)」，
EGA frame 應為 **16×56(CYPHER 8×56)、4-plane**：

```
16/8 bytes/row × 56 rows × 4 planes = 448 bytes/frame(CYPHER: 8/8×56×4=224)
```

用檔案長度除以這個 frame byte 數，**跟 CGA 版本的 frame 數完全對上**：

| 檔案 | 大小(B) | frame bytes | frame 數 | 與 CGA 版frame數對照 |
|---|---|---|---|---|
| `COMBAT.SHE` | 9,856 | 448 | 22 | = `COMBAT.SHP` 22 ✅ |
| `SHIP.SHE` | 7,168 | 448 | 16 | = `SHIP.SHP` 16 ✅ |
| `DEMON.SHE`/`WINTER.SHE` | 22,848 | 448 | 51 | = 51 ✅ |
| `MONSTER.SHE` | 53,760 | 448 | 120 | = 120 ✅ |
| `CYPHER.SHE` | 6,048 | 224 | 27 | = `CYPHER.SHP` 27 ✅ |

6 個檔案、全部整除、frame 數與 CGA 版一一對應——**frame 邊界的位置幾乎確定是對的**，
這不太可能是巧合。

### 4.2 未解的部分：plane 內部佈局

**PLANE 佈局假設全部試過，輸出都是雜訊**，用 `MONSTER.SHE` 逐一排除：

| 假設 | 說明 | 輸出 | 結果 |
|---|---|---|---|
| 逐-frame 4-plane sequential | 每個 frame 自己 plane0 整塊+plane1...(跟人像框同邏輯) | `ega-sprites-MONSTER.SHE-perframe-control.png` | 雜訊 |
| 整檔 4-plane sequential(global) | 把整份檔案當一張高圖解，4 個 plane 是檔案級分塊，再切 frame | `ega-sprites-MONSTER.SHE.png` | 雜訊 |
| 逐-frame row-interleaved | 每列 4 個 plane byte 相鄰 | `sweep2-monster-perframe-rowint-16x56.png` | 雜訊(比其他略「結構化」但仍不成圖) |
| 整檔 row-interleaved(global) | 整檔視為一張 16×6720 的圖，每列 4 plane byte 相鄰 | `sweep2-monster-global-rowint-16xtall.png` | 雜訊 |
| 寬高對調(56×16 global) | 懷疑 1.75× 套在寬度而非高度 | `sweep-monster-global-56x16.png` | 雜訊 |
| 不縮放(32×16 global，色深加倍但不拉高度) | 懷疑 sprite 不套用 1.75× 縮放 | `sweep-monster-global-32x16.png` | 雜訊 |
| chunky 4bpp row-major(16×56，8 byte/列) | 非 planar，每 byte 兩像素 | （協調者複驗，未存檔） | 雜訊 |
| chunky 4bpp column-major(16×56，28 byte/欄) | 同上但逐欄存放 | （協調者複驗，未存檔） | 雜訊 |

> **協調者補充（2026-07-25）**：以上兩個 chunky 假設由協調者另行複驗排除。
> 另外注意逐-frame 4-plane sequential 的輸出有**週期 4 列的疊影**
> （y=0 與 y=4 相同），這是平面切分位置錯位的典型徵狀，
> 支持上述「plane 佈局猜錯」而非「尺寸猜錯」的判斷 —— 16×56 這個尺寸
> （448 = 16×56×4bpp/8，且 56 = 32×1.75）在算術上是自洽的，值得保留。
>
> **一條交叉線索**：本專案在 `SUM.MAP` 的 RLE 解壓已證實原版使用
> **column-major** 走訪（見 `docs/re/03` 的 2026-07-25 修正）。
> 同一個引擎在 sprite 上也用逐欄佈局是合理的猜測方向，
> 但單純的 chunky column-major 已排除，要試的是
> **planar × column-major 的組合**（例如每欄各自 4 個 plane，
> 或整 frame 逐欄但 plane 在欄內交錯）。

**結論：EGA `.SHE` 的 4-plane bit 佈局尚未解出**，只有「frame 從哪個 offset
切」是確定的。可能原因(留給下一輪)：

1. Sprite 資料可能經過 RLE 或其他壓縮（PIE/PIC 全螢幕圖是 raw，但 sprite 檔案
   小、frame 多，遊戲引擎更有動機壓縮）——這會直接解釋為什麼「frame 邊界對、
   內容不對」：邊界(壓縮後大小)剛好整除是**未解碼前**的巧合，不代表解壓後
   還是這個尺寸。
2. Plane 順序可能不是 R/G/B/I 標準序，或每個 plane 內部 bit 順序不同於全螢幕圖。
3. 可能每個 frame 前面藏了一小段 per-frame header（如碰撞框、錨點），
   讓我目前假設的「frame 恰好從 offset N 開始」有系統性偏移。

原本反組譯 `FUN_217b_07cf`（`217b:07cf`，EGA sprite 藍圖 blit 常式，見
`workplace/ghidra/export/decompiled_all.c` 行 22926 附近）看到它用
sequencer map mask（`0x3c4`/`0x3c5`）逐 plane blit、frame stride
`0x1c0`(=448，與本文 frame byte 數吻合)，靜態證據原本支持「448 bytes/frame」，
但同一份反組譯裡的指標運算單位（word pointer vs byte pointer）在decompile
輸出中有歧義，不足以直接讀出 bit 級佈局，仍需要用真正的 disassembly(不是
decompile)逐指令複查，或者先確認是否有壓縮這個更大的問題。**這是本專案
目前唯一真正卡住的部分，如實記錄。**

### 4.3 CGA 精靈圖 vs EGA 精靈圖現況總結

CGA `.SHP` 全部 6 個檔案**已驗證**（清楚的怪物/符文剪影），EGA `.SHE`
**frame 邊界已驗證、內部畫面未解**。引擎若近期要用素材，CGA 版可以直接用，
EGA 版精靈圖需要下一輪繼續破解（建議先排除「是否有壓縮」這個問題，方法見
`rulebook/64`：用已知的 CGA 版當結構 oracle，比對 EGA 資料的 entropy/重複
pattern 判斷是否為 RLE）。

---

## 5. `OPEN.PIE`（102,160B）：未解，確認不符 3.5× 規律

`(102,160 - 16) / 4 = 25,536 = 2^6 × 3 × 7 × 19`，**沒有因數 5**，代表無法
用任何「寬度是 320 或 640 的倍數（含因數 5）」的簡單全螢幕 4-plane 佈局整除。
試過的候選全部輸出雜訊：

| 候選 | 說明 | 結果 |
|---|---|---|
| 320×200 / 320×350 / 320×399 / 608×336 / 456×448（扣 16B 調色盤後） | 湊出能被 25,536 整除的尺寸組合 | 全部雜訊 |
| 640×350（4-plane 標準 EGA 高解析模式） | 尺寸不合，`102,144 < 112,000`，連 decode 都跑不完 | 資料不足，排除 |
| 5 幀動畫，每幀套用人像框公式(144×252，18,144B/幀) | `102,160/18,160=5.6`，非整數，仍探索性跑了 5 幀 | 全部雜訊 |

**結論**：`OPEN.PIE` 確認是 PLAN.md 標記的異常檔案，不符合本文件其他 EGA 素材
共通的「線性 frame × 4-plane sequential」模型。可能性(未驗證，留待下一輪)：

- 含有 §4.2 提到的同一種未知壓縮/編碼，且因為是開場全螢幕大圖，壓縮效果更明顯
  （惡魔插畫大面積同色，RLE 壓縮率會比小 sprite 高很多，與「檔案沒有依 3.5×
  等比放大」的觀察一致）。
- 是多幀動畫（開場可能有漸變/掃入效果），但幀之間可能有差分編碼而非各自獨立
  一張完整圖。
- 有目前未知的檔頭欄位（不只 16 bytes）標示實際尺寸/幀數。

---

## 6. 字型 `.FNT`/`.FNE`：未解

`ASC.FNT`（3,073B = 1(header) + 256×12）試著用「VGA BIOS 標準 8×12
點陣字型」格式解碼（每字元 12 bytes、MSB-first、1bpp），輸出見
`workplace/dump/gfx/font-asc-fnt.png`——**是雜訊，不是可讀的英文字母**。用
ASCII 0x20(space)對應的 glyph 檢查，該位置應為全 0(空白)，實際卻是密集黑白
點陣，證明目前的「1-byte header + 256 連續 8×12 glyph」假設是錯的（可能是
glyph 尺寸不對、bit 順序不對、或者索引不是照 ASCII 碼直接排列）。`ASC.FNE`／
`GOT.FNE`(2,688B = 224×12，去掉 header 後剛好整除)同樣輸出雜訊。

**這部分不在本任務的核心交付範圍內(PLAN 待驗證項是 V2/V3 圖形格式，字型未列入)，
留 TODO**，`internal/assets/gfx/font.go` 的 `DecodeMonoFont8x12` 保留給下一輪用，
但**目前的輸出不能當作字型已解出**。

---

## 7. 總結表

| 項目 | 狀態 | 證據 |
|---|---|---|
| CGA `.PIC` 全螢幕(`OPEN.PIC`) | ✅ 已驗證 | 對上 DOSBox 截圖 05 |
| CGA `.PIC` 人像框(9 檔) | ✅ 已驗證 | 清楚圖案 + 與 EGA 版一致 |
| CGA `.SHP` 精靈圖(6 檔) | ✅ 已驗證 | 清楚怪物/符文剪影 |
| EGA `.PIE` 人像框(9 檔) | ✅ 已驗證 | 清楚圖案 + 與 CGA 版一致 + 調色盤合理 |
| EGA `.SHE` 精靈圖(6 檔) | 🟡 部分:frame 邊界已驗證,plane 內部佈局未解 | 6/6 frame 數與 CGA 版對上;6 種佈局假設全部雜訊 |
| `OPEN.PIE` | ❌ 未解 | 不符 3.5×,6 種候選尺寸全部雜訊 |
| `.FNT`/`.FNE` 字型 | ❌ 未解 | 空白字元位置非空白,判定佈局假設錯誤 |

3.5 倍假設：**對場景/人像框、精靈圖成立(高度縮放 1.75× + 色深加倍)，
對 `OPEN.PIE` 不成立**——原假設「分解成 2×1.75」方向正確，但套用範圍
需要修正為「非全螢幕的個別素材」，且套用軸是高度不是掃描線數。

---

## 8. 給引擎渲染層的建議

- **可以立刻用**：CGA 全部格式(全螢幕/人像框/精靈圖)、EGA 全螢幕圖(人像框)。
  `internal/assets/gfx/` 的 `DecodeCGALinear320`／`DecodeCGASpriteSheet`／
  `DecodeEGAPlanar`(配 `ParsePIEPalette`)可直接串進 Ebiten 渲染管線。
- **不能用**：EGA `.SHE` 精靈圖、`OPEN.PIE`、`.FNT`/`.FNE` 字型——這三塊
  目前解碼結果是雜訊，直接串進引擎會畫出亂碼圖，需要下一輪反組譯或壓縮格式
  分析才能繼續。若引擎近期要跑起來，建議先用 CGA 版精靈圖(視覺效果較陽春
  但正確)頂著，EGA 精靈圖之後補。
