# `217b` 段：EGA 精靈 blit 與 `.SHE` 精靈圖格式（2026-07-25）

本檔攻的是 PLAN 標記投報最高的一塊：`217b` 段（24 個函式，先前只碰過 1 個）的
EGA sprite blit 邏輯，目標是解出 `.SHE` 精靈圖的真實位元佈局。

> **⚠ 2026-07-25 訂正（本檔的核心修正）**
>
> 本檔原本斷言「`.SHE` 的 frame 是 **32×28、448 bytes**」並標「已驗證」。
> **那個尺寸描述的是記憶體緩衝區，不是磁碟檔案。**
> `FUN_217b_07cf` 讀的資料在載入時已被 `FUN_217b_0adf` 就地做過水平像素加倍，
> 所以它看到的 stride `0x1c0`(448) 是加倍後的值。
>
> - **檔案內**：frame **16×28**、**224 bytes**，每列 8 bytes（4 plane 各 2 B）
> - **記憶體內**（本檔 §2、§3 描述的對象）：frame 32×28、448 bytes，
>   每列 16 bytes（4 plane 各 4 B）
>
> `EGAPlanesRowBlocks` 這個 plane 佈局**本身是對的**，錯的只有 rowBytes。
> 載入鏈的證據見 §2.6，檔案格式的完整結論見 `docs/formats/graphics.md` §4。
> 本檔以下各節凡提到 448／32 寬，一律指**記憶體內**的 frame。

**結論先講**：`.SHE` 的 4-plane 佈局**已解出並肉眼驗證**——直接反組譯
`FUN_217b_07cf`（單一 sprite blit 常式）的原始指令（不是 decompile，decompile
對這段的指標單位有歧義，已依任務指示改讀 `disassembly.asm`）得到位元組精確的
定址公式。套用到**檔案**時 rowBytes 減半（16×28），
`MONSTER.SHE`／`COMBAT.SHE`／`DEMON.SHE`／`WINTER.SHE`／`SHIP.SHE`／`CYPHER.SHE`
六個檔全部解出乾淨可辨識、逐格自成一體的圖案，且與 CGA `.SHP` 一格不差對應。

## 讀法／驗證方法

依 `rulebook/62`（靜態溯源）與任務指示：不重試 `docs/formats/graphics.md`
已排除的 8 種佈局假設，改成**逐指令讀 `FUN_217b_07cf` 的 disassembly**，
把指標的「每次迭代淨位移量」跟「迴圈跑幾次」直接讀出來，反推來源資料的
真實排列，寫成解碼器後再用 PNG 肉眼比對驗證（`rulebook/64`：已有解碼器
候選 + 已知 CGA 版當 oracle，不必再靠猜）。

**這條路解對了 plane 佈局，卻解錯了 frame 尺寸**——因為它只讀 blit 端，
而 blit 讀的緩衝區在載入時已經被改過。補上的做法是把溯源往上游再推一段：
從「誰填了這塊緩衝區」找到素材載入器 `FUN_1d9f_0a8b`，
再往上找到視訊初始化裡遊戲自己宣告的檔案格式常數 `[0x5226]`。
**靜態溯源要追到資料的來源端，不能停在消費端。**

---

## 1. `217b` 段函式清單與角色

以下依 `workplace/ghidra/export/functions.csv`（`217b` 段 24 個函式）逐一列出，
角色判斷來源標註「已驗證」（反組譯證據直接支持）或「[推測]」（結構像但未逐句
核對遊戲行為）。

