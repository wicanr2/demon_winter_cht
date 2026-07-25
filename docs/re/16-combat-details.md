# 戰鬥系統補完：六項細節（DEMON.INT 反組譯，2026-07-25 跳表修復後複查）

> 對應 `docs/spec/02-combat.md` 的「未解」表。本檔補完命中率修正項、爆擊觸發、
> 戰敗判定呼叫鏈、Use 道具流程、5×5 AOE、召喚生物機制六項。
> 證據來源：`workplace/ghidra/export/`（2026-07-25 跳表修復後重新匯出，`docs/re/12` 記錄的版本）
> 的 `decompiled/*.c`、`disassembly.asm`，以及對 `workplace/orig/demwin/DEMON.INT`
> 原始位元組的直接讀取核對（`objdump -D -b binary -m i386 -Maddr16,data16`）。
> 位址換算沿用 `docs/re/00-ghidra-setup.md`：`file_offset = segment*16 + offset - 0xC400`。
> 全部函式位址、字串位址、位元組樣式皆可用文末「可重跑的驗證片段」重現。

---

## 0. 方法論與範圍

`docs/re/06`、`docs/re/09` 已完整解出戰鬥框架、普通攻擊、Turn Undead/Dodge/Pray/Leech
四個動作、以及法術效果的通式骨架（`magnitude = RNG(K*SP/M)`，重擲偏向上限 2/3 區間）。
本輪任務鎖定的六項細節中，命中/傷害核心函式 `FUN_138d_25da` **不在**
`docs/re/12` 修復的兩張跳表範圍內（那兩張是 `138d:258f` 戰鬥動作分派、
`222f:12ce` 主指令迴圈分派）——`FUN_138d_25da` 本身反編譯乾淨、無警告，
修復前後內容不變，所以 `docs/re/06` §2/§3 對它的分析原本就是可信的靜態結果，
本輪工作是**逐一核對其「假設」標記、且沒有發現需要推翻的地方**，因此第 1、2 項
的結論延續 `docs/re/06`，並標明本輪複核狀態。

第 3、4、5、6 項則是全新追蹤，過程中額外發現一張**第三張、目前仍未修復的跳表**
（`FUN_138d_3c81` 內部，138d:3f95，18 項，依 spell/item 效果類型 `[0x4e2e]` 分派）。
本輪**直接用 Python 讀取原始位元組解出這張表**（不依賴 Ghidra 反編譯，因為
`FUN_138d_3c81` 的反編譯輸出本身仍不可信——見 §4 說明），沿用 `docs/re/00` §5、
`docs/re/12` 建立的「跳表一律回原始指令讀」紀律。這張表的發現直接解開了第 5 項
（5×5 AOE）與部分第 4 項（Use 道具），細節見下。

---

## 1. 命中率的「各項修正」細目

**結論：延續 `docs/re/06` §2.1 的公式，本輪用修復後的乾淨反編譯逐行複核，
內容完全一致，未發現新增修正項或需要推翻的地方。** 主幹「已驗證」，各修正項
的**存在與計算方式已驗證**（直接讀反編譯 C 碼確認，非猜測），但**部分修正項
所依附的欄位語意**（如 `field_4ed6==0x18` 代表什麼技能/buff）仍是 `[假設]`。

函式：`FUN_138d_25da`（`138d:25da`，2185 bytes），無 decompiler 警告，本輪視為
可信來源（不在任何跳表附近，見 §0）。

```c
// 已驗證：主幹與全部修正項的計算方式（欄位語意見下方標註）
hit_chance = attacker.skill(0x4ec6) * 4;                 // 主項，已驗證

// 已驗證：朝向/位置相符加成
if (target.facing(0x4ec8) == attacker.facing(0x4ec8))
    hit_chance += 12;

// 已驗證：近戰武器類攻擊者（field_4ed4 ∈ {2, 0xb}）額外兩項修正
if (attacker.field_4ed4 ∈ {2, 0xb}) {
    hit_chance += attacker.enchant_dmg_bonus(0x4ece) * 3;      // 武器附魔加成
    if (attacker.field_4ed6 == 0x18) {                          // [假設] 特定技能/種族旗標
        hit_chance += 10;
        crit_threshold -= 8;                                    // 見 §2
    }
}

// 已驗證：目標處於「防禦/狀態計數」中時，攻擊方命中率被扣減
if (target.status(0x4ec4) < 2)                             // 目標存活/未失能
    hit_chance += target.status_counter(0x4ecc) * -4;       // 每點 -4（Dodge 也寫這個欄位，見 spec 02-combat.md）

roll = RNG(100);                    // 1..100
hit = (hit_chance >= roll);         // roll <= hit_chance 才算命中；否則 miss（見下方 miss 分支）
```

**與 `docs/re/06` §2.1 逐項比對結果**：完全一致，字面公式、位移常數、分支結構
全部相同。**修正**：`docs/re/06` 原文把這段標為「假設（未逐項動態驗證）」，
本輪沒有跑 DOSBox 動態驗證，所以動態驗證狀態不變，但**靜態層面**（反編譯是否
可信、是否有跳表污染）本輪已用修復後的乾淨匯出重新核對過一次，結論：**不是
跳表污染造成的假象，這段反編譯在修復前後完全相同**，可以放心把「主幹與各修正項
的存在、計算方式」升級為已驗證，僅欄位語意（朝向具體規則、`field_4ed6==0x18`
是什麼、`status_counter` 的完整語意）維持假設。

