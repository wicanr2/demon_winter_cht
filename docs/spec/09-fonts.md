# 字型與中文化

> 狀態：**READY**（原版字型解碼已驗證；中文化的畫布與排版是本專案的新設計，
> 不是還原原版，另立標記）
> 證據來源：`docs/re/17-font-format.md`（繪字函式 + 肉眼驗證）、
> `docs/formats/graphics.md` §6
> 最後複核：2026-07-25

---

## 行為

原版有兩套字型，依顯示卡自動擇一（**不是玩家可選的**）：

| 檔案 | 用途 |
|---|---|
| `ASC.FNT` | CGA 版一般字型 |
| `ASC.FNE` | EGA 版一般字型 |
| `GOT.FNE` | EGA 版花體（blackletter），用於 UI 標題 |

判別旗標 `DAT_31f0_19fb`：開機時 `INT 10h` 回傳 `0x10` → 設 1（EGA），
否則 0（CGA）。`GOT.FNT`（CGA 版花體）不存在，所以 CGA 版沒有花體。

---

## 資料

| 項目 | CGA `.FNT` | EGA `.FNE` |
|---|---|---|
| 字元尺寸 | **8 × 8** | **16 × 14** |
| 色深／佈局 | 2bpp，**同一列 2 bytes = bit0/bit1 兩個平面** | 1bpp（前景色由呼叫端指定）|
| 每字 bytes | 16（8 列 × 2）| 28（14 列 × 2）|
| bit 順序 | MSB-first | MSB-first |
| 字表起點 | ASCII `0x20`（空白）| ASCII `0x20` |
| 檔頭 | 1 byte（觀察值 `0x00`，用途未查）| 無 |
| 涵蓋字元 | 192 筆（`(3073−1)/16`）| 96 筆（`2688/28`，恰為 `0x20`–`0x7F`）|

硬體常數交叉驗證（`FUN_1d9f_0008`）：
CGA `寬 8, 高 8, 總長 0xc00(3072)`；EGA `寬 0x10(16), 高 0xe(14), 總長 0xa80(2688)`。

> **CGA 只有 `0x20`–`0x7F`（96 字）經可讀文字驗證。**
> `0x80`–`0xDF` 解碼出規律圖案但未經畫面驗證 —— 那段**不要當成已驗證的字形**。
> 曾因為看 atlas 裡那段的雜訊而差點誤判整個解碼器是錯的；
> 實際文字渲染（`DEMON'S WINTER`、`Alternate Character Set`）完全乾淨。

---

## 演算法

### CGA 字型解碼

```
glyph_offset = 1 + (ch − 0x20) × 16          ← 檔頭 1 byte
for row in 0..7:
    b0 = data[glyph_offset + row*2]          ← bit 平面 0
    b1 = data[glyph_offset + row*2 + 1]      ← bit 平面 1
    for col in 0..7:
        pixel = (bit(b1, 7-col) << 1) | bit(b0, 7-col)     ← 2bpp 色號
```

**不是 chunky（每 byte 4 個 2bpp 像素），也不是逐 plane 整塊。**
是「同一列的兩個 byte 各提供一個 bit 平面」。

### EGA 字型解碼

```
glyph_offset = (ch − 0x20) × 28              ← 無檔頭
for row in 0..13:
    w = (data[glyph_offset + row*2] << 8) | data[glyph_offset + row*2 + 1]
    for col in 0..15:
        pixel_on = bit(w, 15-col)            ← 1bpp，顏色由呼叫端給
```

---

## 中文化設計（本專案新增，不是還原原版）

> 以下**不是原版行為**，是為了中文顯示所做的設計決定。
> 與上面的原版格式規格分開看。

### 畫布

拉到 **640×400**。理由：中文需要 16×16 點陣才可讀；
英文 8×8 放大兩倍後正好與中文同高，兩者可以混排對齊。

原版 EGA 目標畫面是 640×350，CGA 是 320×200。

### 字型來源

依全域規則，老遊戲中文化的字形來源用**倚天點陣字**，不是 TTF rasterize。
相關做法見 `~/.claude/knowledge-base/retro-cht/eten-bitmap-font/`。

### 待決定

- 花體（`GOT.FNE`）的中文對應：中文沒有 blackletter 概念，
  UI 標題要用別的方式做視覺區隔（字重？外框？）——未定案
- 原版 CGA 的 `0x80`–`0xDF` 那段若確認是圖形字元，中文版要不要保留 —— 未定案

---

## 邊界條件

| 情況 | 行為 |
|---|---|
| CGA/EGA 選擇 | 硬體層級自動偵測，**不是玩家選單**。重製時應提供手動切換，但預設行為要照原版 |
| CGA 無花體 | `GOT.FNT` 不存在，CGA 版一律用 `ASC.FNT` |
| 字元 < `0x20` | 字表不涵蓋，行為未定義 |

---

## 驗收

1. **字形肉眼比對**：把 `ASC.FNT`／`ASC.FNE`／`GOT.FNE` 全部 dump 成 atlas PNG，
   與 DOSBox 原版截圖的實際文字比對。
   **只比對 `0x20`–`0x7F` 這 96 字**，其餘範圍未驗證。
2. **已知字串重現**：渲染 `DEMON'S WINTER`、`Alternate Character Set`
   兩個已知字串，必須與 `workplace/dosbox/shots/` 的截圖吻合。
3. **花體確認**：`GOT.FNE` 解出來必須是 blackletter 風格，
   與 `smoke-01.png`／`03-ega-ingame.png` 的 UI 文字一致。

---

## 未解 / 風險

| 項目 | 影響 |
|---|---|
| CGA `.FNT` 檔頭那 1 byte 的用途 | 觀察值 `0x00`，未查出。不擋解碼（跳過即可）|
| CGA `0x80`–`0xDF`（96 字）| 解碼出規律圖案但未經畫面驗證。**不要當成已驗證** |
| `ASC` ↔ `GOT` 的切換串接 | 資源表已找到（`31f0:19fd` 起 5 筆遠指標的檔名表），但「Alternate Character Set」選單怎麼串到這張表未追完 |
| 中文花體的替代方案 | 設計問題，非逆向問題。未定案 |