| 位址 | 大小(B) | 角色 | 判斷依據 |
|---|---|---|---|
| `217b:000d` | 15 | INT 10h（BIOS 視訊服務）呼叫 trampoline | `swi(0x10)` 直接呼叫，已驗證 |
| `217b:001c` | 23 | INT 21h（DOS 服務）呼叫 trampoline，回傳值檢查 `AX != -8`（`0xFFF8`，DOS「磁碟已滿」錯誤碼附近） | `swi(0x21)` + 錯誤碼比對，已驗證呼叫慣例；錯誤碼語意 [推測] |
| `217b:0033` | 237 | 橫向掃描線**圖樣填色**（dither fill，用 `0xCC`/`0x33` 棋盤位元遮罩），操作兩個 `0x2000`(8192) bytes 的 bank，帶左右邊界遮罩 | 已驗證是遮罩式橫向 span 填色迴圈；用途(UI 邊框?)[推測] |
| `217b:0120` | 181 | 橫向掃描線**單色實心填色**（帶左右邊界位元遮罩，`param_5`是填色 byte） | 已驗證迴圈與邊界遮罩邏輯；跟 `0033` 是同一個定址公式的姊妹函式 |
| `217b:01d5` | 95 | 固定尺寸(72×2×18 words)區塊複製，來源 `DAT_31f0_5530`，逐列跳 `0x28`(40) elements | 已驗證迴圈結構；內容(視窗框/字型模板?)[推測] |
| `217b:0234` | 38 | 把兩個 `0x2000`(8192) bytes bank 整塊填成同一個 byte pattern（清空/清色） | 已驗證 |
| `217b:025a` | 93 | 從表 `DAT_31f0_5488` 讀 8 個 word（4+4 兩段），逐列寫進輸出、列距 `0x28` elements(=80 bytes，跟螢幕列距相同) | 已驗證是單一 8-row×16px 小圖(單一 plane，無 `OUT` 呼叫，plane 選擇由呼叫端先設好)複製；用途(游標/小圖示?)[推測] |
| `217b:02b7` | 129 | 矩形區域 XOR 全反白（`^0xffff`），操作 offset `0`與`+0x1000` 兩個 bank | 已驗證是反白(選取效果)迴圈 |
| `217b:0338` | 12 | INT 10h trampoline（同 `000d`） | 已驗證 |
| `217b:0344` | 207 | **9×9 網格**批次 blit：依查表 `DAT_31f0_5a82[i]` 取得每格的來源索引，`*0x40`(64 bytes/單位)取資料，逐格複製到列距 `0x140`(320 elements=640 bytes) 的緩衝區，可選 XOR 反白；**不呼叫任何 `OUT` 埠**（不是直接寫 EGA 硬體，是寫一塊一般記憶體緩衝區） | 已驗證迴圈結構；是否為 sprite cache/合成緩衝 [推測] |
| `217b:0413` | 151 | `0344` 的單格版本（同樣 `*0x40`、無 `OUT`） | 已驗證 |
| `217b:04aa` | 209 | 橫向矩形**實心填色**，用 GC（Graphics Controller，port `0x3ce`/`0x3cf`）Bit Mask 暫存器(index 8)做左右邊界，操作 `640` 寬緩衝區(`*0x50+DAT_53de`) | 已驗證是硬體 bit-mask 加速填色（畫 UI 方框/清區域）常式 |
| `217b:057b` | 231 | 同 `04aa`，但填色來源會依 `*(int*)0x5a80` 切換兩種色值（`-0xe8`/`-0x33f8`） | 已驗證邏輯結構；雙色語意 [推測]（可能對應 EGA/CGA 兩種色深模式切換） |
| `217b:0662` | 53 | **清空整個 640×350 4-plane 畫面**（GC index5 設 Write Mode，寫 `28000` bytes 到 `DAT_31f0_53de`；`28000 = 640×350÷8`，剛好是 EGA mode 0x10 單一 plane 大小） | 已驗證，且**是本文件推出「螢幕是 EGA 640×350 模式」的關鍵證據**（見 §2.4） |
| `217b:0697` | 98 | 矩形區塊用 GC Data Rotate 暫存器(index3=`0x18`→ 邏輯功能位元=XOR)做**硬體 XOR 反白填色**，列距 `0x460`(1120)、`param_4*0xe`(14 列一組) | 已驗證是硬體加速反白，可能用於文字/選單游標高亮(14px 列高常見於文字 cell) [推測用途] |
| `217b:06f9` | 214 | **9×9 網格 sprite blit**：依查表 `DAT_31f0_5a82[i]*0x1c0`(448 bytes/frame)取 sprite 資料，用 Sequencer Map Mask(`0x3c4`/`0x3c5`)逐 plane 寫進 640 寬 EGA 畫面；**跟 `07cf` 用完全相同的位移常數**(`+8` elements/列、`-0xd6` elements 換 plane、28 列、mask 1,2,4,8) | **已驗證**——這是本文件破解 `.SHE` 佈局的第二個獨立證據來源，交叉確認 `07cf` 讀出的定址公式 |
| `217b:07cf` | 181 | **單一 sprite blit 常式**（本文件主角，見 §2） | 已驗證，逐指令核對 |
| `217b:0884` | 135 | **`.PIE` 全螢幕圖/人像框的 blit**：迴圈呼叫 `FUN_217b_0a95`(252 列 × 9 次)，外層依 plane mask 1,2,4,8 跑 4 輪。每次呼叫吃 2 個來源 byte 吐 4 個 byte → 18 bytes/列變 36 bytes/列，**即水平像素加倍**，所以 `.PIE` 檔內的 144 寬畫到畫面上是 **288 寬** | 已驗證（`0a95` 的位元展開語意見下一列） |
| `217b:090b` | 113 | 用 GC Bit Mask 暫存器寫**單一 byte**（`1 << (X&7 ^ 7)` 遮罩），像是畫單一 8px 寬直條/游標線 | 已驗證迴圈與遮罩算式；用途 [推測] |
| `217b:097c` | 261 | 14 列、逐列 3-byte 且帶**次 byte 位元位移**（`sVar5`）的小圖 blit，來源 `DAT_31f0_5488` + `param_3*0x1c` | 已驗證結構；這是「非 byte-aligned X 座標」的小圖繪製（跟 `07cf` 的 byte-aligned 版本互補），[推測]用於游標/選取箭頭等允許任意 X 位置的小圖 |
| `217b:0a81` | 20 | INT 10h trampoline（同 `000d`/`0338`） | 已驗證 |
| `217b:0a95` | 74 | **水平像素加倍**：讀 1 個來源 byte，核心迴圈 `SHL AX,1 / MOV DH,AH / AND DH,1 / SHL AH,1 / ADD AH,DH` 逐 bit 複製兩次，吐出 2 個 byte（呼叫端 `0884` 每列連呼 9 次 = 每列 18 B → 36 B） | 已驗證。**⚠ 本檔原斷言「不在 sprite 路徑上」已推翻** —— 它不在 blit 路徑上，但同家族的 `0b19` 在**載入**路徑上，見 §2.6 |
| `217b:0adf` | 58 | **`.SHE` 載入時的就地加倍**：呼叫 `FUN_217b_0b19` `(param_1 & 0x7fff)` 次，由後往前把 N bytes 的緩衝區展開成 2N bytes | 已驗證。由素材載入器 `FUN_1d9f_0a8b` 在副檔名為 `shE`/`SHE` 時呼叫，見 §2.6 |
| `217b:0b19` | 42 | 跟 `0a95` 同家族的水平像素加倍（4 次 shift，兩輪固定跑完寫一個 word） | 已驗證。**在 `.SHE` 載入路徑上**（不是 blit 路徑），見 §2.6 |

