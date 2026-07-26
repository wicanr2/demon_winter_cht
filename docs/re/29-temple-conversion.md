# 29 — 改宗：`0x10 − god_id` 的真相

`docs/re/19` §3.5 把改宗解到只剩一個疑點，而那個疑點擋住了實作：

> `(0x10 - god_id)` 這個索引公式……`god_id` 2–9 會寫到 `+0xc8 + 14`（召喚）
> 以下的格子，語意可疑。需要逐一核對神祇 id 的編號方式。

**那個算式讀錯了一個運算元。** 減掉的不是神祇編號，是**神祇編號除以 2 的餘數**。

---

## 1. 兩個參數，不是一個

改宗那支函式從 `278d:0f38` 開始（`push bp / mov bp,sp`），**吃兩個參數**：

```
 f77  cmp char[+0xf0], [bp+6]      ; 已經信這位神了嗎 → 用 [bp+6]
 fd4  ax = 0x10 − [bp+8]           ; 算教派技能 id   → 用 [bp+8]
```

`docs/re/19` 把兩個都當成 `god_id`。呼叫端（`278d:0b50`）說得很清楚：

```
0b50  push [bp-4]        ; → [bp+8]
0b53  push [bp-0xa]      ; → [bp+6]
0b56  lcall 178d:0f38
```

---

## 2. `[bp+8]` 是「神祇編號 mod 2」

神殿入口 `278d:0930` 有兩條路徑，兩條都算出同一件事。

**固定神殿**（隊伍欄位 `+0xa8` 不是 `0xFF`）：

```
938  ax = party[+0xa8]        ; 這個地點的神祇編號
946  cmp ax,0xff / je 走座標表
94b  cx = 2 / cwd / idiv cx   ; ax ÷ 2
951  [bp-4] = dx              ; ★ 餘數
     [bp-2] = ax（＝神祇編號）
```

**座標表**（`ds:0x310d` 起，4 bytes 一筆 `[X, Y, a, b]`）：

```
95e  cl = [0x310d + bx]       ; X，比對 party.X
972  al = [0x310e + bx]       ; Y，比對 party.Y
984  [bp-4] = [0x3110 + bx]   ; ＝ b
98d  cx = [0x310f + bx] * 2   ; ＝ 2a
99a  cx −= [bp-4]
99c  [bp-2] = 2a − b          ; ＝ 神祇編號
```

表裡存的是 `(a, b) = (⌈god/2⌉, god mod 2)`，所以 `god = 2a − b` 而 `b = god mod 2`。
兩條路徑得到的 `[bp-4]` **都是 0 或 1**。

於是：

```
教派技能 = 0x10 − (神祇編號 mod 2)
         = 16 − 1 = 15  （奇數神 → 薩滿 Shamen）
         = 16 − 0 = 16  （偶數神 → 司祭 Priesthood）
```

索引永遠只會落在 15 或 16 —— 不會掉到「召喚」以下。疑點結案。

> 技能 15 = Shamen、16 = Priesthood 由 `docs/re/21` 的技能 id 表與
> FILES.DTT `[91:123]` 的順序交叉確認。

---

## 3. 十位神交錯排列，第 11 位是例外

`ds:0x310d` 的座標表列出 19 座野外神殿，`2a − b` 算出來的神祇編號涵蓋 **1–10**：

| 神祇 | 名稱 | 教派 |
|---|---|---|
| 1 | Omizeh | 薩滿 |
| 2 | Balmur | 司祭 |
| 3 | Gamur | 薩滿 |
| 4 | Vemarkn | 司祭 |
| 5 | Acisc | 薩滿 |
| 6 | Maldorath | 司祭 |
| 7 | Volobews | 薩滿 |
| 8 | Illo | 司祭 |
| 9 | Theryni | 薩滿 |
| 10 | Camear | 司祭 |
| 11 | Ancient | —（見下）|

奇偶交錯，正好是 `docs/re/19` §8 講的「5 位司祭神 ＋ 5 位薩滿神」。
名稱來自 FILES.DTT `[153:164]`，索引是神祇編號減一（`docs/re/27` §4）。

**第 11 位 Ancient One 不在座標表裡**，只能從 `party+0xa8` 進入，而且入口有專屬分支：

```
9dd  cmp [bp-2], 0xb / jne 一般流程
9e3  印 "Before you is an oddly shaped temple."
     印 "Its priests claim to worship an entity"
     印 "known as the Ancient One."
```

`0x0b mod 2 = 1` → 薩滿。

> **順帶：`party+0xa8` 是雙用途欄位。** `docs/re/19` §2 把它記成「當前所在位置的
> 學院教授技能 ID」，這裡神殿把同一個 byte 當神祇編號讀，兩邊都用 `0xFF`
> 代表「這裡沒有，去查座標表」。它其實是「目前站著的特殊地點的參數」——
> 又一個 1980 年代「一個欄位兩種用途」的例子。

---

## 4. 改宗做了什麼

```
 f70  char[+0xf0] == god_id      → "You already worship our diety."
 f98  char[+0xd7] != 0
 fa0  或 char[+0xd8] != 0        → "You are already devoted to a different god"
 fc5  剩餘智力點數 = FUN_278d_1149(角色)
 fd1  idx = (0x10 − order)*10 + char.class(+0xf6)
 ffc  需要點數 = 技能學費表[0x5508][idx]
1008  需要 > 剩餘                 → "You don't have enough points left"
1057  char[+0xc8 + (0x10 − order)] = 1     ; 學會薩滿或司祭
106e  char[+0xf0] = god_id
107f  char[+0xeb] = 0x14 (20)              ; 祈禱成功率
```

**不扣金幣**，收的是智力點數 —— 與學院教技能查同一張表、用同一個「剩餘點數」函式。

`+0xd7` = `0xc8 + 15`（薩滿旗標）、`+0xd8` = `0xc8 + 16`（司祭旗標）。
兩者任一非零就拒絕，所以**一輩子只能改宗一次**。

---

## 5. 順帶訂正祈禱的前置檢查

`docs/re/19` §3.3 寫：

> `278d:0d9a` 若「已建立信仰關係」旗標非零，才做下面的信仰比對；
> 否則直接略過（新關係／首次互動時不擋）

**跳轉方向讀反了。** 實際是：

```
d8a  si = 角色索引*0x104 + (0x10 − order)
d9a  cmpb $0, char[+0xc8 + ...]
da0  je 0xdb6                     ; 旗標為零 → 跳去「You don't worship %s」
daa  al = char[+0xf0]
db1  cmp ax, [bp+6]
db4  je 0xdf4                     ; 神也相符 → 才進祈禱
      否則落到 0xdb6 拒絕
```

所以祈禱**要求已經改宗到這座神殿**：既要有這個教派的技能，信奉的神也要正好是這一位。
本專案照舊文件寫成「沒有信仰的人不擋」，已一併訂正。

兩條合起來才是完整的流程：**先改宗、才祈禱得了。**

---

## 6. 本專案的實作範圍

`internal/game/services.go` 的 `DeityOrder`／`ConvertAtTemple`，
神殿畫面 `V` 鍵。實機：Norman 在海濱鎮改信 Camear，學會司祭（花 2 點智力），
祈禱成功率立刻變 20%。

**沒解**：`FUN_278d_1149`（剩餘智力點數）本專案是自己依
「智力 − Σ已學技能成本」算的，與原版函式沒有逐指令核對過 ——
數值在起始隊伍上對得起來（`docs/spec/05` 的驗收條件），但那是黑箱比對。
