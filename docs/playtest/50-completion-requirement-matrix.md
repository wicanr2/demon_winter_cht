# 歷次需求與完成證據矩陣

日期：2026-07-30

這張表把長期交接中反覆提出的產品要求，對到目前可重播的程式、文件與畫面
證據。它不是用勾選數量代替品質；「完成」代表已有相稱的自動或實機證據，
「待審」代表必須由使用者看方向後才能繼續，不會冒充完成。

## 一、玩家產品

| 要求 | 狀態 | 證據／邊界 |
|---|---|---|
| 經典規則與主線可完成 | 完成並抽樣複核 | [`docs/re/64`](../re/64-mainline-playable.md)、[`docs/playtest/13`](13-final-a6-sampling-and-three-themes.md)、[`docs/playtest/48`](48-encounter-and-movement-a6-recheck.md)；採前期垂直切片加後期高風險抽樣，不宣稱逐房重玩 |
| 海戰系統 | 完成 | [`docs/re/105`](../re/105-ida94-sea-combat.md)、[`docs/playtest/45`](45-sailing-boundary-and-collision.md)；保留原版相對轉舵與耗點規則 |
| EGA／CGA 還原與 `F8` 切換 | 完成 | README 同狀態比較、[`docs/playtest/13`](13-final-a6-sampling-and-three-themes.md)、[`docs/playtest/21`](21-three-theme-same-state-comparison.md) |
| Modern Icon 世界、角色、怪物與船 | 客觀覆蓋完成 | 世界正常／冬季差集為零；怪物 224/224、隊員 24/24、海戰 runtime 24/24，見 [`docs/playtest/39–42`](39-modern-icon-monster-direction-complete.md) |
| Modern Icon 地城 | **待使用者審方向** | `dungeonTiles` JSON namespace 與安全底稿已完成；[`方向稿`](../design/img/modern-icon-dungeon-direction-v1.png) 尚待審，審後才逐索引量產。見 [`docs/playtest/49`](49-modern-icon-dungeon-namespace.md) |
| 復古／現代操作切換 | 完成、待持續視覺回歸 | `F6`；復古為原版紅色直式命令列與相對轉向，現代為兩欄分組命令卡與絕對方向。見 [`docs/ui/04`](../ui/04-control-modes-and-safe-exit.md) |
| `F1` 固定說明 | 完成 | 各主要模式共用說明入口；[`docs/ui/04`](../ui/04-control-modes-and-safe-exit.md) |
| 關閉視窗前自動存檔 | 完成 | 失敗即關閉（fail-closed）：任一步寫檔失敗就留在遊戲；[`docs/ui/04`](../ui/04-control-modes-and-safe-exit.md) |
| 場景跳轉與攻略輔助除錯 | 完成 | `-scene`、`-list-scenes`、技能與戰鬥 fixture；不偷設劇情旗標。README「開發者場景書籤」 |
| 倚天 16×15 粗體中文 | 完成 | [`docs/re/17`](../re/17-font-format.md)；字型由玩家自行提供，未納入版本控制或發行包 |
| 音效與音樂 | 完成已證實範圍 | 原版沒有背景配樂；PC speaker 效果與死亡旋律觸發見 [`docs/playtest/43`](43-audio-trigger-closure.md)，不虛構新配樂 |
| 宣傳影片 | 完成 | [`短版 MP4`](../promo/demon-winter-cht-promo.mp4)；為 remake 實機畫面剪輯，不含原版資料檔或字型檔 |

## 二、引擎、資料與研究

| 要求 | 狀態 | 證據／邊界 |
|---|---|---|
| 玩家文字不可硬編在 Go | 完成 | 原版資料字串 500/500；介面與規則原因共 836 條 JSON，玩家中文硬編 0。見 [`docs/playtest/47`](47-rule-reason-json-separation.md) |
| 引擎與資料分離 | 完成目前範圍 | `assets/lang/zh-Hant/ui.json` 保存介面文字、順序、分組與格式；Go 只持有 key、參數、熱鍵與 action。缺 key 失敗即關閉，`dwstrings uicheck` 是發行閘門 |
| 少數逆向缺口整理成 Markdown | 持續以證據強度管理 | `docs/re/` 現有 116 篇；怪物繞障見 [`docs/re/116`](../re/116-monster-obstacle-aware-pathing.md)。同長路線 tie-break 尚未逐步動態對拍，標為強證據而非完全相同 |
| 反組譯＋remake 通用範本／skill | 完成 | `reverse-engineer-retro-game-remake` 同步於 `~/.codex/skills/` 與 `~/my_skill/`；repo 內可重建來源由 README 索引 |
| PC-98 Golden Box 通用 knowledge base | 完成 | `research-pc98-golden-box-ui` 以關鍵字觸發；內容在 `~/my_skill/knowledge-base/retro-cht/`，`~/.codex/skills` 使用連結，不要求每次啟動全讀 |
| 遊戲引擎抽離評估 | 研究完成、抽離延後 | [`docs/design/engine-extraction-study.md`](../design/engine-extraction-study.md)；在第二款真實作品接入前，不把單一遊戲假定為通用 SSI 引擎 |
| 長期記憶與主機衛生 | 完成 | `AGENTS.md` 連回 `CONTEXT.md` §7；建置、測試、抓圖與 IDA 僅在 Docker，結束檢查容器與專案映像 |

## 三、目前真正未結案

1. 使用者審閱 Modern Icon 地城方向稿。
2. 依核准方向製作地城逐索引素材、聯絡表與代表 runtime 截圖。
3. 執行 P4 同狀態 EGA／CGA／Modern Icon 最終視覺驗收。

上述三項會改變美術方向，不能由代理在未審稿時自行宣告通過。其餘程式變更仍
必須通過完整 Go、500/500、`uicheck`、A6 抽樣、禁入掃描、解壓後執行檔與
Docker 清理閘門；本文件不取代那些測試記錄。