miss 分支已驗證：印 `misses.`(`31f0:0849`)，依攻擊者槽位 `<7`/`>=7`（怪物/玩家，
`docs/re/06` §1.1 已驗證的槽位配置）選音效 `FUN_1d9f_2a95(1)` 或 `(4)`。

---

## 2. 爆擊觸發條件

**結論：延續 `docs/re/06` §2.2，本輪複核確認公式與觸發時機，無新發現需要推翻。**
已驗證：命中後才擲第二次獨立的 `RNG(100)`，門檻可被兩個條件下修（可疊加）。

```c
crit_threshold = 90;   // 0x5a，基準 10% 爆擊率，已驗證

// 條件一（已驗證存在，欄位語意假設）：
// 近戰武器類攻擊者（field_4ed4 ∈ {2,0xb}）且某張表（見下）該格非零
if (attacker.field_4ed4 ∈ {2, 0xb} &&
    PARTY.DAT_like_table[attacker_slot * 0x104 - 0x64e] != 0)   // [假設，語意不明，見下]
    crit_threshold = 75;   // 0x4b，25%

// 條件二（與 §1 命中率的同一分支共用）：field_4ed6==0x18 時再 -8
if (attacker.field_4ed4 ∈ {2, 0xb} && attacker.field_4ed6 == 0x18)
    crit_threshold -= 8;   // 可與條件一疊加，最低見到 67（33%）

roll2 = RNG(100);
is_critical = (crit_threshold < roll2);   // roll2 落在 (threshold, 100] 才算爆擊
if (is_critical) damage <<= 1;            // 見 §3 傷害公式
```

**新發現（本輪，[假設，未完全解出]）**：條件一查表的定址方式是
`*(char*)((int)*(undefined4*)0x4c7e + attacker_slot * 0x104 - 0x64e)`——
`0x4c7e` 與 `0x104` 的 stride 跟 `PARTY.DAT` 角色記錄陣列的定址方式完全相同
（`docs/re/06` §6 已驗證 `0x104` 是角色記錄大小），但這裡用的索引是
**攻擊者的戰鬥槽位 `attacker_slot`（0-14）直接乘 `0x104`**，不是 `docs/re/09`
其他函式常見的「`slot - 7` 轉成玩家索引 0-7」。這意味著：

- 若 `attacker_slot` 是玩家（7-14），這個定址等於存取 `PARTY.DAT[7..14]`
  的記錄——**超出玩家陣列的合理範圍**（隊伍最多 8 人，索引該是 0-7）。
- 這可能是：(a) `0x4c7e` 在這個查表脈絡下其實指向另一張**不同的表**（剛好
  沿用同一個全域指標變數名稱、但邏輯上是另一張以「戰鬥槽位」而非「玩家索引」
  為鍵的表）；或 (b) 這個分支在實務上只有 `attacker_slot` 恰好對應到某個
  合法範圍時才會被走到（例如某些特殊怪物槽位）。

**未能在本輪釐清這張表的真正身分與 `-0x64e` 這個負偏移的意義**，標記
`[假設，未解]`，留給下一輪。**但爆擊機制本身（存在第二次擲骰、90/75 兩檔基準、
可疊加的 -8 修正）已驗證**，可以直接寫進引擎；只有「25% 檔何時觸發」的精確條件
需要保守處理（可先假設「永不觸發特殊檔」，只用 90 基準 + 已知的 -8 修正，
待欄位語意確認後再補上 75 檔）。

---

## 3. 戰敗判定的確切呼叫點

**結論：已驗證完整呼叫鏈，補完 `docs/re/06` §1.4 留下的缺口。**

### 3.1 核心：`FUN_138d_1d70`（`138d:1d70`，169 bytes，無警告）

這是唯一的勝負判定函式，同時處理「怪物死亡後檢查是否全滅（勝利）」與
「玩家死亡後檢查是否全滅（戰敗）」：

```c
// 已驗證（逐行對照乾淨反編譯）
int FUN_138d_1d70(int dying_slot, int flag) {
    if (combat[dying_slot].field_4ed4 < 10) {
        // 死者是怪物（0-9 範圍）：直接進入「怪物是否全滅」檢查
        goto check_monsters_wiped;
    }
    // 死者是玩家（field_4ed4 >= 10）：呼叫隊伍存活掃描
    result = FUN_138d_1ceb();     // 見 3.2，回傳 3(有人存活) / 2(全滅) / 0(邊界情況)
    // 已驗證：目前追到的呼叫路徑裡 dying_slot 恆為非負，
    // 所以 `dying_slot < 0` 這個子條件在現有呼叫點下恆為 false，
    // 於是這裡直接 return result（不會誤入 check_monsters_wiped）
    return result;

check_monsters_wiped:
    for (i = 0; i < monster_count; i++)          // monster_count 來自 0x4c76+0xa6
        if (combat[i].field_4eb4 != 0 && combat[i].field_4ed4 == 1)  // 存活怪物旗標
            return 3;   // 還有怪物活著，戰鬥繼續
    if (flag == 1) FUN_138d_1e19(1);              // [假設] 經驗值/戰利品相關收尾
    print("All monsters dead");                   // 31f0:073f
    return 1;                                      // 勝利
}
```

