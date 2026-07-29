# 航海邊界與撞岸收尾（2026-07-30）

## 原版證據

`DEMON.INT` 的 `222f:0f8e–106d` 先掃十個船槽，再以兩套世界格網換算
比對相鄰子地圖的船：

- 地圖編號差 10：X 相差 56、Y 相同；
- 地圖編號差 1：Y 相差 56、X 相同。

`222f:0eee–0f04` 的撞岸分支呼叫 `2000:03f2`；
該 wrapper 是 `FUN_1d9f_2a02`，固定播放 effect 1（C3）。
分支沒有改船體；船體傷害只在海戰單位結算後寫回船記錄。

完整指令摘錄與歷史訂正見 [`docs/re/31`](../re/31-sailing.md)。

## 引擎閉合

- `game.ReachableBoatAt` 實作同圖與兩組跨圖換算。
- 世界移動的 `Boardable` 使用上述判定，讓海面上的相鄰地圖船格可走。
- `crossSubMapEdge` 換算到新地圖座標後，以精確船槽完成 `Boat` 與
  `Party.sailing` 狀態。
- 航行狀態下 `MoveBlocked` 播放 `EffectC3`；徒步撞牆不走這條航海音效。

## 驗證

- 四個跨圖方向與兩組反例的表格測試。
- `ReachableBoatAt → world.CrossEdge → BoatAt` 組合測試，證明換圖前後
  命中同一艘船。
- 撞岸／徒步音效映射測試。
- `go test ./internal/game ./cmd/demonwinter`：通過。

