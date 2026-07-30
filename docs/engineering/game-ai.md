# 遊戲 AI 工程摘要

日期：2026-07-30

## 證據原則

DOS `DEMON.INT` 是規則真值；手冊與攻略只作交叉佐證。下列行為若加入防止
無限迴圈的上限，均明列為 remake 工程保護，不改變正常輸入下的選擇分布。

## 戰鬥 AI

- 怪物依狀態、法力、特殊類型與可用目標決定普通攻擊、法術、吐息或移動；
  完整分派與法術效果見 [`docs/re/23`](../re/23-ai-spell-dispatch.md)。
- 範圍法術以候選中心涵蓋的敵我數量評分；中心點與抽樣畫面見
  [`docs/playtest/14`](../playtest/14-ai-area-spell-centre.md)。
- 近戰接近目標時會避開不可通行格與已佔用格。最短距離及障礙繞行已確認；
  多條同長路線的逐步 tie-break 只有強證據，未冒充逐幀完全相同，見
  [`docs/re/116`](../re/116-monster-obstacle-aware-pathing.md)。
- 幻象消失只在回合推進時擲一次；畫面重繪不得額外消耗 RNG，見
  [`docs/re/115`](../re/115-illusion-turn-vanish.md)。

## 遭遇與部署

- 世界／地城遭遇倒數、地圖等級與商隊基準見
  [`docs/re/24`](../re/24-random-encounters.md)、
  [`docs/playtest/46`](../playtest/46-encounter-countdown-and-map-level.md)。
- 戰場部署保留原版中心散佈；remake 為極端無空位資料加上有界退路，避免
  無限重擲，正常地圖不會走到該分支，見
  [`docs/re/35`](../re/35-battle-deployment.md)。

## 海戰 AI

海盜與海怪使用不同的移動／攻擊決策；砲擊方向、誤擊、邊界逃離、沉船與
船體寫回均來自 IDA 位址證據。規則及 16-byte 單位結構見
[`docs/re/105`](../re/105-ida94-sea-combat.md)，航海碰撞實跑見
[`docs/playtest/45`](../playtest/45-sailing-boundary-and-collision.md)。

## 可重播性

AI 測試使用固定亂數種子；`tools/playthrough.sh` 記錄輸入與狀態軌跡。
主題切換、畫面重繪與說明視窗不得改動 RNG。發行驗收採正常玩家前期垂直
切片加後期高風險抽樣，不宣稱逐房重玩。
