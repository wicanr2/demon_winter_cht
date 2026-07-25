# 角色系統與城鎮經濟（DEMON.INT 反組譯 + DOSBox 動態驗證）

> 分析對象：建角（種族/屬性擲點/職業/技能）、升級（城鎮公會）、城鎮經濟（商店/治療所/客棧/神殿）。
> 方法：先照 `docs/re/00-ghidra-setup.md` 的字串錨定法找程式碼，撞牆後改用
> `tools/dosbox_run.sh`（既有環境，未新建流程）動態驗證，並用 `python3` 直接讀
> `workplace/dosbox/game/DEM_DATA/PARTY.DAT` 位元組跟畫面顯示值交叉核對。
> 位址換算公式、字串索引方式同 `docs/re/00`；`31f0` 段字串位址已用原始位元組核對過。
> 本文與 `docs/formats/game-data-tables.md`（角色記錄欄位表）互補，**不修改**該檔，
> 衝突點在第 7 節列出。

## 0. 結論先講

- **建角屬性擲點公式：已驗證（DOSBox 動態 + PARTY.DAT 位元組雙重交叉核對）。**
  基礎骰值與種族無關、期望值 8（強證據：遊戲 UI 自己畫出的「種族平均值」方框，
  5 種族全部等於「8 ± 手冊列出的修正值」，逐一比對 5 種族、25 個修正值**全部吻合**）。
  種族修正是**加法**、套用順序＝**先擲骰（所有種族用同一分佈）、後加種族固定修正**，
  沒有發現任何「先套用上限再擲」或「依種族換骰子大小」的證據。（§1.2）
- **這個公式只能小幅解釋「人類擲點較優」的攻略觀察，無法完全解釋。**
  人類淨修正 0，其餘種族淨修正 −1～−2，期望總分差距只有 1～2 分，
  遠不足以說明攻略講的「人類 60–65 分不難、其他種族從沒破 55」這種量級的落差。
  誠實列為部分佐證，懷疑攻略作者的觀察摻雜小樣本偏誤（見 §1.4）。
- **升級演算法：已完全驗證，含指令級證據。** HP／SP 成長公式、「擲兩次取較大值」機制、
  屬性隨機分配（含「已達種族上限則重骰」邏輯）、公會升級**免費**（程式碼內完全沒有
  扣款路徑，已驗證）。（§2）
- **初始 HP／SP／技能點：已驗證（PARTY.DAT 位元組直接對帳）。**
  初始最大 HP＝耐力（種族修正後最終值）；初始技能點＝智力（1:1，兩次獨立測試皆吻合）；
  初始 SP 因職業而異（巫師＝智力，遊俠＝0，其餘職業未逐一測試）。（§1.3）
- ~~**城鎮經濟（商店定價／說服／治療所／客棧）：反組譯層級未解。**~~
  **⚠ 本條的結論已被 `docs/re/19-town-economy.md` 推翻一半，勿再引用原判斷。**
  當時寫「這些字串完全沒有被 Ghidra 自動分析找到任何引用指令」，實際情況是
  **Ghidra 的線性掃描有正確反組譯出這些指令，只是沒建立函式邊界、decompiler
  沒被觸發**，所以只查 `decompiled_all.c` 與 `functions.csv` 會看不到。
  改為直接在 `disassembly.asm` 裡搜字串引用（含函式間隙、以及 segment 開頭
  第一個命名函式**之前**的區域），神殿／學院／碼頭／治療所／酒館五項全部讀得出來，
  且都有完整公式。詳見 `docs/re/19` §1 的方法論與 §3–§6 的結果。
  真正符合「Ghidra 完全沒走到」定義的只剩市集議價一項。
  本節 §3、§4 的原始位址勘查仍然有效，是 `docs/re/19` 的起點。
- ~~**神殿祈禱／神祇機制：同上，未解。**~~（§4）
  **已解** —— 祈禱費用 `= 角色等級 × 50` 金幣，神祇呼喚成功率存在角色記錄 `+0xeb`
  （0–20 直接存百分比），祈禱重設回 20。見 `docs/re/19` §3.3／§3.4。
- **休息／恢復／狩獵：手冊記載可直接引用（非 RE 驗證），巨魔再生機制未查證。**（§5）

---

## 1. 建角（Character Creation）

### 1.1 UI 流程與觸發鏈 `[已驗證，含反組譯 + DOSBox 雙重證據]`

**反組譯找到的部分**（標題畫面到「Character Utilities」子選單）：

- `FUN_206a_003b`（`206a:003b`，436 bytes，反編譯乾淨）是程式最外層的標題畫面迴圈，
  呼叫 `FUN_2cdc_090d(0)` 取得標題選單選擇（0=Go adventuring、1=Character Utilities、
  2=Alternate Character Set），`iVar3==1` 時呼叫 `FUN_206a_02c7()`。
- `FUN_2cdc_090d`（`2cdc:090d`，1483 bytes，反編譯乾淨）是**泛用選單元件的另一個變體**
  （跟 `docs/re/04` 已解的 `FUN_2cdc_033d` 是同一個家族，但這個版本額外支援
  `param_1==1` 時顯示「PARTY STATUS」角色狀態清單，直接讀 `party+idx*0x104+0xfd`
  （目前 HP）／`+0xff`（目前 SP）——**這是本文找到的新證據，交叉驗證了
  `docs/formats/game-data-tables.md` 已記載的 `0xfd`/`0xff` 欄位語意**）。
  標題選單的三個選項來自資料表 `31f0:437e`（4-byte 遠指標陣列），
  跟旁邊的字串（`DEMON'S WINTER`／`Go adventuring`／`Character Utilities`／
  `Alternate Character Set`……）緊接在一起（檔案位移 `0x29e12` 起，原始位元組核對過）。
