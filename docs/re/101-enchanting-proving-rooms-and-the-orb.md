# 101 — A7 最後四格：附魔工坊、十間試煉室、惡魔水晶與永恆之寶珠

`docs/re/65` §3 對 case 6／7／8／9 **連字串都沒有**（6 與 9 各 8 bytes
是轉呼叫，7 與 8 沒抄）。讀完之後這四格不是零碎小事，
而是三個各自成塊的子系統 —— 其中兩個直接關掉了別處掛著的問題。

---

## 0. 結論先講

| case | 位置 | 是什麼 | 影響 |
|---|---|---|---|
| 6 | 地圖 2 (28,5) | **矮人大師的附魔工坊** | **worklist C2 的入口找到了** —— 不在城鎮設施裡，是一格地點劇情 |
| 9 | 地圖 2，十格 | **十間試煉室**（一個職業一間）| 十座標與資料表 **10/10 零誤差**；過關寫 `+0x8a + 索引` |
| 8 | 地圖 2 (42,28) | **永恆之寶珠**（道具 29）—— 要十間試煉室全過 | **`docs/re/80` §3 那個「找不到的 `+0xb9 = 1` 寫入端」就是這裡** |
| 7 | 地圖 4 (7,4) | **惡魔水晶**（道具 28）| 送道具跳表 param 5 的呼叫端；跳表七格到此全部有主 |

**四格都還沒接進引擎。** 這一篇只是把反組譯收攏起來 ——
附魔是一整套服務、試煉室是十間，兩者都不是「一格小工作」。

---

## 1. case 6 —— 附魔工坊（`1990:21d1` ＝ `0x0f6d1`）

case 6 本體只有 8 bytes，是個轉呼叫：

```
1a228  CALLF 1000:bad1       ; ＝ 0x0f6d1
1a22d  JMP 收場（回傳它的回傳值）
```

那支的開場：

```
0f6e9  1000:b09b(0, 5, 1)                        ; 選角色（第一段）
0f702  印 "Before you is a fabulous workshop"
0f70f  印 "where gleaming weapons are being"
0f71c  印 "enchanted by Dwarven Masters."
0f72d  印 "Character with item to enchant?"
0f73a  who = 1000:b09b(2, 5, 1)                  ; 等選擇
0f75a  if (party[+0x9a] == who) return 0         ; 選到「取消」那一格 → 直接回
0f766      party[+0xa2]++                        ; ← 順帶把 Y 加一（推開一格）
0f781  1000:916d(who, 3, 0)                      ; 列道具
0f790  印 ds:0x1116
0f7a5  slot = 1000:916d(who, 3, 2)               ; 選一件
0f7b3  if (slot == 10) → 取消
0f7bb  ptr = 角色[who]×0x104 + slot×17 + 0xc      ; 那一格的 17 bytes
0f7e8  if (ptr[+0x00] > 0x1b) …                  ; 型別 > 27 → 不能附魔
       …
```

**這就是 worklist C2 要找的入口。** `docs/re/55` 從攻略的 80 個點反推出
`F(n,c) = 35 × (20−c) × n^1.7`，但一直找不到原版哪裡提供這個服務 ——
因為它**不在城鎮八設施、也不在市集 12 選項裡**，而是地圖 2 上的一格劇情事件。

> 教訓與 `docs/re/79`（tile `0x5b`）同型：**「城鎮設施裡沒有」不等於「遊戲裡沒有」。**
> 找一個服務的入口時，地點劇情表和城鎮選單一樣要查。

`party[+0xa2]++`（Y 加一）那一行值得留意：選到「取消」以外的路徑會把隊伍
往南推一格 —— 大概是避免站在同一格上重複觸發。**還沒讀完整支**，
費用公式那一段尚未定位，所以 C2 只從「入口未知」降級成「入口已知、公式待對」。

---

## 2. case 9 —— 十間試煉室（`1990:1cf5` ＝ `0x0f1f5`）

同樣是 8 bytes 的轉呼叫（`1a2cf  CALLF 1000:b5f5` ＝ `0x0f1f5`）。

