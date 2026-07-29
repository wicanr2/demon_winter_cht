# 戰鬥動作與法術效果公式（DEMON.INT 反組譯）

本檔補完 `docs/re/06-combat-system.md` 留下的「7 個戰鬥選單動作 + 35 個法術效果公式」缺口。
分析方法與位址換算沿用 `docs/re/00-ghidra-setup.md`（`file_offset = segment*16 + offset - 0xC400`）、
`docs/re/06` 的既有結論（回合結構、命中/傷害引擎、RNG）。所有結論標示**已驗證**或**假設**；
引用證據附函式位址與關鍵指令/字串。證據來源：`workplace/ghidra/export/`（`disassembly.asm`、
`decompiled_all.c`、`decompiled/*.c`、`strings.csv`、`functions.csv`）以及對
`workplace/orig/demwin/DEMON.INT` 原始位元組的直接讀取核對（`objdump -b binary -m i386
-Maddr16,data16` 搭配自製 Python 位址換算腳本，繞過 Ghidra 對這段程式碼「自動分析未覆蓋」
的區塊）。

---

## 0. 方法論修正（重要，影響本檔與 docs/re/06 §7 的可信度）

**已驗證**：`docs/re/06` §7 對戰鬥選單 `switch` 的 case 編號推測，是直接讀 Ghidra 反編譯出的
C 碼「switch」語句得出的，但這個函式（`FUN_138d_1ef8`）反編譯本身帶有明確警告
（`Control flow encountered bad instruction data`、`Type propagation algorithm not settling`）。
本輪追出該函式在 `138d:25b5` 有一條 `JMP word ptr CS:[BX + 0x258f]`——**這才是真正的跳轉表**，
位於 `138d:258f`（file offset `0x9a5f`），15 個 word 項（case 0–0xe）。直接讀取這 30 bytes
原始位元組核對，得到的 case→handler 位址表**與反編譯 C 碼呈現的 case 分段完全不同**
（反編譯把好幾個真正共用同一 handler 的 case 誤拆成獨立區塊、也把有些 handler 的內容互相
串接錯位）。**結論：`docs/re/06` §7 的 case 編號表（含「case 0xa = Attack」）已被本輪證據推翻，
以下表格取代之。**

驗證片段（可重跑）：

```bash
cd /home/anr2/cht/daemon_winter
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
fo = 0x9a5f  # 138d:258f 換算後的 file offset
import struct
for i,e in enumerate(struct.unpack('<15H', data[fo:fo+30])):
    print(f'case {i:2d}: 138d:{e:04x}')
"
```

---

## 1. 戰鬥選單動作分派表（已驗證，取代 `docs/re/06` §7）

主分派點：`FUN_138d_1ef8`（`138d:1ef8`，玩家回合輸入迴圈）內的
`switch(local_1c[0])`，跳轉表在 `138d:258f`（file `0x9a5f`），**AX 值域 0–14**
（`138d:25ad CMP AX,0xf` / `JNC` 超界時跳到預設處理）。

| case | 目標位址（138d 內或遠呼叫） | 動作 | 信心 | 關鍵證據 |
|---|---|---|---|---|
| 0–3 | `138d:20fb`（共用同一 handler） | **移動／轉向**（不是 8 個具名動作之一） | 已驗證 | 依 case 值設定「目標朝向」，若已面向目標方向直接落入 case 4（前進）；否則更新 `combat_record[slot].facing`(`0x4ec8`) 並呼叫 `FUN_222f_1404`（尋路/畫面更新）。4 個值對應手冊 Return(前進)/→(順轉)/←(逆轉)/(迴轉) 4 個移動鍵 |
| 4 | `138d:2171` | **前進移動** | 已驗證 | 讀 `combat_record[slot].facing` 查方向增量表(`0x15da`/`0x15d2`)，扣 `[0x5190]`(全域點數池，見 §1.2)，寫入新座標 |
| **5** | `138d:23b3` | **Attack（攻擊）** | 已驗證 | file offset `0x98ea` 處是遠呼叫 `9a da 25 8d 03` = `CALLF 138d:25da`，即 `docs/re/06` §2/§3 完整分析過的核心命中/傷害函式 `FUN_138d_25da` |
| **6** | `138d:243c` → 遠呼叫 `138d:4188`(file `0xb658`) | **Cast（施法）** | 已驗證 | `0xb658` 處印出字串 `"Which spell do you wish to cast ?"`(`31f0:0961`)，並呼叫 `FUN_1000_2a53(slot,1)`（任務單已知的法術學校/效果分派函式） |
| **7** | `138d:245f` → 遠呼叫 `17c5:18ab`(file `0xd0fb`) | **Use（使用物品）** | 已驗證 | 印字串 `"Item: "`(`31f0:0b5e`)，呼叫 `FUN_17c5_151d`（物品選單挑選函式，同一函式也在 `FUN_138d_1ef8` 尾端的道具流程被呼叫） |
| **8** | `138d:2482` → 遠呼叫 `17c5:1a65`(file `0xd2b5`) | **Turn Undead（驅散不死）** | 已驗證 | 見 §2.1，字串 `"You have"/"already dispelled"/"ignores the priest"/" dies!"/"They ignore the priest"` |
| **9** | `138d:24ab` | **Dodge（閃避）** | 已驗證 | 見 §2.2，字串 `"  (Dodging)"`(`31f0:0827`) |
| **10 (0xa)** | `138d:24e0` → 遠呼叫 `17c5:1056`(file `0xc8a6`) | **Examine（檢視，「?」鍵）** | 已驗證 | 顯示目標名稱（`0x5194`/`0x5196` 指標表）+ 狀態計數器(`0x4ecc`)對應狀態字串（`0x15ec`/`0x15ee` 表）+ 呼叫 `FUN_222f_1404`，並跳出 3 項子選單（熱鍵字串 `"CBQ"` @ `31f0:0abf`，選項含 `"Continue"` @ `31f0:0adb`），對應手冊「按 ?，可用 →/← 切換角色/怪物，ESC 返回」 |
| **11 (0xb)** | `138d:24f1` | **Sound（音效開關，「S」鍵）** | 已驗證 | 切換全域旗標 `[0x1585]`（0/1），重讀選單字串表 index 11/12（`"Sound off"`/`"Sound on"`，見 §3），呼叫 `FUN_1d9f_29a4(7)`（列 7 = 選單顯示中「Sound off/on」所在列） |
| **12 (0xc)** | `138d:2545` → 遠呼叫 `138d:3ad2`(file `0xafa2`) | **Pray（祈禱，Deity Call，「P」鍵）** | 已驗證 | 見 §2.2，字串 `"doesn't hear you."` / `"hears you!"` |
| **13 (0xd)** | `138d:2567` → 遠呼叫 `17c5:0f2d`(file `0xc77d`) | **Leech（汲取法力，「L」鍵，黑暗精靈）** | 已驗證 | 見 §2.3，字串 `"is drained %d sp"` |
| 14 (0xe) | `138d:2589` | 無實際行為，直接 `mov ax,7; jmp` 到共用結尾 | 已驗證行為，**未定案**用途 | 6 bytes 極簡短 handler，僅回傳常數 7；推測是保留/未使用的 case（ESC「結束回合」很可能不經過這個 switch，而是在更上層的 `FUN_2cdc_033d`/`FUN_2000_d0fd` 讀鍵迴圈中，偵測到 ESC 掃描碼時直接回傳 sentinel，未追完，見 §7 未解問題） |