- `FUN_206a_02c7`（`206a:02c7`，宣告 802 bytes，但反編譯輸出 4242 行——跟
  `docs/re/04` §2.4 描述的「跳表被誤判成一般控制流」同一種 decompiler 損毀）是
  「Character Utilities」子選單（Create／Remove／List／Initialize Party）的處理函式。
  其中 `case 4`（對應 Create Character）**反編譯可讀**的部分確認：
  - 呼叫 `FUN_1d9f_1bf4`（角色名字輸入元件），空白名字時觸發「Do you wish to erase
    this character?」（本文用 DOSBox 直接截圖驗證這條路徑，見 1.1.1）。
  - 找party陣列裡第一個 sentinel（`race`欄位＝`0xFF`的slot） 分配給新角色。
  - 把新角色記錄的一段陣列（`char+0xc`起、每 17 bytes 一組、共 10 組）與
    `char+0x100`／`+0x101`（裝備武器／護甲槽索引）全部初始化成 `0xFF` sentinel。
    **這段 0xc 起始、stride 17 的陣列跟 `docs/formats/game-data-tables.md` 記載的
    「裝備欄 `0x1a`–`0xc3`」在起始位移上兜不起來，見 §7.2 衝突點。**

**反組譯撞牆的部分（誠實列出，不是隨便放棄）**：

「Choose race:」（`31f0:1e1c`）以及緊鄰的建角相關字串（`Create Character` `31f0:1d8c`、
`Remove Character` `31f0:1d9d`、`1Speed` `31f0:17ae`、`1Spell Points` `31f0:18cf`……共
超過 10 個字串）**在整個 `strings.csv`／`decompiled_all.c`／`disassembly.asm` 裡
找不到任何一條指令引用它們**（逐一 grep 十六進位位移、grep decompiled 全部 348 個函式
皆零命中）。已排除的可能：

1. 不是分離出去的另一個執行檔——`DEMON.EXE` 只有 6036 bytes，`strings` 檢查內容
   只是啟動樁（檢查 8087/80287、顯示 `OPEN.PIC`、`EXEC \demon.int`），不含任何
   角色創建邏輯，字串也不在其中。
2. 不是查表遺漏——比照 `docs/re/04` 找到主選單三張表的方法（往資料段找緊鄰的
   遠指標表），對 `Character Utilities` 子選單成功找到了資料表（`31f0:437e`，
   見上），但**同樣的手法對「Choose race:」找不到對應表／呼叫端**。
3. 唯一合理解釋：**這段程式碼落在 Ghidra 自動分析沒有走到的區塊**（間接跳轉／
   跳表分派，`docs/re/00-ghidra-setup.md` 自動分析評估章節已預警的最壞情況），
   `FUN_206a_02c7` 本身反編譯損毀（跳表誤判）支持這個解釋——真正的
   「選種族→擲屬性→選職業→選技能」流程很可能就藏在這個函式被 decompiler
   弄壞、沒有正確展開的那段跳表裡。

依 `rulebook/62-static-provenance-trace` 的紀律，撞牆後改用 DOSBox 動態驗證
（任務單本身也建議「建角流程最適合動態驗證」），詳見 1.1.1 起。

### 1.1.1 DOSBox 操作流程 `[已驗證]`

用既有 `tools/dosbox_run.sh`（未新建流程），流程與熱鍵：

```
標題畫面 → 按 c (Character Utilities)
  → 按 i (Initialize Party) → Return(確認 OK) → Return(確認 "Ready for a new party!")
  → 按 c (Create Character)
    → 數字鍵 1–5 選種族 (1=Human 2=Elf 3=Dwarf 4=Dark Elf 5=Troll)
      → 顯示 "Turn # 1" 五項屬性擲骰結果 + REROLL 按鈕
      → Return 會重骰全部 5 項屬性、Turn 計數 +1，最多到 "Turn # 3"（無 REROLL 選項，強制採用）
        （驗證了手冊「三次機會、最多重骰兩次」的說法，見 1.2）
    → 選職業（字母鍵 A–J）→ 顯示「You have N points usable」技能點畫面
      → Return 略過選技能（手冊允許 0–2 個）
    → 輸入角色名字 → Return
      → 若名字空白，跳出「Do you wish to erase this character?」Yes/No 對話框
      → 若輸入實際名字，直接完成建立，回到 Character Utilities 選單
```

實測 3 次完整建角＋存檔（`TestHP`／`TestWiz`／`TestTroll`），把
`workplace/dosbox/game/DEM_DATA/PARTY.DAT` 複製出來用 `python3` 直接讀位元組，
跟畫面顯示的屬性值**逐一比對，全部吻合**（例：`TestHP` 畫面顯示
Speed12/Str8/Int11/End9/Skill11，`PARTY.DAT` 位元組 `0xf3`/`0xe8`/`0xf9`/`0xfa`/`0xe9`
讀出來正好是 `12/8/11/9/11`）——這確認了 UI 顯示值＝實際存檔值，動態驗證的資料可信。

### 1.2 屬性擲點公式 `[已驗證]`

**核心發現：畫面右側的「種族平均值」方框，直接洩漏了公式。**

建角種族選擇畫面右側有個方框顯示 5 個數字，手冊稱為「該種族這些屬性的平均值」。
本文對 5 個種族各截一張圖，讀出方框數值，跟手冊 `docs/manual/part-1.md` 的種族修正表
逐項比對：

