# 字型格式 `.FNT`/`.FNE`（DEMON.INT 反組譯，2026-07-25）

> **結論先講：兩種字型都已解出並肉眼驗證。** `ASC.FNT`（CGA）是 8×8、
> **packed 2bpp**（每 byte 4 個像素、每列 2 bytes、來源線性、**無 header**），
> 檔案含**兩個 96 字 bank**：bank0 一般（白字黑底）、bank1 反白（黑字亮洋紅底）；
> `ASC.FNE`/`GOT.FNE`（EGA）是 16×14、1bpp。這推翻了 `docs/formats/graphics.md`
> §6 原本「1-byte header + 256×8×12 1bpp」的猜測，**也推翻了本文件 2026-07-25
> 第一版對 CGA 的判讀**（見 §2.3「已作廢的斷言」）。GOT.FNE 解出來是
> 花體(blackletter)風格，與 `workplace/dosbox/shots/smoke-01.png`、
> `03-ega-ingame.png` 的花體 UI 文字一致。

驗證方法與本專案既有格式解密（`.PIC`/`.PIE`/`.SHP`/`.SHE`，見
`docs/formats/graphics.md`、`docs/re/07-sprite-blit.md`）同一套：依假設寫
解碼器 → 輸出 PNG → 肉眼比對。**這次額外做的是「先讀繪字函式的位址算式，
再驗證」**（`rulebook/62`），而不是純靠試佈局，因為字型格式的自由度比
sprite 更高（沒有「frame 邊界」這種可以先用檔案大小整除卡住的錨點）。

---

## 0. 快速結論表

| 項目 | CGA(`.FNT`) | EGA(`.FNE`) |
|---|---|---|
| 字元尺寸 | **8 寬 × 8 高** | **16 寬 × 14 高** |
| 色深/佈局 | **packed 2bpp**（每 byte 4 個像素、byte0=左 4px、byte1=右 4px，就是 CGA mode 4 framebuffer 的原生格式） | 1bpp（前景/背景色由呼叫端指定，字型檔本身無顏色） |
| 每字 bytes | 16（8 rows × 2 bytes/row） | 28（14 rows × 2 bytes/row） |
| bit 順序 | MSB-first（每 byte 最左像素在 bit7–6） | MSB-first |
| 字表起點 | ASCII `0x20`（空白） | ASCII `0x20`（空白） |
| 檔頭 | **無**。3073 = 3072 資料 + 1 個結尾 `0x1A`（DOS EOF 標記） | 無 |
| 涵蓋字元數 | 96（`0x20`–`0x7F`）× **2 個 bank**＝192 個 glyph（`3072/16`）。bank0＝一般、bank1＝反白（見 §4） | 96（`2688/28`，恰為 `0x20`–`0x7F` 全部可印 ASCII，無餘數） |
| 硬體常數交叉驗證 | `FUN_1d9f_0008`：`DAT_53ae=8, DAT_53b0=8, DAT_53b2=0xc00(3072)` | `FUN_1d9f_0008`：`DAT_53ae=0x10(16), DAT_53b0=0xe(14), DAT_53b2=0xa80(2688)` |

CGA/EGA 的判別旗標是全域變數 **`DAT_31f0_19fb`**：`0`＝CGA、非 0＝EGA/圖形卡，
在開機階段用 `INT 10h` 查詢後寫入（見 §5）。

---

## 1. 繪字函式：`FUN_1d9f_0eeb`（`1d9f:0eeb`，149 bytes）

這是任務單指定「泛用選單元件 `FUN_2cdc_033d` 畫選單文字」往回追的目標——
`FUN_2cdc_033d` 印字實際上是呼叫 `FUN_1d9f_1361(string_offset)`（字串印出
函式，`1d9f:1361`，285 bytes），`FUN_1d9f_1361` 逐字元呼叫
**`FUN_1d9f_0eeb(row, col, char_code)`**（`disassembly.asm` `1d9f:1361`
內第 28 行 `FUN_1d9f_0eeb(*(undefined2*)0x4c7c, *(undefined2*)0x4c82, local_6)`，
`local_6` 是目前字元的無號字元碼 0–255）——這就是「單一字元畫到螢幕」的
核心函式，已驗證。