### 3.2 `FUN_138d_1ceb`（`138d:1ceb`，23 bytes，無警告）——隊伍存活掃描

```c
// 已驗證，逐行對照
undefined2 FUN_138d_1ceb(void) {
    for (i = 7; i < 7 + party_size; i++)          // party_size 來自 0x4c76+0x9a
        if (combat[i].field_4eb4 != 0) return 3;  // 有玩家「存在」旗標非零 → 戰鬥繼續
    for (i = 7; i < 7 + party_size; i++)
        if (combat[i].status(0x4ec4) < 2) return 0;  // [假設，邊界情況] 極少數不一致狀態
    return 2;                                          // 全部玩家 status>=2 → 隊伍全滅
}
```

### 3.3 呼叫鏈與main loop 的最終分派——已驗證

`FUN_138d_1d70` 由 `FUN_138d_1c94`（`138d:1c94`，死亡結算函式，先呼叫
`FUN_138d_165d(死者槽位)` 做「HP歸零/清地圖格/status=5」的死亡bookkeeping，
再呼叫 `FUN_138d_1d70`）呼叫。**`FUN_138d_1c94` 本身雖被 Ghidra 反編譯宣告成
`void`，但其呼叫端一致把它的回傳值當數值使用**（`uVar5 = FUN_138d_1c94(...)`），
這是本反組譯專案已知的「far call 隱式用 AX 傳回傳值，decompiler 沒抓到形式化
`return` 陳述式」現象——**`[假設，依常見呼叫慣例推斷，未逐位元組核對 AX
在函式尾端確實留著 `FUN_138d_1d70` 的結果]**，但下面的字串交叉驗證（3.4）
強烈支持這個推斷成立。

這個回傳值沿著 `FUN_138d_1c94` → `FUN_138d_25da`（攻擊核心）→ 攻擊路徑
→ 一路傳回主迴圈 `FUN_138d_0002` 的 `local_14`，主迴圈依此值分派：

```c
// FUN_138d_0002 主迴圈，已驗證（節錄）
if (local_14 == 0 || local_14 == 1) { FUN_1990_0aeb(local_14); return; }
if (local_14 == 2) { FUN_25be_000c(); return; }
if (local_14 == 4) { /* 重置鏡頭，繼續 */ }
// 其他值（含 3）：繼續下一個單位
```

### 3.4 三種結局的字串交叉驗證——已驗證（本輪新查證，糾正一個直覺誤判）

用原始位元組核對 `FUN_1990_0aeb`／`FUN_25be_000c` 內印出的字串，確認三種結局
分別對應哪個數值：

| `local_14` 值 | 呼叫 | 印出訊息（原始位元組核對） | 結局 |
|---|---|---|---|
| `1` | `FUN_1990_0aeb(1)` | 進入完整的經驗值/金幣/戰利品分配邏輯 | **勝利** |
| `0` | `FUN_1990_0aeb(0)` | `31f0:0bb2` = `"You have run"` | **逃跑**（不是戰敗！見下方警語） |
| `2` | `FUN_25be_000c()` | `31f0:274e/275c/276e/2780` = `"A cold breeze" "chills the air..." "...all characters" "have died."` | **戰敗** |

**⚠ 重要警語（本輪修正一個直覺誤判）**：一開始直覺會把 `local_14==0`
當成戰敗（因為 `FUN_138d_1ceb` 回傳 `2` 代表「全滅」，而 `0`/`1` 在主迴圈
共用同一個分支），但**逐位元組核對字串後發現 `local_14==0` 印的是
「你逃跑了」而非戰敗訊息**——真正的戰敗訊息（"A cold breeze chills the air...
all characters have died."）掛在 `local_14==2` 這個分支，與 `FUN_138d_1ceb`
回傳的「隊伍全滅」值 `2` **數值上完全吻合**（`FUN_138d_1d70` 玩家死亡分支
直接 `return FUN_138d_1ceb()` 的結果，`2` 原封不動往上傳）。

**完整鏈路（已驗證）**：玩家死亡 → `FUN_138d_1c94` → `FUN_138d_1d70`
→ `FUN_138d_1ceb()==2`（全部玩家 `status>=2`）→ 沿呼叫鏈回傳 `2`
→ 主迴圈 `local_14==2` → `FUN_25be_000c()` → 印「A cold breeze chills the
air... all characters have died.」→ 戰敗結束。

**已知邊界（[假設，未解]）**：`FUN_138d_1ceb` 回傳 `0` 的情況（極少數：
沒人 `field_4eb4!=0` 但仍有人 `status<2`）目前推得的行為是「當成 local_14==0
處理」，也就是印出「You have run」的逃跑訊息——這看起來語意矛盾（不是玩家
主動逃跑卻印逃跑訊息），研判是原版沒有處理到的極邊界狀況（可能實務上永遠不會
發生），不建議在 Go 引擎刻意重現這個矛盾行為，開發時應直接把這個分支視為
「不可達」防禦性程式碼。

**AOE 相關的行為缺口（[已驗證，見 §5]）**：透過 5×5 AOE（`FUN_138d_134d`）
殺死目標時，呼叫的是 `FUN_138d_165d`（純粹的死亡 bookkeeping），**不是**
`FUN_138d_1c94`——也就是說 **AOE 造成的擊殺不會立即觸發 `FUN_138d_1d70`
勝負判定**。這代表如果一發範圍法術同時清光最後幾隻怪物（或理論上波及到
最後幾個玩家），戰鬥不會在那個瞬間結束，要等到下一次有單位透過一般攻擊路徑
死亡、或某個週期性檢查點才會偵測到。**這是原版遊戲行為本身的特性（已驗證
存在），不是本輪分析的推測**——是否要在 Go 引擎忠實重現這個「AOE 擊殺不會
立即結束戰鬥」的行為，還是視為原版疏漏、順手修正，建議留給協調者裁決。

---

## 4. `Use` 道具（case 7）的處理

**結論：分派、目標選取、可用道具過濾邏輯已驗證；道具效果套用的深層邏輯
部分已驗證（透過 §5 的共用效果引擎）、部分因反編譯不可信而未解。**

函式：`FUN_17c5_18ab`（`17c5:18ab`），**乾淨、無警告**，可信。

### 4.1 前置檢查（已驗證）

```c
if ([0x5190] < 3 || combat[caster].field_4ed4 != 0xb)   // 行動點不足 或 非玩家
    { play_sound(2); return 3; }                          // 動作失敗
