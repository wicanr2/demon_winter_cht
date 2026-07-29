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
- `dwstrings uicheck`：當時為 716/716，41 個動態 key，通過；
- `git diff --check`：通過；
- 所有抽樣都使用 `/tmp` 或唯讀副本，沒有改寫原版資料。

結論：A6 的最後抽樣通過，可進最後打包階段。

## 5. 2026-07-29 gameplay closure 後重跑

營地／戰鬥魔法物品、召喚選點與命中修正接回正式 consumer 後再抽一次：

- Xvfb 下 `go test ./...` 全綠；
- `dwstrings check` 500/500；
- `dwstrings uicheck` 748/748、41 個動態 key，通過；另列出的 16 條是只供
  開發者 `-scene` 清單使用的描述；
- 固定 seed 11 的正常遭遇仍在地圖 34 `(29,45)` 進場，第 7 回合怪物全滅，
  每人 +13 EXP、18 金，Dodge 紀錄可見新命中修正路徑；
- `-scene cage-secret` 由 `(13,26)` 走到 `(9,26)`，密門穿越 assertion 通過；
- Modern EGA 海戰顯示 75/75 船體、6 移動點、海怪 20/20，框、sprite 與繁中
  指令無溢出；
- 由 `-video=cga` 啟動按 `F8`，畫面明示「顯示主題：Modern EGA」，主題切換
  不改座標與隊伍狀態；
- 全部抽樣輸出在 `/tmp/dw-a6-closure`，一擊 Docker 均由 `--rm` 清除。
- 建立 `demonwinter-zh-Hant-2026.07.29-closure-linux-amd64.tar.gz`，SHA-256
  `64bcf7613478171cad0ed5eb23814c1ccdfa3528e7f20606bcabcf438ba0413a`；
  校驗通過，解壓內容不含原版資料／倚天字型，並在 Xvfb 下由解壓後 binary
  實跑 `-list-scenes` 成功。

新增玩法的精確邊界另由單元測試覆蓋：四種充能、營地道具不扣 SP、召喚合法格、
背後／附魔／劍擊／狀態計數命中修正。幻術每回合消失機率仍是明列未解項，
不影響上述已接路徑的驗收。