### 1.1 對 `docs/re/06` 的修正（不要直接改該檔）

- §7 表格全部作廢，以上表取代。**特別是「case 0xa = Attack」是錯的，真正的 Attack 是
  case 5；case 0xa 其實是 Examine。**
- §7 稱「這 8 個選單字串與 switch 沒有在時間內比對」——本輪已比對完成，見上表。
- §1.2 提到 `Move pts` 欄位（`0x4ec2`）「沒有找到明確賦值點」且標為 [假設]。本輪 Leech
  （case 13）的證據顯示：`0x4ec2` 被當作**目標的法力值（SP）**扣減（印出字串
  `"is drained %d sp"`），而不是移動點數。這與 §1.2 的推測衝突，**建議下一輪覆核**：
  `0x4ec2` 可能實際是「當前 SP」，而「本回合剩餘行動點數」是另一個尚未定位的欄位
  （§1.2 案例的 Attack/Dodge 消耗行為可能是消耗 SP 而非移動點數，或者欄位語意隨
  單位種類而不同）。本檔未進一步展開，留給下一輪。

### 1.2 全域 `[0x5190]`：真正的「本回合行動點數池」

**已驗證**：case 4（前進）扣 `[0x5190]`；case 9（閃避）用 `[0x5190]/3` 設定閃避加成
（見 §2.2）；case 6/7/8/12/13 進入前都先檢查 `[0x5190] >= 2` 或 `>= 3`，不足則播放
音效 `FUN_1d9f_2a95(2)` 並回傳 3（動作失敗/无法執行）。這與手冊「移動點數表」
（Return=2、A/C/U/T/P/L=3、→/←//=1）的**消耗量**、以及「D 閃避把剩餘全部點數轉換」
完全吻合，可視為手冊的「Move pts」欄位其實對應這個**全域**變數 `[0x5190]`，而非
`docs/re/06` 猜測的 per-unit 欄位 `0x4ec2`。`[0x5190]` 應該是「目前正在行動的單位剩餘
行動點數」（每次換單位行動時初始化，初始化點在 `FUN_138d_1ef8` 開頭一帶，未逐行追出，
[假設] 標記）。

---

## 2. 七個動作的規則與數值

### 2.1 Turn Undead（驅散不死）

函式：case 8 遠呼叫目標，`17c5:1a65`（file `0xd2b5`，約 340 bytes，未被 Ghidra 認列為
獨立函式，落在 `FUN_17c5_0eff` 結尾之後的分析空隙，用 `objdump` 手動反組譯核對）。

**已驗證的判定流程**：

```c
// 每場戰鬥限一次（per 施法角色）
if (PARTY.DAT[caster].+0xee != 0) {           // 「這場已經驅散過」旗標
    print "You have" + "already dispelled";
    return -5;                                 // 動作取消
}

// 資格檢查：field_4ed4==0xb 且 (+0xd7 或 +0xd8 任一非零)
if (combat[caster].0x4ed4 != 0xb ||
    (PARTY.DAT[caster].+0xd7 == 0 && PARTY.DAT[caster].+0xd8 == 0)) {
    call FUN_17c5_0eff();   // 資格不符的收尾（未展開，推測「你沒有這個技能」一類提示）
    return;
}
PARTY.DAT[caster].+0xee = 1;   // 標記「本場已用」

// 逐一掃描全部 15 個戰鬥槽位，field_4ed6==0xffff 且 HP(0x4eb4 索引)>0 視為「不死怪物」
for slot in 0..14:
    if combat[slot].0x4ed6 == 0xffff and combat[slot].0x4eb4 > 0:
        move_camera_to(slot);           // FUN_138d_1703，鏡頭移動，非判定本身
        roll = RNG(100);
        threshold = (18 * (INT_caster - Level_target) + 18) / 5;   // 見下方推導
        if roll <= threshold:
            print target_name + " dies!";      // 0x31f0:0b93
            combat[slot].HP(0x4eba) = 0;
            combat[slot].status(0x4ec4) = 5;    // 死亡狀態碼
            call FUN_138d_1c94(...);            // 死亡結算/勝負判定（docs/re/06 §1.4 已知函式）
        else:
            print "ignores the priest";         // 0x31f0:0b80
```