```

### 4.2 選道具、選目標（已驗證）

```c
FUN_17c5_151d(char_idx, 2, 0);          // 建立/整理該角色的可用道具清單
print("Item: ");                        // 31f0:0b5e
do {
    item_slot = FUN_17c5_151d(char_idx, 2, 2);   // 玩家挑選道具（返回道具欄索引）
    if (item_slot == 10) return -5;               // 取消（ESC 一類）
} while (item_slot == 0xffff);                     // 無效選擇，重選

item = PARTY.DAT[char_idx].inventory[item_slot];   // 每格 17 bytes，見 docs/re/06 §6
```

### 4.3 可用道具過濾規則（已驗證，本輪新解出，補 `docs/re/06`/`09` 未涵蓋的內容）

道具清單會跳過不合資格的項目，重新選取，直到選到合法項目：

```c
// 已驗證：do-while 條件為「跳過（continue）」的判斷
skip = (item.type > 0xfd)                              // 空格
    || (item.usable_flag == 0)                          // 不可用旗標
    || (item.charges_field_0x11 == item.charges_field_0x12)  // [假設，語意不明]
    || (item.type < 8  && item_slot != PARTY.DAT[char_idx].equipped_weapon_idx)  // 武器類
    || (item.type >= 8 && item.type < 0xd
        && item_slot != PARTY.DAT[char_idx].equipped_armor_idx);                 // 護甲類
```

**已驗證的規則語意**：武器類道具（type 0-7）與護甲類道具（type 8-12）**只有
在「目前已裝備該道具」的情況下才會出現在 Use 選單裡**；一般消耗品（type ≥ 0xd，
藥水/卷軸一類）不受此限制，永遠可選。這與已知遊戲設計吻合——「Use」是拿來
觸發**已裝備**武器/護甲的特殊能力（如附魔武器的主動技），而不是拿未裝備的武器
當道具用；真正的消耗品則不受限制。

### 4.4 效果套用的分派（已驗證框架，深層邏輯部分未解）

```c
FUN_1000_114f(item.effect_index);      // 載入 5-word 效果記錄到 [0x4e2c..0x4e32]
                                         // （與 docs/re/09 §4.1 完全相同的載入函式，
                                         //  證實道具的效果也走同一套「效果記錄」機制）
if (item.type ∈ [8,0xc] 或 item.type ∈ {0x18, 0x19, 0xe})
    result = FUN_138d_2e63(caster, target_hint, cursor_x, cursor_y, magnitude_param, effect_idx, 1);
else
    result = FUN_138d_3c81(caster, magnitude_param, effect_idx, 1);
