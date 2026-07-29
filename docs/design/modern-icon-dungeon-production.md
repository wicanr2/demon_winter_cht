# Modern Icon 地城逐索引量產規格

日期：2026-07-30
狀態：客觀資料與工具閘門完成；視覺方向待使用者審查

## 1. 為什麼不是「畫十二張就完成」

方向稿展示地板、牆、轉角、走廊、門、鐵閘、密牆、柱、樓梯、儀式、冰與
熔岩十二種材質語言；它是審稿用的概念集合，不是原版 atlas 的逐格對照。

專案解析器實際掃過 MAP1–MAP5 的 5×64×64 格後，得到 **59 個使用中索引**。
同一個規則類別也可能有完全不同的外觀：

| 客觀行為 | 索引數 | 美術約束 |
|---|---:|---|
| `blocked` | 17 | 只代表不能走，不等於全是磚牆；含機關、障礙與特殊景物 |
| `exit` | 7 | 都會換圖，但外觀可能是樓梯、洞口、門或地圖邊緣 |
| `special` | 10 | 原始通行值為 `0xfe` 等特殊群，須逐消費端確認，不靠圖猜 |
| `submap-floor` | 2 | `0x14/0x62` 在子地圖是可走例外，不能套世界海面語意 |
| `terrain-0` | 12 | 原版 atlas 的森林套組；是否沿用已核准世界構圖須逐場景審查 |
| `terrain-1` | 5 | 平原／岸線類，需保留原始拓樸 |
| `terrain-4` | 3 | 地城 A 地板：`0x00/0x13/0x53` |
| `terrain-5/6/7` | 各 1 | 凍土、地城 B、沙地 |

因此量產不能以「所有 `blocked` 共用一張牆」或「所有 `exit` 共用一張樓梯」
填表；那會通過檔案存在檢查，卻改壞導航、密門與場景辨識。

## 2. 資料來源

生成命令：

```bash
go run ./tools/mapwindow \
  -data workplace/orig/demwin/DEM_DATA \
  -inventory -min-map 1 -max-map 5 \
  -inventory-json artwork/modern-icon/m1/dungeon-inventory.json
```

輸出 [`dungeon-inventory.json`](../../artwork/modern-icon/m1/dungeon-inventory.json)
每筆包含：

- `index`：theme manifest 使用的十六進位 key；
- `count` 與 `byMap`：總格數及地圖 1–5 各自格數；
- `first`：第一個可重播的實機座標；
- `passabilityRaw`：`FILES.DAT +0x040` 的原始值；
- `behavior`：只由已解規則導出的客觀分類。

JSON 不收玩家資料、原版圖像或字型，可以進版本控制。原始 MAP／FILES 檔仍由
玩家自備且唯讀，沒有被複製到 repo。

## 3. 分批方式

審稿核准後依風險排序，不依索引數字順序：

1. **D1 基底與邊界**：`terrain-4`、`terrain-6`、`submap-floor`，先證明
   64×56 邊界、黑暗／照明及連續鋪設沒有接縫。
2. **D2 牆與走廊**：先從高頻 `blocked` 做逐索引 contact sheet；保留壓牆、
   開門前後及密牆的同構關係。
3. **D3 出口與互動物**：七種 `exit` 各自重畫；正常門、密門、樓梯與洞口
   必須在互動前後可辨識，但不能提前洩漏密門。
4. **D4 稀有機關**：`special`、墓碑、儀式、冰、熔岩及單格物件；用
   `first` 座標逐一抓 runtime 證據。
5. **D5 戶外／地城交界**：森林、平原、岸線、凍土與沙地；可重用已核准
   Modern Icon 構圖的只有視覺語意確實相同者，仍須明列在 `dungeonTiles`，
   不讓 loader 隱式跨 namespace 借圖。

## 4. 每批驗收

每個索引必須同時具備：

- `dungeonTiles` 的明確 JSON entry；
- 64×56 不透明 PNG，非原版素材放大或方向稿裁切；
- 依 `byMap` 選出的代表場景及至少一個稀有 `first` 座標；
- contact sheet 上的 index、行為與檔名一致；
- EGA／CGA／Modern Icon 同狀態截圖，規則資料完全相同；
- 密門、陷阱、照明、碰撞、出口、劇情改圖及 F8 不改狀態測試。

完整度命令：

```bash
go run ./tools/mapwindow \
  -data workplace/orig/demwin/DEM_DATA \
  -inventory -min-map 1 -max-map 5 \
  -theme artwork/modern-icon/m1/trial/theme.json
```

目前 `dungeonTiles` 為空，所以必須列出 59 個 `theme dungeon missing` 並失敗。
只有這份清單變成 `none`，且上述畫面都經人工審查，P3-D 才能改成完成。

## 5. 本輪不越過的決策

[`modern-icon-dungeon-direction-v1.png`](img/modern-icon-dungeon-direction-v1.png)
尚待使用者審查。本輪只完成資料、工具、批次與 fail-closed 閘門；不把方向稿
切成 runtime tile，也不自行宣告其色彩、材質或構圖已核准。