### 1.1 CGA/EGA 分派 `[已驗證，逐指令核對過 raw asm，非 decompile]`

```asm
1d9f:0eee  CMP word ptr [0x19fb],0x0
1d9f:0ef3  JZ   0x33(offset,近跳轉)     ; DAT_31f0_19fb==0 → CGA 路徑
1d9f:0ef5  ...                          ; 否則(fall-through) → EGA 路徑
```

（用 objdump 對原始位元組重新反組譯過，排除 Ghidra 對近跳轉顯示位址的
已知混淆，見 `docs/re/00-ghidra-setup.md` 第 6 條踩雷。）

- **CGA 路徑**（`0x19fb==0`，`1d9f:0f1e` 起）`[已驗證，逐指令核對 disassembly.asm]`：

  ```asm
  1d9f:0f1e  MOV AX,[0x53b2]          ; 3072（開機常數＝字型資料長度）
  1d9f:0f21  MOV DX,[0x53b4]          ; 0（高位字）
  1d9f:0f25  MOV CX,0x2
  1d9f:0f28  MOV BX,0x0
  1d9f:0f2b  CALLF 0x3000:016a        ; 32-bit 除法 helper → 3072/2 = 1536
  1d9f:0f32  MOV AX,[0x1ff0]          ; 反白旗標（0 或 1）
  …          (32×32 乘法展開)          ; 1536 * flag  → bank 起始位移
  1d9f:0f4e  MOV AX,[BP+0xa]          ; 字元碼
  1d9f:0f51  MOV CX,0x4 / SHL AX,CL   ; char * 16       → 每字 16 bytes
  1d9f:0f56  ADD AX,0xfe00            ; -0x200 = -(0x20*16) → 字表起於 0x20
  1d9f:0f60  ADD AX,CX / ADC DX,BX    ; 合成來源位移
  1d9f:0f66  MOV AX,0x140 / IMUL [BP+6]   ; 字元列 × 320 bytes
  1d9f:0f6c  MOV CX,[BP+8] / SHL CX,1     ; 字元欄 × 2 bytes
  1d9f:0f74  CALLF 0x2000:1a0a        ; = 217b:025a（0x21a0a）
  ```

  三件事一次卡死：**每字 16 bytes**、**字表從 `0x20` 開始**、
  **檔案有兩個 1536-byte bank，由 `[0x1ff0]` 選**（§4）。
  目標位址算式 `列*0x140 + 欄*2` 也順帶證實字元格是 **8 px 寬**
  （2 bytes × 4 px/byte）、**8 條掃描線高**（320 bytes ÷ 80 bytes/field row
  = 4 個 field row = 8 條實際掃描線）。

  > `0x3000:016a` 這支 helper 的本體落在 Ghidra 未分析的間隙（見 §7），
  > 但回傳值被兩邊夾死：算術上 `3072/2 = 1536`，而檔案實際 dump 出來就是
  > 「兩組各 96 字、間距 1536 bytes」的排列（§6），互相印證。
- **EGA 路徑**（`0x19fb!=0`）：
  ```c
  FUN_1d9f_0eb1(param_1 * DAT_53b0, param_2 * DAT_53ae, param_3 - 0x20);
  ```
  `param_3 - 0x20`（`addw $0xffe0,...`）同樣已用 objdump 核對，確認字表
  同樣從 `0x20` 開始。

---

## 2. CGA 字型繪製：`FUN_217b_025a`（`217b:025a`，93 bytes）

完整原始指令（`disassembly.asm` 24941–24985 行）：