| 種族 | 方框顯示（Speed/Str/Int/End/Skill） | 手冊修正（Speed/Str/Int/End/Skill） | 8 + 修正 | 吻合？ |
|---|---|---|---|---|
| Human | 8,8,8,8,8 | 無 | 8,8,8,8,8 | **完全吻合** |
| Elf | 8,6,10,7,8 | −,−2,+2,−1,− | 8,6,10,7,8 | **完全吻合** |
| Dwarf | 7,9,6,8,8 | −1,+1,−2,−,− | 7,9,6,8,8 | **完全吻合** |
| Dark Elf | 8,6,10,7,7 | −,−2,+2,−1,−1 | 8,6,10,7,7 | **完全吻合** |
| Troll | 7,8,5,10,8（Speed/Str 兩碼因字型解析度低、肉眼判讀信心中等） | −1,−1,−3,+2,− | 7,7,5,10,8 | 中/末三碼吻合，前兩碼判讀有疑義（見下） |

（Troll 的 Speed/Str 兩碼截圖在 8×8 點陣字型下 `7` 與 `8` 容易混淆，只有這組數字
可能有肉眼誤判；Intellect=5、Endurance=10、Skill=8 三碼清楚可辨且與公式完全吻合，
考慮到其餘 4 種族、21 碼全部精確吻合，本文判定公式成立、Troll 那兩碼視為讀圖雜訊
而非公式反例，但誠實標註信心等級。）

**結論**：

```
displayed_stat = base_roll + race_modifier(race, stat)
```

其中 `base_roll` 是**與種族無關**的同一分佈，期望值精確等於 8（不是約略值——5 個種族
25 個修正值全部整數對上，沒有任何取整誤差空間）。`race_modifier` 就是
`docs/manual/part-1.md` 種族表列出的整數修正值，**用加法套用，沒有套用上限、
沒有依種族換骰子大小**。

**套用順序的答案**：任務單問「擲點方式與種族修正的套用順序」——答案是**沒有交錯
順序可言，是單一階段**：擲一次與種族無關的骰子，加上種族固定修正，就是最終顯示值。
不存在「先套用種族上限、超過的部分重骰」這種二階段機制（那是**升級**時才有的邏輯，
見 §2.2，不要混淆兩者）。

**基礎骰子的具體分佈（`Roll(15)`，均勻 `[1,15]`）**：`[假設，中高信心]`。
證據：(a) 5 種族方框顯示的期望值精確等於整數 8——若骰子是 `Roll(n)`（均勻 `[1,n]`，
已知全域 RNG 介面 `FUN_1d9f_0e0b`，語意見 `docs/re/06-combat-system.md` §4.1），
`E[Roll(n)] = (n+1)/2`，要精確等於整數 8 只有 `n=15`（`(15+1)/2=8`）；
`n=16` 會得到 8.5，不會是乾淨整數。(b) 觀察到的最大顯示值是 15（人類無修正時直接
等於骰值），沒有樣本超過 15。(c) 只有 20 幾筆樣本（DOSBox 手動操作成本高，
未窮舉），樣本量不足以完全排除其他「期望值也是 8」的分佈（例如兩個小骰子相加），
但 `Roll(15)` 是與此引擎既有 RNG 介面（`docs/re/06` 已驗證的 `FUN_1d9f_0e0b(max)`）
最自然、最省一次呼叫的實作方式，且與升級階段「`Roll(N)` 兩次取較大值」的呼叫慣例
（§2.2）一致，判定為合理假設。**沒有找到實際呼叫此骰子的組合語言**（原因見 §1.1
「反組譯撞牆」段落），這點誠實列為假設而非已驗證。

### 1.3 初始 HP／SP／技能點 `[已驗證，PARTY.DAT 位元組直接對帳]`

用兩個乾淨樣本（`TestHP`＝人類遊俠、`TestWiz`＝人類巫師）直接讀存檔位元組：

| 角色 | 種族 | 職業 | 最終屬性（擲骰後）Speed/Str/Int/End/Skill | 初始最大HP（`0xfc`） | 初始SP（`0xea`/`0xfe`/`0xff`） | 技能點（畫面顯示） |
|---|---|---|---|---|---|---|
| TestHP | Human | Ranger | 12/8/11/9/11 | **9** | **0 / 0 / 0** | **11** |
| TestWiz | Human | Wizard | 13/10/9/10/7 | **10** | **9 / 9 / 9** | **9** |

**結論**：

- **初始最大 HP ＝ 耐力（種族修正後最終值），1:1，兩個樣本都精確相等。**`[已驗證]`
- **初始技能點 ＝ 智力，1:1，兩個樣本都精確相等**（`TestHP` 智力 11→技能點 11；
  `TestWiz` 智力 9→技能點 9）。`[已驗證]` 這解答了任務單「初始技能點數＝智力值？」
  的問題：是，且是精確相等，不是某個函式的結果。
- **初始 SP 因職業而異**：巫師拿到「SP＝智力」（跟 HP 公式同構），遊俠拿到 0，
  即使遊俠智力高達 11 也一樣。跟 `docs/walkthrough/part-1.md` 記載的
  「智力決定你適合的職業的初始法力值（SP，例如蠻族一開始沒有 SP，巫師則有）」
  完全吻合。`[已驗證：巫師=智力、遊俠=0；其餘 8 個職業未逐一測試，標為假設，
  推測依「該職業是否天生具備符文/吟唱魔法」決定，跟 `docs/manual/part-1.md`
  各職業說明描述的「魔法親和度」一致]`。

### 1.4 這個公式能不能解釋「人類擲點較優」？`[誠實評估，部分佐證]`

`docs/walkthrough/part-2.md` 攻略原文：「人類的擲骰結果整體就是比較好……五項屬性
加起來擲到 60 到 65 並不難，相較之下我玩其他種族從沒擲出超過 55 的總和」。

用本文找到的公式計算各種族**淨修正**（5 項相加）：

| 種族 | 淨修正 | 期望總分（40 + 淨修正） |
|---|---|---|
| Human | 0 | 40 |
| Elf | −2+2−1 = −1 | 39 |
| Dwarf | −1+1−2 = −2 | 38 |
| Dark Elf | −2+2−1−1 = −2 | 38 |
| Troll | −1−3+2 = −2 | 38 |

