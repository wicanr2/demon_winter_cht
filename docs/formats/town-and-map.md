# 城鎮定義與地圖資料格式

> **⚠ 2026-07-24 反組譯修正**（兩處，均見 [`docs/re/02`](../re/02-data-loading-functions.md)
> 與 [`docs/re/03`](../re/03-audio-and-resources.md)）：
> 1. **SUM.MAP 已完全解開**：它是 23 個 sub-map segment 的串接（RLE 壓縮），
>    由 `FUN_222f_28d0` 依 map ID 載入。size 表（`31f0:24b6`，23 筆）加總**恰為 15,743**，
>    等於檔案大小（已獨立複核）。這也解答了本文「ITEMLOCB 引用 `map_id=2/4` 卻無對應檔案」之謎
>    ——那些 map 就在 SUM.MAP 裡。本文對 SUM.MAP「是否壓縮／結構未解」的判斷以 `docs/re/03` 為準。
> 2. **城鎮 28 個 type_code 與 7 個設施字串無關**：設施顯示由記錄 29 的固定 payload 欄位決定
>    （`FUN_278d_0098`），不是 type_code 查表。28 個 code 的真正語意仍未解，見 `docs/re/02`。

> 分析對象：`workplace/orig/demwin/DEM_DATA/` 底下的 `TOWN1.DAT`–`TOWN25.DAT`、`EXITS.DAT`、
> `MAP1.MAP`/`MAP3.MAP`/`MAP5.MAP`、`SUM.MAP`、`ITEMLOCB.DAT`/`ITEMLOCX.DAT`。
> 解析工具：`tools/parse_town.py`、`tools/parse_map.py`（純標準庫，可重跑）。
> 每個結論都標了「已驗證」（有跨檔案一致性、可程式驗證的證據）或「假設」（合理但未證實）。

---

## 1. TOWN*.DAT — 城鎮記錄結構

### 1.1 記錄長度：17 bytes（已驗證）

對全部 25 個 `TOWN*.DAT` 掃描位元組序列 `0a 01 01`，共找到 211 筆命中，其中 **203 筆（96%）**
落在 `(offset − 14) mod 17 == 0` 的位置上——也就是說,只要把檔案從 offset 0 開始，每 17 bytes
切一刀，這個序列幾乎永遠出現在同一筆記錄的最後 3 個 byte（local offset 14–16）。未對齊的 8 筆
全部落在檔案後段（offset 350–493 附近），屬於巧合命中（那一段是別的子結構，見 1.4）。

**512 = 30 × 17 + 2**，且驗證後確定是「30 筆 17-byte 記錄 **在前**，2 bytes **在檔尾**」，
不是先前假設的「2 bytes header 在檔頭」——因為 record 0 就是從 offset 0 開始，切下去欄位對齊得非常乾淨
（下方 1.2 的逐筆 dump 可直接看出規律）。

驗證方式：`python3 tools/parse_town.py town workplace/orig/demwin/DEM_DATA/TOWN1.DAT`

### 1.2 單筆記錄欄位（部分已驗證、部分待補）

```
偏移(記錄內)  長度   欄位            狀態
0             1      類型碼 (code)    已驗證存在,語意未定
1–13          13     payload         已驗證存在,多為 0,有時是數值(見1.3)
14–16         3      尾端 marker     已驗證存在,值不固定(見下)
```

以 `TOWN1.DAT`（Seaside）前 11 筆記錄為例（`tools/parse_town.py town` 的實際輸出）：

```
rec 0 @0x000  code=0x00(  0)  payload=00×13              tail=0a 01 01
rec 1 @0x011  code=0x01(  1)  payload=00×13              tail=0a 01 01
rec 2 @0x022  code=0x02(  2)  payload=00×13              tail=0a 01 01
rec 3 @0x033  code=0x03(  3)  payload=00×13              tail=0a 01 01
rec 4 @0x044  code=0x1a( 26)  payload=00×13              tail=0a 01 01
rec 5 @0x055  code=0x08(  8)  payload=00×13              tail=0a 01 01
rec 6 @0x066  code=0x09(  9)  payload=00×13              tail=0a 01 01
rec 7 @0x077  code=0x19( 25)  payload=00 00 00 00 c8 00 0d 06 00 00 00 00 00  tail=00 01 01
rec 8 @0x088  code=0x19( 25)  payload=00 00 00 00 c8 00 20 05 00 00 00 00 00  tail=00 01 01
rec 9 @0x099  code=0x05(  5)  payload=00×13              tail=0a 04 01
rec10 @0x0aa  code=0x00(  0)  payload=00×13              tail=0b 02 01
```