```asm
217b:025a  PUSH BP / MOV BP,SP / PUSH AX,SI,DI,ES,DS
217b:0262  MOV AX,0xb800 / MOV ES,AX     ; 目標 = CGA framebuffer
217b:0267  MOV DI,[BP+0x6]               ; arg1 = framebuffer 位移
217b:026a  MOV AX,0x31f0 / MOV DS,AX
217b:026f  MOV SI,[0x5488]               ; 字型緩衝區 far pointer（offset）
217b:0273  MOV BX,0x5488 / MOV AX,[BX+2] / MOV DS,AX   ; （segment）
217b:027b  ADD SI,[BP+0x8]               ; arg2 = glyph 位移
217b:027e  MOV CX,0x4
217b:0281  PUSH DI / PUSH SI
217b:0283  MOVSW                          ; ← 每次只搬 2 bytes
217b:0284  ADD DI,0x4e                    ;   DI 淨進 0x50 = 80 bytes（一條掃描線）
217b:0287  ADD SI,0x2                     ;   SI 淨進 4 bytes
217b:028a  LOOP 217b:0283
217b:028c  MOV CX,0x4
217b:028f  POP SI / POP DI
217b:0291  ADD SI,0x2                     ; 來源回到起點 +2
217b:0294  CMP DI,0x1fff / JG 217b:02a0
217b:029a  ADD DI,0x2000                  ; 切到奇數掃描線 bank
217b:029e  JMP 217b:02a7
217b:02a0  ADD DI,0x50 / SUB DI,0x2000    ; （DI 已在奇 bank 時的等價換算）
217b:02a7  MOVSW / ADD DI,0x4e / ADD SI,0x2 / LOOP 217b:02a7
217b:02b0  POP DS,ES,DI,SI,AX,BP / RETF
```

### 2.1 來源位移 → 視覺列的對應 `[已驗證，逐指令核對]`

第一組迭代（`SI` 不偏移）讀來源位移 `0,4,8,12`，寫到 `DI, DI+0x50, DI+0xA0,
DI+0xF0`（＝偶數 bank 的連續四行 ＝ 螢幕列 0,2,4,6）；第二組（`SI` 起點 `+2`）
讀 `2,6,10,14`，寫到 `DI+0x2000` 起的同樣四行（＝螢幕列 1,3,5,7）。

兩組合起來：

```
螢幕列 r ← 來源位移 2r        r ∈ [0,8)
```

也就是**來源資料是線性的**——「先偶後奇」的交錯只發生在**目的地位址**上，
是這支函式為了配合 CGA 硬體 framebuffer 而做的，**字型檔本身沒有交錯**。
這跟 `docs/formats/graphics.md` 記載的「`.PIC` 全螢幕圖也是線性、載入時才
交錯」是同一個結論，不是相反。

### 2.2 每列 2 bytes 的 bit 佈局：packed 2bpp `[已驗證，PNG 逐字母肉眼核對]`

`MOVSW` 把來源那 2 個 byte **原封不動**寫進 `0xB800` 相鄰的兩個 byte。
而 CGA mode 4（320×200、4 色）framebuffer 的定義就是「每 byte 4 個像素、
2 bits/像素、最左像素在 bit7–6」。因此字型檔那 2 個 byte 的語意由硬體直接
決定，沒有詮釋空間：

```
frame_offset(row) = row * 2
color_index(col)  = (byte[row*2 + col/4] >> (6 - 2*(col%4))) & 3     // col ∈ [0,8)
```

byte0 = 該列**左邊 4 個像素**，byte1 = **右邊 4 個像素**。

寬高的獨立佐證在呼叫端（§1）：`欄 × 2 bytes` → 8 px 寬；
`列 × 0x140(320) bytes` ÷ 80 bytes/field row = 4 個 field row = 8 條掃描線 → 8 px 高。

### 2.3 已作廢的斷言（2026-07-25 第一版）`[已推翻，不要再套用]`

本文件第一版寫「同列 2 byte 是 bit0/bit1 兩個平面（planar）」並標為已驗證，
**那是錯的**——依該假設寫的解碼器輸出整張都是雜訊，沒有任何可辨識字母。
同時作廢的還有「檔案開頭 1 byte 是 header」（實際是**結尾**的 `0x1A`
DOS EOF）以及「`0x80`–`0xDF` 是未知圖形字元」（實際是反白 bank，見 §4）。

失誤原因值得記著：`MOVSW` 這條指令**不需要**知道 byte 的色彩語意就能寫出
正確的搬運迴圈，所以「讀懂 blit 迴圈」並不等於「讀懂像素格式」——像素格式
是由**目的地（CGA 硬體 framebuffer 規格）**決定的，當時卻繞過硬體規格去
猜資料佈局，還把「猜對了嗎」的判斷交給沒有真正肉眼複核的 atlas。
對應紀律：`rulebook/65`（驗收要對 reference 實測，不能拿內部訊號當通過）。