**評估**：公式確實預測人類期望總分**嚴格最高**（其餘 4 種族淨修正全部是負值），
方向正確，這是弱佐證。但差距只有 1～2 分，而攻略描述的落差是「人類不難到
60–65、其他種族從沒破 55」——若基礎骰子是 5 個獨立 `Roll(15)`（§1.2 假設），
標準差約 9.66，2 分的期望值差距遠不足以解釋這種尾端機率的懸殊落差
（理論上任一種族衝上 60+、或人類角色從沒破 55，發生機率量級應該接近）。

**本文判斷**：這個公式**只能部分解釋**攻略的觀察，不是強力佐證。懷疑攻略作者
（自述打法是靠人類「刷」到破關等級 12）主要玩人類、對其他種族的取樣次數少很多，
「從沒破 55」更可能是小樣本的主觀印象而非硬性遊戲機制。也不排除還有本文
沒抓到的第二機制（例如種族上限在某個極端情境下介入，但 §1.2 沒有找到擲點階段
套用上限的證據，§2.2 確認上限檢查是**升級**才有的邏輯）。誠實列為
「部分佐證、不完全解釋」，供後續驗證。

### 1.5 職業與技能起始清單 `[已驗證，交叉核對攻略成本表]`

建角選職業後顯示的「可選技能」清單與智力點數成本，跟 `docs/walkthrough/part-3.md`
的 31 技能 × 10 職業成本表**逐項吻合**（本文用 DOSBox 實測 Ranger 與 Wizard 兩個職業）：

- Ranger 顯示：Sword=3, Axe=3, Mace=2, Karate=3, Hunting=1, Monster lore=2
  （對照攻略表 Ranger 欄：Axe=3, Karate=3, Mace=2, Monster Lore=2, Hunting=1, Sword=3——**全部吻合**）
- Wizard 顯示：Fire runes=5, Metal runes=4, Wind runes=5, Ice runes=4, Spirit runes=4,
  Potion lore=3, Item lore=6（對照攻略表 Wizard 欄——**全部吻合**）

這確認手冊「角色剛創建時只能從該職業常見的簡短技能列表中選擇兩項」對應到遊戲內
一張固定的「職業→可選技能子集」清單（本文沒有追出這張清單在記憶體/檔案裡的位址，
因為顯示這個畫面的程式碼跟 §1.1 同一段「未被發現的程式碼區」，列為未解）。

---

## 2. 升級（Level Up，城鎮公會）

### 2.1 觸發函式 `FUN_2aed_07be`（`2aed:07be`，1622 bytes）`[核心已驗證，含指令級證據]`

字串錨定：`Guild.`（`31f0:3bc7`）、`The Guild decides you need %ld / experience before
gaining a level`（`31f0:3bfc`/`3c1b`）、`%d hit points, %d spell points`（`31f0:3c59`）
都在這個函式內。這是城鎮「Town guild」設施的處理函式，反編譯部分損毀（跳表誤判，
同 §1.1 的模式），但**核心的升級計算段落反編譯完全乾淨、且用原始
`disassembly.asm` 逐行核對過（見下），是本文信心最高的一段成果**。

### 2.2 經驗值門檻檢查 `[已驗證機制，門檻表內容未逐一解碼]`

```c
// 讀角色目前經驗值（char+0xc4 起）——注意：讀成兩個 word（0xc4-0xc5, 0xc6-0xc7）
// 合併成 32-bit 值，這跟 docs/formats/game-data-tables.md 記載「EXP 是 3 bytes」
// 不一致，見 §7.3 衝突點
cur_exp = *(uint32*)(char + 0xc4);
need_exp = FUN_2aed_0747(char_idx);   // 內部呼叫 FUN_2aed_0770(char.level) 查表
if (cur_exp < need_exp) {
    print("The Guild decides you need %ld experience before gaining a level");
    break;   // 不升級
}
```

`FUN_2aed_0747`→`FUN_2aed_0770`→`FUN_310e_012d`/`FUN_310e_021e`（310e 段＝軟體浮點
正規化函式庫，`docs/re/06-combat-system.md` §4 已確認同一套函式庫）→`FUN_1990_0a83`，
這條呼叫鏈**已驗證存在、機制是「依角色目前等級（`char+0xf4`）查一張門檻表」**，
但門檻表本身是用浮點運算存取，本文沒有逐一解碼出 20 個等級對應的門檻數值，
**假設**跟 `docs/manual/part-1.md`／`docs/walkthrough/part-1.md` 記載的經驗值表
（300, 700, 1100, 1800……12,752,200）一致，但未獨立驗證，標為假設。

### 2.3 等級遞增 `[已驗證，指令級證據]`

`disassembly.asm`（`2aed:0902`–`0906`）：

```asm
2aed:0902  LES BX,[0x4c7e]              ; ES:BX = party 陣列基底
2aed:0906  INC byte ptr ES:[BX+SI+0xf4] ; char.level += 1   (SI = char_idx * 0x104)
```

**這確認 `char+0xf4` 是角色等級欄位**（`docs/formats/game-data-tables.md` 目前沒有
記載這個欄位，是本文新發現，見 §7.1）。用 DOSBox 建立的新角色 `PARTY.DAT` 位元組
直接讀出 `0xf4=1`（剛建立、1 級），跟 UI 顯示一致，交叉驗證。

### 2.4 HP 成長公式 `[已驗證，指令級證據]`

`disassembly.asm`（`2aed:091b`–`098e`）逐行核對，翻譯成虛擬碼：