```

**`FUN_138d_3c81`（消耗品/一般道具路徑）——已驗證框架，內部有一張未修復的
第三張跳表**：

這個函式一開始就 `mov ax,[0x4e2e]; jmp <switch>`，跳表在 `138d:3f95`
（18 項，索引即效果類型 `0x4e2e`）。**本輪直接用 Python 讀出這張表**（不依賴
Ghidra，因為 `docs/re/12` 只修復了另外兩張已知表，這張沒被處理，反編譯輸出
不可信）：

```
effect_type  0 -> 138d:3fc6   （default）
effect_type  1 -> 138d:3c8d   （5×5 AOE，見 §5）
effect_type  2 -> 138d:3cc5   （未查）
effect_type  3-7,0xd -> 138d:3cff  （單體屬性/HP/SP 效果，見下）
effect_type  8,0xc,0x11 -> 138d:3d98  （未完全查，見 §6）
effect_type  9 -> 138d:3dcf   （未完全查，見 §6）
effect_type 10 -> 138d:3e25   （未查）
effect_type 11 -> 138d:3e7f   （未查）
effect_type 14 -> 138d:3f0f   （未查）
effect_type 15 -> 138d:3f48   （未查）
effect_type 16 -> 138d:3f5c   （未查）
```

`effect_type ∈ {3,4,5,6,7,0xd}` 共用的 handler（`138d:3cff`）**已驗證**：
先呼叫 `FUN_138d_3fc9` 挑一個單一目標（與 Leech 使用同一個目標挑選函式），
接著有一個**新發現的免疫/抵抗分支**：

```c
// 已驗證（原始位元組核對）：單體版本的「抵抗」判定
if (effect_type != 7 &&           // 護甲類效果不受此限
    spell_school_id(0x4e2c) == 4) {                    // [假設] 某個「屬性/系別」== 4
    if (target.field_4ed6 == 7 || target.field_4ed6 == 10) {   // [假設] 目標種族/類型旗標
        print("Spell fails");      // 31f0:19aa
        FUN_138d_1e19(sp_cost_param);
        return;                     // 抵抗，效果不套用
    }
}
// 否則落入與 docs/re/09 §4.2/§4.3 完全相同的
// magnitude = RNG(K*SP/M)（reroll 偏向上限 2/3）+ 依 effect_type 選欄位套用
```

這條「`spell_school_id==4` 且目標種族 `∈{7,10}` → 抵抗」規則與 §5（AOE 迴圈）
裡發現的同一條規則**完全吻合**（兩個獨立函式各自實作了一次，互相印證），
可視為高信心結論，但 `spell_school_id==4` 具體代表哪個屬性系（冷屬性？
對不死系？）與 `field_4ed6∈{7,10}` 具體代表哪個種族仍是 `[假設]`。

**`FUN_138d_2e63`（護甲類/特殊道具路徑）——本輪判定反編譯不可信，未展開**：

這個函式（1728 行反編譯、`Control flow encountered bad instruction data`、
`Removing unreachable block` 等明確警告）本身也是一個以 `[0x4e2e]` 為索引的
17 項 switch，**同樣缺少跳表修復**，且比 `FUN_138d_3c81` 更龐大混亂
（混雜了戰鬥外的功能，例如其中一個 case 直接呼叫 `FUN_138d_0002` 重啟整場
戰鬥、另一個 case 處理隊伍逃跑後的地圖恢復）。函式尾端有一段乾淨、可信的
「套用傷害、扣護甲、判定死亡」邏輯，與 `FUN_138d_25da`（§1-§3）末段結構
逐字相同——這是它的「default」case（找不到特殊效果類型時的一般傷害路徑）。
**本輪未能在時間內修復它的跳表並逐一核對其餘 16 個 case**，這是本輪最大的
未解缺口，標記 `[未解]`，建議下一輪比照 `docs/re/12` 的
`AnnotateJumpTables.java` 模式修復（跳表位址：函式起點附近讀 `[0x4e2e]`
後的 `jmp` 指令，索引範圍 0-0x10）。

**已驗證但值得留意**：`FUN_138d_2e63` 也被 `FUN_138d_25da`（一般攻擊，14%
機率的「追加攻擊」分支）呼叫，所以它不只是 Use 道具專用，也是「多重攻擊」
機制的一部分，兩者共用同一段程式碼。

---

## 5. 5×5 AOE 的實作

**結論：已驗證完整迴圈、範圍判定、第一回合限制。這是本輪最主要的新發現，
直接解開 `docs/re/06`/`09` 都沒找到的迴圈本體。**

### 5.1 呼叫鏈（已驗證）

`FUN_138d_3c81` 的 `effect_type==1` handler（`138d:3c8d`）：

```c
// 已驗證，原始位元組逐行核對
target = FUN_138d_3fc9(caster, ...);      // 挑一個中心座標（螢幕游標選點，
                                            // 副作用：更新全域游標 [0x50f0]/[0x50ee]）
if (target == -5) return -5;               // 取消
FUN_138d_134d(caster, [0x50ee], [0x50f0], sp_invested);   // 見 5.2
```

`FUN_138d_134d`（`138d:134d`）**另外還被兩處呼叫**（本輪用原始 `CALLF`
位元組樣式 `9a 4d 13 8d 03` 全檔掃描確認，僅 3 處命中）：

- `FUN_138d_065e`（怪物/法術 AI 選擇引擎，`docs/re/09` §2.5 已知）
- `FUN_138d_2e63`（§4.4 的護甲類道具路徑／多重攻擊路徑）

**這證實 5×5 AOE 是一套共用機制，不論觸發來源是玩家 Cast、玩家 Use 道具、
或怪物 AI 施法，都收斂到同一個 `FUN_138d_134d`**——跟普通攻擊的
`FUN_1990_0002`→`FUN_138d_25da` 共用模式（`docs/re/06` §7 已驗證）是同一種
設計慣例。

### 5.2 `FUN_138d_134d` 主體——已驗證，無警告，可直接照抄

```c
// 已驗證：視覺高亮 5×5 範圍（不影響邏輯，僅畫面提示）
for (dy = -2; dy <= 2; dy++)
    for (dx = -2; dx <= 2; dx++)
        highlight_tile(centerX+dx, centerY+dy);   // 純顯示

