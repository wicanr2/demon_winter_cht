# 70 — `T2C.TXT` / `T2D.TXT` 的觸發條件

`docs/formats/event-script.md` §5 從一開始就列著未解項：
「`T2C.TXT`、`T2D.TXT` 的觸發條件與 `T.TXT`／`EREGORE.TXT`／`WIN.TXT` 的觸發時機。」

前半解掉了。

---

## 1. 先修正那張表的位址

`docs/re/60` §3 說四個檔名「透過遠指標表索引存取，所以掃 offset 掃不到」。
**表的位址記錯了**，所以掃不到是掃錯地方。

正確的結構：

```
ds:0x2f3e  42 2f f0 21              ← T2C.TXT 的 far ptr（單獨一筆）
ds:0x2f42  "T2C.TXT"
ds:0x2f4c  "CYPHER.SHP"
ds:0x2f59  6b 2f f0 21              ┐
ds:0x2f5d  77 2f f0 21              ├ 三筆連續：EREGORE / WIN / T
ds:0x2f61  7f 2f f0 21              ┘
ds:0x2f6b  "EREGORE.TXT"
ds:0x2f77  "WIN.TXT"
ds:0x2f7f  "T.TXT"
```

掃 `ds:0x2f3e` 立刻命中兩處。

---

## 2. 兩個檔名是同一個字串

```
1ae7a  les bx, party
1ae7e  cmp party[+0xa3], 3          ; 子地圖 < 3？
1ae84  jae 0x1ae96

       ; 子地圖 1 或 2
1ae86  [bp-6] = 0x25d               ; 605
1ae8b  les bx, ds:0x2f3e            ; T2C.TXT 的指標
1ae8f  es:[bx+2] = 'C'              ; 第 3 個字元 → C

       ; 子地圖 >= 3
1ae96  [bp-6] = 0x141d              ; 5149
1ae9b  les bx, ds:0x2f3e
1ae9f  es:[bx+2] = 'D'              ; 第 3 個字元 → D

1aeb2  push 檔名指標、緩衝區、偏移
1aebf  call 1d9f:0a8b               ; 素材載入器
```

**`T2C.TXT` 與 `T2D.TXT` 從來不是兩個字串** ——
資料段裡只有一份 `"T2C.TXT"`，第 3 個 byte 被就地改成 `C` 或 `D`。

這就是為什麼 `strings` 掃不到 `T2D.TXT`：**它在檔案裡不存在。**

### 條件

| 子地圖 `+0xa3` | 檔名 | 讀取偏移 |
|---|---|---|
| 1、2 | `T2C.TXT` | 605 |
| ≥ 3 | `T2D.TXT` | 5149 |

兩者不只換檔名，**讀的偏移也不同** —— 同一份資料的兩段。

---

## 3. 同一招早就記過

載入器 `FUN_1d9f_0a8b` 不是新東西：

- `docs/re/07` §38 從「誰填了這塊緩衝區」找到它
- `docs/re/17` §221 記著「它**也會把 `.FNT` 的副檔名改成 `.FNE`**」

**就地改寫檔名字元是這個引擎的既有手法**，`docs/re/17` 已經寫下來了。
如果一開始就查那一行，這一輪會快很多 —— 又一次同樣的教訓。

---

## 4. 還沒解的

`ds:0x2f59` 那三筆（`EREGORE.TXT`／`WIN.TXT`／`T.TXT`）**仍然沒有引用**。
掃過 `mov ax/bx/dx/cx, imm16` 與 `les bx, ds:xxxx` 都沒命中，
所以它們是用**算出來的索引**存取的（例如 `表基底 + n×4`）。

不過 `WIN.TXT` 已經不重要 —— `docs/re/61` §2 確認結局文字寫死在
`ds:0x066a`，不是從檔案讀的。剩下 `EREGORE.TXT` 與 `T.TXT`。

`CYPHER.SHP` 夾在中間也還沒查（`docs/re/65` 猜與兩道密碼有關）。

---

## 5. worklist

- **A3 的「其餘劇情文字時機」部分解決**：`T2C`／`T2D` 完整
- 更正 `docs/re/60` §3 的表位址（`0x2f3e` 與 `0x2f59` 是兩張，不是一張）
- 剩 `EREGORE.TXT`／`T.TXT` 的觸發、`CYPHER.SHP` 的用途
