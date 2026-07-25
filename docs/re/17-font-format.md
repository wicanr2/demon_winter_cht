# 字型格式 `.FNT`/`.FNE`（DEMON.INT 反組譯，2026-07-25）

> **結論先講：兩種字型都已解出並肉眼驗證。** `ASC.FNT`（CGA）是 8×8、2bpp、
> 「同一列 2 個 byte 是 bit0/bit1 兩個平面」；`ASC.FNE`/`GOT.FNE`（EGA）是
> 16×14、1bpp。這推翻了 `docs/formats/graphics.md` §6 原本「1-byte header +
> 256×8×12 1bpp」的猜測——**尺寸、色深、bit 佈局全部猜錯**，正確答案是靠
> `rulebook/62`（讀繪字程式碼，不猜佈局）配合 `rulebook/64`（拿字母形狀當
> oracle 逐一測試假設）解出來的。GOT.FNE 解出來是花體(blackletter)風格，
> 與 `workplace/dosbox/shots/smoke-01.png`、`03-ega-ingame.png` 的花體 UI
> 文字一致。

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
| 色深/佈局 | 2bpp，**同列 2 byte = bit0/bit1 兩個平面**（非 chunky、非逐 plane 整塊） | 1bpp（前景/背景色由呼叫端指定，字型檔本身無顏色） |
| 每字 bytes | 16（8 rows × 2 bytes/row） | 28（14 rows × 2 bytes/row） |
| bit 順序 | MSB-first | MSB-first |
| 字表起點 | ASCII `0x20`（空白） | ASCII `0x20`（空白） |
| 檔頭 | 1 byte（觀察值 `0x00`，用途未查出） | 無 |
| 涵蓋字元數 | 192（`(3073-1)/16`），但**只有 0x20–0x7F(96個)已用可讀文字驗證**，`0x80–0xDF` 解碼出規律但未經畫面驗證的圖案（見 §4） | 96（`2688/28`，恰為 `0x20`–`0x7F` 全部可印 ASCII，無餘數） |
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

- **CGA 路徑**（`0x19fb==0`）：
  ```c
  uVar4 = param_3 * 0x10 - 0x200;              // (char_code - 0x20) * 16
  FUN_217b_025a(param_1*0x140 + param_2*2,      // 目標 CGA 位移
                uVar3 + uVar4, ...);            // 來源字型緩衝區位移
  ```
  `uVar4 = char*16-0x200` 已用 objdump 逐指令核對（`add $0xfe00,%ax` = `+(-0x200)`），
  **16 bytes/字**、字表從 `0x20` 開始，兩者都已驗證。
- **EGA 路徑**（`0x19fb!=0`）：
  ```c
  FUN_1d9f_0eb1(param_1 * DAT_53b0, param_2 * DAT_53ae, param_3 - 0x20);
  ```
  `param_3 - 0x20`（`addw $0xffe0,...`）同樣已用 objdump 核對，確認字表
  同樣從 `0x20` 開始。

---

## 2. CGA 字型繪製：`FUN_217b_025a`（`217b:025a`，93 bytes）

完整 objdump（見 §附錄）：把 CGA graphics framebuffer 段（`ES=0xB800`）
用 **8 行、每行一個 `MOVSW`（2 bytes）** 的方式直接搬字型資料進畫面，
搬運分兩組各 4 次迭代，利用 CGA 硬體「偶數掃描線存前半 8000B、奇數掃描線
存後半」的位址規律（`+0x50` 一次跳 2 條實際掃描線，`+0x2000`/`-0x2000`
在偶/奇兩個 bank 間切換）——**這一段本身是本專案已知的 CGA 硬體定址模式**
（`docs/formats/graphics.md` §2.1 的 `.PIC` 全螢幕圖已驗證過同一套邏輯，
但那邊是**磁碟檔案不交錯、載入時才交錯**；這裡是**函式執行期間直接對
硬體 framebuffer 做交錯寫入**，兩者不衝突，是同一套 CGA 硬體知識的
兩種應用場景）。

### 2.1 來源位移 → 視覺列的對應 `[已驗證，逐指令核對 objdump]`

第一組迭代（`SI` 起點不偏移）依序讀來源 offset `0,4,8,12`，寫到目的地
`DI,DI+0x50,DI+0xA0,DI+0xF0`；第二組（`SI` 起點 `+2`）依序讀 `2,6,10,14`，
寫到與第一組交錯的位置。兩組合起來，**視覺列 row 對應來源 offset
`row*2`**（row0←offset0、row1←offset2、…、row7←offset14）——也就是
「16 bytes 依序排列成 8 列、每列 2 bytes」，這點跟直覺一致，**真正反直覺
的地方在下一節（bit 佈局）**。

### 2.2 每列 2 bytes 的 bit 佈局 `[已驗證，PNG 逐字母肉眼核對]`