**已驗證**：
- byte0（類型碼）在每一筆記錄都獨立存在，不是延續前一筆的衍生值——起初以為 `0a 01 01 XX` 是
  「上一筆的尾端 + 下一筆的類型碼」構成的鏈結，但跨 25 檔驗證後發現 **record 0 的 byte0 本身就會變化**
  （`TOWN2.DAT`=26、`TOWN9.DAT`=2、`TOWN15.DAT`=8……），代表每筆記錄的 byte0 是獨立欄位，不是鏈結指標。
  之前那個「連續遞增」的印象只是 `TOWN1.DAT` 前 7 筆剛好code是 1,2,3 這種巧合。
- payload 全零時，尾端幾乎固定是 `0a 01 01`（本文稱 **type-A：純旗標記錄**，可能代表「這個設施存在，
  沒有附加數值」）。
- payload 非全零時（**type-B：有資料記錄**），數字通常出現在 payload 中段，例如 `TOWN1.DAT` record 7/8
  的 `c8` = **0xC8 = 200**，很可能是價格（金幣）——`DEMON.INT` 內確實有 `Marketplace`／`Healing %3d`
  這類含數值的字串，price 欄位存在合理。

**未驗證（待下一步）**：
- 類型碼（0–27，共 28 種相異值）與 `DEMON.INT` 內找到的設施短字串 `Healers / Rest / Town guild /
  Church / Docks / Pub`（外加 `!Marketplace`，共 **7 個字串**，見 §1.5）**對不起來**——28 種碼遠多於
  7 個設施字串，代表 TOWN*.DAT 的類型碼**不是**「選單設施類型」這麼單純，更可能是更細的
  「事件/建築/NPC 觸發 ID」，需要對照可執行檔的 dispatch table 才能逐一定案，這超出本次資料格式分析
  的範圍（屬於另一位 agent 負責的 `docs/formats/event-script.md` 領域）。
- 尾端 3 bytes 的規律不只 `0a 01 01`：也出現過 `0a 02 01`、`0a 04 01`、`0b 01 01`、`0b 02 01`、
  `0d 01 01` 等變體，第一個 byte（0x0a/0x0b/0x0d）疑似另一個子欄位而非固定 magic number，語意未解。
- 檔案最尾端 2 bytes（offset 510–511）：多數城鎮是 `00 00`，但 `TOWN5.DAT`/`TOWN6.DAT` 是 `4f 50`、
  `TOWN7.DAT` 是 `e1 00`、`TOWN25.DAT` 是 `02 00`——不是恆定值,可能是 checksum、旗標或殘留資料，未解。

### 1.3 高信心度的深入發現：「ELRIC」樣板角色殘留

25 座城鎮中有 **14 座**（`TOWN1,2,3,4,11,12,13,14,16,18,19,22,23,24`）在 payload 陣列後段
（大約 record 15–17，隨每座城鎮的設施數量浮動）出現**逐 byte 完全相同**的字串：

```
ff 45 4c 52 49 43 20 20 20 20 20 03 07 0a 08 0b 0b   <- 0xff + "ELRIC" + 5個空白 + 03 07 0a 08 0b 0b
ff 0b 0b 0b 0b ff ff 00 ...                          <- 接續 0b 0b 0b 0b 再 ff ff 結束
```

`0xFF` 開頭是這段陣列（見下方 §1.4）的「空位」標記；`ELRIC     `（10 字元含 5 個空白）後面接的
`03 07 0a 08 0b`（3,7,10,8,11）跟 `translations/glossary.md` 第 2 節列出的 5 個角色屬性
（Speed / Strength / Intellect / Endurance / Skill）數量吻合。