**已驗證**：`INT_caster` 讀自 `PARTY.DAT[caster].+0xf9`（一個 byte 欄位，與 §2.3 Leech
的成功率讀取**同一欄位**，兩者都符合手冊「取決於智力」的描述，交叉驗證度高）。
`Level_target` 讀自 `combat[target].0x4ece`——**[假設]**：這個位移在 §3（普通攻擊）
脈絡下被 `docs/re/06` 判讀為「附魔傷害加成」，但那是**玩家攻擊者**視角；這裡的
`target` 是**怪物**，同一個戰鬥記錄位移在怪物槽位上可能儲存不同語意的資料
（很多這類遊戲會把攻擊者/怪物的同一塊記憶體按不同布局複用）。實際門檻算式：

```
threshold = (18 * (INT_caster - target_level) + 18) / 5     （整數除法，向零捨去）
成功條件：RNG(100) <= threshold
```

若沒有任何符合資格的不死怪物，印 `"They ignore the priest"`(`31f0:0b9a`)。

**與手冊的對照**：手冊「成功率取決於怪物等級以及該司祭或薩滿的智力」——**完全吻合**
（智力越高、怪物等級越低，threshold 越大，成功率越高）。「每場戰鬥一次」——
**完全吻合**（`+0xee` 旗標機制）。手冊未提供具體公式係數，本次反組譯是唯一數值來源。

### 2.2 Pray（祈禱，Deity Call）

函式：case 12 遠呼叫目標，`138d:3ad2`（file `0xafa2`，同樣落在
`FUN_138d_34d6` 結尾之後的分析空隙）。

**已驗證的判定流程**：

```c
// 資格檢查同 Turn Undead：field_4ed4==0xb 且 (+0xd7 或 +0xd8 任一非零)
if (combat[caster].0x4ed4 != 0xb ||
    (PARTY.DAT[caster].+0xd7 == 0 && PARTY.DAT[caster].+0xd8 == 0)) {
    call FUN_17c5_0eff();
    return;
}

print deity_name;                        // 神祇名表 0x4c98/0x4c9a，索引 = +0xf0 − 1
roll = RNG(100);
chance = PARTY.DAT[caster].+0xeb;        // 目前祈禱成功率（byte）
if (roll <= chance) {
    print "hears you!";                   // 31f0:0918
    PARTY.DAT[caster].+0xeb -= 5;         // 成功後 -5（永久，跨戰鬥持續遞減）
    return 4;
} else {
    print "doesn't hear you.";            // 31f0:0906
    return 4;
}
```

**已驗證**：成功後固定 **-5** 遞減，讀寫的是 `PARTY.DAT` 的持久欄位（不是戰鬥記錄，
不會隨戰鬥結束重置）。

**初始值 20 已核對到**（原本標為未解）：神殿那一支 `DEMON.INT 0x1c54f` 在寫入
角色記錄 `+0xf0`（信奉的神祇編號）的同一段裡，把 `+0xeb` 設成 `0x14` = 20。
所以手冊的「20% 成功率」是原版真的寫進存檔的初值，配合 -5 遞減，
成功率序列是 20% → 15% → 10% → 5% → 0%。

`print deity_name` 印的是**神祇的名字**（Omizeh、Balmur…共 11 個），不是施法者
姓名 —— 那張 11 條的表與讀取端見 `docs/re/27` §4。所以訊息是
「Omizeh hears you!」這種形式。

### 2.3 Leech（汲取法力，黑暗精靈種族技能）

函式：case 13 遠呼叫目標，`17c5:0f2d`（file `0xc77d`）。

**已驗證的判定流程**：

```c
if ([0x5190] < 3) { play_sound(2); return 3; }   // 行動點數不足

// party_idx = slot - 7（只有玩家才能用，符合「黑暗精靈種族技能」的前提）
if (PARTY.DAT[caster].+0xf5 != 3) {   // [假設] 種族/職業旗標，猜測=「是黑暗精靈」
    call FUN_17c5_0eff();
    return;
}

do {
    target_slot = FUN_138d_3fc9(caster_slot, 0, 2);   // 目標尋路/挑選迭代器（Attack case 也用同一函式）
    if (target_slot == -5) return -5;                  // 取消
} while (target_slot >= 0xf);                           // 找到合法目標(0-14)才跳出

roll = RNG(100);
threshold = 2 * PARTY.DAT[caster].+0xf9;   // INT，同一欄位，見 §2.1
if (roll > threshold) {
    print "Fails!";                         // 31f0:0a97
    return 4;
}

// 成功：目標 SP(0x4ec2) 減半（無條件捨去），並印出實際扣除量
half = (target.0x4ec2 == 1) ? 1 : target.0x4ec2 / 2;   // idiv，==1 時特例扣光最後 1 點
target.0x4ec2 -= half;
print target_name + " is drained " + half + " sp";     // 31f0:0a9e
```

