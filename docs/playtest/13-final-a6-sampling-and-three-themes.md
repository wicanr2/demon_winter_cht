# 13 — 最後 A6 抽樣：正常遭遇、密門、海戰與三主題

日期：2026-07-29。這不是重跑整座大型地城，而是依使用者決定的 A6 範圍，
抽取最容易在串接／呈現層退步、且彼此機制不同的路徑。

## 1. 正常玩家遭遇（固定 seed 11）

來源：既有 A6 已裝備存檔的唯讀副本；腳本
`tools/playthrough/a6-grind-smoke.txt`；參數 `-autofight -seed=11 -video=modern`。

- 世界圖 34 的 (29,45) 進入蜘蛛／巨蜘蛛／巴格姆遭遇；
- 第 7 回合怪物全滅；
- 每人 +13 EXP，取得 18 金幣；
- 正常回到世界、進出海濱鎮、遇商隊，再走到 (35,45)；
- 時間進到 7 時，存檔成功；
- 過程沒有使用 `-battle-win`、`-gold`、`-sp` 或劇情解鎖旗標。

這一段同時抽到戰鬥重建、傷害寫回、金幣、商隊 modal、城鎮進出、時間與存檔。

## 2. 密門不是「理論上可走」

新增 `tools/playthrough/a6-secret-door-smoke.txt`，由具名書籤
`-scene cage-secret` 只定位到地圖 1 (13,26)，不改地圖／道具／劇情旗標：

```text
(13,26) → Down → Left×4 → Up → (9,26)
```

腳本的兩個 `at` assertion 均通過，證明鐵籠房間的西牆密門可由一般移動穿過。
Modern EGA 只換顏色、不改 tile index，因此既沒有把門變阻擋，也沒有新增可辨識門框。
C10 結案。

## 3. F8 與海戰視覺

- 以 `-video=ega` 啟動同一座標，送 `Return,F8,F8`；
- 畫面訊息為「顯示主題：Modern EGA」，地圖內容與直接
  `-video=modern` 相同，只多了切換訊息；
- `-sea-battle -video=modern -seed=11` 成功顯示 75/75 船體、6 移動點、
  船與海怪 sprite、雙線戰場框與繁中指令；
- theme cycle 單元測試固定 EGA → CGA → Modern EGA → EGA；
- atlas 轉換測試固定 frame 數、尺寸、索引及來源不被就地改寫。

## 4. 發行前閘門

- `go test ./...`：全綠（Ebiten 套件在 Xvfb 下執行）；
- `dwstrings check`：500/500；
- `dwstrings uicheck`：716/716，41 個動態 key，通過；
- `git diff --check`：通過；
- 所有抽樣都使用 `/tmp` 或唯讀副本，沒有改寫原版資料。

結論：A6 的最後抽樣通過，可進最後打包階段。
