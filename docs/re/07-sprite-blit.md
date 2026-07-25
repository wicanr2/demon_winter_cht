# `217b` 段：EGA 精靈 blit 與 `.SHE` 精靈圖格式（2026-07-25）

本檔攻的是 PLAN 標記投報最高的一塊：`217b` 段（24 個函式，先前只碰過 1 個）的
EGA sprite blit 邏輯，目標是解出 `.SHE` 精靈圖的真實位元佈局。

**結論先講**：`.SHE` 的 4-plane 佈局**已解出並肉眼驗證**——直接反組譯
`FUN_217b_07cf`（單一 sprite blit 常式）的原始指令（不是 decompile，decompile
對這段的指標單位有歧義，已依任務指示改讀 `disassembly.asm`）得到位元組精確的
定址公式，套用後 `MONSTER.SHE`／`COMBAT.SHE`／`DEMON.SHE`／`WINTER.SHE`／
`SHIP.SHE` 全部解出乾淨可辨識的怪物/場景圖案，`CYPHER.SHE`（不同 frame 大小,
未被 `07cf` 直接處理)用同一條公式外推也解出乾淨的符文圖示。

## 讀法／驗證方法

依 `rulebook/62`（靜態溯源）與任務指示：不重試 `docs/formats/graphics.md`
已排除的 8 種佈局假設，改成**逐指令讀 `FUN_217b_07cf` 的 disassembly**，
把指標的「每次迭代淨位移量」跟「迴圈跑幾次」直接讀出來，反推來源資料的
真實排列，寫成解碼器後再用 PNG 肉眼比對驗證（`rulebook/64`：已有解碼器
候選 + 已知 CGA 版當 oracle，不必再靠猜）。

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
| `217b:0884` | 135 | 迴圈呼叫 `FUN_217b_0a95`(252×9次)，外層依 plane mask 1,2,4,8 跑 4 輪 | 已驗證迴圈結構；`0a95` 的位元解包語意未深究，[推測]跟壓縮/轉場特效有關，不影響 `.SHE` 判讀（見 §2.5） |
| `217b:090b` | 113 | 用 GC Bit Mask 暫存器寫**單一 byte**（`1 << (X&7 ^ 7)` 遮罩），像是畫單一 8px 寬直條/游標線 | 已驗證迴圈與遮罩算式；用途 [推測] |
| `217b:097c` | 261 | 14 列、逐列 3-byte 且帶**次 byte 位元位移**（`sVar5`）的小圖 blit，來源 `DAT_31f0_5488` + `param_3*0x1c` | 已驗證結構；這是「非 byte-aligned X 座標」的小圖繪製（跟 `07cf` 的 byte-aligned 版本互補），[推測]用於游標/選取箭頭等允許任意 X 位置的小圖 |
| `217b:0a81` | 20 | INT 10h trampoline（同 `000d`/`0338`） | 已驗證 |
| `217b:0a95` | 74 | `__cdecl16near` 內部函式：讀 `SI` 指標 byte、4 次 shift 累加、依累加值 `1`/`3` continue、`4` 結束，寫 `DI` 指標 word | 已驗證是移位型位元解包/展開迴圈；**不在 `.SHE` sprite 讀取路徑上**（`07cf`/`06f9` 都是直接 `LODSW`/`MOVSW`，沒呼叫這條）——直接證據支持「`.SHE` sprite 資料未壓縮」，見 §2.5 |
| `217b:0adf` | 58 | 呼叫 `FUN_217b_0b19` `(param_1 & 0x7fff)` 次的迴圈包裝 | 已驗證 |
| `217b:0b19` | 42 | `__cdecl16near`：跟 `0a95`同家族但更簡化的位元展開（4 次 shift，兩輪固定跑完寫一個 word） | 已驗證結構；同上，不在 sprite 路徑上 |

**小結**：24 個函式可分 4 群——
1. **sprite blit 家族**（`07cf`、`06f9`）：本文件核心，見 §2。
2. **實心/圖樣填色、反白家族**（`0033`、`0120`、`02b7`、`04aa`、`057b`、`0697`、`090b`）：UI 方框、選取高亮、清區域，共用同一個「640 寬 4-plane 畫面」定址模型，跟 sprite blit 是同一張畫布，但不讀 `.SHE`。
3. **一般記憶體緩衝 blit 家族**（`0344`、`0413`、`025a`、`01d5`）：不呼叫 `OUT`，寫的是系統 RAM 裡的合成/快取緩衝，不是直接寫 EGA 硬體；`0344`/`0413` 用的 frame 單位是 `0x40`(64 bytes)，明顯不是 `.SHE` 用的 448 bytes，可能對應另一種更小的圖示資源（超出本次任務範圍，留待後續）。
4. **位元解包/展開家族**（`0a95`、`0adf`、`0b19`，被 `0884` 呼叫）+ **INT 10h/21h trampoline**（`000d`、`0338`、`0a81`、`001c`）：與 `.SHE` 讀取路徑無關（見上表 `0a95` 說明），不深入。

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

