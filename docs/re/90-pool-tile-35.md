# 90 — tile `0x35` 不是牆，是可以喝的水池

`docs/re/05` §3 把 tile `0x35` 記成「寫死的阻擋（**推測**：撞牆／無法進入）」，
`docs/re/58` §2 已經指出那一格對應動作 `0x11`、字串是 `The pool is empty`，
但沒有往下讀。引擎照舊把它當 `TriggerHardBlock` —— **踩上去什麼都不會發生**。

讀完 `222f:37c4` 之後：那是一口**治療用的水池**，一天七次。

---

## 1. 動作 `0x11` 的全文（`222f:37c4` ＝ `0x196b4`，312 bytes）

`docs/re/58` 記的「56 bytes」是字串錨定當時的估法，實際到 `0x197e9`。

```
196bb  ds:0x15ce = 0x80                    ; 這一步不要再觸發 tile 事件
       loop:
196c1  call 1cdc:1727                      ; 清文字框
196ca  if (trailer[+0xaa] == 0) {
196d2      印 "The pool is empty"
196df      等按鍵
196e4      return 2
       }
196ec  印 "Enter Character to"
196f9  印 "drink from pool"
1970e  0x990:179b(0, 5, 1)                 ; 畫選人清單
19722  who = 0x990:179b(2, 5, 1)           ; 選人
19741  if (who == trailer[+0x9a]) return 2 ; 選到「隊伍人數」那一格 ＝ 取消
1974a  trailer[+0xaa]--                    ; ← 扣一次
1975b  cur  = char[who][+0xfd]             ; 目前 HP
19769  heal = Roll(4)                      ; 1–4
1977e  max  = char[who][+0xfc]             ; 最大 HP
1978d  if (max < cur + heal) heal = max − cur     ; 鉗到上限
19795  sprintf(buf, "He is healed %d", heal); 印出來
197c3  char[who][+0xfd] += heal
197d2  重畫 + 等按鍵
197e6  goto loop                           ; ← 回去再問一次
```

`0x990:179b(參數, 5, 1)` 就是 `docs/re/33` §4 那組「畫選人清單／讀選擇」的
共用常式，取消的判別式（選到編號 == 隊伍人數）也與 Trade 那裡一字不差。

`+0xfc`／`+0xfd` 是角色記錄的最大／目前 HP —— 相對 `expOffset`(`0xc4`)
就是 `0x38`／`0x39`，`internal/assets/scenario/save.go` 早就命名好了
（`attrMaxHPOffset`／`attrCurrentHPOffset`）。**這裡不要再拿字面值去 grep**，
理由見 `docs/re/89`。

## 2. `+0xaa` ＝ 一天七次

全檔只有三處碰 `trailer+0xaa`，掃三種定址形式就掃乾淨：

| 位址 | 動作 |
|---|---|
| `0x196ca` | `cmp byte es:[bx+0xaa], 0` → 空了 |
| `0x1974a` | `dec byte es:[bx+0xaa]` → 喝一次 |
| `0x1eee6` | `mov byte es:[bx+0xaa], 7` → **睡覺補回 7** |

`0x1eee6` 落在 `2aed:03e4`，就是印 `You sleep.` 的那支（`docs/re/26` 的休息常式）。
同一段還把 `+0xa7` 設回 1（光源）、`+0xac`／`+0xad`／`+0xae` 清 0（每日旗標）。

所以 `+0xaa` 不是「這口池子還剩幾口水」，而是**隊伍層級的每日次數**，
與觀地（`+0xac`）同一組，只是額度是 7 不是 1。**換一口池子也不會回滿。**

## 3. 兩個「pool」不是同一件事

| | 治療水池 | 水池陷阱 |
|---|---|---|
| 觸發 | 走向 tile `0x35` | `nSS.DAT` 類別 3／6 的陷阱表 |
| 效果 | 選一人，回 1–4 HP | 掉進去，每回合 `Roll(3)−1` 傷害（酸池 +2）|
| 出處 | 本文 | `docs/re/68` |

手冊「地底 → 陷阱」列的是**後者**；前者手冊沒寫。
名字撞在一起，別把兩邊的數值搬來搬去。

## 4. 隊伍不會走上去

回傳 `0x11` 發生在 `222f:0b0e` 的 tile 檢查段，**在寫座標之前**
（`docs/re/58` §2 的表裡 `0x13`／`0x14` 都註明「寫座標」，`0x11` 沒有）。
所以喝完水隊伍還站在原地，面向水池。

`ds:0x15ce = 0x80` 是那一段檢查的閘門（`if (ds:0x15ce < 0x80)`，`docs/re/05` §3）。
水池在進場時就把它關掉，等下一次移動指令（`0x16cef` 清 0）才重新開啟 ——
否則同一步會不斷重入。這個手法與 `docs/re/83` 的地點劇情（`0x19f4f`）一樣。

## 5. 對引擎的影響

`internal/game/events.go` 的 `TriggerHardBlock`／`tileHardBlock` 是**被推翻的命名**：
`0x35` 不是「寫死的阻擋」，是「不移動，但要跑一段互動」。
`docs/spec/03-events.md` §觸發閘門那一行同樣要改。

## 6. 沒做完的

- 動作 `0x09` `Take:`／`0x0b` `Move:`（共用 `222f:2da5`）仍未讀。
  水池讀完之後，`docs/re/58` §4 的清單只剩這兩格與它們背後的地城道具系統。
