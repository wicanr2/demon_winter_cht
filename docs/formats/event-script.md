# DEM_DATA 事件腳本格式（DATA*.TXT 系列）

> **⚠ 2026-07-24 反組譯修正**：本文以純資料分析建立的「trailer（0~3 個數字）語意未解」模型
> **已被反組譯推翻**。讀取函式 `FUN_25be_0e77`（`25be:0e77`）的實際 parse 流程證實：
> 所謂「trailer」不是變長欄位，而是兩個固定單值槽 + 下一筆記錄的 leading picture ID
> 被線性 tokenize 誤歸給前一筆。完整結論（含反組譯證據與獨立模擬器驗證）見
> **[`docs/re/02-data-loading-functions.md`](../re/02-data-loading-functions.md)**。
> 本文以下的「未解」段落請以該文件為準。

> 分析對象：`workplace/orig/demwin/DEM_DATA/` 底下的 `DATA1~5.TXT`、`T.TXT`、`T2C.TXT`、`T2D.TXT`、
> `EREGORE.TXT`、`WIN.TXT`、`TOWN.TXT`。
> 解析工具：`tools/parse_datatxt.py`（可重跑，見檔案內 docstring 用法）。
> 前提認知：`DEMON.INT` 是原生 8086 機器碼，**沒有 bytecode 虛擬機**；本文分析的是唯一「資料驅動」的一層。

---

## 0. 結論先講：破解到什麼程度

- **DATA1~5.TXT 的整體記錄結構：已破解，且有強交叉驗證**（見 §2、§4）。每筆記錄 = 一段敘述文字 +
  一組數字欄位；數字欄位裡的「怪物 ID」已用 `MONSTER.DAT` 的怪物名稱表逐一核對，並且能對回
  `docs/walkthrough/part-4.md` 描述的實際劇情場景（加穆爾神殿地下墓穴、狗頭人營地、庫瑞克、Xeres 決戰等）。
- **數字欄位裡的「trailer」（怪物清單結束後剩下的 1~3 個數字）：語意未解**，只確認它存在、長度不固定、
  常見值是 `1, 255` 或單獨 `255`，少數記錄出現具體小數字（`1`、`2`、`3`）。這是本文檔最大的未解問題。
- **T.TXT / T2C.TXT / T2D.TXT / EREGORE.TXT / WIN.TXT / TOWN.TXT：純文字索引表，無數字欄位**，
  這點在原任務「已知事實」中已提到，本次分析對 T.TXT/T2C.TXT 之外的檔案也做了同樣驗證，全部成立。
  額外發現：這些檔案裡用單獨一個欄位 `*`（或 `**`、`***`）當作「場景/段落結束」的控制標記，
  且 `T.TXT`、`EREGORE.TXT` 的每個文字欄位長度都 ≤ 40 字元，符合「一欄 = 螢幕上一行」的假說（見 §5）。

---

## 1. 各檔案的角色

