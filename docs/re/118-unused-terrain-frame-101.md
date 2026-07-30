# 118 — DEMON／WINTER 索引 101 是未被消費的額外水紋 frame

日期：2026-07-30

## 問題

`DEMON.SHP/.SHE` 與 `WINTER.SHP/.SHE` 各有 102 格，但 `FILES.DAT` 通行表
只有 tile 0–100。最後的 index 101 是否仍由程式動態使用，長期標成未知。

## 證據

1. 以 `tools/cgaatlas`、`tools/tileatlas -gamepal` 分別畫出 CGA／EGA、
   DEMON／WINTER 四份 atlas。index 101 四者都是同構的密集水紋；它不是
   解碼尾端垃圾，也不是隊伍或船 glyph。
2. `mapwindow -inventory -min-map 1 -max-map 66` 掃過五座地城與 `SUM.MAP`
   全部區段，最大實際索引是 `0x64`（100），沒有 `0x65`（101）。
3. 已解的動態寫入只有：
   - 隊伍 glyph：24–25、27–28、30–31、33–34；
   - 船 glyph：63–66；
   - 海面隨機替換：`0x14 → 0x62`。
   三條來源分別由 `docs/re/97` 與 `docs/re/03` 的指令級證據固定，均不會
   產生 101。
4. IDA 9.4 全檔文字反組譯沒有「把立即值 `0x65` 當 tile 寫入」的候選；
   出現的 `65h` 都是位址位移、資料 byte 或其他較大立即值的一部分。

## 裁決

**強證據：index 101 是隨素材表一起出貨、但這版 DOS runtime 沒有消費的額外
水紋 frame。** 由於沒有找到直接的設計註解或符號名稱，不能進一步證明它原先
預定是第三種海面動畫，因此不替它發明用途。

remake 仍完整解碼並保留 102 格原版 atlas，但規則、Modern Icon inventory 與
完成度只涵蓋真正可達的 0–100。這不是素材缺漏，也不要求為 Modern Icon 虛構
index 101。
