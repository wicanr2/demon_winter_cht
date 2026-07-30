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
| Modern Icon 地城 | **完成，P4 已由使用者核准** | `dungeonTiles` 涵蓋 MAP1–MAP5 實際 59/59 索引，門、閘、樓梯、冰牆、機關與轉角已有實機證據；三主題五類場景 P4 畫板於 2026-07-30 獲核准。見 [`docs/playtest/53`](53-modern-icon-dungeon-atlas-complete.md)、[`docs/playtest/55`](55-p4-three-theme-review-board.md) |
| 復古／現代操作切換 | 完成、待持續視覺回歸 | `F6`；復古為原版紅色直式命令列與相對轉向，現代為兩欄分組命令卡與絕對方向。見 [`docs/ui/04`](../ui/04-control-modes-and-safe-exit.md) |
| `F1` 固定說明 | 完成 | 各主要模式共用說明入口；[`docs/ui/04`](../ui/04-control-modes-and-safe-exit.md) |
| 關閉視窗前自動存檔 | 完成 | 失敗即關閉（fail-closed）：任一步寫檔失敗就留在遊戲；[`docs/ui/04`](../ui/04-control-modes-and-safe-exit.md) |
| 場景跳轉與攻略輔助除錯 | 完成 | `-scene`、`-list-scenes`、技能與戰鬥 fixture；不偷設劇情旗標。README「開發者場景書籤」 |
| 倚天 16×15 粗體中文 | 完成 | [`docs/re/17`](../re/17-font-format.md)；字型由玩家自行提供，未納入版本控制或發行包 |
| 原版音效 | **完成，使用者已核准** | 原版沒有背景配樂；八個單音與死亡旋律已合成。吐息錯誤已修，IDA 證實非戰鬥死亡不播、effect 3/6/7 為未使用音階；見 [`docs/re/117`](../re/117-audio-xrefs-and-breath-correction.md)、[`docs/playtest/54`](54-independent-audio-music-audit.md) |
| remake 遊戲配樂 | **完成，使用者已核准** | 探索、休整、戰鬥、終局四組原創程式合成循環；無 SoundFont／第三方取樣，`F7` 與 `-music-volume` 獨立控制，不冒充原版 BGM。四段 WAV 已供試聽並於 2026-07-30 獲核准 |
| 跨平台正式包工具鏈 | 完成 | Type 2 AppImage、Windows DLL 稽核 ZIP、macOS amd64／arm64 `.app` 均建置、禁入、完整手札逐位元組比對、smoke 與 artifact 上傳全綠；macOS 另通過 Mach-O 架構與系統函式庫白名單。[run 30523217956](https://github.com/wicanr2/demon_winter_cht/actions/runs/30523217956)、[`docs/playtest/57`](57-remake-music-and-release-packages.md)。正式發布另有手動布林閘門、四產物數量與 SHA-256 驗證 |
| 宣傳影片 | 完成 | [`短版 MP4`](../promo/demon-winter-cht-promo.mp4)；為 remake 實機畫面剪輯，不含原版資料檔或字型檔 |

## 二、引擎、資料與研究

| 要求 | 狀態 | 證據／邊界 |
|---|---|---|
| 玩家文字不可硬編在 Go | 完成 | 原版資料字串 500/500；介面與規則原因共 839 條 JSON，玩家中文硬編 0。見 [`docs/playtest/47`](47-rule-reason-json-separation.md) |
| 引擎與資料分離 | 完成目前範圍 | `assets/lang/zh-Hant/ui.json` 保存介面文字、順序、分組與格式；Go 只持有 key、參數、熱鍵與 action。缺 key 失敗即關閉，`dwstrings uicheck` 是發行閘門 |
| 少數逆向缺口整理成 Markdown | 持續以證據強度管理 | `docs/re/` 現有 118 篇編號研究；怪物繞障見 [`docs/re/116`](../re/116-monster-obstacle-aware-pathing.md)，音效 XREF 見 [`docs/re/117`](../re/117-audio-xrefs-and-breath-correction.md)，未使用水紋 frame 見 [`docs/re/118`](../re/118-unused-terrain-frame-101.md)。同長路線 tie-break 尚未逐步動態對拍，標為強證據而非完全相同 |
| 反組譯＋remake 通用範本／skill | 完成 | `reverse-engineer-retro-game-remake` 同步於 `~/.codex/skills/` 與 `~/my_skill/`；repo 內可重建來源由 README 索引 |
| PC-98 Golden Box 通用 knowledge base | 完成 | `research-pc98-golden-box-ui` 以關鍵字觸發；內容在 `~/my_skill/knowledge-base/retro-cht/`，`~/.codex/skills` 使用連結，不要求每次啟動全讀 |
| 遊戲引擎抽離評估 | 研究完成、抽離延後 | [`docs/design/engine-extraction-study.md`](../design/engine-extraction-study.md)；在第二款真實作品接入前，不把單一遊戲假定為通用 SSI 引擎 |
| 長期記憶與主機衛生 | 完成 | `AGENTS.md` 連回 `CONTEXT.md` §7；建置、測試、抓圖與 IDA 僅在 Docker，結束檢查容器與專案映像 |

## 三、目前真正未結案

配樂與 P4 已由使用者於 2026-07-30 核准。現在只剩：

1. 以 `v0.1.0` 正式版本號重建並建立 GitHub 乾淨版 Release；
2. 將相同四平台公開產物注入合法原版資料與本機字型，建立只留在
   `dist-all/` 的完整版；
3. 逐包驗證 GitHub 禁入、本地素材完整度、SHA-256 與啟動路徑。

公開版與本地完整版是兩條不可混用的發行路徑；後者不得上傳 GitHub、Actions
artifact 或納入版控。