| 檔案 | 大小 | 欄位數 | 型別 | 角色（已驗證 / 推測） |
|---|---|---|---|---|
| `DATA1.TXT` | 6,973 B | 221 | 混合（數字+文字） | **已驗證**：加穆爾神殿地下墓穴（Temple of Gamur catacombs）事件表，含 Remondadin、Eregore 的手下守衛戰、Hall of Bones 等場景，對應 `docs/walkthrough/part-4.md` §3.2 |
| `DATA2.TXT` | 2,131 B | 82 | 混合 | **已驗證**：狗頭人營地（Kobold Camp）事件表，含 Uffuspgot 首領戰，對應 §3.1 |
| `DATA3.TXT` | 4,049 B | 152 | 混合 | **已驗證**：庫瑞克（Qoorik）事件表，含符文密語「YMROS IS MINE」、Grave Keeper 首領戰，對應 §3.3 |
| `DATA4.TXT` | 4,299 B | 118 | 混合 | **已驗證**：Xeres 決戰所在地城（冰之教堂 / 惡魔神殿附近）事件表，含 Jesric、Xeres 本尊戰鬥房 |
| `DATA5.TXT` | 2,095 B | 59 | 混合 | **已驗證**：另一地城事件表，含 Guardian、Great dragon + Demon lord 的橋上戰鬥 |
| `T.TXT` | 1,670 B | 57 | 純文字 | **推測**：開場惡夢過場文字（內容是「諸神已死」的預言夢境），每欄 ≤40 字元，疑似逐行顯示的過場劇情腳本 |
| `T2C.TXT` | 605 B | 6 | 純文字 | **推測**：某個法器/精靈相關的簡短過場對白（內容提及「orb」「spirit」），欄位極少，可能是特定道具觸發的插敘 |
| `T2D.TXT` | 5,149 B | 48 | 純文字 | **推測**：另一段較長的過場劇情（提及 White Knights、Orb、Imprison spell 等劇情關鍵詞），欄位長度不像 T.TXT/EREGORE.TXT 固定在 40 字元，比較像整段落文字 |
| `EREGORE.TXT` | 3,621 B | 129 | 純文字 | **已驗證**：Eregore 首領戰的逐行過場文字，每欄 ≤40 字元，內容清楚描述「Eregore 只剩 2 名手下」到「Eregore lies dead at your feet」的戰鬥流程 |
| `WIN.TXT` | 3,987 B | 94 | 純文字 | **推測**：破關結局文字（提及 Malifon、Imprison spell 生效、火山崩塌），欄位長度不固定（最長到 192 字元），比較像段落文字而非逐行腳本 |
| `TOWN.TXT` | 2,063 B | 37 | 純文字 | **已驗證，且修正了原「已知事實」的描述**：欄位 0–24 是 25 座城鎮名稱（與 `TOWN1~25.DAT` 對應，且與攻略提到的「海濱鎮 Seaside」「新格里昂 New Gleon」名稱吻合）；欄位 25–36 其實是**城鎮 NPC 閒聊傳聞文字**（rumor barks），不是城鎮名稱。原「已知事實 #2」說它是「純字串清單（城鎮名）」，嚴格說是對的（無數字欄位），但內容不是只有城鎮名，這點做個修正 |

---

## 2. DATA*.TXT 的記錄結構（已驗證核心）

### 2.1 分隔與型別（沿用已知事實，並用位元組層級驗證）

- 分隔符是 **NUL (0x00)**，不是換行。
- 數字欄位是 **ASCII 十進位文字**（例如 `"255"` = 3 個位元組 `0x32 0x35 0x35`），不是二進位整數。
- 用 `tools/parse_datatxt.py dump` 可以看到逐欄位的型別與 offset，例如 `DATA1.TXT` 開頭：

  ```
  [   0] @0x00000 N 255
  [   1] @0x00004 T 'Throughout the length of the hallway you see skeletons entombed...'
  [   2] @0x000ab N 0
  [   3] @0x000ad N 255
  [   4] @0x000b1 N 1
  [   5] @0x000b3 N 255
  [   6] @0x000b7 T 'Throughout the length of the hallway you see skeletons entombed...'
  ```

  這和任務背景給的位元組範例完全一致。

### 2.2 檔案層級結構

```
file := header_number  record*
```

- 每個 `DATA*.TXT` 都以**單一數字欄位 `255`** 開頭，接著才是第一筆記錄的文字。五個檔案的 header 完全一樣，
  都是 `[255]`。**這個 header 欄位的用途未解**——不像是「記錄總數」（因為五個檔的記錄數分別是
  40/14/29/26/16，header 卻恆為 255），比較像是固定的檔案格式起始哨兵，或是讀取迴圈裡「上一筆記錄的
  trailer 佔位」造成的巧合。需要反組譯 `DEMON.INT` 讀檔函式才能確認（見 §6）。

### 2.3 記錄結構（每筆事件）

```
record := TEXT  number_list
number_list := []                                   # 極少見，見 DATA1 #11 之後的邊界情形
             | [255]                                 # 「反應/密語文字」類型，見 §2.5
             | [count, id_1 .. id_count, 255?, trailer...]   # 一般房間記錄，見下
```

其中：

| 欄位 | 已驗證 / 推測 | 說明 |
|---|---|---|
| `TEXT` | 已驗證 | 進入該房間 / 觸發該事件時顯示的敘述文字，長度不固定（可到數百字元） |
| `count` | **已驗證**（值域 0–7） | 這筆記錄的怪物遭遇組數。0 = 無戰鬥的純敘述房間 |
| `id_1..id_count` | **已驗證** | 依序是 `MONSTER.DAT` 的怪物索引（0-based，值域 0–98），可重複（例如同種怪物出現兩次） |
| `255`（monster list 後） | **已驗證為終止哨兵**，但不是每筆都有（DATA1 #34、DATA2 多筆缺這個 255，見 §4） | 標記怪物清單結束 |
| `trailer` | **已解** | 見 `docs/re/02` §[F]：第一個數字是**續接碼**，反組譯 + Python 逐欄位模擬對 5 個 `DATA*.TXT` 做過 100% 消耗驗證。~~推測是「強制移動出口 / 事件旗標 / 下一段劇情索引」之一~~ 那三個猜測都不必再考慮 |