**小結**：24 個函式可分 4 群——
1. **sprite blit 家族**（`07cf`、`06f9`）：本文件核心，見 §2。
2. **實心/圖樣填色、反白家族**（`0033`、`0120`、`02b7`、`04aa`、`057b`、`0697`、`090b`）：UI 方框、選取高亮、清區域，共用同一個「640 寬 4-plane 畫面」定址模型，跟 sprite blit 是同一張畫布，但不讀 `.SHE`。
3. **一般記憶體緩衝 blit 家族**（`0344`、`0413`、`025a`、`01d5`）：不呼叫 `OUT`，寫的是系統 RAM 裡的合成/快取緩衝，不是直接寫 EGA 硬體；`0344`/`0413` 用的 frame 單位是 `0x40`(64 bytes)，不是 `.SHE` 緩衝區的 448。值得注意的是 64 剛好等於 **CGA `.SHP` 的 frame 大小**，這條線索留待後續追（超出本次任務範圍）。
4. **水平像素加倍家族**（`0a95`、`0adf`、`0b19`）+ **INT 10h/21h trampoline**（`000d`、`0338`、`0a81`、`001c`）：這一群是 EGA 素材「檔案存半寬、顯示時 ×2 寬」規則的實作，`.PIE` 走 `0884`→`0a95`（blit 時加倍）、`.SHE` 走 `FUN_1d9f_0a8b`→`0adf`→`0b19`（載入時加倍）。見 §2.6。

---

## 2. EGA sprite blit 完整流程（`FUN_217b_07cf`）

### 2.1 函式簽章與暫存器慣例

```
217b:07cf FUN_217b_07cf(byte param_1 /*BP+6*/, byte param_2 /*BP+8*/, int param_3 /*BP+0xA*/)
```

- `param_1`：目標**列格**座標（Y grid cell）。
- `param_2`：目標**欄格**座標（X grid cell）。
- `param_3`：**frame 索引**（第幾個 sprite frame）。

### 2.2 目標位址計算（畫面座標 → EGA video memory offset）

```asm
217b:07d7  MOV AX,0x31f0        ; DS = 0x31f0（本程式的全域資料段）
217b:07da  MOV DS,AX
217b:07dc  MOV AX,0xa000        ; ES = 0xA000（EGA/VGA 顯示記憶體固定段位）
217b:07df  MOV ES,AX
217b:07e1  XOR AH,AH
217b:07e3  MOV AL,[BP+0x6]      ; AL = param_1（列格）
217b:07e6  MOV CL,0x1c          ; CL = 28
217b:07e8  MUL CL                ; AX = param_1 * 28（8-bit MUL，只用 AL，等於後續有隱含 &0xff）
217b:07ea  ADD AX,[0x53e2]      ; AX += DAT_31f0_53e2（視窗/捲動 Y 基準）
217b:07ee  MOV CL,0x50          ; CL = 80（= 640px÷8，畫面一條 plane 掃描線的 byte 數）
217b:07f0  MUL CL                ; AX = (以上 &0xff) * 80  → Y 方向的 byte 位移
217b:07f2  MOV DI,AX
217b:07f4  XOR AH,AH
217b:07f6  MOV AL,[BP+0x8]      ; AL = param_2（欄格）
217b:07f9  MOV CL,0x5
217b:07fb  SHL AX,CL             ; AX = param_2 * 32（欄格間距 = 32 px）
217b:07fd  ADD AX,[0x53e0]      ; AX += DAT_31f0_53e0（視窗/捲動 X 基準，單位是 px）
217b:0801  SHR AX,0x1
217b:0803  SHR AX,0x1
217b:0805  SHR AX,0x1           ; AX >>= 3（px → byte，因為 1bpp/plane，8px/byte）
217b:0807  ADD DI,AX             ; DI += X 方向 byte 位移
217b:0809  ADD DI,[0x53de]      ; DI += DAT_31f0_53de（畫面緩衝區基底位移，於 ES:A000 段內）
```