**已驗證**：成功率公式 `RNG(100) <= 2 * INT`，與手冊「取決於黑暗精靈的智力」吻合。
被奪走的量 = 目標**當前**SP 的一半（無條件捨去），且手冊強調「抽到的法力值不會進
你自己口袋」——**已驗證**：程式碼裡確實沒有把扣除量加回施法者的任何欄位，
只有從目標身上扣除，符合手冊描述。

**[假設]**：`+0xf5 == 3` 是否真的等於「種族=黑暗精靈」未直接核對種族表，是依上下文
（黑暗精靈限定技能、且此欄位在別處未見用途）推斷。

### 2.4 Dodge（閃避）

函式：case 9，`138d:24ab`（file `0x997b`，17 bytes 極短）。

**已驗證**：

```c
combat[slot].status_counter(0x4ecc) = [0x5190] / 3;   // 整數除法
print "  (Dodging)";   // 31f0:0827
return 7;
```

`[0x5190]` 是**目前剩餘的全域行動點數池**（見 §1.2），除以 3 存進
`status_counter`（`docs/re/06` §2.1 命中公式已用到這個欄位：
`hit_chance += target.status_counter(0x4ecc) * -4`，即**每點 status_counter 讓攻擊方
命中率 -4**）。合併兩式：

```
閃避加成 = floor([0x5190] / 3)      （status_counter 增量）
命中率修正 = 閃避加成 * -4
```

**與手冊對照**：手冊說「每 3 點移動點數，可讓所有攻擊者對你的命中率降低 -1」。
本次反組譯的係數是 **-4**（`0x4ecc*-4` 的 -4，不是手冊講的 -1）。**這是一個明確的
數值衝突**：可能是手冊/攻略用的是簡化描述，也可能命中率公式本身有额外縮放
（`docs/re/06` §2.1 的命中公式整體是 `skill*4 + 修正項`，其他加成項如朝向也是
±12 這種較大數值，若整個命中率系統本身是以「4 倍」為單位在運算，則「每 3 點
移動點數 -4」除以整體系統的 4 倍縮放，換算成玩家看到的「有效命中率」確實接近
每 3 點 -1。**已驗證的是反組譯內的原始係數 -4；手冊的「-1」很可能是換算後的
玩家可讀說法，兩者本質上一致，不算真衝突**，但寫入 spec 時仍以反組譯的 -4 為準。

### 2.5 Cast（施法）與 Use（使用物品）

**已驗證**：這兩個 case 只是「入口」，把玩家帶到後續的法術/物品選單，不含具體
效果公式本身：

- Cast（case 6）→ `FUN_1000_2a53(slot, 1)`（法術學校選單，任務單已知）→
  `FUN_1000_11e5`（單一效果套用，見 §4）或 `FUN_138d_10bc`（戰鬥中即時套用，見 §4）。
- Use（case 7）→ `FUN_17c5_151d`（道具挑選器，重複呼叫兩次：第一次列出可用道具，
  第二次列出「對誰使用」的目標），物品資料讀自 `+0xc`（武器類型索引，`docs/re/06`
  §3.2 已驗證的欄位）與 `4c7e`/`4c80` 一帶（`ITEMS.DAT` 相關指標）。

**Cast 額外發現（回應任務單「AOE 第一回合不能施放」的驗證需求）**：`FUN_138d_065e`
(`138d:065e`，1263 bytes，法術/怪物 AI 選擇引擎，Ghidra 反編譯本身有警告，可信度中等)
內有這一段：

```c
} while ((*(int *)0x4e2e == 1) && (*(int *)0x518e == 1));
```

`0x4e2e` 是「目前選定法術的效果類型 ID」（見 §4），`0x518e` 是 `docs/re/06` 已驗證
的**回合計數器**。**已驗證**：效果類型 `1`（依 §4 的 K/M 表結構，`1` 很可能對應
「範圍傷害」類——見 §4.3 的推論）在回合計數器 `==1`（第一回合）時，這個 do-while
迴圈條件會強制**不接受**該選擇（重新迴圈），等同「範圍傷害法術第一回合不能選取」。
**這與攻略「範圍傷害法術無法在戰鬥第一回合施放」的說法方向一致**，可視為初步驗證，
但**未逐一核對「5×5 範圍」本身**（沒有在時間內找到範圍套用的巢狀迴圈邊界，
[假設，未解]，見 §7）。

---

## 3. 選單顯示表（輔助佐證，非分派表本身）

**已驗證**（直接讀原始位元組核對，file offset 對照見 §0 換算公式）：戰鬥選單有一份
獨立於 §1 跳轉表的「顯示用」資料結構，位於 `31f0:075e`（13 個遠指標，每個 4 bytes）
與 `31f0:07aa`（熱鍵字元陣列）：

| 顯示列 index | 字串 offset | 文字 | 熱鍵（來自 `31f0:07aa` "WACUTDESPLQWalk"） |
|---|---|---|---|
| 0 | `0x7b5` | Walk | W |
| 1 | `0x7ba` | Attack | A |
| 2 | `0x7c1` | Cast | C |
| 3 | `0x7c6` | Use | U |
| 4 | `0x7ca` | Turn Undead | T |
| 5 | `0x7d6` | Dodge | D |
| 6 | `0x7dc` | Examine | E |
| 7 | `0x7e4` / `0x808` | Sound off / Sound on（依 `[0x1585]` 切換） | S |
| 8 | `0x7ee` | Pray | P |
| 9 | `0x7f3` | Leech | L |
| 10 | `0x7f9` | Quit | Q |
| 11 | `0x7fe` | Sound off（重複項，用途未明） | — |
| 12 | `0x808` | Sound on | — |