### 2.4 交叉驗證證據（這是本文最有把握的部分）

把 `count` 之後的怪物 ID 對回 `MONSTER.DAT`（0-based 索引 → 怪物名稱），得到的名稱和該筆記錄的敘述文字、
以及 `docs/walkthrough/part-4.md` 的劇情描述**高度吻合**，逐一列舉：

| 檔案 #記錄 | 敘述文字節錄 | 解出的怪物 ID → 名稱 | 對應攻略內容 |
|---|---|---|---|
| DATA1 #21 | "A bit fat **hill giant** guards this room" | `[10]` → **Hill giant** | part-4.md §3.2「丘陵巨人的房間」 |
| DATA1 #11 | "occupied by a ghostly white figure"（下一筆 #12 開頭是 "With **Remondadin** dead…"） | `[93]` → **Remondadin** | part-4.md §3.2 地下墓穴頭目 Remondadin |
| DATA1 #36 | "yells **Eregore** from the northern edge…" | `[83,35,36,32,31,31,82]` → Master thief / Lvl12 fighter / Lvl13 mage / Lvl8 mage / Lvl8 fighter ×2 / Highwayman（Eregore 本人另有 `EREGORE.TXT` 專屬過場，不在這組怪物清單內） | 對應 Eregore 手下守衛，Eregore 本尊另見 §1 EREGORE.TXT |
| DATA2 #0 | "tent of **four kobolds**. Kobolds are placed 4 to a tent" | `[26,26,26,26]` → **Kobold ×4** | part-4.md §3.1 狗頭人營地 |
| DATA2 #3 | "tent of **Uffuspgot** the kobold Captain" | `[26,26,85,2,2]` → Kobold ×2, **Uffuspgot**, Orc ×2 | 首領戰帳篷 |
| DATA3 #9 | 純密語文字 `%YMROS.IS...MINE` | （無怪物，見 §2.5） | part-4.md §3.3「符文對應『YMROS IS MINE』」——**逐字吻合** |
| DATA3 #22 | "The **Grave Keeper**. A large burial mound…" | `[11,23,8,22,17,7,89]` → Evil spirit / Ghoul / Skeleton mage / Ghost / Zombie / Skeleton / **Grave keeper** | 墓穴守護者首領戰，7 隻不死生物同時出現 |
| DATA4 #5 | "**Xeres** looks down on you and speaks…" | `[90,90,30,87,91]` → Salamander ×2 / Gargoyle / Slaver / **Xeres** | part-4.md 全書最終頭目戰 |
| DATA4 #21 | "'Welcome fools! I am **Jesric** of the High Temple…" | `[96]` → **Jesric** | 高階神殿頭目 |
| DATA5 #15 | "golden bridge spans the lake of fire… **Two [dragons]**…" | `[53,53,46]` → **Great dragon** ×2, **Demon lord** | 橋上頭目戰 |

以上每一列都是「文字內容明確點名某怪物/數量」+「解出的 ID 對回 `MONSTER.DAT` 名稱完全一致」的雙重確認，
不是巧合。這也順帶**驗證了 `MONSTER.DAT` 是 0-based、依檔案順序索引的怪物表**（`Hill giant`=10、
`Kobold`=26、`Uffuspgot`=85、`Xeres`=91、`Remondadin`=93、`Jesric`=96、`Eregore`=97、`Guardian`=98）。

### 2.5 特殊記錄子型別：控制字元開頭的文字

有一批記錄的 `TEXT` 欄位**第一個字元本身是控制字元**（`3` 或 `%`），且它們的 `number_list` 永遠只有
單一個 `255`（沒有 count、沒有怪物清單、也沒有 trailer）：