**已驗證**：目標定址公式（`DI` 相對 `ES:0xA000`）為

```
DI = (((param_1*28 + Y基準) & 0xff) * 80) + (((param_2*32 + X基準) >> 3)) + 畫面基底
```

- Y 方向：每個「列格」間距 **28 條掃描線**，每條掃描線在單一 plane 裡佔 **80 bytes**（`0x50`）。
- X 方向：每個「欄格」間距 **32 像素** = 4 bytes/plane。
- **`80 bytes/掃描線 × 350 列 = 28,000 bytes`，這正是 `FUN_217b_0662`（清畫面函式）寫入的位元組數**——直接證據指出這是 **EGA 640×350 16 色模式（BIOS mode 0x10）的單一 plane 大小**，本 segment 所有畫面相關函式（填色/反白/blit）共用同一張 640×350 4-plane 畫布。這比 `docs/formats/graphics.md` 原先只從 `.PIE`/`.SHE` 檔案大小反推「EGA 掃描線拉高 1.75 倍」更進一步：**畫面本身就是 640×350 高解析模式**，不是 320×200。

### 2.3 來源位址計算（`.SHE` 資料 → frame 起點）

```asm
217b:080d  MOV AX,[BP+0xa]      ; AX = param_3（frame 索引）
217b:0810  MOV CX,0x1c0         ; CX = 448
217b:0813  MUL CX                ; DX:AX = param_3 * 448（16-bit 全字寬乘法，無截斷）
217b:0815  MOV SI,AX
217b:081c  ADD SI,[0x5222]      ; SI += DAT_31f0_5222（.SHE 資料 far pointer 的 offset 部分）
217b:0820  MOV BX,0x5222
217b:0823  MOV AX,[BX+0x2]      ; AX = word at 0x5224 = DAT_31f0_5224（far pointer 的 segment 部分）
217b:0826  MOV DS,AX             ; DS = .SHE 資料所在 segment
```

**已驗證**：frame 起點 = `DS:SI`，其中 `DS:0x5222/0x5224` 是一組指向 `.SHE` **記憶體緩衝區**的 far pointer，`SI = far_ptr.offset + frame_index * 448`。

> **448 是記憶體內的 frame 大小，不是檔案內的。** 這個緩衝區在載入時已經被
> `FUN_217b_0adf` 就地水平加倍過（檔案 224 B/frame → 記憶體 448 B/frame）。
> 同一份初始化程式碼把兩個值分開存在兩個變數裡：
> `[0x5226]` = 224（檔案內）、`[0x521a]` = 448（記憶體內）。見 §2.6。

### 2.4 4-plane 迴圈與 Map Mask 切換時機

```asm
217b:0828  MOV CX,0x1c          ; CX = 28（內層列數，每個 plane 要處理 28 列）
217b:082b  MOV DX,0x3c4
217b:082e  MOV AL,0x2
217b:0830  OUT DX,AL             ; Sequencer Index = 2（選中 Map Mask 暫存器）
217b:083c  MOV BH,0x1            ; BH = plane bitmask，從 1 開始（plane 0）
; ---- 外層迴圈起點（每個 plane 一輪）----
217b:083e  MOV AL,BH
217b:0840  MOV DX,0x3c5
217b:0843  OUT DX,AL             ; Map Mask 資料埠 = BH（1/2/4/8）→ 選定要寫哪個 plane
217b:0844  CMP BL,0x0            ; BL = DAT_31f0_5192（反白旗標）
217b:0847  JZ  217b:0855         ; 0 → 直接複製；非 0 → XOR 反白
   217b:0849  LODSW / XOR AX,0xffff / STOSW   ; 反白路徑：讀 2 bytes、反白、寫 2 bytes
   217b:084e  LODSW / XOR AX,0xffff / STOSW   ; 再一次（共 4 bytes/列）
   217b:0853  JMP 217b:0857
   217b:0855  MOVSW ES:DI,SI     ; 正常路徑：直接複製 2 bytes
   217b:0856  MOVSW ES:DI,SI     ; 再一次（共 4 bytes/列）
217b:0857  ADD DI,0x4c           ; DI 淨位移 = 4(MOVSW已走的) + 0x4c = 0x50(80) → 下一條掃描線
217b:085a  ADD SI,0xc            ; SI 淨位移 = 4(LODSW/MOVSW已走的) + 0xc = 0x10(16) → 下一列來源
217b:085d  LOOP 217b:083e        ; CX-- ，28 次跑完才離開內層迴圈
; ---- 內層 28 列跑完，準備下一個 plane ----
217b:085f  SUB DI,0x8c0          ; DI -= 2240 (=28×80)，抵銷內層迴圈的全部位移 → DI 歸零，回到同一個矩形左上角
217b:0863  SUB SI,0x1bc          ; SI -= 444，抵銷內層迴圈位移 448 之後淨移動 +4 bytes → 下一個 plane 的來源起點
217b:0867  MOV CX,0x1c           ; CX 重置 = 28
217b:086a  SHL BH,0x1            ; BH <<= 1（下一個 plane bitmask）
217b:086c  CMP BH,0x8
217b:086f  JLE 217b:083e         ; BH <= 8 就繼續（跑完 1,2,4,8 共 4 輪才離開）
217b:0871  ~ 217b:087c            ; 收尾：Map Mask 還原成 0xF（4 個 plane 全開，一般繪圖預設值）
```

