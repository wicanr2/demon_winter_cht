# 戰鬥狀態機與核心演算法(DEMON.INT 反組譯)

分析對象：戰鬥回合流程、命中/傷害計算、行動分派、RNG。全部證據來自
`workplace/ghidra/export/`(`disassembly.asm`、`decompiled_all.c`、`decompiled/*.c`、
`strings.csv`、`functions.csv`),以及對 `workplace/orig/demwin/DEMON.INT` 原始位元組的
直接讀取核對。位址換算一律套用 `docs/re/00-ghidra-setup.md` 的公式：
`file_offset = segment*16 + offset - 0xC400`。本文件的絕對位址(如 `0x4eba`)若未特別標
segment,一律是 DS 段內位移(這顆二進位 DS=SS=`31f0`,見 `docs/re/00`/`03` 已驗證的結論),
可直接視為 `31f0:xxxx`。

---

## 0. 總覽(最重要的結論先講)

**已驗證**:

1. **回合結構不是「玩家全體→敵人全體」,是全體單位混合、依速度排序的統一行動佇列**,
   每回合重新排序一次(見 §1)。
2. **命中/傷害計算全部是整數運算**,沒有在戰鬥公式本身用到 8087/浮點——**唯一找到的
   浮點使用點是 RNG 產生器內部**(軟體浮點函式庫,見 §3、§5)。這修正了任務單原先
   「命中/傷害最可能用浮點」的預期方向:浮點確實被用了,但用在 RNG 這一層,不是
   命中率或傷害公式本身。
3. **RNG 找到了完整呼叫鏈**:核心是一個 32-bit 同餘產生器(乘數 125),搭配 DOS
   系統時鐘(`INT 21h AH=2Ch`)機率性重播種,最終透過一套自製的軟體浮點函式庫
   (mimicking 80-bit 擴充精度格式)算出範圍內的整數。**遞迴公式的精確模數/化簡式
   沒有完全解出**(見 §4 的誠實標註),但狀態變數位址、種子來源、呼叫順序都已驗證,
   足以在 Go 端重建一個「近似同構」的產生器,若要逐位元對拍還需要再深挖一層。
4. **意外收穫**:命中/傷害函式反查出好幾個 `docs/formats/game-data-tables.md` 標記
   「未知」的 `PARTY.DAT` 欄位語意(裝備武器/護甲槽位索引、中毒狀態、盾牌加成),
   以及一張 14 項武器傷害骰表(直接讀原始位元組核對),見 §6。

---

## 1. 戰鬥回合結構與主迴圈

### 1.1 主迴圈:`FUN_138d_0002`(`138d:0002`,1009 bytes)

錨點:`End of round`(`31f0:06cb`)在此函式內被引用(`138d:03df`,`FUN_1d9f_1361(0x6cb)`),
已用 `grep "0x6cb\b" disassembly.asm` 驗證只有這一處引用。

**已驗證的回合流程**(逐段對照反編譯碼):

```c
// 精簡重現,完整原始碼見 decompiled/138d_0002_FUN_138d_0002.c
do {                                   // 每次迭代 = 一個「回合」
    round_counter += 1;                // [0x518e]
    refresh_screen();

    // 1) 建立本回合的行動佇列:掃描全部 15 個戰鬥單位槽(index 0-14)
    for (slot = 0; slot < 15; slot++) {
        unit = &combat_record[slot];   // 每筆 0x26=38 bytes,基底 0x4eb4 一帶
        if (unit.status < 2 && unit.exists != 0 && unit.status_counter >= 0) {
            queue[n].speed = unit.speed;      // [slot*0x26 + 0x4eb8]
            queue[n].slot  = slot;
            n++;
        }
    }

    // 2) 依速度「由大到小」氣泡排序 —— 這就是行動順序
    bubble_sort_descending(queue, n);

    // 3) 依排序後順序逐一行動
    for (i = 0; i <= n; i++) {
        unit = combat_record[queue[i].slot];
        if (!alive_check(unit)) continue;
        move_camera_toward(unit);              // 鏡頭/游標移動到該單位

        if (unit.attack_type in {3, 0xd} && RNG(10) < 3) {   // 30% 機率
            result = breath_weapon_path(...);    // 特殊能力(吐息類),見 §1.3
        } else {
            if (unit.attack_type == 2 && RNG(5) == 2) unit.attack_type = 0xb;  // 狀態轉換(未查明)
            if (unit.attack_type < 0xb)
                result = FUN_138d_03f5(unit);     // 怪物 AI 回合
            else
                result = FUN_138d_1ef8(unit);     // 玩家行動選單
        }

        if (result == 0 || result == 1) { FUN_1990_0aeb(result); return; }  // 戰鬥結束(勝/敗)
        if (result == 2) { FUN_25be_000c(); return; }                       // 另一種結束(逃跑/其他)
        if (result == 4) { /* 重置鏡頭,繼續 */ }
    }

    print_round_end_summary();
    print("End of round");             // 0x6cb
} while (true);
```

**已驗證**:
- 行動順序 = **全體(玩家+怪物混合)依速度屬性由高到低排序,每回合重排一次**,
  不是「玩家全體行動完再輪敵人」。