| 前綴 | 出現次數（5 個 DATA 檔合計） | 內容特徵 | 推測用途 |
|---|---|---|---|
| `3` | 12 | 緊接在頭目戰記錄之後，內容是「戰後反應」敘述（例如 `3With Remondadin dead the black room is strangely silent...`、`3Eregore stands before a large black mirror...`） | **推測**：由前一筆記錄的戰鬥結算觸發的「後續劇情文字」，不是玩家走進地圖座標觸發的一般房間，`3` 可能是「文字顯示模式/類別」代碼 |
| `%` | 8 | 內容是**符文/密語提示**，例如 `%YMROS.IS...MINE`、`%RING.BELL...AT.....MIDNIGHT`、`%DIVINITY`，句中空格被替換成句點 `.` | **已驗證**（`docs/re/72`）：`%` 標記「用符文字型顯示」。`0x1a85c` 判斷字串首字元 `0x25`（＝`%`）才呼叫 `25be:18fa`，而那支的第一件事就是載入 `CYPHER.SHP`（1728 bytes ÷ 64 ＝ **27 個 16×16 符文字形**，26 字母 + 句點）。句點取代空白是因為**字型裡有句點的字形、沒有空白的字形**。原本標「推測」的兩點都成立 |

例外：`DATA4 #4`（"3Xeres speaks in a voice that bellows..."）的 `number_list` 是 `[5]` 而非 `[255]`，
是這個子型別裡唯一的離群值，未找到解釋，見 §4。

---

## 3. TOWN.TXT / T.TXT 等純文字檔的結構

```
file := field*     # 全部是文字欄位，無數字
```

- `TOWN.TXT`：欄位 0–24 是城鎮名稱（已用攻略地名核對，如 `Seaside`=海濱鎮、`New Gleon`=新格里昂），
  25–36 是城鎮裡 NPC 的傳聞閒聊文字（rumor barks），內容包含劇情伏筆（Jesric、White Knights、
  Malifon 等）。
- `T.TXT`、`EREGORE.TXT`：每個文字欄位都 **≤40 字元**，且以獨立的 `*` 欄位收尾，符合「一欄 = 螢幕上
  一行、`*` = 這段過場結束」的假說（**推測**，未反組譯證實，但長度上限精準卡在 40 這個典型 DOS 文字
  視窗寬度，加上多筆連續短句組成完整段落，強烈提示是逐行顯示的過場腳本）。
- `WIN.TXT`、`T2D.TXT`：欄位長度不受 40 字元限制（`WIN.TXT` 最長到 192 字元），比較像是「一欄 = 一整段
  文字」而非逐行文字。同樣以 `*`/`**`/`***` 收尾。**推測**兩種檔案對應引擎裡不同的文字顯示函式
  （逐行 typewriter 效果 vs. 整段捲動），但沒有反組譯佐證。

---

## 4. 未解的部分（誠實列出）

1. **DATA*.TXT 檔案開頭的 header 數字 `255` 的用途。** 五個檔案完全一樣，不隨記錄數變化，不像是
   「記錄總數」。需要反組譯 `DEMON.INT` 裡讀取 `DATA%d.TXT` 的函式，看這個欄位有沒有被讀取、讀到哪個
   變數。
2. **`trailer`（怪物清單終止 `255` 之後剩下的 0–3 個數字）的語意。** 這是最關鍵的未解問題，因為如果
   trailer 裡藏著「事件跳轉/下一個房間」的索引，就是任務要的「條件→文字→後續跳轉」三元組的最後一塊；
   如果只是旗標，那事件之間的「跳轉」邏輯可能根本不在這個檔案裡（例如完全由 `MAP*.MAP` 的座標決定，
   `DATA*.TXT` 只是被動的「座標對應到第 N 筆記錄」查表，沒有記錄與記錄之間的跳轉關係）。目前手上的證據
   只能確定：
   - 出現最多的 trailer 是 `[1, 255]`（怪物清單有內容時）或單純 `[]`（清單後直接結束，如 DATA1 #11、#21、
     #36）。
   - 少數 trailer 出現具體小數字（DATA1 #10 是 `[1, 3]`、DATA3 #21 是 `[1, 1]`、DATA4 #13 是 `[1, 2]`），
     且這幾筆記錄的敘述文字都跟「移動/通道/出口」有關（"A large black door..."、"You have exited the
     tombstone maze..."、"To the south is a door carved of ice..."），**推測**（信心中等）trailer 第二個
     值是某種「強制出口/傳送目標」索引，`255` = 無特殊出口。但樣本太少（僅 3 筆），不足以確立。
3. **`count`+`id*count` 之後為什麼不是每筆都接著 `255` 終止符。** DATA1 #34（`[1,46,4,1,255]`，count=1
   卻在單一 id 之後多了 `4` 才接 `255`）、DATA2 多筆 4-kobold 記錄（`count=4` 之後直接接 trailer，中間沒有
   `255`）是明確的反例，代表 `count-prefix` 假說**不是 100% 覆蓋**，只是目前找得到、且有交叉驗證支持的
   最佳近似模型。有兩種可能：(a) 真實 grammar 比目前假說複雜（例如 trailer 本身也是變動長度，`255` 只是
   湊巧出現在多數 trailer 裡而非固定終止符）；(b) 這些是資料輸入時的不規則/手動誤植。無法單從資料本身
   分辨，需要反組譯佐證。