```
0f20c  idx = -1
       for i := 0; i < 10; i++ {
0f21a      if (ds:0x0cae[i] != party[+0xa1]) continue      ; X 表
0f234      if (ds:0x0cc2[i] != party[+0xa2]) continue      ; Y 表
0f247      idx = i
       }
0f25e  party[+0xab] = idx + 1                              ; 記下「在第幾間」
0f276  sprintf("Proving Room of the %s.", ds:0x17e1[idx])   ; 職業名
0f2a6  sprintf("You are in a %s room",    ds:0x0c86[idx])   ; 顏色
0f2cd  印 "with highly polished floors"
0f2db  印 "and walls, bare of furnishings."
0f2f1  印 "The room is suddenly flooded" …
```

### 2.1 三張十格表

| 索引 | X | Y | 職業（`ds:0x17e1`）| 顏色（`ds:0x0c86`）|
|---|---|---|---|---|
| 0 | 49 | 12 | Ranger | green |
| 1 | 35 | 37 | Paladin | silver |
| 2 | 49 | 5 | Barbarian | brown |
| 3 | 56 | 31 | Monk | violet |
| 4 | 35 | 12 | Cleric | white |
| 5 | 42 | 13 | Thief | grey |
| 6 | 49 | 37 | Wizard | crimson |
| 7 | 42 | 5 | Sorcerer | black |
| 8 | 28 | 31 | Visionary | blue |
| 9 | 35 | 5 | Scholar | beige |

**`2SS.DAT` 的 case 9 剛好十筆，座標與順序完全相同 —— 10/10 零誤差。**
（這是這一輪唯一需要的交叉驗證：座標表在程式碼裡、觸發格在資料檔裡，
兩邊獨立，對得上就不是巧合。）

### 2.2 過關的旗標是 `+0x8a` 起的十個 byte

`tools/fieldscan.py 0x8a` 全檔只有三處：

```
0x0e1d2  mov byte [bx+si+0x8a], 1
0x0f677  mov byte [bx+si+0x8a], 1     ← 在這支函式的尾段
0x1a29c  cmp byte [bx+si+0x8a], 0     ← case 8 的檢查（見 §3）
```

`si` 選的是第幾間，所以 `+0x8a`–`+0x93` 是**十間各一格的「過了沒」**。

---

## 3. case 8 —— 永恆之寶珠，以及 `+0xb9 = 1` 的寫入端

```
1a267  if (party[+0xb9] != 0) → 只放個音效就收場          ; 拿過了
1a26f  2000:020a(4); 2000:7345(); 2000:7274(2)            ; 演出
1a288  count = 0
       for i := 0; i < 10; i++
1a29b      if (party[+0x8a + i] == 0) count++             ; ← 數「還沒過的」
1a2af  if (count != 0) → 只放個音效就收場                 ; **十間沒全過就沒有**
1a2b5  25be:11ff(6)                                       ; 送道具 param 6
```

跳表 param 6 的分支是 `[+0x00] = param + 0x17` ＝ **型別 29**，
而 `ITEMS.DAT` 第 29 件是 **`Orb/Evertime`**（永恆之寶珠）。

### 3.1 `docs/re/80` §3 掛了兩輪的那個缺口，答案在這裡

`docs/re/80` §3 記著：

> ⚠ **`+0xb9 = 1` 的寫入端找不到**，而攻略證明它存在 →
> `tools/fieldscan.py` 只掃常數位移，**否定式斷言不能只靠它**。

而 `docs/re/81` 拿 DOSBox 裁決之後，從馬利馮預言的第一行
`The Orb of Evertime now is yours` **推測寫入端在寶珠事件裡**。

**那個推測是對的，而且這一輪靜態就對上了。** 送道具常式的旗標寫入是

```
1a9f3  ptr = party + 0xb3            ; 存成區域遠指標 [BP-0xc]/[BP-0xa]
1aacc  LES BX,[BP-0xc]
1aacf  MOV SI,[BP+6]                 ; param
1aad2  MOV byte ptr ES:[BX+SI],0x1   ; ← party[0xb3 + param] = 1
```