// 已驗證：真正的效果套用——用「掃全部 15 個戰鬥槽位、篩選是否落在框內」
// 實作 5×5，不是逐格套用
for (slot = 0; slot < 15; slot++) {
    if (abs(combat[slot].x - centerX) >= 3) continue;   // |dx| > 2 → 不在範圍內
    if (abs(combat[slot].y - centerY) >= 3) continue;   // |dy| > 2 → 不在範圍內
    // 走到這裡代表 slot 落在「中心點 ±2 格」的 5×5 方框內（含中心本身）

    magnitude = RNG(K(0x4e30) * SP_invested / M(0x4e32));      // 已驗證，與 §4 單體版本
    while (magnitude < K * SP_invested / (M * 3)) {             // 同一條 reroll 公式，
        magnitude = RNG(K * SP_invested / M);                   // 三個獨立函式互相印證
    }

    // 已驗證：與 §4.4 完全相同的抵抗規則（本輪新發現，兩處各自實作一次）
    if (spell_school_id(0x4e2c) == 4 &&
        target.field_4ed6 ∈ {7, 10})
        magnitude = 0;

    if (combat[slot].hp > magnitude) {
        combat[slot].hp -= magnitude;
    } else {
        print(target_name + " dies!");        // " %s dies!"，0x6fa
        play_sound(8);
        FUN_138d_165d(slot);                   // 死亡 bookkeeping（不含勝負判定，見 §3.4 警語）
    }
}
```

**已驗證的兩個關鍵事實**（回應任務單「驗證這兩點」的要求）：

1. **範圍確實是 5×5，以施法者選定的座標為中心，`±2` 格（Chebyshev 距離）**——
   透過「逐一檢查全部 15 個戰鬥單位是否落在框內」實作，不是逐格掃描地圖。
   **框的大小是寫死的常數（`-2..2`），不隨法術威力/SP 投入而變化**——
   所有範圍法術共用同一個 5×5 大小，威力只影響 magnitude（傷害/治療量），
   不影響範圍。
2. **AOE 對「戰鬥單位」生效，不分敵我**——迴圈掃描全部 15 個槽位（怪物與
   玩家都在內），只要座標落在框內都會被套用效果（含正負號，`K` 決定增益/
   傷害方向）。範圍法術理論上可能誤傷己方，原版沒有排除隊友。

### 5.3 第一回合限制——延續 `docs/re/09` §2.5，本輪未再深入

`docs/re/09` §2.5 已用 `FUN_138d_065e` 內的
`while ((*(int*)0x4e2e == 1) && (*(int*)0x518e == 1))`（回合計數器 `0x518e`
第一回合時強制拒絕選取 `effect_type==1`）建立了間接證據。**本輪透過 §4.4
的跳表直接解出 `effect_type==1` 對應的正是這裡的 AOE handler**，兩者互相
印證，**升級為已驗證**：`effect_type==1` 確實就是範圍效果的類型碼，
且第一回合（`0x518e==1`）無法選取這個類型的效果——與攻略「範圍傷害法術無法
在戰鬥第一回合施放」逐字吻合。

---

## 6. 召喚生物的屬性來源

**結論：仍未解出。本輪順著 `FUN_138d_3c81` 的跳表追了 `effect_type 8/9`
（`docs/re/09` §5 認定為幻術/召喚），但發現的內容與「屬性來源」無直接關聯，
如實記錄供下一輪參考,不勉強拼湊結論。**

`FUN_138d_3c81` 的 `effect_type==8` handler（`138d:3d98`）與 `effect_type==9`
handler（`138d:3dcf`）已用原始位元組讀出：

```c
// 已驗證（原始位元組核對），但語意不明確，見下方警語
// effect_type==8：
if (sub_mode(bp+0xc) == 3 && spell_school_id(0x4e2c) == 0x11)
    goto shared_with_type9;
if (sub_mode(bp+0xc) == 3) {
    combat[target].sp_field(0x4ec2) += value;   // 直接加值到某個「類SP」欄位
} else {
    FUN_138d_2f7e(...);   // 未展開
}

// effect_type==9（與部分 type==8 共用）：
shared_with_type9:
target = FUN_138d_3fc9(...);          // 挑目標
if (target == -5) return -5;
while (target >= 15) target = FUN_138d_3fc9(...);   // 重選
if (combat[target].field_4ed4 < 0xa)  // 目標必須是「玩家」（field_4ed4>=10）
    goto shared_with_type9;            // 否則重選