**這是本文件唯一需要靠「肉眼比對排除假設」而非純讀指令解出的部分**——
`MOVSW` 只告訴我們「兩個 byte 一起被搬到相鄰的目的地位置」，沒有告訴我們
這兩個 byte 在**色彩語意**上是什麼關係。試過的假設：

| 假設 | 說明 | 結果 |
|---|---|---|
| chunky 2bpp（byte0=左4px、byte1=右4px，比照 `.PIC` 全螢幕圖的慣例） | 每 byte 4 個像素、MSB-first，兩個 byte 左右接續成 8px 一列 | **雜訊**：字母出現「十字部件在兩列間漏出/重複」的撕裂感，且最後一列固定出現一個跟字母無關的殘影 |
| **同列雙平面（byte0=bit0 平面、byte1=bit1 平面，MSB-first，逐 bit 組成 2bpp 色碼）** | `color = bit(byte0,n) \| (bit(byte1,n)<<1)`，n=0..7(MSB-first) | **✅ 清楚**：`H`/`E`/`L`/`O`/`A`–`Z`/`0`–`9`/`!` 全部渲染成正確、無雜訊的字母形狀 |
| 1bpp、16 px 寬(兩 byte 併成一個 16-bit 遮罩) | 比照 EGA 的 1bpp 解法 | **雜訊**：字母的「莖」與「底」出現在不同水平位置，明顯對不齊 |
| 僅用 byte0（byte1 視為未用/metadata） | 猜測 byte1 是保留欄位 | **部分正確但不完整**：多數字母清楚，但如 `L` 這種筆畫集中在 byte1 位元的字母會缺筆畫（`L` 只剩底部橫筆，直筆消失） |

**已驗證的公式**：

```
frame_offset(row) = row * 2            // row ∈ [0,8)，byte0=偏移+0、byte1=偏移+1
color_index(col)  = bit(byte0, 7-col) | (bit(byte1, 7-col) << 1)   // col ∈ [0,8)，MSB-first
```

`color_index` 對應調色盤 `CGAPalette1High`（黑／亮青／亮洋紅／白，與
`docs/formats/graphics.md` 其他 CGA 素材共用同一份調色盤——這是 CGA 硬體
只有兩組固定四色盤可選，不是本專案臆測）。

**為什麼是「同列雙平面」而不是 chunky？** 這跟 EGA 的 Write Mode 2（見
§3）是同一個設計哲學的簡化版：EGA 用硬體暫存器讓「一個 bit 位置同時決定
4 個 plane 的顏色」，CGA 沒有那組硬體，退而求其次在**資料本身**先做好
「同一像素的兩個色彩位元分開存但位置對齊」，搬運時兩個 plane 各自
`MOVSW` 到 CGA 記憶體裡的正確 bit 位置（`.PIC`/`.SHP` 用的 chunky 佈局，
是「一個 byte 內塞滿好幾個完整像素」；字型的雙平面佈局，是「一個 byte
內塞滿好幾個像素的『半顆』」，兩者是 CGA 2bpp 資料常見的兩種互斥慣例，
本專案在 `.PIC`/`.SHP` 用前者、字型用後者——**同一硬體格式在不同素材類型
上可以有不同的位元組織慣例，不能互相套用**，這點呼應
`docs/formats/graphics.md` 已記錄的「算術對不代表方向對」教訓，這次是
「同色深不代表同位元組織」的新變體）。

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