**已驗證的關鍵時機點**：
- **Map Mask 只在「換 plane」時真正改變值**，硬體暫存器的 `OUT` 指令雖然在內層 28 次迴圈裡逐列都執行一次（`217b:0843`），但寫入的都是同一個 `BH` 值——效果上等同「每個 plane 一次」，不是「每列一次」。
- **目的地 `DI` 每個 plane 都繞回同一個矩形起點**（4 個 plane 疊在同一塊畫面記憶體上，靠 Map Mask 決定實際寫哪個 bit-plane，這正是 EGA 硬體 planar 寫入的標準用法）。
- **來源 `SI` 每個 plane 精確往後移動 4 bytes**（不是隨機跳動）——這是解出 `.SHE` 佈局的核心線索，見 §3。

### 2.5 `.SHE` 資料未壓縮（已驗證，但推論範圍要收窄）

`FUN_217b_07cf` 與 `FUN_217b_06f9`（9×9 網格版本，用完全相同的位移常數）都是用
`LODSW`/`MOVSW` **直接**從 `.SHE` 緩衝區讀 word 搬進畫面，
沒有任何解壓縮步驟。`.SHE` 是 raw 4-plane 點陣資料，不是 RLE。

> **⚠ 原斷言「`0a95`／`0b19` 家族與 `.SHE` 路徑無關」已推翻。**
> 正確的說法是：這組函式不在 **blit** 路徑上，但 `0b19` 在 **載入** 路徑上
> （`FUN_1d9f_0a8b` → `FUN_217b_0adf` → `FUN_217b_0b19`）。
> 它做的也不是解壓縮，是**水平像素加倍**——這正是本檔原本把記憶體 frame 大小
> 誤當成檔案格式的原因：只看 blit 路徑，看不到緩衝區在被讀之前已經被改過。
>
> 教訓：「blit 路徑上沒有解包呼叫」只能證明 blit 當下不解包，
> **證明不了緩衝區的內容等於磁碟上的內容**。要問「這塊記憶體是誰填的」，
> 得往回追到載入器。

### 2.6 載入鏈：`.SHE` 在進入 blit 之前先被水平加倍（已驗證）

**初始化時就宣告了兩個不同的 frame 大小**（`disassembly.asm` 約 19682–19775 行）：

```asm
; EGA 分支
1d9f:00bf  MOV word ptr [0x5226],0xe0    ; 224 = 檔案內 frame 大小
1d9f:00fa  MOV AX,[0x5226]
1d9f:0101  SHL AX,0x1
1d9f:0109  MOV [0x521a],AX               ; 448 = 0x1c0 = 記憶體內 frame 大小

; CGA 分支
1d9f:018d  MOV word ptr [0x5226],0x40    ; 64
1d9f:01aa  MOV AX,[0x5226]
1d9f:01b5  MOV [0x521a],AX               ; 64（不加倍）
```

同一段用 `[0x5226]` 乘出各素材檔的大小，兩種模式各自吻合磁碟上的實際檔案大小
（`× 0x66`(102) → EGA 22,848 = `DEMON.SHE`、CGA 6,528 = `DEMON.SHP`；
`× 0x1b`(27) → EGA 6,048 = `CYPHER.SHE`、CGA 1,728 = `CYPHER.SHP`）。
這等於遊戲親口說出檔案格式，是本次訂正最直接的一條證據。

**載入鏈**：