FUN_138d_2fa0(target, ...);            // 未展開
```

**這與「召喚一隻新怪物到場上」的直覺語意不符**——這裡的 `effect_type==8/9`
似乎要求「目標必須是玩家」（`field_4ed4>=0xa`），比較像是套用在**既有隊友**
身上的效果（例如召喚一個「幻影分身」疊加在某玩家身上、或某種友方增益），
而不是在空戰鬥槽位建立一個新的怪物單位。

**沒有找到**：

- `MONSTER.DAT` 索引在這兩個 handler 裡完全沒有出現任何引用痕跡。
- 新戰鬥單位（新的 `combat_record` 槽位）被寫入屬性的那一步——`FUN_138d_2f7e`、
  `FUN_138d_2fa0` 兩個被呼叫但未展開的函式最可能是真正做這件事的地方，
  本輪沒有時間深入（`FUN_138d_2fa0` 是個約 30-40 bytes 的短函式，值得下一輪
  優先追）。
- 幻術每回合消失的機率、召喚單位「當回合不能行動」的具體實作——延續
  `docs/re/09` §5 的 `[未解]` 狀態，本輪沒有新進展。

**與 `docs/re/09` §5 的關係**：`docs/re/09` §5 分析的是 `FUN_1000_11e5`
（Cast 路徑的效果套用函式）裡的 `effect_type∈{8,9}`，本輪分析的是
`FUN_138d_3c81`（Use 道具/怪物 AI 路徑的效果套用函式）裡**同樣編號**的
`effect_type∈{8,9}`——**兩者的行為看起來不完全一致**（`docs/re/09` §5 描述
的是「幻術/召喚共用骨架、兩次擲骰取小值」，本輪看到的是「限定玩家目標、
呼叫兩個未展開的子函式」），**這可能是同一組效果類型碼在不同觸發情境下
語意不同（如已知的欄位重用模式），也可能是本輪讀碼不夠深入所以看起來不一致
——如實記錄此落差,不擅自判定哪個對,留給下一輪或協調者裁決。**

---

## 7. 完整公式彙整（給 Go 引擎）

### 7.1 命中率（已驗證框架，欄位語意部分假設）

```
hit_chance = 攻擊者.技巧 * 4
           + (朝向相符 ? 12 : 0)
           + (近戰武器類 ? 附魔加成*3 + (特定旗標==0x18 ? 10 : 0) : 0)
           + (目標未失能 ? 目標.status_counter * -4 : 0)

命中 = RNG(100) <= hit_chance
```

### 7.2 爆擊（已驗證框架，25% 檔觸發條件未解，建議先只做 90/82 兩檔）

```
crit_threshold = 90
             - (近戰武器類 && 特定旗標==0x18 ? 8 : 0)
             （25% 檔：近戰武器類 && 某未解欄位非零 → 改用 75 為基準，再套用上面 -8）

爆擊 = RNG(100) > crit_threshold
```

### 7.3 戰敗/勝利/逃跑判定（已驗證）

```
死者是怪物 且 場上再無存活怪物（field_4ed4==1 者）→ 勝利，local_14=1
死者是玩家 且 全部玩家 status>=2               → 戰敗，local_14=2
其餘：戰鬥繼續（local_14=3）

主迴圈：
  local_14==1 → 完整戰利品/經驗值結算 → 勝利畫面
  local_14==2 → 「A cold breeze chills the air... all characters have died.」→ 戰敗畫面
  local_14==0 → 「You have run」→ 逃跑畫面（與戰敗不同！）
```

⚠ **AOE 擊殺不會立即觸發這個判定**（見 §3.4），Go 引擎需要決定是否忠實重現
這個原版行為缺口。

### 7.4 Use 道具（已驗證框架 + 部分未解）

```
前置：行動點>=3 且 施法者是玩家
選道具：只列出「已裝備的武器/護甲」或「任意消耗品(type>=0xd)」
效果套用：載入道具內嵌的效果記錄（K/M/effect_type，與法術共用同一套結構）
  → type∈[8,0xc]或{0x18,0x19,0xe}：走 FUN_138d_2e63（[未解，跳表未修復]）
  → 其他：走 FUN_138d_3c81 的 18 項效果分派（第 5 項已驗證，其餘見 §4.4 表）
```

### 7.5 5×5 AOE（已驗證）

```
中心 = 玩家/AI 選定座標
範圍 = 中心 ± 2 格（Chebyshev 距離，寫死常數，不隨法術威力縮放）
對範圍內全部戰鬥單位（不分敵我）：
    magnitude = RNG(K*SP/M)，若 < 上限的1/3 則重擲
    若「屬性系==4」且「目標種族∈{7,10}」→ magnitude = 0（抵抗）
    HP -= magnitude；HP<=0 則死亡