**重要**：這份顯示列表的 index（0=Walk、1=Attack…）**不等於** §1 跳轉表的 case 值
（Attack 實際是 case 5，不是 1）。兩者是完全獨立的編號空間——顯示表決定「畫面上
第幾行、對應哪個熱鍵字元」，交給一個通用選單元件（`FUN_2cdc_033d`／
`2000:d0fd`）處理游標與按鍵，該元件回傳的並非「畫面行號」而是**已經轉換過的
case 值**（轉換邏輯應在該通用元件或其呼叫端某處完成，本輪未追出精確的轉換規則，
只確認了兩個編號空間確實不同且各自的內容已驗證，[假設，未解] 轉換規則本身）。

---

## 4. 法術效果公式

### 4.1 法術記錄結構（已驗證）

**已驗證**：每個法術對應一筆 **5 word（10 bytes）** 的記錄，由 `FUN_1000_114f(spell_index)`
從一個學校專屬的 base pointer `[0x4e28]`（遠指標，猜測在選擇法術學校時設定）載入，
寫入 5 個全域變數：

```c
FUN_1000_114f(spell_index):
    base = far_ptr[0x4e28]
    rec  = base + spell_index * 10          // 每筆 10 bytes
    [0x4e2c] = word[rec+0]     // 用途未定案（見下方 case 8/9/10 的個別使用）
    [0x4e2e] = word[rec+2]     // 效果類型 ID（1..0x11 已觀察到的值域）
    [0x4e30] = word[rec+4]     // K：有號係數，正負號決定增益/減損方向
    [0x4e32] = word[rec+6]     // M：除數
    // rec+8（第 5 個 word）在本輪追蹤的函式內未見使用，推測是法術名稱字串指標
```

**證據**：`FUN_1000_114f` 反編譯完整可讀（58 bytes，無警告），5 個全域寫入位置
與 `FUN_1000_11e5`／`FUN_138d_10bc` 讀取這 5 個變數的用法完全對應。

### 4.2 通用 SP → 效果強度換算公式（已驗證，回應任務單核心問題）

**已驗證**，證據來自兩個獨立函式（`FUN_1000_11e5` 的 case 5/0xd、`FUN_138d_10bc`
全函式）用幾乎相同的程式碼結構重複實作：

```c
do {
    K_abs = abs(K);                          // FUN_206a_000a = abs()
    magnitude = RNG( K_abs * SP_invested / M );
} while (magnitude < K_abs * SP_invested / (M * 3));

signed_delta = sign(K) * magnitude;          // FUN_206a_0020 = sign()
```

即：**先算出上限 `K*SP/M`，用 `RNG()` 在 `[1, 上限]` 中擲骰，若擲出的值低於上限的
1/3 就重擲**，等於把亂數分布偏向上限的 **後 2/3 區間**（`[上限/3, 上限]`）。
`K` 的正負號決定這是增益（如治療、加屬性）還是減損（如傷害、降屬性）。

**這正是任務單要找的「SP 投入 → 效果強度」換算式**。以攻略「每點 SP 約 1–3 點效果」
的觀察反推：若某法術的 `K/M ≈ 3`，則上限 `≈ 3*SP`，重擲下限 `≈ SP`，平均落在
`[SP, 3*SP]` 之間、統計期望值約 `2*SP` 左右——與攻略描述的量級吻合。**但本輪未能
逐一取出 35 個法術各自的 `K`、`M` 具體數值**（`[0x4e28]` 指向的每校法術表位址、
以及各校的表格內容本身沒有在時間內定位，[未解]，見 §7）。**因此公式的「結構」
已驗證，但無法給出逐一法術的精確倍率——這是誠實的邊界，不要用猜測填補。**

### 4.3 效果類型 ID（`0x4e2e`）→ 作用欄位對照（已驗證）

證據：`FUN_138d_10bc`（單體戰鬥內效果套用函式，1263 bytes 中的核心段落已讀完），
`docs/re/06` 已驗證的 `combat_record` 欄位表可交叉核對：

| 效果類型 ID | 作用欄位（`combat_record` 基底 `0x4eb4`） | 對照 `docs/re/06` 已驗證欄位語意 | 對應法術類別（推論） |
|---|---|---|---|
| 3 | `0x4eb4 + 9*2 = 0x4ec6` | 技巧(Skill) | 屬性增減：勝利之翼／寒顫／笨拙 |
| 4 | `0x4eb4 + 6*2 = 0x4ec0` | 力量(Strength) | 屬性增減：力量術／衰弱 |
| 5（預設值，未被 3/4/6/7/0xd 覆寫時） | `0x4eb4 + 3*2 = 0x4eba` | HP | **傷害／治療**：烈焰之柱／劍刃術／治療／生命之息 |
| 6 | `0x4eb4 + 2*2 = 0x4eb8` | 速度(Speed) | 屬性增減：羽翼／緩速 |
| 7 | 直接對 `0x4ebe`（不經過索引公式，獨立分支） | 護甲(Armor) | 護甲增減：護甲術／烈焰護盾／寒冰護盾／聖域／鏽蝕護甲 |
| 0xd | `0x4eb4 + 7*2 = 0x4ec2` | SP-like 欄位（見 §1.1 修正） | 法力值相關：法力轉移 一類 |