---

## 3. EGA 字型繪製：`FUN_1d9f_0eb1` → `FUN_217b_097c`

### 3.1 `FUN_1d9f_0eb1`（`1d9f:0eb1`，58 bytes）`[已驗證]`

```c
void FUN_1d9f_0eb1(x, y, char_minus_0x20) {
    if (*(int*)0x1ff0 == 0)
        FUN_217b_097c(x, y, char_minus_0x20, 0xff00);   // fg=0xF(白) bg=0x0(黑)
    else
        FUN_217b_097c(x, y, char_minus_0x20, 0xb);      // fg=0x0(黑) bg=0xB(亮青)
}
```

`*(int*)0x1ff0` 是**反白（highlight）旗標**，已驗證：`1d9f:12ea` 設 1、
`1d9f:1357` 設 0，中間夾的正是「填底色 → 畫置中文字」這段（`1d9f:12c7`
填 EGA 亮青 `0xb`／`1d9f:12ce` 填 CGA `0xaa`＝全部色號 2 亮洋紅）。
CGA 路徑不是換色而是換 bank（§4），兩條路徑用同一個旗標達成同一個
視覺效果。
**重要**：這證明**顏色不是字型檔案本身的資料**，是呼叫端在繪製當下決定
的——`.FNE` 檔案內容只是純粹的 1bpp 前景遮罩。

### 3.2 `FUN_217b_097c`（`217b:097c`，261 bytes）`[已驗證，逐指令核對 disassembly.asm]`

用 EGA Graphics Controller **Write Mode 2**（`port 0x3ce/0x3cf`，
`GC Mode register(index5)=2`）逐列繪製，外層迴圈 `MOV DI,0xe`（**14**）
明確是 14 列；`ADD SI,0x2`（每列結束後）明確是**每列消耗 2 bytes 來源**；
`ADD BX,0x4e` 加上兩次 `INC BX`（每列內）淨移動 `0x50`(80 bytes)，正是
EGA 640 寬畫面的單一 plane 掃描線 byte 數（與 `.SHE` 精靈圖 blit
`FUN_217b_07cf` 用的 `0x50` 完全一致，見 `docs/re/07-sprite-blit.md`）。

**位元遮罩推導**（Write Mode 2 標準用法）：`AL=ES:[SI]`(來源 byte)
先 `SHL AX,CL`（`CL=8-(X&7)`，處理非 byte-aligned 的 X 座標）產生一個
橫跨最多 3 個目的地 byte 的遮罩視窗，對每個目的地 byte 各做兩次
`OUT`+寫入：一次用位移後的遮罩寫前景色（`DH`），一次用遮罩取反寫背景色
（`DL`）——**這代表 EGA 版字型繪製是「整格覆寫」（前景+背景都畫），
不是只畫前景的透明疊加**，跟 CGA 路徑的直接 `MOVSW`（也是整格覆寫）
語意一致。

```
frame_offset(row) = row * 2      // row ∈ [0,14)
pixel(row, x) = bit((row 的 2-byte 合併成 16-bit MSB-first), 15-x)   // x ∈ [0,16)
```

**14 列、16 寬、1bpp、28 bytes/字**，`14/8=1.75`、`16/8=2`——寬度剛好是
CGA 版的 2 倍、高度是 1.75 倍。

> **`.FNE` 不受 `.SHE` 的「載入時水平加倍」影響**（2026-07-25 複核）：
> 素材載入器 `FUN_1d9f_0a8b` 雖然也會把 `.FNT` 的副檔名改成 `.FNE`，
> 但加倍那一步只在副檔名比對到 `shE`/`SHE` 時才觸發（見 `docs/re/07` §2.6）。
> 所以 `.FNE` 檔案裡的 16×14 就是記憶體裡的 16×14，本節結論不變。
>
> 附帶一提，字型的 CGA→EGA 縮放（寬 ×2、高 ×1.75）跟 sprite
> （CGA 16×16 → EGA 16×28，檔案層寬度不變、高度 ×1.75）在**檔案層**確實不同；
> 但把 sprite 的顯示層加倍算進去之後，兩者的「顯示寬 ×2、高 ×1.75」是一致的。

### 3.3 硬體常數交叉驗證 `[已驗證，獨立於繪字函式的第二條證據鏈]`

