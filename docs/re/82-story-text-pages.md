# 82 — 劇情文字頁：`25be:19d1(page, mode)` 與那三個「沒有載入路徑」的檔案

`docs/re/70` §4 下過一個結論：`EREGORE.TXT`／`WIN.TXT`／`T.TXT`
「**在這份執行檔裡沒有載入路徑**」。

`docs/re/81` 用 DOSBox 播出第二場夢，畫面上的字**逐字就是 `T.TXT` 的第二頁**。
所以那個結論錯了。這一篇把載入路徑讀出來。

---

## 1. `25be:19d1(page, mode)` ＝ 分頁劇情文字的顯示器

```
1b1b1  FUN_25be_19d1(page = [bp+6], mode = [bp+8])
1b1b7  if (page >= 0) goto 1b20e             ; ★ page < 0（0xffff）才走載入
1b1bd  bx = mode * 2
1b1c4  ax = ds:[0x2f65 + bx]                 ; ★ 檔案大小表（3 筆 word）
1b1c9  cwd ; push dx:ax                      ; 長度（long）
1b1cb  push ds:0x4c8a / ds:0x4c88            ; 目的緩衝區
1b1d3  bx = mode * 4
1b1dc  push ds:[0x2f5b + bx]                 ; ★ 檔名遠指標表（3 筆 far ptr）
1b1e0  push ds:[0x2f59 + bx]
1b1e4  push ds:0x522a
1b1e9  call 1d9f:0a8b                        ; 素材載入器
```

**`mode` 選檔案，`page` 選頁。** `page == 0xffff` 是「把檔案載進來」，
`page >= 0` 是「顯示第 page 頁」。這就是為什麼每次呼叫都成對出現：

```
FUN_25be_19d1(0xffff, m);   ← 載入 mode m 的檔案
FUN_25be_19d1(n, m);        ← 顯示第 n 頁
```

### 兩張表，值全部對得上

| `mode` | `ds:0x2f59 + mode*4` | `ds:0x2f65 + mode*2` | 實際檔案大小 | |
|---|---|---|---|---|
| 0 | `EREGORE.TXT` | 3621 | 3621 | ✓ |
| 1 | `WIN.TXT` | 3987 | 3987 | ✓ |
| 2 | `T.TXT` | 1670 | 1670 | ✓ |

**三個檔案大小逐一相符**，不是判讀，是對答案。

---

## 2. 為什麼 `docs/re/70` 找不到

`docs/re/70` §4 的推理是：

> ~~所以它們是用算出來的索引存取的~~ —— 這個推測已收回（`docs/re/71`）。
> 索引式存取的基底也要出現在某處，而基底同樣掃不到。
> 更簡單的解釋是：這三個檔案在這份執行檔裡沒有載入路徑。

**「用算出來的索引存取」那個原始推測是對的**，收回錯了。
而收回的理由「基底也要出現在某處，而基底掃不到」——
基底**確實出現了**，只是形式不是預期的那種：

```
1b1e0   ff b7 59 2f      push word ptr [bx+0x2f59]
                                      └── 基底在這裡，是 ModRM 的 disp16
```

當時掃的是「`0x2f59` 當**立即數**」（`mov ax, 0x2f59` 這種），
那個形式全檔 **0 次**。但索引式存取根本不需要把基底載進暫存器 ——
`[bx + 常數基底]` 一條指令就做完，基底是**位移**不是立即數。

> 這與 `docs/re/80` 踩的是**同一個形狀的錯**，只是方向相反：
> 那次是「掃常數位移、漏掉暫存器偏移」，這次是「掃立即數、漏掉常數位移」。
> 兩次都下了否定式結論（「沒有寫入端」「沒有載入路徑」），兩次都錯。
>
> **否定式結論要列出「我掃了哪些定址形式」才算數。**
> 8086 的同一個語意有好幾種編碼，掃一種就宣告不存在，命中率不會高。

---

## 3. `T.TXT` ＝ 三場夢，而且第三頁逐句對上程式碼

`*` 分頁，共 3 頁。

| 頁 | 內容 | 對應 |
|---|---|---|
| 0 | `Your sleep is disturbed by horrible dreams…` `Like the Night I have come / But at dawn I shall stay…` | 第一場夢（`+0xbd`）|
| 1 | `The Orb of Evertime now is yours…` 馬利馮的預言 | 第二場夢（`+0xb9` 1→2）|
| 2 | `As you sleep a thousand images pass through your mind…` | 第三場夢（神殿全毀）|

