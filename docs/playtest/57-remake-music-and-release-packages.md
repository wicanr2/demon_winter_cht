# remake 場景配樂與正式封裝證據

日期：2026-07-30
狀態：配樂程式、四段 WAV、Linux AppImage、Windows ZIP、macOS 原生 CI、
A6 正常玩家第一段、使用者聽感／P4 核准與 `v0.1.0` 正式發布均完成

## 配樂邊界

原版 DOS 無背景配樂。`internal/audio/music` 新增的是明確標示的 remake 原創
配樂，以程式合成三角波，不使用 SoundFont、第三方取樣或原版素材：

| 場景 | 用途 |
|---|---|
| Exploration | 世界與地城探索 |
| Sanctuary | 標題、城鎮、營地、商隊 |
| Battle | 一般戰鬥與海戰 |
| Finale | 死亡、魔王與結局 |

`F7` 獨立切換配樂；`-music-volume 0–1` 控制音量。原版 PC speaker 音效仍走
既有 Sound 旗標，兩者可同時存在。單元測試確認四組 PCM 非空、16-bit 對齊，
零音量與 Silent 不產生波形；完整 Go 與 `uicheck` 839/839、硬編中文 0 通過。
同一套合成器另輸出四段 44.1 kHz、16-bit、單聲道 WAV 至 `docs/audio/`，供
GitHub 直接試聽；它們不是另一套人工後製音檔。

## Help 與包內文件

`F1` 固定開啟完整繁中手札；第一章新增 remake 操作、F6／F7／F8／F10 與探索
命令速查，後續原有故事、規則、數值及查詢訊息全部保留。AppImage、Windows ZIP
與 macOS `.app` 的 staging 都包含 `README.md`、`開始遊戲.txt` 與同一份手札。

## Linux AppImage

- 以官方 AppDir → appimagetool Type 2 流程建立，不是更名的 tar。
- appimagetool 與 Type 2 runtime 均釘選 SHA-256。
- 私有 `usr/lib` 收錄非 glibc 的直接動態相依，`AppRun` 指向包內翻譯、手札與
  Modern Icon。
- RC1 已成功產生並可 `--appimage-extract`；包內執行檔、README 與手札存在。

## Windows ZIP

- `CGO_ENABLED=0` 的 amd64 PE 經 `objdump -p` 稽核 import。
- 現況只有 Windows 系統 DLL，需隨包附帶的第三方 DLL 為 0；ZIP 內附實際
  import 清單與「第三方 DLL：0」說明，避免含糊宣稱。
- ZIP 包含引擎、翻譯、手札、README、開始說明與 Modern Icon。

## macOS