- **`FUN_1d9f_0a8b`**（素材載入器，`decompiled_all.c` 約 8644 行起）：
  EGA 模式（`DAT_31f0_19fb != 0`）下改寫副檔名末字元——
  `C`→`E`（`.PIC`→`.PIE`）、`P` 且前一字元為 `H`→`E`（`.SHP`→`.SHE`）、
  `T` 且前一字元為 `N`→`E`（`.FNT`→`.FNE`）。
  讀檔後比對副檔名是不是 `shE`/`SHE`，命中就呼叫 `FUN_217b_0adf(size)`
  就地把緩衝區加倍。比對用的字串常數在 `DEMON.INT` 檔案位移
  `0x27592`/`0x27596` = `"shE\0SHE\0"`。
  **只有 `SHE` 會被加倍**——`.PIE` 的加倍發生在 blit（`0884`），
  `.FNE` 則完全不加倍，所以 `docs/re/17` 的 16×14 字型結論不受影響。
- **`FUN_217b_0adf`** → **`FUN_217b_0b19`**：由後往前就地展開 N bytes → 2N bytes，
  每 byte 逐 bit 複製兩次。
- **載入呼叫端**（`1990:2e3c`、`138d:3794`、`17c5:0581`）：讀取單位是
  `[0x5226] << 3` = 1,792 = **8 個 224-byte frame**（一隻怪物 8 個動作，
  `MONSTER` 240/8 = 30 隻），檔內 seek 是 `索引 × 1792`；
  目的位址是 `[0x521e] + 槽位 × [0x521a]`（×448），
  正好就是 `07cf` 用 `[0x5222]/[0x5224]` 定址的那塊緩衝區。

**為什麼加倍後 `EGAPlanesRowBlocks` 公式仍成立**：檔案內每列 8 bytes
（4 個 plane 各 2 B），逐 byte 加倍後每列 16 bytes（4 個 plane 各 4 B）——
結構同構，只有 rowBytes 從 2 變 4。所以「檔案當 16×28 解」與
「記憶體當 32×28 解」是同一張圖。

---

## 3. `.SHE` 位元佈局規格（已驗證，spec 級）

### 3.1 Frame 尺寸

**這一節描述的是記憶體內、載入加倍後的 frame。**磁碟檔案是它的一半寬
（16×28、224 bytes、每列 8 bytes），見 §2.6。

| 項目 | 記憶體內的值 | 驗證狀態 |
|---|---|---|
| Frame 寬度 | **32 px**（4 bytes/plane/列） | 已驗證：`217b:07fb SHL AX,0x5`（欄格間距 ×32）+ 內層迴圈每列寫 2×STOSW/MOVSW=4 bytes |
| Frame 高度 | **28 列** | 已驗證：`217b:0828 MOV CX,0x1c`，內層迴圈精確跑 28 次 |
| Plane 數 | 4（標準 EGA） | 已驗證：外層迴圈跑 4 輪（mask 1,2,4,8） |
| Frame stride | **448 bytes**（`32/8 × 28 × 4`） | 已驗證：`217b:0810 MOV CX,0x1c0`；等於 `[0x521a]` = `[0x5226] × 2` |

對應的**檔案格式**是 frame 16×28、224 bytes、`offset = row × 8 + plane × 2 + col`
（`col ∈ [0,2)`），由 `1d9f:00bf` 的 `[0x5226] = 0xe0` 與檔案大小乘法直接證實。

> **這一項本專案錯了兩輪。** 第一輪從 CGA（當時也錯，見下）的 16×32 套
> 「高度 ×1.75」推出 16×56，八種 plane 佈局假設全是雜訊；
> 第二輪讀 blit 常數改成 32×28，plane 佈局對了、圖案出來了，於是標「已驗證」，
> 但那是把兩個相鄰 frame 的左右半邊錯位疊在一起。
> 真正的檔案 frame 是 16×28。
>
> 三個候選（16×56 / 32×28 / 16×28+載入加倍）在**磁碟位元組數上完全等價**，
> 前兩者的總像素數都是 896。算術從頭到尾分不出來，
> 分得出來的是兩件事：**遊戲初始化時自己宣告的 `[0x5226]`**，
> 以及**「每個 frame 必須自成一體」的肉眼判準**（見 §4）。

### 3.2 Byte 佈局（frame 內部，位移相對 frame 起點 0）

**排列規則**：frame 內部**逐列（row-major）排列**，每一列裡 **4 個 plane 各自連續存放一個區塊**（不是逐 byte 交錯，也不是整個 frame 先存完一個 plane 再存下一個）。這個結構在檔案與記憶體兩邊同構，只有每個 plane 區塊的 byte 數不同（檔案 2、記憶體 4）。

以下是**記憶體內**的版本；讀檔案時把 `16` 換成 `8`、`4` 換成 `2`、`col ∈ [0,2)`。

```
frame_offset(plane, row, col) = row * 16 + plane * 4 + col

其中：
  row  ∈ [0, 28)   -- 第幾條掃描線（0 = 最上面）
  plane ∈ [0, 4)    -- 第幾個 bit-plane（0 = Map Mask bit0，對應 EGA_PALETTE 色碼 bit0）
  col  ∈ [0, 4)     -- 該 plane 該列的第幾個 byte（每 byte 8 個像素，col=0 是最左 8px）
```