開機初始化函式 `FUN_1d9f_0008`（`1d9f:0008`）在偵測到 EGA/圖形卡
（`INT 10h` 回傳碼 `0x10` → `DAT_31f0_19fb=1`，見 §5）後，設定：

```c
DAT_53ae = 0x10;   // 16  ← 這正是 FUN_1d9f_0eeb 拿去乘 col 的乘數
DAT_53b0 = 0xe;    // 14  ← 這正是 FUN_1d9f_0eeb 拿去乘 row 的乘數
DAT_53b2 = 0xa80;  // 2688 ← 與 ASC.FNE/GOT.FNE 檔案大小完全相等
```

CGA 分支則是：

```c
DAT_53ae = 8;
DAT_53b0 = 8;
DAT_53b2 = 0xc00;  // 3072 ← ASC.FNT 的資料長度（檔案 3073，扣掉結尾的 0x1A DOS EOF）
```

**這是一條完全獨立於繪字函式本身的證據鏈**——不是靠「讀 blit 迴圈次數」
推出來的，而是靠「開機時寫死的常數表」直接寫出「這個模式下字元格是
幾乗幾」，兩條證據鏈（繪字迴圈結構 + 開機常數表）互相印證同一組數字，
可信度極高。

---

## 4. `.FNT` 的兩個 bank：一般 / 反白 `[已驗證]`

`ASC.FNT` 的 3072 bytes 不是「192 個不同的字」，而是**同一批 96 字
（`0x20`–`0x7F`）的兩個版本**，各 1536 bytes：

| bank | 檔案位移 | 內容 | 用到的色號 |
|---|---|---|---|
| 0 | `0x000`–`0x5FF` | 一般：白字黑底 | 0（黑）、3（白）|
| 1 | `0x600`–`0xBFF` | 反白：黑字亮洋紅底 | 0（黑）、2（亮洋紅）|

三條互相獨立的證據：

1. **選擇算式**（§1）：來源位移 = `1536 * [0x1ff0] + (char-0x20)*16`。
   `1536` 是 `[0x53b2]`(3072) 除以 2，`[0x1ff0]` 只取 0/1。
2. **`[0x1ff0]` 是反白旗標**：`1d9f:12ea MOV word ptr [0x1ff0],0x1` →
   畫一段置中文字 → `1d9f:1357 MOV word ptr [0x1ff0],0x0`。同一支函式在
   `1d9f:12c7`/`12ce` 依 CGA/EGA 先填底色：EGA 用 `0xb`（亮青），
   CGA 用 `0xaa`——`0xAA` 在 2bpp packed 下正是「四個像素全為色號 2
   （亮洋紅）」。EGA 側對照組更直接：`FUN_1d9f_0eb1` 用同一個 `[0x1ff0]`
   在 `0xff00`（白底黑字）與 `0xb`（青底黑字）之間切換（§3.1）。
3. **檔案內容**：實際 dump 出來，bank1 每個 byte 都等於
   `(~bank0_byte) & 0xAA`——例如 `A` 的第一列 bank0 `03 c0` ↔ bank1
   `a8 2a`。像素上就是「色號 3 換成 0、色號 0 換成 2」，底色恰好對上
   `0xAA` 填色。

所以 CGA 版**沒有**「未驗證的 `0x80`–`0xDF` 圖形字元」——那個說法是把
反白 bank 誤讀成延伸字元集，已作廢。EGA 版不需要第二個 bank，因為
Write Mode 2 是在繪製當下指定前景/背景色（§3.1）。

> 注意名詞：這裡的 bank1 是**反白（highlight）**，跟遊戲選單裡的
> `Alternate Character Set`（§5，`ASC` ↔ `GOT` 花體切換）是兩回事。

---

## 5. `Alternate Character Set`：切換機制（部分已驗證）

### 5.1 硬體模式偵測（CGA vs EGA）`[已驗證]`

`FUN_1d9f_0008`（開機初始化）最前段：

```asm
pcVar1 = swi(0x10);      ; INT 10h（BIOS 視訊服務）
cVar2 = pcVar1();
if (cVar2 == 0x10) DAT_31f0_19fb = 1;   ; 偵測到 EGA/更高階顯示卡 → 設旗標
```

