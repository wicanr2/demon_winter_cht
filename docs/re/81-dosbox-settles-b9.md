# 81 — DOSBox 裁決 `+0xb9`：路徑是活的

`docs/re/80` §3 留下一個衝突：反組譯**普查**顯示 `+0xb9` 沒有任何一處寫 1
（常數位移 19 處全列、計算式寫入 13 處全追、loader 也查了），
所以「從 0 出發永遠動不了」；但攻略說那兩場夢確實會發生。

那一篇自己寫了裁決方式：**第 2 級 oracle —— DOSBox 實跑原版**。
這一篇就是去做，而且十分鐘就結案了。

---

## 1. 做法：把旗標改成 1，然後睡一覺

`workplace/dosbox/game/` 是可寫副本（`workplace/orig/` 唯讀）。
改三個 byte：

```
+0xb9 = 1     劇情階段設成「該播第二場夢」
+0xbd = 1     第一場夢當作已播，跳過
+0x9f = 16    時辰 16（15–24 才睡得著，docs/re/26 §36）
```

然後：

```
tools/dosbox_run.sh ega "wait:2;key:Return;wait:3;key:Return;wait:2;
                         key:c;wait:2;key:s;wait:3;shot:..."
```

`c` 紮營、`s` 睡覺。**結果第二場夢播出來了。**

---

## 2. 夢境全文（`93-malifon-dream.png`）

```
The Orb of Evertime now is yours
    But your quest remains in vain
I was alive before mortals touched the Earth
    And no mortal there can touch me.
It is true that with the Demon crystal
    I can lay not tooth nor claw upon you
But I can make you wish
    you were never born
Let these the words of Malifon be heard-
The sky shall Flame
    The Earth shall Bleed
All that is Dark shall rise up in glory
    And all that is Light
        Shall be trampled underfoot
```

三件事一次拿到：

1. **`+0xb9 == 1` 的分支是活的** —— 衝突結案。
   `docs/re/80` 的「照程式碼看永遠動不了」是**靜態掃描找不到寫入端**，
   不是那條路徑死的。
2. **第一行就是觸發點**：`The Orb of Evertime now is yours`。
   攻略說的「拿到恆世寶珠、回祭壇向遠古者回報，離開後第一次睡眠」
   完全對上 —— **寫 `+0xb9 = 1` 的就是給寶珠那個事件**。
3. **文字拿到了**（`docs/re/80` §5 記著「文字頁的實際文字沒 dump」）。
   這是中文化要翻的內容，而且是主線最重要的一段預言。

### 那個寫入端為什麼掃不到（仍未定）

已排除：常數位移（19 處全列，opcode 無關重掃過）、
計算式寫入（13 處全追，`si` 值域 0–8 的陣型格）、`DEMON.EXE`。

剩下最可能的是**基底位移過的別名指標** —— 若某個遠指標指向
trailer 減去 `N`，那麼寫 `[bx + 0xb9 + N]` 就會落在 `+0xb9` 上，
而「掃 disp16 == `0x00b9`」看不到它。

**下一步很明確**：從寶珠事件那邊往下追，不要再從欄位那邊往上掃。
（`rulebook/62` 的靜態反追溯源，但這次方向要反過來。）

> 教訓：**「普查完整」不等於「結論完整」。**
> `docs/re/80` 的普查沒有錯（那 19 處確實沒有寫 1），
> 錯在把「掃不到」讀成「不存在」。
> 而裁決它只花了改三個 byte + 一次 DOSBox ——
> **比繼續擴大靜態掃描便宜一個量級。**
> 卡在靜態層時，先問「有沒有辦法直接跑一次」。

---

## 3. 順帶拿到原版兩張選單的完整文字

`docs/re/12` 記了「主指令迴圈分派 19 項」但沒有名字；
`docs/re/33` 解出紮營選單 14 項。實跑一次，兩張都在畫面上。

### 主指令選單（`94-orig-main-menu.png`）

```
Walk          Party info    Save Game     Camp          Look
Take          Drop          Move          Examine       Use
Inspect       View room     X)View item   Read descr…   Quit
```

`X)View item` 帶明示的鍵 —— 因為 `V` 已經被 `View room` 佔掉了。
這種「首字母撞了就換一個鍵並在文字裡標出來」的慣例，
與紮營選單的 `Xorcise` 同一種處理（見下）。

### 紮營選單 14 項（`95-orig-camp-menu.png`）

```
Party    Reorder   Sleep    Identify   Worship   Xorcise   View land
Trade    Drop      Equip    Use        Hunt      Cast      Quit
```

**項數與 `docs/re/33` 解出的 14 項一致** —— 反組譯與畫面互相對上。

而且原版把 Exorcise 拼成 **`Xorcise`**：`E` 被 `Equip` 佔了，
所以連拼字都改掉。手冊（`docs/manual-cht` p.28–31）列的是 `X` Exorcise ——
**手冊與畫面不一致，畫面才是實際的**。

> 這兩張選單是 C1（介面文案）的原文對照表。
> 之前只能從反組譯猜項目，現在有原版逐字的畫面。

---

## 4. 更正：原版的紮營鍵是 `C`

`docs/re/26` §162 寫「紮營的進入鍵 `R` 是本作自己選的 —— **原版用哪個鍵沒查**」。

**手冊早就寫了**（`docs/manual-cht/01-transcript.md` §407）：

> 只要不在城鎮內隨時都可按 `C` 紮營休息，但戰鬥中無效。

而且 `docs/re/54-reference-comparison.md` §13 的 DOSBox 指令列裡
本來就用 `key:c` 進紮營。**兩個地方都有，只有 `docs/re/26` 標成未查。**

> 又一次「動手挖之前先查手上已有的」。這次特別便宜 ——
> 手冊有一節就叫「控制方法」，而且開頭就說「選單指令一律鍵入該選項的
> 第一個英文字母」。**問「原版按什麼鍵」時，手冊是第一個該翻的地方，
> 不是反組譯。**
>
> 順帶：本專案的引擎目前 `R` 紮營、`C` 建立角色，與原版相反。
> 這是可以改的（`C` 紮營才對齊原版），但那是介面決定，另案處理。

---

## 5. 對 worklist

- **`docs/re/80` §3 的衝突結案**：`+0xb9 == 1` 的路徑是活的（DOSBox 實測）
- **新增**：第二場夢的全文（馬利馮的預言）—— 中文化內容
- **新增**：原版主指令選單 15 項、紮營選單 14 項的**逐字原文**（C1 的對照表）
- **更正**：`docs/re/26` §162「原版紮營鍵沒查」→ 是 `C`（手冊 §407）
- **新發現**：原版把 Exorcise 拼成 `Xorcise`，手冊寫的是 Exorcise —— 畫面為準
- 仍未解：寫 `+0xb9 = 1` 的指令在哪（改從寶珠事件往下追，別再從欄位往上掃）