**已驗證**：`HP` 分支（type 5 預設路徑）有明確的死亡處理——`新HP < 1` 時扣血至 0、
呼叫 `FUN_138d_1c94`（`docs/re/06` §1.4 已驗證的死亡/勝負判定函式）；非 HP 的其他
數值欄位一律 **clamp 到 `[3, 255]`**——**這與攻略「屬性削弱類法術沒辦法把屬性壓到 3
以下」逐字吻合**，是很強的交叉驗證。HP 分支另外會用玩家的「最大 HP」欄位
（`PARTY.DAT[+(-0x620 相對位移，即角色記錄另一處欄位)]`）鉗制治療上限，
不能超過角色 HP 上限。

**護甲（type 7）的實作方式不同**：不經過索引公式，直接
`combat[target].armor(0x4ebe) += signed_delta`，且**沒有看到 clamp**（[假設，未逐一
確認上下界，理論上可能被鏽蝕護甲降到負值，符合攻略「鏽蝕護甲能把護甲值降到負值」
的描述）。

### 4.4 法術分類與公式骨架彙整

依任務單要求的分類，逐項標示解出程度：

**傷害類（烈焰之柱、劍刃術、火焰風暴、冰雹風暴、暴風）**

- 單體傷害（烈焰之柱、劍刃術）：**已驗證公式結構**——`type=5`，`K<0`，
  `magnitude = RNG(|K|*SP/M)`（偏向上限 2/3 區間），`HP -= magnitude`，護甲**不生效**
  （攻略「護甲對此法術無效」——已驗證，因為 §4.3 的 type-5 路徑直接改 HP，
  完全繞過 `docs/re/06` §3.4 那條「`damage -= target.armor`」的**普通攻擊**路徑，
  兩者是不同函式，法術傷害沒有經過扣護甲那一步）。
- 範圍傷害（火焰風暴、冰雹風暴、暴風）：**[未解]**——§2.5 找到「第一回合不能選取
  type=1 法術」的間接證據，但 5×5 範圍套用的迴圈本身（對每個受影響格子重複呼叫
  §4.2 的傷害公式）沒有在時間內定位到，`[0x4e28]` 各校法術表也沒有取出，
  無法確認 type=1 是否就是「範圍傷害」的類型碼（依「回合限制」證據做的推論，
  標記為 **[假設]**）。

**治療類（治療、生命之息）**：**已驗證公式結構**，同 §4.2/§4.3，`type=5`，`K>0`，
`HP += magnitude`，clamp 到角色 HP 上限。「治療至少等於投入 SP」（手冊用語）與
「magnitude 下限 = 上限的 1/3」的關係：若 `K/M` 恰好使 `magnitude` 下限 `≈ SP`，
兩者吻合，但**確切 K/M 數值未取出**，無法逐一驗證每個法術是否剛好滿足這個下限
（[假設]）。

**護甲增減（烈焰護盾、護甲術、寒冰護盾、聖域、鏽蝕護甲）**：**已驗證公式結構**，
`type=7`，直接對 `armor(0x4ebe)` 做有號加減，`K` 正負號決定增甲/減甲。

**屬性增減（力量術、羽翼、勝利之翼、寒顫、緩速、笨拙、衰弱）**：**已驗證公式結構**，
`type∈{3,4,6}` 分別對應技巧/力量/速度，clamp `[3,255]`。

**束縛/解縛（鎖鏈束縛、冰封、凝滯之氣／熔解、掙脫束縛、自由之風）**：**[未解]**。
本輪找到一個疑似候選函式 `FUN_138d_1e72`（每回合狀態衰減檢查，讀 `param_1[5]`
與 `[6]` 兩個 byte，用 `RNG(100) <= (值-100)` 判定「掙脫」），**但沒有時間逐一核對
它是否真的是束縛類法術的判定核心**，也沒有找到攻略提到的「成功率隨 SP 投入倍數
提升」（例如冰封每 9 點一級）的判定式本身。標記為未解，留給下一輪，
**不要用猜測填補**。

**即死類（烈焰打擊、死亡之刃、靈魂折磨、枯萎打擊）**：**[未解]**。§4.1 觀察到的
效果類型清單裡有 `0xc`（"Special powers" 欄位，clamp `[1,4]`，推測與光源類法術
魔法火炬/晶光的「亮度等級」有關，非即死）與 `0x11`（二選一固定閾值表 `0x1c/0x32/0x22`
或 `0x27/0x23/0x2b`，寫入角色欄位 `+0xa1/+0xa2/+0xa3`，依角色某旗標 `+0xbe` 切換；
成功與否由 `FUN_1000_1902` 的回傳值決定，這個函式同時也是 §4.2 每次施法都會呼叫
的「扣 SP」共用函式，這裡的呼叫方式明顯不同——不帶 SP 投入量，只是固定扣一筆
成本後看回傳碼）。**[假設，未定案]**：`0x11` 這條路徑很可能是**復活術**（手冊「最低
25 點、成功率 25%」是全遊戲唯一的「固定機率、不隨 SP 投入變化」的法術），但沒有
找到 RNG(100) 對比 25 的直接證據，只看到二選一的欄位表寫入，可能是復活成功後
設定的「甦醒狀態」相關數值而非判定本身。烈焰打擊/死亡之刃/靈魂折磨/枯萎打擊
這 4 個一擊必殺法術**完全沒有找到判定函式**，[未解]。

**AOE（範圍法術）**：見上方「範圍傷害」小節——「第一回合不可施放」有間接證據
支持；「5×5 範圍」**未驗證**（[未解]）。

---

## 5. 幻術／召喚機制

**部分已驗證**，來自 `FUN_1000_11e5` case 8/9（`0x4e2e ∈ {8,9}`）：

```c
do {
    target_slot = FUN_1000_1877();               // 游標選位置（IJKM+空白鍵，手冊描述的操作方式）
    if (target_slot == sentinel) return 0;         // 取消
    creature_flag = PARTY.DAT[target_slot].+0x102; // 已知欄位：docs/re/06 標為「0=正常,1=中毒」
} while ((creature_flag != 1 && spell_id==0x21) || (creature_flag != 5 && type==8));

deduct_SP();
roll_a = RNG(100);
[0x4e2c] = roll_a;
roll_b = RNG(100);
if (type == 8 && roll_b < roll_a) [0x4e2c] = roll_b;    // 幻術(type 8)取兩次擲骰較小值

if ((SP * K/M) < [0x4e2c]) {
    print fail_message;                                   // "You don't have enough points left" 一類
} else {
    print success_message;
    if (type == 8) {
        PARTY.DAT[target].+0x102 = 0;    // 清除某狀態旗標
        PARTY.DAT[target].+0xfd = 1;     // 設定另一狀態
    }
    if (spell_id == 0x21) PARTY.DAT[target].+0x102 = 0;
    if (spell_id == 0x27) PARTY.DAT[target].+0x102 = 1;
}
```

**已驗證**：幻術（type 8）與召喚（type 9）共用同一套判定骨架，差異在於**幻術會多
擲一次骰取較小值**（`roll_b < roll_a` 才採用 `roll_b`）——這會讓幻術的「失敗機率」
系統性地比召喚更高（兩次取小 ≈ 更容易落在低值區），可能就是攻略描述的「幻術比較
容易半路消失」背後的其中一個機制（但這段只在**施放當下**判定一次成功/失敗，
不是「每回合消失機率」，見下方）。

**未驗證的部分**：

- 召喚出的單位屬性從哪來（`MONSTER.DAT` 索引？）：**[未解]**，`PARTY.DAT[target_slot]`
  在這段程式碼裡被當作「目標」而非「新召喚出的生物記錄」，本輪沒有追到生物屬性
  真正寫入 `combat_record` 新槽位的那一步。
- 幻術每回合消失的機率：**已於 `docs/re/115` 結案**。戰鬥主迴圈
  `138d:026D–02D4` 對 side 3／13 執行 `Roll(10) < 3`；`Roll` 是 1-based，
  因此是輪到行動前 **20%** 消失。舊的 `FUN_138d_065e` 候選方向不正確。
- 「建立當回合不能行動」已由 `docs/re/20` §9.1 後續結案：主迴圈在
  `138d:0033–011A` 先一次性建立 `DS:5152/5154` 行動佇列，召喚函式
  `138d:3161` 是在逐單位行動期間才寫入 12–14 槽，且完全不追加佇列。
  因此新單位自然等到下一回合重建清單，不需要另寫負狀態值。

12 種生物的 SP 成本表（`translations/glossary.md` 第 7 節）本身是文件既有權威資料，
未在本輪重新驗證（不在任務範圍內，該表已與 `docs/manual`/`docs/walkthrough` 交叉核對
一致，見 `docs/walkthrough/part-2.md` 核對結果段落）。

---

## 6. 與手冊/攻略記載的衝突點彙整

| 項目 | 反組譯結果 | 手冊/攻略描述 | 判定 |
|---|---|---|---|
| Dodge 命中率修正係數 | `status_counter * -4` | 「每 3 點移動點數 -1」 | 不算真衝突，見 §2.4，反組譯給出原始係數 -4，手冊可能是換算後的玩家可讀說法 |
| `0x4ec2` 欄位語意 | Leech 證據顯示是「目標 SP」 | `docs/re/06` 猜測是「Move pts」 | **衝突**，建議下一輪覆核，見 §1.1 |
| 復活術判定 | 找到疑似候選（type 0x11）但無法確認是 RNG(100)<25 的直接判定 | 手冊「最低 25 點、成功率 25%」 | 未能驗證一致或衝突，[未解] |
| 束縛法術「SP 投入倍數提升成功率」 | 未找到判定式 | 攻略：每投入一個「束縛等級」（最低成本的倍數）成功率提升 | 未能驗證，[未解] |
| AOE「5×5 範圍」 | 未找到範圍套用迴圈 | 攻略明確描述 5×5 | 未能驗證，[未解] |
| AOE「第一回合不能施放」 | 找到間接證據（`type==1 && round==1` 時強制重選） | 攻略明確描述 | 方向吻合，**間接驗證**（type=1 是否真的等於「範圍傷害」是推論，非直接證據） |

---

## 7. 卡住的地方（誠實列出，供下一輪接續）

1. **`[0x4e28]` 各法術學校資料表的位址與內容**：這是解出全部 35 個法術精確 `K`/`M`
   係數的關鍵，本輪只確認了「有這張表、5-word/筆記錄」的**結構**，沒有找到
   base pointer 實際指向哪裡、也沒有逐校 dump 出內容。建議：從 `FUN_1000_2a53`
   （學校選單）往下追 `[0x4e28]` 的賦值點，應該是選定學校時從一個 per-school
   常數表複製指標。
2. **束縛/解縛法術（鎖鏈束縛/冰封/凝滯之氣）的成功率判定式**：`FUN_138d_1e72`
   是候選但未核對；攻略提到的「SP 投入倍數→成功率」判定邏輯完全沒有定位。
3. **即死類法術（烈焰打擊/死亡之刃/靈魂折磨/枯萎打擊）**：完全沒有找到判定函式，
   連候選都沒有。`枯萎打擊` 由於同時降三個屬性，理論上可能是「連續呼叫三次
   §4.3 的 type 3/4/6 路徑」而非獨立公式，但未驗證。
4. **AOE 範圍套用迴圈（5×5）**：只驗證了「第一回合限制」的間接證據，範圍本身
   （中心點 ± 2 格）的實際迴圈邊界沒有定位。
5. **顯示選單 index → switch case 值的轉換規則**（§3 提到的兩個獨立編號空間）：
   確認了兩者不同，但轉換邏輯（大概率在 `FUN_2cdc_033d`/`2000:d0fd` 內部，或呼叫
   端有一張額外的映射表）沒有追出。這不影響本檔已驗證的 case 行為結論
   （因為 case 值本身是直接從 §0 的跳轉表位元組讀出的 ground truth，不依賴這個
   轉換規則），但會影響「未來要在 Go 引擎重建這個選單 UI」時的實作。
6. **case 14（0xe）與 ESC/End Turn 的關係**：不確定 ESC 鍵是否真的映射到這個
   幾乎是空動作的 case，或是在更上層被攔截處理。
7. **召喚生物屬性來源與幻術消失機率**：見 §5。
8. **`PARTY.DAT[+0xeb]`（Pray 成功率）的初始值與重置時機**：只驗證了 -5 遞減，
   沒找到初始化為 20 的賦值點。

---

## 8. 給 Go 引擎的建議（延續 `docs/re/06` §8）

1. **戰鬥動作分派**：直接照抄 §1 的 case 表——8 個具名動作（Attack/Cast/Use/
   TurnUndead/Dodge/Examine/Pray/Leech）+ 移動（4 個共用 case）+ 音效切換，
   全部已有具體行為可以實作，不需要再等待法術公式完全解出。
2. **已驗證公式可以直接照抄**：Turn Undead（§2.1）、Pray（§2.2）、Leech（§2.3）、
   Dodge（§2.4）都有完整、可執行的整數公式，沒有精度對齊問題。
3. **法術系統先做骨架，公式留可調參數**：§4.2 的「reroll-biased RNG」公式結構、
   §4.3 的效果類型→欄位對照表可以直接實作成引擎架構，但 35 個法術各自的
   `K`/`M` 數值目前**沒有**，建議：
   - 短期：用 `translations/glossary.md` 第 6 節的「最低 SP」與攻略「每點 SP
     約 1–3 點效果」的量級描述，手動估一組合理的 `K`/`M` 預設值（明確標記為
     「暫定，非反組譯數值」），讓遊戲先能玩；
   - 中期：回頭找出 §7.1 的 `[0x4e28]` 表位址，逐校 dump 出真實 `K`/`M`，
     取代暫定值。
4. **束縛/即死/AOE 三類法術**：目前沒有可靠公式，建議先用「近似合理」的暫定
   規則（例如即死類固定機率、束縛類線性隨 SP 提升），並在程式碼與文件中
   明確標記「非反組譯數值，待覆核」，避免未來被誤認為已驗證。
5. **幻術/召喚**：判定成功/失敗的骨架可以照抄 §5，但召喚生物的屬性生成邏輯
   要另外設計（原版資料來源未定位），可以先用 `MONSTER.DAT` 對應生物的
   基礎屬性頂替，待驗證。

---

## 附：可重跑的驗證片段彙整

```bash
cd /home/anr2/cht/daemon_winter

# 1. 戰鬥選單跳轉表（§0/§1 的 ground truth）
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
fo = 0x9a5f
import struct
for i,e in enumerate(struct.unpack('<15H', data[fo:fo+30])):
    print(f'case {i:2d}: 138d:{e:04x}')
"

# 2. Attack 直接呼叫核心命中函式（case 5 的證據）
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
pattern = b'\x9a' + bytes([0xda,0x25]) + bytes([0x8d,0x03])  # far call 038d:25da (pre-reloc)
idx = data.find(pattern)
print('found far-call to FUN_138d_25da at file offset', hex(idx))
"

# 3. Turn Undead / Pray / Leech 關鍵字串核對
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
seg = 0x31f0
for off,label in [(0xb65,'Turn Undead: You have'),(0xb6e,'Turn Undead: already dispelled'),
                   (0xb93,'Turn Undead: dies!'),(0xb9a,'Turn Undead: They ignore the priest'),
                   (0x906,'Pray: doesnt hear you'),(0x918,'Pray: hears you!'),
                   (0xa97,'Leech: Fails!'),(0xa9e,'Leech: is drained sp'),
                   (0x827,'Dodge: (Dodging)'),(0x961,'Cast: which spell')]:
    fo = seg*16+off-0xC400
    print(label, '->', data[fo:fo+40].split(b'\x00')[0])
"

# 4. 用 objdump 手動反組譯 Ghidra 未覆蓋的區塊（16-bit real mode）
objdump -D -b binary -m i386 -Maddr16,data16 \
  --start-address=0xd2b5 --stop-address=0xd500 \
  workplace/orig/demwin/DEMON.INT 2>/dev/null | tail -100

# 5. 法術記錄結構載入函式
cat workplace/ghidra/export/decompiled/1000_114f_FUN_1000_114f.c

# 6. 通用 SP→magnitude 公式（兩處獨立實作互相印證）
cat workplace/ghidra/export/decompiled/1000_11e5_FUN_1000_11e5.c
cat workplace/ghidra/export/decompiled/138d_10bc_FUN_138d_10bc.c
```