```c
N = (endurance * 10) / 17 + 1;      // endurance = char+0xfa，整數除法（IDIV，捨去餘數）
roll1 = Roll(N);
roll2 = Roll(N);
gain = max(roll1, roll2);           // 「擲兩次取較大值」
new_hp = char.max_hp(0xfc) + gain;
if (new_hp > 255) new_hp = 255;     // CMP AX,0xff / JLE / MOV 0xff
char.max_hp(0xfc) = new_hp;
```

對應組合語言關鍵片段（`2aed:091b`–`0941`）：

```asm
MOV AL,[char+0xfa]     ; AL = endurance
SUB AH,AH               ; 零延伸成 word
MOV DX,AX
SHL AX,1                ; *2
SHL AX,1                ; *4
ADD AX,DX                ; +1 => *5
SHL AX,1                 ; *2 => endurance*10
MOV CX,0x11               ; 17
CWD / IDIV CX              ; endurance*10 / 17
INC AX                      ; +1  => N
PUSH AX / CALLF Roll        ; roll1 = Roll(N)
PUSH [N] / CALLF Roll       ; roll2 = Roll(N)
CMP roll2,roll1 / JLE ...    ; 取較大值
```

### 2.5 SP 成長公式 `[已驗證，同一模式的反編譯交叉核對]`

反編譯（`decompiled/2aed_07be_FUN_2aed_07be.c` 行 233–266）結構跟 HP 完全同構，
只是欄位與係數不同：

```c
N = intellect(0xf9) / 2 + 1;        // 整數除法
gain = max(Roll(N), Roll(N));
new_sp_base = min(200, char.sp_base(0xea) + gain);
new_sp_bonus = min(200, char.sp_bonus(0xfe) + gain);   // 同一個 gain，兩個欄位都寫
char.sp_base(0xea) = new_sp_base;
char.sp_bonus(0xfe) = new_sp_bonus;
```

**注意**：HP 只更新一個欄位（`0xfc`，因為 HP 沒有「含裝備加成」的獨立欄位），
SP 更新**兩個**欄位（`0xea` 天生值、`0xfe` 含裝備加成值），上限也不同
（HP 封頂 255＝`0xff`，SP 封頂 200）。「目前 HP/SP」（`0xfd`/`0xff`）**不會**
被升級邏輯觸碰——升級只加最大值，不會順便回滿血/回滿魔（跟一般 CRPG 直覺可能不同，
誠實記錄）。

**這解答了任務單的問題**：「攻略說受耐力/智力影響但感受不明顯」——本文找到的公式
確實用耐力/智力當骰子上限（`endurance*10/17+1`、`intellect/2+1`），但因為擲骰有
「取兩次較大值」的機制、且封頂在 255/200（角色等級 12 封頂時遠遠到不了這個量級），
攻略作者「感受不明顯」合理：耐力每差 1 點，骰子上限只差 `10/17≈0.59`，要差
17 點耐力骰子上限才會整數增加 1，難怪體感不明顯。

### 2.6 屬性隨機分配演算法 `[已驗證，含種族上限重骰邏輯]`

反編譯（同檔案行 280–300、828–979，跳過中間被 decompiler 損毀的段落）：

```c
// 先做一次總量檢查：若角色 5 項屬性(base值)總和 >= 該種族 5 項上限總和，直接跳過分配
attr_sum = char.speed_base + char.str_base + char.intellect + char.endurance + char.skill_base;
cap_sum = sum(race_cap_table[race][0..4]);   // 見下方「未解」
if (attr_sum >= cap_sum) goto 結束（不分配）;

// 分配 3 點，每點：
for (point = 0; point < 3; point++) {
    do {
        attr_idx = Roll(5) - 1;              // [假設：Roll(5) 的 5 未經指令級核對，
                                               //  見下方信心說明] 0..4 對應 Speed/Str/Int/End/Skill
        cur_val = *attr_field[attr_idx];      // 該屬性目前 base 值
        cap_val = race_cap_table[race][attr_idx];
    } while (cur_val == cap_val);             // 已達種族上限 → 重骰換一個屬性
    *attr_field[attr_idx] += 1;
    if (attr_idx in {0(Speed), 1(Strength), 4(Skill)}) {
        *attr_bonus_field[attr_idx] += 1;     // 這三項有獨立的「含裝備加成」欄位，一併+1
    }
    // Intellect(2)/Endurance(3) 沒有獨立 bonus 欄位，上面 base 的 +1 已經足夠
}
```

**跟手冊「五項屬性共加 3 點，隨機分配，已達種族上限者不分到」完全吻合**，
且解出了攻略作者沒寫清楚的**具體演算法**：不是「先排除已達上限的屬性再從剩下的
隨機挑」，而是「隨機挑一個屬性、如果已達上限就整次重骰（可能挑到同一個已達上限的
屬性、也可能換到別的），直到挑中一個沒達上限的屬性才真正加值」——這個 do-while
結構在 5 項屬性大多沒達上限時效率很高，但理論上如果只剩 1 項屬性還沒達上限、
其餘 4 項都封頂，會產生較多次的重骰（但不會無限迴圈，因為前面的
`attr_sum >= cap_sum` 總量檢查已經保證「至少還有 1 項屬性沒封頂」才會進到這個迴圈）。

**`Roll(5)` 的「5」信心等級**：`[假設，中高信心]`。這段程式碼所在的函式反編譯
被 decompiler 嚴重破壞（同一份輸出裡混進了明顯不相關的商店/道具程式碼，判定是
`docs/re/00` 預警的「跳表污染鄰近函式反編譯」的又一個實例），呼叫 `Roll()` 的
實際參數本身在反編譯輸出中被拆成不可信的位元組片段，**沒能用原始
`disassembly.asm` 追出這個特定呼叫的立即數**（`CALLF` 目的地在 raw disassembly
裡顯示的是未重定位的原始段值，跟函式名稱對不起來，見 `docs/re/00` 沒記載、
本文新踩到的這個環境限制，詳見 §7.4）。判定為 `5` 的理由：`switch` 結構明確有
`case 0/1/2/3/4` 五個分支對應 5 項屬性，`Roll(5)-1` 是解釋這個結構最直接的假設，
且跟 §1.2 的擲點慣例（`Roll(n)` 回傳 `[1,n]`，減 1 轉成 0-based 索引）一致。