第一回合（round==1）無法選取此類效果
```

### 7.6 召喚生物（未解）

無可用公式，維持 `docs/re/09` §5 的暫定建議（先用 `MONSTER.DAT` 對應生物的
基礎屬性頂替，待驗證）。

---

## 8. 與既有文件的衝突/補充彙整（供協調者裁決）

| 項目 | `docs/re/06`/`09` 原狀態 | 本輪發現 | 判定 |
|---|---|---|---|
| 隊伍全滅判定呼叫點 | 未找到 | 已驗證完整鏈路（§3） | 補完，非衝突 |
| `local_14==0` 是否為戰敗 | 未討論 | **不是**，是逃跑（"You have run"） | 新發現，提醒下一輪別把 0/1/2 三者的语意搞混 |
| 5×5 AOE 迴圈 | 未找到 | 已驗證（§5），範圍為寫死常數 `±2`，不隨威力縮放 | 補完，非衝突 |
| AOE 抵抗規則（屬性系==4 對 種族∈{7,10}） | 未提及 | 已驗證存在，兩個獨立函式互相印證 | 新發現 |
| effect_type==1 是否為範圍效果 | `docs/re/09` §4.4 標為 `[假設]`（間接證據） | 本輪透過跳表直接證實 | **升級為已驗證** |
| Use 道具的武器/護甲類過濾規則（只能用已裝備的） | 未提及 | 已驗證 | 新發現 |
| `effect_type 8/9` 在 Use/AI 路徑 vs Cast 路徑的行為 | `docs/re/09` §5 只分析了 Cast 路徑 | 本輪發現 Use/AI 路徑的同編號行為明顯不同（限定玩家目標） | **落差，未判定哪個對，列出供裁決** |
| AOE 擊殺不會立即觸發勝負判定 | 未提及 | 已驗證 | 新發現，行為缺口，需要裁決是否忠實重現 |
| 爆擊 25% 檔觸發條件 | `docs/re/06` 標為欄位語意未查明 | 本輪定位到具體查表位址（`0x4c7e + slot*0x104 - 0x64e`），但索引方式疑似與已知 PARTY.DAT 定址慣例不一致 | 未解，新增細節，未推翻既有結論 |

---

## 9. 未解項目與下一步

1. **`FUN_138d_2e63` 的跳表未修復**：17 項效果（`[0x4e2e]` 0-0x10），
   反編譯完全不可信（`Control flow encountered bad instruction data` 等
   明確警告）。這是 Use 道具（護甲類）與多重攻擊機制共用的函式，優先度高。
   建議比照 `docs/re/12` 的 `AnnotateJumpTables.java` 模式處理，跳表位址
   約在函式開頭附近（讀 `[0x4e2e]` 後緊接的間接 `jmp`）。
2. **爆擊 25% 檔的觸發欄位**（`0x4c7e + attacker_slot*0x104 - 0x64e`）：
   索引方式疑似不是標準 `PARTY.DAT` 定址（見 §2），需要先確認 `0x4c7e`
   在這個脈絡下究竟指向哪張表。
3. **召喚生物屬性來源**：`FUN_138d_2f7e`、`FUN_138d_2fa0`
   （分別在 `effect_type==8`、`9` 內被呼叫但未展開）是最可能藏著答案的兩個
   函式，建議下一輪優先讀這兩個（都不大，粗估 `FUN_138d_2fa0` 約 30-40
   bytes）。同時要釐清 §6 提到的「Use/AI 路徑 vs Cast 路徑行為落差」是否
   代表兩套獨立機制。
4. **道具 `charges_field_0x11 == charges_field_0x12` 過濾條件**（§4.3）
   語意不明，需要動態驗證或找到欄位初始化點。
5. **`FUN_138d_1ceb` 回傳 `0` 的邊界情況**（§3.2）語意不明確，建議先當成
   「不可達」處理，不必特別在 Go 引擎重現。

---

## 附：可重跑的驗證片段

```bash
cd /home/anr2/cht/daemon_winter

# 1. 命中/傷害核心函式，確認無跳表污染（乾淨反編譯）
cat workplace/ghidra/export/decompiled/138d_25da_FUN_138d_25da.c

# 2. 戰敗/勝利/逃跑三個字串，原始位元組核對
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
seg = 0x31f0
for off,label in [(0xbb2,'local_14=0'),(0x73f,'local_14=1 (monsters)'),
                   (0x274e,'local_14=2 part1'),(0x275c,'local_14=2 part2'),
                   (0x276e,'local_14=2 part3'),(0x2780,'local_14=2 part4')]:
    fo = seg*16+off-0xC400
    print(label, '->', data[fo:fo+40].split(b'\x00')[0])
"

# 3. 第三張跳表（FUN_138d_3c81，效果類型分派，138d:3f95，18 項）
python3 -c "
import struct
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
seg,off = 0x138d, 0x3f95
fo = seg*16+off-0xC400
entries = struct.unpack_from('<18H', data, fo)
for i,e in enumerate(entries):
    print(f'effect_type {i:2d}: 138d:{e:04x}')
"

# 4. FUN_138d_134d（5x5 AOE 主體）原始位元組反組譯
objdump -D -b binary -m i386 -Maddr16,data16 \
  --start-address=0x881d --stop-address=0x8a20 \
  workplace/orig/demwin/DEMON.INT 2>/dev/null

# 5. 確認 FUN_138d_134d 只有 3 個呼叫端（AOE 是共用機制）
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
pattern = bytes([0x9a, 0x4d, 0x13, 0x8d, 0x03])
idx = 0
while True:
    idx = data.find(pattern, idx)
    if idx == -1: break
    print(hex(idx))
    idx += 1
"

# 6. FUN_138d_1d70 / FUN_138d_1ceb（戰敗判定鏈）
cat workplace/ghidra/export/decompiled/138d_1d70_FUN_138d_1d70.c
cat workplace/ghidra/export/decompiled/138d_1ceb_FUN_138d_1ceb.c

# 7. Use 道具入口
cat workplace/ghidra/export/decompiled/17c5_18ab_FUN_17c5_18ab.c
```