**判斷（假設，中信心）**：這不是每座城鎮各自的 NPC 資料，而是 SSI 開發用的「Town Maker」編輯工具
在存檔時把自己記憶體裡的一個**測試角色樣板**（開發者自己的角色，取名 Elric）原封不動存進了
「NPC/招募名單」欄位的空位，剛好落在同一個相對位置沒被覆寫掉。這解釋了為什麼 14 座城鎮的這段
byte 完全一致——它不是遊戲內容，是未清空 buffer 的殘留。同一批城鎮檔案裡另外也發現
`"SELECT ONE"`、`"MENU"`、`"CHAR GEN"`、`"DEMON'S WINTER TOWN MAKE..."`、`"PLAY"`、`"OBJ"`、`"ADD"`
等明顯是**編輯工具自己 UI 文字**的殘片（在 `TOWN5/6/7/10/25.DAT` 裡），進一步支持這個判斷。

### 1.4 記錄陣列的三段式結構（已驗證分段界線，語意部分待解）

把 30 筆記錄依內容特徵分三段（界線因城鎮而異，見 `tools/parse_town.py town-all` 輸出）：

| 段落 | 記錄範圍(視城鎮而定) | byte0 特徵 | 判斷 |
|---|---|---|---|
| A. 設施/事件列表 | 通常 record 0–10 附近 | code 0x00–0x1b，type-A(純flag)居多 | 已驗證存在；語意未解(見1.2) |
| B. NPC/名單陣列 | 通常 record 11–28 | 幾乎全部 `0xFF`(空位)，偶爾出現實際資料(如 ELRIC 樣板) | 假設：招募 NPC 或商店店員名單，未證實 |
| C. 尾端 | record 29 + 尾 2 bytes | 混合內容 | 未解 |

25 座城鎮的「段落 A 有效記錄數」統計（`tools/parse_town.py town-all` 輸出，節錄）：

| 編號 | 城名 | type-A | type-B | FF空位 |
|---|---|---|---|---|
| 1 | Seaside | 7 | 3 | 18 |
| 9 | Myrquacid | 4 | 8 | 17 |
| 13 | Iris | 0 | 6 | 24 |
| 20 | Terlabba | 12 | 14 | 1 |
| 24 | Land's Edge | 0 | 8 | 22 |

`Iris`（13）與 `Land's Edge`（24）的 type-A 記錄數為 0，且這兩個檔案完全沒出現 §1.1 的
`0a 01 01` 錨點——與這兩座是「小型/次要地點」的合理推測一致（`TOWN.TXT` 裡它們的名字也偏向
邊陲小島/據點類地名）。`Terlabba`（20）則是資料最豐富的城鎮（26 筆有效記錄，只剩 1 個空位），
可能是主線核心城鎮。這個對照尚未逐一比對攻略內文確認，標記為**假設**。

### 1.5 城鎮設施：DEMON.INT 內找到的字串（已驗證字串存在，未驗證與 TOWN*.DAT 欄位的對應）

`DEMON.INT` 內以緊密相鄰、null-terminated 的方式排列著兩組共通字串（已驗證位置與間距完全對齊）：

```
短標籤(選單用):  Healers / Rest / Town guild / Church / Docks / Pub    (+ 前面另有 !Marketplace)
完整句(場景用):  Go to marketplace / Healers / Rest in the Inn / Town guild / Church of  / Docks / Pub
```

這證實遊戲確實有 **7 種通用城鎮設施**（Marketplace、Healers、Inn、Town guild、Church、Docks、Pub），
與任務說明裡列的 7 種吻合。但如 §1.2 所述，TOWN*.DAT 的類型碼有 28 種，遠多於 7，**兩者的對應關係
未能在本次資料格式分析中解出**——需要反組譯執行檔找到「哪個 code 呼叫哪個設施字串」的 dispatch
邏輯才能確定，建議交給有反組譯環境的後續工作（`docs/re/`）處理。

### 1.6 25 座城鎮對照表

城名取自 `TOWN.TXT`（已驗證：檔案以 null-terminated 字串排列，前 25 個字串 = 25 座城鎮，
第 26 個字串起是劇情文字，索引順序與 `TOWN1..25.DAT` 的編號一一對應——`translations/glossary.md`
第 10 節前兩筆的翻譯也與此順序吻合）：