`DAT_31f0_19fb` 這個旗標就是 §1 繪字函式判斷 CGA/EGA 路徑的依據，**在
遊戲啟動時由硬體自動偵測決定，不是玩家選單可以切換的**——這解釋了為何
`GOT.FNT`（CGA 版花體）不存在於這份 zip 也不影響 EGA 版可玩：CGA/EGA
的選擇是硬體層級，不是「Alternate Character Set」選單在切換的東西。

### 5.2 `ASC` vs `GOT`（花體切換）：資源表已找到，選單串接未追完 `[部分已驗證]`

在 `DEMON.INT` 資料段找到一張**5 筆遠指標構成的檔名表**（`31f0:19fd`
起，每筆 4 bytes）：

```
index 0 → "asc.fnt"       index 1 → "got.fnt"
index 2 → "party.dat"     index 3 → "itemlocb.dat"
index 4 → "itemlocx.dat"
```

（原始位元組核對：檔案位移 `0x274fd` 起，逐筆比對過內容。）

**已驗證**：`FUN_206a_01fa`（`206a:01fa`，Character Utilities 段）是一個
「依 index 取檔名+大小、呼叫共用載入函式」的泛用資源載入器：

```asm
MOV AX,[BP+6]          ; param_1 = index
SHL AX,1
MOV BX,AX
MOV AX,[BX+0x1a45]     ; 大小表(word)，開機時被寫入目前模式的字型大小
                        ;（DAT_53b2＝3072(CGA)或2688(EGA)，見 §3.3）
CWD
...
MOV AX,[BP+6]
SHL AX,1
SHL AX,1
MOV BX,AX
PUSH [BX+0x19ff]       ; 檔名遠指標的 segment
PUSH [BX+0x19fd]       ; 檔名遠指標的 offset
PUSH DS
MOV AX,0x522a
PUSH AX
CALLF <未解析目標>      ; 見下方「未解」
```

**未解**：這個 `CALLF` 的目標位址（原始立即值 `1000:e47b`）指向 Ghidra
自動分析完全沒有觸及的空白段（`functions.csv` 裡不存在任何 `2000:*`
或 `1000:e47b` 附近的函式，`disassembly.asm` 在那個位址範圍沒有任何
反組譯輸出）——這是 `docs/re/00-ghidra-setup.md` §5 描述的「間隙」，
需要專門對這塊位址範圍重跑分析或手動 `objdump` 硬解才能繼續，
超出本次任務的時間預算。這代表：

1. **確認**了遊戲有一個「依 index 選字型檔（0=ASC／1=GOT）」的通用載入
   路徑，且這個 index 與資源表的順序完全對應「Alternate Character Set」
   這個名稱暗示的功能（ASC=標準字元集、GOT=花體=「替代字元集」）。
2. **沒有追到**「Alternate Character Set」選單項目被選中後，實際傳了
   `index=0` 還是 `index=1` 給 `FUN_206a_01fa`（或它是否直接呼叫這個
   函式，也可能是透過另一層 wrapper）——也就是說，**猜測**這個選單就是
   在 ASC/GOT 之間切換（合情合理，`"got.fnt"` 這個檔名的存在本身就是
   強證據，畢竟遊戲不會平白無故載入一個不會用到的花體字型檔），但**沒有
   逐指令證實選單→呼叫這一段**。

### 5.3 字型緩衝區實際填入：`FUN_1d9f_03e9`（`1d9f:03e9`，901 bytes）`[已驗證]`

字型繪製函式讀的全域指標 `DAT_31f0_5488/548a`（far pointer）**不是**
在 §1、§2、§3 的繪字函式裡設定的，而是在 `FUN_1d9f_03e9`（一個把單一
已載入資源大 blob 切成多個子資源指標的初始化函式）裡，用

```c
DAT_5488 = DAT_5484;
DAT_548a = DAT_5486;
```

從另一個指標 `DAT_5484/5486` **複製**過來的——這代表實際的磁碟讀取發生
在**更早一層**（設定 `0x5484/0x5486` 的地方，本次沒有追到，但 §5.2 找到
的 `FUN_206a_01fa` 是已知會呼叫進一個未解析函式做實際 I/O 的候選路徑，
兩者高度可能是同一條），`FUN_1d9f_03e9` 本身只做指標算術（把一個大
resource blob 切成字型、道具範本等一串子區塊指標，篇幅所限不逐一展開，
與字型格式本身無關）。