**已驗證**：frame 起點 = `DS:SI`，其中 `DS:0x5222/0x5224` 是一組指向已載入 `.SHE` 檔案資料的 far pointer，`SI = far_ptr.offset + frame_index * 448`。**這再次確認 448 bytes/frame 是對的**（跟 `docs/formats/graphics.md` §4.1 用檔案大小整除反推的結論一致，這裡是從程式碼常數直接驗證）。

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

### 2.5 `.SHE` 資料未壓縮（已驗證）

`FUN_217b_07cf` 與 `FUN_217b_06f9`（9×9 網格版本，用完全相同的位移常數）都是用 `LODSW`/`MOVSW` **直接**從 `.SHE` 緩衝區讀 word 搬進畫面，全程沒有呼叫 `217b` 段裡唯一像壓縮/解包邏輯的 `FUN_217b_0a95`/`FUN_217b_0b19` 家族（那組函式只被 `FUN_217b_0884`——一個跟 sprite blit 完全獨立的呼叫鏈——用到）。**這排除了 `docs/formats/graphics.md` §4.2 列的「可能原因 1：sprite 資料可能經過 RLE 或其他壓縮」**：`.SHE` 是 raw 4-plane 點陣資料，frame 邊界與內部佈局都不涉及解壓縮步驟。

---

## 3. `.SHE` 位元佈局規格（已驗證，spec 級）

### 3.1 Frame 尺寸

| 項目 | 值 | 驗證狀態 |
|---|---|---|
| Frame 寬度 | **32 px**（4 bytes/plane/列） | 已驗證：`217b:07fb SHL AX,0x5`（欄格間距 ×32）+ 內層迴圈每列寫 2×STOSW/MOVSW=4 bytes |
| Frame 高度 | **28 列** | 已驗證：`217b:0828 MOV CX,0x1c`，內層迴圈精確跑 28 次 |
| Plane 數 | 4（標準 EGA） | 已驗證：外層迴圈跑 4 輪（mask 1,2,4,8） |
| Frame 大小 | **448 bytes**（`32/8 × 28 × 4`） | 已驗證：`217b:0810 MOV CX,0x1c0` 直接對應 frame stride 常數 |

這推翻了 `docs/formats/graphics.md` §4.2 原先「16×56（高度縮放 1.75×）」的假設——**總像素數兩者都是 896（16×56=32×28），檔案大小整除完全無法區分這兩種切法**，這正是本專案第三次「除法對得上、方向/佈局猜錯」的案例（前兩次是 CGA 全螢幕圖偶奇交錯、CGA sprite 寬高對調）。唯一能分辨的方法就是本文件採用的：**直接讀 blit 常式的迴圈次數與位移常數**。

### 3.2 Byte 佈局（frame 內部，file offset 相對 frame 起點 0）

**排列規則**：frame 內部**逐列（row-major）排列**，每一列裡 **4 個 plane 各自連續存放一個 4-byte 區塊**（不是逐 byte 交錯，也不是整個 frame 先存完一個 plane 再存下一個）。

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

### 3.3 對應到 `internal/assets/gfx/ega.go` 的新 API

新增 `EGAPlanesRowBlocks` layout（對應上面 §3.2 公式），呼叫方式：

```go
frames, err := gfx.DecodeEGASpriteSheet(data, 32, 28, gfx.EGAPlanesRowBlocks)
```

---

## 4. PNG 肉眼比對結果

用 `internal/assets/gfx/gfx_test.go` 的 `TestDecodeEGASprites` 產生下列輸出（`workplace/dump/gfx/`，不入版控）：