| # | 城名 (原文) | # | 城名 (原文) | # | 城名 (原文) |
|---|---|---|---|---|---|
| 1 | Seaside | 10 | Urlock | 19 | Idlewood |
| 2 | Elbarat | 11 | Dragontooth | 20 | Terlabba |
| 3 | Akistu | 12 | Chandris | 21 | Loven |
| 4 | Alynhawk | 13 | Iris | 22 | Mojured |
| 5 | Paladine | 14 | Asaht | 23 | Lumisle Island |
| 6 | Erguard | 15 | Irondome | 24 | Land's Edge |
| 7 | Janthrin | 16 | Ynoth | 25 | Pirate's Cove |
| 8 | New Gleon | 17 | Aurora | | |
| 9 | Myrquacid | 18 | Woodhaven | | |

完整每座城鎮的 type-A/type-B/FF 統計見上方 §1.4 或直接執行：
```
python3 tools/parse_town.py town-all workplace/orig/demwin/DEM_DATA
```

---

## 2. MAP1.MAP / MAP3.MAP / MAP5.MAP — 地城地圖

### 2.1 64×64 假設：已驗證

3 個檔案都是 4097 bytes = 1 + 64×64。把 offset 1 之後的 4096 bytes 當 64×64 陣列印成 ASCII 圖
（`tools/parse_map.py ascii`），三張圖都清楚浮現房間、走廊、對稱牆體等人造建築結構——不是雜訊。

**MAP1.MAP 節錄**（第 9–14 列，第 40–56 行——一個乾淨的矩形房間，牆體用 `#`）：
```
++.....   .......
++.####..#####...
++.###########.  
++.###########. .
++.###########. .
++.###########. .
```

**MAP5.MAP** 呈現左右鏡射對稱的走廊+房間結構（`.* *.` 重複的長走道、`^^^^`/`*****` 對稱色塊），
視覺上非常像一個有規劃的地城樓層。

**MAP3.MAP** 則明顯不同：單一 tile（86, 字元 `"`）佔了 63% 的面積、另一 tile（49, `=`）佔 24%，
兩者合計 87%，形狀是大面積的開放區塊被 `=` 外框圈住，比較像洞窟/野外大空間而非房間迷宮式地城。

各檔第一個 byte（header）：`MAP1=0x00`、`MAP3=0x97`、`MAP5=0x09`。此 header 的意義未解——
不是簡單的 map_id（3,5 不吻合），也未找到其他解釋，標記為**假設**：可能是調色盤/樓層編號/音樂編號等。

完整 ASCII 圖存於 `workplace/dump/maps/MAP1.ascii.txt`、`MAP3.ascii.txt`、`MAP5.ascii.txt`（不入版控）。

### 2.2 Tile 編號語意：大多未解，僅有粗略猜測

沒有找到 tileset 圖檔可以做 palette-free 結構比對（`rulebook/64` 的「截圖 oracle」手法在這裡缺 oracle
素材，此路暫時走不通），因此 tile 編號語意目前只能靠**視覺形狀 + 出現頻率**做粗略推測，
**信心度低**，僅供後續美術資產比對時參考：

| tile值 | 字元 | 出現頻率(MAP1/3/5) | 猜測 |
|---|---|---|---|
| 13 | `.` | 高 (MAP1 53%) | 地板 |
| 0 | (空白) | 中 (MAP1 19%, MAP5 23%) | 未映射區域/邊界外 |
| 98 | `#` | 中 (MAP1 7%) | 牆 |
| 35 | `+` | 中 | 門或另一種牆 |
| 86 | `"` | MAP3 主體(63%) | 開放地形(野外/洞窟?) |
| 49 | `=` | MAP3 次要(24%) | 邊框/水域? |
| 90 | `^` | MAP5 對稱區塊 | 特殊地形(山?屋頂?) |

### 2.3 攻略路線交叉驗證：結果不確定（誠實回報，非強驗證）

任務說明提供 `docs/walkthrough/part-4.md`「加穆爾神殿（Temple of Gamur）」段落的路線字串
`4E 3N 3E 5S 3E 3S 5W S 2W 2N 2W N 6W`（40 步，13 段轉折）。用 `tools/parse_map.py route`
搭配窮舉起點做了兩種嚴格度的測試：