4. **`DATA4 #4` 的離群值** `[5]`（見 §2.5）：控制字元 `3` 開頭卻不是慣常的 `[255]`，語意未解。
5. **~~`T2C.TXT`、`T2D.TXT` 的觸發條件~~ 已解（`docs/re/70`）：子地圖 `+0xa3 < 3` 讀 `T2C.TXT`（偏移 605）、`>= 3` 讀 `T2D.TXT`（偏移 5149），而且兩者是同一個字串被就地改寫第 3 個 byte。剩 `T.TXT`／`EREGORE.TXT` 的觸發時機（`WIN.TXT` 已不重要，結局文字寫死在資料段）。** 這些純文字檔沒有
   任何欄位表明「什麼時候顯示第幾筆」，索引邏輯必然在 `DEMON.INT` 裡（可能是寫死的常數索引，對應特定
   劇情節點），本文分析範圍內無法確定。
6. **DATA*.TXT 記錄之間、以及記錄與地圖座標（`MAP1/3/5.MAP`）之間的對應關係完全未驗證。** 本文只驗證了
   「記錄內容 = 某個房間/事件」，沒有驗證「哪個地圖座標對應第幾筆記錄」。這是任務背景提出的「房間編號 /
   事件編號索引」假說裡唯一還沒被證實的一環。

### 建議下一步

- **優先**：反組譯 `DEMON.INT` 裡讀取 `DATA%d.TXT` 的函式（依 `62-static-provenance-trace` 的溯源
  SOP，從「這個檔案被 `fopen`/讀取的那行」往回追，通常不深），確認：
  (a) header `255` 有沒有被用到、(b) trailer 欄位被讀進哪個變數、是否影響控制流、(c) 記錄的索引方式
  （循序讀取 vs. 隨機存取某個 record number）。
- **次要**：若能在 DOSBox 裡實跑，進入加穆爾神殿地下墓穴（DATA1 對應地城），刻意觸發幾個已標注
  「trailer 有具體小數字」的房間（如 DATA1 #10 黑色大門），觀察走進去之後角色實際被傳送到哪裡，
  用來反證「trailer 第二值 = 傳送目標」的假說。
- **次要**：比對 `MAP1.MAP`／`MAP3.MAP`／`MAP5.MAP`（已知存在，`MAP2`/`MAP4` 缺）的座標資料，看能不能
  在地圖檔裡找到「指向 DATA*.TXT 第 N 筆記錄」的索引欄位，建立座標↔記錄的對應表。

---

## 5. 是否推翻「已知事實」？

- 沒有推翻既有已知事實，但有兩點**補充/修正**：
  1. 已知事實只點出 `T.TXT`、`T2C.TXT` 開頭是敘事文字、無數字前綴；本次分析確認 **`T2D.TXT`、
     `EREGORE.TXT`、`WIN.TXT`、`TOWN.TXT` 全部同構**（純文字表，無數字欄位），把已知事實的範圍完整補齊。
  2. `TOWN.TXT`「純字串清單（城鎮名）」的描述需要修正：**它確實無數字欄位，但內容不是只有城鎮名**——
     後半段（欄位 25–36）是 NPC 傳聞閒聊文字，不是地名。

---

## 6. 工具使用

```bash
# 逐欄位 dump（型別 + offset + 內容），任何檔案都能跑，不做結構假設
python3 tools/parse_datatxt.py dump workplace/orig/demwin/DEM_DATA/DATA1.TXT

# 針對 DATA*.TXT 套用本文 §2 的記錄結構，並把怪物 ID 換成名稱
python3 tools/parse_datatxt.py records workplace/orig/demwin/DEM_DATA/DATA1.TXT \
    --monster-dat workplace/orig/demwin/DEM_DATA/MONSTER.DAT

# 輸出 JSON，供其他程式（例如日後的 Go 引擎資產轉換腳本）消費
python3 tools/parse_datatxt.py records workplace/orig/demwin/DEM_DATA/DATA1.TXT \
    --monster-dat workplace/orig/demwin/DEM_DATA/MONSTER.DAT --json > data1.json
```