| 檔案 | frame 數 | 輸出 | 肉眼比對結果 |
|---|---|---|---|
| `MONSTER.SHE` | 120 | `ega-sprites-MONSTER.SHE.png` | **清楚**：武士、骷髏、各式怪物剪影，跟已驗證的 `cga-sprites-MONSTER.SHP.png` 同一批造型、色彩合理（綠/紫/白/紅橙，符合 16 色 EGA 調色盤），完全不是雜訊 |
| `COMBAT.SHE` | 22 | `ega-sprites-COMBAT.SHE.png` | **清楚**：前段幾格是戰鬥背景用的網底/紋理圖樣（合理的美術素材，不是解碼錯誤——見下方放大圖），後段是清楚的戰士/骷髏持武器剪影 |
| `DEMON.SHE` | 51 | `ega-sprites-DEMON.SHE.png` | **清楚**：可辨識城門/建築結構、樹木、道具圖示（如綠色寶石圖案） |
| `WINTER.SHE` | 51 | `ega-sprites-WINTER.SHE.png` | **清楚**：場景/建築剪影與網底紋理，風格與 `DEMON.SHE` 一致 |
| `SHIP.SHE` | 16 | `ega-sprites-SHIP.SHE.png` | **清楚**：幾何船體/帆具圖示 |
| `CYPHER.SHE`（frame 224B，非 448B，見 §5 假設說明） | 27 | `ega-sprites-CYPHER.SHE.png` | **清楚**：一排符文/幾何符號圖示，跟已驗證的 `cga-sprites-CYPHER.SHP.png`（"God Runes/Fire Runes..." 符文 UI）風格一致 |

**單一 frame 放大比對**（`TestDecodeCGAMonsterFrame0Zoom` + `TestDecodeEGASprites` 各自輸出的 `zoom-*-frame0.png`，8 倍放大）：

- CGA `MONSTER.SHP` frame 0（`zoom-cga-MONSTER.SHP-frame0-correct.png`，16×32，已驗證正確的尺寸）：清楚看到**兩個上下堆疊、拿劍的武士小圖案**（16×16 各一個）。
- EGA `MONSTER.SHE` frame 0（`zoom-ega-MONSTER.SHE-frame0.png`，32×28）：清楚看到**四個分佈成 2×2 的武士/骷髏小圖案**，個個輪廓完整、色彩不糊、無雜訊斑點。

**觀察（非本次任務核心，留給後續 agent）**：CGA 版每個「frame」單位本身就已經包含 2 個堆疊的小圖（16×16），不是單一怪物的完整立繪；EGA 版對應變成 2×2＝4 個小圖。由於這個「一個 frame 裝多個小圖」的現象在 CGA（已驗證多年不會錯的格式）跟 EGA 都一致存在，這是**美術資源打包慣例本身如此**（可能是同一個怪物的不同朝向/姿勢，或多隻同類怪物的隊列圖示），不是解碼佈局錯誤——`.SHE` 的原始位元佈局（本文件的交付範圍）已經解對，這個「frame 內部語意細分」是遊戲引擎更高層的邏輯，建議後續 agent 檢查呼叫 `07cf`/`06f9` 的上層函式（`param_3` 怎麼算出來的）來確認。

---

## 5. 已驗證 vs 假設 一覽

| 結論 | 狀態 | 證據 |
|---|---|---|
| `.SHE` frame = 448 bytes（大 sprite）/ 224 bytes（`CYPHER.SHE`） | **已驗證** | `217b:0810 MOV CX,0x1c0` 直接對應（大 sprite）；`CYPHER.SHE` 檔案大小整除仍成立 |
| Frame 尺寸 = 32×28（大 sprite） | **已驗證** | `217b:07fb`（×32 欄距）+ `217b:0828`（CX=28 列數），逐指令核對 |
| 4-plane 排列 = 逐列、每列 4 個 plane 各自連續 4 bytes（`EGAPlanesRowBlocks`） | **已驗證** | `217b:085a`(SI 淨位移 16/列) + `217b:0863`(換 plane 淨位移 4)，逐指令核對且與 `06f9` 交叉驗證同一組常數 |
| Map Mask 切換時機 = 每個 plane 一次（4 輪：1,2,4,8） | **已驗證** | `217b:086a SHL BH,1` / `217b:086c CMP BH,8` 迴圈結構 |
| bit 順序 = MSB-first | **已驗證（沿用既有結果）** | EGA 硬體規格 + `MOVSW`/`STOSW` 不改動 bit 順序，與本專案已驗證的 `.PIE` 解碼慣例一致 |
| 畫面模式 = EGA 640×350（16 色，mode 0x10） | **已驗證** | `217b:0662` 清畫面寫入 28,000 bytes = 640×350÷8，且跟 `07cf`/`04aa` 等函式的 `0x50`(80 bytes/列) 列距完全吻合 |
| `.SHE` 資料未壓縮（raw 4-plane 點陣） | **已驗證** | `07cf`/`06f9` 全程 `LODSW`/`MOVSW` 直接搬移，未呼叫段內唯一的位元解包函式家族(`0a95`/`0b19`) |
| `CYPHER.SHE` frame = 16×28（同一條 `EGAPlanesRowBlocks` 公式外推） | **假設**（視覺已驗證乾淨，但未被任何已讀出常數的函式直接證實） | `07cf` 的 448-byte stride 硬編碼不適用於 224-byte frame 的 `CYPHER.SHE`；16×28 是套用同一佈局公式反推出的候選尺寸(16/8×28×4=224 整除)，解出的符文圖案清楚，但沒有找到專門處理 `CYPHER.SHE` 的函式來逐指令核對 |
| `217b:0344`/`0413`（`*0x40`=64 bytes 單位的另一種 blit）用途 | **假設** | 結構清楚但未深入其上層呼叫鏈，可能是另一種較小的圖示資源(非 `.SHE`)，超出本次任務範圍 |
| 「一個 frame 裝 2(CGA)/4(EGA) 個小圖」的遊戲語意 | **未解**，已誠實記錄於 §4 | 只確認現象存在（且 CGA/EGA 一致），未追出上層邏輯 |