1. **嚴格版**（沿途每一格都必須是「該地圖出現頻率最高的單一 tile」）：
   - `MAP1`：0 個起點可以完整走完整條路線。
   - `MAP5`：0 個起點可以完整走完整條路線。
   - `MAP3`：**18 個起點**可以完整走完（例如從 (47,41) 出發）。
2. **寬鬆版**（沿途只要落在該地圖出現頻率前 3 名的 tile 即可）：三張圖都有數十到數百個起點符合——
   **太寬鬆，沒有鑑別力**（因為前 3 名 tile 往往佔了地圖 6–9 成面積，隨便走都會中）。

**結論（誠實回報，非「已驗證」）**：只有 `MAP3.MAP` 在嚴格版測試中對這條路線「打得通」，
`MAP1`/`MAP5` 完全打不通。但把其中一條命中路徑疊圖檢視後（見下），這條路徑只是在 MAP3
大面積開放地形裡繞來繞去，**看不出「兩間大型側室各由鬼火看守」這種攻略描述的房間感**，
加上遊戲共有 7 個地城、但目錄裡只有 3 個 `.MAP` 檔，**「加穆爾神殿」到底對應哪個檔案（甚至是否
對應這三個檔案之一）仍是未知數**。這條驗證線索**不足以下定論**，僅記錄嘗試過程供後續參考：

```
（MAP3.MAP, 起點(47,41), @ = 路徑，僅節錄，完整見 workplace/dump/maps/）
""???"""""""""""""""?"
"?????"*"""""?""""""""
??"""??*""""""""""""""
*""-""**""""@@@@""""""
***"***"""""@""@""""""
```

### 2.4 SUM.MAP：未解，判斷「不是簡單壓縮，但也不是簡單方陣」

- 15743 bytes，`125² = 15625`、`126² = 15876` 都不吻合，**不是簡單方陣 tile array**（已驗證：無 header
  情況下無法整除任何合理正方形邊長；15743 = 7 × 13 × 173，因數分解也沒有給出合理的地圖寬高）。
- **熵值分析**（`tools/parse_map.py entropy`）：Shannon entropy = **5.53 bits/byte**，
  distinct byte 值 160/256。做為對照：`MAP1.MAP`（已知是原始 tile array，非壓縮）= 2.37 bits/byte；
  `TOWN.TXT`（已知是純英文文字）= 4.50 bits/byte。SUM.MAP 的熵值明顯比這兩者都高，但離
  常見壓縮演算法輸出的 7.5–8.0 bits/byte 還有距離——**不像典型壓縮資料，但也不是簡單的原始 tile 陣列**，
  判斷它可能是某種**變長記錄的複合結構**（座標列表、分段索引等），而非單一壓縮流。
- 測試了一個簡單 RLE 假設（byte 最高位=1 時，`byte & 0x7f` 是重複次數、下一 byte 是值）：
  解出的總 tile 數是 156865 = 5 × 137 × 229，**沒有對應到任何合理的地圖尺寸**，這個 RLE 假設
  **未通過驗證**，予以否證。
- 對 `0x94`（出現 1860 次，11.8%，全檔最高頻）與 `0xA3`（953 次）做間距統計，兩者的間距分布都在
  4 上有明顯峰值，暗示可能是某種「4-byte 為主、變長」的記錄流（例如 marker+座標+其他欄位），
  但未能進一步拆解出穩定欄位。
- **結論：未解**。建議下一步方向：(a) 找 `SUM` 這個名稱在反組譯結果裡的讀取邏輯（讀取 stride、
  迴圈上限），走 `rulebook/62` 的靜態溯源路徑會比繼續猜測有效；(b) 如果遊戲有「顯示世界地圖」的
  畫面，可用 `rulebook/64` 的截圖 oracle 手法反推。這兩者都超出本次任務的工具與時間範圍。

---

## 3. EXITS.DAT — ~~假設：165 筆 (X,Y) 座標對~~ → 已由反組譯修正為 110 筆 3-byte 記錄