每一列 16 bytes 的排列具體長這樣：

```
+-------------------+-------------------+-------------------+-------------------+
| plane0 byte0..3    | plane1 byte0..3    | plane2 byte0..3    | plane3 byte0..3    |  <- row 0（offset 0..15）
+-------------------+-------------------+-------------------+-------------------+
| plane0 byte0..3    | plane1 byte0..3    | plane2 byte0..3    | plane3 byte0..3    |  <- row 1（offset 16..31）
+-------------------+-------------------+-------------------+-------------------+
...（重複 28 次，row 27 在 offset 432..447）
```

**bit 順序**：沿用本專案已驗證的 EGA 慣例（`.PIE` 全螢幕圖、`internal/assets/gfx/ega.go` 既有的 `DecodeEGAPlanar`）——**MSB-first**，每個 plane byte 的 bit7 是該 8px 區段最左邊的像素，bit0 是最右邊。這點本身是 EGA 硬體規格（Map Mask 寫入時 bit 對應畫面像素的順序是硬體固定的，不是遊戲自訂），`217b:07cf` 用 `MOVSW`/`STOSW` 整字搬移不改動 bit 順序，因此沿用既有驗證結果，不需要重新反推。

**像素色碼組成**：座標 `(row, x)`（`x = col*8 + bit`，`bit=7..0` 對應像素由左至右）的顏色索引：

```
color_index = plane0_bit | (plane1_bit << 1) | (plane2_bit << 2) | (plane3_bit << 3)
```

跟 `EGAPalette`（`internal/assets/gfx/palette.go`）的既有映射一致（bit0=Blue, bit1=Green, bit2=Red, bit3=Intensity）。

### 3.3 對應到 `internal/assets/gfx/ega.go` 的 API

`EGAPlanesRowBlocks` layout 對應上面 §3.2 的公式。**解磁碟檔案時用 16×28**：

```go
frames, err := gfx.DecodeEGASpriteSheet(data, 16, 28, gfx.EGAPlanesRowBlocks)
```

（`32, 28` 只適用於「已經被 `FUN_217b_0adf` 加倍過的記憶體緩衝區」，
重寫版不需要複製那一步。）

---

## 4. PNG 肉眼比對結果

依**檔案** 16×28（`.SHE`）／16×16（`.SHP`）解出的結果：

| 檔案 | frame 數 | 肉眼比對結果 |
|---|---|---|
| `MONSTER.SHE` | 240 | **清楚**：武士、骷髏、各式怪物，色彩合理（綠/紫/白/紅橙，符合 16 色 EGA 調色盤）。frame 0–7 是同一個戰士的 8 個姿勢 |
| `COMBAT.SHE` | 44 | **清楚**：戰鬥背景用的網底/紋理圖樣，以及戰士/骷髏持武器的圖形 |
| `DEMON.SHE` | 102 | **清楚**：地形圖塊集（城門/建築結構、樹木、地面），見下方 |
| `WINTER.SHE` | 102 | **清楚**：`DEMON` 的雪地版，逐格對應同一地形 |
| `SHIP.SHE` | 32 | **清楚**：船體/帆具圖形 |
| `CYPHER.SHE` | 27 | **清楚**：一排符文/幾何符號圖示，對應 "God Runes / Fire Runes..." 符文 UI |

**每個 frame 都是一個完整、自成一體的圖形**——這是判斷尺寸解對沒有的關鍵判準。
CGA 側依 16×16 解出的圖案、順序、空白格位置與 EGA 版**一格不差**。

> **⚠ 本節原本記錄的觀察「一個 frame 內裝 2 個（CGA）／4 個（EGA）小圖，
> 是美術資源打包慣例」已刪除——那是解碼錯誤造成的假象。**
>
> 用 16×32 解 `.SHP`，會把上下兩個 16×16 frame 疊進一格；
> 用 32×28 解 `.SHE`，會把兩個相鄰 frame 的左右半邊錯位疊成 2×2。
> 當時把這個徵狀解釋成「美術慣例」，等於替錯誤的尺寸找了一個合理化的理由，
> 反而讓「已驗證」標籤撐了下來。
>
> **判準記下來**：sprite 解碼後若出現「一格裡有 2 個或 4 個並排／堆疊的小圖」，
> 第一個該懷疑的永遠是 frame 尺寸，不是美術打包。

### 4.1 `DEMON`／`WINTER` 是地形圖塊集（已驗證）

這兩個檔的 102 格不是角色 sprite，是**地形圖塊集**：與 `FILES.DAT` offset
`0x040` 的可通行性表（tile 0–100，共 101 項，見 `docs/re/22`）對應，
最後一格（索引 101）未被該表涵蓋。`WINTER` 是 `DEMON` 的雪地版，同索引同地形。