---

## 6. PNG 肉眼比對結果

用 `internal/assets/gfx/gfx_test.go` 的 `TestDecodeFonts` 產生
（`workplace/dump/gfx/`，不入版控）：

| 輸出 | 內容 | 比對結果 |
|---|---|---|
| `font-asc-fnt-atlas.png`（+`-zoom4x`） | CGA `ASC.FNT` 全部 192 個 glyph（16 欄 × 12 列）| **前 6 列＝bank0**：`space ! " # $ % & ' ( ) * + , - . /` → `0`–`9` → `A`–`Z` → `a`–`z`，白字黑底，逐字清楚可讀；**後 6 列＝bank1**：同一批字的黑字亮洋紅底反白版 |
| `font-ASC.FNE-atlas.png`（+`-zoom3x`） | EGA `ASC.FNE` 全部 96 個 glyph | 清楚、平頭無襯線字母，字形工整 |
| `font-GOT.FNE-atlas.png`（+`-zoom3x`） | EGA `GOT.FNE` 全部 96 個 glyph | **清楚的花體(blackletter)字母**——每個字母都有哥德式的裝飾筆畫（如 `A` 的頂端分岔、`D` 的弧形襯線） |
| `font-render-got-0.png`~`7.png` | 用 `GOT.FNE` 直接渲染 `"DEMON'S WINTER"`、`"Go adventuring"`、`"Character Utilities"`、`"Alternate Character Set"`、`"Walk"`、`"Party info"`、`"Save Game"`、`"Camp"` | **與 `workplace/dosbox/shots/smoke-01.png`（主選單）、`03-ega-ingame.png`（遊戲內選單）肉眼比對，字形風格一致**——兩張截圖裡的標題與選單文字全部是同一種花體，本文渲染出的同一批字串筆畫特徵（如 `D`、`C`、`S`、`W` 的裝飾轉折）吻合 |
| `font-render-asc-0.png`~`7.png` | 同一批字串改用 `ASC.FNE`（平頭字）渲染 | 作為對照組，證明 GOT 的花體風格是**字型檔本身的差異**，不是解碼器的錯覺 |

**判別依據回應驗收標準**：
- ASCII `0x20`（空白）：兩種字型解碼結果都是全空，已驗證。
- `A`–`Z`、`0`–`9`：CGA 與 EGA 兩版都清楚可辨，已驗證。
- **CGA 的驗證只到「atlas 逐字可讀且順序正確」這一層**：`workplace/dosbox/
  shots/` 裡唯一的 CGA 截圖是 `05-cga-hang-open-pic.png`（開場就卡住，
  畫面上沒有文字），所以 CGA 字形沒有原版畫面可比。下面「與 DOSBox 截圖
  比對」那幾條**全部是 EGA（`GOT.FNE`）的結果**，不要當成 CGA 的證據。
- `TestDecodeFonts` 另外用 `assertCGAFontLayout` 把 `A` 的 8×8 圖樣、
  空白字元全空、兩個 bank 的用色與互補關係寫成斷言，讓佈局判錯時測試會紅，
  不必每次都靠人看圖。
- 與 DOSBox 截圖比對：`smoke-01.png` 的 `"Go adventuring"`／
  `"Character Utilities"`／`"Alternate Character Set"`／花體標題
  `"DEMON'S WINTER"`，以及 `03-ega-ingame.png` 的選單文字（`"Walk"`／
  `"Party info"`／`"Save Game"`／`"Camp"` 等），**字形風格對得上**——
  都是同一種花體、筆畫粗細與裝飾轉折一致。

---

## 7. 未解部分（誠實列出）

1. `0x3000:016a`（32-bit 除法 helper）與實際磁碟 I/O 函式本體落在 Ghidra
   未分析的位址間隙（見 `docs/re/00-ghidra-setup.md` §5），沒有逐指令反組譯。
   除法的回傳值 1536 已被算術與檔案實際佈局雙向夾死（§1、§4），不影響結論。