**種族上限表位置**：`[未解]`。程式碼用 `race*6+attr_idx` 的位移（乘 2 或當
4-byte 存取，寬度不確定）去查一張表，指標運算指向絕對位址 `0x5510`。
直接用位址換算公式回查 `DEMON.INT` 檔案，**這個位移落在檔案範圍之外**
（檔案只有 173,380 bytes，換算後的位移超出檔尾），代表這張表**不是靜態內嵌
在執行檔裡**，很可能是 DOS 載入時保留的額外零初始化記憶體（MZ header 的
extra alloc），在遊戲啟動時由某段初始化程式碼寫入或從別的檔案讀入。本文沒有
追出這段初始化邏輯，種族上限表的實際數值**假設**跟 `docs/walkthrough/part-2.md`
記載的種族屬性上限表一致（人類 20/24/32/22/21……），但未獨立驗證。

### 2.7 公會升級免費 `[已驗證]`

對 `FUN_2aed_07be` 全文（含反編譯與原始碼字串）搜尋金幣／`0x51e`（隊伍金幣欄位，
`docs/formats/game-data-tables.md` §1.5）／`gold`／`Gold` 關鍵字，**零命中**。
整個升級流程從進公會、經驗值門檻檢查、HP/SP/屬性成長，**完全沒有任何扣款的
程式碼路徑**，驗證了手冊「公會服務本身不收費」的說法。

---

## 3. 城鎮經濟 `[大部分未解，列為下一輪]`

### 3.1 已驗證的部分

- **Town guild 升級免費**（§2.7，已驗證）。
- 城鎮設施選單本身（`Healers`／`Rest in the Inn`／`Town guild`／`Church`／`Docks`／
  `Pub`／`Go to marketplace`）由 `FUN_278d_0098` 決定顯示內容，這是
  `docs/re/04-main-loop-state-machine.md` §2.3 已解的部分，本文沒有新增。

### 3.2 撞牆、誠實列為未解的部分

跟 §1.1 同一種模式：用 `strings.csv` 對「Healers」（`31f0:2ff8`）、「Haggle」
（`31f0:37b9`）、「Character possesses superior haggling skills」（`31f0:3f96`）、
「You don't worship %s」（`31f0:32cf`）等城鎮經濟相關字串逐一 grep
`decompiled_all.c`／`disassembly.asm`，**全部零命中**——這些字串存在於
`31f0` 資料段（原始位元組核對過），但沒有任何一條 Ghidra 找到的指令引用它們。

已知的入口點候選（`docs/re/04-main-loop-state-machine.md` §1.3 已列出，本文
沒有新展開）：

| 設施 | 候選函式 | 反編譯狀態 |
|---|---|---|
| 神殿／教堂 | `FUN_278d_0932`（`278d:0932`，宣告 1050 bytes） | 反編譯膨脹到 6598 行，跳表污染嚴重 |
| 角色能力值/技能檢視 | `FUN_2aed_14c2`（`2aed:14c2`，宣告 2563 bytes） | 反編譯膨脹到 5695 行 |
| 市集／商人 | 未定位到具體函式 | 對應字串（`31f0` 段的道具價格提示等）同樣零指令引用 |

**沒有找到的東西**：商店買價/賣價換算公式、Haggle（議價）成功率、
Persuasiveness（說服）技能的加成、商人等級生成規則、治療所收費公式。

### 3.3 已知但非 RE 驗證的規則（照抄手冊，供對照用）

- `docs/manual/part-2.md`「市集」一節：鑑定固定約 75 金幣；出售未鑑定裝備約拿
  「該類型全新品」半價；已裝備的物品不能出售。
- 「治療所」：解毒固定費用；解除束縛費用依法術等級；復活費用依角色等級，
  城鎮復活必定成功且恢復 1 點生命值。
- 「客棧」：住宿恢復速度是野外紮營兩倍（每晚 10 SP、2 HP），房費含餐點。
- `ITEMS.DAT` 的 `price` 欄位（`docs/formats/game-data-tables.md` §3.3，已驗證）
  是道具的基礎售價，但**買價/賣價的實際換算公式（含 Haggle／Persuasiveness
  加成）本文沒有解出**。

---

## 4. 祈禱／神祇機制 `[未解]`

`FUN_278d_0932`（神殿/教堂，見 §3.2）反編譯損毀，本文沒有解出：

- 改宗（變更信奉的司祭/薩滿神祇）的觸發條件與程式碼路徑。
- 神祇好感度（「20% 機率被聽見，每次成功 −5%」，`docs/manual/part-2.md`「神祇呼喚」）
  的實際儲存欄位與遞減公式。
- 賜福觸發判定與效果數值範圍。

手冊記載的規則（非 RE 驗證，僅供對照）：捐獻 1 金幣＝1 經驗值；神殿祈禱恢復
呼喚成功率到 20%；改宗免費（僅需司祭/薩滿技能本身的智力點數成本）。

---

## 5. 時間與休息 `[大部分未解，手冊規則供對照]`

### 5.1 睡眠恢復

`docs/walkthrough/part-1.md`：「睡眠會為隊伍每位成員恢復 1 點 HP、5 點 SP，
前提是你有存糧。如果沒有食物，休息會讓你損失 1 點 HP，但 SP 仍然正常恢復 5 點」；
客棧恢復量加倍（2 HP/10 SP）。**本文沒有找到對應的程式碼**（DOSBox 實測時
`docs/re/01-dosbox-reference.md` 已記錄「Sleep 動作在測試中一直回報 'You are restless'」，
本文沒有進一步排查，跟 §3/§4 同樣的「未被發現程式碼區」問題）。