- 排序鍵是 `combat_record[slot].speed`(位移 `0x4eb8`),追溯來源見 §6,
  已驗證等於 `PARTY.DAT` 記錄 `0xf7`(速度含裝備加成)。
- 戰鬥單位槽位配置:**index 0–6 = 怪物(最多 7 隻)、index 7–14 = 玩家隊伍
  (最多 8,含召喚物?)**——依據 `FUN_138d_1ceb`(`138d:1ceb`,隊伍存活掃描)
  掃描範圍是 `[7, 7+party_size)`,以及 `FUN_138d_25da` 用 `if (attacker_slot < 7)`
  選擇不同音效(見 §6 的音效修正建議)。**這點與音效文件 `03-audio-and-resources.md`
  §1.5 原先「近戰/遠端」的解讀不同,建議下一位 agent 覆核**(見 §7)。

### 1.2 每單位一回合能做什麼、Move pts 怎麼消耗

錨點:`Move pts: %2d`(`31f0:0815`)在 `FUN_138d_1ef8`(`138d:1faf`)引用,已驗證是
玩家行動選單函式(`138d:1ef8`,565 bytes)開頭印出的狀態列(顯示該角色本回合剩餘
移動點數,再讓玩家選擇行動)。

**已驗證**:
- `Move pts` 對應欄位是戰鬥記錄 `0x4ec2`(**假設**:語意為「本回合可用行動/移動點數」,
  依據是這個欄位在多處被「花費」:選單 case 1(攻擊)結尾扣掉一個依武器傷害加成算出
  的量、case 2 扣一半(`/2`,除非本身是 1)。**沒有找到「每回合重新填滿 Move pts」的
  明確賦值點**,推測在回合開始的某個尚未追到的初始化路徑,未完全定案。
- 每個單位一回合可以:移動(尋路,`FUN_222f_1404` / `FUN_1990_050d` 找最近目標)、
  攻擊(選單 **case 5**,呼叫 `FUN_1990_0002` → `FUN_138d_25da`;
  ~~case 0xa~~ 是舊的錯誤編號,見 §5 修正)、其他選單動作
  (施法/用道具/逃跑等,見 §5)。行動選完後扣點數、回到主迴圈換下一個單位。

### 1.3 怪物特殊能力(吐息)觸發

`breathes!`(`31f0:0717`)在 `FUN_138d_17b8`(`138d:17b8`,1244 bytes,即命中/傷害
函式旁邊那個處理「特殊傷害結算」的函式)內被引用。主迴圈用
`unit.attack_type ∈ {3, 0xd}` 且 `RNG(10) < 3`(30% 機率)進這條分支
(**假設**:`attack_type` 這兩個值在此處被借用為「有吐息能力」的旗標,不是字面上
Shortsword/Bite 的意思,見 §6 對 `attack_type` 欄位角色的說明)。

### 1.4 勝負判定

**已驗證**:
- **勝利**:`FUN_138d_1d70`(`138d:1d70`)在 `FUN_138d_1c94`(單位死亡後的清理函式)
  裡被呼叫。邏輯是:若剛死亡的是怪物(`unit.attack_type < 10`),掃描全部怪物槽
  (`0x4eb4 != 0 && attack_type == 1` —— **[假設]** `attack_type==1` 在此處代表
  「仍存活的怪物」,語意跟前面 attack_type 當武器索引用不完全一致,懷疑主迴圈把
  死亡怪物的 attack_type 改寫成哨兵值,沒有逐一追證),若一隻都沒有,印
  `All monsters dead`(`31f0:073f`)並回傳勝利碼 `1`。
- **失敗(隊伍全滅)**:`FUN_138d_1ceb`(`138d:1ceb`)掃描 index 7 起 `party_size` 個槽,
  若 `0x4eb4 != 0` 存在則回傳 3(戰鬥繼續);若全部槽位 `status(0x4ec4) >= 2` 回傳 2。
  **沒有在時間內找到「隊伍全滅→印失敗訊息→結束戰鬥」的明確呼叫點**,只確認了
  這個「掃描存活」的子函式存在,判定邏輯本身待下一位 agent 接續(見 §8)。
- 戰鬥結束的收尾統一由 `FUN_1990_0aeb`(`1990:0aeb`,1564 bytes)處理
  (主迴圈 `result==0` 或 `1` 時呼叫),內部有隊伍狀態欄位的批次更新,細節未深入
  (不影響戰鬥核心公式,屬於收尾 UI/存檔邏輯)。

---

## 2. 命中判定公式(已驗證框架,部分係數為假設)

核心函式:**`FUN_138d_25da`**(`138d:25da`,2185 bytes)。錨點:
`is hit for %d damage`(`31f0:072a`,實際在鄰近的 `FUN_138d_17b8` 內)、
`misses.`(`31f0:0849`,直接在 `FUN_138d_25da` 內,`138d:2861`)。

參數:`FUN_138d_25da(attacker_slot, target_slot, param_3, param_4)`。

### 2.1 命中率公式(已驗證主幹,部分加成項為假設)

```c
hit_chance = attacker.skill * 4;                     // 0x4ec6,已驗證來源=PARTY.DAT[+0xfb]技巧(含加成)

if (target.facing(0x4ec8) == attacker.facing(0x4ec8))
    hit_chance += 12;                                 // [假設] 朝向/位置相符加成

if (attacker.weapon_type ∈ {2, 0xb}) {                 // 小斧型武器 或 玩家
    hit_chance += attacker.enchant_dmg_bonus(0x4ece) * 3;   // 已驗證來源=裝備附魔值,見 §6
    if (attacker.field_4ed6 == 0x18) {                 // [假設] 特定 buff/技能 ID
        hit_chance += 10;
        crit_threshold -= 8;                           // 見 2.2
    }
}

if (target.status(0x4ec4) < 2)                         // 目標存活/未失能
    hit_chance += target.status_counter(0x4ecc) * -4;   // [假設] 「防禦中」計數器懲罰攻擊方命中率

roll = RNG(100);                                        // 1..100
if (hit_chance < roll) => MISS;                          // roll <= hit_chance 才算命中
```

**已驗證**:命中率門檻(`local_8`)與 `RNG(100)` 比較,`hit_chance < roll` 時未命中,
印 `misses.`(`0x849`),依攻擊者槽位(`<7` 或 `>=7`,見 §1.1 修正)選音效
`FUN_1d9f_2a95(1)` 或 `(4)`(對照 `docs/re/03-audio-and-resources.md` §1.5 已記錄的
呼叫點 `138d:25da`,完全吻合)。

**假設(未逐項動態驗證)**:各加成項的確切語意(朝向/位置比對、`field_4ed6==0x18`
的 buff 語意、目標防禦計數器懲罰)是依欄位使用模式推斷,沒有用 DOSBox 逐項對照
UI 數值驗證。

### 2.2 暴擊判定(已驗證有此機制,係數為觀察值)

```c
crit_threshold = 0x5a;   // 90(基準:100 中有 10 個值算爆擊,即 10% 機率)
if (attacker.weapon_type ∈ {2, 0xb} && *(charbase - 0x64e) != 0)
    crit_threshold = 0x4b;  // 75(25% 機率)—— [假設] charbase-0x64e 疑似職業/技能旗標表,未追到來源

roll2 = RNG(100);
if (crit_threshold < roll2) is_critical = true;   // 傷害後面 <<1(雙倍),見 §3
```

**已驗證**:命中後才擲第二次 `RNG(100)`,門檻預設 90(對應 10% 基礎爆擊率),
特定條件下降到 75(25%)。是否真的是「爆擊」還是別的傷害類型分歧,依據是
`if (local_6 == -1) local_12 <<= 1`(傷害左移 1 位,即乘 2),語意上等同雙倍傷害。

---

## 3. 傷害計算公式(已驗證主幹,武器骰表已用原始位元組核對)

延續 `FUN_138d_25da`,命中後的傷害計算(`local_12`):

### 3.1 一般武器攻擊(最常見路徑)

```c
weapon_type = abs(attacker.equipped_weapon_type);   // 0x4ebc,abs() = FUN_206a_000a
if (weapon_type == 0xd) {
    damage = RNG(attacker.strength / 2);             // 特殊武器類型(徒手/未知武器 13)
} else {
    dice_max = WEAPON_DICE_TABLE[weapon_type];       // 見 3.2,DS:0x1785,14 bytes
    damage = RNG(dice_max);                          // 武器基礎傷害骰
    damage += (attacker.strength(0x4ec0) - 7) / 2;    // 力量加成,已驗證來源=PARTY.DAT[+0xf8]力量(含加成)
    if (attacker.weapon_type ∈ {2, 0xb}) {
        damage += attacker.enchant_dmg_bonus(0x4ece) + attacker.enchant_bonus2(0x4ed0);
        if (target.field_4ed6 ∈ {7, 10})              // [假設] 目標種族/類型 ID(可能是「剋不死系」判定)
            damage += attacker.enchant_bonus2(0x4ed0);
    }
}
```

### 3.2 武器類型基礎傷害骰表(已驗證,直接讀原始位元組核對)

位址 `DS:0x1785`(= `31f0:1785`,file offset `0x27285`),14 bytes,索引對應
`docs/formats/game-data-tables.md` 已記錄的 14 項「攻擊/武器類型」清單
(`Hands, Dagger, Small ax, Shortsword, Mace, Morningstar, Broadsword, Battle ax,
2-hand sword, Mace, 2-hand sword, Battle ax, 2-hand sword, Bite`,索引 0–13):

| 索引 | 武器類型 | 傷害骰上限(RNG(N),1..N) |
|---|---|---|
| 0 | Hands(徒手) | 2 |
| 1 | Dagger(匕首) | 3 |
| 2 | Small ax(小斧) | 4 |
| 3 | Shortsword(短劍) | 6 |
| 4 | Mace(釘頭鎚) | 6 |
| 5 | Morningstar(晨星鎚) | 7 |
| 6 | Broadsword(闊劍) | 8 |
| 7 | Battle ax(戰斧) | 10 |
| 8 | 2-hand sword(雙手劍) | 12 |
| 9 | Mace(第二種/變體) | 7 |
| 10 | 2-hand sword(變體) | 13 |
| 11 | Battle ax(變體) | 11 |
| 12 | 2-hand sword(變體) | 14 |
| 13 | Bite(咬擊) | 3 |

**已驗證**(直接用 Python 讀 `DEMON.INT` file offset `0x27285` 起 14 bytes 核對,
與反組譯出的 `*(undefined1*)(iVar4+0x1785)` 索引邏輯完全吻合):
`[2, 3, 4, 6, 6, 7, 8, 10, 12, 7, 13, 11, 14, 3]`。

**額外發現(已驗證,補強 `docs/formats/game-data-tables.md`)**:玩家角色裝備武器時,
`ITEMS.DAT` 的武器索引(0–7,`dagger/small axe/short sword/mace/morning star/
broad sword/battle axe/2-hand sword`)會在裝備欄位(見 §6)存成 `slot_byte[+0xc] + 1`,
**這個 `+1` 恰好把 `ITEMS.DAT` 的 0-based 武器索引對齊到這張 14 項攻擊類型表的
索引 1–8**(`ITEMS.DAT` 索引 0=dagger → 表索引 1=Dagger,以此類推)。兩份文件的
武器清單順序完全對得上,不是巧合。

### 3.3 特殊攻擊路徑(未使用一般武器骰表)

```c
if (attacker.ranged_flag(0x4ebc) == 0 &&
    attacker.weapon_type ∈ {2, 0xb} &&
    attacker.field_4ed6 ∈ {0x15, 0x17}) {              // [假設] 特殊技能 ID(疑似"backstab"類技能)
    damage = RNG(attacker.skill(0x4ec6) - 5);
    damage += (attacker.strength - 7) / 2;
}
```

### 3.4 最終傷害修正與套用(已驗證)

```c
if (is_critical) damage <<= 1;                       // 爆擊雙倍,見 2.2
damage -= target.armor(0x4ebe);                        // 已驗證來源=裝備護甲的最終防禦值(見 §6)

if (damage < 1) {
    // 印「no damage.」,仍算命中,回傳碼 3
} else {
    target.hp(0x4eba) -= damage;                        // 扣血
    if (target 存活 && attacker 為近戰(2/0xb) && RNG(7)==3)   // ~14%
        觸發二次攻擊(FUN_138d_2e63,隨機部位)；            // [假設] 追加攻擊/多重連擊機制
    if (target.hp <= 0) {
        印 "It dies!"(0x891);
        FUN_138d_1c94(...);                              // 死亡處理 + 勝負判定,見 §1.4
    }
}
```

**已驗證的額外狀態效果(同一函式內)**:
- **中毒**:`attacker.equipped_weapon_type < 0`(負值)且 `RNG(100) < 15`(15%)且
  目標未中毒 → 目標中毒(`target.status = 1`),印 `poisoned`(`0x87e`)。
  **這與 `docs/formats/game-data-tables.md` 已記錄的
  「MONSTER.DAT attack_type 負值=帶毒」完全吻合**,是很強的交叉驗證。
- **暈眩/擾亂**:近戰武器(2/0xb)、特定 `field_4ed6`(0x16/0x17)、
  `RNG(100) <= attacker.skill*2` → 目標 `status_counter` 疊加,印 `stunned`(`0x888`)。
- 命中後 20% 機率(`RNG(5)==2`)讓目標把攻擊焦點(`field_4ed2`)轉向攻擊者(仇恨/aggro)。

**已驗證**:命中/傷害的整條計算鏈**全部是整數運算(加減乘除、位移)**,
沒有出現任何 x87 指令或軟體浮點函式庫呼叫。8087 相關發現全部在 RNG 那一層,見 §5。

---

## 4. RNG 演算法與 seed 來源(對拍基石,已驗證框架與呼叫鏈,精確遞迴式未完全解出)

### 4.1 頂層介面:`FUN_1d9f_0e0b(max)`(`1d9f:0e0b`)

**已驗證**:全檔案被呼叫 **234 次**,是遍布全遊戲(不只戰鬥)的唯一擲骰函式。
語意:回傳 `[1, max]` 的均勻亂數整數。

```c
int FUN_1d9f_0e0b(int max) {
    if (max < 0) max = FUN_206a_000a(max);      // abs()
    if (max == 0 || max == 1) return 1;          // 邊界情況直接回傳 1
    tick = FUN_2f46_000a();                       // 更新系統時鐘計數器,見 4.2
    if (tick & 1) FUN_1d9f_0dda();                 // 奇數 tick 時額外呼叫一次浮點正規化
    FUN_1d9f_0dd4();                               // 恆定呼叫:推進 LCG 狀態(內部呼叫 FUN_30c2_0006)
    // 透過軟體浮點庫(310e 段)把結果轉成 float、乘上 max、再轉回 int
    result = float_to_int(...);
    return result + 1;
}
```

用法交叉驗證:主迴圈用 `RNG(10) < 3` 判斷 30% 觸發吐息(§1.3),命中/傷害函式用
`RNG(100)` 兩次(命中/爆擊)、`RNG(dice_max)`(武器傷害)——都符合「`RNG(N)` 回傳
`1..N` 均勻整數」的解讀。

### 4.2 種子來源:`FUN_2f46_000a`(`2f46:000a`)——已驗證是 DOS 系統時鐘

```c
void FUN_2f46_000a(void) {
    swi(0x21);   // INT 21h,依暫存器用法對應 AH=0x2Ch(Get System Time)
                  // CX = 小時*256+分鐘,DX = 秒*256+百分秒
    tick_delta = (CX_hour*60 + CX_min) * 6000 + (DX_sec*100 + DX_hundredths);
    accumulator = accumulator_prev + tick_delta;
    if (跨日:新值小於等於舊值) accumulator += 0x83D600;   // 0x83D600 = 8,640,000 = 一天的百分秒數,已驗證
    保存 accumulator 到 [0x457c:0x457e](32-bit,低位:高位);
}
```

**已驗證**:`0x83D600`(8,640,000)精確等於 `24*60*60*100`(一天的百分之一秒總數),
證實這是「以百分秒為單位、處理跨日進位的系統時鐘累加器」,不是巧合。這是整個
RNG 鏈唯一的「真隨機」熵來源(來自 DOS 系統時鐘,每次呼叫都重新讀取)。

### 4.3 核心狀態推進:`FUN_30c2_0006`(`30c2:0006`)——32-bit 同餘產生器

```c
void FUN_30c2_0006(void) {
    // state 是 [0x481c(低字):0x481e(高字)] 的 32-bit 整數
    lo = state.lo;
    state.hi = (uint32)(lo * 0x7d) >> 16 + state.hi * 0x7d;   // *125,含進位處理
    state.lo = (uint16)(lo * 0x7d);
    q = FUN_3016_000a();          // [未完全解出] 疑似 state/3 一類的除法(見 4.4)
    state -= q * 0xAAAB;           // 0xAAAB=43691,是「除以 3」的乘法-移位近似值(0xAAAB*3=0x20001)
    FUN_310e_07a4(...); FUN_310e_0196(...); FUN_310e_045e(...);   // 軟體浮點正規化(段 310e)
}
```

**已驗證**:狀態變數位址 `[0x481c:0x481e]`(32-bit)、乘數 `125`(`0x7d`)、以及一個
「乘以 `0xAAAB` 做除以 3 近似」的修正項,這個結構模式清楚(乘法同餘 + 模運算修正),
但**沒有完全化簡出乾淨的閉式遞迴公式**(`state = (state * A) mod M` 的精確 `A`、`M`
值未定案,因為 `FUN_3016_000a` 的除法語意還沒追完,見 §8 未解問題)。

**假設**:這極可能是編譯器執行期函式庫(Microsoft C / Turbo C 一類)內建的
`random()`/`rand()` 實作,不是遊戲自己寫的公式(因為整段程式碼風格、與時鐘/浮點
函式庫緊密耦合的寫法,很像標準庫慣例),但沒有比對到已知編譯器原始碼確認。

### 4.4 除法輔助:`FUN_3016_000a` / `FUN_3016_0068`(`3016:000a`/`3016:0068`)

只確認了呼叫存在、參數是隱含暫存器傳遞(`in_DX`/`in_BX` 符號正負處理,典型的
「除法结果依除數/被除數正負號調整符號」寫法),**沒有把 `FUN_3016_0068` 的內部
除法演算法解出來**(時間所限)。

### 4.5 對「Go 引擎對拍」的務實建議

**已驗證的結論**:這個 RNG 不是簡單的線性同餘公式,而是「整數 LCG 狀態 + 軟體浮點
正規化 + 機率性的時鐘重播種」三層疊加,逐位元重現需要把 `FUN_30c2_0006`、
`FUN_3016_0068`、以及 `310e` 段的軟體浮點函式庫(80-bit 擴充精度格式,§5)完整
移植成 Go 程式碼——這是可行的(全部是確定性整數/位元運算,沒有真隨機的部分,
真隨機只有時鐘重播種那一步),但工作量不小。**如果目標只是「行為統計上像原版」
而非「逐位元對拍」,用 Go 標準庫的 PRNG 加上同樣的 `RNG(N)` 介面語意(均勻
`[1,N]`)就足夠**;**如果目標是逐位元對拍(例如要重播原版錄影驗證),必須完整
移植 `FUN_30c2_0006`+`FUN_3016_0068`+軟體浮點庫,且要重現「時鐘種子」這個非
確定性輸入——這代表對拍時必須用同一份「錄製下來的擲骰序列」而非重新播種,
否則時鐘種子不同就對不上。這點請務必先跟主持人確認優先度,再決定要不要投入
完整移植**。

---

## 5. 8087 浮點的使用點(對接 PLAN V7)

**已驗證(反面證據優先)**:

1. 用 Ghidra 反組譯掃描全域,**只找到 1 筆看起來像 x87 指令的結果**(`138d:3ce8
   FDIV ST7,ST0`),但上下文核對後**判定是誤判**——它夾在明顯未被正確分析的資料
   區塊中間(前後都是無法連續解碼的位元組、且下一行緊接跳到完全不相干的段
   `1000:75c8`),不是真正的程式碼。
2. 用 `objdump` 對整份 `DEMON.INT` 做平坦反組譯(不依賴 Ghidra 的函式邊界分析),
   搜尋全部 x87 助憶符(`FLD/FMUL/FADD/FDIV/FIST/FBLD/FWAIT/FLDCW/FNSTCW`…),
   **只找到 6 筆命中**,且全部落在檔案很前段(file offset `0x5e0`–`0x2fc8`),
   內容是 `fbld`/`fwait`/`fnstcw`/`fldcw`——這是**C 執行期函式庫在程式啟動時
   偵測/初始化 8087(或設定浮點模擬)的標準樣板碼**,不是遊戲邏輯本身用到硬體浮點。

**結論(已驗證)**:**DEMON.INT 幾乎沒有真的執行 x87 硬體指令**。取而代之,遊戲的
浮點需求(目前唯一找到的用途是 RNG,見 §4)是透過**一套自製的軟體浮點函式庫**
(segment `310e`,約 27 個小函式)實作的——這套函式庫維護一個「軟體 FP 堆疊」
(堆疊指標存在 `[0x482e]`),用純整數指令(`MUL`/`ADD`/`ADC`,可見 `FUN_310e_0588`
內一整段多字組乘法)手動模擬 80-bit 擴充精度浮點數的尾數乘法、正規化、
整數↔浮點轉換(`FUN_310e_0732`=int→float、`FUN_310e_07e2`=float→int),
指數偏移量 `0x3ff`(1023)與 x87/IEEE 擴充精度的偏移慣例一致。

**對 PLAN V7 的修正建議**:「8087 協同處理器支援」這件事本身成立(執行期函式庫
確實有偵測/初始化 8087 的樣板碼),但**目前反組譯範圍內沒有找到任何遊戲邏輯
路徑真的執行硬體 x87 指令**——已知的浮點使用者(RNG)走的是純軟體模擬路徑
(可能是因為編譯器設定成「不論硬體是否存在都用軟體函式庫」的保守模式,或是
戰鬥/RNG 這段程式碼被編譯器選擇走 emulation 分支)。**Go 引擎不需要模擬真正的
x87 seg register/tag word 語意去對齊這顆程式**——只要把 `310e` 段的軟體浮點函式庫
邏輯翻成 Go(用 `float64` 或直接搬過去的定點/整數運算都可以,因為原版本身就是
純運算),精度不會因為「x87 80-bit vs float64 53-bit」產生原任務單擔心的那種偏差
(除非之後在其他子系統挖到真正用 x87 指令的地方,那才需要重新評估)。

---

## 6. 對既有文件的修正/補充(供主持人裁決,未直接修改對應文件)

反查命中/傷害函式時,順帶解出了 `docs/formats/game-data-tables.md` 標記「未知」的
幾個 `PARTY.DAT` 欄位,建議之後併入該文件:

| 欄位 | 原文件狀態 | 本次發現 |
|---|---|---|
| `PARTY.DAT` 角色記錄 `+0x100` | 「未知,4 bytes 區塊 `0x100-0x103` 之一」 | **已驗證**:裝備武器的「裝備欄槽位索引」(0-9,`0xFF`=未裝備),戰鬥開始時複製進戰鬥記錄 `0x4ebc`,再被改寫成 `slot_byte[+0xc]+1`(即武器類型索引,見 §3.2) |
| `PARTY.DAT` 角色記錄 `+0x101` | 同上 | **已驗證**:裝備護甲的「裝備欄槽位索引」(`0xFF`=未裝備),用於算出最終護甲值(`0x4ebe`,見下) |
| `PARTY.DAT` 角色記錄 `+0x102` | 同上 | **已驗證**:戰鬥狀態旗標(0=正常,1=中毒——與戰鬥函式的中毒判定寫回位置一致) |
| `PARTY.DAT` 角色記錄 `+0xec` | 未在既有文件出現(落在 `0xc4`-`0xe8` 間未記錄的空隙) | **[假設]**:狀態持續計數器(暈眩/防禦等),條件式複製進戰鬥記錄 `0x4ecc` |
| `PARTY.DAT` 角色記錄 `+0xe6` | 未在既有文件出現 | **[假設]**:盾牌裝備旗標,非零時護甲值 `+2` |
| 裝備欄 slot byte0(`+0x1a` 起,17 bytes/slot 的第 0 byte) | 「幾乎全部是 `0x0a`(=10),疑似『已使用』常數旗標」 | **[假設,較原文件更進一步]**:基準值 10 = 無附魔加成,`byte0 - 10` 被戰鬥公式直接當「武器附魔傷害加成」使用(`0x4ece`),不只是「已使用」旗標 |
| 護甲最終數值來源 | 未提及 | **已驗證有此機制,細節未解**:`0x4ebe` 從護甲槽位索引查一張位於 `31f0:16c5` 的表(依裝備槽 byte0 值查表),再視盾牌旗標 `+2`,得出最終護甲值,用於傷害扣減。表格內容本次未逐一核對(时間所限) |
| `ITEMS.DAT` 武器索引順序 | 已驗證(0-7 對應 `dagger`…`2-hand sword`) | **交叉驗證成功**:與本文件 §3.2 的 14 項攻擊類型表用 `+1` 對齊,兩份獨立解出的資料完全吻合 |

`docs/re/03-audio-and-resources.md` §1.5 的修正建議:文件原本把
`FUN_138d_25da` 的 miss 音效 `local_4=1(param_1<7)/4(else)` 解讀為「近戰/遠端」,
本次分析發現 **`param_1 < 7` 精確對應「攻擊者槽位屬於怪物區(index 0-6)」**
(依據 `FUN_138d_1ceb` 掃描 `[7, 7+party_size)` 判定玩家槽位起點是 7),
**建議修正為「怪物攻擊 vs 玩家攻擊」的音效分流,而非武器射程分流**——但這點
沒有在音效文件範圍內驗證(不屬本次任務邊界,列在這裡供裁決/下一位 agent 覆核)。

---

## 7. 行動分派表

主要分派點是 `FUN_138d_1ef8`(玩家回合)裡的 `switch(選單索引)`(索引 0–14,
由 `FUN_2cdc_033d(0x75e)` 讀取玩家按鍵/點選結果):

> **⚠ 2026-07-25 全表作廢**：下表是根據**反編譯 C** 推測的，**case 編號全部錯誤**。
> 正解見 [`docs/re/09-spells-and-actions.md`](09-spells-and-actions.md)，
> 那份是直接讀 `138d:258f` 的**原始跳表**得到的：
>
> ```asm
> 138d:25ad  CMP AX,0xf                     ; 邊界檢查,15 個項
> 138d:25b3  SHL BX,0x1                     ; index × 2
> 138d:25b5  JMP word ptr CS:[BX + 0x258f]  ; ← 跳表本體
> ```
>
> 正確對照：**Attack = case 5**、Cast = 6、Use = 7、Turn Undead = 8、Dodge = 9、
> **Examine = 10 (0xa)**、Sound = 11、Pray = 12、Leech = 13。
>
> 本表最嚴重的錯誤是把 **case 0xa 標成「已驗證 = Attack」**——實際上 0xa 是 Examine。
> 協調者已獨立複核：case 5 的進入點先檢查行動點 `[0x5190] >= 3`、
> 再以 stride 38 索引戰鬥單位陣列 `0x4ec8`（有成本、會動戰鬥狀態，符合攻擊）；
> case 0xa 只 push 單位索引 → 一次 far call → 設旗標返回（純顯示，符合檢視）。
>
> **教訓**：Ghidra 對含跳表的 switch 反編譯不可信，它會把跳表資料誤讀成程式碼。
> 這類分派表一律要回原始指令讀，不能用反編譯結果。
> 下表保留作為「錯誤示範」與探索紀錄，**不可作為實作依據**。

<details>
<summary>作廢的舊推測表（點開僅供追溯）</summary>

| case | 已知行為 | 當時的信心標註 |
|---|---|---|
| 1 | 複雜的傷害/移動點數處理,結尾呼叫 `FUN_138d_1e19` | 假設:可能是「Cast」或攻擊變體 |
| 0xa | 呼叫 `FUN_1990_0002`(尋路+呼叫 `FUN_138d_25da`) | ~~已驗證=Attack~~ **錯誤,實為 Examine** |
| 0xd | `Move pts` 計數器 -2、鏡頭歸位、回傳 `0x20` | 假設:疑似撤退/逃跑 |
| 3 | 涉及 `0x530c` 一帶陣列的複雜運算 | 假設:法術/群體效果 |
| 4 | 涉及道具清單、文字排序/選擇邏輯 | 假設:「Use」道具 |
| 0xc | HP 上限鉗制與回復 | 假設:疑似 Leech 或回復 |
| 其他 | 未逐一比對 | 未定案 |

</details>

**已驗證但未逐一對應到選單文字**:同一個 `FUN_1990_0002` 也被怪物 AI 函式
`FUN_138d_03f5` 直接呼叫(不經過選單),證實它是「移動到目標+執行普通攻擊」的
**共用**核心邏輯,玩家與怪物的普通攻擊最終都收斂到同一段命中/傷害計算
(`FUN_138d_25da`)——這對 Go 引擎是好消息:**普通攻擊只需要實作一套邏輯,
不用分玩家/怪物兩份**。

**未解**:`Attack/Cast/Use/Turn Undead/Dodge/Examine/Pray/Leech` 這 8 個選單字串
與 `switch` 的 case 索引沒有在時間內逐一比對(選單顯示與索引的對應需要另外追
`FUN_2cdc_033d` 的選單字串來源表)。

> **2026-07-25 更新**:全部 8 個動作的 case 編號已由 `docs/re/09` 讀原始跳表解出
> (Attack=5、Cast=6、Use=7、Turn Undead=8、Dodge=9、Examine=10、Sound=11、
> Pray=12、Leech=13)。本節原文「除了 case 0xa(已確認=Attack)之外其餘留給下一輪」
> 的敘述已作廢 —— 那個 0xa 的認定本身就是錯的。

---

## 8. 給 Go 引擎的建議

1. **戰鬥狀態機**:實作成「每回合重建行動佇列(依速度降冪排序全體單位)→
   依序執行每個存活單位的行動→行動後檢查勝負→回合結束印摘要」的迴圈,對應
   `FUN_138d_0002` 的結構(§1.1)。不要做成「玩家階段/怪物階段」兩段式。
2. **命中/傷害**:§2、§3 給出的整數公式可以直接照抄(全部整數運算,沒有精度
   對齊問題)。武器基礎傷害骰表(§3.2 的 14 項表)、力量加成 `(str-7)/2`、
   命中率 `skill*4 + 修正項`、爆擊 `<<1`、護甲扣減,都已有具體數值可以帶入實作。
   建議先把 §2/§3 標「已驗證」的部分做完整測試覆蓋,標「假設」的加成項先用
   保守預設值(或做成可調參數),等後續有 DOSBox 動態驗證再鎖定。
3. **RNG(最關鍵)**:如果要求「逐位元對拍原版錄影/存檔」,必須完整移植
   `FUN_30c2_0006`(32-bit LCG,乘數 125)+ `FUN_3016_0068`(除法輔助,未完全解出)+
   `310e` 段軟體浮點庫(int↔float 轉換、乘法正規化),並且比對種子時鐘輸入用
   同一份錄製序列而非重新取樣系統時間。如果只要求「統計行為像原版」,可以用
   Go 標準庫 RNG 包一層 `Roll(n int) int { return rand.Intn(n) + 1 }` 介面對齊
   `FUN_1d9f_0e0b` 的語意即可,不需要移植內部演算法。**這個取捨請先跟主持人
   確認**,因為完整移植的工作量顯著更高。
4. **8087/浮點**:不需要特別处理 x87 相容性問題(§5 已驗證遊戲邏輯本身不執行
   x87 指令),`float64` 或整數定點都可以安全用在 RNG 的移植上,不會有原任務單
   擔心的「x87 80-bit vs float64 精度差」問題——除非之後在別的子系統(未反組譯
   到的部分)發現真正的 x87 指令,屆時要重新評估。
5. **行動分派**:普通攻擊(玩家與怪物共用)可以合併成一套邏輯(§7 已驗證
   `FUN_1990_0002`+`FUN_138d_25da` 是共用路徑)。其餘 7 個選單動作(法術/道具/
   逃跑/特殊技能)因為 case 對應未完全解出,建議留到下一輪反組譯定案後再實作,
   避免用猜測的分派表寫進引擎造成行為偏差。

---

## 9. 最關鍵的未解問題(供下一輪接續)

1. **RNG 精確遞迴公式**:`FUN_30c2_0006` 的乘法同餘結構已看懂大致形狀,但
   `FUN_3016_0068`(除法輔助)內部演算法沒解出,導致沒辦法寫出一行乾淨的
   `state = f(state)` 公式。如果要逐位元對拍,這是必須突破的點。
2. **隊伍全滅(戰敗)判定的確切呼叫點**:只找到「掃描隊伍存活」的子函式
   (`FUN_138d_1ceb`),沒追到「呼叫它、判定戰敗、印訊息」的上層邏輯。
3. ~~**完整的選單 case ↔ 動作名稱對照表**~~ —— **已於 2026-07-25 解決**,
   全部 8 個動作的 case 編號見 `docs/re/09`(讀原始跳表 `138d:258f` 取得)。
   本項原文寫「只確認 `0xa`=普通攻擊」,**該認定本身是錯的**(0xa 實為 Examine)。
4. **`field_4ed6`、`field_4ed2`、`0x64e` 旗標表的確切語意**:命中/傷害公式裡多處
   用到這些欄位做加成判斷,已經確認「有這個機制」但沒有逐一定案是哪個具體
   遊戲規則(職業技能?種族被動?)。
5. **裝備護甲最終數值表**(`31f0:16c5`)與武器附魔加成的另一張表(`+0x15`/`+0x17`
   偏移那段)未逐位元核對,只確認了「有這個查表機制」。

---

## 附:可重跑的驗證片段

```bash
cd /home/anr2/cht/daemon_winter

# 1. 字串錨定驗證(戰鬥字串全部落在 138d/1990 段)
grep -F "End of round" workplace/ghidra/export/strings.csv
grep -n "0x6cb\b" workplace/ghidra/export/disassembly.asm

# 2. 武器傷害骰表(§3.2),直接讀取原始位元組核對
python3 -c "
data = open('workplace/orig/demwin/DEMON.INT','rb').read()
seg, off = 0x31f0, 0x1785
fo = seg*16 + off - 0xC400
print([data[fo+i] for i in range(14)])
"

# 3. 排除 x87 誤判 / 確認軟體浮點庫路徑(§5)
objdump -D -b binary -m i386 -Maddr16,data16 workplace/orig/demwin/DEMON.INT 2>/dev/null \
  | grep -inE '\b(fld|fmul|fadd|fistp|fstp|fdiv|fcom|fild|fsub|fwait|fnstcw|fldcw)\b'

# 4. RNG 呼叫鏈(§4)
grep -n "FUN_1d9f_0e0b(" workplace/ghidra/export/decompiled_all.c | wc -l   # 234 次呼叫
cat workplace/ghidra/export/decompiled/1d9f_0e0b_FUN_1d9f_0e0b.c
cat workplace/ghidra/export/decompiled/30c2_0006_FUN_30c2_0006.c
```
