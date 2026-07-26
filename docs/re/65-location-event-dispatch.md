# 65 — 光之環下游那張 16 格表：地點劇情事件

`CONTEXT.md` §7 的 B7 列了六張未讀的 switch 跳表，`0x1a5f3` 是其中一張，
也是 `docs/re/59` §6 留下的「光之環裡面有什麼」。

讀完發現它是**主線劇情事件的分派表** —— 兩個密碼謎題就在裡面。

---

## 1. 先把表定位出來

`0x1a5f3` 的指令是 `jmp cs:[bx+0x0deb]`。難點在 `cs` 是執行期的值，
靜態看不到 —— 但可以用**已知的一張表反推公式**：

`docs/re/50` 那張的分派是 `jmp cs:[bx+0x563]`，表在檔位移 `0x16453`，
所以 `cs base（檔位移）= 0x16453 − 0x563 = 0x15ef0`，
而 `0x15ef0 + 0xC400 = 0x222F0 = 0x222f × 16` —— **segment 就是 `222f`**，
與文件記的一致。公式確立：`cs base = seg × 16 − 0xC400`。

拿這條去試 `0x1a5f3`：

| segment | 表位址 | 16 個目標落在函式內 |
|---|---|---|
| `222f` | `0x16cdb` | 5/16 |
| `1cdc` | `0x117ab` | 2/16 |
| **`25be`** | **`0x1a5cb`** | **16/16** |

`25be` 全中。而 case 15 指向 `0x1a565` —— 正是 `docs/re/59` §3 的
光之環三旗標檢查。**自我驗證**。

---

## 2. 這支就是移動分派表的動作 `0x12`

往前找函式入口：`0x19a43` ＝ **`25be:0263`**。

那正是 `docs/re/50` 表裡的動作 `0x12`（當時只認出字串 `"A trap!"`）。

所以事件是**兩層分派**：

```
走一步 → 222f:0b0e 回傳碼 → 21 格表（docs/re/58）
                              └ 動作 0x12 → 25be:0263
                                              └ 16 格表（本篇）
```

內層的選擇子是 **`ds:0x52f6`**（`0x19f55  mov ax,ds:0x52f6`），
緊鄰 `docs/re/58` 提過的 `ds:0x52f4`（tile 暫存）。
它在全檔有六處寫、七處比較，是個狀態變數 —— **誰在什麼時候設哪個值還沒追**。

---

## 3. 16 格的內容

| case | 位址 | 字串 | 判讀 |
|---|---|---|---|
| 0 | `0x1a5f8` | — | 預設（回 2）|
| 1 | `0x19f5b` | `You hear the whirring of massive machinery` | 機關啟動 |
| 2 | `0x19f84` | `Your party has been` / `crushed by the walls.` | **壓牆陷阱：全隊死亡** |
| 3 | `0x1a03c` | `%s%s%s` / `Do you approach?` | 接近詢問 |
| 4 | `0x1a0f9` | `I have been working on a new weapon.` / `It is designed for the ones` / `who wish to kill Xeres!` | **NPC：打造對付 Xeres 的武器** |
| 5 | `0x1a169` | `The tombstones` / `shift before you` | 墓碑移動 |
| 6 | `0x1a228` | — | 8 bytes，轉呼叫 |
| 7 | `0x1a230` | — | |
| 8 | `0x1a263` | — | |
| 9 | `0x1a2cf` | — | 8 bytes，轉呼叫 |
| 10 | `0x1a2d7` | `A spectral priest utters a chant:` / `'Power, Divinity, Spirit...' and` / `awaits the final word of the spell` / **`VOID`** | **密碼謎題** |
| 11 | `0x1a393` | `A voice from nowhere speaks: 'Only` / `those who worship Malifon may enter` / `this temple. What is thy name?'` / **`JESRIC`** | **密碼謎題** |
| 12 | `0x1a43f` | `Do you wish to ring the bell?` / `Nothing happens.` / `The sound of angels crying opens up from the heavens…` | 鐘 |
| 13 | `0x1a4b8` | `Do you wish to sleep here?` / `She bids you farewell and adds 'May the Ancients protect you...'` | NPC 過夜 |
| 14 | `0x1a54b` | — | 26 bytes，檢查 `party+0xbe` |
| 15 | `0x1a565` | `A crimson forcefield bars your entry to the Circle of Light` | **光之環的門**（`docs/re/59`）|

### 兩個密碼

**答案直接寫在資料段裡**：`VOID`（case 10）與 `JESRIC`（case 11）。
玩家要輸入的字串就是它們 —— 這種「把答案存成明文再比對」是那個年代的常態。

`docs/re/60` §3 那張檔名指標表裡的 **`CYPHER.SHP`** 應該與此有關
（cypher ＝ 密碼），但還沒確認。

### 順帶解掉一個 trailer 欄位的線索

case 14（`0x1a54b`）檢查 `party+0xbe`：不為 0 就回 3，否則呼叫 `15be:1ae2`。
`+0xbe` 是 `CONTEXT.md` §7 B8 列的未解閘門之一 —— 現在知道**它擋的是這一格**，
但語意仍未定。

---

## 4. 對 worklist 的影響

- **B7 少一張**：`0x1a5f3` 已讀（剩 `0x171ec`、`0x19ed7`、`0x1ab90`、`0x1bddc`、`0x1c0d8`）
- **A3 的「其餘劇情文字時機」有進展**：主線不只符印與禁錮，
  中間還有 Xeres 的武器、兩道密碼、壓牆陷阱這些關卡
- **新的未解**：`ds:0x52f6` 這個事件狀態變數誰在設

---

## 5. 方法：用已知的一張表反推 `cs base`

`jmp cs:[bx+d16]` 的難處是 `cs` 靜態未知，但只要手上有**一張已經定位過的表**，
就能反推公式（`cs base = seg × 16 − 0xC400`），再對候選 segment 逐一試，
用「16 個目標是不是都落在這個函式的位址範圍內」當判準。

這次 `25be` 是 16/16、次高的只有 5/16 —— **差距夠大就不必猶豫**。
而 case 15 指到一段已知程式碼（光之環的門），等於免費的交叉驗證。