### 5.2 巨魔再生

`docs/manual/part-1.md`：「再生（Regeneration）讓角色每小時自動回 1 點生命值」。
**完全未查證**——連「遊戲內一小時」對應到哪個記憶體欄位都還沒解出
（`docs/re/04-main-loop-state-machine.md` §4.3 已經誠實記錄「時間推進的三層計數器
wrap 值是 38/35/23，沒有一層等於手冊講的 26 小時一天」，本文延續同一個未解狀態，
巨魔再生機制自然也無從查證）。

### 5.3 狩獵

`docs/manual/part-2.md`：「每位擁有狩獵技能的角色，每天都有機會外出尋找食物一次，
每次找到的份量取決於運氣」。**完全未查證**成功率與收穫量的具體公式。

---

## 6. 已驗證的角色記錄新欄位（供 `docs/formats/game-data-tables.md` 後續合併）

> 本文不修改 `docs/formats/`，這裡只列出**本次反組譯新確認、該檔案目前沒有記載**
> 的欄位，留給負責維護該檔案的人合併：

| 相對位移 | 欄位 | 驗證方式 |
|---|---|---|
| `0xf4` | 角色等級（Level），1 byte | 指令級驗證：`INC byte ptr ES:[BX+SI+0xf4]`（`2aed:0906`），且新建角色存檔讀出 `0xf4=1` |
| `0xc4`–`0xc7` | 經驗值，**可能是 4 bytes 而非 `docs/formats` 記載的 3 bytes** | 見 §7.3，反編譯讀成兩個相鄰 word 合併成 32-bit，未做位元組級交叉驗證，標為觀察非定論 |

---

## 7. 與既有文件的衝突點

### 7.1 `char+0xf4` 欄位語意：新發現，非衝突

`docs/formats/game-data-tables.md` 目前沒有記載 `0xf4` 這個欄位（該文件的欄位表
把 `0xc4` 到 `0x100` 之間標記為「攻略已知的屬性區加上欄位間空隙」，沒有逐一列出
`0xf4`）。本文用指令級證據補上：`0xf4` ＝角色等級。

### 7.2 「裝備欄」起始位移：`0x1a` vs `0xc`，未解決的疑點

`docs/formats/game-data-tables.md` §1.3（已用 `PARTY.DAT`/`PARTY.BAK` diff 交叉驗證）
記載裝備欄是 `0x1a`–`0xc3`（170 bytes，10 slot × 17 bytes），且該文件對 `0x0c`
的判定是「種族欄位」（1 byte，`0xFF`＝未設定，見該文件 §1.2）。

本文在 `FUN_206a_02c7` 的建角初始化程式碼（§1.1）看到一段**從 `char+0xc` 起、
stride 17、寫 10 次 `0xFF`** 的迴圈，且本文用 DOSBox 建立的兩個全新人類角色
（`TestHP`、`TestWiz`）存檔讀出來，`0xc` 欄位都是 `0xFF`（而不是「人類＝種族
索引 0」的 `0x00`）。這跟兩份文件各自的證據都有一定張力：

- 若 `0xc` 真的是「種族」欄位，新建人類角色理論上應該顯示 `0x00`（依
  `docs/formats` §1.2 的種族索引假設），但本文兩個樣本都是 `0xFF`。
- 若 `0xc` 其實是「裝備欄 slot 0」的起點（本文在建角初始化程式碼看到的證據），
  那 `docs/formats` §1.3 用 `PARTY.DAT`/`PARTY.BAK` diff 驗證出的
  「裝備欄從 `0x1a` 開始」就需要重新檢視。

**本文的立場**：兩邊證據都不是決定性的（`docs/formats` 的判定基於既有存檔的
diff 交叉驗證，方法紮實；本文的判定基於全新角色初始化程式碼＋兩個新建角色樣本，
但都是**人類**，沒有測試到非人類種族的 `0xc` 實際數值，見 §8 未解項目）。
**不覆寫既有文件的結論，誠實列出這個疑點，建議下一輪用非人類角色樣本
（本文建立 `TestTroll` 時因操作失誤沒有成功留存位元組資料，見 §8）交叉驗證。**

### 7.3 經驗值欄位寬度：`3 bytes` vs 觀察到的 `4 bytes` 讀取

`docs/formats/game-data-tables.md` §1.2 記載經驗值（`0xc4`）「3 bytes，little-endian」，
依據是攻略原文「跟經驗值一樣是反序儲存」的說法搭配欄位間距推算。

本文在 `FUN_2aed_07be` 的經驗值門檻檢查（§2.2）看到程式碼讀 `char+0xc4` 時是讀
**兩個相鄰的 16-bit word**（`0xc4`-`0xc5` 一個 word、`0xc6`-`0xc7` 另一個 word）
再合併成 32-bit 值比較，這暗示實際欄位寬度可能是 4 bytes（`0xc4`-`0xc7`），
不是 3 bytes（`0xc4`-`0xc6`）。**本文沒有做位元組級的存檔交叉驗證**（沒有建立
一個經驗值超過 3-byte 上限 16,777,215 的角色來檢驗第 4 個 byte 是否真的被使用），
只是反組譯讀取寬度的觀察，標為「觀察，非定論」，供下一輪驗證。

> **✅ 已於 2026-07-25 結案**（`docs/re/19-town-economy.md` §3.2.1）：
> 神殿捐獻的程式碼給出決定性證據 —— EXP 加法是 `ADD word[+0xc4],AX` / `ADC word[+0xc6],DX`
> 的 32-bit 加法，緊接著的封頂檢查是 `CMP word[+0xc6],0xFF` / `CMP word[+0xc4],0xFFFF`。
> **結論：儲存寬度是 4 bytes，數值上限封頂在 `0x00FFFFFF`。** 本文的「4 bytes」觀察
> 與 `docs/formats` 的「3 bytes」各對一半，描述的是同一件事的兩個層面。