`*(int*)0x1ff0` 是另一個開關（**用途待查**，推測是「反白/選取狀態」，
因為兩組色碼恰好是「白底黑字」與「反過來的青底黑字」，符合選單被選取項目
反白顯示的常見做法，但本次沒有逐一追出所有寫入點，標記為 [推測]）。
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
CGA 版的 2 倍、高度是 1.75 倍，跟 `.SHE` 精靈圖（`docs/re/07` §3.1：
CGA 16×32 → EGA 32×28，寬×2、高×0.875）**不是同一個縮放比**，
再次印證「不同素材類型的 CGA→EGA 縮放規則不能互套」這條 `docs/formats/
graphics.md` 已有的教訓在字型上又成立一次。

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
DAT_53b2 = 0xc00;  // 3072 ← 與 ASC.FNT 扣掉 1-byte header 後的大小完全相等
```

**這是一條完全獨立於繪字函式本身的證據鏈**——不是靠「讀 blit 迴圈次數」
推出來的，而是靠「開機時寫死的常數表」直接寫出「這個模式下字元格是
幾乗幾」，兩條證據鏈（繪字迴圈結構 + 開機常數表）互相印證同一組數字，
可信度極高。

---

## 4. 已驗證 vs 未驗證的字元範圍

- **`0x20`–`0x7F`(96 個，含空白到 `~`)：已驗證**。CGA、EGA 兩版都用完整
  A–Z/a–z/0–9/常見標點渲染過，PNG 逐字清楚可讀，見 §6。
- **CGA `0x80`–`0xDF`（額外 96 個，佔滿 `ASC.FNT` 扣 header 後 3072 bytes
  的後半）：解碼結果有規律結構（重複的棋盤位元圖樣、看起來像陰影/框線
  字元），但目前找不到任何遊戲字串使用這個範圍（`strings.csv` 714 條
  字串全部是純 7-bit ASCII），也沒有對應畫面可以肉眼核對，**誠實標記
  為未驗證**——不是雜訊（有結構），但也不是「已驗證正確」，可能是
  CGA 版額外附帶的框線/陰影圖形字元（1980 年代常見做法），也可能是檔案
  裡緊接在字型表後面的下一個資源（本次沒有排查 `ASC.FNT` 除字型外
  是否還有其他附加資料）。
- **EGA 版沒有這個問題**：`ASC.FNE`/`GOT.FNE` 都是 2688 bytes，
  `2688/28=96` 整除無餘數，檔案大小本身就界定了字元範圍恰好是
  `0x20`–`0x7F`，沒有多餘資料。

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
| `font-asc-fnt-atlas.png`（+`-zoom4x`） | CGA `ASC.FNT` 全部 192 個 glyph 排版 | `0x20`–`0x7F` 清楚可讀英文字母/數字/標點；`0x80`+ 有規律但未驗證（見 §4） |
| `font-ASC.FNE-atlas.png`（+`-zoom3x`） | EGA `ASC.FNE` 全部 96 個 glyph | 清楚、平頭無襯線字母，字形工整 |
| `font-GOT.FNE-atlas.png`（+`-zoom3x`） | EGA `GOT.FNE` 全部 96 個 glyph | **清楚的花體(blackletter)字母**——每個字母都有哥德式的裝飾筆畫（如 `A` 的頂端分岔、`D` 的弧形襯線） |
| `font-render-got-0.png`~`7.png` | 用 `GOT.FNE` 直接渲染 `"DEMON'S WINTER"`、`"Go adventuring"`、`"Character Utilities"`、`"Alternate Character Set"`、`"Walk"`、`"Party info"`、`"Save Game"`、`"Camp"` | **與 `workplace/dosbox/shots/smoke-01.png`（主選單）、`03-ega-ingame.png`（遊戲內選單）肉眼比對，字形風格一致**——兩張截圖裡的標題與選單文字全部是同一種花體，本文渲染出的同一批字串筆畫特徵（如 `D`、`C`、`S`、`W` 的裝飾轉折）吻合 |
| `font-render-asc-0.png`~`7.png` | 同一批字串改用 `ASC.FNE`（平頭字）渲染 | 作為對照組，證明 GOT 的花體風格是**字型檔本身的差異**，不是解碼器的錯覺 |

**判別依據回應驗收標準**：
- ASCII `0x20`（空白）：兩種字型解碼結果都是全空，已驗證。
- `A`–`Z`、`0`–`9`：CGA 與 EGA 兩版都清楚可辨，已驗證。
- 與 DOSBox 截圖比對：`smoke-01.png` 的 `"Go adventuring"`／
  `"Character Utilities"`／`"Alternate Character Set"`／花體標題
  `"DEMON'S WINTER"`，以及 `03-ega-ingame.png` 的選單文字（`"Walk"`／
  `"Party info"`／`"Save Game"`／`"Camp"` 等），**字形風格對得上**——
  都是同一種花體、筆畫粗細與裝飾轉折一致。

---

## 7. 未解部分（誠實列出）

1. **CGA 字型 `0x80`–`0xDF` 範圍**的真實用途未驗證（見 §4）。
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
5. `DAT_31f0_1ff0`（EGA 路徑的「反白/選取」旗標，§3.1）的完整寫入點
   未逐一追出，只確認它切換兩組固定的前景/背景色。

---

## 8. 給引擎渲染層的建議

- `internal/assets/gfx/font.go` 的 `DecodeCGAFont`（8×8、2bpp 雙平面）與
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
| `FUN_217b_025a` | `217b:025a` | CGA 字元 blit（8×8、2bpp 雙平面） |
| `FUN_1d9f_0eb1` | `1d9f:0eb1` | EGA 字元繪製前處理（座標縮放、選色） |
| `FUN_217b_097c` | `217b:097c` | EGA 字元 blit（16×14、1bpp、Write Mode 2） |
| `FUN_1d9f_0008` | `1d9f:0008` | 開機初始化（`INT 10h` 偵測、寫入 `DAT_53ae/53b0/53b2` 等模式常數） |
| `FUN_1d9f_03e9` | `1d9f:03e9` | 資源 blob 切割成子指標（含字型緩衝區指標 `DAT_5488` 的來源） |
| `FUN_206a_01fa` | `206a:01fa` | 依 index 從 5 筆資源表載入檔案（`asc.fnt`/`got.fnt`/`party.dat`/`itemlocb.dat`/`itemlocx.dat`），實際 I/O 呼叫目標未解析 |
