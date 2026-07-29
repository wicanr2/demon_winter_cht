# 17 — Modern EGA M1-B 原生 32×28 bounded runtime 試片

日期：2026-07-29。

## 1. 範圍與證據邊界

設計試片位於 `artwork/modern-ega/m1/`。它不讀取或內嵌原版 bitmap，而是由
`generate.go` 的像素 primitive 直接生成真正的 32×28、不透明 PNG。

本輪只把具有 code／data 語意證據的七個 index 放入實機：

```text
0x01 forest
0x14 deep water A
0x62 deep water B
0x23 plain
0x2e town
0x1e/0x1f party north animation
```

`0x17` 海岸與 `0x63` 山峰仍是視覺候選；海岸缺少互補方向、山峰尚未完成語意
證明，所以本輪明確不覆蓋它們。

## 2. 可重跑方法

```bash
tools/go.sh run artwork/modern-ega/m1/generate.go

tools/go.sh run ./tools/modernpreview \
  -data workplace/orig/demwin/DEM_DATA \
  -out workplace/dump/modern-m1-b-runtime \
  -terrain-overlays artwork/modern-ega/m1/tiles \
  -overlay-indices 01,14,1e,1f,23,2e,62

tools/screenshot.sh /tmp/modern-m1-b-party-a.png KEYS=Return,Up \
  -video=modern \
  -modern-theme-dir=workplace/dump/modern-m1-b-runtime \
  -map=34 -x=28 -y=50 -seed=11

tools/screenshot.sh /tmp/modern-m1-b-party-b.png KEYS=Return,Up,Up \
  -video=modern \
  -modern-theme-dir=workplace/dump/modern-m1-b-runtime \
  -map=34 -x=28 -y=50 -seed=11
```

`modernpreview` 要求顯式列出 `-overlay-indices`，且每個 index 必須同時有 DEMON
與 WINTER 版本；錯尺寸、半透明、重複、缺檔或超出 102 格都會立即失敗。這避免
未批准的試片因為放在同目錄就悄悄進入測試 atlas。

## 3. 實機觀察

![M1-B 七索引實機試片](../design/img/modern-ega-m1-b-runtime-trial.png)

- 平原不再產生 B 直接縮圖的規律高亮噪音，四邊沒有 tile 色縫。
- 兩個深水 frame 的底色與 quiet border 相同，交錯排列沒有硬縫。
- 北向隊伍改為 18×24 大形，兩步後仍以同一中心與腳底 anchor 顯示；只換腿。
- 城鎮的門洞在原生尺寸仍是第一讀點。
- 未替換的森林、海岸與其他 terrain 仍保留調色預覽，與新試片形成明顯風格跳接。

最後一點是正確的 bounded-trial 證據：七個 index 已證明管線與原生像素方法可行，
但它**不能被稱為完整 Modern EGA 主題或可發布 atlas**。只有把同一方法擴到完整、
已證語意的索引族後，風格才會連續。

兩張移動截圖 SHA-256：

```text
9f0b3effa6d0240bddc6f086f872d12a310eddea8282b934970d7eee44b19a32  party-a
43e7a2a0e5224512578024485fbfeab1c446cac11fbb409c44bda69307bc89eb  party-b
```

## 4. 裁決與清理

M1-B 的「直接在 32×28 手工控制大形、低頻紋理、固定 anchor」方法通過這次
bounded runtime 驗證；美術風格仍待使用者過目，完整 atlas 仍未完成。

臨時完整 theme `workplace/dump/modern-m1-b-runtime` 含原版衍生像素，驗後刪除。
版控只保留可獨立生成的新試片、實機證據與組裝工具。