---

## 5. 已驗證 vs 假設 一覽

| 結論 | 狀態 | 證據 |
|---|---|---|
| `.SHE` **檔案內** frame = 16×28、224 bytes（六個檔一律如此） | **已驗證** | `1d9f:00bf MOV word ptr [0x5226],0xe0` + 同段用 `[0x5226] × 0x66`／`× 0x1b` 乘出的檔案大小與磁碟實際大小吻合 + 逐格肉眼比對 |
| `.SHE` **記憶體內** frame = 32×28、448 bytes | **已驗證** | `1d9f:0101 SHL AX,1` → `[0x521a]`=448；`217b:0810 MOV CX,0x1c0`（blit stride）；`217b:07fb`(×32 欄距) + `217b:0828`(CX=28 列數) |
| 載入時就地水平加倍（檔案 → 記憶體） | **已驗證** | `FUN_1d9f_0a8b` 比對 `"shE"/"SHE"`（`DEMON.INT` 位移 `0x27592`/`0x27596`）後呼叫 `FUN_217b_0adf`；`0b19` 的逐 bit 複製迴圈 |
| 4-plane 排列 = 逐列、每列 4 個 plane 各自連續（`EGAPlanesRowBlocks`） | **已驗證** | `217b:085a`(SI 淨位移 16/列) + `217b:0863`(換 plane 淨位移 4)，逐指令核對且與 `06f9` 交叉驗證同一組常數；檔案側 rowBytes 減半後結構同構 |
| Map Mask 切換時機 = 每個 plane 一次（4 輪：1,2,4,8） | **已驗證** | `217b:086a SHL BH,1` / `217b:086c CMP BH,8` 迴圈結構 |
| bit 順序 = MSB-first | **已驗證（沿用既有結果）** | EGA 硬體規格 + `MOVSW`/`STOSW` 不改動 bit 順序，與本專案已驗證的 `.PIE` 解碼慣例一致 |
| 畫面模式 = EGA 640×350（16 色，mode 0x10） | **已驗證** | `217b:0662` 清畫面寫入 28,000 bytes = 640×350÷8，且跟 `07cf`/`04aa` 等函式的 `0x50`(80 bytes/列) 列距完全吻合 |
| `.SHE` 資料未壓縮（raw 4-plane 點陣） | **已驗證** | `07cf`/`06f9` 全程 `LODSW`/`MOVSW` 直接搬移。⚠ 注意這只證明 blit 當下不解包，不證明緩衝區內容等於磁碟內容（見 §2.5） |
| `.PIE` 顯示尺寸 = 288×252（檔內 144×252） | **已驗證** | `FUN_217b_0884` 每列呼叫 `0a95` 九次，每次 2 source byte → 4 byte |
| `MONSTER` 的 8 frame/怪物分組（240/8 = 30 隻） | **已驗證** | 載入呼叫端 `1990:2e3c`／`138d:3794`／`17c5:0581` 的讀取單位 `[0x5226] << 3` = 1792 + 肉眼確認 frame 0–7 是同一個戰士的 8 個姿勢 |
| `DEMON`/`WINTER` 102 格 = 地形圖塊集，`WINTER` 是雪地版 | **已驗證** | 逐格肉眼比對 + 對應 `FILES.DAT` `0x040` 可通行性表 tile 0–100 |
| `217b:0344`/`0413`（`*0x40`=64 bytes 單位的另一種 blit）用途 | **假設** | 結構清楚但未深入其上層呼叫鏈。注意 64 剛好等於 CGA 的 frame 大小，值得從這個角度再查一次 |
| `DEMON`/`WINTER` 索引 101 | **強證據：runtime 未使用的額外水紋 frame** | 四套 atlas 同構、全地圖 inventory 無 101、三類動態 tile 寫入也不產生 101；見 `docs/re/118` |

---

## 6. 給引擎渲染層的建議

- `internal/assets/gfx/ega.go` 的 `EGAPlanesRowBlocks` layout 可直接串進
  Ebiten 渲染管線，六個 `.SHE` 檔一律用
  `DecodeEGASpriteSheet(data, 16, 28, EGAPlanesRowBlocks)`。
  CGA 側一律 16×16。
- **不要複製原版的載入時加倍**（`FUN_217b_0adf`）。那是 8086 為了讓 blit
  退化成單純 `MOVSW` 而做的預處理；重寫版直接以檔案原尺寸解碼，
  在繪製時處理寬度縮放即可。
- `DecodeEGASpriteSheetGlobalPlanes` 與 `EGAPlanesSequential`／
  `EGAPlanesRowInterleaved` 這幾個舊假設**已確認是錯的**，新程式碼不要再用。
- 畫面模式已確認是 **EGA 640×350 16 色**（不是 320×200）。
  EGA 素材一律「檔案存半寬、顯示時寬度 ×2、高度 ×1.75」。