**第 2 頁把 `docs/re/79`／`docs/re/80` 讀出來的程式碼逐句敘述了一遍：**

| 文字 | 程式碼 |
|---|---|
| `Clerics all over Ymros wake up with the feeling that something has been torn from their souls` | `char[i].+0xd7 = 0`／`+0xd8 = 0`（薩滿與司祭技能）|
| `Images of great flaming thunderbolts turning temples to ruins appear again and again` | `+0xba = 0xff` → 繪製時神殿 tile `0x25` 換成廢墟 `0x5b` |
| `The gods are dead.` | `char[i].+0xf0 = 0`（信奉的神祇）|

**敘事與實作一句一句對得上。** 這是比任何靜態推論都強的佐證 ——
之前是從指令推出「信仰整套歸零」，現在遊戲自己用一頁文字說了同一件事。

---

## 4. `+0xbe` 的劇情有名字了：艾瑞戈爾與黑鏡

`docs/re/78` §6 解出 `+0xbe` 是一次性劇情的單向閂鎖，
但註明「事件本體 `25be:1ae2` 沒有字串，是哪一段劇情仍未定」。

那支函式呼叫 `25be:19d1(9, 0)` —— **mode 0 ＝ `EREGORE.TXT`，第 9 頁**：

```
The voice from the mirror now speaks to you:
"Your meddling has become a t[hreat]…"
```

`EREGORE.TXT` 共 11 頁，是**艾瑞戈爾（高階祭司）與黑鏡**那場戲：
他愈來愈慌張、不斷向鏡子求助，最後鏡子裡的聲音（馬利馮）直接對玩家說話。

**所以 `+0xbe`（城鎮全成廢墟）就是這場戲的結果。**
`docs/re/79` §4 讀出「`+0xbe != 0` 之後 tile `0x2e` 的城鎮走廢墟路徑」，
現在知道是誰立的旗標、以及演的是哪一段。

攻略 `part-4` §135 那句「世界已成廢墟，所有城鎮都消失了」正好對上。

---

## 5. `WIN.TXT` ＝ 結局 7 頁

`docs/re/04` §5.1 讀出結局序列用 `mode 1` 播頁 0、1、2、3、5 ——
`WIN.TXT` 切出來正好 **7 頁（0–6）**，範圍相符。
頁 4／6 沒被那條路徑用到（`docs/re/04` 記著結局有一個是非題分歧，
所以有頁只在另一條分支出現）。

---

## 6. 對中文化

三個檔案是**主線最重要的敘事文字**，而且結構單純（`*` 分頁、無壓縮、純 ASCII）：

| 檔案 | 大小 | 頁 | 內容 |
|---|---|---|---|
| `T.TXT` | 1670 | 3 | 三場夢 |
| `WIN.TXT` | 3987 | 7 | 結局 |
| `EREGORE.TXT` | 3621 | 11 | 艾瑞戈爾與黑鏡 |

目前引擎**一頁都還沒接** —— `docs/re/61` §2 的結局是拿寫死的
`ds:0x066a` 那一句頂著，夢境與艾瑞戈爾整段都沒有。
接上之後 C 系列會多出約 9 KB 的敘事文字要翻。

---

## 7. 對 worklist

- **`docs/re/70` §4 推翻**：三個檔案都有載入路徑（`mode` 索引兩張 3 筆表）
- **`docs/re/71` 的收回本身要收回**：「用算出來的索引存取」原本是對的
- **`25be:19d1` 完整解出**：`mode` 選檔、`page` 選頁、`0xffff` 是載入
- **`+0xbe` 的劇情定案**：艾瑞戈爾與黑鏡（`EREGORE.TXT` 第 9 頁）
- **`T.TXT` 第 2 頁與程式碼逐句相符** —— `docs/re/79`／`80` 的獨立佐證
- 新增（C 系列）：三個劇情文字檔共約 9 KB，引擎一頁都還沒接
- 仍未解：寫 `+0xb9 = 1` 的指令（`T2C.TXT` 那段拿寶珠的事件裡，
  但那支函式的寫入端還沒找到 —— 下一步從 `1ae7a` 一帶往下讀）