### 7.4 環境筆記：`disassembly.asm` 的 `CALLF` 目的地不可直接拿來核對函式名稱

`docs/re/00-ghidra-setup.md` 沒有記載這點，本文分析升級演算法時新踩到：
`workplace/ghidra/export/disassembly.asm` 裡的 `CALLF` 指令，目的地顯示的
segment 值（例如 `CALLF 0x1000:ee6e`、`CALLF 0x2000:e4e7`）**跟 `functions.csv`／
`decompiled_all.c` 裡實際存在的函式位址對不上**（`functions.csv` 完全沒有
`1000:ee6e`、`2000:xxxx` 這類項目）。反覆核對後判定：`disassembly.asm` 匯出時顯示的
是**連結期／未套用某層重定位的原始運算元值**，而 `decompiled_all.c` 的
`FUN_xxxx_yyyy()` 呼叫名稱是 Ghidra 內部**已經正確解析**的符號。**這代表要核對
「這個 CALLF 呼叫的是哪個函式」時，不能只看 `disassembly.asm` 的目的地位址，
要交叉比對 `decompiled_all.c`／`decompiled/*.c` 對同一段位址範圍給出的函式名稱**
（本文因為這個環境細節，多花了不少時間才確認 §2.6 提到的 `Roll(5)` 呼叫本身的
立即數無法從 `disassembly.asm` 單獨核對出來）。建議把這點併入
`docs/re/00-ghidra-setup.md`「踩到的雷」章節（本文依邊界規則不能自己改，
留給下一手）。

---

## 8. 未解項目與下一步建議

按優先度排序：

1. **找到「Choose race」到「屬性擲點」畫面的實際程式碼**（§1.1 撞牆段落）。
   建議：對 `FUN_206a_02c7` 照 `docs/re/00-ghidra-setup.md` 建議的方式，放棄
   decompiler、逐行讀 `disassembly.asm` 對應的位址範圍（`206a:02c7` 起
   802 bytes），找出這個函式裡**沒被 decompiler 正確展開的跳表分派點**——
   跳表本體很可能就藏在這個範圍內，往裡面跳的目的地才是建角流程真正的程式碼。
2. **用非人類種族樣本交叉驗證 §7.2 的 `0xc` 欄位疑點**。本次建立 `TestTroll`
   （巨魔）時，因為 DOSBox 容器一次啟動失敗（`Exit to error: Can't init SDL`）
   導致複製到 `/tmp` 的存檔其實是前一輪殘留的人類角色資料，重跑後又因為
   後續測試呼叫了 `Initialize Party` 覆蓋掉，**沒有留存巨魔角色的位元組資料**。
   下一輪重跑一次「建巨魔角色→立刻複製 PARTY.DAT→讀 `0xc` 欄位」即可補上。
3. **城鎮經濟四大子系統**（商店定價、治療所、客棧、神殿）：`FUN_278d_0932`、
   `FUN_2aed_14c2` 這兩個函式反編譯損毀，照 §8.1 同樣的「放棄 decompiler、
   逐行讀 disassembly」方法應該可行，但預期工作量與 §1.1 相當（每個函式
   都要重新從跳表開始手動展開）。
4. **驗證 §7.3 經驗值欄位寬度**：DOSBox 內想辦法把某角色經驗值刷到超過
   16,777,215（3-byte 上限），存檔後看第 4 個 byte 是否非零。
5. **睡眠/休息、巨魔再生、狩獵**：都需要先解掉 §5.2 提到的「遊戲內時間欄位」
   （`docs/re/04-main-loop-state-machine.md` 已列為未解，非本文範圍），
   建議跟那個未解項目一起處理，不要重複起頭。
6. **種族屬性上限表的實際記憶體/檔案來源**（§2.6）：`0x5510` 這個位址算出來
   落在檔案範圍外，代表是執行期填入的，需要往回追是哪段初始化程式碼寫入
   （或者跟 `MONSTER.DAT`/`ITEMS.DAT` 一樣是另一個外部資料檔，本文没有查
   `DEM_DATA/` 目錄下是否還有沒被解析過的檔案，值得先看一眼檔案清單）。

## 附錄：本文用到的 DOSBox 操作紀錄

實驗過程存的截圖在 `workplace/dosbox/shots/`（不入版控），關鍵幾張：

- `t07-create3.png`／`t08-human-roll.png`：「Choose race:」畫面、人類 Turn#1 擲點結果。
- `e01-elf-roll.png`／`dwarf1.png`／`darkelf1.png`／`troll1.png`：四個非人類種族的
  Turn#1 擲點結果與「種族平均值」方框，§1.2 公式的主要證據來源。
- `t11-after-class.png`：Ranger 技能選擇畫面（「You have 11 points usable」）。
- `wiz-class.png`：Wizard 技能選擇畫面（「You have 9 points usable」），交叉驗證
  技能點＝智力公式在不同角色上都成立。
- `hp-afternamed.png`／後續 `/tmp/party-testhp-human.dat` 等（未入版控，本機暫存）：
  完整建角流程 + 存檔位元組核對，§1.3 HP/SP 公式的證據來源。

重跑方式（範例，建一個人類角色看擲點）：

```bash
cd /home/anr2/cht/daemon_winter
./tools/dosbox_run.sh ega "wait:2.5;key:c;wait:1;key:i;wait:1;key:Return;wait:1;key:Return;wait:1;key:c;wait:1;key:1;wait:1;shot:my-human-roll"
# 讀截圖：workplace/dosbox/shots/my-human-roll.png
```