> **⚠ 2026-07-24 反組譯修正**：本節「165 筆 2-byte 座標對」的假設**已被推翻**。
> 讀取函式 `FUN_222f_1321`（`222f:1321`）的 stride 證實記錄是 **3-byte `[X, Y, type_byte]`**，
> 共 330÷3 = **110 筆**。獨立複核：3-byte 分組下 X/Y 全落在 1–62（吻合 64×64 地圖）；
> 若按 2-byte 分組，X 會出現 64（超出 0–63 的有效座標），故 2-byte 假設不成立。
> `type_byte // 32` 是類別（0–7），類別 4 = 傳送點（讀第二組座標覆寫玩家位置，已驗證）。
> `EXITS.DAT` 不是靜態全域表，跨 dataset（地城）時會被整份覆寫。
> 完整分析見 **[`docs/re/05-event-triggering.md`](../re/05-event-triggering.md)**。
> 一個未解矛盾：類別 0 佔 94/110 卻被觸發閘門排除，見該文件 §4。

330 bytes。**已驗證的客觀事實**：
- 全檔案 **沒有任何一個 0x00 byte**（min=1）。
- 最大值 64（僅 2 個 byte 超過 63，其餘都 ≤ 63）。
- 330 能被 2、3、5、6、10、11、15、22、30、33 整除，測試多種 stride 找「每筆記錄第一個 byte
  是否落在 1–25（城鎮編號範圍）」，沒有一個 stride 能讓第一欄乾淨地落在該範圍——**不支援
  「每筆記錄以城鎮編號起頭」的假設**。

**假設（中信心）**：330 ÷ 2 = **165 筆 (X, Y) 座標對**，數值範圍（1–64，幾乎不超過 63）與
64×64 地圖座標高度吻合，是目前最簡單、與已知資料最一致的假說。**未驗證**：這些座標對應哪張地圖
（`MAP1`/`3`/`5`？還是城鎮內部小地圖？）、以及每個座標代表出口/樓梯/門的哪一種。
`tools/parse_town.py exits` 可完整列出所有 165 筆座標供後續交叉比對用。

---

## 4. ITEMLOCB.DAT / ITEMLOCX.DAT — 已驗證：85 筆 (X, Y, map_id) 三元組

### 4.1 兩檔逐 byte 完全相同（已驗證）

```
$ md5sum ITEMLOCB.DAT ITEMLOCX.DAT
c7a453491abe6ab08483a92e72a218eb  ITEMLOCB.DAT
c7a453491abe6ab08483a92e72a218eb  ITEMLOCX.DAT
```

兩個檔案是同一份資料。推測是類似 `PARTY.DAT`/`PARTY.BAK` 那種主檔/備份檔的關係（該目錄底下確實
存在 `PARTY.DAT`+`PARTY.BAK` 這組慣例），但 `ITEMLOCB`/`ITEMLOCX` 的檔名不是 `.BAK` 尾碼，
確切的 B/X 命名意圖未解——**假設**是主檔/備份檔，尚未找到反例。

### 4.2 記錄格式：3-byte (X, Y, map_id)，已驗證

256 bytes ÷ 3 = 85 筆記錄 + 1 byte 尾端（`0xff`）。**驗證方式**：把資料切成 3-byte 一組，
檢視第三欄（map_id）的分布，結果如下（`tools/parse_town.py itemloc` 完整輸出）：

| map_id | 筆數 | 備註 |
|---|---|---|
| 1 | 17 | 對應現存的 `MAP1.MAP` |
| 3 | 27 | 對應現存的 `MAP3.MAP` |
| 4 | 5 | **`MAP4.MAP` 檔案不存在於此資料目錄，但這裡明確出現 map_id=4**——證明 4 號地城在遊戲邏輯上確實存在，只是它的 `.MAP` 檔沒有被收錄在這份資料裡（可能是原版就沒有獨立存檔、或封裝/擷取時遺漏，需要後續在別處找） |
| 5 | 1 | 對應現存的 `MAP5.MAP` |
| 160/238/239/243 | 各1–2 | 明顯是雜訊(見下) |
| 255(0xff) | 30 | 空位 sentinel |