`.github/workflows/cross-platform-release.yml` 已在原生 Intel／Apple Silicon runner
建立各自 `.app` ZIP，包含 `Info.plist`、Resources、ad-hoc codesign 與
`-list-scenes` smoke。包含四段 WAV 預覽的最終候選版，兩架構與
AppImage／Windows job 全綠：
[run 30523217956](https://github.com/wicanr2/demon_winter_cht/actions/runs/30523217956)。
該輪另以 `lipo -archs` 驗證 Intel 為 `x86_64`、Apple Silicon 為
`arm64`，並以 `otool -L` 拒絕 Homebrew、`@rpath` 或建置目錄相依；實際
系統函式庫清單隨 `.app` 附上。

## Help 與既有訊息保留

F1 開啟的手札只在最前方新增 remake 操作章；原有繁中說明、規則提示、數值
與查詢訊息仍完整接在後方。發行包同時放入 `README.md`、`開始遊戲.txt` 與
`assets/manual/zh-Hant/manual.txt`，因此遊戲內、離線套件與 GitHub 三處都能
查到操作方式，而不是用新版速查取代舊訊息。

## 最後 A6 抽樣

`a6-leg1.txt` 以 `-newgame -autofight -seed 11` 重跑正常玩家第一段；沒有
`-scene` 或劇情旗標。Trace 共 19 個狀態節點：五人建角、首次存檔、抵達海濱
鎮、購買武器／護甲、離城與再次存檔均成功；產出四張實機截圖、`PARTY.DAT`、
五張 `nSS.DAT` 與 `ITEMLOCB.DAT`。新增配樂同步及 Help 沒有改動正常流程。

## 硬閘門結果

使用者已於 2026-07-30 核准四段 remake 配樂與 P4 最終畫面。正式 tag
`v0.1.0` 隨後重建四平台乾淨包，版本、數量、SHA-256、禁入、完整手札、
Windows DLL、macOS 架構／系統相依與解壓 smoke 全部通過：
[run 30523970829](https://github.com/wicanr2/demon_winter_cht/actions/runs/30523970829)；
[GitHub Release](https://github.com/wicanr2/demon_winter_cht/releases/tag/v0.1.0)。

## 正式發布路徑補齊

`cross-platform-release.yml` 不再只停在短期 artifact。更新後的候選 workflow
[run 30522874027](https://github.com/wicanr2/demon_winter_cht/actions/runs/30522874027)
已由 GitHub 實際解析並全綠；一般 push 中發布 job 如設計般跳過。手動執行時可輸入正式
`X.Y.Z` 版本並勾選 `publish_release`；只有 Linux／Windows 與兩個 macOS job
全部成功後，發布 job 才會執行。它會先斷言恰有一份 AppImage、一份 Windows
ZIP、Intel／Apple Silicon 各一份 macOS ZIP 與四份 SHA-256，逐份驗證校驗碼
及版本名，再建立 `vX.Y.Z` GitHub Release。一般 push 不會發布。

2026-07-30 RC3 本機封裝抽樣另確認：

- AppImage 與 Windows ZIP 的 SHA-256 均通過；
- AppImage 實際以 `-list-scenes` 啟動成功；
- 兩包各含四段配樂 WAV、466 張 Modern Icon PNG、README、開始說明與發行說明；
- 原版資料／原版圖檔／倚天字型禁入掃描為零；
- Windows 第三方 DLL 為 0，只有包內列出的 Windows 系統 DLL；
- CI 新增逐位元組 `cmp`，確保包內 F1 手札與 repo 的完整手札相同，舊查詢
  訊息不會在 staging 過程遺失。

## 本地完整版

公開 Release 維持原版素材與倚天字型禁入。`tools/package-full-local.sh`
只在 Docker 內將已下載且通過 SHA-256 的四份公開包，注入唯讀來源
`workplace/orig/demwin/DEM_DATA` 與 `workplace/eten`，輸出到 Git 忽略的
`dist-all/`。四包均逐檔比對 94 個原版資料檔與兩個字型的 SHA-256；兩個
macOS `.app` 與公開版逐檔相同，只有包外增加本地資料與啟動器。Linux
完整版另以固定 armory 場景實際啟動並讀取資料／字型。

本地完整版 SHA-256：

| 平台 | SHA-256 |
|---|---|
| Linux x86_64 | `931e01d9208b61554dfdd8915920fd28b519fbaa5f66abcea7c123b36eb10bc4` |
| Windows x86_64 | `613bb468d865249fc242a544c925c9ea8f10a62b31ec8925af66f776df0e0583` |
| macOS Intel | `bb5c88a699285ed36ae6fbc1d935fc60664ebe60be67748de686a85d4dba651f` |
| macOS Apple Silicon | `ec65c2294b650067a11c8208a94c7a2bf1d2bebd94e867fb5dec4e8726322d14` |

上述完整版只存在本地主機，不在 Git、GitHub Release 或 Actions artifact。

## 結案重驗

在目前 `main` HEAD 重新執行：

- Docker／Xvfb 下 `go test ./...`：全數通過；
- `dwstrings check`：500/500（100%）；
- `dwstrings uicheck`：目錄 839/839、動態 key 135、玩家中文硬編 0；
- `git diff --check`：通過；
- Docker 批次結束後無冬之魔執行中／停止容器。

這輪只重新驗證現況，不把先前 A6 正常玩家第一段冒充成逐房重玩；A6 的
19 狀態節點與後期高風險抽樣邊界仍以上節及既有 playtest 文件為準。