2. **`Alternate Character Set` 選單項目→實際呼叫鏈的最後一段未追完**
   （§5.2）：找到了資源表與泛用載入器，但選單本身怎麼觸發、傳哪個
   index，因為卡在 Ghidra 分析間隙（`CALLF` 目標落在未分析區）而沒有
   逐指令證實，只能算「高度合理的推測」不是「已驗證」。
3. **實際磁碟讀取（`fopen`/`fread` 等價物）的函式本體**同樣落在同一塊
   分析間隙，未解析出來，因此無法直接證實「載入時機」（開機時全部
   預載、還是選單切換當下才讀）。
4. **`GOT.FNT`（CGA 版花體）**確認不存在於這份 zip 素材，且 §5.1 已確認
   CGA/EGA 是硬體自動偵測、非玩家選單切換，所以**CGA 模式下若切換到
   「Alternate Character Set」，理論上仍會嘗試載入不存在的 `GOT.FNT`
   而失敗／卡死**——這與 `CONTEXT.md` 記載的「CGA 模式因此會卡死」
   吻合，等於是本文件提供了那個已知現象的可能成因（未經 DOSBox 實測，
   標記為推論）。
5. `DAT_31f0_1ff0`（反白旗標）已確認語意與 `1d9f:12ea`/`1d9f:1357`
   這組寫入點，但沒有窮舉其他寫入點，不排除還有別處會設它。

---

## 8. 給引擎渲染層的建議

- `internal/assets/gfx/font.go` 的 `DecodeCGAFont`（8×8、packed 2bpp，回傳
  192 個 glyph：`[0,96)` 一般、`[96,192)` 反白）與
  `DecodeEGAFont`（16×14、1bpp，fg/bg 由呼叫端指定）可直接使用。
  `RenderText` 提供最簡單的字串→影像組字，`GlyphForChar` 處理
  ASCII→glyph 索引（`FontFirstChar=0x20`）。
- **中文化的關鍵前提已成立**：EGA 路徑的字元繪製是「1bpp 遮罩 + 呼叫端
  指定顏色」，且繪字函式本身（`FUN_1d9f_0eeb`/`FUN_217b_097c`）只是
  「查表取 glyph、blit 到畫面」，沒有跟 ASCII 編碼本身耦合的邏輯——
  Go 引擎重寫時，只要新增一張「Unicode/Big5 碼→自訂點陣字 glyph」的
  對照表，繞過原本 `char-0x20` 這條查表算式，就能插入中文字型，
  不需要改動 blit 邏輯本身。
- CGA 路徑因為色深(2bpp)與畫面模式(320×200)較受限，加上 `GOT.FNT` 缺檔，
  建議中文化實作優先鎖定 EGA 路徑（640×350，已是本專案其他素材
  `.PIE`/`.SHE` 的目標畫面模式）。

---

## 附錄：關鍵函式位址一覽

| 函式 | 位址 | 角色 |
|---|---|---|
| `FUN_1d9f_1361` | `1d9f:1361` | 字串印出（逐字元呼叫 `FUN_1d9f_0eeb`） |
| `FUN_1d9f_0eeb` | `1d9f:0eeb` | 單一字元繪製分派（CGA/EGA） |
| `FUN_217b_025a` | `217b:025a` | CGA 字元 blit（8×8、packed 2bpp，直接 MOVSW 進 0xB800） |
| `FUN_1d9f_0eb1` | `1d9f:0eb1` | EGA 字元繪製前處理（座標縮放、選色） |
| `FUN_217b_097c` | `217b:097c` | EGA 字元 blit（16×14、1bpp、Write Mode 2） |
| `FUN_1d9f_0008` | `1d9f:0008` | 開機初始化（`INT 10h` 偵測、寫入 `DAT_53ae/53b0/53b2` 等模式常數） |
| `FUN_1d9f_03e9` | `1d9f:03e9` | 資源 blob 切割成子指標（含字型緩衝區指標 `DAT_5488` 的來源） |
| `FUN_206a_01fa` | `206a:01fa` | 依 index 從 5 筆資源表載入檔案（`asc.fnt`/`got.fnt`/`party.dat`/`itemlocb.dat`/`itemlocx.dat`），實際 I/O 呼叫目標未解析 |