---

## 6. 對 `docs/formats/graphics.md` 的修正建議

（依任務邊界：不直接修改 `docs/formats/graphics.md`，修正建議寫在這裡，留給下一輪或維護者手動併入）

1. **§4「EGA 精靈圖 `.SHE`：格式部分未解」整節可以升級為「已驗證」**，用本文件 §3 的公式取代原本列出的 8 種已排除假設表；保留那張表當「刻意排除的候選」歷史記錄即可，不用刪。
2. **frame 尺寸從「16×56」改為「32×28」**——原文件的 1.75× 高度縮放假設（來自 `.PIE` 場景圖的驗證結果）**不適用於 `.SHE` 精靈圖**；`.SHE` 的縮放模式其實是「寬度 ×2（16→32，對應 320→640 解析度加倍）、高度 ×0.875（32→28）」，不是單純套用 `.PIE` 的 1.75× 高度公式。這點值得在文件裡明確拆開：**`.PIE`（全螢幕/人像框）跟 `.SHE`（精靈圖）的 EGA 縮放規則不一樣**，不能套同一條 3.5× 分解公式。
3. **§4.2 的「可能原因 1：sprite 資料可能經過 RLE 或其他壳縮」可以排除**，改標「已排除：`07cf`/`06f9` 皆直接 `LODSW`/`MOVSW`，未呼叫段內的位元解包函式」。
4. **新增一節「一個 frame 內部其實裝多個小圖（CGA 2 個、EGA 4 個）」**，把本文件 §4 的觀察收進去，並標記為留給後續 agent 的 TODO（上層 `param_3` frame index 怎麼選、是否對應怪物數量/朝向）。
5. `CYPHER.SHE` 16×28 的結論建議標「假設（視覺已驗證，未經反組譯逐指令核對）」，跟其餘 5 個檔案的「已驗證」分開列，避免之後被誤讀成同等級證據。
6. **總結表（§7）的 EGA `.SHE` 狀態欄可以從「🟡 部分」改成「✅ 已驗證（`CYPHER.SHE` 除外標假設）」**。

---

## 7. 給引擎渲染層的建議

- `internal/assets/gfx/ega.go` 新增的 `EGAPlanesRowBlocks` layout + `DecodeEGASpriteSheet(data, 32, 28, EGAPlanesRowBlocks)` 可以直接串進 Ebiten 渲染管線，用於 `MONSTER.SHE`／`COMBAT.SHE`／`DEMON.SHE`／`WINTER.SHE`／`SHIP.SHE`。
- `CYPHER.SHE` 用 `DecodeEGASpriteSheet(data, 16, 28, EGAPlanesRowBlocks)`，但因為是外推假設，串進引擎前建議先找出處理它的實際 blit 函式（`217b` 段之外也要查，`097c`/`0344` 家族值得優先看）逐指令核對，或至少多找幾張截圖交叉比對。
- `DecodeEGASpriteSheetGlobalPlanes` 與 `EGAPlanesSequential`/`EGAPlanesRowInterleaved` 這兩個舊假設**已確認是錯的**（見 `ega.go` 內已更正的文件註解），新程式碼不要再用。
- 畫面模式已確認是 **EGA 640×350 16 色**（不是 320×200），渲染層若要模擬原版畫面比例，這是需要的關鍵參數。