`map_id` 只出現 1/3/4/5，從不出現 2——與「`MAP2.MAP` 也不存在」完全吻合，形成雙重交叉驗證：
**兩個獨立資料來源（ITEMLOC 的 map_id 欄位、以及檔案系統裡缺少 MAP2/MAP4 檔案）一致指向同一個結論
「2 號地城沒有被存成獨立 tile map」**，這是本次分析裡少數有雙重證據支撐的結論。

前 50 筆（index 0–49）座標值都落在合理的 0–63 範圍內，且以 map_id 分段排列（先是全部 map_id=1
的 17 筆，接著全部 map_id=3 的 27 筆，中間穿插幾筆 map_id=4／5），**已驗證**是有意義的資料。
index 14/17/24 出現 `(0,0)`，推測是「此格未放道具」的空紀錄。

index 50–54 這 5 筆資料呈現 `map_id` 落在 160/238/239/243 這種明顯超出範圍的雜訊值（例如
`x=239 y=199 map_id=160`），介於「有效資料」和「後段全 0xff 空位」之間，判斷是又一批**未清空
buffer 殘留**（呼應 §1.3 的 ELRIC 樣板現象，同一套存檔機制的通病）。index 55–84 全部是
`ff ff ff`（空位 sentinel），與 TOWN*.DAT 段落B的 `0xFF` 慣例一致。

---

## 5. 未解問題與建議下一步

依重要性排序：

1. **TOWN*.DAT 類型碼（0–27）與具體設施/事件的對應關係未解**——`DEMON.INT` 只找到 7 個通用設施
   字串，數量對不上 28 種類型碼。建議：反組譯執行檔，找「讀到某類型碼後跳到哪段程式」的
   dispatch table（`rulebook/62` 靜態溯源），或用存檔差分法（`rulebook/02-data-formats.md` 的
   DOSBox 差分驗證手法）：進某座已知只有「Healers」的城鎮，比對它的類型碼是否穩定對應。
2. **SUM.MAP 完全未解**——不是簡單方陣、不像典型壓縮、RLE 假設已被否證。需要反組譯或截圖 oracle
   才有機會突破，本次分析只能停在「排除了幾個簡單假設」這一步。
3. **EXITS.DAT 的座標對應哪張地圖、哪種出口類型**——目前只驗證了數值範圍吻合，語意未解。
4. **MAP*.MAP 的 tile 編號語意**（牆/門/樓梯/陷阱）幾乎全靠猜，沒有找到 tileset 圖檔可以用
   `rulebook/64` 的截圖 oracle 手法反推，是後續最值得投入的方向（如果能找到原版實機截圖）。
5. **7 個地城裡只有 3 個有對應的 `.MAP` 檔**（1/3/5），`Temple of Gamur` 等其他地城資料在哪裡
   仍是未知數；§2.3 的路線走圖測試沒有給出決定性答案。
6. **TOWN*.DAT 尾端 2 bytes 與段落B（NPC陣列）的確切用途**未解，僅有 ELRIC 樣板殘留這一項
   高信心度的旁證（證明那段是「有意義但常被覆寫」的資料區，而非固定填充）。

---

## 附：如何重跑本文件的驗證

```bash
# TOWN*.DAT
python3 tools/parse_town.py town workplace/orig/demwin/DEM_DATA/TOWN1.DAT
python3 tools/parse_town.py town-all workplace/orig/demwin/DEM_DATA
python3 tools/parse_town.py exits workplace/orig/demwin/DEM_DATA/EXITS.DAT
python3 tools/parse_town.py itemloc workplace/orig/demwin/DEM_DATA/ITEMLOCB.DAT

# MAP*.MAP / SUM.MAP
python3 tools/parse_map.py ascii workplace/orig/demwin/DEM_DATA/MAP1.MAP --out workplace/dump/maps/MAP1.ascii.txt
python3 tools/parse_map.py stats workplace/orig/demwin/DEM_DATA/MAP3.MAP
python3 tools/parse_map.py entropy workplace/orig/demwin/DEM_DATA/SUM.MAP
python3 tools/parse_map.py route workplace/orig/demwin/DEM_DATA/MAP3.MAP \
    --start 47,41 --dirs "4E 3N 3E 5S 3E 3S 5W S 2W 2N 2W N 6W"
```