param 6 → `0xb3 + 6` ＝ **`+0xb9`**。

**為什麼掃不到**：`fieldscan.py` 找的是 `disp16 == 0xb9` 的常數位移。
這一處的 disp 是 **0**（基底已經是 `party+0xb3`），偏移在 `si` 裡。
`docs/re/80` §3 追過的那 20 處 `mov es:[bx+si], al` 也不含它 ——
那些的基底是結構本身或區域陣列，而這一處的基底是**結構加一個常數之後
存進區域變數的遠指標**。三種形狀都要想到才掃得全。

### 3.2 所以 `+0xb9` 是「寶珠拿過了」還是「劇情階段」？

**兩個都是。** 同一個 byte：

- 對送道具常式而言它是**旗標陣列的第 6 格**（case 8 自己的閘門也讀它）。
- 對睡覺流程而言它是**劇情階段**（`docs/re/80`：`== 1` 播第二場夢並進階段 2；
  `docs/re/81` 用 DOSBox 驗過）。

拼起來就是一條完整的因果：**十間試煉室全過 → 拿到永恆之寶珠 →
`+0xb9` 變 1 → 下一次睡覺播馬利馮的預言 → 劇情進階段 2。**
`docs/re/81` dump 到的那句 `The Orb of Evertime now is yours` 正是預言的第一行。

> ⚠ **`scenario.PlotGiftCount` 是 6 而不是 7 的那個警示要改寫。**
> 目前的註解說「param 6 會寫到 `+0xb9`，兩者衝突，在查清楚之前不擴到 7 格」——
> 現在查清楚了：**不是衝突，是刻意共用**。但**也不要就這樣擴成 7 格**，
> 因為 `+0xb9` 已經有 `plotStageOffset` 這個名字與讀取端；
> 兩個名字指同一個 byte 才是真正要小心的地方（`one-implementation-per-rule`）。
> 正確做法是讓 case 8 直接寫 `PlotStage = 1`，而不是多一格旗標。

---

## 4. case 7 —— 惡魔水晶（`0x1a230`）

```
1a230  2000:7274(4)                              ; 音效
1a23a  if (party[+0xb8] != 0) return 2            ; 0xb3 + 5，拿過了
1a24c  2000:7274(1)
1a256  25be:11ff(5)                               ; 送道具 param 5 → 型別 28
```

`ITEMS.DAT` 第 28 件是 **`Demon Crystal`**（惡魔水晶）——
主線解謎提示鏈裡本來就有它。

**跳表七個 param 到此全部有主**（`docs/re/99` §2 那張表的最後兩格）：

| param | 呼叫端 | 道具 |
|---|---|---|
| 0–3 | case 3 兵器庫四座台座 | 銀鏈甲／釘頭鎚／水晶匕首／冰藍短劍 |
| 4 | case 4 鐵匠鋪 | 闊劍（附魔 −1）|
| 5 | **case 7**（地圖 4 (7,4)）| **惡魔水晶** |
| 6 | **case 8**（地圖 2 (42,28)）| **永恆之寶珠** |

---

## 5. 對 worklist 的影響

- **A7 的反組譯全解**（16 格表讀完），但**引擎只接了 12 格**。
  剩下四格各自的規模：
  - case 7 最小：一個旗標 ＋ 送道具 param 5（型別 28）。
  - case 8 中等：十個旗標的檢查 ＋ 送道具 param 6，而且要與
    `plotStageOffset` 對齊（見 §3.2）。
  - case 9 較大：十間試煉室，而「過關」那一段（`0x0f677` 寫旗標之前）還沒讀。
  - case 6 最大：**整套附魔服務**（＝ C2）。
- **C2 從「入口未知」降級**：入口是地圖 2 (28,5)。費用公式那一段還沒定位，
  所以 `docs/re/55` 的 `F(n,c)` 仍是假說。
- **`docs/re/80` §3 的缺口關閉**，而且那條「否定式斷言不能只靠 fieldscan」的
  教訓要補一個形狀：**基底存進區域遠指標之後的 `[bx+si]` 寫入**。
